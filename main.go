package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/resoul/avcompression/config"
	"github.com/resoul/avcompression/services"
)

func main() {
	cfg := config.MustLoad()
	cfg.Print()

	var metrics *services.Metrics
	if cfg.Metrics.Enabled {
		metrics = services.NewMetrics()
		log.Println("📊 Metrics initialized")
	}

	minioService, err := services.NewMinioService(cfg.Minio)
	if err != nil {
		log.Fatalf("❌ failed to init MinIO: %v", err)
	}

	rabbitService, err := services.NewRabbitMQService(cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("❌ failed to init RabbitMQ: %v", err)
	}
	defer rabbitService.Close()

	processor := services.NewProcessor(minioService, metrics)

	if cfg.Metrics.Enabled {
		go func() {
			http.Handle(cfg.Metrics.Path, promhttp.Handler())
			addr := fmt.Sprintf(":%d", cfg.Metrics.Port)
			log.Printf("📈 Metrics server started on %s%s", addr, cfg.Metrics.Path)
			if err := http.ListenAndServe(addr, nil); err != nil {
				log.Printf("⚠️  metrics server error: %v", err)
			}
		}()
	}

	log.Println("🎧 Waiting for jobs...")
	if err := rabbitService.Consume(processor.HandleJob); err != nil {
		log.Fatalf("❌ failed to consume messages: %v", err)
	}
}
