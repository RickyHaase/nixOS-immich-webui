package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultBackupDataDir = "/root/backup_data"
	DefaultConfigFile    = "system_config.yaml"
)

type BackupConfig struct {
	DataDir             string                    `yaml:"data_dir"`
	QualityTiers        []QualityTier            `yaml:"quality_tiers"`
	StorageLimitGB      int64                    `yaml:"storage_limit_gb"`
	JobRetentionDays    int                      `yaml:"job_retention_days"`
	LogLevel            string                   `yaml:"log_level"`
	ProcessingSettings  ProcessingSettings       `yaml:"processing_settings"`
	UserPreferences     UserPreferences          `yaml:"user_preferences"`
	mutex               sync.RWMutex             `yaml:"-"`
}

type QualityTier struct {
	Name                string    `yaml:"name"`
	AgeThresholdDays    int       `yaml:"age_threshold_days"`
	PhotoMaxResolution  int       `yaml:"photo_max_resolution"`
	PhotoMaxResolutionMP int      `yaml:"-"` // Calculated field for templates
	PhotoQuality        int       `yaml:"photo_quality"`
	VideoMaxHeight      int       `yaml:"video_max_height"`
	VideoMaxFPS         int       `yaml:"video_max_fps"`
	VideoCRF            int       `yaml:"video_crf"`
	MetadataLevel       string    `yaml:"metadata_level"`
}

type ProcessingSettings struct {
	MaxConcurrentJobs         int     `yaml:"max_concurrent_jobs"`
	TempDir                   string  `yaml:"temp_dir"`
	SpacePressureThreshold    float64 `yaml:"space_pressure_threshold"`
	SpacePressureThresholdPercent int `yaml:"-"` // Calculated field for templates
	QualityAdjustmentStep     int     `yaml:"quality_adjustment_step"`
}

type UserPreferences struct {
	EmailNotifications  bool              `yaml:"email_notifications"`
	FolderOverrides     map[string]string `yaml:"folder_overrides"`
	DateRangeExceptions []DateRange       `yaml:"date_range_exceptions"`
}

type DateRange struct {
	StartDate   time.Time `yaml:"start_date"`
	EndDate     time.Time `yaml:"end_date"`
	ForceTier   string    `yaml:"force_tier"`
	Description string    `yaml:"description"`
}

var (
	defaultConfig *BackupConfig
	configOnce    sync.Once
)

func GetDefaultConfig() *BackupConfig {
	configOnce.Do(func() {
		defaultConfig = &BackupConfig{
			DataDir:          DefaultBackupDataDir,
			StorageLimitGB:   100,
			JobRetentionDays: 30,
			LogLevel:         "info",
			QualityTiers: []QualityTier{
				{
					Name:                 "High Quality (0-12 months)",
					AgeThresholdDays:     365,
					PhotoMaxResolution:   12000000, // 12MP
					PhotoMaxResolutionMP: 12,
					PhotoQuality:         92,
					VideoMaxHeight:       1080,
					VideoMaxFPS:          60,
					VideoCRF:             20,
					MetadataLevel:        "full",
				},
				{
					Name:                 "Medium Quality (1-3 years)",
					AgeThresholdDays:     1095, // 3 years
					PhotoMaxResolution:   8000000, // 8MP
					PhotoMaxResolutionMP: 8,
					PhotoQuality:         88,
					VideoMaxHeight:       1080,
					VideoMaxFPS:          30,
					VideoCRF:             23,
					MetadataLevel:        "essential",
				},
				{
					Name:                 "Space Optimized (3+ years)",
					AgeThresholdDays:     999999, // effectively unlimited
					PhotoMaxResolution:   8000000, // 8MP
					PhotoMaxResolutionMP: 8,
					PhotoQuality:         80,
					VideoMaxHeight:       720,
					VideoMaxFPS:          30,
					VideoCRF:             26,
					MetadataLevel:        "minimal",
				},
			},
			ProcessingSettings: ProcessingSettings{
				MaxConcurrentJobs:            2,
				TempDir:                      "/tmp/backup_processing",
				SpacePressureThreshold:       0.9, // 90%
				SpacePressureThresholdPercent: 90,
				QualityAdjustmentStep:        2,
			},
			UserPreferences: UserPreferences{
				EmailNotifications:  false,
				FolderOverrides:     make(map[string]string),
				DateRangeExceptions: []DateRange{},
			},
		}
	})
	return defaultConfig
}

func LoadConfig(configPath string) (*BackupConfig, error) {
	if configPath == "" {
		configPath = filepath.Join(DefaultBackupDataDir, "config", DefaultConfigFile)
	}

	// If config file doesn't exist, create it with defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := GetDefaultConfig()
		if err := config.Save(configPath); err != nil {
			return nil, fmt.Errorf("creating default config: %w", err)
		}
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config BackupConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Calculate derived fields for templates
	config.ProcessingSettings.SpacePressureThresholdPercent = int(config.ProcessingSettings.SpacePressureThreshold * 100)
	
	for i := range config.QualityTiers {
		config.QualityTiers[i].PhotoMaxResolutionMP = config.QualityTiers[i].PhotoMaxResolution / 1000000
	}

	return &config, nil
}

func (c *BackupConfig) Save(configPath string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if configPath == "" {
		configPath = filepath.Join(c.DataDir, "config", DefaultConfigFile)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Atomic write using temporary file
	tempFile := configPath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("writing temp config file: %w", err)
	}

	if err := os.Rename(tempFile, configPath); err != nil {
		os.Remove(tempFile) // cleanup on failure
		return fmt.Errorf("moving temp config file: %w", err)
	}

	return nil
}

func (c *BackupConfig) Validate() error {
	if c.DataDir == "" {
		return fmt.Errorf("data_dir cannot be empty")
	}

	if c.StorageLimitGB <= 0 {
		return fmt.Errorf("storage_limit_gb must be positive")
	}

	if len(c.QualityTiers) == 0 {
		return fmt.Errorf("at least one quality tier must be defined")
	}

	// Validate quality tiers
	for i, tier := range c.QualityTiers {
		if tier.Name == "" {
			return fmt.Errorf("tier %d: name cannot be empty", i)
		}

		if tier.PhotoQuality < 1 || tier.PhotoQuality > 100 {
			return fmt.Errorf("tier %d: photo_quality must be between 1-100", i)
		}

		if tier.VideoCRF < 0 || tier.VideoCRF > 51 {
			return fmt.Errorf("tier %d: video_crf must be between 0-51", i)
		}

		if tier.MetadataLevel != "full" && tier.MetadataLevel != "essential" && tier.MetadataLevel != "minimal" {
			return fmt.Errorf("tier %d: metadata_level must be 'full', 'essential', or 'minimal'", i)
		}
	}

	return nil
}

func (c *BackupConfig) GetTierByAge(fileAge time.Duration) QualityTier {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	ageDays := int(fileAge.Hours() / 24)

	for _, tier := range c.QualityTiers {
		if ageDays <= tier.AgeThresholdDays {
			return tier
		}
	}

	// Return the last tier if age exceeds all thresholds
	return c.QualityTiers[len(c.QualityTiers)-1]
}

func (c *BackupConfig) EnsureDirectories() error {
	dirs := []string{
		c.DataDir,
		filepath.Join(c.DataDir, "jobs"),
		filepath.Join(c.DataDir, "jobs", "active"),
		filepath.Join(c.DataDir, "jobs", "completed"),
		filepath.Join(c.DataDir, "state"),
		filepath.Join(c.DataDir, "config"),
		filepath.Join(c.DataDir, "logs"),
		c.ProcessingSettings.TempDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	return nil
}