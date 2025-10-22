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

// Environment-agnostic file name constants
const (
	NixConfigFile    string = "nixconfig.json"     // NixOS JSON configuration file
	ImmichConfigFile string = "immich-config.json" // Immich configuration file
	TempSuffix       string = ".tmp"               // Temporary file suffix
)

// Environment-specific path constants (NixDir, ImmichDir, TankImmich) are defined in:
// - paths_dev.go (when built with -tags dev)
// - paths_prod.go (when built without tags, production default)

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
	configPath := NixDir + NixConfigFile

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
	file, err := os.Open(TankImmich + ImmichConfigFile)
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

	fileName := TankImmich + ImmichConfigFile + TempSuffix

	if err := os.WriteFile(fileName, b, 0644); err != nil {
		slog.Debug("Error writing to file:", "err", err)
		return err
	}

	configFile := TankImmich + ImmichConfigFile

	return CopyFile(fileName, configFile)
}

// SetMLModel updates the Immich machine learning CLIP model
func SetMLModel(modelName string) error {
	slog.Debug("setMLModel()", "modelName", modelName)

	// Validate model name - only allow specific models
	validModels := map[string]bool{
		"ViT-B-32__openai":                true,
		"ViT-B-16-SigLIP__webli":          true,
		"ViT-SO400M-14-SigLIP-384__webli": true,
	}

	if !validModels[modelName] {
		slog.Error("| Invalid ML model name |", "modelName", modelName)
		return fmt.Errorf("invalid model name: %s", modelName)
	}

	immichConfig, err := GetImmichConfig()
	if err != nil {
		slog.Debug("Error reading immich config file", "err", err)
		return err
	}

	immichConfig.MachineLearning.Clip.ModelName = modelName

	b, err := json.MarshalIndent(immichConfig, "", "  ")
	if err != nil {
		slog.Debug("Error generating JSON", "err", err)
		return err
	}

	slog.Debug(string(b))

	fileName := TankImmich + ImmichConfigFile + TempSuffix

	if err := os.WriteFile(fileName, b, 0644); err != nil {
		slog.Debug("Error writing to file:", "err", err)
		return err
	}

	configFile := TankImmich + ImmichConfigFile

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

	tmpPath := NixDir + NixConfigFile + TempSuffix
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
