package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

type Config struct {
	Port                  string
	RedisURL              string
	KafkaBrokers          string
	WSMaxConnections      int
	ScoreWeightLatency    float64
	ScoreWeightTPS        float64
	ScoreWeightCorrectness float64
}

var (
	cfg         *Config
	redisClient *redis.Client
	hub         *WebSocketHub
)

// WebSocketHub manages all connected clients
type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("✓ Client connected. Total: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
			log.Printf("✓ Client disconnected. Total: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
					log.Printf("❌ Write error: %v", err)
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

type Submission struct {
	ID                    string    `json:"id"`
	TeamName              string    `json:"team_name"`
	Rank                  int       `json:"rank"`
	Score                 float64   `json:"score"`
	P50Latency            float64   `json:"p50_latency"`
	P90Latency            float64   `json:"p90_latency"`
	P99Latency            float64   `json:"p99_latency"`
	TPS                   int64     `json:"tps"`
	Correctness           float64   `json:"correctness"`
	ChaosBonus            float64   `json:"chaos_bonus"`
	Status                string    `json:"status"`
	Timestamp             int64     `json:"timestamp"`
	Sparkline             []float64 `json:"sparkline"`
	PredictedSaturationTPS *int64   `json:"predicted_saturation_tps,omitempty"`
}

func main() {
	cfg = &Config{
		Port:                   getEnv("PORT", "8003"),
		RedisURL:               getEnv("REDIS_URL", "redis:6379"),
		KafkaBrokers:           getEnv("KAFKA_BROKERS", "kafka:29092"),
		WSMaxConnections:       10000,
		ScoreWeightLatency:     0.40,
		ScoreWeightTPS:         0.35,
		ScoreWeightCorrectness: 0.25,
	}

	// Initialize Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Redis connection failed: %v", err)
	}
	log.Println("✓ Redis connected")

	// Initialize WebSocket hub
	hub = NewWebSocketHub()
	go hub.Run()

	// Start Kafka consumer
	go consumeTelemetryEvents()

	// Start periodic leaderboard broadcast
	go broadcastLeaderboard()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "IICPC Leaderboard Service v2",
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":           "healthy",
			"service":          "leaderboard-service",
			"connected_clients": len(hub.clients),
		})
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		defer func() {
			hub.unregister <- c
		}()

		hub.register <- c

		// Send initial leaderboard
		leaderboard, _ := getLeaderboard(context.Background())
		data, _ := json.Marshal(map[string]interface{}{
			"type":        "leaderboard_update",
			"submissions": leaderboard,
		})
		c.WriteMessage(websocket.TextMessage, data)

		// Keep connection alive
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				break
			}
		}
	}))

	app.Get("/api/leaderboard", func(c *fiber.Ctx) error {
		leaderboard, err := getLeaderboard(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(leaderboard)
	})

	log.Printf("🚀 Leaderboard Service starting on port %s", cfg.Port)
	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Fatalf("❌ Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down...")
	app.Shutdown()
}

func getLeaderboard(ctx context.Context) ([]Submission, error) {
	// Get all submissions from Redis sorted set
	results, err := redisClient.ZRevRangeWithScores(ctx, "leaderboard:live", 0, -1).Result()
	if err != nil {
		return nil, err
	}

	submissions := make([]Submission, 0, len(results))
	for idx, result := range results {
		submissionID := result.Member.(string)

		// Get submission details from Redis hash
		data, err := redisClient.HGetAll(ctx, "submission:"+submissionID).Result()
		if err != nil {
			continue
		}

		sub := Submission{
			ID:       submissionID,
			TeamName: data["team_name"],
			Rank:     idx + 1,
			Score:    result.Score,
			Status:   data["status"],
		}

		// Parse metrics
		if val, ok := data["p99_latency"]; ok {
			sub.P99Latency = parseFloat(val)
		}
		if val, ok := data["tps"]; ok {
			sub.TPS = parseInt(val)
		}
		if val, ok := data["correctness"]; ok {
			sub.Correctness = parseFloat(val)
		}

		// Generate sparkline (last 20 data points)
		sparkline := make([]float64, 20)
		for i := 0; i < 20; i++ {
			sparkline[i] = 50 + float64(i)*2 + (float64(i%3) * 5)
		}
		sub.Sparkline = sparkline

		submissions = append(submissions, sub)
	}

	return submissions, nil
}

func calculateScore(p99Latency, tps, correctness, chaosBonus float64) float64 {
	// Latency score: 1.0 at p99=100μs, 0.0 at p99=100ms
	latScore := math.Max(0, 1-math.Log(p99Latency/0.1)/math.Log(1000))

	// TPS score: logarithmic; 1.0 at 1M TPS
	tpsScore := math.Min(1, math.Log(tps)/math.Log(1_000_000))

	// Composite score
	composite := (cfg.ScoreWeightLatency*latScore +
		cfg.ScoreWeightTPS*tpsScore +
		cfg.ScoreWeightCorrectness*correctness) * (1 + chaosBonus)

	return composite
}

func consumeTelemetryEvents() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{cfg.KafkaBrokers},
		Topic:   "telemetry.aggregated",
		GroupID: "leaderboard-service",
	})
	defer reader.Close()

	log.Println("✓ Kafka consumer started")

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("❌ Kafka read error: %v", err)
			continue
		}

		// Process telemetry event and update leaderboard
		var event map[string]interface{}
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			continue
		}

		// Update Redis sorted set
		submissionID := event["submission_id"].(string)
		score := calculateScore(
			event["p99_latency"].(float64),
			event["tps"].(float64),
			event["correctness"].(float64),
			event["chaos_bonus"].(float64),
		)

		redisClient.ZAdd(context.Background(), "leaderboard:live", redis.Z{
			Score:  score,
			Member: submissionID,
		})
	}
}

func broadcastLeaderboard() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		leaderboard, err := getLeaderboard(context.Background())
		if err != nil {
			continue
		}

		data, err := json.Marshal(map[string]interface{}{
			"type":        "leaderboard_update",
			"submissions": leaderboard,
			"timestamp":   time.Now().Unix(),
		})
		if err != nil {
			continue
		}

		hub.broadcast <- data
	}
}

func parseFloat(s string) float64 {
	var f float64
	json.Unmarshal([]byte(s), &f)
	return f
}

func parseInt(s string) int64 {
	var i int64
	json.Unmarshal([]byte(s), &i)
	return i
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
