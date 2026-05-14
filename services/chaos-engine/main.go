package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"github.com/gofiber/fiber/v2"
	"github.com/segmentio/kafka-go"
)

func main() {
	port := getEnv("PORT", "8005")
	kafkaBrokers := getEnv("KAFKA_BROKERS", "kafka:29092")

	// Initialize Docker client
	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("❌ Docker client failed: %v", err)
	}
	defer dockerCli.Close()

	// Initialize Chaos Engine
	engine := NewChaosEngine(dockerCli)

	// Kafka Reader for submission events
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaBrokers},
		Topic:   "submission.events",
		GroupID: "chaos-engine",
	})
	defer reader.Close()

	// Fiber App for health checks and status
	app := fiber.New()
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "service": "chaos-engine"})
	})

	// Start Kafka consumer
	go func() {
		log.Println("🤖 Chaos Engine listening for submission events...")
		for {
			m, err := reader.ReadMessage(context.Background())
			if err != nil {
				log.Printf("❌ Kafka read error: %v", err)
				break
			}

			var event struct {
				Type        string `json:"type"`
				ID          string `json:"id"`
				ContainerID string `json:"container_id"`
			}
			if err := json.Unmarshal(m.Value, &event); err == nil {
				if event.Type == "SUBMISSION_READY" {
					log.Printf("🔥 Starting chaos session for submission %s (container %s)", event.ID, event.ContainerID)
					engine.StartSession(context.Background(), event.ID, event.ContainerID)
				}
			}
		}
	}()

	// Start HTTP server
	go func() {
		log.Printf("🚀 Chaos Engine starting on port %s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("❌ HTTP server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down chaos engine...")
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
