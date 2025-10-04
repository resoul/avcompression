package config

import "os"

type Config struct {
	Minio    MinioConfig
	RabbitMQ RabbitMQConfig
}

type MinioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Secure    bool
}

type RabbitMQConfig struct {
	URL       string
	QueueName string
}

func Load() *Config {
	return &Config{
		Minio: MinioConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "minio:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "airlance"),
			SecretKey: getEnv("MINIO_SECRET_KEY", "airlance"),
			Secure:    false,
		},
		RabbitMQ: RabbitMQConfig{
			URL:       getEnv("RABBITMQ_URL", "amqp://airlance:airlance@rabbitmq:5672/"),
			QueueName: getEnv("RABBITMQ_QUEUE", "jobs"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
