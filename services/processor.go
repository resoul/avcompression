package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

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

	if err := p.runFFmpeg(imageLocal, audioLocal, outputLocal); err != nil {
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

func (p *Processor) runFFmpeg(imagePath, audioPath, outputPath string) error {
	cmd := exec.Command("ffmpeg",
		"-loop", "1",
		"-i", imagePath,
		"-i", audioPath,
		"-vf", "scale=1280:1280",
		"-c:v", "libx264",
		"-tune", "stillimage",
		"-c:a", "aac",
		"-b:a", "192k",
		"-pix_fmt", "yuv420p",
		"-shortest",
		outputPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
