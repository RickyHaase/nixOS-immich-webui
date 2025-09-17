package processor

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/RickyHaase/nixOS-immich-webui/internal/backup/config"
	"github.com/RickyHaase/nixOS-immich-webui/internal/backup/storage"
)

type Pipeline struct {
	config         *config.BackupConfig
	tieringEngine  *config.TieringEngine
	photoProcessor *PhotoProcessor
	videoProcessor *VideoProcessor
	fileTracker    *storage.FileTracker
	stateManager   *storage.StateManager
}

type ProcessingJob struct {
	ID              string
	SourcePath      string
	DestinationPath string
	IncludePatterns []string
	ExcludePatterns []string
	DeleteOriginals bool
	VerifyChecksums bool
	MaxConcurrency  int
	ProgressCallback func(progress ProcessingProgress)
}

type ProcessingProgress struct {
	JobID              string    `json:"job_id"`
	Phase              string    `json:"phase"`
	CurrentFile        string    `json:"current_file"`
	ProcessedFiles     int       `json:"processed_files"`
	TotalFiles         int       `json:"total_files"`
	ProcessedBytes     int64     `json:"processed_bytes"`
	TotalBytes         int64     `json:"total_bytes"`
	Progress           float64   `json:"progress"`
	StartTime          time.Time `json:"start_time"`
	ElapsedTime        time.Duration `json:"elapsed_time"`
	EstimatedRemaining time.Duration `json:"estimated_remaining"`
	ProcessingRate     float64   `json:"processing_rate"`
	Errors             []string  `json:"errors"`
	CurrentOperation   string    `json:"current_operation"`
}

type ProcessingResult struct {
	JobID            string                     `json:"job_id"`
	Status           string                     `json:"status"`
	StartTime        time.Time                  `json:"start_time"`
	EndTime          time.Time                  `json:"end_time"`
	TotalFiles       int                        `json:"total_files"`
	ProcessedFiles   int                        `json:"processed_files"`
	FailedFiles      int                        `json:"failed_files"`
	SkippedFiles     int                        `json:"skipped_files"`
	PhotoResults     []*PhotoProcessingResult   `json:"photo_results"`
	VideoResults     []*VideoProcessingResult   `json:"video_results"`
	TotalOriginalSize int64                     `json:"total_original_size"`
	TotalProcessedSize int64                    `json:"total_processed_size"`
	CompressionRatio  float64                   `json:"compression_ratio"`
	ProcessingTime    time.Duration             `json:"processing_time"`
	Errors           []string                   `json:"errors"`
}

func NewPipeline(cfg *config.BackupConfig) (*Pipeline, error) {
	// Initialize processors
	photoProcessor, err := NewPhotoProcessor(cfg)
	if err != nil {
		return nil, fmt.Errorf("initializing photo processor: %w", err)
	}

	videoProcessor, err := NewVideoProcessor(cfg)
	if err != nil {
		return nil, fmt.Errorf("initializing video processor: %w", err)
	}

	// Initialize other components
	tieringEngine := config.NewTieringEngine(cfg)
	fileTracker := storage.NewFileTracker(cfg.DataDir)
	stateManager := storage.NewStateManager(cfg.DataDir)

	return &Pipeline{
		config:         cfg,
		tieringEngine:  tieringEngine,
		photoProcessor: photoProcessor,
		videoProcessor: videoProcessor,
		fileTracker:    fileTracker,
		stateManager:   stateManager,
	}, nil
}

func (p *Pipeline) ProcessDirectory(job ProcessingJob) (*ProcessingResult, error) {
	startTime := time.Now()
	
	result := &ProcessingResult{
		JobID:     job.ID,
		Status:    "running",
		StartTime: startTime,
	}

	// Initialize progress tracking
	progress := ProcessingProgress{
		JobID:     job.ID,
		Phase:     "discovery",
		StartTime: startTime,
	}

	slog.Info("Starting backup processing job", "job_id", job.ID, "source", job.SourcePath)

	// Phase 1: Discover files
	if job.ProgressCallback != nil {
		progress.Phase = "discovery"
		progress.CurrentOperation = "Scanning for media files..."
		job.ProgressCallback(progress)
	}

	p.stateManager.AddPhaseStats(job.ID, "discovery", 0)

	files, err := p.discoverFiles(job.SourcePath, job.IncludePatterns, job.ExcludePatterns)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("File discovery failed: %v", err))
		return result, err
	}

	result.TotalFiles = len(files)
	progress.TotalFiles = len(files)

	slog.Info("File discovery completed", "total_files", len(files))

	// Calculate total size for progress tracking
	var totalSize int64
	for _, file := range files {
		if info, err := os.Stat(file.Path); err == nil {
			totalSize += info.Size()
		}
	}
	progress.TotalBytes = totalSize
	result.TotalOriginalSize = totalSize

	// Phase 2: Process files with concurrency control
	if job.ProgressCallback != nil {
		progress.Phase = "processing"
		progress.CurrentOperation = "Processing media files..."
		job.ProgressCallback(progress)
	}

	p.stateManager.AddPhaseStats(job.ID, "processing", len(files))

	processedFiles, photoResults, videoResults, err := p.processFiles(files, job, &progress)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("File processing failed: %v", err))
	}

	result.ProcessedFiles = processedFiles
	result.PhotoResults = photoResults
	result.VideoResults = videoResults

	// Calculate final statistics
	for _, photoResult := range photoResults {
		result.TotalProcessedSize += photoResult.ProcessedSize
		if photoResult.Error != nil {
			result.FailedFiles++
			result.Errors = append(result.Errors, fmt.Sprintf("Photo processing error (%s): %v", photoResult.OriginalPath, photoResult.Error))
		}
	}

	for _, videoResult := range videoResults {
		result.TotalProcessedSize += videoResult.ProcessedSize
		if videoResult.Error != nil {
			result.FailedFiles++
			result.Errors = append(result.Errors, fmt.Sprintf("Video processing error (%s): %v", videoResult.OriginalPath, videoResult.Error))
		}
	}

	if result.TotalOriginalSize > 0 {
		result.CompressionRatio = 1.0 - (float64(result.TotalProcessedSize) / float64(result.TotalOriginalSize))
	}

	result.EndTime = time.Now()
	result.ProcessingTime = result.EndTime.Sub(result.StartTime)

	if len(result.Errors) == 0 {
		result.Status = "completed"
	} else if result.ProcessedFiles > 0 {
		result.Status = "completed_with_errors"
	} else {
		result.Status = "failed"
	}

	slog.Info("Backup processing job completed", 
		"job_id", job.ID,
		"status", result.Status,
		"processed_files", result.ProcessedFiles,
		"failed_files", result.FailedFiles,
		"compression_ratio", fmt.Sprintf("%.1f%%", result.CompressionRatio*100),
		"processing_time", result.ProcessingTime,
	)

	return result, nil
}

type FileInfo struct {
	Path     string
	Type     string // "photo" or "video"
	Size     int64
	ModTime  time.Time
}

func (p *Pipeline) discoverFiles(sourcePath string, includePatterns, excludePatterns []string) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("Error accessing file during discovery", "path", path, "err", err)
			return nil // Continue walking
		}

		if info.IsDir() {
			return nil // Continue into directories
		}

		// Check if file should be included
		if !p.shouldIncludeFile(path, includePatterns, excludePatterns) {
			return nil
		}

		// Check if it's a supported media file
		var fileType string
		if p.photoProcessor.IsPhotoFile(path) {
			fileType = "photo"
		} else if p.videoProcessor.IsVideoFile(path) {
			fileType = "video"
		} else {
			return nil // Skip unsupported files
		}

		files = append(files, FileInfo{
			Path:    path,
			Type:    fileType,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})

		return nil
	})

	return files, err
}

func (p *Pipeline) shouldIncludeFile(filePath string, includePatterns, excludePatterns []string) bool {
	// Check exclude patterns first
	for _, pattern := range excludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
			return false
		}
		if strings.Contains(filePath, pattern) {
			return false
		}
	}

	// If no include patterns specified, include by default
	if len(includePatterns) == 0 {
		return true
	}

	// Check include patterns
	for _, pattern := range includePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
			return true
		}
		if strings.Contains(filePath, pattern) {
			return true
		}
	}

	return false
}

func (p *Pipeline) processFiles(files []FileInfo, job ProcessingJob, progress *ProcessingProgress) (int, []*PhotoProcessingResult, []*VideoProcessingResult, error) {
	var photoResults []*PhotoProcessingResult
	var videoResults []*VideoProcessingResult
	
	// Separate photos and videos
	var photos, videos []FileInfo
	for _, file := range files {
		if file.Type == "photo" {
			photos = append(photos, file)
		} else if file.Type == "video" {
			videos = append(videos, file)
		}
	}

	// Create worker pool for concurrent processing
	concurrency := job.MaxConcurrency
	if concurrency <= 0 {
		concurrency = p.config.ProcessingSettings.MaxConcurrentJobs
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	
	// Channel for work items
	workChan := make(chan FileInfo, len(files))
	
	// Send all files to work channel
	for _, file := range files {
		workChan <- file
	}
	close(workChan)

	processedCount := 0

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for file := range workChan {
				// Update progress
				mu.Lock()
				processedCount++
				progress.ProcessedFiles = processedCount
				progress.CurrentFile = filepath.Base(file.Path)
				progress.Progress = float64(processedCount) / float64(len(files)) * 100
				progress.ElapsedTime = time.Since(progress.StartTime)
				
				if processedCount > 0 {
					progress.ProcessingRate = float64(processedCount) / progress.ElapsedTime.Minutes()
					if progress.ProcessingRate > 0 {
						remaining := float64(len(files)-processedCount) / progress.ProcessingRate
						progress.EstimatedRemaining = time.Duration(remaining) * time.Minute
					}
				}
				
				// Call progress callback
				if job.ProgressCallback != nil {
					job.ProgressCallback(*progress)
				}
				mu.Unlock()

				// Determine quality tier for this file
				tier, err := p.tieringEngine.DetermineTier(file.ModTime, file.Path)
				if err != nil {
					slog.Error("Failed to determine quality tier", "file", file.Path, "err", err)
					tier = p.config.QualityTiers[len(p.config.QualityTiers)-1] // Use lowest tier as fallback
				}

				// Generate destination path
				relPath, _ := filepath.Rel(job.SourcePath, file.Path)
				destPath := filepath.Join(job.DestinationPath, relPath)

				// Check if file already processed
				if processed, processedFile, err := p.fileTracker.IsFileProcessed(file.Path); err == nil && processed {
					slog.Debug("File already processed, skipping", "file", file.Path, "processed_at", processedFile.ProcessedAt)
					continue
				}

				// Process based on file type
				if file.Type == "photo" {
					result, err := p.photoProcessor.ProcessPhoto(file.Path, destPath, tier)
					if err != nil {
						slog.Error("Photo processing failed", "file", file.Path, "err", err)
						p.stateManager.IncrementErrorCount(job.ID, err.Error())
					} else {
						// Track processed file
						trackedFile := storage.ProcessedFile{
							OriginalPath:     result.OriginalPath,
							ProcessedPath:    result.ProcessedPath,
							OriginalSize:     result.OriginalSize,
							ProcessedSize:    result.ProcessedSize,
							ProcessedAt:      time.Now(),
							QualityTier:      result.QualityTier,
							CompressionRatio: result.CompressionRatio,
							ProcessingTime:   result.ProcessingTime.Milliseconds(),
							Status:           "completed",
						}
						
						if result.Error != nil {
							trackedFile.Status = "error"
							trackedFile.ErrorMessage = result.Error.Error()
						}
						
						p.fileTracker.AddProcessedFile(trackedFile)
						p.stateManager.UpdateCompressionStats(job.ID, result.OriginalSize, result.ProcessedSize)
					}

					mu.Lock()
					photoResults = append(photoResults, result)
					mu.Unlock()
					
				} else if file.Type == "video" {
					result, err := p.videoProcessor.ProcessVideo(file.Path, destPath, tier)
					if err != nil {
						slog.Error("Video processing failed", "file", file.Path, "err", err)
						p.stateManager.IncrementErrorCount(job.ID, err.Error())
					} else {
						// Track processed file
						trackedFile := storage.ProcessedFile{
							OriginalPath:     result.OriginalPath,
							ProcessedPath:    result.ProcessedPath,
							OriginalSize:     result.OriginalSize,
							ProcessedSize:    result.ProcessedSize,
							ProcessedAt:      time.Now(),
							QualityTier:      result.QualityTier,
							CompressionRatio: result.CompressionRatio,
							ProcessingTime:   result.ProcessingTime.Milliseconds(),
							Status:           "completed",
						}
						
						if result.Error != nil {
							trackedFile.Status = "error"
							trackedFile.ErrorMessage = result.Error.Error()
						}
						
						p.fileTracker.AddProcessedFile(trackedFile)
						p.stateManager.UpdateCompressionStats(job.ID, result.OriginalSize, result.ProcessedSize)
					}

					mu.Lock()
					videoResults = append(videoResults, result)
					mu.Unlock()
				}

				// Update processing progress state
				p.stateManager.UpdateJobProgress(job.ID, processedCount, len(files), file.Path)
				
				// Check space pressure and adjust quality if needed
				p.tieringEngine.UpdateSpaceUsage()
			}
		}()
	}

	wg.Wait()

	return processedCount, photoResults, videoResults, nil
}

func (p *Pipeline) EstimateProcessingTime(sourcePath string) (time.Duration, error) {
	files, err := p.discoverFiles(sourcePath, nil, nil)
	if err != nil {
		return 0, fmt.Errorf("discovering files: %w", err)
	}

	var totalEstimate time.Duration
	
	for _, file := range files {
		if file.Type == "photo" {
			// Photos typically process quickly
			totalEstimate += 100 * time.Millisecond
		} else if file.Type == "video" {
			// Estimate based on file size (rough approximation)
			estimateSeconds := file.Size / (1024 * 1024) // 1 second per MB
			if estimateSeconds < 1 {
				estimateSeconds = 1
			}
			totalEstimate += time.Duration(estimateSeconds) * time.Second
		}
	}

	// Adjust for concurrency
	concurrency := float64(p.config.ProcessingSettings.MaxConcurrentJobs)
	if concurrency > 1 {
		totalEstimate = time.Duration(float64(totalEstimate) / concurrency)
	}

	return totalEstimate, nil
}

func (p *Pipeline) GetProcessingStatistics() (map[string]interface{}, error) {
	fileStats := p.fileTracker.GetStats()
	processingStats, err := p.stateManager.GetProcessingStatistics()
	if err != nil {
		return nil, err
	}

	// Combine stats
	combined := make(map[string]interface{})
	
	// File tracker stats
	combined["total_files_tracked"] = fileStats.TotalFiles
	combined["total_original_size"] = fileStats.TotalOriginalSize
	combined["total_processed_size"] = fileStats.TotalProcessedSize
	combined["average_compression"] = fileStats.AverageCompression
	combined["average_compression_percent"] = fmt.Sprintf("%.1f", fileStats.AverageCompression*100)
	combined["processing_errors"] = fileStats.ProcessingErrors
	combined["last_processed"] = fileStats.LastProcessed

	// Processing stats
	for k, v := range processingStats {
		combined[k] = v
	}

	// Tier statistics
	tierStats := p.tieringEngine.GetTierStatistics()
	combined["tier_statistics"] = tierStats

	return combined, nil
}