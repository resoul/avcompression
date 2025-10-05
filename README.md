# avcompression

A scalable video processing worker that consumes jobs from RabbitMQ, processes media files (images and videos) stored in MinIO, and generates video outputs using FFmpeg.

## Overview

`avcompression` is a microservice worker designed to handle video generation tasks. It listens to a RabbitMQ queue for incoming jobs, downloads media (images or videos) and audio files from MinIO object storage, combines them into a video using FFmpeg with intelligent duration synchronization, and uploads the result back to MinIO.

## Features

- **Queue-based processing**: Consumes jobs from RabbitMQ for distributed workload
- **Object storage integration**: Downloads/uploads media files from/to MinIO
- **Multi-format support**:
    - Images: JPG, JPEG, PNG, WebP
    - Videos: MP4, MOV, AVI, MKV, WebM
    - Audio: MP3, WAV, M4A, AAC
- **Smart duration synchronization**:
    - If video is shorter than audio → video loops until audio ends
    - If audio is shorter than video → audio is padded with silence
    - If using image → displays for entire audio duration
- **Smart resolution scaling**: Automatically selects optimal video resolution based on source aspect ratio
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
- **High-quality video encoding**: Uses libx264 with optimized settings
- **Audio replacement**: Replaces existing video audio with uploaded audio track
- **Structured logging**: Beautiful, contextual logs using logrus

## Architecture

```
RabbitMQ Queue → Worker → MinIO (download) → FFmpeg Processing → MinIO (upload)
```

The worker:
1. Connects to RabbitMQ and listens for job messages
2. Downloads source media (image/video) and audio files from MinIO
3. Analyzes media dimensions and duration using ffprobe
4. Calculates optimal output resolution based on aspect ratio
5. Generates video using FFmpeg with proper encoding settings and duration sync
6. Uploads the final video back to MinIO

## Job Message Format

```json
{
  "uuid": "unique-job-id",
  "media": "path/to/file.mp4",
  "audio": "path/to/audio.mp3",
  "bucket": "media-bucket"
}
```

**Note**: The `media` field can point to either an image or video file.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MINIO_ENDPOINT` | `minio:9000` | MinIO server endpoint |
| `MINIO_ACCESS_KEY` | `airlance` | MinIO access key |
| `MINIO_SECRET_KEY` | `airlance` | MinIO secret key |
| `MINIO_SECURE` | `false` | Use SSL for MinIO connection |
| `RABBITMQ_URL` | `amqp://airlance:airlance@rabbitmq:5672/` | RabbitMQ connection URL |
| `RABBITMQ_QUEUE` | `jobs` | RabbitMQ queue name |
| `APP_ENV` | `development` | Application environment |
| `APP_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `APP_WORKER_ID` | `worker-1` | Unique worker identifier |
| `APP_TIMEOUT` | `5m` | Job processing timeout |
| `APP_MAX_RETRIES` | `3` | Maximum retry attempts |

## Requirements

- Go 1.25.1+
- FFmpeg with libx264 codec installed
- ffprobe (usually comes with FFmpeg)
- RabbitMQ server
- MinIO server

## Installation

```bash
# Clone the repository
git clone https://github.com/resoul/avcompression.git
cd avcompression

# Install dependencies
go mod download

# Build the worker
go build -o avcompression
```

## Usage

### Development

```bash
# Copy example env file
cp .env.example .env

# Edit configuration
vim .env

# Run the worker
go run main.go
```

### Production

```bash
# Build binary
go build -o avcompression

# Run with environment variables
export MINIO_ENDPOINT=your-minio:9000
export RABBITMQ_URL=amqp://user:pass@your-rabbitmq:5672/
./avcompression
```

### Docker

```bash
# Build image
docker build -t avcompression:latest .

# Run container
docker run -d \
  --name avcompression-worker \
  -e MINIO_ENDPOINT=minio:9000 \
  -e RABBITMQ_URL=amqp://user:pass@rabbitmq:5672/ \
  avcompression:latest
```

## Project Structure

```
.
├── config/          # Configuration management
│   └── config.go   # Config loading and validation
├── models/          # Data models
│   └── job.go      # Job message structure
├── services/        # Business logic
│   ├── minio.go    # MinIO client wrapper
│   ├── rabbitmq.go # RabbitMQ consumer
│   └── processor.go # Job processing logic
├── main.go         # Application entry point
├── Dockerfile      # Docker image definition
├── .env            # Environment variables (local)
└── README.md       # This file
```

## Processing Flow

### For Images:
1. Download image and audio from MinIO
2. Analyze image dimensions
3. Calculate target resolution based on aspect ratio
4. Create video with image displayed for entire audio duration
5. Encode with libx264 (stillimage tune)
6. Upload to MinIO

### For Videos:
1. Download video and audio from MinIO
2. Analyze video (resolution, duration, audio tracks)
3. Calculate target resolution
4. If video < audio: loop video until audio ends
5. If audio < video: pad audio with silence
6. Replace original audio with new audio track
7. Scale and encode with libx264
8. Upload to MinIO

## FFmpeg Processing Details

### Image to Video
```bash
ffmpeg -loop 1 -i image.jpg -i audio.mp3 \
  -vf scale=1920:1080 \
  -c:v libx264 -tune stillimage \
  -c:a aac -b:a 192k \
  -pix_fmt yuv420p \
  -t <audio_duration> \
  output.mp4
```

### Video + Audio
```bash
ffmpeg -i video.mp4 -i audio.mp3 \
  -filter_complex "[0:v]loop=N:1:0,scale=...,setpts=PTS-STARTPTS[v];[1:a]apad[a]" \
  -map "[v]" -map "[a]" \
  -c:v libx264 -preset medium -crf 23 \
  -c:a aac -b:a 192k \
  -t <max_duration> \
  output.mp4
```

## Logging

The worker uses structured logging with logrus. Log levels can be configured via `APP_LOG_LEVEL`:

```
2025-01-15 10:30:45 INFO Configuration loaded environment=development worker_id=worker-1
2025-01-15 10:30:46 INFO Processing job started job_uuid=abc123 bucket=uploads
2025-01-15 10:30:47 DEBUG Downloaded image file=photo.jpg
2025-01-15 10:30:48 DEBUG Target resolution calculated resolution=1080p width=1920 height=1080
2025-01-15 10:30:55 INFO Job processing completed job_uuid=abc123 duration=9.2s
```

## Performance Considerations

- Each job runs in a separate goroutine for parallel processing
- Temporary files are stored in `/tmp/{job_uuid}` and cleaned up after processing
- Video encoding uses `preset=medium` for balanced speed/quality
- Image encoding uses `tune=stillimage` for optimal quality
- CRF 23 provides good quality with reasonable file sizes

## Error Handling

The worker handles various error scenarios:
- MinIO connection failures
- Invalid media formats
- FFmpeg processing errors
- RabbitMQ connection issues

All errors are logged with context (job UUID, operation, error details) for debugging.

## Roadmap

- [x] Support for video input files
- [x] Smart duration synchronization
- [x] Audio replacement in videos
- [x] Structured logging with logrus
- [ ] Metrics and monitoring (Prometheus)
- [ ] Retry logic with exponential backoff
- [ ] Support for multiple audio tracks
- [ ] Video filters and transitions
- [ ] Batch processing optimization
- [ ] Health check endpoint
- [ ] Kubernetes deployment manifests
- [ ] Horizontal scaling support

## Contributing

This project is under active development. If you have questions about how the worker operates or want to contribute, please create an issue.

## License

not yet

## Support

For issues and questions:
- GitHub Issues: https://github.com/resoul/avcompression/issues
- Documentation: https://github.com/resoul/avcompression/wiki