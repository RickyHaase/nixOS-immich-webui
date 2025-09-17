package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	validTierNames = regexp.MustCompile(`^[a-zA-Z0-9\s\(\)\-]+$`)
	validLogLevels = map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

func ValidateBackupConfig(config *BackupConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate data directory
	if err := validateDataDir(config.DataDir); err != nil {
		errors = append(errors, ValidationError{
			Field:   "data_dir",
			Message: err.Error(),
		})
	}

	// Validate storage limit
	if config.StorageLimitGB <= 0 {
		errors = append(errors, ValidationError{
			Field:   "storage_limit_gb",
			Message: "must be greater than 0",
		})
	}

	// Validate job retention
	if config.JobRetentionDays < 1 {
		errors = append(errors, ValidationError{
			Field:   "job_retention_days",
			Message: "must be at least 1 day",
		})
	}

	// Validate log level
	if !validLogLevels[config.LogLevel] {
		errors = append(errors, ValidationError{
			Field:   "log_level",
			Message: "must be one of: debug, info, warn, error",
		})
	}

	// Validate quality tiers
	if tierErrors := validateQualityTiers(config.QualityTiers); len(tierErrors) > 0 {
		errors = append(errors, tierErrors...)
	}

	// Validate processing settings
	if procErrors := validateProcessingSettings(&config.ProcessingSettings); len(procErrors) > 0 {
		errors = append(errors, procErrors...)
	}

	// Validate user preferences
	if prefErrors := validateUserPreferences(&config.UserPreferences); len(prefErrors) > 0 {
		errors = append(errors, prefErrors...)
	}

	return errors
}

func validateDataDir(dataDir string) error {
	if dataDir == "" {
		return fmt.Errorf("cannot be empty")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(dataDir)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if parent directory exists and is writable
	parentDir := filepath.Dir(absPath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		return fmt.Errorf("parent directory %s does not exist", parentDir)
	}

	// Test write permissions by trying to create a temp file
	tempFile := filepath.Join(parentDir, ".backup_test_write")
	if file, err := os.Create(tempFile); err != nil {
		return fmt.Errorf("directory not writable: %w", err)
	} else {
		file.Close()
		os.Remove(tempFile)
	}

	return nil
}

func validateQualityTiers(tiers []QualityTier) ValidationErrors {
	var errors ValidationErrors

	if len(tiers) == 0 {
		errors = append(errors, ValidationError{
			Field:   "quality_tiers",
			Message: "at least one quality tier must be defined",
		})
		return errors
	}

	seenNames := make(map[string]bool)
	lastThreshold := 0

	for i, tier := range tiers {
		prefix := fmt.Sprintf("quality_tiers[%d]", i)

		// Validate name
		if tier.Name == "" {
			errors = append(errors, ValidationError{
				Field:   prefix + ".name",
				Message: "cannot be empty",
			})
		} else if !validTierNames.MatchString(tier.Name) {
			errors = append(errors, ValidationError{
				Field:   prefix + ".name",
				Message: "contains invalid characters",
			})
		} else if seenNames[tier.Name] {
			errors = append(errors, ValidationError{
				Field:   prefix + ".name",
				Message: "duplicate tier name",
			})
		}
		seenNames[tier.Name] = true

		// Validate age threshold progression
		if tier.AgeThresholdDays <= lastThreshold && i > 0 {
			errors = append(errors, ValidationError{
				Field:   prefix + ".age_threshold_days",
				Message: "must be greater than previous tier threshold",
			})
		}
		lastThreshold = tier.AgeThresholdDays

		// Validate photo settings
		if tier.PhotoMaxResolution <= 0 {
			errors = append(errors, ValidationError{
				Field:   prefix + ".photo_max_resolution",
				Message: "must be greater than 0",
			})
		}

		if tier.PhotoQuality < 1 || tier.PhotoQuality > 100 {
			errors = append(errors, ValidationError{
				Field:   prefix + ".photo_quality",
				Message: "must be between 1 and 100",
			})
		}

		// Validate video settings
		if tier.VideoMaxHeight <= 0 {
			errors = append(errors, ValidationError{
				Field:   prefix + ".video_max_height",
				Message: "must be greater than 0",
			})
		}

		if tier.VideoMaxFPS <= 0 {
			errors = append(errors, ValidationError{
				Field:   prefix + ".video_max_fps",
				Message: "must be greater than 0",
			})
		}

		if tier.VideoCRF < 0 || tier.VideoCRF > 51 {
			errors = append(errors, ValidationError{
				Field:   prefix + ".video_crf",
				Message: "must be between 0 and 51",
			})
		}

		// Validate metadata level
		validMetadata := map[string]bool{
			"full":      true,
			"essential": true,
			"minimal":   true,
		}
		if !validMetadata[tier.MetadataLevel] {
			errors = append(errors, ValidationError{
				Field:   prefix + ".metadata_level",
				Message: "must be 'full', 'essential', or 'minimal'",
			})
		}
	}

	return errors
}

func validateProcessingSettings(settings *ProcessingSettings) ValidationErrors {
	var errors ValidationErrors

	if settings.MaxConcurrentJobs < 1 {
		errors = append(errors, ValidationError{
			Field:   "processing_settings.max_concurrent_jobs",
			Message: "must be at least 1",
		})
	}

	if settings.MaxConcurrentJobs > 10 {
		errors = append(errors, ValidationError{
			Field:   "processing_settings.max_concurrent_jobs",
			Message: "should not exceed 10 for system stability",
		})
	}

	if settings.TempDir == "" {
		errors = append(errors, ValidationError{
			Field:   "processing_settings.temp_dir",
			Message: "cannot be empty",
		})
	}

	if settings.SpacePressureThreshold <= 0 || settings.SpacePressureThreshold > 1 {
		errors = append(errors, ValidationError{
			Field:   "processing_settings.space_pressure_threshold",
			Message: "must be between 0 and 1",
		})
	}

	if settings.QualityAdjustmentStep < 1 || settings.QualityAdjustmentStep > 10 {
		errors = append(errors, ValidationError{
			Field:   "processing_settings.quality_adjustment_step",
			Message: "must be between 1 and 10",
		})
	}

	return errors
}

func validateUserPreferences(prefs *UserPreferences) ValidationErrors {
	var errors ValidationErrors

	// Validate folder overrides
	for folder, tier := range prefs.FolderOverrides {
		if folder == "" {
			errors = append(errors, ValidationError{
				Field:   "user_preferences.folder_overrides",
				Message: "folder path cannot be empty",
			})
		}

		if tier == "" {
			errors = append(errors, ValidationError{
				Field:   "user_preferences.folder_overrides",
				Message: fmt.Sprintf("tier for folder '%s' cannot be empty", folder),
			})
		}
	}

	// Validate date range exceptions
	for i, dateRange := range prefs.DateRangeExceptions {
		prefix := fmt.Sprintf("user_preferences.date_range_exceptions[%d]", i)

		if dateRange.StartDate.IsZero() {
			errors = append(errors, ValidationError{
				Field:   prefix + ".start_date",
				Message: "cannot be empty",
			})
		}

		if dateRange.EndDate.IsZero() {
			errors = append(errors, ValidationError{
				Field:   prefix + ".end_date",
				Message: "cannot be empty",
			})
		}

		if !dateRange.StartDate.IsZero() && !dateRange.EndDate.IsZero() && dateRange.EndDate.Before(dateRange.StartDate) {
			errors = append(errors, ValidationError{
				Field:   prefix + ".end_date",
				Message: "must be after start_date",
			})
		}

		if dateRange.ForceTier == "" {
			errors = append(errors, ValidationError{
				Field:   prefix + ".force_tier",
				Message: "cannot be empty",
			})
		}
	}

	return errors
}

func SanitizeConfig(config *BackupConfig) {
	// Sanitize paths
	config.DataDir = filepath.Clean(config.DataDir)
	config.ProcessingSettings.TempDir = filepath.Clean(config.ProcessingSettings.TempDir)

	// Sanitize tier names
	for i := range config.QualityTiers {
		config.QualityTiers[i].Name = strings.TrimSpace(config.QualityTiers[i].Name)
	}

	// Ensure reasonable defaults
	if config.JobRetentionDays < 1 {
		config.JobRetentionDays = 30
	}

	if config.ProcessingSettings.MaxConcurrentJobs < 1 {
		config.ProcessingSettings.MaxConcurrentJobs = 2
	}

	if config.ProcessingSettings.SpacePressureThreshold <= 0 || config.ProcessingSettings.SpacePressureThreshold > 1 {
		config.ProcessingSettings.SpacePressureThreshold = 0.9
	}
}