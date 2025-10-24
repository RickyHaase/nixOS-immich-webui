package handlers

import (
	"embed"
	"fmt"
	htmltemplate "html/template"
	"log/slog"
	"net/http"

	"github.com/RickyHaase/nixOS-immich-webui/internal/config"
	"github.com/RickyHaase/nixOS-immich-webui/internal/system"
)

// ImmichHandler handles Immich service management
type ImmichHandler struct {
	templates embed.FS
}

// NewImmichHandler creates a new Immich handler
func NewImmichHandler(templates embed.FS) *ImmichHandler {
	return &ImmichHandler{
		templates: templates,
	}
}

// HandleStatus returns the current status of Immich service
func (h *ImmichHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Received Status Request")
	w.Write([]byte(system.GetStatus()))
}

// HandleStop stops the Immich service
func (h *ImmichHandler) HandleStop(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received Stop Request")

	err := system.ImmichService("stop")
	if err != nil {
		slog.Error("| Error stopping immich-app.service |", "err", err)
		http.Error(w, "Issue stopping Immich"+err.Error(), http.StatusInternalServerError)
	}
}

// HandleStart starts the Immich service
func (h *ImmichHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received Start Request")

	err := system.ImmichService("start")
	if err != nil {
		slog.Error("| Error starting immich-app.service |", "err", err)
		http.Error(w, "Issue starting Immich"+err.Error(), http.StatusInternalServerError)
	}
}

// HandleUpdate updates Immich containers
func (h *ImmichHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received Update Request")

	if err := system.ImmichService("stop"); err != nil {
		slog.Error("| Error stopping immich-app.service |", "err", err)
		http.Error(w, "Issue stopping Immich"+err.Error(), http.StatusInternalServerError)
	}

	if err := system.UpdateImmichContainer(); err != nil {
		slog.Error("| Error updating Immich |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := system.ImmichService("start"); err != nil {
		slog.Error("| Error starting immich-app.service |", "err", err)
		http.Error(w, "Issue starting Immich"+err.Error(), http.StatusInternalServerError)
	}

	w.Write([]byte("Pulled new containers successfully"))
}

// HandleEmailPost processes email configuration updates
func (h *ImmichHandler) HandleEmailPost(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received Email Post")

	err := r.ParseForm()
	if err != nil {
		slog.Error("| Error parsing email form submission |", "err", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	email := r.FormValue("gmail-address")
	password := r.FormValue("gmail-password")

	// Validate email format
	if err := config.ValidateEmail(email); err != nil {
		slog.Error("| Invalid email format |", "err", err)
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	if err := config.SetImmichEmail(email, password); err != nil {
		slog.Error("| Failed to set Immich email config |", "err", err)
		http.Error(w, "Failed to set Immich email config.", http.StatusInternalServerError)
		return
	}

	// Get email settings directly from immich-config.json
	immich, err := config.GetImmichConfig()
	if err != nil {
		slog.Error("| Error parsing immich-config.json |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create simple struct for template data
	emailData := struct {
		Email     string
		EmailPass bool
		Saved     bool
	}{
		Email:     immich.Notifications.SMTP.Transport.Username,
		EmailPass: immich.Notifications.SMTP.Transport.Password != "",
		Saved:     true,
	}

	tmpl, err := htmltemplate.ParseFS(h.templates, "web/email_form.html")
	if err != nil {
		slog.Error("| Error parsing email form template |", "err", err)
		http.Error(w, "Failed to render email form", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, emailData); err != nil {
		slog.Error("| Error executing email form template |", "err", err)
		http.Error(w, "Failed to render email form", http.StatusInternalServerError)
		return
	}
}

// HandleMLModelPost processes machine learning model configuration updates
func (h *ImmichHandler) HandleMLModelPost(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received ML Model Post")

	err := r.ParseForm()
	if err != nil {
		slog.Error("| Error parsing ML model form submission |", "err", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	modelName := r.FormValue("ml-model")

	// Validate model name using centralized validation
	if !config.IsValidMLModel(modelName) {
		slog.Error("| Invalid ML model submitted |", "modelName", modelName)
		http.Error(w, "Invalid model selection.", http.StatusBadRequest)
		return
	}

	if err := config.SetMLModel(modelName); err != nil {
		slog.Error("| Failed to set ML model |", "err", err)
		http.Error(w, "Failed to set ML model.", http.StatusInternalServerError)
		return
	}

	// Get current model from immich-config.json
	immich, err := config.GetImmichConfig()
	if err != nil {
		slog.Error("| Error parsing immich-config.json |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create struct for template data
	mlData := struct {
		ModelName string
		Saved     bool
	}{
		ModelName: immich.MachineLearning.Clip.ModelName,
		Saved:     true,
	}

	tmpl, err := htmltemplate.ParseFS(h.templates, "web/ml_form.html")
	if err != nil {
		slog.Error("| Error parsing ML model form template |", "err", err)
		http.Error(w, "Failed to render ML model form", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, mlData); err != nil {
		slog.Error("| Error executing ML model form template |", "err", err)
		http.Error(w, "Failed to render ML model form", http.StatusInternalServerError)
		return
	}
}

// HandleOAuthPost processes OAuth configuration updates
func (h *ImmichHandler) HandleOAuthPost(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received OAuth Post")

	err := r.ParseForm()
	if err != nil {
		slog.Error("| Error parsing OAuth form submission |", "err", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Extract form values
	oauthEnabled := config.ParseBool(r.FormValue("oauth-enabled"))
	passwordLogin := config.ParseBool(r.FormValue("password-login"))
	clientId := r.FormValue("oauth-client-id")
	clientSecret := r.FormValue("oauth-client-secret")
	issuerUrl := r.FormValue("oauth-issuer-url")
	publicDomain := r.FormValue("oauth-public-domain")

	// Validate inputs
	if err := config.ValidateOAuthClientId(clientId, oauthEnabled); err != nil {
		slog.Error("| Invalid OAuth client ID |", "err", err)
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	if err := config.ValidateOAuthClientSecret(clientSecret, oauthEnabled); err != nil {
		slog.Error("| Invalid OAuth client secret |", "err", err)
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	if err := config.ValidateOAuthIssuerUrl(issuerUrl, oauthEnabled); err != nil {
		slog.Error("| Invalid OAuth issuer URL |", "err", err)
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	if err := config.ValidatePublicDomain(publicDomain, oauthEnabled); err != nil {
		slog.Error("| Invalid public domain |", "err", err)
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	// Apply OAuth configuration
	if err := config.SetOAuthConfig(oauthEnabled, passwordLogin, clientId, clientSecret, issuerUrl, publicDomain); err != nil {
		slog.Error("| Failed to set OAuth config |", "err", err)
		http.Error(w, "Failed to set OAuth config.", http.StatusInternalServerError)
		return
	}

	// Get current OAuth settings from immich-config.json
	immich, err := config.GetImmichConfig()
	if err != nil {
		slog.Error("| Error parsing immich-config.json |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create struct for template data
	oauthData := struct {
		OAuthEnabled         bool
		PasswordLoginEnabled bool
		OAuthClientId        string
		OAuthClientSecretSet bool
		OAuthIssuerUrl       string
		OAuthPublicDomain    string
		Saved                bool
	}{
		OAuthEnabled:         immich.OAuth.Enabled,
		PasswordLoginEnabled: immich.PasswordLogin.Enabled,
		OAuthClientId:        immich.OAuth.ClientId,
		OAuthClientSecretSet: immich.OAuth.ClientSecret != "",
		OAuthIssuerUrl:       immich.OAuth.IssuerUrl,
		OAuthPublicDomain:    immich.Server.ExternalDomain,
		Saved:                true,
	}

	// Strip https:// prefix from externalDomain for display
	if len(oauthData.OAuthPublicDomain) > 8 && oauthData.OAuthPublicDomain[:8] == "https://" {
		oauthData.OAuthPublicDomain = oauthData.OAuthPublicDomain[8:]
	} else if len(oauthData.OAuthPublicDomain) > 7 && oauthData.OAuthPublicDomain[:7] == "http://" {
		oauthData.OAuthPublicDomain = oauthData.OAuthPublicDomain[7:]
	}

	tmpl, err := htmltemplate.ParseFS(h.templates, "web/index.html", "web/oauth_form.html")
	if err != nil {
		slog.Error("| Error parsing OAuth form template |", "err", err)
		http.Error(w, "Failed to render OAuth form", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "oauth_form", oauthData); err != nil {
		slog.Error("| Error executing OAuth form template |", "err", err)
		http.Error(w, "Failed to render OAuth form", http.StatusInternalServerError)
		return
	}
}