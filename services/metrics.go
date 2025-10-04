package services

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics содержит все метрики для мониторинга
type Metrics struct {
	// Счетчики задач
	JobsTotal      *prometheus.CounterVec
	JobsSuccessful prometheus.Counter
	JobsFailed     *prometheus.CounterVec

	// Время выполнения
	JobDuration      prometheus.Histogram
	FFmpegDuration   prometheus.Histogram
	DownloadDuration prometheus.Histogram
	UploadDuration   prometheus.Histogram

	// Размеры файлов
	ImageSizeBytes prometheus.Histogram
	AudioSizeBytes prometheus.Histogram
	VideoSizeBytes prometheus.Histogram

	// Активные задачи
	ActiveJobs prometheus.Gauge

	// Разрешения видео
	VideoResolutions *prometheus.CounterVec
}

// NewMetrics создает и регистрирует все метрики
func NewMetrics() *Metrics {
	return &Metrics{
		JobsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "avcompression_jobs_total",
				Help: "Total number of jobs processed",
			},
			[]string{"status"}, // success, failed
		),

		JobsSuccessful: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "avcompression_jobs_successful_total",
				Help: "Total number of successful jobs",
			},
		),

		JobsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "avcompression_jobs_failed_total",
				Help: "Total number of failed jobs by error type",
			},
			[]string{"error_type", "operation"}, // minio/ffmpeg, download/upload/encode
		),

		JobDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "avcompression_job_duration_seconds",
				Help:    "Time taken to process a job",
				Buckets: prometheus.ExponentialBuckets(0.5, 2, 10), // 0.5s to ~256s
			},
		),

		FFmpegDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "avcompression_ffmpeg_duration_seconds",
				Help:    "Time taken for FFmpeg encoding",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
			},
		),

		DownloadDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "avcompression_download_duration_seconds",
				Help:    "Time taken to download files from MinIO",
				Buckets: prometheus.ExponentialBuckets(0.05, 2, 10),
			},
		),

		UploadDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "avcompression_upload_duration_seconds",
				Help:    "Time taken to upload files to MinIO",
				Buckets: prometheus.ExponentialBuckets(0.05, 2, 10),
			},
		),

		ImageSizeBytes: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "avcompression_image_size_bytes",
				Help:    "Size of input images",
				Buckets: prometheus.ExponentialBuckets(1024, 2, 15), // 1KB to ~16MB
			},
		),

		AudioSizeBytes: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "avcompression_audio_size_bytes",
				Help:    "Size of input audio files",
				Buckets: prometheus.ExponentialBuckets(1024, 2, 15),
			},
		),

		VideoSizeBytes: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "avcompression_video_size_bytes",
				Help:    "Size of output video files",
				Buckets: prometheus.ExponentialBuckets(1024, 2, 20), // 1KB to ~512MB
			},
		),

		ActiveJobs: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "avcompression_active_jobs",
				Help: "Number of jobs currently being processed",
			},
		),

		VideoResolutions: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "avcompression_video_resolutions_total",
				Help: "Count of videos created by resolution",
			},
			[]string{"resolution"}, // 1080p, 720p, 1:1, etc.
		),
	}
}
