package processor

import (
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

type PhotoProcessor struct {
	config     *config.BackupConfig
	tempDir    string
	magickPath string
	exiftool   string
}

type PhotoMetadata struct {
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Format      string    `json:"format"`
	ColorSpace  string    `json:"color_space"`
	DateTaken   time.Time `json:"date_taken"`
	CameraModel string    `json:"camera_model"`
	GPSLocation string    `json:"gps_location"`
	ISO         int       `json:"iso"`
	Aperture    string    `json:"aperture"`
	ShutterSpeed string   `json:"shutter_speed"`
	FileSize    int64     `json:"file_size"`
}

type PhotoProcessingResult struct {
	OriginalPath     string        `json:"original_path"`
	ProcessedPath    string        `json:"processed_path"`
	OriginalSize     int64         `json:"original_size"`
	ProcessedSize    int64         `json:"processed_size"`
	CompressionRatio float64       `json:"compression_ratio"`
	ProcessingTime   time.Duration `json:"processing_time"`
	QualityTier      string        `json:"quality_tier"`
	Metadata         PhotoMetadata `json:"metadata"`
	Error            error         `json:"error,omitempty"`
}

var supportedPhotoFormats = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".tiff": true,
	".tif":  true,
	".heic": true,
	".heif": true,
	".webp": true,
	".bmp":  true,
	".gif":  true,
}

func NewPhotoProcessor(cfg *config.BackupConfig) (*PhotoProcessor, error) {
	// Find ImageMagick convert command (optional)
	magickPath, err := exec.LookPath("convert")
	if err != nil {
		slog.Warn("ImageMagick convert not found - photo processing will be limited to copying files", "err", err)
		magickPath = ""
	}

	// Find exiftool (optional but preferred for metadata)
	exiftool, _ := exec.LookPath("exiftool")

	return &PhotoProcessor{
		config:     cfg,
		tempDir:    cfg.ProcessingSettings.TempDir,
		magickPath: magickPath,
		exiftool:   exiftool,
	}, nil
}

func (pp *PhotoProcessor) IsPhotoFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return supportedPhotoFormats[ext]
}

func (pp *PhotoProcessor) ProcessPhoto(sourcePath, destPath string, tier config.QualityTier) (*PhotoProcessingResult, error) {
	startTime := time.Now()
	
	result := &PhotoProcessingResult{
		OriginalPath: sourcePath,
		ProcessedPath: destPath,
		QualityTier:  tier.Name,
	}

	// Get original file size
	if info, err := os.Stat(sourcePath); err == nil {
		result.OriginalSize = info.Size()
	}

	// Extract metadata first
	metadata, err := pp.extractMetadata(sourcePath)
	if err != nil {
		slog.Debug("Failed to extract metadata", "file", sourcePath, "err", err)
		// Continue processing even if metadata extraction fails
	}
	result.Metadata = metadata

	// Determine if we need to resize or recompress
	needsProcessing := pp.needsProcessing(metadata, tier)
	
	if !needsProcessing || pp.magickPath == "" {
		// Copy file as-is if no processing needed or ImageMagick not available
		if err := pp.copyFile(sourcePath, destPath); err != nil {
			result.Error = fmt.Errorf("copying file: %w", err)
			return result, err
		}
		if pp.magickPath == "" && needsProcessing {
			slog.Debug("ImageMagick not available, copying file without processing", "file", sourcePath)
		}
	} else {
		// Process the image
		if err := pp.processImageWithMagick(sourcePath, destPath, tier, metadata); err != nil {
			result.Error = fmt.Errorf("processing image: %w", err)
			return result, err
		}

		// Preserve metadata based on tier level
		if err := pp.preserveMetadata(sourcePath, destPath, tier.MetadataLevel); err != nil {
			slog.Warn("Failed to preserve metadata", "file", destPath, "err", err)
			// Continue even if metadata preservation fails
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
	
	slog.Debug("Photo processed", 
		"file", filepath.Base(sourcePath),
		"tier", tier.Name,
		"original_size", result.OriginalSize,
		"processed_size", result.ProcessedSize,
		"compression", fmt.Sprintf("%.1f%%", result.CompressionRatio*100),
		"duration", result.ProcessingTime,
	)

	return result, nil
}

func (pp *PhotoProcessor) extractMetadata(filePath string) (PhotoMetadata, error) {
	metadata := PhotoMetadata{
		FileSize: 0,
	}

	// Get file size
	if info, err := os.Stat(filePath); err == nil {
		metadata.FileSize = info.Size()
	}

	// Use exiftool if available, otherwise fallback to identify
	if pp.exiftool != "" {
		return pp.extractMetadataWithExiftool(filePath)
	}

	return pp.extractMetadataWithIdentify(filePath)
}

func (pp *PhotoProcessor) extractMetadataWithExiftool(filePath string) (PhotoMetadata, error) {
	cmd := exec.Command(pp.exiftool, 
		"-ImageWidth", "-ImageHeight", "-FileType", "-ColorSpace",
		"-DateTimeOriginal", "-Make", "-Model", "-ISO", "-FNumber", 
		"-ShutterSpeedValue", "-GPSPosition", "-j", filePath)
	
	output, err := cmd.Output()
	if err != nil {
		return PhotoMetadata{}, fmt.Errorf("exiftool execution failed: %w", err)
	}

	// Parse JSON output from exiftool
	// This is a simplified parser - in production you'd use a proper JSON parser
	metadata := PhotoMetadata{}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "ImageWidth") {
			if width := extractNumber(line); width > 0 {
				metadata.Width = width
			}
		}
		if strings.Contains(line, "ImageHeight") {
			if height := extractNumber(line); height > 0 {
				metadata.Height = height
			}
		}
		if strings.Contains(line, "FileType") {
			metadata.Format = extractString(line)
		}
		if strings.Contains(line, "Model") && !strings.Contains(line, "CameraModel") {
			metadata.CameraModel = extractString(line)
		}
	}

	return metadata, nil
}

func (pp *PhotoProcessor) extractMetadataWithIdentify(filePath string) (PhotoMetadata, error) {
	cmd := exec.Command(pp.magickPath, "identify", "-format", 
		"%w,%h,%m,%[colorspace]", filePath)
	
	output, err := cmd.Output()
	if err != nil {
		return PhotoMetadata{}, fmt.Errorf("identify execution failed: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) < 4 {
		return PhotoMetadata{}, fmt.Errorf("unexpected identify output format")
	}

	metadata := PhotoMetadata{}
	
	if width, err := strconv.Atoi(parts[0]); err == nil {
		metadata.Width = width
	}
	if height, err := strconv.Atoi(parts[1]); err == nil {
		metadata.Height = height
	}
	metadata.Format = parts[2]
	metadata.ColorSpace = parts[3]

	return metadata, nil
}

func (pp *PhotoProcessor) needsProcessing(metadata PhotoMetadata, tier config.QualityTier) bool {
	// Calculate current resolution
	currentResolution := metadata.Width * metadata.Height
	
	// Check if resolution exceeds tier limit
	if currentResolution > tier.PhotoMaxResolution {
		return true
	}

	// Check if format needs conversion (e.g., HEIC to JPEG)
	if metadata.Format == "HEIC" || metadata.Format == "HEIF" {
		return true
	}

	// Check if we need to adjust quality (for JPEG files)
	if metadata.Format == "JPEG" && tier.PhotoQuality < 95 {
		return true
	}

	return false
}

func (pp *PhotoProcessor) processImageWithMagick(sourcePath, destPath string, tier config.QualityTier, metadata PhotoMetadata) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	// Build ImageMagick command
	args := []string{sourcePath}

	// Resize if needed
	currentResolution := metadata.Width * metadata.Height
	if currentResolution > tier.PhotoMaxResolution {
		// Calculate new dimensions maintaining aspect ratio
		ratio := float64(tier.PhotoMaxResolution) / float64(currentResolution)
		newWidth := int(float64(metadata.Width) * ratio)
		newHeight := int(float64(metadata.Height) * ratio)
		
		args = append(args, "-resize", fmt.Sprintf("%dx%d>", newWidth, newHeight))
	}

	// Set quality for JPEG output
	args = append(args, "-quality", strconv.Itoa(tier.PhotoQuality))

	// Convert HEIC/HEIF to JPEG for compatibility
	if metadata.Format == "HEIC" || metadata.Format == "HEIF" {
		// Change extension to .jpg
		destPath = strings.TrimSuffix(destPath, filepath.Ext(destPath)) + ".jpg"
	}

	// Preserve color profile if specified
	args = append(args, "-colorspace", "sRGB")

	// Auto-orient based on EXIF
	args = append(args, "-auto-orient")

	// Strip unnecessary metadata (we'll add back essential metadata later)
	args = append(args, "-strip")

	// Output file
	args = append(args, destPath)

	cmd := exec.Command(pp.magickPath, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ImageMagick conversion failed: %w", err)
	}

	return nil
}

func (pp *PhotoProcessor) preserveMetadata(sourcePath, destPath string, metadataLevel string) error {
	if pp.exiftool == "" {
		return nil // Skip if exiftool not available
	}

	var tags []string
	
	switch metadataLevel {
	case "full":
		// Copy all metadata
		tags = []string{"-all:all=", "-tagsFromFile", sourcePath, "-all:all>all:all"}
	case "essential":
		// Copy essential metadata only
		tags = []string{
			"-all:all=", "-tagsFromFile", sourcePath,
			"-DateTimeOriginal", "-CreateDate", "-Make", "-Model",
			"-GPSLatitude", "-GPSLongitude", "-GPSPosition",
			"-ImageWidth", "-ImageHeight", "-Orientation",
		}
	case "minimal":
		// Copy only date and basic camera info
		tags = []string{
			"-all:all=", "-tagsFromFile", sourcePath,
			"-DateTimeOriginal", "-CreateDate", "-Make", "-Model",
		}
	default:
		return nil // No metadata preservation
	}

	// Add the destination file
	tags = append(tags, destPath)

	cmd := exec.Command(pp.exiftool, tags...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("exiftool metadata preservation failed: %w", err)
	}

	return nil
}

func (pp *PhotoProcessor) copyFile(src, dst string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	cmd := exec.Command("cp", src, dst)
	return cmd.Run()
}

func (pp *PhotoProcessor) BatchProcessPhotos(photos []string, destDir string, tier config.QualityTier, progressCallback func(int, int, string)) ([]*PhotoProcessingResult, error) {
	results := make([]*PhotoProcessingResult, 0, len(photos))
	
	for i, photo := range photos {
		if progressCallback != nil {
			progressCallback(i, len(photos), photo)
		}

		// Generate destination path
		relativePath, _ := filepath.Rel(filepath.Dir(photo), photo)
		destPath := filepath.Join(destDir, relativePath)

		result, err := pp.ProcessPhoto(photo, destPath, tier)
		if err != nil {
			slog.Error("Failed to process photo", "file", photo, "err", err)
			result.Error = err
		}

		results = append(results, result)
	}

	return results, nil
}

func (pp *PhotoProcessor) GetProcessingStats(results []*PhotoProcessingResult) map[string]interface{} {
	stats := map[string]interface{}{
		"total_files":       len(results),
		"successful_files":  0,
		"failed_files":      0,
		"total_original_bytes": int64(0),
		"total_processed_bytes": int64(0),
		"average_compression": 0.0,
		"total_processing_time": int64(0),
	}

	var totalCompression float64
	var successfulFiles int

	for _, result := range results {
		stats["total_original_bytes"] = stats["total_original_bytes"].(int64) + result.OriginalSize
		stats["total_processing_time"] = stats["total_processing_time"].(int64) + result.ProcessingTime.Milliseconds()

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

// Helper functions
func extractNumber(line string) int {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(line)
	if match != "" {
		if num, err := strconv.Atoi(match); err == nil {
			return num
		}
	}
	return 0
}

func extractString(line string) string {
	// Extract quoted string from JSON-like output
	re := regexp.MustCompile(`"([^"]*)"`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}