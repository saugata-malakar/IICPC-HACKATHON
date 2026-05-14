package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/kafka-go"
)

type Config struct {
	Port            string
	PostgresDSN     string
	KafkaBrokers    string
	SandboxCPULimit float64
	SandboxMemLimit string
	MaxSubmissionMB int64
}

var (
	cfg         *Config
	db          *pgx.Conn
	kafkaWriter *kafka.Writer
	dockerCli   *client.Client
)

func main() {
	cfg = &Config{
		Port:            getEnv("PORT", "8001"),
		PostgresDSN:     getEnv("POSTGRES_DSN", "postgresql://platform:platform2026@postgres:5432/iicpc"),
		KafkaBrokers:    getEnv("KAFKA_BROKERS", "kafka:29092"),
		SandboxCPULimit: 2.0,
		SandboxMemLimit: "512m",
		MaxSubmissionMB: 50,
	}

	var err error
	// Initialize Docker client
	dockerCli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("❌ Docker client failed: %v", err)
	}
	defer dockerCli.Close()
	log.Println("✓ Docker client connected")

	// Initialize Postgres
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err = pgx.Connect(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("❌ Postgres connection failed: %v", err)
	}
	defer db.Close(context.Background())
	log.Println("✓ Postgres connected")

	// Initialize Kafka
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(cfg.KafkaBrokers),
		Topic:    "submission.events",
		Balancer: &kafka.Hash{},
	}
	defer kafkaWriter.Close()
	log.Println("✓ Kafka writer initialized")

	// Fiber App
	app := fiber.New(fiber.Config{
		BodyLimit: int(cfg.MaxSubmissionMB * 1024 * 1024),
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "service": "submission-service"})
	})

	app.Post("/submit", handleSubmission)
	app.Get("/status/:id", handleStatus)
	app.Post("/rebuild/:id", handleRebuild)

	go func() {
		log.Printf("🚀 Submission Service starting on port %s", cfg.Port)
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Fatalf("❌ HTTP server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down submission service...")
}

func handleSubmission(c *fiber.Ctx) error {
	userID := c.FormValue("user_id")
	teamName := c.FormValue("team_name")
	sandboxType := c.FormValue("sandbox_type", "docker")
	protocol := c.FormValue("protocol", "rest")

	file, err := c.FormFile("code")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "missing_file"})
	}

	submissionID := uuid.New().String()
	uploadDir := filepath.Join("./uploads", submissionID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "upload_dir_creation_failed"})
	}

	filePath := filepath.Join(uploadDir, file.Filename)
	if err := c.SaveFile(file, filePath); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "file_save_failed"})
	}

	// Insert into DB
	_, err = db.Exec(context.Background(),
		"INSERT INTO submissions (id, user_id, team_name, filename, file_size_bytes, file_hash, sandbox_type, protocol, status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		submissionID, userID, teamName, file.Filename, file.Size, "hash_placeholder", sandboxType, protocol, "pending")
	if err != nil {
		log.Printf("❌ DB insert failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "db_insert_failed"})
	}

	// Start build process in background
	go buildAndDeploy(submissionID, uploadDir, file.Filename, sandboxType)

	return c.JSON(fiber.Map{
		"submission_id": submissionID,
		"status":        "pending",
		"message":       "Submission received, building in background",
	})
}

func buildAndDeploy(id, dir, filename, sandboxType string) {
	updateStatus(id, "building")
	log.Printf("🔨 Building submission %s...", id)

	// In a real scenario, we'd detect language, choose Dockerfile, build image
	// For this prototype, we'll simulate building an image from the uploaded code
	imageName := fmt.Sprintf("contestant-%s:latest", id)

	// Create build context
	buildCtx, err := archive.TarWithOptions(dir, &archive.TarOptions{})
	if err != nil {
		updateStatusWithError(id, "failed", "failed to create build context")
		return
	}

	// Docker Build
	buildResponse, err := dockerCli.ImageBuild(context.Background(), buildCtx, types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile", // Expecting a Dockerfile in the upload
		Remove:     true,
	})
	if err != nil {
		updateStatusWithError(id, "failed", fmt.Sprintf("docker build failed: %v", err))
		return
	}
	defer buildResponse.Body.Close()
	io.Copy(os.Stdout, buildResponse.Body)

	updateStatus(id, "deploying")

	// Deploy container with resource limits
	resp, err := dockerCli.ContainerCreate(context.Background(), &container.Config{
		Image: imageName,
		ExposedPorts: map[string]struct{}{
			"8888/tcp": {},
		},
	}, &container.HostConfig{
		Resources: container.Resources{
			CPUShares: int64(cfg.SandboxCPULimit * 1024),
			Memory:    512 * 1024 * 1024, // 512MB
		},
		PortBindings: map[string][]container.PortBinding{
			"8888/tcp": {{HostIP: "0.0.0.0", HostPort: ""}}, // Auto-assign port
		},
	}, nil, nil, "sandbox-"+id)

	if err != nil {
		updateStatusWithError(id, "failed", fmt.Sprintf("container creation failed: %v", err))
		return
	}

	if err := dockerCli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		updateStatusWithError(id, "failed", fmt.Sprintf("container start failed: %v", err))
		return
	}

	// Get assigned port
	inspect, _ := dockerCli.ContainerInspect(context.Background(), resp.ID)
	port := inspect.NetworkSettings.Ports["8888/tcp"][0].HostPort
	endpoint := fmt.Sprintf("http://localhost:%s", port)

	// Update DB with endpoint
	_, err = db.Exec(context.Background(),
		"UPDATE submissions SET status='running', container_id=$1, endpoint=$2, deployed_at=NOW() WHERE id=$3",
		resp.ID, endpoint, id)

	// Notify via Kafka
	msg := fmt.Sprintf(`{"type":"SUBMISSION_READY","id":"%s","endpoint":"%s"}`, id, endpoint)
	kafkaWriter.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(id),
		Value: []byte(msg),
	})

	log.Printf("✅ Submission %s deployed at %s", id, endpoint)
}

func handleStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	var status, endpoint string
	err := db.QueryRow(context.Background(), "SELECT status, endpoint FROM submissions WHERE id=$1", id).Scan(&status, &endpoint)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not_found"})
	}
	return c.JSON(fiber.Map{"id": id, "status": status, "endpoint": endpoint})
}

func handleRebuild(c *fiber.Ctx) error {
	// Logic to restart/rebuild
	return c.JSON(fiber.Map{"message": "rebuild initiated"})
}

func updateStatus(id, status string) {
	db.Exec(context.Background(), "UPDATE submissions SET status=$1 WHERE id=$2", status, id)
}

func updateStatusWithError(id, status, err string) {
	db.Exec(context.Background(), "UPDATE submissions SET status=$1, error_message=$2 WHERE id=$3", status, err, id)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
