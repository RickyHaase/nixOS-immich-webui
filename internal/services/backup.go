package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/RickyHaase/nixOS-immich-webui/internal/config"
)

// BackupState tracks the current state of a backup operation
type BackupState struct {
	InProgress      bool      `json:"inProgress"`
	Status          string    `json:"status"` // "idle", "mounting", "configs", "library", "complete", "error"
	CurrentStep     string    `json:"currentStep"`
	ProgressPercent int       `json:"progressPercent"`
	TotalFiles      int64     `json:"totalFiles"`
	ProcessedFiles  int64     `json:"processedFiles"`
	CurrentFile     string    `json:"currentFile"`
	StartTime       time.Time `json:"startTime"`
	ErrorMessage    string    `json:"errorMessage"`
	VerifyChecksum  bool      `json:"verifyChecksum"`
}

// BackupHistoryEntry represents a single backup operation record
type BackupHistoryEntry struct {
	Timestamp        time.Time `json:"timestamp"`
	Status           string    `json:"status"` // "success", "failed"
	DurationSec      int       `json:"durationSec"`
	FilesBackedUp    int64     `json:"filesBackedUp"`
	TotalSizeMB      int64     `json:"totalSizeMB"`
	DiskUsed         string    `json:"diskUsed"`
	BackupType       string    `json:"backupType"` // "usb" (future: "internal", "safety")
	ErrorMessage     string    `json:"errorMessage"`
	VerifiedChecksum bool      `json:"verifiedChecksum"`
}

// BackupHistory maintains a log of backup operations
type BackupHistory struct {
	Backups []BackupHistoryEntry `json:"backups"`
}

// BackupService handles backup operations
type BackupService struct {
	state   BackupState
	stateMu sync.RWMutex
}

// NewBackupService creates a new backup service
func NewBackupService() *BackupService {
	return &BackupService{
		state: BackupState{
			Status: "idle",
		},
	}
}

// GetState returns a copy of the current backup state (thread-safe)
func (s *BackupService) GetState() BackupState {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state
}

// setState updates the backup state (thread-safe)
func (s *BackupService) setState(status string, progressPercent int, currentStep string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.state.Status = status
	s.state.ProgressPercent = progressPercent
	s.state.CurrentStep = currentStep
	slog.Debug("Backup state updated", "status", status, "progress", progressPercent, "step", currentStep)
}

// setError sets an error state (thread-safe)
func (s *BackupService) setError(errorMsg string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.state.Status = "error"
	s.state.InProgress = false
	s.state.ErrorMessage = errorMsg
	slog.Error("Backup error", "error", errorMsg)
}

// resetState resets the backup state to idle (thread-safe)
func (s *BackupService) resetState() {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.state = BackupState{
		Status: "idle",
	}
}

// BackupToUSB performs a complete backup to the specified USB disk
func (s *BackupService) BackupToUSB(disk string, verifyChecksum bool) (string, error) {
	slog.Debug("backupToUSB() - Start", "disk", disk, "verify", verifyChecksum)

	// Initialize backup state
	s.stateMu.Lock()
	s.state = BackupState{
		InProgress:      true,
		Status:          "starting",
		CurrentStep:     "Initializing backup",
		ProgressPercent: 0,
		StartTime:       time.Now(),
		VerifyChecksum:  verifyChecksum,
	}
	s.stateMu.Unlock()

	startTime := time.Now()
	var filesBackedUp int64 = 0

	// Helper function to handle errors and update history
	handleError := func(err error, step string) (string, error) {
		s.setError(fmt.Sprintf("Error during %s: %v", step, err))
		_ = AddBackupEntry(BackupHistoryEntry{
			Timestamp:        time.Now(),
			Status:           "failed",
			DurationSec:      int(time.Since(startTime).Seconds()),
			DiskUsed:         disk,
			BackupType:       "usb",
			ErrorMessage:     fmt.Sprintf("%s: %v", step, err),
			VerifiedChecksum: verifyChecksum,
		})
		return "", err
	}

	// ============== Mount Disk ==============
	s.setState("mounting", 0, "Mounting backup disk")

	mountCheckCmd := exec.Command("lsblk", "-no", "MOUNTPOINT", "/dev/"+disk)
	mountPoint, err := mountCheckCmd.Output()
	if err != nil {
		slog.Error("Error checking if disk is mounted:", "err", err)
		return handleError(err, "mount check")
	}
	slog.Debug("Mount point check output", "mountPoint", string(mountPoint))

	if len(mountPoint) == 1 && mountPoint[0] == 10 { // Checks that the mountpoint is just an empty line
		slog.Debug("Disk is not mounted, attempting to mount", "disk", disk)
		mountCmd := exec.Command("udisksctl", "mount", "-b", "/dev/"+disk)
		err := mountCmd.Run()
		if err != nil {
			slog.Error("Error mounting disk:", "err", err)
			return handleError(err, "mount")
		}

		mountCheckCmd = exec.Command("lsblk", "-no", "MOUNTPOINT", "/dev/"+disk)
		mountPoint, err = mountCheckCmd.Output()
		if err != nil {
			slog.Error("Error re-checking mount point:", "err", err)
			return handleError(err, "mount re-check")
		}
		slog.Debug("Mount point re-check output", "mountPoint", string(mountPoint))
	}

	mountPointStr := string(mountPoint)
	mountPointStr = mountPointStr[:len(mountPointStr)-1]
	slog.Debug("Final mount point", "mountPointStr", mountPointStr)

	// Check if [mountpoint]/immich-server-backup exists
	backupDir := mountPointStr + "/immich-server-backup"
	slog.Info("Ensuring backup directory exists...", "backupDir", backupDir)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		slog.Error("Error creating backup directory:", "err", err)
		return handleError(err, "create backup directory")
	}

	// =============== Config Backups (0-10%) ===================
	s.setState("configs", 5, "Backing up configuration files and database")
	if err := s.backupConfigs(backupDir); err != nil {
		return handleError(err, "config backup")
	}

	// ===================Library Backup with Rsync (10-100%)==========================
	stepMsg := "Backing up photo library"
	if verifyChecksum {
		stepMsg = "Backing up photo library (with checksum verification)"
	}
	s.setState("library", 10, stepMsg)
	if err := s.backupLibrary(backupDir, verifyChecksum); err != nil {
		return handleError(err, "library backup")
	}

	// Get file count for history (best effort)
	// TODO: Track this more accurately during rsync
	filesBackedUp = s.state.ProcessedFiles

	// ================= Backups done - can unmount disk =============
	s.setState("unmounting", 95, "Unmounting backup disk")
	if err := s.unmountDisk(disk); err != nil {
		return handleError(err, "unmount")
	}

	// ============== Complete ==============
	duration := int(time.Since(startTime).Seconds())
	s.setState("complete", 100, "Backup completed successfully")

	// Add successful entry to history
	_ = AddBackupEntry(BackupHistoryEntry{
		Timestamp:        time.Now(),
		Status:           "success",
		DurationSec:      duration,
		FilesBackedUp:    filesBackedUp,
		DiskUsed:         disk,
		BackupType:       "usb",
		VerifiedChecksum: verifyChecksum,
	})

	// Reset state after a brief moment
	time.Sleep(2 * time.Second)
	s.resetState()

	slog.Debug("backupToUSB() - End")
	return "backup complete", nil
}

// backupConfigs backs up configuration files and database dumps
func (s *BackupService) backupConfigs(backupDir string) error {
	// Create a temporary directory for the backup files
	tempDir := "/tmp/nixmichBackup"
	slog.Debug("Creating temporary directory for backup files", "tempDir", tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		slog.Error("Error creating temporary directory:", "err", err)
		return err
	}

	// Create organized folder structure
	essentialDir := tempDir + "/essential"
	supplementalDir := tempDir + "/supplemental"
	nixFilesDir := supplementalDir + "/nix-files"

	for _, dir := range []string{essentialDir, supplementalDir, nixFilesDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			slog.Error("Error creating backup subdirectory:", "err", err, "dir", dir)
			return err
		}
	}

	// ============= ESSENTIAL FILES (minimum for restore) =============
	slog.Debug("Backing up essential files")

	// Copy nixconfig.json (the single source of truth for NixOS config)
	slog.Debug("Copying nixconfig.json")
	if err := config.CopyFile(config.NixDir+"nixconfig.json", essentialDir+"/nixconfig.json"); err != nil {
		slog.Error("Error copying nixconfig.json:", "err", err)
		return err
	}

	// Copy the latest immich db dump
	slog.Debug("Copying latest immich db dump")
	cmd := exec.Command("sh", "-c", fmt.Sprintf(`cd /tank/immich/backups && cp "$(ls -t /tank/immich/backups/ | head -n 1)" %s/"$(ls -t /tank/immich/backups/ | head -n 1)"`, essentialDir))
	if err := cmd.Run(); err != nil {
		slog.Error("Error copying latest immich db dump:", "err", err)
		return err
	}

	// ============= SUPPLEMENTAL FILES (makes restore easier - no rebuild needed) =============
	slog.Debug("Backing up supplemental files")

	// Copy all .nix files (allows direct restore without rebuilding modules)
	slog.Debug("Copying .nix files")
	cmd = exec.Command("bash", "-c", fmt.Sprintf("cp %s*.nix %s/", config.NixDir, nixFilesDir))
	if err := cmd.Run(); err != nil {
		slog.Warn("Warning: Error copying .nix files (may not exist yet):", "err", err)
		// Don't fail the backup if .nix files don't exist - nixconfig.json is sufficient
	}

	// Copy docker-compose.yml and .env
	slog.Debug("Copying docker-compose.yml")
	if err := config.CopyFile(config.ImmichDir+"docker-compose.yml", supplementalDir+"/docker-compose.yml"); err != nil {
		slog.Error("Error copying docker-compose.yml:", "err", err)
		return err
	}

	slog.Debug("Copying .env")
	if err := config.CopyFile(config.ImmichDir+".env", supplementalDir+"/.env"); err != nil {
		slog.Error("Error copying .env:", "err", err)
		return err
	}

	// Copy immich-config.json
	slog.Debug("Copying immich-config.json")
	if err := config.CopyFile(config.TankImmich+"immich-config.json", essentialDir+"/immich-config.json"); err != nil {
		slog.Error("Error copying immich-config.json:", "err", err)
		return err
	}

	// ============= README =============
	slog.Debug("Adding readme file")
	readmeContent := `Immich Server Backup - Restore Instructions

This backup contains everything needed to restore your Immich server.

FOLDER STRUCTURE:
  essential/
    - immich-config.json   : Immich application settings
    - nixconfig.json       : NixOS configuration (required)
    - *.sql.gz             : Database dump (required)

  supplemental/
    - nix-files/*.nix      : NixOS module files (optional - can be regenerated)
    - docker-compose.yml   : Immich container config
    - .env                 : Immich environment variables

RESTORE PRIORITY:
  1. ESSENTIAL files are the minimum needed to restore your server
  2. SUPPLEMENTAL files make restoration faster (no need to rebuild)

For detailed restore instructions, visit:
https://github.com/rickyhaase/nixos-immich-webui/docs/restore-from-backup

Generated: ` + time.Now().Format("2006-01-02 15:04:05")

	if err := os.WriteFile(tempDir+"/readme.txt", []byte(readmeContent), 0644); err != nil {
		slog.Error("Error writing readme file:", "err", err)
		return err
	}

	// ============= ZIP AND SAVE =============
	// Create backup directory on the USB disk
	configBackupDir := backupDir + "/config"
	slog.Debug("Creating backup directory on USB disk", "configBackupDir", configBackupDir)
	if err := os.MkdirAll(configBackupDir, 0755); err != nil {
		slog.Error("Error creating backup directory on USB disk:", "err", err)
		return err
	}

	// Zip the backup files and add to USB disk
	zipFileName := fmt.Sprintf("\"%s/config-%s.zip\"", configBackupDir, time.Now().Format("2006-01-02"))
	cmd = exec.Command("bash", "-c", fmt.Sprintf("cd %s && zip -r %s .", tempDir, zipFileName))
	if err := cmd.Run(); err != nil {
		slog.Error("Error zipping backup files:", "err", err)
		return err
	}

	// Remove temporary files
	slog.Debug("Removing temporary files", "tempDir", tempDir)
	cmd = exec.Command("bash", "-c", fmt.Sprintf("rm -rf %s/*", tempDir))
	if err := cmd.Run(); err != nil {
		slog.Error("Error removing temporary files:", "err", err)
		return err
	}

	slog.Info("Config backup completed successfully")
	return nil
}

// backupLibrary backs up the Immich photo library using rsync
func (s *BackupService) backupLibrary(backupDir string, verifyChecksum bool) error {
	slog.Debug("Starting rsync for library backup", "source", "/tank/immich/library", "destination", backupDir, "verify", verifyChecksum)

	// Build rsync arguments
	args := []string{"-a", "--info=progress2", "--delete"}
	if verifyChecksum {
		args = append(args, "--checksum")
		slog.Info("Verification enabled - using checksums for integrity check")
	}
	args = append(args, "/tank/immich/library", backupDir)

	slog.Debug("rsync", "args", strings.Join(args, " "))

	cmd := exec.Command("rsync", args...)

	// Create a pipe to capture stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("Error creating stdout pipe:", "err", err)
		return err
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		slog.Error("Error starting rsync:", "err", err)
		return err
	}

	// Parse rsync progress output
	// Format: "    123,456,789  45%  123.45MB/s    0:12:34 (xfr#123, to-chk=456/789)"
	// We need to handle commas and extract both percentage and file counts
	buf := make([]byte, 4096)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			line := string(buf[:n])
			slog.Debug("Rsync output", "line", line)

			// Look for percentage (format: " 45%")
			if strings.Contains(line, "%") {
				// Find percentage value - it's between spaces and %
				parts := strings.Fields(line) // Split by whitespace
				for _, part := range parts {
					if strings.HasSuffix(part, "%") {
						percentStr := strings.TrimSuffix(part, "%")
						var pct int
						if _, parseErr := fmt.Sscanf(percentStr, "%d", &pct); parseErr == nil {
							// Map rsync 0-100% to our 10-95% range
							adjustedPercent := 10 + int(float64(pct)*0.85)

							s.stateMu.Lock()
							s.state.ProgressPercent = adjustedPercent
							s.state.CurrentStep = fmt.Sprintf("Syncing library (%d%%)", pct)
							s.stateMu.Unlock()

							slog.Debug("Rsync progress updated", "percent", pct, "adjustedPercent", adjustedPercent)
						}
					}
				}
			}

			// Look for file transfer info: "to-chk=remaining/total"
			if strings.Contains(line, "to-chk=") {
				// Extract file counts from "to-chk=remaining/total"
				start := strings.Index(line, "to-chk=")
				if start >= 0 {
					remaining := 0
					total := 0
					substring := line[start:]
					if _, scanErr := fmt.Sscanf(substring, "to-chk=%d/%d", &remaining, &total); scanErr == nil {
						processed := int64(total - remaining)

						s.stateMu.Lock()
						s.state.ProcessedFiles = processed
						s.state.TotalFiles = int64(total)
						s.stateMu.Unlock()

						slog.Debug("Rsync file progress", "processed", processed, "total", total)
					}
				}
			}
		}
		if err != nil {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		slog.Error("Error running rsync for library backup:", "err", err)
		return err
	}

	slog.Info("Library backup completed successfully")
	return nil
}

// unmountDisk safely unmounts the backup disk
func (s *BackupService) unmountDisk(disk string) error {
	slog.Debug("Unmounting disk", "disk", disk)
	unmountCmd := exec.Command("udisksctl", "unmount", "-b", "/dev/"+disk)
	err := unmountCmd.Run()
	if err != nil {
		slog.Error("Error unmounting disk:", "err", err)
		return err
	}
	slog.Info("Disk unmounted successfully")
	return nil
}

// LoadBackupHistory loads the backup history from JSON file
func LoadBackupHistory() (*BackupHistory, error) {
	slog.Debug("LoadBackupHistory()", "path", config.BackupHistoryFile)

	// If file doesn't exist, return empty history
	if _, err := os.Stat(config.BackupHistoryFile); os.IsNotExist(err) {
		slog.Debug("Backup history file does not exist, returning empty history")
		return &BackupHistory{Backups: []BackupHistoryEntry{}}, nil
	}

	data, err := os.ReadFile(config.BackupHistoryFile)
	if err != nil {
		slog.Error("Error reading backup history file", "err", err)
		return nil, err
	}

	var history BackupHistory
	if err := json.Unmarshal(data, &history); err != nil {
		slog.Error("Error unmarshaling backup history", "err", err)
		return nil, err
	}

	slog.Debug("Backup history loaded successfully", "entries", len(history.Backups))
	return &history, nil
}

// SaveBackupHistory saves the backup history to JSON file
func SaveBackupHistory(history *BackupHistory) error {
	slog.Debug("SaveBackupHistory()", "entries", len(history.Backups))

	// Auto-prune to keep last 100 entries
	if len(history.Backups) > 100 {
		history.Backups = history.Backups[len(history.Backups)-100:]
		slog.Debug("Pruned backup history to last 100 entries")
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		slog.Error("Error marshaling backup history", "err", err)
		return err
	}

	if err := os.WriteFile(config.BackupHistoryFile, data, 0644); err != nil {
		slog.Error("Error writing backup history file", "err", err)
		return err
	}

	slog.Debug("Backup history saved successfully")
	return nil
}

// AddBackupEntry adds a new entry to the backup history
func AddBackupEntry(entry BackupHistoryEntry) error {
	slog.Debug("AddBackupEntry()", "status", entry.Status, "timestamp", entry.Timestamp)

	history, err := LoadBackupHistory()
	if err != nil {
		return err
	}

	history.Backups = append(history.Backups, entry)

	return SaveBackupHistory(history)
}

// GetLastBackup returns the most recent backup entry, or nil if no backups exist
func GetLastBackup() (*BackupHistoryEntry, error) {
	history, err := LoadBackupHistory()
	if err != nil {
		return nil, err
	}

	if len(history.Backups) == 0 {
		return nil, nil
	}

	return &history.Backups[len(history.Backups)-1], nil
}
