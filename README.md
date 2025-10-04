# avcompression

A scalable video processing worker that consumes jobs from RabbitMQ, processes media files stored in MinIO, and generates video outputs using FFmpeg.

## Overview

`avcompression` is a microservice worker designed to handle video generation tasks. It listens to a RabbitMQ queue for incoming jobs, downloads image and audio files from MinIO object storage, combines them into a video using FFmpeg, and uploads the result back to MinIO.

## Features

- **Queue-based processing**: Consumes jobs from RabbitMQ for distributed workload
- **Object storage integration**: Downloads/uploads media files from/to MinIO
- **Smart resolution scaling**: Automatically selects optimal video resolution based on source image aspect ratio
- **Standard resolutions support**:
    - 480p (SD) - 854×480
    - 720p (HD) - 1280×720
    - 1080p (Full HD) - 1920×1080
    - 1440p (2K) - 2560×1440
    - 2160p (4K) - 3840×2160
    - 1:1 (Square) - 1080×1080
    - 9:16 (Vertical) - 1080×1920
    - 4:5 (Portrait) - 1080×1350
- **Concurrent job processing**: Each job runs in a separate goroutine
- **High-quality video encoding**: Uses libx264 with optimized settings for still images

## Architecture

```
RabbitMQ Queue → Worker → MinIO (download) → FFmpeg Processing → MinIO (upload)
```

The worker:
1. Connects to RabbitMQ and listens for job messages
2. Downloads source image and audio files from MinIO
3. Analyzes image dimensions and calculates optimal output resolution
4. Generates video using FFmpeg with proper encoding settings
5. Uploads the final video back to MinIO

## Job Message Format

```json
{
  "uuid": "unique-job-id",
  "image": "path/to/image.jpg",
  "audio": "path/to/audio.mp3",
  "bucket": "media-bucket"
}
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MINIO_ENDPOINT` | `minio:9000` | MinIO server endpoint |
| `MINIO_ACCESS_KEY` | `airlance` | MinIO access key |
| `MINIO_SECRET_KEY` | `airlance` | MinIO secret key |
| `RABBITMQ_URL` | `amqp://airlance:airlance@rabbitmq:5672/` | RabbitMQ connection URL |
| `RABBITMQ_QUEUE` | `jobs` | RabbitMQ queue name |

## Requirements

- Go 1.25.1+
- FFmpeg installed on the system
- RabbitMQ server
- MinIO server

## Installation

```bash
go mod download
go build -o avcompression
```

## Usage

```bash
./avcompression
```

The worker will start listening for jobs and process them automatically.

## Project Structure

```
.
├── config/          # Configuration management
├── models/          # Data models
├── services/        # Business logic
│   ├── minio.go    # MinIO client wrapper
│   ├── rabbitmq.go # RabbitMQ consumer
│   └── processor.go # Job processing logic
└── main.go         # Application entry point
```

## Roadmap

- [ ] Enhanced configuration management with environment variable validation
- [ ] Additional video processing functions (trimming, filters, transitions)
- [ ] Support for multiple input formats
- [ ] Batch processing capabilities
- [ ] Metrics and monitoring integration
- [ ] Error handling improvements and retry logic
- [ ] Docker containerization
- [ ] Kubernetes deployment manifests

## Contributing

This project is under active development and will scale to support more video processing features. If you have questions about how the worker operates or want to contribute, please create an issue.