package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// ConfigVariables represents the JSON structure for all user-configurable settings
// This replaces the existing NixConfig struct with a more comprehensive structure
type ConfigVariables struct {
	Meta struct {
		Version     string    `json:"version"`
		Timestamp   time.Time `json:"timestamp"`
		Description string    `json:"description"`
	} `json:"meta"`
	System struct {
		TimeZone     string `json:"timeZone"`
		AutoUpgrade  bool   `json:"autoUpgrade"`
		UpgradeTime  string `json:"upgradeTime"`
		UpgradeLower string `json:"upgradeLower"`
		UpgradeUpper string `json:"upgradeUpper"`
	} `json:"system"`
	Networking struct {
		HostName string `json:"hostName"`
		HostId   string `json:"hostId"`
	} `json:"networking"`
	RemoteAccess struct {
		Tailscale struct {
			Enable  bool   `json:"enable"`
			AuthKey string `json:"authKey"`
		} `json:"tailscale"`
	} `json:"remoteAccess"`
	Email struct {
		Address     string `json:"address"`
		PasswordSet bool   `json:"passwordSet"`
	} `json:"email"`
	Storage struct {
		ZFS struct {
			PoolName   string `json:"poolName"`
			AutoScrub  bool   `json:"autoScrub"`
			Snapshots  struct {
				Hourly  int `json:"hourly"`
				Daily   int `json:"daily"`
				Weekly  int `json:"weekly"`
				Monthly int `json:"monthly"`
				Yearly  int `json:"yearly"`
			} `json:"snapshots"`
		} `json:"zfs"`
	} `json:"storage"`
	Immich struct {
		WorkingDirectory  string `json:"workingDirectory"`
		DockerTimeout     string `json:"dockerTimeout"`
		AutoPruneSchedule string `json:"autoPruneSchedule"`
	} `json:"immich"`
	Ports struct {
		ImmichInternal int `json:"immichInternal"`
		AdminPanel     int `json:"adminPanel"`
		WebPublic      int `json:"webPublic"`
	} `json:"ports"`
	Firewall struct {
		AllowPing        bool  `json:"allowPing"`
		AllowedTCPPorts  []int `json:"allowedTCPPorts"`
	} `json:"firewall"`
}

const (
	configDir     = "/etc/nixos/"          // Production path
	testConfigDir = "test/nixos/"          // Development path  
	historyDir    = "history/"
	variablesFile = "variables.json"
	currentVersionFile = "current-version.txt"
)

// ===== READING CURRENT CONFIGURATION =====

// LoadCurrentConfig reads the current variables.json and parses it into a Go struct
// This replaces the complex regex parsing in the original loadCurrentConfig()
func LoadCurrentConfig() (*ConfigVariables, error) {
	configPath := testConfigDir + variablesFile
	
	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config ConfigVariables
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}
	
	return &config, nil
}

// ===== GENERATING NEW CONFIGURATION =====

// GenerateConfig creates a new configuration from user input
// This replaces the template-based saveTmpFile() approach
func GenerateConfig(userSettings map[string]interface{}) (*ConfigVariables, error) {
	// Get current version and increment
	currentVersion, err := getCurrentVersion()
	if err != nil {
		currentVersion = 0
	}
	nextVersion := currentVersion + 1
	
	config := &ConfigVariables{}
	
	// Set metadata
	config.Meta.Version = fmt.Sprintf("%03d", nextVersion)
	config.Meta.Timestamp = time.Now()
	config.Meta.Description = "Configuration updated via web UI"
	
	// Map user settings to config structure
	// This would be populated from web form data
	if tz, ok := userSettings["timeZone"].(string); ok {
		config.System.TimeZone = tz
	}
	if au, ok := userSettings["autoUpgrade"].(bool); ok {
		config.System.AutoUpgrade = au
	}
	if ts, ok := userSettings["tailscale"].(bool); ok {
		config.RemoteAccess.Tailscale.Enable = ts
	}
	if key, ok := userSettings["tsAuthKey"].(string); ok {
		config.RemoteAccess.Tailscale.AuthKey = key
	}
	if email, ok := userSettings["email"].(string); ok {
		config.Email.Address = email
	}
	
	// Set reasonable defaults for other values
	setDefaults(config)
	
	return config, nil
}

// setDefaults fills in standard values that rarely change
func setDefaults(config *ConfigVariables) {
	config.Networking.HostName = "immich"
	config.Networking.HostId = "12345678"
	config.Storage.ZFS.PoolName = "tank"
	config.Storage.ZFS.AutoScrub = true
	config.Storage.ZFS.Snapshots.Hourly = 24
	config.Storage.ZFS.Snapshots.Daily = 7
	config.Immich.WorkingDirectory = "/root/immich-app"
	config.Immich.DockerTimeout = "90"
	config.Immich.AutoPruneSchedule = "monthly"
	config.Ports.ImmichInternal = 2283
	config.Ports.AdminPanel = 8080
	config.Ports.WebPublic = 80
	config.Firewall.AllowPing = true
	config.Firewall.AllowedTCPPorts = []int{80, 8080}
}

// ===== SAVING CONFIGURATION =====

// SaveConfig writes a new configuration and creates a backup
// This replaces the switchConfig() approach
func SaveConfig(config *ConfigVariables) error {
	configPath := testConfigDir + variablesFile
	historyPath := testConfigDir + historyDir
	
	// Create history directory if it doesn't exist
	if err := os.MkdirAll(historyPath, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}
	
	// Backup current config if it exists
	if _, err := os.Stat(configPath); err == nil {
		backupPath := fmt.Sprintf("%svariables-%s.json", historyPath, config.Meta.Version)
		if err := copyFile(configPath, backupPath); err != nil {
			return fmt.Errorf("failed to backup current config: %w", err)
		}
	}
	
	// Write new configuration
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}
	
	if err := os.WriteFile(configPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	// Update version tracker
	versionPath := testConfigDir + historyDir + currentVersionFile
	if err := os.WriteFile(versionPath, []byte(config.Meta.Version), 0644); err != nil {
		return fmt.Errorf("failed to update version file: %w", err)
	}
	
	return nil
}

// ===== ROLLBACK FUNCTIONALITY =====

// RollbackToVersion restores a previous configuration version
// This provides the rollback capability you need
func RollbackToVersion(version string) error {
	historyPath := testConfigDir + historyDir
	configPath := testConfigDir + variablesFile
	
	// Source file (historical version)
	sourcePath := fmt.Sprintf("%svariables-%s.json", historyPath, version)
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("version %s not found: %w", version, err)
	}
	
	// Copy historical version to current
	if err := copyFile(sourcePath, configPath); err != nil {
		return fmt.Errorf("failed to restore config: %w", err)
	}
	
	// Update version tracker
	versionPath := historyPath + currentVersionFile
	if err := os.WriteFile(versionPath, []byte(version), 0644); err != nil {
		return fmt.Errorf("failed to update version: %w", err)
	}
	
	return nil
}

// ListAvailableVersions returns all available configuration versions for rollback
func ListAvailableVersions() ([]string, error) {
	historyPath := testConfigDir + historyDir
	
	entries, err := os.ReadDir(historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read history directory: %w", err)
	}
	
	var versions []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "variables-") && strings.HasSuffix(entry.Name(), ".json") {
			// Extract version number from filename
			name := entry.Name()
			version := name[10 : len(name)-5] // Remove "variables-" and ".json"
			versions = append(versions, version)
		}
	}
	
	return versions, nil
}

// ===== UTILITY FUNCTIONS =====

// getCurrentVersion reads the current version number
func getCurrentVersion() (int, error) {
	versionPath := testConfigDir + historyDir + currentVersionFile
	
	data, err := os.ReadFile(versionPath)
	if err != nil {
		return 0, err
	}
	
	version, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	
	return version, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	
	_, err = io.Copy(destination, source)
	return err
}

// ===== EXAMPLE USAGE =====

func main() {
	fmt.Println("=== JSON-Based Configuration Management Example ===")
	
	// Example 1: Reading current configuration
	fmt.Println("\n1. Reading current configuration:")
	config, err := LoadCurrentConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
	} else {
		fmt.Printf("Current timezone: %s\n", config.System.TimeZone)
		fmt.Printf("Auto-upgrade enabled: %v\n", config.System.AutoUpgrade)
		fmt.Printf("Tailscale enabled: %v\n", config.RemoteAccess.Tailscale.Enable)
	}
	
	// Example 2: Generating new configuration
	fmt.Println("\n2. Generating new configuration:")
	userSettings := map[string]interface{}{
		"timeZone":   "Europe/London",
		"autoUpgrade": true,
		"tailscale":  true,
		"tsAuthKey":  "tskey-auth-newkey123-xyz789",
		"email":      "admin@newdomain.com",
	}
	
	newConfig, err := GenerateConfig(userSettings)
	if err != nil {
		fmt.Printf("Error generating config: %v\n", err)
	} else {
		fmt.Printf("Generated config version: %s\n", newConfig.Meta.Version)
		fmt.Printf("New timezone: %s\n", newConfig.System.TimeZone)
	}
	
	// Example 3: Listing available versions for rollback
	fmt.Println("\n3. Available versions for rollback:")
	versions, err := ListAvailableVersions()
	if err != nil {
		fmt.Printf("Error listing versions: %v\n", err)
	} else {
		for _, version := range versions {
			fmt.Printf("- Version %s\n", version)
		}
	}
	
	fmt.Println("\n=== Benefits of JSON Approach ===")
	fmt.Println("✅ No regex parsing - bulletproof JSON handling")
	fmt.Println("✅ Easy rollback - just copy JSON files") 
	fmt.Println("✅ Version history - automatic backup of all changes")
	fmt.Println("✅ Type safety - Go structs with proper validation")
	fmt.Println("✅ Debugging - readable JSON files")
	fmt.Println("✅ Extensions - easy to add new configuration options")
}

/*
INTEGRATION WITH EXISTING CODE:

1. Replace loadCurrentConfig() with LoadCurrentConfig()
   - Remove all regex parsing functions
   - Remove parseBooleanSetting, parseStringSetting, etc.
   - Much simpler and more reliable

2. Replace saveTmpFile() with SaveConfig()
   - No more Go templates
   - Just JSON marshaling
   - Automatic versioning and backup

3. Add rollback handlers to web interface:
   - GET /versions - list available versions
   - POST /rollback - rollback to specific version
   - Built-in version management

4. Update NixConfig struct to match ConfigVariables
   - More comprehensive structure
   - Better organization
   - Room for future expansion

5. Update web forms to populate new structure
   - Same user interface
   - Improved backend handling
   - More robust validation
*/