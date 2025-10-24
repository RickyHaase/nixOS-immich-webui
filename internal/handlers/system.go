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

	cfgJSON, err := config.LoadCurrentConfigJSON()
	if err != nil {
		slog.Error("| Error loading JSON config |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to old format for template compatibility
	cfg := cfgJSON.ToNixConfig()

	tmpl, err := htmltemplate.ParseFS(h.templates, "web/index.html", "web/oauth_form.html")
	if err != nil {
		slog.Error("| Error rendering template |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, cfg); err != nil {
		slog.Error("| Error executing root template |", "err", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// HandleConfig serves the configuration page
func (h *SystemHandler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	slog.Info("| Received Request at /config |", "IP", r.Header.Get("X-Forwarded-For"))

	cfgJSON, err := config.LoadCurrentConfigJSON()
	if err != nil {
		slog.Error("| Error loading JSON config |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to old format for template compatibility
	cfg := cfgJSON.ToNixConfig()

	tmpl, err := htmltemplate.ParseFS(h.templates, "web/config.html", "web/oauth_form.html")
	if err != nil {
		slog.Error("| Error rendering config template |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, cfg); err != nil {
		slog.Error("| Error executing config template |", "err", err)
		http.Error(w, "Failed to render configuration page", http.StatusInternalServerError)
		return
	}
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

	// Extract form values
	timezone := r.FormValue("timezone")
	updateTime := r.FormValue("update-time")
	tailscaleAuthKey := r.FormValue("tailscale-authkey")
	tailscaleEnabled := config.ParseBool(r.FormValue("tailscale"))
	cloudflaredToken := r.FormValue("cloudflared-token")
	cloudflaredEnabled := config.ParseBool(r.FormValue("cloudflared"))

	// Validate inputs
	if err := config.ValidateTimezone(timezone); err != nil {
		slog.Error("| Invalid timezone |", "err", err)
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	if err := config.ValidateTimeFormat(updateTime); err != nil {
		slog.Error("| Invalid time format |", "err", err)
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	// Only validate Tailscale auth key if Tailscale is enabled
	if tailscaleEnabled {
		if err := config.ValidateTailscaleAuthKey(tailscaleAuthKey); err != nil {
			slog.Error("| Invalid Tailscale auth key |", "err", err)
			http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Only validate Cloudflared token if Cloudflared is enabled
	if cloudflaredEnabled {
		if err := config.ValidateCloudflaredToken(cloudflaredToken); err != nil {
			slog.Error("| Invalid Cloudflare tunnel token |", "err", err)
			http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Load current config to preserve existing secrets if not provided
	currentCfg, err := config.LoadCurrentConfigJSON()
	if err != nil {
		slog.Error("| Error loading current config |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build new JSON configuration structure
	cfgJSON := &config.ConfigVariables{}
	cfgJSON.System.TimeZone = timezone
	cfgJSON.System.AutoUpgrade = config.ParseBool(r.FormValue("auto-updates"))
	cfgJSON.System.UpgradeTime = updateTime
	cfgJSON.RemoteAccess.Tailscale.Enable = tailscaleEnabled
	// Preserve existing auth key if empty submitted (field is blank in UI when key is set)
	if tailscaleAuthKey == "" {
		cfgJSON.RemoteAccess.Tailscale.AuthKey = currentCfg.RemoteAccess.Tailscale.AuthKey
	} else {
		cfgJSON.RemoteAccess.Tailscale.AuthKey = tailscaleAuthKey
	}
	cfgJSON.RemoteAccess.Cloudflared.Enable = cloudflaredEnabled
	// Preserve existing token if empty submitted (field is blank in UI when token is set)
	if cloudflaredToken == "" {
		cfgJSON.RemoteAccess.Cloudflared.Token = currentCfg.RemoteAccess.Cloudflared.Token
	} else {
		cfgJSON.RemoteAccess.Cloudflared.Token = cloudflaredToken
	}

	t1, t2, err := config.GetLowerUpper(cfgJSON.System.UpgradeTime)
	if err != nil {
		slog.Error("| Error calculating time setting |", "err", err)
		http.Error(w, "Issue with time setting"+err.Error(), http.StatusInternalServerError)
		return
	}
	cfgJSON.System.UpgradeLower = t1
	cfgJSON.System.UpgradeUpper = t2

	slog.Debug("Updated JSON config", "config", cfgJSON)

	err = config.SaveConfigJSON(cfgJSON)
	if err != nil {
		slog.Error("| Error saving JSON config |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to old format for template compatibility
	cfg := cfgJSON.ToNixConfig()


	tmpl, err := htmltemplate.ParseFS(h.templates, "web/save.html")
	if err != nil {
		slog.Error("| Error rendering save template |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, cfg); err != nil {
		slog.Error("| Error executing save template |", "err", err)
		http.Error(w, "Failed to render save confirmation", http.StatusInternalServerError)
		return
	}
}

// HandleApply applies configuration changes
func (h *SystemHandler) HandleApply(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received Apply Request")

	if err := config.SwitchConfigJSON(); err != nil {
		slog.Error("| Error when switching JSON config files |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := system.ApplyChanges(); err != nil {
		slog.Error("| Error Applying Changes - attempting rollback |", "err", err)

		// Attempt to rollback configuration to previous working state
		if rollbackErr := system.RollbackConfigJSON(); rollbackErr != nil {
			slog.Error("| Rollback failed |", "rollbackErr", rollbackErr)
			http.Error(w, fmt.Sprintf("nixos-rebuild failed and rollback also failed: %v. Original error: %v", rollbackErr, err), http.StatusInternalServerError)
			return
		}

		slog.Info("| Configuration rolled back successfully |")
		http.Error(w, fmt.Sprintf("nixos-rebuild failed: %v. Configuration has been rolled back to previous version.", err), http.StatusInternalServerError)
		return
	}

	tmpl, err := htmltemplate.ParseFS(h.templates, "web/apply_success.html")
	if err != nil {
		slog.Error("| Error parsing apply success template |", "err", err)
		http.Error(w, "Rebuild completed but failed to render success page", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		slog.Error("| Error executing apply success template |", "err", err)
		http.Error(w, "Rebuild completed but failed to render success page", http.StatusInternalServerError)
		return
	}
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

