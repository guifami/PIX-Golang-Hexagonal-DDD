package main

// @title           PIX API
// @version         1.0
// @description     Simulação do sistema PIX com arquitetura hexagonal + DDD + Kafka
// @host            localhost:8000
// @BasePath        /

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	kafkain "go-api/internal/adapters/in/kafka"
	httpadapter "go-api/internal/adapters/in/httpadapter"
	kafkaout "go-api/internal/adapters/out/kafka"
	pgadapter "go-api/internal/adapters/out/postgres"
	"go-api/internal/application/usecase"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/database"
	infrakafka "go-api/internal/infrastructure/kafka"
	"go-api/internal/infrastructure/logger"

	_ "go-api/docs"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	if err := godotenv.Load(".env.local"); err != nil {
		log.Println("no .env.local file found, using environment variables")
	}

	logger.Init()
	defer logger.Sync()

	logger.Log.Info("starting PIX API")

	cfg := config.Load()

	// --- Infrastructure ---
	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		logger.Log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		logger.Log.Fatal("database ping failed", zap.Error(err))
	}
	logger.Log.Info("database connected")

	// --- Repositories ---
	accountRepo := pgadapter.NewAccountRepository(db)
	pixKeyRepo := pgadapter.NewPixKeyRepository(db)
	txRepo := pgadapter.NewTransactionRepository(db)

	// --- Event Publisher (Kafka out adapter) ---
	publisher := kafkaout.NewKafkaEventPublisher(cfg.KafkaBroker)
	defer publisher.(*kafkaout.KafkaEventPublisher).Close()

	// --- Use Cases ---
	accountUC := usecase.NewAccountService(accountRepo)
	pixKeyUC := usecase.NewPixKeyService(pixKeyRepo, accountRepo)
	paymentUC := usecase.NewPaymentService(txRepo, accountRepo, pixKeyRepo, publisher)

	// --- Kafka Topic Bootstrap ---
	if err := infrakafka.EnsureTopics(context.Background(), cfg.KafkaBroker, []string{
		kafkaout.TopicPaymentInitiated,
		kafkaout.TopicPaymentCompleted,
		kafkaout.TopicPaymentFailed,
	}); err != nil {
		logger.Log.Fatal("failed to ensure kafka topics", zap.Error(err))
	}
	logger.Log.Info("kafka topics ready")

	// --- Kafka Consumers ---
	paymentReader := infrakafka.NewReader(cfg.KafkaBroker, kafkaout.TopicPaymentInitiated, cfg.KafkaGroupID)
	completedReader := infrakafka.NewReader(cfg.KafkaBroker, kafkaout.TopicPaymentCompleted, cfg.KafkaGroupID+"-logger")
	failedReader := infrakafka.NewReader(cfg.KafkaBroker, kafkaout.TopicPaymentFailed, cfg.KafkaGroupID+"-logger")

	paymentConsumer := kafkain.NewPaymentConsumer(paymentReader, paymentUC)
	resultLogger := kafkain.NewResultLogger(completedReader, failedReader)

	consumerCtx, cancelConsumers := context.WithCancel(context.Background())
	defer cancelConsumers()

	go paymentConsumer.Start(consumerCtx)
	resultLogger.Start(consumerCtx)

	defer paymentConsumer.Close()
	defer resultLogger.Close()

	// --- HTTP Router ---
	routerCfg := httpadapter.RouterConfig{
		APIKeys:        httpadapter.ParseAPIKeys(cfg.APIKeys),
		AllowedOrigins: httpadapter.ParseAllowedOrigins(cfg.AllowedOrigins),
		RateLimitRPS:   cfg.RateLimitRPS,
	}
	router := httpadapter.SetupRouter(accountUC, pixKeyUC, paymentUC, db, routerCfg)

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	// --- Graceful Shutdown ---
	go func() {
		logger.Log.Info("HTTP server running", zap.String("port", cfg.AppPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("shutting down server...")
	cancelConsumers()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("server forced to shutdown", zap.Error(err))
	}

	logger.Log.Info("server stopped")
}
