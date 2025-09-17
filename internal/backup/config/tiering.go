package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type TieringEngine struct {
	config           *BackupConfig
	spaceMonitor     *SpaceMonitor
	currentUsage     int64
	maxUsage         int64
	adjustmentActive bool
	mutex            sync.RWMutex
}

type SpaceMonitor struct {
	dataPath         string
	thresholdReached bool
	lastCheck        time.Time
	totalSpace       int64
	usedSpace        int64
	mutex            sync.RWMutex
}

type TierAdjustment struct {
	TierName      string
	OriginalCRF   int
	AdjustedCRF   int
	OriginalQuality int
	AdjustedQuality int
	Reason        string
	Timestamp     time.Time
}

func NewTieringEngine(config *BackupConfig) *TieringEngine {
	return &TieringEngine{
		config:       config,
		spaceMonitor: NewSpaceMonitor(config.DataDir),
		maxUsage:     config.StorageLimitGB * 1024 * 1024 * 1024, // Convert to bytes
	}
}

func NewSpaceMonitor(dataPath string) *SpaceMonitor {
	return &SpaceMonitor{
		dataPath:  dataPath,
		lastCheck: time.Now(),
	}
}

func (te *TieringEngine) DetermineTier(fileDate time.Time, filePath string) (QualityTier, error) {
	te.mutex.RLock()
	defer te.mutex.RUnlock()

	// Check for user-defined folder overrides
	if tier, found := te.checkFolderOverrides(filePath); found {
		return tier, nil
	}

	// Check for date range exceptions
	if tier, found := te.checkDateRangeExceptions(fileDate); found {
		return tier, nil
	}

	// Use standard age-based tiering
	age := time.Since(fileDate)
	baseTier := te.config.GetTierByAge(age)

	// Apply space pressure adjustments if needed
	if te.adjustmentActive {
		adjustedTier := te.applySpacePressureAdjustment(baseTier)
		return adjustedTier, nil
	}

	return baseTier, nil
}

func (te *TieringEngine) checkFolderOverrides(filePath string) (QualityTier, bool) {
	for folder, tierName := range te.config.UserPreferences.FolderOverrides {
		if filepath.HasPrefix(filePath, folder) {
			for _, tier := range te.config.QualityTiers {
				if tier.Name == tierName {
					return tier, true
				}
			}
		}
	}
	return QualityTier{}, false
}

func (te *TieringEngine) checkDateRangeExceptions(fileDate time.Time) (QualityTier, bool) {
	for _, exception := range te.config.UserPreferences.DateRangeExceptions {
		if fileDate.After(exception.StartDate) && fileDate.Before(exception.EndDate) {
			for _, tier := range te.config.QualityTiers {
				if tier.Name == exception.ForceTier {
					return tier, true
				}
			}
		}
	}
	return QualityTier{}, false
}

func (te *TieringEngine) applySpacePressureAdjustment(tier QualityTier) QualityTier {
	adjustedTier := tier

	// Increase compression for older tiers first
	step := te.config.ProcessingSettings.QualityAdjustmentStep

	// Adjust video CRF (higher = more compression)
	if adjustedTier.VideoCRF+step <= 51 {
		adjustedTier.VideoCRF += step
	}

	// Adjust photo quality (lower = more compression)
	if adjustedTier.PhotoQuality-step >= 50 {
		adjustedTier.PhotoQuality -= step
	}

	return adjustedTier
}

func (te *TieringEngine) UpdateSpaceUsage() error {
	usage, err := te.spaceMonitor.GetCurrentUsage()
	if err != nil {
		return fmt.Errorf("updating space usage: %w", err)
	}

	te.mutex.Lock()
	defer te.mutex.Unlock()

	te.currentUsage = usage
	
	// Check if we need to activate space pressure adjustments
	threshold := float64(te.maxUsage) * te.config.ProcessingSettings.SpacePressureThreshold
	if float64(te.currentUsage) > threshold {
		te.adjustmentActive = true
	} else {
		te.adjustmentActive = false
	}

	return nil
}

func (te *TieringEngine) GetSpaceStatus() (float64, bool, error) {
	if err := te.UpdateSpaceUsage(); err != nil {
		return 0, false, err
	}

	te.mutex.RLock()
	defer te.mutex.RUnlock()

	usagePercent := float64(te.currentUsage) / float64(te.maxUsage) * 100
	return usagePercent, te.adjustmentActive, nil
}

func (te *TieringEngine) EstimateSpaceRequired(fileCount int, avgFileSizeMB float64) int64 {
	// Rough estimation based on file count and average size
	// This would be refined with actual statistics over time
	estimatedBytes := int64(float64(fileCount) * avgFileSizeMB * 1024 * 1024)
	
	// Apply compression ratio estimates based on tiers
	// Tier 1: ~30% compression, Tier 2: ~50% compression, Tier 3: ~70% compression
	compressionRatio := 0.4 // Average across tiers
	return int64(float64(estimatedBytes) * compressionRatio)
}

func (te *TieringEngine) RecommendTierAdjustments(targetUsagePercent float64) []TierAdjustment {
	var recommendations []TierAdjustment

	currentUsagePercent, _, _ := te.GetSpaceStatus()
	if currentUsagePercent <= targetUsagePercent {
		return recommendations // No adjustments needed
	}

	// Calculate how much space reduction is needed
	reductionNeeded := currentUsagePercent - targetUsagePercent

	// Generate recommendations starting with oldest tier
	for i := len(te.config.QualityTiers) - 1; i >= 0 && reductionNeeded > 0; i-- {
		tier := te.config.QualityTiers[i]
		
		adjustment := TierAdjustment{
			TierName:        tier.Name,
			OriginalCRF:     tier.VideoCRF,
			OriginalQuality: tier.PhotoQuality,
			Timestamp:       time.Now(),
		}

		step := te.config.ProcessingSettings.QualityAdjustmentStep
		
		// Adjust video CRF
		if tier.VideoCRF+step <= 51 {
			adjustment.AdjustedCRF = tier.VideoCRF + step
		} else {
			adjustment.AdjustedCRF = tier.VideoCRF
		}

		// Adjust photo quality
		if tier.PhotoQuality-step >= 50 {
			adjustment.AdjustedQuality = tier.PhotoQuality - step
		} else {
			adjustment.AdjustedQuality = tier.PhotoQuality
		}

		adjustment.Reason = fmt.Sprintf("Reduce usage by ~%.1f%%", reductionNeeded/float64(i+1))
		recommendations = append(recommendations, adjustment)

		// Estimate reduction achieved (rough calculation)
		reductionNeeded -= 5.0 // Assume ~5% reduction per tier adjustment
	}

	return recommendations
}

func (sm *SpaceMonitor) GetCurrentUsage() (int64, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Only check every 5 minutes to avoid excessive disk I/O
	if time.Since(sm.lastCheck) < 5*time.Minute {
		return sm.usedSpace, nil
	}

	var totalSize int64
	err := filepath.Walk(sm.dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("calculating directory size: %w", err)
	}

	sm.usedSpace = totalSize
	sm.lastCheck = time.Now()

	return totalSize, nil
}

func (sm *SpaceMonitor) GetAvailableSpace() (int64, error) {
	// Get filesystem stats for the data directory
	// This would need platform-specific implementation using syscall.Statfs_t on Linux
	// For now, return a placeholder
	return 1024 * 1024 * 1024 * 100, nil // 100GB placeholder
}

func (te *TieringEngine) ShouldReduceQuality(currentSpaceUsage int64) bool {
	te.mutex.RLock()
	defer te.mutex.RUnlock()

	threshold := float64(te.maxUsage) * te.config.ProcessingSettings.SpacePressureThreshold
	return float64(currentSpaceUsage) > threshold
}

func (te *TieringEngine) GetTierStatistics() map[string]interface{} {
	te.mutex.RLock()
	defer te.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_tiers"] = len(te.config.QualityTiers)
	stats["current_usage_bytes"] = te.currentUsage
	stats["max_usage_bytes"] = te.maxUsage
	stats["usage_percentage"] = float64(te.currentUsage) / float64(te.maxUsage) * 100
	stats["space_pressure_active"] = te.adjustmentActive
	
	tierInfo := make([]map[string]interface{}, len(te.config.QualityTiers))
	for i, tier := range te.config.QualityTiers {
		tierInfo[i] = map[string]interface{}{
			"name":              tier.Name,
			"age_threshold":     tier.AgeThresholdDays,
			"video_crf":         tier.VideoCRF,
			"photo_quality":     tier.PhotoQuality,
			"metadata_level":    tier.MetadataLevel,
		}
	}
	stats["tiers"] = tierInfo

	return stats
}