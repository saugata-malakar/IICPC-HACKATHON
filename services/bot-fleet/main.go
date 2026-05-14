package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
)

// Configuration
type Config struct {
	Port              string
	GRPCPort          string
	KafkaBrokers      string
	RedisURL          string
	MaxBotsPerTarget  int
	BotThinkTimeMS    int
	DisruptorRingSize int
}

var (
	cfg         *Config
	redisClient *redis.Client
	kafkaWriter *kafka.Writer
	botRegistry sync.Map // submission_id -> *BotFleet
	orderSeq    atomic.Uint64
)

// Bot Fleet manages thousands of concurrent trading bots
type BotFleet struct {
	SubmissionID string
	Endpoint     string
	Protocol     string
	BotCount     int
	Active       atomic.Bool
	OrdersSent   atomic.Uint64
	FillsRecv    atomic.Uint64
	Errors       atomic.Uint64
	StartTime    time.Time
	StopChan     chan struct{}
	Wg           sync.WaitGroup
}

// Order represents a trading order
type Order struct {
	OrderID     string
	ClientID    string
	Symbol      string
	Side        string // "BUY" or "SELL"
	OrderType   string // "LIMIT" or "MARKET"
	Price       float64
	Quantity    int
	Timestamp   int64
	BotPersona  string
}

// Bot personas with different trading strategies
const (
	PersonaMarketMaker  = "market_maker"
	PersonaArbitrageur  = "arbitrageur"
	PersonaMomentum     = "momentum"
	PersonaMeanRevert   = "mean_revert"
	PersonaNoiseTrader  = "noise_trader"
	PersonaHFT          = "hft"
)

func main() {
	cfg = &Config{
		Port:              getEnv("PORT", "8002"),
		GRPCPort:          getEnv("GRPC_PORT", "9091"),
		KafkaBrokers:      getEnv("KAFKA_BROKERS", "kafka:29092"),
		RedisURL:          getEnv("REDIS_URL", "redis:6379"),
		MaxBotsPerTarget:  5000,
		BotThinkTimeMS:    10,
		DisruptorRingSize: 65536,
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

	// Initialize Kafka writer
	kafkaWriter = &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBrokers),
		Topic:        "order.events",
		Balancer:     &kafka.Hash{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		Compression:  kafka.Snappy,
		Async:        true,
	}
	defer kafkaWriter.Close()
	log.Println("✓ Kafka writer initialized")

	// Start HTTP server
	app := fiber.New(fiber.Config{
		AppName: "IICPC Bot Fleet v2",
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":      "healthy",
			"service":     "bot-fleet",
			"active_fleets": countActiveFleets(),
			"total_orders": orderSeq.Load(),
		})
	})

	app.Post("/spawn", handleSpawnBots)
	app.Post("/stop", handleStopBots)
	app.Get("/status/:submission_id", handleFleetStatus)

	go func() {
		log.Printf("🚀 Bot Fleet HTTP server starting on port %s", cfg.Port)
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Fatalf("❌ HTTP server failed: %v", err)
		}
	}()

	// Start gRPC server
	go startGRPCServer()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down bot fleets...")
	stopAllFleets()
	log.Println("✓ All fleets stopped")
}

func handleSpawnBots(c *fiber.Ctx) error {
	var req struct {
		SubmissionID    string `json:"submission_id"`
		Endpoint        string `json:"endpoint"`
		Protocol        string `json:"protocol"`
		BotCount        int    `json:"bot_count"`
		DurationSeconds int    `json:"duration_seconds"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid_request"})
	}

	if req.BotCount > cfg.MaxBotsPerTarget {
		return c.Status(400).JSON(fiber.Map{
			"error": "bot_count_exceeded",
			"max":   cfg.MaxBotsPerTarget,
		})
	}

	// Create bot fleet
	fleet := &BotFleet{
		SubmissionID: req.SubmissionID,
		Endpoint:     req.Endpoint,
		Protocol:     req.Protocol,
		BotCount:     req.BotCount,
		StartTime:    time.Now(),
		StopChan:     make(chan struct{}),
	}
	fleet.Active.Store(true)

	botRegistry.Store(req.SubmissionID, fleet)

	// Spawn bots
	go fleet.Run(req.DurationSeconds)

	log.Printf("✓ Spawned %d bots for submission %s", req.BotCount, req.SubmissionID)

	return c.JSON(fiber.Map{
		"status":        "spawned",
		"submission_id": req.SubmissionID,
		"bot_count":     req.BotCount,
		"duration":      req.DurationSeconds,
	})
}

func handleStopBots(c *fiber.Ctx) error {
	var req struct {
		SubmissionID string `json:"submission_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid_request"})
	}

	if val, ok := botRegistry.Load(req.SubmissionID); ok {
		fleet := val.(*BotFleet)
		fleet.Stop()
		return c.JSON(fiber.Map{"status": "stopped"})
	}

	return c.Status(404).JSON(fiber.Map{"error": "fleet_not_found"})
}

func handleFleetStatus(c *fiber.Ctx) error {
	submissionID := c.Params("submission_id")

	if val, ok := botRegistry.Load(submissionID); ok {
		fleet := val.(*BotFleet)
		return c.JSON(fiber.Map{
			"submission_id": fleet.SubmissionID,
			"active":        fleet.Active.Load(),
			"bot_count":     fleet.BotCount,
			"orders_sent":   fleet.OrdersSent.Load(),
			"fills_recv":    fleet.FillsRecv.Load(),
			"errors":        fleet.Errors.Load(),
			"uptime_sec":    time.Since(fleet.StartTime).Seconds(),
		})
	}

	return c.Status(404).JSON(fiber.Map{"error": "fleet_not_found"})
}

// Run starts the bot fleet
func (f *BotFleet) Run(durationSeconds int) {
	log.Printf("🤖 Starting %d bots for %d seconds", f.BotCount, durationSeconds)

	// Spawn bot goroutines
	for i := 0; i < f.BotCount; i++ {
		f.Wg.Add(1)
		go f.runBot(i)
	}

	// Auto-stop after duration
	if durationSeconds > 0 {
		time.AfterFunc(time.Duration(durationSeconds)*time.Second, func() {
			f.Stop()
		})
	}

	f.Wg.Wait()
	log.Printf("✓ Bot fleet %s completed", f.SubmissionID)
}

// runBot simulates a single trading bot
func (f *BotFleet) runBot(botID int) {
	defer f.Wg.Done()

	// Assign persona
	personas := []string{
		PersonaMarketMaker,
		PersonaArbitrageur,
		PersonaMomentum,
		PersonaMeanRevert,
		PersonaNoiseTrader,
		PersonaHFT,
	}
	persona := personas[botID%len(personas)]

	// Poisson inter-arrival time (λ = 100 orders/sec per bot)
	lambda := 100.0
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(botID)))

	for {
		select {
		case <-f.StopChan:
			return
		default:
			// Generate order based on persona
			order := f.generateOrder(botID, persona, rng)

			// Send order via Kafka
			if err := f.sendOrder(order); err != nil {
				f.Errors.Add(1)
			} else {
				f.OrdersSent.Add(1)
				orderSeq.Add(1)
			}

			// Poisson wait time: -ln(U) / λ
			waitTime := -math.Log(rng.Float64()) / lambda
			time.Sleep(time.Duration(waitTime*1000) * time.Millisecond)
		}
	}
}

// generateOrder creates an order based on bot persona
func (f *BotFleet) generateOrder(botID int, persona string, rng *rand.Rand) *Order {
	// GBM price simulation: S(t+1) = S(t) * exp((μ - σ²/2)Δt + σ√Δt * Z)
	basePrice := 100.0
	mu := 0.0001  // drift
	sigma := 0.02 // volatility
	dt := 0.01
	z := rng.NormFloat64()
	price := basePrice * math.Exp((mu-sigma*sigma/2)*dt+sigma*math.Sqrt(dt)*z)

	side := "BUY"
	if rng.Float64() > 0.5 {
		side = "SELL"
	}

	orderType := "LIMIT"
	if persona == PersonaHFT || persona == PersonaNoiseTrader {
		if rng.Float64() > 0.7 {
			orderType = "MARKET"
		}
	}

	quantity := 10 + rng.Intn(90) // 10-100 shares

	return &Order{
		OrderID:    fmt.Sprintf("%s-%d-%d", f.SubmissionID, botID, orderSeq.Load()),
		ClientID:   fmt.Sprintf("bot-%d", botID),
		Symbol:     "IICPC",
		Side:       side,
		OrderType:  orderType,
		Price:      math.Round(price*100) / 100,
		Quantity:   quantity,
		Timestamp:  time.Now().UnixNano(),
		BotPersona: persona,
	}
}

// sendOrder publishes order to Kafka
func (f *BotFleet) sendOrder(order *Order) error {
	msg := kafka.Message{
		Key:   []byte(order.OrderID),
		Value: []byte(fmt.Sprintf(`{"order_id":"%s","client_id":"%s","symbol":"%s","side":"%s","type":"%s","price":%.2f,"quantity":%d,"timestamp":%d,"persona":"%s","submission_id":"%s"}`,
			order.OrderID, order.ClientID, order.Symbol, order.Side, order.OrderType,
			order.Price, order.Quantity, order.Timestamp, order.BotPersona, f.SubmissionID)),
	}

	return kafkaWriter.WriteMessages(context.Background(), msg)
}

// Stop gracefully stops the bot fleet
func (f *BotFleet) Stop() {
	if f.Active.CompareAndSwap(true, false) {
		close(f.StopChan)
		log.Printf("🛑 Stopping bot fleet %s", f.SubmissionID)
	}
}

func stopAllFleets() {
	botRegistry.Range(func(key, value interface{}) bool {
		fleet := value.(*BotFleet)
		fleet.Stop()
		return true
	})
}

func countActiveFleets() int {
	count := 0
	botRegistry.Range(func(key, value interface{}) bool {
		fleet := value.(*BotFleet)
		if fleet.Active.Load() {
			count++
		}
		return true
	})
	return count
}

func startGRPCServer() {
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("❌ gRPC listener failed: %v", err)
	}

	grpcServer := grpc.NewServer()
	// Register gRPC services here

	log.Printf("🚀 Bot Fleet gRPC server starting on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ gRPC server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
