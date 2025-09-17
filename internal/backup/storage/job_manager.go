package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type JobManager struct {
	dataDir   string
	jobsDir   string
	stateDir  string
	fileLocks map[string]*sync.RWMutex
	lockMutex sync.RWMutex
}

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusRunning    JobStatus = "running"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCanceled   JobStatus = "canceled"
)

type BackupJob struct {
	ID              string            `yaml:"id"`
	Name            string            `yaml:"name"`
	Status          JobStatus         `yaml:"status"`
	SourcePath      string            `yaml:"source_path"`
	DestinationPath string            `yaml:"destination_path"`
	QualityTiers    []string          `yaml:"quality_tiers"`
	IncludePatterns []string          `yaml:"include_patterns"`
	ExcludePatterns []string          `yaml:"exclude_patterns"`
	ScheduleEnabled bool              `yaml:"schedule_enabled"`
	ScheduleCron    string            `yaml:"schedule_cron"`
	CreatedAt       time.Time         `yaml:"created_at"`
	StartedAt       *time.Time        `yaml:"started_at,omitempty"`
	CompletedAt     *time.Time        `yaml:"completed_at,omitempty"`
	LastRunAt       *time.Time        `yaml:"last_run_at,omitempty"`
	NextRunAt       *time.Time        `yaml:"next_run_at,omitempty"`
	ErrorMessage    string            `yaml:"error_message,omitempty"`
	Settings        JobSettings       `yaml:"settings"`
	Statistics      JobStatistics     `yaml:"statistics"`
}

type JobSettings struct {
	MaxConcurrency      int     `yaml:"max_concurrency"`
	RetryAttempts       int     `yaml:"retry_attempts"`
	DeleteOriginals     bool    `yaml:"delete_originals"`
	VerifyChecksums     bool    `yaml:"verify_checksums"`
	NotifyOnCompletion  bool    `yaml:"notify_on_completion"`
	NotifyOnError       bool    `yaml:"notify_on_error"`
	SpaceLimitGB        int64   `yaml:"space_limit_gb"`
	QualityAdjustment   bool    `yaml:"quality_adjustment"`
}

type JobStatistics struct {
	TotalFiles        int     `yaml:"total_files"`
	ProcessedFiles    int     `yaml:"processed_files"`
	FailedFiles       int     `yaml:"failed_files"`
	SkippedFiles      int     `yaml:"skipped_files"`
	TotalSizeBytes    int64   `yaml:"total_size_bytes"`
	ProcessedSizeBytes int64  `yaml:"processed_size_bytes"`
	CompressionRatio  float64 `yaml:"compression_ratio"`
	ProcessingTimeMs  int64   `yaml:"processing_time_ms"`
	LastUpdated       time.Time `yaml:"last_updated"`
}

func NewJobManager(dataDir string) *JobManager {
	return &JobManager{
		dataDir:   dataDir,
		jobsDir:   filepath.Join(dataDir, "jobs"),
		stateDir:  filepath.Join(dataDir, "state"),
		fileLocks: make(map[string]*sync.RWMutex),
	}
}

func (jm *JobManager) CreateJob(job *BackupJob) error {
	if job.ID == "" {
		job.ID = generateJobID()
	}

	job.CreatedAt = time.Now()
	job.Status = JobStatusPending
	job.Statistics = JobStatistics{
		LastUpdated: time.Now(),
	}

	// Set default settings if not provided
	if job.Settings.MaxConcurrency == 0 {
		job.Settings.MaxConcurrency = 2
	}
	if job.Settings.RetryAttempts == 0 {
		job.Settings.RetryAttempts = 3
	}

	jobPath := filepath.Join(jm.jobsDir, job.ID+".yaml")
	return jm.saveJob(job, jobPath)
}

func (jm *JobManager) GetJob(jobID string) (*BackupJob, error) {
	jobPath := filepath.Join(jm.jobsDir, jobID+".yaml")
	return jm.loadJob(jobPath)
}

func (jm *JobManager) UpdateJob(job *BackupJob) error {
	if job.ID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}

	job.Statistics.LastUpdated = time.Now()
	jobPath := filepath.Join(jm.jobsDir, job.ID+".yaml")
	return jm.saveJob(job, jobPath)
}

func (jm *JobManager) DeleteJob(jobID string) error {
	jobPath := filepath.Join(jm.jobsDir, jobID+".yaml")
	
	// Move to completed directory for record keeping
	completedPath := filepath.Join(jm.jobsDir, "completed", jobID+".yaml")
	if err := os.MkdirAll(filepath.Dir(completedPath), 0755); err != nil {
		return fmt.Errorf("creating completed directory: %w", err)
	}

	if err := os.Rename(jobPath, completedPath); err != nil {
		// If rename fails, just delete
		return os.Remove(jobPath)
	}

	return nil
}

func (jm *JobManager) ListJobs() ([]*BackupJob, error) {
	files, err := filepath.Glob(filepath.Join(jm.jobsDir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("listing job files: %w", err)
	}

	var jobs []*BackupJob
	for _, file := range files {
		job, err := jm.loadJob(file)
		if err != nil {
			continue // Skip corrupted job files
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (jm *JobManager) ListJobsByStatus(status JobStatus) ([]*BackupJob, error) {
	jobs, err := jm.ListJobs()
	if err != nil {
		return nil, err
	}

	var filtered []*BackupJob
	for _, job := range jobs {
		if job.Status == status {
			filtered = append(filtered, job)
		}
	}

	return filtered, nil
}

func (jm *JobManager) StartJob(jobID string) error {
	job, err := jm.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("getting job: %w", err)
	}

	if job.Status != JobStatusPending {
		return fmt.Errorf("job is not in pending status (current: %s)", job.Status)
	}

	// Create lock file to indicate job is running
	lockFile := filepath.Join(jm.jobsDir, "active", jobID+".lock")
	if err := os.MkdirAll(filepath.Dir(lockFile), 0755); err != nil {
		return fmt.Errorf("creating active directory: %w", err)
	}

	if err := os.WriteFile(lockFile, []byte(time.Now().Format(time.RFC3339)), 0644); err != nil {
		return fmt.Errorf("creating lock file: %w", err)
	}

	// Update job status
	now := time.Now()
	job.Status = JobStatusRunning
	job.StartedAt = &now
	job.LastRunAt = &now

	return jm.UpdateJob(job)
}

func (jm *JobManager) CompleteJob(jobID string, success bool, errorMsg string) error {
	job, err := jm.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("getting job: %w", err)
	}

	now := time.Now()
	job.CompletedAt = &now

	if success {
		job.Status = JobStatusCompleted
		job.ErrorMessage = ""
	} else {
		job.Status = JobStatusFailed
		job.ErrorMessage = errorMsg
	}

	// Remove lock file
	lockFile := filepath.Join(jm.jobsDir, "active", jobID+".lock")
	os.Remove(lockFile)

	return jm.UpdateJob(job)
}

func (jm *JobManager) CancelJob(jobID string) error {
	job, err := jm.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("getting job: %w", err)
	}

	if job.Status != JobStatusRunning {
		return fmt.Errorf("job is not running (current: %s)", job.Status)
	}

	now := time.Now()
	job.Status = JobStatusCanceled
	job.CompletedAt = &now

	// Remove lock file
	lockFile := filepath.Join(jm.jobsDir, "active", jobID+".lock")
	os.Remove(lockFile)

	return jm.UpdateJob(job)
}

func (jm *JobManager) UpdateJobProgress(jobID string, stats JobStatistics) error {
	job, err := jm.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("getting job: %w", err)
	}

	stats.LastUpdated = time.Now()
	job.Statistics = stats

	return jm.UpdateJob(job)
}

func (jm *JobManager) IsJobRunning(jobID string) bool {
	lockFile := filepath.Join(jm.jobsDir, "active", jobID+".lock")
	_, err := os.Stat(lockFile)
	return !os.IsNotExist(err)
}

func (jm *JobManager) GetRunningJobs() ([]*BackupJob, error) {
	files, err := filepath.Glob(filepath.Join(jm.jobsDir, "active", "*.lock"))
	if err != nil {
		return nil, fmt.Errorf("listing active jobs: %w", err)
	}

	var jobs []*BackupJob
	for _, file := range files {
		jobID := filepath.Base(file)
		jobID = jobID[:len(jobID)-5] // Remove .lock extension
		
		job, err := jm.GetJob(jobID)
		if err != nil {
			continue // Skip if job file is missing
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (jm *JobManager) CleanupOldJobs(daysToKeep int) error {
	cutoff := time.Now().AddDate(0, 0, -daysToKeep)

	jobs, err := jm.ListJobs()
	if err != nil {
		return fmt.Errorf("listing jobs for cleanup: %w", err)
	}

	for _, job := range jobs {
		shouldDelete := false

		if job.Status == JobStatusCompleted || job.Status == JobStatusFailed {
			if job.CompletedAt != nil && job.CompletedAt.Before(cutoff) {
				shouldDelete = true
			} else if job.CreatedAt.Before(cutoff) {
				shouldDelete = true
			}
		}

		if shouldDelete {
			if err := jm.DeleteJob(job.ID); err != nil {
				// Log error but continue cleanup
				continue
			}
		}
	}

	return nil
}

func (jm *JobManager) CleanupOrphanedLocks() error {
	files, err := filepath.Glob(filepath.Join(jm.jobsDir, "active", "*.lock"))
	if err != nil {
		return fmt.Errorf("listing lock files: %w", err)
	}

	for _, file := range files {
		jobID := filepath.Base(file)
		jobID = jobID[:len(jobID)-5] // Remove .lock extension

		// Check if job still exists
		if _, err := jm.GetJob(jobID); os.IsNotExist(err) {
			// Job doesn't exist, remove orphaned lock
			os.Remove(file)
		}
	}

	return nil
}

func (jm *JobManager) getFileLock(jobID string) *sync.RWMutex {
	jm.lockMutex.Lock()
	defer jm.lockMutex.Unlock()

	if lock, exists := jm.fileLocks[jobID]; exists {
		return lock
	}

	lock := &sync.RWMutex{}
	jm.fileLocks[jobID] = lock
	return lock
}

func (jm *JobManager) saveJob(job *BackupJob, jobPath string) error {
	lock := jm.getFileLock(job.ID)
	lock.Lock()
	defer lock.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(jobPath), 0755); err != nil {
		return fmt.Errorf("creating jobs directory: %w", err)
	}

	data, err := yaml.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshaling job: %w", err)
	}

	// Atomic write using temporary file
	tempFile := jobPath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("writing temp job file: %w", err)
	}

	if err := os.Rename(tempFile, jobPath); err != nil {
		os.Remove(tempFile) // cleanup on failure
		return fmt.Errorf("moving temp job file: %w", err)
	}

	return nil
}

func (jm *JobManager) loadJob(jobPath string) (*BackupJob, error) {
	data, err := os.ReadFile(jobPath)
	if err != nil {
		return nil, fmt.Errorf("reading job file: %w", err)
	}

	var job BackupJob
	if err := yaml.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("parsing job file: %w", err)
	}

	return &job, nil
}

func generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}