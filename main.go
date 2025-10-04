package main

import (
	"log"

	"github.com/resoul/avcompression/config"
	"github.com/resoul/avcompression/services"
)

func main() {
	cfg := config.Load()
	minioService, err := services.NewMinioService(cfg.Minio)
	if err != nil {
		log.Fatalf("❌ failed to init MinIO: %v", err)
	}

	rabbitService, err := services.NewRabbitMQService(cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("❌ failed to init RabbitMQ: %v", err)
	}
	defer rabbitService.Close()

	processor := services.NewProcessor(minioService)

	log.Println("🎧 Waiting for jobs...")
	if err := rabbitService.Consume(processor.HandleJob); err != nil {
		log.Fatalf("❌ failed to consume messages: %v", err)
	}
}
