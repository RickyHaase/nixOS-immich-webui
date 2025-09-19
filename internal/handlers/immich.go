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

	if err := config.SetImmichConfig(r.FormValue("gmail-address"), r.FormValue("gmail-password")); err != nil {
		slog.Error("| Failed to set Immich config |", "err", err)
		http.Error(w, "Failed to set Immich config.", http.StatusInternalServerError)
		return
	}

	// Parse settings out of immich-config.json - might need to refactor these 15 lines out into a function to maintain DRY best practice
	cfg := config.NixConfig{}
	immich, err := config.GetImmichConfig()
	if err != nil {
		slog.Error("| Error parisng immich-config.json |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cfg.Email = immich.Notifications.SMTP.Transport.Username
	if immich.Notifications.SMTP.Transport.Password != "" {
		slog.Debug("Contains Password = True")
		cfg.EmailPass = true
	} else {
		slog.Debug("Contains Password = False")
		cfg.EmailPass = false
	}

	htmlStr := `    <form id="email-form" action="/email" method="post">
        <label for="gmail-address">Gmail Address:</label>
        <input type="email" id="gmail-address" name="gmail-address" placeholder="example@gmail.com" value="{{if .Email}}{{.Email}}{{else}}{{end}}">        <label for="gmail-password">Gmail App Password:</label>
        <input type="password" id="gmail-password" name="gmail-password" placeholder="{{if .EmailPass}}password is set{{else}}fded beid aibr kxps{{end}}">
        <button type="submit" hx-post="/email" hx-target="#email-form">Submit</button>
        <br><small>Use your gmail account with an <a href="https://support.google.com/mail/answer/185833">app password</a> to allow for immich to send emails</small>
    </form>`

	tmpl, _ := htmltemplate.New("t").Parse(htmlStr)
	tmpl.Execute(w, cfg)
}