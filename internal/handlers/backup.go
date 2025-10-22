package handlers

import (
	"embed"
	"fmt"
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
		slog.Error("Error getting eiligible disks", "err", err)
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

// HandleBackup processes backup requests
func (h *BackupHandler) HandleBackup(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received Backup Request")

	err := r.ParseForm()
	if err != nil {
		slog.Error("| Error parsing backup form submission |", "err", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	fmt.Println(r.FormValue("select-disk"))

	disks, err := system.GetEligibleDisks()
	if err != nil {
		slog.Error("| Error getting eiligible disks |", "err", err)
		http.Error(w, "Error getting eiligible disks", http.StatusInternalServerError)
		return
	}

	selectedDisk := r.FormValue("select-disk")
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

	backupResult, err := h.backupService.BackupToUSB(selectedDisk)
	if err != nil {
		slog.Error("| Error backing up to disk |", "err", err)
		http.Error(w, "Error backing up to disk", http.StatusInternalServerError)
		return
	}
	slog.Info(backupResult)

	tmpl, err := htmltemplate.ParseFS(h.templates, "web/backup_form.html")
	if err != nil {
		slog.Error("| Error parsing backup success template |", "err", err)
		http.Error(w, "Backup completed but failed to render response", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		slog.Error("| Error executing backup success template |", "err", err)
		http.Error(w, "Backup completed but failed to render response", http.StatusInternalServerError)
		return
	}
}

// HandleGetBackupStatus returns backup status information
func (h *BackupHandler) HandleGetBackupStatus(w http.ResponseWriter, r *http.Request) {
	tmpl, err := htmltemplate.ParseFS(h.templates, "web/backup_status.html")
	if err != nil {
		slog.Error("| Error parsing backup status template |", "err", err)
		http.Error(w, "Failed to render backup status", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		slog.Error("| Error executing backup status template |", "err", err)
		http.Error(w, "Failed to render backup status", http.StatusInternalServerError)
		return
	}
}