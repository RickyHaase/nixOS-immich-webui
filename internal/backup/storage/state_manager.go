package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type StateManager struct {
	dataDir   string
	stateDir  string
	mutex     sync.RWMutex
}

type JobState struct {
	ID                  string             `json:"id"`
	Status              JobStatus          `json:"status"`
	Progress            float64            `json:"progress"`
	ProcessedFiles      int                `json:"processed_files"`
	TotalFiles          int                `json:"total_files"`
	StartTime           time.Time          `json:"start_time"`
	EstimatedCompletion *time.Time         `json:"estimated_completion,omitempty"`
	CurrentFile         string             `json:"current_file"`
	CurrentOperation    string             `json:"current_operation"`
	ProcessingRate      float64            `json:"processing_rate"` // files per minute
	ErrorMessage        string             `json:"error_message,omitempty"`
	ErrorCount          int                `json:"error_count"`
	BytesProcessed      int64              `json:"bytes_processed"`
	BytesTotal          int64              `json:"bytes_total"`
	CompressionStats    CompressionStats   `json:"compression_stats"`
	PhaseStats          map[string]PhaseStats `json:"phase_stats"`
}

type CompressionStats struct {
	OriginalBytes    int64   `json:"original_bytes"`
	CompressedBytes  int64   `json:"compressed_bytes"`
	CompressionRatio float64 `json:"compression_ratio"`
	SpaceSaved       int64   `json:"space_saved"`
}

type PhaseStats struct {
	Phase       string    `json:"phase"`
	StartTime   time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	FilesCount  int       `json:"files_count"`
	ElapsedMs   int64     `json:"elapsed_ms"`
}

type SystemState struct {
	ActiveJobs          []string          `json:"active_jobs"`
	TotalDiskUsage      int64             `json:"total_disk_usage"`
	AvailableDiskSpace  int64             `json:"available_disk_space"`
	ProcessingLoad      float64           `json:"processing_load"`
	LastHealthCheck     time.Time         `json:"last_health_check"`
	SystemStats         SystemStats       `json:"system_stats"`
	BackupStatistics    BackupStatistics  `json:"backup_statistics"`
}

type SystemStats struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskIO      float64 `json:"disk_io"`
	NetworkIO   float64 `json:"network_io"`
}

type BackupStatistics struct {
	TotalJobsRun        int       `json:"total_jobs_run"`
	SuccessfulJobs      int       `json:"successful_jobs"`
	FailedJobs          int       `json:"failed_jobs"`
	TotalFilesProcessed int64     `json:"total_files_processed"`
	TotalBytesProcessed int64     `json:"total_bytes_processed"`
	TotalSpaceSaved     int64     `json:"total_space_saved"`
	AverageCompression  float64   `json:"average_compression"`
	LastBackupTime      time.Time `json:"last_backup_time"`
}

func NewStateManager(dataDir string) *StateManager {
	return &StateManager{
		dataDir:  dataDir,
		stateDir: filepath.Join(dataDir, "state"),
	}
}

func (sm *StateManager) SaveJobState(state *JobState) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Update estimated completion if we have enough data
	if state.ProcessedFiles > 0 && state.TotalFiles > 0 && state.ProcessingRate > 0 {
		remainingFiles := state.TotalFiles - state.ProcessedFiles
		remainingMinutes := float64(remainingFiles) / state.ProcessingRate
		estimatedCompletion := time.Now().Add(time.Duration(remainingMinutes) * time.Minute)
		state.EstimatedCompletion = &estimatedCompletion
	}

	// Calculate progress percentage
	if state.TotalFiles > 0 {
		state.Progress = float64(state.ProcessedFiles) / float64(state.TotalFiles) * 100
	}

	// Update compression stats
	if state.BytesProcessed > 0 && state.CompressionStats.OriginalBytes > 0 {
		state.CompressionStats.CompressionRatio = 1.0 - (float64(state.CompressionStats.CompressedBytes) / float64(state.CompressionStats.OriginalBytes))
		state.CompressionStats.SpaceSaved = state.CompressionStats.OriginalBytes - state.CompressionStats.CompressedBytes
	}

	statePath := filepath.Join(sm.stateDir, fmt.Sprintf("progress-%s.json", state.ID))
	return sm.saveStateFile(state, statePath)
}

func (sm *StateManager) GetJobState(jobID string) (*JobState, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	statePath := filepath.Join(sm.stateDir, fmt.Sprintf("progress-%s.json", jobID))
	
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("job state not found")
		}
		return nil, fmt.Errorf("reading job state: %w", err)
	}

	var state JobState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing job state: %w", err)
	}

	return &state, nil
}

func (sm *StateManager) DeleteJobState(jobID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	statePath := filepath.Join(sm.stateDir, fmt.Sprintf("progress-%s.json", jobID))
	err := os.Remove(statePath)
	if os.IsNotExist(err) {
		return nil // Already deleted
	}
	return err
}

func (sm *StateManager) UpdateJobProgress(jobID string, processedFiles, totalFiles int, currentFile string) error {
	state, err := sm.GetJobState(jobID)
	if err != nil {
		// Create new state if it doesn't exist
		state = &JobState{
			ID:         jobID,
			Status:     JobStatusRunning,
			StartTime:  time.Now(),
			PhaseStats: make(map[string]PhaseStats),
		}
	}

	state.ProcessedFiles = processedFiles
	state.TotalFiles = totalFiles
	state.CurrentFile = currentFile

	// Calculate processing rate (files per minute)
	elapsed := time.Since(state.StartTime).Minutes()
	if elapsed > 0 {
		state.ProcessingRate = float64(processedFiles) / elapsed
	}

	return sm.SaveJobState(state)
}

func (sm *StateManager) AddPhaseStats(jobID, phase string, filesCount int) error {
	state, err := sm.GetJobState(jobID)
	if err != nil {
		return fmt.Errorf("getting job state: %w", err)
	}

	// End previous phase if exists
	for phaseName, phaseStats := range state.PhaseStats {
		if phaseStats.EndTime == nil {
			now := time.Now()
			phaseStats.EndTime = &now
			phaseStats.ElapsedMs = now.Sub(phaseStats.StartTime).Milliseconds()
			state.PhaseStats[phaseName] = phaseStats
		}
	}

	// Start new phase
	state.PhaseStats[phase] = PhaseStats{
		Phase:      phase,
		StartTime:  time.Now(),
		FilesCount: filesCount,
	}

	state.CurrentOperation = phase

	return sm.SaveJobState(state)
}

func (sm *StateManager) UpdateCompressionStats(jobID string, originalBytes, compressedBytes int64) error {
	state, err := sm.GetJobState(jobID)
	if err != nil {
		return fmt.Errorf("getting job state: %w", err)
	}

	state.CompressionStats.OriginalBytes += originalBytes
	state.CompressionStats.CompressedBytes += compressedBytes
	state.BytesProcessed += originalBytes

	return sm.SaveJobState(state)
}

func (sm *StateManager) IncrementErrorCount(jobID string, errorMsg string) error {
	state, err := sm.GetJobState(jobID)
	if err != nil {
		return fmt.Errorf("getting job state: %w", err)
	}

	state.ErrorCount++
	state.ErrorMessage = errorMsg

	return sm.SaveJobState(state)
}

func (sm *StateManager) SaveSystemState(systemState *SystemState) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	systemState.LastHealthCheck = time.Now()
	statePath := filepath.Join(sm.stateDir, "system_state.json")
	return sm.saveStateFile(systemState, statePath)
}

func (sm *StateManager) GetSystemState() (*SystemState, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	statePath := filepath.Join(sm.stateDir, "system_state.json")
	
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default system state
			return &SystemState{
				ActiveJobs:      []string{},
				LastHealthCheck: time.Now(),
				SystemStats:     SystemStats{},
				BackupStatistics: BackupStatistics{},
			}, nil
		}
		return nil, fmt.Errorf("reading system state: %w", err)
	}

	var state SystemState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing system state: %w", err)
	}

	return &state, nil
}

func (sm *StateManager) UpdateBackupStatistics(stats BackupStatistics) error {
	systemState, err := sm.GetSystemState()
	if err != nil {
		return fmt.Errorf("getting system state: %w", err)
	}

	systemState.BackupStatistics = stats
	return sm.SaveSystemState(systemState)
}

func (sm *StateManager) GetAllJobStates() (map[string]*JobState, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	files, err := filepath.Glob(filepath.Join(sm.stateDir, "progress-*.json"))
	if err != nil {
		return nil, fmt.Errorf("listing state files: %w", err)
	}

	states := make(map[string]*JobState)
	for _, file := range files {
		filename := filepath.Base(file)
		// Extract job ID from filename: progress-{jobID}.json
		jobID := filename[9 : len(filename)-5] // Remove "progress-" prefix and ".json" suffix

		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip corrupted files
		}

		var state JobState
		if err := json.Unmarshal(data, &state); err != nil {
			continue // Skip corrupted files
		}

		states[jobID] = &state
	}

	return states, nil
}

func (sm *StateManager) CleanupOldStates(daysToKeep int) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	cutoff := time.Now().AddDate(0, 0, -daysToKeep)

	files, err := filepath.Glob(filepath.Join(sm.stateDir, "progress-*.json"))
	if err != nil {
		return fmt.Errorf("listing state files: %w", err)
	}

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			os.Remove(file)
		}
	}

	return nil
}

func (sm *StateManager) saveStateFile(data interface{}, filePath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	// Atomic write using temporary file
	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
		return fmt.Errorf("writing temp state file: %w", err)
	}

	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile) // cleanup on failure
		return fmt.Errorf("moving temp state file: %w", err)
	}

	return nil
}

func (sm *StateManager) GetJobProgress(jobID string) (float64, error) {
	state, err := sm.GetJobState(jobID)
	if err != nil {
		return 0, err
	}

	return state.Progress, nil
}

func (sm *StateManager) GetProcessingStatistics() (map[string]interface{}, error) {
	states, err := sm.GetAllJobStates()
	if err != nil {
		return nil, err
	}

	var totalFiles, totalProcessed, totalErrors int
	var totalOriginal, totalCompressed int64
	var totalElapsed int64

	for _, state := range states {
		if state.Status == JobStatusCompleted || state.Status == JobStatusRunning {
			totalFiles += state.TotalFiles
			totalProcessed += state.ProcessedFiles
			totalErrors += state.ErrorCount
			totalOriginal += state.CompressionStats.OriginalBytes
			totalCompressed += state.CompressionStats.CompressedBytes

			// Calculate total processing time
			for _, phase := range state.PhaseStats {
				totalElapsed += phase.ElapsedMs
			}
		}
	}

	stats := map[string]interface{}{
		"total_files_discovered": totalFiles,
		"total_files_processed":  totalProcessed,
		"total_errors":           totalErrors,
		"total_original_bytes":   totalOriginal,
		"total_compressed_bytes": totalCompressed,
		"total_processing_time_ms": totalElapsed,
		"active_jobs":            len(states),
	}

	if totalOriginal > 0 {
		stats["compression_ratio"] = 1.0 - (float64(totalCompressed) / float64(totalOriginal))
		stats["space_saved_bytes"] = totalOriginal - totalCompressed
	}

	return stats, nil
}