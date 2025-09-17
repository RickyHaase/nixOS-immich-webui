package processor

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/RickyHaase/nixOS-immich-webui/internal/backup/config"
)

type VideoProcessor struct {
	config     *config.BackupConfig
	tempDir    string
	ffmpegPath string
	ffprobePath string
}

type VideoMetadata struct {
	Width        int           `json:"width"`
	Height       int           `json:"height"`
	Duration     time.Duration `json:"duration"`
	FrameRate    float64       `json:"frame_rate"`
	Bitrate      int64         `json:"bitrate"`
	Format       string        `json:"format"`
	VideoCodec   string        `json:"video_codec"`
	AudioCodec   string        `json:"audio_codec"`
	FileSize     int64         `json:"file_size"`
	CreationTime time.Time     `json:"creation_time"`
}

type VideoProcessingResult struct {
	OriginalPath     string        `json:"original_path"`
	ProcessedPath    string        `json:"processed_path"`
	OriginalSize     int64         `json:"original_size"`
	ProcessedSize    int64         `json:"processed_size"`
	CompressionRatio float64       `json:"compression_ratio"`
	ProcessingTime   time.Duration `json:"processing_time"`
	QualityTier      string        `json:"quality_tier"`
	Metadata         VideoMetadata `json:"metadata"`
	Error            error         `json:"error,omitempty"`
	ProgressCallback func(float64) `json:"-"`
}

var supportedVideoFormats = map[string]bool{
	".mp4":  true,
	".mov":  true,
	".avi":  true,
	".mkv":  true,
	".m4v":  true,
	".hevc": true,
	".h264": true,
	".h265": true,
	".wmv":  true,
	".flv":  true,
	".webm": true,
}

func NewVideoProcessor(cfg *config.BackupConfig) (*VideoProcessor, error) {
	// Find ffmpeg (optional)
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		slog.Warn("ffmpeg not found - video processing will be limited to copying files", "err", err)
		ffmpegPath = ""
	}

	// Find ffprobe (optional)
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		slog.Warn("ffprobe not found - video metadata extraction will be limited", "err", err)
		ffprobePath = ""
	}

	return &VideoProcessor{
		config:      cfg,
		tempDir:     cfg.ProcessingSettings.TempDir,
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
	}, nil
}

func (vp *VideoProcessor) IsVideoFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return supportedVideoFormats[ext]
}

func (vp *VideoProcessor) ProcessVideo(sourcePath, destPath string, tier config.QualityTier) (*VideoProcessingResult, error) {
	startTime := time.Now()
	
	result := &VideoProcessingResult{
		OriginalPath:  sourcePath,
		ProcessedPath: destPath,
		QualityTier:   tier.Name,
	}

	// Get original file size
	if info, err := os.Stat(sourcePath); err == nil {
		result.OriginalSize = info.Size()
	}

	// Extract metadata
	metadata, err := vp.extractVideoMetadata(sourcePath)
	if err != nil {
		slog.Debug("Failed to extract video metadata", "file", sourcePath, "err", err)
		// Continue processing even if metadata extraction fails
	}
	result.Metadata = metadata

	// Determine if we need to transcode
	needsTranscoding := vp.needsTranscoding(metadata, tier)
	
	if !needsTranscoding || vp.ffmpegPath == "" {
		// Copy file as-is if no transcoding needed or ffmpeg not available
		if err := vp.copyFile(sourcePath, destPath); err != nil {
			result.Error = fmt.Errorf("copying file: %w", err)
			return result, err
		}
		if vp.ffmpegPath == "" && needsTranscoding {
			slog.Debug("ffmpeg not available, copying file without transcoding", "file", sourcePath)
		}
	} else {
		// Transcode the video
		if err := vp.transcodeVideo(sourcePath, destPath, tier, metadata, result); err != nil {
			result.Error = fmt.Errorf("transcoding video: %w", err)
			return result, err
		}
	}

	// Get processed file size
	if info, err := os.Stat(destPath); err == nil {
		result.ProcessedSize = info.Size()
		if result.OriginalSize > 0 {
			result.CompressionRatio = 1.0 - (float64(result.ProcessedSize) / float64(result.OriginalSize))
		}
	}

	result.ProcessingTime = time.Since(startTime)
	
	slog.Debug("Video processed", 
		"file", filepath.Base(sourcePath),
		"tier", tier.Name,
		"original_size", result.OriginalSize,
		"processed_size", result.ProcessedSize,
		"compression", fmt.Sprintf("%.1f%%", result.CompressionRatio*100),
		"duration", result.ProcessingTime,
	)

	return result, nil
}

func (vp *VideoProcessor) extractVideoMetadata(filePath string) (VideoMetadata, error) {
	metadata := VideoMetadata{}

	// Get file size
	if info, err := os.Stat(filePath); err == nil {
		metadata.FileSize = info.Size()
	}

	// Return basic metadata if ffprobe not available
	if vp.ffprobePath == "" {
		return metadata, nil
	}

	// Use ffprobe to get detailed metadata
	cmd := exec.Command(vp.ffprobePath, 
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath)
	
	output, err := cmd.Output()
	if err != nil {
		return metadata, fmt.Errorf("ffprobe execution failed: %w", err)
	}

	// Parse the JSON output (simplified parsing)
	// In production, you'd use a proper JSON parser
	lines := strings.Split(string(output), "\n")
	var inVideoStream bool
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.Contains(line, `"codec_type": "video"`) {
			inVideoStream = true
		} else if strings.Contains(line, `"codec_type": "audio"`) {
			inVideoStream = false
		}

		if inVideoStream {
			if strings.Contains(line, `"width"`) {
				metadata.Width = extractJSONNumber(line)
			}
			if strings.Contains(line, `"height"`) {
				metadata.Height = extractJSONNumber(line)
			}
			if strings.Contains(line, `"codec_name"`) {
				metadata.VideoCodec = extractJSONString(line)
			}
			if strings.Contains(line, `"r_frame_rate"`) {
				rateStr := extractJSONString(line)
				if parts := strings.Split(rateStr, "/"); len(parts) == 2 {
					if num, err := strconv.ParseFloat(parts[0], 64); err == nil {
						if den, err := strconv.ParseFloat(parts[1], 64); err == nil && den != 0 {
							metadata.FrameRate = num / den
						}
					}
				}
			}
		}

		if strings.Contains(line, `"duration"`) {
			durationStr := extractJSONString(line)
			if duration, err := strconv.ParseFloat(durationStr, 64); err == nil {
				metadata.Duration = time.Duration(duration * float64(time.Second))
			}
		}

		if strings.Contains(line, `"bit_rate"`) {
			metadata.Bitrate = int64(extractJSONNumber(line))
		}

		if strings.Contains(line, `"format_name"`) {
			metadata.Format = extractJSONString(line)
		}
	}

	return metadata, nil
}

func (vp *VideoProcessor) needsTranscoding(metadata VideoMetadata, tier config.QualityTier) bool {
	// Check resolution - transcode if higher than tier limit
	if metadata.Height > tier.VideoMaxHeight {
		return true
	}

	// Check frame rate - transcode if higher than tier limit
	if metadata.FrameRate > float64(tier.VideoMaxFPS) {
		return true
	}

	// Check codec - transcode if not x264/h264
	if metadata.VideoCodec != "h264" && metadata.VideoCodec != "x264" {
		return true
	}

	// Always transcode for consistency and optimal compression
	// unless it's already optimized (small enough and good codec)
	return true
}

func (vp *VideoProcessor) transcodeVideo(sourcePath, destPath string, tier config.QualityTier, metadata VideoMetadata, result *VideoProcessingResult) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	// Force .mp4 extension for consistency
	if filepath.Ext(destPath) != ".mp4" {
		destPath = strings.TrimSuffix(destPath, filepath.Ext(destPath)) + ".mp4"
		result.ProcessedPath = destPath
	}

	// Build ffmpeg command
	args := []string{
		"-i", sourcePath,
		"-c:v", "libx264",         // Use x264 video codec
		"-preset", "medium",       // Balance encoding speed vs compression
		"-crf", strconv.Itoa(tier.VideoCRF), // Quality setting
		"-c:a", "aac",            // Use AAC audio codec
		"-b:a", "128k",           // Audio bitrate
		"-movflags", "+faststart", // Optimize for streaming
		"-y",                     // Overwrite output file
	}

	// Set video resolution if needed
	if metadata.Height > tier.VideoMaxHeight {
		// Calculate new width maintaining aspect ratio
		aspectRatio := float64(metadata.Width) / float64(metadata.Height)
		newHeight := tier.VideoMaxHeight
		newWidth := int(float64(newHeight) * aspectRatio)
		
		// Ensure width is even (required for most codecs)
		if newWidth%2 != 0 {
			newWidth--
		}
		
		args = append(args, "-vf", fmt.Sprintf("scale=%d:%d", newWidth, newHeight))
	}

	// Set frame rate if needed
	if metadata.FrameRate > float64(tier.VideoMaxFPS) {
		args = append(args, "-r", strconv.Itoa(tier.VideoMaxFPS))
	}

	// Add output file
	args = append(args, destPath)

	cmd := exec.Command(vp.ffmpegPath, args...)
	
	// Set up progress monitoring
	if result.ProgressCallback != nil {
		return vp.runWithProgress(cmd, metadata.Duration, result.ProgressCallback)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg transcoding failed: %w", err)
	}

	return nil
}

func (vp *VideoProcessor) runWithProgress(cmd *exec.Cmd, totalDuration time.Duration, progressCallback func(float64)) error {
	// Set up stderr pipe to capture ffmpeg progress
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting ffmpeg: %w", err)
	}

	// Parse progress from stderr
	scanner := bufio.NewScanner(stderr)
	progressRegex := regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2})\.(\d{2})`)
	
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			matches := progressRegex.FindStringSubmatch(line)
			if len(matches) >= 4 {
				hours, _ := strconv.Atoi(matches[1])
				minutes, _ := strconv.Atoi(matches[2])
				seconds, _ := strconv.Atoi(matches[3])
				centiseconds, _ := strconv.Atoi(matches[4])
				
				currentTime := time.Duration(hours)*time.Hour +
					time.Duration(minutes)*time.Minute +
					time.Duration(seconds)*time.Second +
					time.Duration(centiseconds)*time.Millisecond*10
				
				if totalDuration > 0 {
					progress := float64(currentTime) / float64(totalDuration) * 100
					if progress <= 100 {
						progressCallback(progress)
					}
				}
			}
		}
	}()

	return cmd.Wait()
}

func (vp *VideoProcessor) copyFile(src, dst string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	cmd := exec.Command("cp", src, dst)
	return cmd.Run()
}

func (vp *VideoProcessor) BatchProcessVideos(videos []string, destDir string, tier config.QualityTier, progressCallback func(int, int, string)) ([]*VideoProcessingResult, error) {
	results := make([]*VideoProcessingResult, 0, len(videos))
	
	for i, video := range videos {
		if progressCallback != nil {
			progressCallback(i, len(videos), video)
		}

		// Generate destination path
		relativePath, _ := filepath.Rel(filepath.Dir(video), video)
		destPath := filepath.Join(destDir, relativePath)

		result, err := vp.ProcessVideo(video, destPath, tier)
		if err != nil {
			slog.Error("Failed to process video", "file", video, "err", err)
			result.Error = err
		}

		results = append(results, result)
	}

	return results, nil
}

func (vp *VideoProcessor) EstimateProcessingTime(metadata VideoMetadata, tier config.QualityTier) time.Duration {
	// Rough estimation based on video duration and quality settings
	// Lower CRF (higher quality) takes longer to encode
	baseMultiplier := float64(tier.VideoCRF) / 23.0 // 23 is a reasonable baseline
	if baseMultiplier < 0.5 {
		baseMultiplier = 0.5
	}
	
	// Estimate processing time as 0.1x to 2x of video duration
	// depending on quality settings and resolution
	processingRatio := 0.1 * baseMultiplier
	
	if metadata.Height > 720 {
		processingRatio *= 2.0 // HD content takes longer
	}
	if metadata.Height > 1080 {
		processingRatio *= 1.5 // 4K content takes much longer
	}

	return time.Duration(float64(metadata.Duration) * processingRatio)
}

func (vp *VideoProcessor) GetProcessingStats(results []*VideoProcessingResult) map[string]interface{} {
	stats := map[string]interface{}{
		"total_files":          len(results),
		"successful_files":     0,
		"failed_files":         0,
		"total_original_bytes": int64(0),
		"total_processed_bytes": int64(0),
		"average_compression":  0.0,
		"total_processing_time": int64(0),
		"total_duration":       int64(0),
	}

	var totalCompression float64
	var successfulFiles int

	for _, result := range results {
		stats["total_original_bytes"] = stats["total_original_bytes"].(int64) + result.OriginalSize
		stats["total_processing_time"] = stats["total_processing_time"].(int64) + result.ProcessingTime.Milliseconds()
		stats["total_duration"] = stats["total_duration"].(int64) + result.Metadata.Duration.Milliseconds()

		if result.Error == nil {
			successfulFiles++
			stats["total_processed_bytes"] = stats["total_processed_bytes"].(int64) + result.ProcessedSize
			totalCompression += result.CompressionRatio
		} else {
			stats["failed_files"] = stats["failed_files"].(int) + 1
		}
	}

	stats["successful_files"] = successfulFiles
	if successfulFiles > 0 {
		stats["average_compression"] = totalCompression / float64(successfulFiles)
	}

	return stats
}

func (vp *VideoProcessor) OptimizeForMobile(sourcePath, destPath string) error {
	// Create mobile-optimized version with smaller size and lower quality
	args := []string{
		"-i", sourcePath,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "28",
		"-maxrate", "1M",
		"-bufsize", "2M",
		"-vf", "scale=720:480:force_original_aspect_ratio=decrease",
		"-c:a", "aac",
		"-b:a", "96k",
		"-movflags", "+faststart",
		"-y", destPath,
	}

	cmd := exec.Command(vp.ffmpegPath, args...)
	return cmd.Run()
}

// Helper functions
func extractJSONNumber(line string) int {
	re := regexp.MustCompile(`:\s*(\d+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		if num, err := strconv.Atoi(matches[1]); err == nil {
			return num
		}
	}
	return 0
}

func extractJSONString(line string) string {
	re := regexp.MustCompile(`:\s*"([^"]*)"`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}