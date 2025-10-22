package handlers

import (
	"embed"
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

	if err := config.SetImmichEmail(r.FormValue("gmail-address"), r.FormValue("gmail-password")); err != nil {
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
	}{
		Email:     immich.Notifications.SMTP.Transport.Username,
		EmailPass: immich.Notifications.SMTP.Transport.Password != "",
	}

	htmlStr := `    <form id="email-form" action="/email" method="post">
        <label for="gmail-address">Gmail Address:</label>
        <input type="email" id="gmail-address" name="gmail-address" placeholder="example@gmail.com" value="{{if .Email}}{{.Email}}{{else}}{{end}}">        <label for="gmail-password">Gmail App Password:</label>
        <input type="password" id="gmail-password" name="gmail-password" placeholder="{{if .EmailPass}}password is set{{else}}fded beid aibr kxps{{end}}">
        <button type="submit" hx-post="/email" hx-target="#email-form">Submit</button>
        <br><small>Use your gmail account with an <a href="https://support.google.com/mail/answer/185833">app password</a> to allow for immich to send emails</small>
    </form>`

	tmpl, err := htmltemplate.New("t").Parse(htmlStr)
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

	// Validate model name before processing
	validModels := map[string]bool{
		"ViT-B-32__openai":                true,
		"ViT-B-16-SigLIP__webli":          true,
		"ViT-SO400M-14-SigLIP-384__webli": true,
	}

	if !validModels[modelName] {
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
	}{
		ModelName: immich.MachineLearning.Clip.ModelName,
	}

	htmlStr := `    <form id="ml-form" action="/mlmodel" method="post">
        <label for="ml-model">CLIP Model:</label>
        <select name="ml-model" id="ml-model">
            <option value="ViT-B-32__openai" {{if eq .ModelName "ViT-B-32__openai"}}selected{{end}}>Default (ViT-B-32__openai)</option>
            <option value="ViT-B-16-SigLIP__webli" {{if eq .ModelName "ViT-B-16-SigLIP__webli"}}selected{{end}}>Improved (ViT-B-16-SigLIP__webli)</option>
            <option value="ViT-SO400M-14-SigLIP-384__webli" {{if eq .ModelName "ViT-SO400M-14-SigLIP-384__webli"}}selected{{end}}>Beefy (ViT-SO400M-14-SigLIP-384__webli)</option>
        </select>
        <button type="submit" hx-post="/mlmodel" hx-target="#ml-form">Submit</button>
        <br><small>Select the machine learning model for image recognition. Higher quality models require more resources.</small>
    </form>`

	tmpl, err := htmltemplate.New("t").Parse(htmlStr)
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