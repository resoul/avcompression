package main

import (
	"os"

	"github.com/resoul/avcompression/config"
	"github.com/resoul/avcompression/services"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg := config.MustLoad()
	setupLogger(cfg.App.LogLevel)
	cfg.Print()

	minioService, err := services.NewMinioService(cfg.Minio)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize MinIO service")
	}

	rabbitService, err := services.NewRabbitMQService(cfg.RabbitMQ)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize RabbitMQ service")
	}
	defer rabbitService.Close()

	processor := services.NewProcessor(minioService)

	logrus.Info("Waiting for jobs...")
	if err := rabbitService.Consume(processor.HandleJob); err != nil {
		logrus.WithError(err).Fatal("Failed to consume messages")
	}
}

func setupLogger(level string) {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
	})

	logrus.SetOutput(os.Stdout)

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.Warnf("Invalid log level '%s', defaulting to 'info'", level)
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)
}
