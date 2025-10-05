package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Minio    MinioConfig    `envconfig:"MINIO"`
	RabbitMQ RabbitMQConfig `envconfig:"RABBITMQ"`
	App      AppConfig      `envconfig:"APP"`
}

type MinioConfig struct {
	Endpoint  string `envconfig:"ENDPOINT" default:"minio:9000" required:"true"`
	AccessKey string `envconfig:"ACCESS_KEY" default:"airlance" required:"true"`
	SecretKey string `envconfig:"SECRET_KEY" default:"airlance" required:"true"`
	Secure    bool   `envconfig:"SECURE" default:"false"`
}

type RabbitMQConfig struct {
	URL       string `envconfig:"URL" default:"amqp://airlance:airlance@rabbitmq:5672/" required:"true"`
	QueueName string `envconfig:"QUEUE" default:"jobs"`
}

type AppConfig struct {
	Environment string        `envconfig:"ENV" default:"development"`
	LogLevel    string        `envconfig:"LOG_LEVEL" default:"info"`
	WorkerID    string        `envconfig:"WORKER_ID" default:"worker-1"`
	Timeout     time.Duration `envconfig:"TIMEOUT" default:"5m"`
	MaxRetries  int           `envconfig:"MAX_RETRIES" default:"3"`
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found, using environment variables")
	}

	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}
	return cfg
}

func (c *Config) Validate() error {
	if c.Minio.Endpoint == "" {
		return fmt.Errorf("minio endpoint is required")
	}
	if c.Minio.AccessKey == "" {
		return fmt.Errorf("minio access key is required")
	}
	if c.Minio.SecretKey == "" {
		return fmt.Errorf("minio secret key is required")
	}

	if c.RabbitMQ.URL == "" {
		return fmt.Errorf("rabbitmq url is required")
	}
	if c.RabbitMQ.QueueName == "" {
		return fmt.Errorf("rabbitmq queue name is required")
	}

	if c.App.Timeout < 1*time.Second {
		return fmt.Errorf("app timeout must be at least 1 second")
	}
	if c.App.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	return nil
}

func (c *Config) Print() {
	logrus.WithFields(logrus.Fields{
		"environment": c.App.Environment,
		"worker_id":   c.App.WorkerID,
		"log_level":   c.App.LogLevel,
		"timeout":     c.App.Timeout,
		"max_retries": c.App.MaxRetries,
	}).Info("Configuration loaded")

	logrus.WithFields(logrus.Fields{
		"endpoint":   c.Minio.Endpoint,
		"secure":     c.Minio.Secure,
		"access_key": c.Minio.AccessKey[:3] + "***",
	}).Info("MinIO configured")

	logrus.WithFields(logrus.Fields{
		"host":  extractHost(c.RabbitMQ.URL),
		"queue": c.RabbitMQ.QueueName,
	}).Info("RabbitMQ configured")
}

func extractHost(url string) string {
	start := len("amqp://")
	for i := start; i < len(url); i++ {
		if url[i] == '@' {
			for j := i + 1; j < len(url); j++ {
				if url[j] == '/' {
					return url[i+1 : j]
				}
			}
			return url[i+1:]
		}
	}
	return "unknown"
}
