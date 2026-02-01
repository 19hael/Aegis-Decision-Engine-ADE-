package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aegis-decision-engine/ade/internal/config"
	"github.com/aegis-decision-engine/ade/internal/decision"
	"github.com/aegis-decision-engine/ade/internal/ingest"
	"github.com/aegis-decision-engine/ade/internal/policy"
	"github.com/aegis-decision-engine/ade/internal/state"
	"github.com/aegis-decision-engine/ade/internal/storage/kafka"
	"github.com/aegis-decision-engine/ade/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("starting ADE server", "version", cfg.Version, "port", cfg.Port)

	// Initialize PostgreSQL
	pgClient, err := postgres.NewClient(cfg)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		// Continue without DB for now in dev
		slog.Warn("running without database connection")
	} else {
		defer pgClient.Close()
		slog.Info("connected to postgres")
	}

	// Initialize Kafka
	kafkaClient := kafka.NewClient(cfg.KafkaBrokers)
	if err := kafkaClient.Health(context.Background()); err != nil {
		slog.Warn("kafka not available", "error", err)
	}

	// Create Kafka writer for events topic
	eventsWriter := kafkaClient.NewWriter("ade.events")
	defer eventsWriter.Close()

	// Initialize stores
	var eventStore *postgres.EventStore
	var featureStore *postgres.FeatureStore
	if pgClient != nil {
		eventStore = postgres.NewEventStore(pgClient)
		featureStore = postgres.NewFeatureStore(pgClient)
	}

	// Initialize services
	ingestService := ingest.NewService(eventStore, eventsWriter, logger)
	ingestHandler := ingest.NewHandler(ingestService)

	stateService := state.NewService(eventStore, featureStore, logger)
	stateHandler := state.NewHandler(stateService)

	// Initialize decision service
	policyEngine := policy.NewEngine(logger)
	var decisionStore *postgres.DecisionStore
	if pgClient != nil {
		decisionStore = postgres.NewDecisionStore(pgClient)
	}
	decisionService := decision.NewService(policyEngine, decisionStore, logger)
	decisionHandler := decision.NewHandler(decisionService)
	
	// Load default policy
	if err := decisionHandler.LoadDefaultPolicy(); err != nil {
		slog.Warn("failed to load default policy", "error", err)
	}

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readyHandler(pgClient))
	
	// Register service routes
	ingestHandler.RegisterRoutes(mux)
	stateHandler.RegisterRoutes(mux)
	decisionHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("server started", "addr", server.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func readyHandler(pgClient *postgres.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ready"
		code := http.StatusOK

		if pgClient != nil {
			if err := pgClient.Health(r.Context()); err != nil {
				status = "not_ready"
				code = http.StatusServiceUnavailable
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		w.Write([]byte(`{"status":"` + status + `"}`))
	}
}
