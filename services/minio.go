package services

import (
	"context"
	"fmt"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/resoul/avcompression/config"
)

type MinioService struct {
	client *minio.Client
}

func NewMinioService(cfg config.MinioConfig) (*MinioService, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.Secure,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &MinioService{client: client}, nil
}

func (s *MinioService) DownloadFile(ctx context.Context, bucket, object, localPath string) error {
	obj, err := s.client.GetObject(ctx, bucket, object, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("get object failed (bucket=%s, object=%s): %w", bucket, object, err)
	}
	defer obj.Close()

	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local file failed (path=%s): %w", localPath, err)
	}
	defer out.Close()

	if _, err = out.ReadFrom(obj); err != nil {
		return fmt.Errorf("write to local file failed: %w", err)
	}

	return nil
}

func (s *MinioService) UploadFile(ctx context.Context, bucket, object, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file failed (path=%s): %w", localPath, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat file failed: %w", err)
	}

	_, err = s.client.PutObject(ctx, bucket, object, file, stat.Size(), minio.PutObjectOptions{
		ContentType: "video/mp4",
	})
	if err != nil {
		return fmt.Errorf("put object failed (bucket=%s, object=%s): %w", bucket, object, err)
	}

	return nil
}
