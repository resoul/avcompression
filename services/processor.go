package services

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "image/jpeg"
	_ "image/png"

	"github.com/resoul/avcompression/models"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	minio *MinioService
}

type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
)

type MediaInfo struct {
	Type     MediaType
	Width    int
	Height   int
	Duration float64
	HasAudio bool
}

func NewProcessor(minio *MinioService) *Processor {
	return &Processor{
		minio: minio,
	}
}

func (p *Processor) HandleJob(job models.JobMessage) {
	startTime := time.Now()

	log := logrus.WithFields(logrus.Fields{
		"job_uuid": job.UUID,
		"bucket":   job.Bucket,
	})

	log.Info("Processing job started")

	if err := p.processJob(job); err != nil {
		log.WithError(err).WithField("duration", time.Since(startTime)).Error("Job processing failed")
		return
	}

	log.WithField("duration", time.Since(startTime)).Info("Job processing completed")
}

func (p *Processor) processJob(job models.JobMessage) error {
	ctx := context.Background()

	tmpDir := filepath.Join("/tmp", job.UUID)
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			logrus.WithError(err).Warn("Failed to cleanup temp directory")
		}
	}()

	mediaLocal := filepath.Join(tmpDir, filepath.Base(job.MediaPath))
	if err := p.minio.DownloadFile(ctx, job.Bucket, job.MediaPath, mediaLocal); err != nil {
		return fmt.Errorf("download media: %w", err)
	}
	logrus.WithField("file", filepath.Base(job.MediaPath)).Debug("Downloaded media")

	audioLocal := filepath.Join(tmpDir, filepath.Base(job.AudioPath))
	if err := p.minio.DownloadFile(ctx, job.Bucket, job.AudioPath, audioLocal); err != nil {
		return fmt.Errorf("download audio: %w", err)
	}
	logrus.WithField("file", filepath.Base(job.AudioPath)).Debug("Downloaded audio")

	mediaInfo, err := p.analyzeMedia(mediaLocal)
	if err != nil {
		return fmt.Errorf("analyze media: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"type":      mediaInfo.Type,
		"width":     mediaInfo.Width,
		"height":    mediaInfo.Height,
		"duration":  mediaInfo.Duration,
		"has_audio": mediaInfo.HasAudio,
	}).Debug("Media analyzed")

	outputLocal := filepath.Join(tmpDir, "output.mp4")
	resolution, err := p.createVideo(mediaLocal, audioLocal, outputLocal, mediaInfo)
	if err != nil {
		return fmt.Errorf("create video: %w", err)
	}
	logrus.WithField("resolution", resolution).Debug("Video created")

	outputObj := filepath.Join(job.UUID, "output.mp4")
	if err := p.minio.UploadFile(ctx, job.Bucket, outputObj, outputLocal); err != nil {
		return fmt.Errorf("upload video: %w", err)
	}
	logrus.WithField("path", outputObj).Debug("Video uploaded")

	return nil
}

func (p *Processor) analyzeMedia(mediaPath string) (*MediaInfo, error) {
	if p.isImage(mediaPath) {
		width, height, err := p.getImageDimensions(mediaPath)
		if err != nil {
			return nil, err
		}
		return &MediaInfo{
			Type:   MediaTypeImage,
			Width:  width,
			Height: height,
		}, nil
	}

	return p.getVideoInfo(mediaPath)
}

func (p *Processor) isImage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp"
}

func (p *Processor) getImageDimensions(imagePath string) (int, int, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return 0, 0, fmt.Errorf("open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, fmt.Errorf("decode image config: %w", err)
	}

	return img.Width, img.Height, nil
}

func (p *Processor) getVideoInfo(videoPath string) (*MediaInfo, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe execution failed: %w", err)
	}

	var result struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}

	info := &MediaInfo{Type: MediaTypeVideo}

	for _, stream := range result.Streams {
		if stream.CodecType == "video" {
			info.Width = stream.Width
			info.Height = stream.Height
		}
		if stream.CodecType == "audio" {
			info.HasAudio = true
		}
	}

	if result.Format.Duration != "" {
		fmt.Sscanf(result.Format.Duration, "%f", &info.Duration)
	}

	return info, nil
}

func (p *Processor) createVideo(mediaPath, audioPath, outputPath string, mediaInfo *MediaInfo) (string, error) {
	audioDuration, err := p.getAudioDuration(audioPath)
	if err != nil {
		return "", fmt.Errorf("get audio duration: %w", err)
	}

	targetWidth, targetHeight := p.calculateTargetResolution(mediaInfo.Width, mediaInfo.Height)
	resolution := p.formatResolution(targetWidth, targetHeight)

	logrus.WithFields(logrus.Fields{
		"resolution":     resolution,
		"width":          targetWidth,
		"height":         targetHeight,
		"audio_duration": audioDuration,
		"media_duration": mediaInfo.Duration,
	}).Debug("Target resolution and duration calculated")

	var cmd *exec.Cmd

	if mediaInfo.Type == MediaTypeImage {
		cmd = p.buildImageCommand(mediaPath, audioPath, outputPath, targetWidth, targetHeight, audioDuration)
	} else {
		cmd = p.buildVideoCommand(mediaPath, audioPath, outputPath, targetWidth, targetHeight, audioDuration, mediaInfo)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg execution failed: %w\nOutput: %s", err, string(output))
	}

	return resolution, nil
}

func (p *Processor) buildImageCommand(imagePath, audioPath, outputPath string, width, height int, audioDuration float64) *exec.Cmd {
	scaleFilter := fmt.Sprintf("scale=%d:%d", width, height)

	return exec.Command("ffmpeg",
		"-loop", "1",
		"-i", imagePath,
		"-i", audioPath,
		"-vf", scaleFilter,
		"-c:v", "libx264",
		"-tune", "stillimage",
		"-c:a", "aac",
		"-b:a", "192k",
		"-pix_fmt", "yuv420p",
		"-color_range", "tv",
		"-colorspace", "bt709",
		"-t", fmt.Sprintf("%.2f", audioDuration),
		"-y",
		outputPath,
	)
}

func (p *Processor) buildVideoCommand(videoPath, audioPath, outputPath string, width, height int, audioDuration float64, mediaInfo *MediaInfo) *exec.Cmd {
	scaleFilter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2", width, height, width, height)

	videoDuration := mediaInfo.Duration
	maxDuration := audioDuration
	if videoDuration > audioDuration {
		maxDuration = videoDuration
	}

	videoFilter := scaleFilter
	if videoDuration < audioDuration {
		loops := int(audioDuration/videoDuration) + 1
		videoFilter = fmt.Sprintf("loop=%d:1:0,%s", loops, scaleFilter)
	}

	args := []string{
		"-i", videoPath,
		"-i", audioPath,
		"-filter_complex",
		fmt.Sprintf("[0:v]%s,setpts=PTS-STARTPTS[v];[1:a]apad[a]", videoFilter),
		"-map", "[v]",
		"-map", "[a]",
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "192k",
		"-pix_fmt", "yuv420p",
		"-color_range", "tv",
		"-colorspace", "bt709",
		"-t", fmt.Sprintf("%.2f", maxDuration),
		"-y",
		outputPath,
	}

	return exec.Command("ffmpeg", args...)
}

func (p *Processor) getAudioDuration(audioPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		audioPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe execution failed: %w", err)
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return 0, fmt.Errorf("parse ffprobe output: %w", err)
	}

	var duration float64
	fmt.Sscanf(result.Format.Duration, "%f", &duration)

	return duration, nil
}

func (p *Processor) calculateTargetResolution(width, height int) (int, int) {
	aspectRatio := float64(width) / float64(height)

	type resolution struct {
		width  int
		height int
		ratio  float64
	}

	standardResolutions := []resolution{
		{854, 480, 854.0 / 480.0},
		{1280, 720, 1280.0 / 720.0},
		{1920, 1080, 1920.0 / 1080.0},
		{2560, 1440, 2560.0 / 1440.0},
		{3840, 2160, 3840.0 / 2160.0},
		{1080, 1080, 1.0},
		{1080, 1920, 1080.0 / 1920.0},
		{1080, 1350, 1080.0 / 1350.0},
	}

	minDiff := 999999.0
	var bestRes resolution

	for _, res := range standardResolutions {
		diff := abs(aspectRatio - res.ratio)
		if diff < minDiff {
			minDiff = diff
			bestRes = res
		}
	}

	if width < bestRes.width && height < bestRes.height {
		return width, height
	}

	return bestRes.width, bestRes.height
}

func (p *Processor) formatResolution(width, height int) string {
	switch {
	case width == 3840 && height == 2160:
		return "4K"
	case width == 2560 && height == 1440:
		return "2K"
	case width == 1920 && height == 1080:
		return "1080p"
	case width == 1280 && height == 720:
		return "720p"
	case width == 854 && height == 480:
		return "480p"
	case width == 1080 && height == 1080:
		return "1:1"
	case width == 1080 && height == 1920:
		return "9:16"
	case width == 1080 && height == 1350:
		return "4:5"
	default:
		return fmt.Sprintf("%dx%d", width, height)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
