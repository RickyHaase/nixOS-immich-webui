package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"time"
)

const (
	NixDir      string = "test/nixos/"           // to actually modify the nix config used by the system, this const needs to be set for "/etc/nixos/"
	ConfigFile  string = "nixconfig.json"       // JSON configuration file
	ImmichDir   string = "/tank/immich-config/"  // docker-compose.yml and .env stored on tank dataset for backup protection
	TankImmich  string = "test/tank/immich/"     // really only for immich-config.json. Not certain where this will end up in the end
)


// ParseBool converts string to boolean with error handling
func ParseBool(value string) bool {
	slog.Debug("parseBool(string)", "string", value)
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		slog.Error("| Error parsing boolean value - defaulting to False |", "err", err)
		return false
	}
	return boolValue
}

// GetLowerUpper calculates 30min and 60min time offsets from given time string
func GetLowerUpper(timeStr string) (string, string, error) {
	slog.Debug("getLowerUpper()")
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		slog.Debug("Error parsing time:", "err", err)
		return "", "", err
	}

	t1 := t.Add(30 * time.Minute)
	t2 := t.Add(time.Hour)

	newTimeStr1 := t1.Format("15:04")
	newTimeStr2 := t2.Format("15:04")

	return newTimeStr1, newTimeStr2, nil
}

// LoadCurrentConfigJSON reads and parses the current JSON configuration
func LoadCurrentConfigJSON() (*ConfigVariables, error) {
	slog.Debug("LoadCurrentConfigJSON()")
	configPath := NixDir + ConfigFile
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		slog.Debug("Error reading nixconfig.json:", "err", err)
		return nil, err
	}

	var config ConfigVariables
	if err := json.Unmarshal(data, &config); err != nil {
		slog.Debug("Error parsing JSON config:", "err", err)
		return nil, err
	}

	// Email configuration is managed separately via /email endpoint and immich-config.json
	// No need to include email data in NixOS configuration
	
	slog.Debug("Loaded JSON config", "timeZone", config.System.TimeZone, "tailscale", config.RemoteAccess.Tailscale.Enable)
	return &config, nil
}


// GetImmichConfig reads and parses the Immich configuration JSON file
func GetImmichConfig() (*ImmichConfig, error) {
	slog.Debug("getImmichConfig()")
	file, err := os.Open(TankImmich + "immich-config.json")
	if err != nil {
		slog.Debug("| Error opening immich config file |", "err", err)
		return nil, err
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)

	var immichConfig ImmichConfig
	json.Unmarshal(byteValue, &immichConfig)

	return &immichConfig, nil
}

// SetImmichConfig updates the Immich configuration with email settings
func SetImmichConfig(email string, password string) error {
	slog.Debug("setImmichConfig()")
	// NOT using templating because we've got all the JSON we need... should cut down on errors but we need a "default" value somewhere
	immichConfig, err := GetImmichConfig()
	if err != nil {
		slog.Debug("Error reading immich config file", "err", err)
		return err
	}

	immichConfig.Notifications.SMTP.From = "Immich Server <" + email + ">"
	immichConfig.Notifications.SMTP.Transport.Username = email
	immichConfig.Notifications.SMTP.Transport.Password = password

	if email == "" || password == "" {
		immichConfig.Notifications.SMTP.Enabled = false
	} else {
		immichConfig.Notifications.SMTP.Enabled = true
	}

	b, err := json.MarshalIndent(immichConfig, "", "  ")
	if err != nil {
		slog.Debug("Error generating JSON", "err", err)
		return err
	}

	slog.Debug(string(b))

	fileName := TankImmich + "immich-config.tmp"

	if err := os.WriteFile(fileName, b, 0644); err != nil {
		slog.Debug("Error writing to file:", "err", err)
		return err
	}

	configFile := TankImmich + "immich-config.json"

	return CopyFile(fileName, configFile)
}

// SaveConfigJSON writes ConfigVariables to JSON file with .tmp extension
func SaveConfigJSON(cfg *ConfigVariables) error {
	slog.Debug("SaveConfigJSON()")
	
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		slog.Debug("Error marshaling JSON config:", "err", err)
		return err
	}

	tmpPath := NixDir + ConfigFile + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		slog.Debug("Error writing JSON config tmp file:", "err", err)
		return err
	}

	slog.Debug("JSON config saved to tmp file", "path", tmpPath)
	return nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	slog.Debug("CopyFile()")
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy data from %s to %s: %w", src, dst, err)
	}

	return nil
}