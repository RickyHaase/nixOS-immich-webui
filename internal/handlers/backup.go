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
		htmlStr := `<option>No eligible disks found</option>`
		tmpl, _ := htmltemplate.New("t").Parse(htmlStr)
		tmpl.Execute(w, disks)
		return
	}

	htmlStr := `
	{{range .}}
	<option value={{.Identifier}}>{{.PartitionLabel}} ({{.PartitionSize}}) on {{.Model}}</option>
	{{end}}
	`
	tmpl, _ := htmltemplate.New("t").Parse(htmlStr)
	tmpl.Execute(w, disks)
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

	htmlStr := `
 		<label for="select-disk">Select Disk:</label>
        <select name="select-disk" id="select-disk" hx-get="/disks" hx-trigger="load" hx-confirm="Backup Completed Successfully!">
            <option>Refresh page to re-load backup options</option>
        </select>
        <button id="refresh" type="button" hx-get="/disks" hx-target="#select-disk" hx-swap="innerHTML">Refresh List</button>
        <button id="start-backup" type="submit" hx-post="/backup" hx-target="#backup-form" hx-confirm="Are you sure you want to start the backup? This may take some time.">Start Backup</button>
        <br><small>Select backup disk from list. In order for a disk to be eligible, it must be connected via USB and have a partition formatted exFAT.</small>
	`
	tmpl, _ := htmltemplate.New("t").Parse(htmlStr)
	tmpl.Execute(w, "")
}

// HandleGetBackupStatus returns backup status information
func (h *BackupHandler) HandleGetBackupStatus(w http.ResponseWriter, r *http.Request) {
	htmlStr := `
 		<label for="select-disk">Select Disk:</label>
        <select name="select-disk" id="select-disk" hx-get="/disks" hx-trigger="load">
            <option>Requires JavaScript to be Enabled</option>
        </select>
        <button id="refresh" type="button" hx-get="/disks" hx-target="#select-disk" hx-swap="innerHTML">Refresh List</button>
        <button id="start-backup" type="submit" hx-post="/backup" hx-target="#backup-form" hx-confirm="Are you sure you want to start the backup? This may take some time.">Start Backup</button>
        <br><small>Select backup disk from list. In order for a disk to be eligible, it must be connected via USB and have a partition formatted exFAT.</small>
	`
	tmpl, _ := htmltemplate.New("t").Parse(htmlStr)
	tmpl.Execute(w, "")
}