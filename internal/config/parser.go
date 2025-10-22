package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
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

// Package-level mutexes for thread-safe configuration operations
var (
	nixConfigMu    sync.Mutex // Protects nixconfig.json read/write operations
	immichConfigMu sync.Mutex // Protects immich-config.json read/write operations
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
	nixConfigMu.Lock()
	defer nixConfigMu.Unlock()

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

// getImmichConfigUnsafe reads immich config without mutex protection (internal use only)
// This function should only be called from within immichConfigMu-protected contexts
func getImmichConfigUnsafe() (*ImmichConfig, error) {
	slog.Debug("getImmichConfigUnsafe()")
	file, err := os.Open(TankImmich + ImmichConfigFile)
	if err != nil {
		slog.Debug("| Error opening immich config file |", "err", err)
		return nil, err
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		slog.Debug("| Error reading immich config file contents |", "err", err)
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var immichConfig ImmichConfig
	if err := json.Unmarshal(byteValue, &immichConfig); err != nil {
		slog.Debug("| Error unmarshaling immich config JSON |", "err", err)
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return &immichConfig, nil
}

// GetImmichConfig reads and parses the Immich configuration JSON file
func GetImmichConfig() (*ImmichConfig, error) {
	immichConfigMu.Lock()
	defer immichConfigMu.Unlock()

	return getImmichConfigUnsafe()
}

// SetImmichEmail updates the Immich email/SMTP configuration
func SetImmichEmail(email string, password string) error {
	immichConfigMu.Lock()
	defer immichConfigMu.Unlock()

	slog.Debug("setImmichEmail()")
	immichConfig, err := getImmichConfigUnsafe()
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

	return switchImmichConfigJSON()
}

// SetMLModel updates the Immich machine learning CLIP model
func SetMLModel(modelName string) error {
	immichConfigMu.Lock()
	defer immichConfigMu.Unlock()

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

	immichConfig, err := getImmichConfigUnsafe()
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

	return switchImmichConfigJSON()
}

// SaveConfigJSON writes ConfigVariables to JSON file with .tmp extension
func SaveConfigJSON(cfg *ConfigVariables) error {
	nixConfigMu.Lock()
	defer nixConfigMu.Unlock()

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

// SwitchConfigJSON atomically backs up current config and activates temp config
// This function handles the critical operation of replacing the active config file
func SwitchConfigJSON() error {
	nixConfigMu.Lock()
	defer nixConfigMu.Unlock()

	slog.Debug("SwitchConfigJSON()")
	configPath := NixDir + NixConfigFile
	backupPath := NixDir + NixConfigFile + ".old"
	tmpPath := NixDir + NixConfigFile + TempSuffix

	slog.Info("Backing up nixconfig.json to nixconfig.json.old...")
	if err := copyFileUnsafe(configPath, backupPath); err != nil {
		slog.Debug("Error backing up JSON config file", "err", err)
		return err
	}

	slog.Info("Replacing nixconfig.json with nixconfig.json.tmp...")
	if err := copyFileUnsafe(tmpPath, configPath); err != nil {
		slog.Debug("Error replacing JSON config file", "err", err)
		return err
	}

	slog.Info("JSON configuration file switch complete.")
	return nil
}

// switchImmichConfigJSON backs up current immich config and replaces with temp config
// Must be called within immichConfigMu lock (internal use only)
func switchImmichConfigJSON() error {
	slog.Debug("switchImmichConfigJSON()")
	configFile := TankImmich + ImmichConfigFile
	backupFile := TankImmich + ImmichConfigFile + ".old"
	tmpFile := TankImmich + ImmichConfigFile + TempSuffix

	slog.Info("Backing up immich-config.json to immich-config.json.old...")
	if err := copyFileUnsafe(configFile, backupFile); err != nil {
		slog.Debug("Error backing up immich config file", "err", err)
		return err
	}

	slog.Info("Replacing immich-config.json from immich-config.json.tmp...")
	if err := copyFileUnsafe(tmpFile, configFile); err != nil {
		slog.Debug("Error replacing immich config file", "err", err)
		return err
	}

	slog.Info("Immich configuration file switch complete.")
	return nil
}

// copyFileUnsafe performs file copy without mutex protection (internal use only)
// This function should only be called from within mutex-protected contexts
func copyFileUnsafe(src, dst string) error {
	slog.Debug("copyFileUnsafe()", "src", src, "dst", dst)
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

// isNixConfigFile checks if a path refers to a NixOS configuration file
func isNixConfigFile(path string) bool {
	return strings.Contains(path, NixConfigFile)
}

// isImmichConfigFile checks if a path refers to an Immich configuration file
func isImmichConfigFile(path string) bool {
	return strings.Contains(path, ImmichConfigFile)
}

// CopyFile copies a file from src to dst with appropriate mutex protection
// Automatically determines which mutex to use based on file paths
func CopyFile(src, dst string) error {
	// Determine which resource we're copying and acquire appropriate mutex
	if isNixConfigFile(src) || isNixConfigFile(dst) {
		nixConfigMu.Lock()
		defer nixConfigMu.Unlock()
	} else if isImmichConfigFile(src) || isImmichConfigFile(dst) {
		immichConfigMu.Lock()
		defer immichConfigMu.Unlock()
	}

	return copyFileUnsafe(src, dst)
}
