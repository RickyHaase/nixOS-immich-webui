package services

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/RickyHaase/nixOS-immich-webui/internal/config"
)

// BackupService handles backup operations
type BackupService struct{}

// NewBackupService creates a new backup service
func NewBackupService() *BackupService {
	return &BackupService{}
}

// BackupToUSB performs a complete backup to the specified USB disk
func (s *BackupService) BackupToUSB(disk string) (string, error) {
	slog.Debug("backupToUSB() - Start", "disk", disk)

	// Check if /dev/[disk] is mounted
	mountCheckCmd := exec.Command("lsblk", "-no", "MOUNTPOINT", "/dev/"+disk)
	mountPoint, err := mountCheckCmd.Output()
	if err != nil {
		slog.Error("Error checking if disk is mounted:", "err", err)
		return "", err
	}
	slog.Debug("Mount point check output", "mountPoint", string(mountPoint))

	if len(mountPoint) == 1 && mountPoint[0] == 10 { // Checks that the mountpoint is just an empty line
		slog.Debug("Disk is not mounted, attempting to mount", "disk", disk)
		mountCmd := exec.Command("udisksctl", "mount", "-b", "/dev/"+disk)
		err := mountCmd.Run()
		if err != nil {
			slog.Error("Error mounting disk:", "err", err)
			return "", err
		}

		mountCheckCmd = exec.Command("lsblk", "-no", "MOUNTPOINT", "/dev/"+disk)
		mountPoint, err = mountCheckCmd.Output()
		if err != nil {
			slog.Error("Error re-checking mount point:", "err", err)
			return "", err
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
		return "", err
	}

	// =============== Config Backups ===================
	if err := s.backupConfigs(backupDir); err != nil {
		return "", err
	}

	// ===================Library Backup with Rsync==========================
	if err := s.backupLibrary(backupDir); err != nil {
		return "", err
	}

	// ================= Backups done - can unmount disk =============
	if err := s.unmountDisk(disk); err != nil {
		return "", err
	}

	slog.Debug("backupToUSB() - End")
	return "backup complete", nil
}

// backupConfigs backs up configuration files and database dumps
func (s *BackupService) backupConfigs(backupDir string) error {
	// Create a temporary directory for the backup files
	tempDir := "/root/tempconfig"
	slog.Debug("Creating temporary directory for backup files", "tempDir", tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		slog.Error("Error creating temporary directory:", "err", err)
		return err
	}

	// Copy the latest immich db dump
	slog.Debug("Copying latest immich db dump")
	cmd := exec.Command("sh", "-c", fmt.Sprintf(`cd /tank/immich/backups && cp "$(ls -t /tank/immich/backups/ | head -n 1)" %s/"$(ls -t /tank/immich/backups/ | head -n 1)"`, tempDir))
	if err := cmd.Run(); err != nil {
		slog.Error("Error copying latest immich db dump:", "err", err)
		return err
	}

	// Copy the current immich-config.json
	slog.Debug("Copying immich-config.json")
	if err := config.CopyFile(config.TankImmich+"immich-config.json", tempDir+"/immich-config.json"); err != nil {
		slog.Error("Error copying immich-config.json:", "err", err)
		return err
	}

	// Copy nixos config folder
	slog.Debug("Copying nixos config folder")
	cmd = exec.Command("cp", "-r", "/etc/nixos", tempDir+"/nixos")
	if err := cmd.Run(); err != nil {
		slog.Error("Error copying nixos config folder:", "err", err)
		return err
	}

	// Copy immich compose
	slog.Debug("Copying immich compose")
	cmd = exec.Command("cp", "-r", config.ImmichDir, tempDir+"/immich-app")
	if err := cmd.Run(); err != nil {
		slog.Error("Error copying immich compose:", "err", err)
		return err
	}

	// Add readme
	slog.Debug("Adding readme file")
	readmeContent := "For restore instructions, go to https://github.com/rickyhaase/nixos-immich-webui/docs/restore-from-backup"
	if err := os.WriteFile(tempDir+"/readme.txt", []byte(readmeContent), 0644); err != nil {
		slog.Error("Error writing readme file:", "err", err)
		return err
	}

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

	return nil
}

// backupLibrary backs up the Immich photo library using rsync
func (s *BackupService) backupLibrary(backupDir string) error {
	slog.Debug("Starting rsync for library backup", "source", "/tank/immich/library", "destination", backupDir)
	cmd := exec.Command("rsync", "-a", "--info=progress2", "--delete", "/tank/immich/library", backupDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
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