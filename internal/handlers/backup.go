package handlers

import (
	"embed"
	htmltemplate "html/template"
	"log/slog"
	"net/http"

	"github.com/RickyHaase/nixOS-immich-webui/internal/services"
	"github.com/RickyHaase/nixOS-immich-webui/internal/system"
)

// BackupHandler handles backup operations
type BackupHandler struct {
	templates     embed.FS
	backupService *services.BackupService
}

// NewBackupHandler creates a new backup handler
func NewBackupHandler(templates embed.FS, backupService *services.BackupService) *BackupHandler {
	return &BackupHandler{
		templates:     templates,
		backupService: backupService,
	}
}

// HandleGetDisks returns eligible disks for backup
func (h *BackupHandler) HandleGetDisks(w http.ResponseWriter, r *http.Request) {
	disks, err := system.GetEligibleDisks()
	if err != nil {
		slog.Error("Error getting eligible disks", "err", err)
	}

	if len(disks) == 0 {
		slog.Debug("No eligible disks found")
		tmpl, err := htmltemplate.ParseFS(h.templates, "web/no_disks.html")
		if err != nil {
			slog.Error("| Error parsing no disks template |", "err", err)
			http.Error(w, "Failed to render disk list", http.StatusInternalServerError)
			return
		}

		if err := tmpl.Execute(w, nil); err != nil {
			slog.Error("| Error executing no disks template |", "err", err)
			http.Error(w, "Failed to render disk list", http.StatusInternalServerError)
			return
		}
		return
	}

	tmpl, err := htmltemplate.ParseFS(h.templates, "web/disk_options.html")
	if err != nil {
		slog.Error("| Error parsing disk options template |", "err", err)
		http.Error(w, "Failed to render disk list", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, disks); err != nil {
		slog.Error("| Error executing disk options template |", "err", err)
		http.Error(w, "Failed to render disk list", http.StatusInternalServerError)
		return
	}
}

// HandleBackup processes backup requests (starts backup in background)
func (h *BackupHandler) HandleBackup(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received Backup Request")

	// Check if backup is already in progress
	state := h.backupService.GetState()
	if state.InProgress {
		slog.Warn("| Backup already in progress |")
		http.Error(w, "A backup is already in progress. Please wait for it to complete.", http.StatusConflict)
		return
	}

	err := r.ParseForm()
	if err != nil {
		slog.Error("| Error parsing backup form submission |", "err", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	selectedDisk := r.FormValue("select-disk")
	verifyChecksum := r.FormValue("verify-backup") == "true"
	slog.Debug("Selected disk", "disk", selectedDisk, "verify", verifyChecksum)

	// Validate disk selection
	disks, err := system.GetEligibleDisks()
	if err != nil {
		slog.Error("| Error getting eligible disks |", "err", err)
		http.Error(w, "Error getting eligible disks", http.StatusInternalServerError)
		return
	}

	matchFound := false
	for _, disk := range disks {
		if disk.Identifier == selectedDisk {
			matchFound = true
			break
		}
	}

	if !matchFound {
		slog.Error("| Invalid disk selection |", "selectedDisk", selectedDisk)
		http.Error(w, "Disk is not available for backups. Please refresh page and try again.", http.StatusBadRequest)
		return
	}

	// Start backup in background goroutine
	go func() {
		slog.Info("Starting backup in background", "disk", selectedDisk, "verify", verifyChecksum)
		_, err := h.backupService.BackupToUSB(selectedDisk, verifyChecksum)
		if err != nil {
			slog.Error("| Backup failed |", "err", err)
		} else {
			slog.Info("| Backup completed successfully |")
		}
	}()

	// Return immediately with "backup started" message that triggers status polling
	htmlResponse := `<div style="padding: 10px; background: #d4edda; color: #155724; border: 1px solid #c3e6cb; border-radius: 4px; margin: 10px 0;"
	                      hx-get="/backupstatus" hx-trigger="load delay:500ms" hx-target="#backup-status" hx-swap="innerHTML">
		<strong>Backup Started!</strong><br>
		The backup is now running in the background. Progress will be shown below.
	</div>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(htmlResponse))
}

// HandleGetBackupStatus returns backup status information with current state
func (h *BackupHandler) HandleGetBackupStatus(w http.ResponseWriter, r *http.Request) {
	// Get current backup state
	state := h.backupService.GetState()

	// Get last backup from history
	lastBackup, err := services.GetLastBackup()
	if err != nil {
		slog.Error("| Error getting last backup |", "err", err)
		// Continue anyway, just won't show last backup info
	}

	// Prepare template data
	data := struct {
		State      services.BackupState
		LastBackup *services.BackupHistoryEntry
	}{
		State:      state,
		LastBackup: lastBackup,
	}

	tmpl, err := htmltemplate.ParseFS(h.templates, "web/backup_status.html")
	if err != nil {
		slog.Error("| Error parsing backup status template |", "err", err)
		http.Error(w, "Failed to render backup status", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("| Error executing backup status template |", "err", err)
		http.Error(w, "Failed to render backup status", http.StatusInternalServerError)
		return
	}
}