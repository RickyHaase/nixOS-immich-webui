package handlers

import (
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/RickyHaase/nixOS-immich-webui/internal/backup/config"
	"github.com/RickyHaase/nixOS-immich-webui/internal/backup/processor"
	"github.com/RickyHaase/nixOS-immich-webui/internal/backup/storage"
)

type BackupHandlers struct {
	config       *config.BackupConfig
	pipeline     *processor.Pipeline
	jobManager   *storage.JobManager
	stateManager *storage.StateManager
	templates    map[string]*htmltemplate.Template
}

type PageData struct {
	Title            string
	BackupConfig     *config.BackupConfig
	Jobs             []*storage.BackupJob
	RunningJobs      []*storage.BackupJob
	RecentJobs       []*storage.BackupJob
	SystemStats      map[string]interface{}
	ProcessingStats  map[string]interface{}
	StorageUsage     StorageInfo
	QualityTiers     []config.QualityTier
	Errors           []string
	Success          string
}

type StorageInfo struct {
	UsedBytes      int64   `json:"used_bytes"`
	TotalBytes     int64   `json:"total_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
	SpacePressure  bool    `json:"space_pressure"`
	AvailableBytes int64   `json:"available_bytes"`
	UsedGB         float64 `json:"used_gb"`
	TotalGB        float64 `json:"total_gb"`
}

func NewBackupHandlers(cfg *config.BackupConfig, templates map[string]*htmltemplate.Template) (*BackupHandlers, error) {
	pipeline, err := processor.NewPipeline(cfg)
	if err != nil {
		return nil, fmt.Errorf("initializing processing pipeline: %w", err)
	}

	jobManager := storage.NewJobManager(cfg.DataDir)
	stateManager := storage.NewStateManager(cfg.DataDir)

	return &BackupHandlers{
		config:       cfg,
		pipeline:     pipeline,
		jobManager:   jobManager,
		stateManager: stateManager,
		templates:    templates,
	}, nil
}

// GET /backup - Main backup dashboard
func (bh *BackupHandlers) HandleBackupDashboard(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Received backup dashboard request")

	data := &PageData{
		Title:        "Internal Backup System",
		BackupConfig: bh.config,
		QualityTiers: bh.config.QualityTiers,
	}

	// Get jobs
	if jobs, err := bh.jobManager.ListJobs(); err == nil {
		data.Jobs = jobs
	}

	if runningJobs, err := bh.jobManager.GetRunningJobs(); err == nil {
		data.RunningJobs = runningJobs
	}

	// Get recent completed jobs
	if recentJobs, err := bh.jobManager.ListJobsByStatus(storage.JobStatusCompleted); err == nil {
		// Limit to last 10
		if len(recentJobs) > 10 {
			recentJobs = recentJobs[:10]
		}
		data.RecentJobs = recentJobs
	}

	// Get statistics
	if stats, err := bh.pipeline.GetProcessingStatistics(); err == nil {
		data.ProcessingStats = stats
	}

	// Get storage info
	data.StorageUsage = bh.getStorageInfo()

	bh.renderTemplate(w, "backup_dashboard", data)
}

// GET /backup/config - Configuration form
func (bh *BackupHandlers) HandleBackupConfig(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Received backup config request")

	data := &PageData{
		Title:        "Backup Configuration",
		BackupConfig: bh.config,
		QualityTiers: bh.config.QualityTiers,
	}

	bh.renderTemplate(w, "backup_config", data)
}

// POST /backup/config - Save configuration
func (bh *BackupHandlers) HandleBackupConfigSave(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received backup config save request")

	err := r.ParseForm()
	if err != nil {
		slog.Error("Error parsing backup config form", "err", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Update configuration from form
	if dataDir := r.FormValue("data_dir"); dataDir != "" {
		bh.config.DataDir = dataDir
	}

	if storageLimitStr := r.FormValue("storage_limit_gb"); storageLimitStr != "" {
		if storageLimit, err := strconv.ParseInt(storageLimitStr, 10, 64); err == nil {
			bh.config.StorageLimitGB = storageLimit
		}
	}

	if retentionStr := r.FormValue("job_retention_days"); retentionStr != "" {
		if retention, err := strconv.Atoi(retentionStr); err == nil {
			bh.config.JobRetentionDays = retention
		}
	}

	if logLevel := r.FormValue("log_level"); logLevel != "" {
		bh.config.LogLevel = logLevel
	}

	if maxConcurrentStr := r.FormValue("max_concurrent_jobs"); maxConcurrentStr != "" {
		if maxConcurrent, err := strconv.Atoi(maxConcurrentStr); err == nil {
			bh.config.ProcessingSettings.MaxConcurrentJobs = maxConcurrent
		}
	}

	// Validate and save configuration
	if validationErrors := config.ValidateBackupConfig(bh.config); len(validationErrors) > 0 {
		data := &PageData{
			Title:        "Backup Configuration",
			BackupConfig: bh.config,
			QualityTiers: bh.config.QualityTiers,
			Errors:       make([]string, len(validationErrors)),
		}
		
		for i, err := range validationErrors {
			data.Errors[i] = err.Error()
		}

		bh.renderTemplate(w, "backup_config", data)
		return
	}

	// Ensure directories exist
	if err := bh.config.EnsureDirectories(); err != nil {
		slog.Error("Failed to create backup directories", "err", err)
		http.Error(w, "Failed to create backup directories", http.StatusInternalServerError)
		return
	}

	// Save configuration
	if err := bh.config.Save(""); err != nil {
		slog.Error("Failed to save backup configuration", "err", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	data := &PageData{
		Title:        "Backup Configuration",
		BackupConfig: bh.config,
		QualityTiers: bh.config.QualityTiers,
		Success:      "Configuration saved successfully",
	}

	bh.renderTemplate(w, "backup_config", data)
}

// POST /backup/jobs/create - Create new backup job
func (bh *BackupHandlers) HandleCreateJob(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received create backup job request")

	err := r.ParseForm()
	if err != nil {
		slog.Error("Error parsing create job form", "err", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Create job from form data
	job := &storage.BackupJob{
		Name:            r.FormValue("name"),
		SourcePath:      r.FormValue("source_path"),
		DestinationPath: r.FormValue("destination_path"),
		QualityTiers:    strings.Split(r.FormValue("quality_tiers"), ","),
		IncludePatterns: strings.Split(r.FormValue("include_patterns"), ","),
		ExcludePatterns: strings.Split(r.FormValue("exclude_patterns"), ","),
	}

	// Parse settings
	if deleteOriginals := r.FormValue("delete_originals"); deleteOriginals == "on" {
		job.Settings.DeleteOriginals = true
	}

	if verifyChecksums := r.FormValue("verify_checksums"); verifyChecksums == "on" {
		job.Settings.VerifyChecksums = true
	}

	if notifyCompletion := r.FormValue("notify_on_completion"); notifyCompletion == "on" {
		job.Settings.NotifyOnCompletion = true
	}

	if spaceLimitStr := r.FormValue("space_limit_gb"); spaceLimitStr != "" {
		if spaceLimit, err := strconv.ParseInt(spaceLimitStr, 10, 64); err == nil {
			job.Settings.SpaceLimitGB = spaceLimit
		}
	}

	// Create the job
	if err := bh.jobManager.CreateJob(job); err != nil {
		slog.Error("Failed to create backup job", "err", err)
		http.Error(w, "Failed to create backup job", http.StatusInternalServerError)
		return
	}

	// Return updated job list component
	jobs, _ := bh.jobManager.ListJobs()
	data := &PageData{
		Jobs:    jobs,
		Success: "Backup job created successfully",
	}

	bh.renderTemplate(w, "job_list", data)
}

// POST /backup/jobs/{id}/start - Start backup job
func (bh *BackupHandlers) HandleStartJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	slog.Info("Received start job request", "job_id", jobID)

	// Start the job
	if err := bh.jobManager.StartJob(jobID); err != nil {
		slog.Error("Failed to start backup job", "job_id", jobID, "err", err)
		http.Error(w, "Failed to start backup job", http.StatusInternalServerError)
		return
	}

	// Start processing in background
	go bh.runBackupJob(jobID)

	// Return updated job status
	job, _ := bh.jobManager.GetJob(jobID)
	bh.renderJobStatus(w, job)
}

// POST /backup/jobs/{id}/stop - Stop backup job
func (bh *BackupHandlers) HandleStopJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	slog.Info("Received stop job request", "job_id", jobID)

	if err := bh.jobManager.CancelJob(jobID); err != nil {
		slog.Error("Failed to stop backup job", "job_id", jobID, "err", err)
		http.Error(w, "Failed to stop backup job", http.StatusInternalServerError)
		return
	}

	// Return updated job status
	job, _ := bh.jobManager.GetJob(jobID)
	bh.renderJobStatus(w, job)
}

// GET /backup/jobs/{id}/status - Get job status
func (bh *BackupHandlers) HandleJobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")

	job, err := bh.jobManager.GetJob(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	bh.renderJobStatus(w, job)
}

// GET /backup/storage - Get storage information
func (bh *BackupHandlers) HandleStorageInfo(w http.ResponseWriter, r *http.Request) {
	storageInfo := bh.getStorageInfo()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(storageInfo)
}

// DELETE /backup/jobs/{id} - Delete backup job
func (bh *BackupHandlers) HandleDeleteJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	slog.Info("Received delete job request", "job_id", jobID)

	if err := bh.jobManager.DeleteJob(jobID); err != nil {
		slog.Error("Failed to delete backup job", "job_id", jobID, "err", err)
		http.Error(w, "Failed to delete backup job", http.StatusInternalServerError)
		return
	}

	// Return updated job list
	jobs, _ := bh.jobManager.ListJobs()
	data := &PageData{
		Jobs:    jobs,
		Success: "Backup job deleted successfully",
	}

	bh.renderTemplate(w, "job_list", data)
}

// Internal helper methods

func (bh *BackupHandlers) runBackupJob(jobID string) {
	job, err := bh.jobManager.GetJob(jobID)
	if err != nil {
		slog.Error("Failed to get job for processing", "job_id", jobID, "err", err)
		return
	}

	// Create processing job
	processingJob := processor.ProcessingJob{
		ID:              jobID,
		SourcePath:      job.SourcePath,
		DestinationPath: job.DestinationPath,
		IncludePatterns: job.IncludePatterns,
		ExcludePatterns: job.ExcludePatterns,
		DeleteOriginals: job.Settings.DeleteOriginals,
		VerifyChecksums: job.Settings.VerifyChecksums,
		MaxConcurrency:  job.Settings.MaxConcurrency,
		ProgressCallback: func(progress processor.ProcessingProgress) {
			// Update job progress in state manager
			stats := storage.JobStatistics{
				TotalFiles:         progress.TotalFiles,
				ProcessedFiles:     progress.ProcessedFiles,
				TotalSizeBytes:     progress.TotalBytes,
				ProcessedSizeBytes: progress.ProcessedBytes,
				LastUpdated:        time.Now(),
			}
			bh.jobManager.UpdateJobProgress(jobID, stats)
		},
	}

	// Run the processing pipeline
	result, err := bh.pipeline.ProcessDirectory(processingJob)
	
	// Update job completion status
	if err != nil {
		bh.jobManager.CompleteJob(jobID, false, err.Error())
		slog.Error("Backup job failed", "job_id", jobID, "err", err)
	} else {
		errorMsg := ""
		if len(result.Errors) > 0 {
			errorMsg = strings.Join(result.Errors, "; ")
		}
		
		success := result.Status == "completed"
		bh.jobManager.CompleteJob(jobID, success, errorMsg)
		
		slog.Info("Backup job completed", 
			"job_id", jobID,
			"status", result.Status,
			"processed_files", result.ProcessedFiles,
			"failed_files", result.FailedFiles,
			"compression_ratio", fmt.Sprintf("%.1f%%", result.CompressionRatio*100),
		)
	}
}

func (bh *BackupHandlers) getStorageInfo() StorageInfo {
	// This would need actual filesystem stats implementation
	// For now, return placeholder data
	totalBytes := bh.config.StorageLimitGB * 1024 * 1024 * 1024
	usedBytes := int64(totalBytes / 4) // Placeholder: 25% used
	
	return StorageInfo{
		UsedBytes:      usedBytes,
		TotalBytes:     totalBytes,
		UsagePercent:   float64(usedBytes) / float64(totalBytes) * 100,
		SpacePressure:  false,
		AvailableBytes: totalBytes - usedBytes,
		UsedGB:         float64(usedBytes) / (1024 * 1024 * 1024),
		TotalGB:        float64(totalBytes) / (1024 * 1024 * 1024),
	}
}

func (bh *BackupHandlers) renderJobStatus(w http.ResponseWriter, job *storage.BackupJob) {
	// Get current state if job is running
	var jobState *storage.JobState
	if job.Status == storage.JobStatusRunning {
		if state, err := bh.stateManager.GetJobState(job.ID); err == nil {
			jobState = state
		}
	}

	data := struct {
		Job   *storage.BackupJob
		State *storage.JobState
	}{
		Job:   job,
		State: jobState,
	}

	bh.renderTemplate(w, "job_status", data)
}

func (bh *BackupHandlers) renderTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	tmpl, exists := bh.templates[templateName]
	if !exists {
		slog.Error("Template not found", "template", templateName)
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("Template execution failed", "template", templateName, "err", err)
		http.Error(w, "Template execution failed", http.StatusInternalServerError)
		return
	}
}