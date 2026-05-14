package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Configuration from environment
type Config struct {
	Port           string
	GRPCPort       string
	PostgresDSN    string
	RedisURL       string
	KafkaBrokers   string
	JWTSecret      string
	RateLimitRPS   int
	BotFleetGRPC   string
}

// Global dependencies
var (
	cfg         *Config
	redisClient *redis.Client
	grpcConn    *grpc.ClientConn
)

func main() {
	// Load configuration
	cfg = &Config{
		Port:           getEnv("PORT", "8080"),
		GRPCPort:       getEnv("GRPC_PORT", "9090"),
		PostgresDSN:    getEnv("POSTGRES_DSN", ""),
		RedisURL:       getEnv("REDIS_URL", "redis:6379"),
		KafkaBrokers:   getEnv("KAFKA_BROKERS", "kafka:29092"),
		JWTSecret:      getEnv("JWT_SECRET", "iicpc_jwt_supersecret_2026"),
		RateLimitRPS:   10000,
		BotFleetGRPC:   getEnv("BOT_FLEET_GRPC", "bot-fleet:9091"),
	}

	// Initialize Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Redis connection failed: %v", err)
	}
	log.Println("✓ Redis connected")

	// Initialize gRPC connection to bot-fleet
	var err error
	grpcConn, err = grpc.Dial(cfg.BotFleetGRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("❌ gRPC connection to bot-fleet failed: %v", err)
	}
	defer grpcConn.Close()
	log.Println("✓ gRPC connected to bot-fleet")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "IICPC API Gateway v2",
		DisableStartupMessage: false,
		Prefork:               false,
		StrictRouting:         true,
		CaseSensitive:         true,
		ServerHeader:          "IICPC-Gateway",
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:           120 * time.Second,
	})

	// Middleware stack
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "15:04:05",
		TimeZone:   "UTC",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: false,
	}))
	app.Use(limiter.New(limiter.Config{
		Max:        cfg.RateLimitRPS,
		Expiration: 1 * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "rate_limit_exceeded",
				"msg":   "Too many requests. Max 10k RPS per IP.",
			})
		},
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"service":   "api-gateway",
			"timestamp": time.Now().Unix(),
			"version":   "2.0.0",
		})
	})

	// API routes
	api := app.Group("/api/v1")

	// Authentication routes
	auth := api.Group("/auth")
	auth.Post("/register", handleRegister)
	auth.Post("/login", handleLogin)

	// Submission routes (protected)
	submissions := api.Group("/submissions", jwtMiddleware)
	submissions.Post("/", handleSubmissionUpload)
	submissions.Get("/:id", handleGetSubmission)
	submissions.Get("/:id/status", handleSubmissionStatus)
	submissions.Delete("/:id", handleDeleteSubmission)

	// Bot control routes (protected)
	bots := api.Group("/bots", jwtMiddleware)
	bots.Post("/spawn", handleSpawnBots)
	bots.Post("/stop", handleStopBots)
	bots.Get("/status/:submission_id", handleBotStatus)

	// Leaderboard routes (public)
	leaderboard := api.Group("/leaderboard")
	leaderboard.Get("/", handleGetLeaderboard)
	leaderboard.Get("/live", handleLeaderboardSSE)

	// Telemetry routes (protected)
	telemetry := api.Group("/telemetry", jwtMiddleware)
	telemetry.Get("/submission/:id", handleGetTelemetry)
	telemetry.Get("/submission/:id/latency", handleGetLatencyDistribution)

	// Chaos control routes (admin only)
	chaos := api.Group("/chaos", jwtMiddleware, adminMiddleware)
	chaos.Post("/inject", handleInjectChaos)
	chaos.Post("/stop", handleStopChaos)
	chaos.Get("/status/:submission_id", handleChaosStatus)

	// Start server
	log.Printf("🚀 API Gateway starting on port %s", cfg.Port)
	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Fatalf("❌ Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down gracefully...")
	if err := app.Shutdown(); err != nil {
		log.Printf("❌ Shutdown error: %v", err)
	}
	log.Println("✓ Server stopped")
}

// JWT Middleware
func jwtMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(401).JSON(fiber.Map{"error": "missing_token"})
	}

	tokenString := authHeader[7:] // Remove "Bearer "
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(cfg.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return c.Status(401).JSON(fiber.Map{"error": "invalid_token"})
	}

	claims := token.Claims.(jwt.MapClaims)
	c.Locals("user_id", claims["user_id"])
	c.Locals("team_name", claims["team_name"])
	c.Locals("role", claims["role"])

	return c.Next()
}

// Admin middleware
func adminMiddleware(c *fiber.Ctx) error {
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "admin_only"})
	}
	return c.Next()
}

// Handlers (stubs - implement in separate files)
func handleRegister(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "register endpoint"})
}

func handleLogin(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "login endpoint"})
}

func handleSubmissionUpload(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "submission upload endpoint"})
}

func handleGetSubmission(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "get submission endpoint"})
}

func handleSubmissionStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "submission status endpoint"})
}

func handleDeleteSubmission(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "delete submission endpoint"})
}

func handleSpawnBots(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "spawn bots endpoint"})
}

func handleStopBots(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "stop bots endpoint"})
}

func handleBotStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "bot status endpoint"})
}

func handleGetLeaderboard(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "leaderboard endpoint"})
}

func handleLeaderboardSSE(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "leaderboard SSE endpoint"})
}

func handleGetTelemetry(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "telemetry endpoint"})
}

func handleGetLatencyDistribution(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "latency distribution endpoint"})
}

func handleInjectChaos(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "inject chaos endpoint"})
}

func handleStopChaos(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "stop chaos endpoint"})
}

func handleChaosStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"msg": "chaos status endpoint"})
}

// Utility
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
