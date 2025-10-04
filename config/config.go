package config

import (
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Minio    MinioConfig    `envconfig:"MINIO"`
	RabbitMQ RabbitMQConfig `envconfig:"RABBITMQ"`
	Metrics  MetricsConfig  `envconfig:"METRICS"`
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

type MetricsConfig struct {
	Enabled bool   `envconfig:"ENABLED" default:"true"`
	Port    int    `envconfig:"PORT" default:"2112"`
	Path    string `envconfig:"PATH" default:"/metrics"`
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
		log.Println("⚠️  No .env file found, using environment variables")
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
		log.Fatalf("❌ Failed to load config: %v", err)
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

	if c.Metrics.Port < 1024 || c.Metrics.Port > 65535 {
		return fmt.Errorf("metrics port must be between 1024 and 65535")
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
	log.Println("📋 Configuration loaded:")
	log.Printf("  Environment: %s", c.App.Environment)
	log.Printf("  Worker ID: %s", c.App.WorkerID)
	log.Printf("  Log Level: %s", c.App.LogLevel)
	log.Printf("  Timeout: %s", c.App.Timeout)
	log.Printf("  Max Retries: %d", c.App.MaxRetries)
	log.Println("  MinIO:")
	log.Printf("    Endpoint: %s", c.Minio.Endpoint)
	log.Printf("    Secure: %v", c.Minio.Secure)
	log.Printf("    Access Key: %s***", c.Minio.AccessKey[:3])
	log.Println("  RabbitMQ:")
	log.Printf("    URL: amqp://***@%s", extractHost(c.RabbitMQ.URL))
	log.Printf("    Queue: %s", c.RabbitMQ.QueueName)
	log.Println("  Metrics:")
	log.Printf("    Enabled: %v", c.Metrics.Enabled)
	log.Printf("    Port: %d", c.Metrics.Port)
	log.Printf("    Path: %s", c.Metrics.Path)
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
