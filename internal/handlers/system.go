package handlers

import (
	"embed"
	htmltemplate "html/template"
	"log/slog"
	"net/http"
	"os"
	texttemplate "text/template"

	"github.com/RickyHaase/nixOS-immich-webui/internal/config"
	"github.com/RickyHaase/nixOS-immich-webui/internal/system"
)

// SystemHandler handles system configuration and management
type SystemHandler struct {
	templates embed.FS
}

// NewSystemHandler creates a new system handler
func NewSystemHandler(templates embed.FS) *SystemHandler {
	return &SystemHandler{
		templates: templates,
	}
}

// HandleRoot serves the main admin panel
func (h *SystemHandler) HandleRoot(w http.ResponseWriter, r *http.Request) {
	slog.Info("| Received Request at root |", "IP", r.Header.Get("X-Forwarded-For"))

	cfg, err := config.LoadCurrentConfig()
	if err != nil {
		slog.Error("| Error loading config |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := htmltemplate.ParseFS(h.templates, "web/index.html")
	if err != nil {
		slog.Error("| Error rendering template |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, cfg)
}

// HandleSave processes configuration save requests
func (h *SystemHandler) HandleSave(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received Save Request")

	err := r.ParseForm()
	if err != nil {
		slog.Error("| Error parsing form |", "err", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	slog.Debug("Received Form", "body", r.Form)

	cfg := &config.NixConfig{
		TimeZone:    r.FormValue("timezone"),
		AutoUpgrade: config.ParseBool(r.FormValue("auto-updates")),
		UpgradeTime: r.FormValue("update-time"),
		Tailscale:   config.ParseBool(r.FormValue("tailscale")),
		TSAuthkey:   r.FormValue("tailscale-authkey"),
	}

	t1, t2, err := config.GetLowerUpper(cfg.UpgradeTime)
	if err != nil {
		slog.Error("| Error calculating time setting |", "err", err)
		http.Error(w, "Issue with time setting"+err.Error(), http.StatusInternalServerError)
		return
	}
	cfg.UpgradeLower = t1
	cfg.UpgradeUpper = t2

	slog.Debug("Updated config", "config", cfg)

	err = h.saveTmpFile(cfg)
	if err != nil {
		slog.Error("| Error saving tmp file |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := htmltemplate.ParseFS(h.templates, "web/save.html")
	if err != nil {
		slog.Error("| Error rendering save template |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, cfg)
}

// HandleApply applies configuration changes
func (h *SystemHandler) HandleApply(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received Apply Request")

	if err := system.SwitchConfig(); err != nil {
		slog.Error("| Error when switching config files |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := system.ApplyChanges(); err != nil {
		slog.Error("| Error Applying Changes |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Rebuild Completed Successfully"))
}

// HandlePoweroff handles system poweroff requests
func (h *SystemHandler) HandlePoweroff(w http.ResponseWriter, r *http.Request) {
	if err := system.PowerOff(); err != nil {
		http.Error(w, "Failed to execute poweroff", http.StatusInternalServerError)
	}
}

// HandleReboot handles system reboot requests
func (h *SystemHandler) HandleReboot(w http.ResponseWriter, r *http.Request) {
	if err := system.Reboot(); err != nil {
		http.Error(w, "Failed to execute reboot", http.StatusInternalServerError)
	}
}

// saveTmpFile saves configuration to temporary file
func (h *SystemHandler) saveTmpFile(cfg *config.NixConfig) error {
	slog.Debug("saveTmpFile()")
	tmpl, err := texttemplate.ParseFS(h.templates, "nixos/configuration.nix")
	if err != nil {
		slog.Debug("| Error rendering template |", "err", err)
		return err
	}

	outFile, err := os.Create(config.NixDir + "configuration.tmp")
	if err != nil {
		slog.Debug("| Error creating .tmp file |", "err", err)
		return err
	}
	defer outFile.Close()

	err = tmpl.Execute(outFile, cfg)
	if err != nil {
		slog.Debug("| Error writing .tmp file |", "err", err)
		return err
	}

	return nil
}