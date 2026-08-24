package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apisentinel/apisentinel/internal/ai"
	"github.com/apisentinel/apisentinel/internal/config"
	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/service"
	transportgrpc "github.com/apisentinel/apisentinel/internal/transport/grpc"
	transporthttp "github.com/apisentinel/apisentinel/internal/transport/http"
	"github.com/apisentinel/apisentinel/internal/valkey"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Logger Setup
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

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

	queries := database.New(dbPool)

	// 2. Valkey Connection
	valkeyClient, err := valkey.New(cfg.ValkeyURL)
	if err != nil {
		log.Warn().Err(err).Msg("Valkey connection failed (running without cache/streams)")
	} else {
		defer valkeyClient.Close()
	}

	// 3. Services Initialization
	authService := service.NewAuthService(queries, cfg.JWTSecret)
	projectService := service.NewProjectService(queries)
	endpointService := service.NewEndpointService(queries)
	alertService := service.NewAlertService(queries)
	forwardingService := service.NewForwardingService(queries)
	ingestionService := service.NewIngestionService(queries, valkeyClient, alertService, forwardingService)
	requestService := service.NewRequestService(queries)
	replayService := service.NewReplayService(queries)
	mockService := service.NewMockService(queries)
	explainer := ai.NewExplainer("")

	// 4. gRPC Server (Port 50051)
	grpcServer := transportgrpc.NewServer(queries, cfg.GRPCPort)
	go func() {
		if err := grpcServer.Start(); err != nil {
			log.Fatal().Err(err).Msg("gRPC server failed")
		}
	}()

	// 5. HTTP Handlers & Router (Port 3001)
	handlers := &transporthttp.Handlers{
		AuthHandler:       transporthttp.NewAuthHandler(authService),
		ProjectHandler:    transporthttp.NewProjectHandler(projectService),
		EndpointHandler:   transporthttp.NewEndpointHandler(endpointService),
		IngestionHandler:  transporthttp.NewIngestionHandler(ingestionService),
		RequestHandler:    transporthttp.NewRequestHandler(requestService),
		SSEHandler:        transporthttp.NewSSEHandler(valkeyClient),
		ReplayHandler:     transporthttp.NewReplayHandler(replayService),
		MockHandler:       transporthttp.NewMockHandler(mockService),
		AIHandler:         transporthttp.NewAIHandler(explainer),
		AlertHandler:      transporthttp.NewAlertHandler(alertService),
		ForwardingHandler: transporthttp.NewForwardingHandler(forwardingService),
	}

	router := transporthttp.SetupRouter(handlers, cfg.JWTSecret)

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
	log.Info().Msg("Shutting down servers gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server forced to shutdown")
	}

	grpcServer.Stop()
	log.Info().Msg("ApiSentinel Backend stopped cleanly.")
}
