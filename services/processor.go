package services

import (
	"context"
	"fmt"
	"image"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	_ "image/jpeg"
	_ "image/png"

	"github.com/resoul/avcompression/models"
)

type Processor struct {
	minio *MinioService
}

func NewProcessor(minio *MinioService) *Processor {
	return &Processor{minio: minio}
}

func (p *Processor) HandleJob(job models.JobMessage) {
	fmt.Printf("📥 Processing job %s\n", job.UUID)

	ctx := context.Background()

	tmpDir := filepath.Join("/tmp", job.UUID)
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		log.Printf("❌ failed to create temp dir: %v", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	imageLocal := filepath.Join(tmpDir, filepath.Base(job.ImagePath))
	if err := p.minio.DownloadFile(ctx, job.Bucket, job.ImagePath, imageLocal); err != nil {
		log.Printf("❌ failed to download image: %v", err)
		return
	}

	audioLocal := filepath.Join(tmpDir, filepath.Base(job.AudioPath))
	if err := p.minio.DownloadFile(ctx, job.Bucket, job.AudioPath, audioLocal); err != nil {
		log.Printf("❌ failed to download audio: %v", err)
		return
	}

	outputLocal := filepath.Join(tmpDir, "output.mp4")

	if err := p.createVideoFromImageAndAudio(imageLocal, audioLocal, outputLocal); err != nil {
		log.Printf("❌ ffmpeg failed: %v", err)
		return
	}

	outputObj := filepath.Join(job.UUID, "output.mp4")
	if err := p.minio.UploadFile(ctx, job.Bucket, outputObj, outputLocal); err != nil {
		log.Printf("❌ failed to upload output: %v", err)
		return
	}

	fmt.Printf("✅ Job %s done. Output: %s\n", job.UUID, outputObj)
}

func (p *Processor) createVideoFromImageAndAudio(imagePath, audioPath, outputPath string) error {
	targetWidth, targetHeight, err := p.calculateTargetResolution(imagePath)
	if err != nil {
		return fmt.Errorf("failed to calculate resolution: %w", err)
	}

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
		outputPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (p *Processor) calculateTargetResolution(imagePath string) (int, int, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
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
		{854, 480, 854.0 / 480.0},     // 480p (SD)
		{1280, 720, 1280.0 / 720.0},   // 720p (HD)
		{1920, 1080, 1920.0 / 1080.0}, // 1080p (Full HD)
		{2560, 1440, 2560.0 / 1440.0}, // 1440p (2K)
		{3840, 2160, 3840.0 / 2160.0}, // 2160p (4K)
		{1080, 1080, 1.0},             // 1:1 (Square)
		{1080, 1920, 1080.0 / 1920.0}, // 9:16 (Vertical)
		{1080, 1350, 1080.0 / 1350.0}, // 4:5 (Portrait)
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

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
