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

// EnteHandler handles Ente service management and admin panel
type EnteHandler struct {
	templates embed.FS
}

// NewEnteHandler creates a new Ente handler
func NewEnteHandler(templates embed.FS) *EnteHandler {
	return &EnteHandler{
		templates: templates,
	}
}

// entePageData holds all data needed to render the Ente admin page
type entePageData struct {
	EnteEnabled  bool
	EnteStatus   string
	GarageStatus string
}

// HandleEntePage serves the Ente admin panel page
func (h *EnteHandler) HandleEntePage(w http.ResponseWriter, r *http.Request) {
	slog.Info("| Received Request at /ente |", "IP", r.Header.Get("X-Forwarded-For"))

	cfgJSON, err := config.LoadCurrentConfigJSON()
	if err != nil {
		slog.Error("| Error loading JSON config |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := entePageData{
		EnteEnabled: cfgJSON.Ente.Enable,
	}

	if cfgJSON.Ente.Enable {
		data.EnteStatus = system.GetEnteStatus()
		data.GarageStatus = system.GetGarageStatus()
	} else {
		data.EnteStatus = "Disabled"
		data.GarageStatus = "Disabled"
	}

	tmpl, err := htmltemplate.ParseFS(h.templates, "web/ente.html", "web/ente_status.html")
	if err != nil {
		slog.Error("| Error rendering ente template |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("| Error executing ente template |", "err", err)
		http.Error(w, "Failed to render Ente page", http.StatusInternalServerError)
		return
	}
}

// HandleEnteStatus returns the current status of Ente and Garage services as an HTMX fragment
func (h *EnteHandler) HandleEnteStatus(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Received Ente Status Request")

	cfgJSON, err := config.LoadCurrentConfigJSON()
	if err != nil {
		slog.Error("| Error loading JSON config |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := entePageData{
		EnteEnabled: cfgJSON.Ente.Enable,
	}

	if cfgJSON.Ente.Enable {
		data.EnteStatus = system.GetEnteStatus()
		data.GarageStatus = system.GetGarageStatus()
	} else {
		data.EnteStatus = "Disabled"
		data.GarageStatus = "Disabled"
	}

	tmpl, err := htmltemplate.ParseFS(h.templates, "web/ente_status.html")
	if err != nil {
		slog.Error("| Error parsing ente status template |", "err", err)
		http.Error(w, "Failed to render status", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "ente_status", data); err != nil {
		slog.Error("| Error executing ente status template |", "err", err)
		http.Error(w, "Failed to render status", http.StatusInternalServerError)
		return
	}
}

// HandleEnteStart starts the Ente service
func (h *EnteHandler) HandleEnteStart(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received Ente Start Request")

	if err := system.GarageService("start"); err != nil {
		slog.Error("| Error starting garage.service |", "err", err)
		http.Error(w, "Issue starting Garage: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := system.EnteService("start"); err != nil {
		slog.Error("| Error starting ente.service |", "err", err)
		http.Error(w, "Issue starting Ente: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Ente started"))
}

// HandleEnteStop stops the Ente service
func (h *EnteHandler) HandleEnteStop(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received Ente Stop Request")

	if err := system.EnteService("stop"); err != nil {
		slog.Error("| Error stopping ente.service |", "err", err)
		http.Error(w, "Issue stopping Ente: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Ente stopped"))
}

// HandleEnteRestart restarts the Ente service
func (h *EnteHandler) HandleEnteRestart(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received Ente Restart Request")

	if err := system.GarageService("restart"); err != nil {
		slog.Error("| Error restarting garage.service |", "err", err)
		http.Error(w, "Issue restarting Garage: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := system.EnteService("restart"); err != nil {
		slog.Error("| Error restarting ente.service |", "err", err)
		http.Error(w, "Issue restarting Ente: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Ente restarted"))
}

// HandleEnteToggle enables or disables Ente in the NixOS configuration
func (h *EnteHandler) HandleEnteToggle(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received Ente Toggle Request")

	err := r.ParseForm()
	if err != nil {
		slog.Error("| Error parsing ente toggle form |", "err", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	enable := config.ParseBool(r.FormValue("ente-enable"))

	cfgJSON, err := config.LoadCurrentConfigJSON()
	if err != nil {
		slog.Error("| Error loading config |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cfgJSON.Ente.Enable = enable

	if err := config.SaveConfigJSON(cfgJSON); err != nil {
		slog.Error("| Error saving config |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := config.SwitchConfigJSON(); err != nil {
		slog.Error("| Error switching config |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := system.ApplyChanges(); err != nil {
		slog.Error("| Error applying changes — attempting rollback |", "err", err)

		if rollbackErr := system.RollbackConfigJSON(); rollbackErr != nil {
			slog.Error("| Rollback failed |", "rollbackErr", rollbackErr)
			http.Error(w, fmt.Sprintf("nixos-rebuild failed and rollback also failed: %v. Original error: %v", rollbackErr, err), http.StatusInternalServerError)
			return
		}

		http.Error(w, fmt.Sprintf("nixos-rebuild failed: %v. Configuration has been rolled back.", err), http.StatusInternalServerError)
		return
	}

	// Redirect back to Ente page to show updated state
	http.Redirect(w, r, "/ente", http.StatusSeeOther)
}
