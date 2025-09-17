package storage

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileTracker struct {
	dataDir     string
	trackedFile string
	processedFiles map[string]ProcessedFile
	mutex       sync.RWMutex
}

type ProcessedFile struct {
	OriginalPath    string    `json:"original_path"`
	ProcessedPath   string    `json:"processed_path"`
	OriginalSize    int64     `json:"original_size"`
	ProcessedSize   int64     `json:"processed_size"`
	OriginalHash    string    `json:"original_hash"`
	ProcessedHash   string    `json:"processed_hash"`
	ProcessedAt     time.Time `json:"processed_at"`
	QualityTier     string    `json:"quality_tier"`
	CompressionRatio float64  `json:"compression_ratio"`
	ProcessingTime  int64     `json:"processing_time_ms"`
	Status          string    `json:"status"`
	ErrorMessage    string    `json:"error_message,omitempty"`
}

type FileStats struct {
	TotalFiles       int     `json:"total_files"`
	TotalOriginalSize int64  `json:"total_original_size"`
	TotalProcessedSize int64 `json:"total_processed_size"`
	AverageCompression float64 `json:"average_compression"`
	ProcessingErrors  int     `json:"processing_errors"`
	LastProcessed     time.Time `json:"last_processed"`
}

func NewFileTracker(dataDir string) *FileTracker {
	trackedFile := filepath.Join(dataDir, "state", "processed_files.json")
	
	ft := &FileTracker{
		dataDir:        dataDir,
		trackedFile:    trackedFile,
		processedFiles: make(map[string]ProcessedFile),
	}

	// Load existing tracked files
	ft.load()
	
	return ft
}

func (ft *FileTracker) AddProcessedFile(file ProcessedFile) error {
	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	// Calculate compression ratio
	if file.OriginalSize > 0 {
		file.CompressionRatio = 1.0 - (float64(file.ProcessedSize) / float64(file.OriginalSize))
	}

	// Use original file hash as key
	ft.processedFiles[file.OriginalHash] = file

	return ft.save()
}

func (ft *FileTracker) IsFileProcessed(filePath string) (bool, ProcessedFile, error) {
	hash, err := ft.calculateFileHash(filePath)
	if err != nil {
		return false, ProcessedFile{}, fmt.Errorf("calculating file hash: %w", err)
	}

	ft.mutex.RLock()
	defer ft.mutex.RUnlock()

	file, exists := ft.processedFiles[hash]
	return exists, file, nil
}

func (ft *FileTracker) MarkFileError(filePath, errorMsg string) error {
	hash, err := ft.calculateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("calculating file hash: %w", err)
	}

	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	file := ft.processedFiles[hash]
	file.Status = "error"
	file.ErrorMessage = errorMsg
	file.ProcessedAt = time.Now()
	ft.processedFiles[hash] = file

	return ft.save()
}

func (ft *FileTracker) GetStats() FileStats {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()

	stats := FileStats{
		TotalFiles: len(ft.processedFiles),
	}

	var totalCompression float64
	var validCompressions int
	var errors int

	for _, file := range ft.processedFiles {
		stats.TotalOriginalSize += file.OriginalSize
		stats.TotalProcessedSize += file.ProcessedSize

		if file.Status == "error" {
			errors++
		} else if file.CompressionRatio > 0 {
			totalCompression += file.CompressionRatio
			validCompressions++
		}

		if file.ProcessedAt.After(stats.LastProcessed) {
			stats.LastProcessed = file.ProcessedAt
		}
	}

	if validCompressions > 0 {
		stats.AverageCompression = totalCompression / float64(validCompressions)
	}

	stats.ProcessingErrors = errors

	return stats
}

func (ft *FileTracker) GetFilesByTier(tier string) []ProcessedFile {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()

	var files []ProcessedFile
	for _, file := range ft.processedFiles {
		if file.QualityTier == tier {
			files = append(files, file)
		}
	}

	return files
}

func (ft *FileTracker) GetRecentFiles(hours int) []ProcessedFile {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()

	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	var files []ProcessedFile

	for _, file := range ft.processedFiles {
		if file.ProcessedAt.After(cutoff) {
			files = append(files, file)
		}
	}

	return files
}

func (ft *FileTracker) CleanupOldEntries(daysToKeep int) error {
	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	cutoff := time.Now().AddDate(0, 0, -daysToKeep)
	var toDelete []string

	for hash, file := range ft.processedFiles {
		if file.ProcessedAt.Before(cutoff) {
			toDelete = append(toDelete, hash)
		}
	}

	for _, hash := range toDelete {
		delete(ft.processedFiles, hash)
	}

	if len(toDelete) > 0 {
		return ft.save()
	}

	return nil
}

func (ft *FileTracker) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("reading file for hash: %w", err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func (ft *FileTracker) load() error {
	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(ft.trackedFile), 0755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	data, err := os.ReadFile(ft.trackedFile)
	if os.IsNotExist(err) {
		// File doesn't exist yet, initialize empty map
		ft.processedFiles = make(map[string]ProcessedFile)
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading tracked files: %w", err)
	}

	if err := json.Unmarshal(data, &ft.processedFiles); err != nil {
		return fmt.Errorf("parsing tracked files: %w", err)
	}

	return nil
}

func (ft *FileTracker) save() error {
	data, err := json.MarshalIndent(ft.processedFiles, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling tracked files: %w", err)
	}

	// Atomic write using temporary file
	tempFile := ft.trackedFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("writing temp tracked file: %w", err)
	}

	if err := os.Rename(tempFile, ft.trackedFile); err != nil {
		os.Remove(tempFile) // cleanup on failure
		return fmt.Errorf("moving temp tracked file: %w", err)
	}

	return nil
}

func (ft *FileTracker) ExportStats(outputPath string) error {
	stats := ft.GetStats()
	
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling stats: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("writing stats file: %w", err)
	}

	return nil
}

func (ft *FileTracker) GetCompressionRateByExtension() map[string]float64 {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()

	extensionStats := make(map[string][]float64)

	for _, file := range ft.processedFiles {
		if file.Status != "error" && file.CompressionRatio > 0 {
			ext := filepath.Ext(file.OriginalPath)
			extensionStats[ext] = append(extensionStats[ext], file.CompressionRatio)
		}
	}

	// Calculate averages
	averages := make(map[string]float64)
	for ext, ratios := range extensionStats {
		var sum float64
		for _, ratio := range ratios {
			sum += ratio
		}
		averages[ext] = sum / float64(len(ratios))
	}

	return averages
}

func (ft *FileTracker) GetProcessingTimeStats() map[string]interface{} {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()

	var times []int64
	var totalTime int64

	for _, file := range ft.processedFiles {
		if file.Status != "error" && file.ProcessingTime > 0 {
			times = append(times, file.ProcessingTime)
			totalTime += file.ProcessingTime
		}
	}

	if len(times) == 0 {
		return map[string]interface{}{
			"count":   0,
			"average": 0,
			"total":   0,
		}
	}

	return map[string]interface{}{
		"count":               len(times),
		"average_ms":          totalTime / int64(len(times)),
		"total_ms":            totalTime,
		"total_seconds":       totalTime / 1000,
		"files_per_minute":    float64(len(times)) / (float64(totalTime) / 60000),
	}
}