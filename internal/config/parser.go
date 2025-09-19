package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"time"
)

const (
	NixDir      string = "test/nixos/"           // to actually modify the nix config used by the system, this const needs to be set for "/etc/nixos/"
	ImmichDir   string = "/tank/immich-config/"  // docker-compose.yml and .env stored on tank dataset for backup protection
	TankImmich  string = "test/tank/immich/"     // really only for immich-config.json. Not certain where this will end up in the end
)

// Helper function to parse boolean values from the configuration file
func parseBooleanSetting(fileContent []byte, setting string) (bool, error) {
	slog.Debug("parseBooleanSetting", "setting", setting)
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*%s\s*=\s*(true|false)\s*;`, setting))
	match := re.FindSubmatch(fileContent)
	if match == nil {
		slog.Debug("No Match Found", "setting", setting)
		return false, fmt.Errorf("%s not found", setting)
	}
	return string(match[1]) == "true", nil
}

// Helper function to parse string values from the configuration file
func parseStringSetting(fileContent []byte, setting string) (string, error) {
	slog.Debug("parseStringSetting", "setting", setting)
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*%s\s*=\s*"(.*?)"\s*;`, setting))
	match := re.FindSubmatch(fileContent)
	if match == nil {
		slog.Debug("No Match Found", "setting", setting)
		return "", fmt.Errorf("%s not found", setting)
	}
	return string(match[1]), nil
}

// Helper function to parse Tailscale auth key from configuration file
func parseAuthKeySetting(fileContent []byte) (string, error) {
	slog.Debug("parseAuthKeySetting()")
	re := regexp.MustCompile(`\btskey-auth-[a-zA-Z0-9]+-[a-zA-Z0-9]+\b`)
	match := re.Find(fileContent)
	if match == nil {
		slog.Debug("No Match Found for authkey")
		return "", fmt.Errorf("tskey-auth not found")
	}
	return string(match), nil
}

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

// LoadCurrentConfig reads and parses the current NixOS configuration
func LoadCurrentConfig() (*NixConfig, error) {
	slog.Debug("loadCurrentConfig()")
	file, err := os.ReadFile(NixDir + "configuration.nix")
	if err != nil {
		slog.Debug("Error opening file:", "err", err)
		return nil, err
	}

	config := NixConfig{}

	// Parse the relevant values out of the settings in the config file
	config.TimeZone, err = parseStringSetting(file, "time.timeZone")
	if err != nil {
		slog.Debug("Error parsing TimeZone:", "err", err)
		return nil, err
	}

	config.AutoUpgrade, err = parseBooleanSetting(file, "system.autoUpgrade.enable")
	if err != nil {
		slog.Debug("Error parsing AutoUpgrade Enable:", "err", err)
		return nil, err
	}

	config.UpgradeTime, err = parseStringSetting(file, "system.autoUpgrade.dates")
	if err != nil {
		slog.Debug("Error parsing UpdgradeTime:", "err", err)
		return nil, err
	}

	config.Tailscale, err = parseBooleanSetting(file, "services.tailscale.enable")
	if err != nil {
		slog.Debug("Error parsing Tailscale Enable", "err", err)
		return nil, err
	}

	config.TSAuthkey, err = parseAuthKeySetting(file)
	if err != nil {
		slog.Debug("Error parsing Tailscale AuthKey", "err", err)
		return nil, err
	}

	// Parse settings out of immich-config.json
	immich, err := GetImmichConfig()
	if err != nil {
		slog.Debug("Error parsing Immich Config", "err", err)
		return nil, err
	}

	config.Email = immich.Notifications.SMTP.Transport.Username

	if immich.Notifications.SMTP.Transport.Password != "" {
		slog.Debug("IF was met")
		config.EmailPass = true
	} else {
		slog.Debug("ELSE was met")
		config.EmailPass = false
	}
	slog.Debug("Password Boolean", "EmailPass", config.EmailPass)

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