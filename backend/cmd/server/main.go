package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/apisentinel/apisentinel/internal/ai"
	"github.com/apisentinel/apisentinel/internal/config"
	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/service"
	transportgrpc "github.com/apisentinel/apisentinel/internal/transport/grpc"
	transporthttp "github.com/apisentinel/apisentinel/internal/transport/http"
	"github.com/apisentinel/apisentinel/internal/valkey"
	"github.com/apisentinel/apisentinel/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Logger Setup
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	// Only load .env in development environments. In production, secrets must be provided via platform/container environment variables (#240).
	env := strings.ToLower(os.Getenv("APP_ENV"))
	if env == "" {
		env = strings.ToLower(os.Getenv("NODE_ENV"))
	}
	if env != "production" {
		if err := config.LoadDotEnv(".env", "../.env"); err != nil {
			log.Warn().Err(err).Msg("Unable to load local .env file")
		}
	}

	cfg := config.Load()

	log.Info().Msg("Starting ApiSentinel Go Backend...")

	// 1. Database Connection (pgxpool)
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to connect to PostgreSQL")
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("PostgreSQL ping failed")
	}
	log.Info().Msg("Connected to PostgreSQL (PostgreSQL 16)")

	// Auto-migrate database schema
	if err := database.RunMigrations(ctx, dbPool); err != nil {
		log.Fatal().Err(err).Msg("Database migration failed")
	}

	queries := database.New(dbPool)

	// 2. Valkey Connection
	valkeyClient, err := valkey.New(cfg.ValkeyURL)
	if err != nil {
		log.Warn().Err(err).Msg("Valkey connection failed (running without cache/streams)")
	} else {
		defer valkeyClient.Close()
	}

	// 3. Worker Pool & Services Initialization
	workerPool := worker.NewPool(20, 10000)

	encryptionKey := os.Getenv("WEBHOOK_SECRET_ENCRYPTION_KEY")
	if encryptionKey == "" {
		encryptionKey = cfg.JWTSecret
	}

	authService := service.NewAuthService(queries, cfg.JWTSecret)
	projectService := service.NewProjectService(queries)
	endpointService := service.NewEndpointService(queries)
	alertService := service.NewAlertService(queries, workerPool, encryptionKey)
	forwardingService := service.NewForwardingService(queries, workerPool, encryptionKey)
	ingestionService := service.NewIngestionService(queries, valkeyClient, alertService, forwardingService, workerPool)
	requestService := service.NewRequestService(queries)
	replayService := service.NewReplayService(queries)
	mockService := service.NewMockService(queries)
	findingService := service.NewFindingService(queries)
	apiKeyService := service.NewAPIKeyService(queries)
	webhookSecurityService := service.NewWebhookSecurityService(queries, encryptionKey)
	deliveryService := service.NewDeliveryService(queries, workerPool, encryptionKey)
	deliveryService.SetAlertService(alertService)
	explainer := ai.NewExplainer("")

	// 4. gRPC Server (Port 50051) with Token Auth Interceptor & TLS
	grpcServer := transportgrpc.NewServer(queries, cfg.GRPCPort, cfg.JWTSecret, os.Getenv("TLS_CERT_FILE"), os.Getenv("TLS_KEY_FILE"), valkeyClient)
	go func() {
		if err := grpcServer.Start(); err != nil {
			log.Fatal().Err(err).Msg("gRPC server failed")
		}
	}()

	// 5. HTTP Handlers & Router (Port 3001)
	handlers := &transporthttp.Handlers{
		AuthHandler:            transporthttp.NewAuthHandler(authService),
		ProjectHandler:         transporthttp.NewProjectHandler(projectService),
		EndpointHandler:        transporthttp.NewEndpointHandler(endpointService),
		IngestionHandler:       transporthttp.NewIngestionHandler(ingestionService),
		RequestHandler:         transporthttp.NewRequestHandler(requestService),
		SSEHandler:             transporthttp.NewSSEHandler(valkeyClient),
		ReplayHandler:          transporthttp.NewReplayHandler(replayService),
		MockHandler:            transporthttp.NewMockHandler(mockService),
		AIHandler:              transporthttp.NewAIHandler(explainer),
		AlertHandler:           transporthttp.NewAlertHandler(alertService),
		ForwardingHandler:      transporthttp.NewForwardingHandler(forwardingService),
		FindingHandler:         transporthttp.NewFindingHandler(findingService),
		APIKeyHandler:          transporthttp.NewAPIKeyHandler(apiKeyService),
		AgentHandler:           transporthttp.NewAgentHandler(grpcServer),
		WebhookSecurityHandler: transporthttp.NewWebhookSecurityHandler(webhookSecurityService),
		DeliveryHandler:        transporthttp.NewDeliveryHandler(queries, deliveryService),
		TemplateHandler:        transporthttp.NewTemplateHandler(),
	}

	router := transporthttp.SetupRouter(handlers, cfg.JWTSecret, queries, cfg.CORSOrigin)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Graceful shutdown listener
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info().Int("port", cfg.Port).Msgf("ApiSentinel HTTP Gateway listening on http://localhost:%d", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server crashed")
		}
	}()

	<-stop
	log.Info().Msg("Received termination signal. Initiating cascade graceful shutdown...")

	// 1. Stop HTTP Gateway (stop accepting new requests, drain in-flight)
	log.Info().Msg("[1/4] Shutting down HTTP Gateway...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP Gateway forced shutdown")
	} else {
		log.Info().Msg("HTTP Gateway stopped cleanly")
	}

	// 2. Stop gRPC Server (graceful stop for connected agents)
	log.Info().Msg("[2/4] Stopping gRPC Agent Server...")
	grpcServer.Stop()

	// 3. Drain Background Worker Pool
	log.Info().Msg("[3/4] Draining background Worker Pool tasks...")
	if err := workerPool.Stop(5 * time.Second); err != nil {
		log.Error().Err(err).Msg("Worker pool shutdown timeout")
	}

	// 4. Close database and Valkey connections
	log.Info().Msg("[4/4] Closing database & cache connection pools...")
	if valkeyClient != nil {
		_ = valkeyClient.Close()
	}
	dbPool.Close()

	log.Info().Msg("ApiSentinel Backend shutdown completed cleanly.")
}
