package system

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/RickyHaase/nixOS-immich-webui/internal/config"
)

// SwitchConfig backs up current config and replaces with temp config
func SwitchConfig() error {
	slog.Debug("switchConfig()")
	configPath := config.NixDir + "configuration.nix"
	backupPath := config.NixDir + "configuration.old"
	tmpPath := config.NixDir + "configuration.tmp"

	slog.Info("Backing up configuration.nix to configuration.old...")
	if err := config.CopyFile(configPath, backupPath); err != nil {
		slog.Debug("Error backing up config file", "err", err)
		return err
	}

	slog.Info("Replacing configuration.nix with configuration.tmp...")
	if err := config.CopyFile(tmpPath, configPath); err != nil {
		slog.Debug("Error replacing config file", "err", err)
		return err
	}

	slog.Info("Configuration file swtich complete.")
	return nil
}

// ApplyChanges runs nixos-rebuild switch to apply configuration changes
func ApplyChanges() error {
	slog.Debug("applyChanges()")
	cmd := exec.Command("nixos-rebuild", "switch")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		slog.Debug("| error running 'nixos-rebuild switch' |", "err", err)
		return fmt.Errorf("failed to execute nixos-rebuild: %w", err)
	}

	slog.Info("NixOS rebuild completed successfully.")
	return nil
}

// GetStatus returns the status of immich-app.service
func GetStatus() string {
	slog.Debug("getStatus()")
	cmd := exec.Command("systemctl", "show", "-p", "ActiveState", "--value", "immich-app.service")
	output, err := cmd.Output()
	if err != nil {
		slog.Error("| Error getting status of immich-app.service |", "err", err)
		return "Error getting status"
	}

	status := string(output)
	switch status {
	case "active\n":
		return "Running"
	case "inactive\n":
		return "Stopped"
	default:
		slog.Error("| Unexpected status of immich-app.service |", "err", err)
		return "Error getting status"
	}
}

// ImmichService controls the immich-app.service (start, stop, etc.)
func ImmichService(command string) error {
	slog.Debug("immichService(string)", "string", command)
	cmd := exec.Command("systemctl", command, "immich-app.service")
	err := cmd.Run()
	if err != nil {
		slog.Error("Error running %s against immich-app.service: %v", command, err)
		return err
	}
	return nil
}

// UpdateImmichContainer pulls new Immich container images
func UpdateImmichContainer() error {
	slog.Debug("updateImmichContainer()")
	path := config.ImmichDir + "docker-compose.yml"
	cmd := exec.Command("docker", "compose", "-f", path, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println(cmd)
	err := cmd.Run()
	if err != nil {
		slog.Debug("| Error executing 'docker compose pull' |", "cmd", cmd, "err", err)
		return fmt.Errorf("failed to pull new containers: %w", err)
	}

	slog.Info("compose pull completed successfully")
	return nil
}

// GetEligibleDisks returns a list of USB disks with exFAT partitions eligible for backup
func GetEligibleDisks() ([]config.EligibleDisk, error) {
	eligibleDisks := []config.EligibleDisk{}

	cmd := exec.Command("lsblk", "-J", "-o", "NAME,SIZE,FSTYPE,TRAN,MODEL,LABEL")
	out, err := cmd.Output()
	if err != nil {
		slog.Error("Error running lsblk:", "err", err)
		return eligibleDisks, err
	}

	var lsblkData config.LSBLKOutput
	if err := json.Unmarshal(out, &lsblkData); err != nil {
		slog.Error("Error parsing JSON:", "err", err)
		return eligibleDisks, err
	}

	// Filters out eligible disks (must be connected via usb AND be formatted exfat) before adding relevant info into an array of eligible disks
	for _, device := range lsblkData.BlockDevices {
		if device.Transport == "usb" {
			for _, part := range device.Children {
				if part.FSType == "exfat" {
					slog.Debug("Eligible Block Drive Found", "Partition Name", part.Label, "Partition Size", part.Size, "Device Model", device.Model, "Device Name", part.Name)
					disk := config.EligibleDisk{part.Label, part.Size, device.Model, part.Name}
					eligibleDisks = append(eligibleDisks, disk)
				}
			}
		}
	}

	return eligibleDisks, nil
}

// PowerOff executes system poweroff command
func PowerOff() error {
	slog.Info("Received Poweroff Request")

	cmd := exec.Command("poweroff")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		slog.Error("| Error executing poweroff |", "err", err)
		return fmt.Errorf("failed to execute poweroff: %w", err)
	}
	return nil
}

// Reboot executes system reboot command
func Reboot() error {
	slog.Info("Received Reboot Request")

	cmd := exec.Command("reboot")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		slog.Error("| Error executing reboot |", "err", err)
		return fmt.Errorf("failed to execute reboot: %w", err)
	}
	return nil
}