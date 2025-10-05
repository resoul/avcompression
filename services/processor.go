package services

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	_ "image/jpeg"
	_ "image/png"

	"github.com/resoul/avcompression/models"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	minio *MinioService
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

	imageLocal := filepath.Join(tmpDir, filepath.Base(job.ImagePath))
	if err := p.minio.DownloadFile(ctx, job.Bucket, job.ImagePath, imageLocal); err != nil {
		return fmt.Errorf("download image: %w", err)
	}
	logrus.WithField("file", filepath.Base(job.ImagePath)).Debug("Downloaded image")

	audioLocal := filepath.Join(tmpDir, filepath.Base(job.AudioPath))
	if err := p.minio.DownloadFile(ctx, job.Bucket, job.AudioPath, audioLocal); err != nil {
		return fmt.Errorf("download audio: %w", err)
	}
	logrus.WithField("file", filepath.Base(job.AudioPath)).Debug("Downloaded audio")

	outputLocal := filepath.Join(tmpDir, "output.mp4")
	resolution, err := p.createVideoFromImageAndAudio(imageLocal, audioLocal, outputLocal)
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

func (p *Processor) createVideoFromImageAndAudio(imagePath, audioPath, outputPath string) (string, error) {
	targetWidth, targetHeight, err := p.calculateTargetResolution(imagePath)
	if err != nil {
		return "", fmt.Errorf("calculate resolution: %w", err)
	}

	resolution := p.formatResolution(targetWidth, targetHeight)

	logrus.WithFields(logrus.Fields{
		"resolution": resolution,
		"width":      targetWidth,
		"height":     targetHeight,
	}).Debug("Target resolution calculated")

	scaleFilter := fmt.Sprintf("scale=%d:%d", targetWidth, targetHeight)

	cmd := exec.Command("ffmpeg",
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
		"-shortest",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg execution failed: %w\nOutput: %s", err, string(output))
	}

	return resolution, nil
}

func (p *Processor) calculateTargetResolution(imagePath string) (int, int, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return 0, 0, fmt.Errorf("open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, fmt.Errorf("decode image config: %w", err)
	}

	width := img.Width
	height := img.Height
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
		return width, height, nil
	}

	return bestRes.width, bestRes.height, nil
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
