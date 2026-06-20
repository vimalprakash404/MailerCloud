package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"mailercloud-backend/config"
	"mailercloud-backend/db"
	"mailercloud-backend/handler"
	"mailercloud-backend/ingestion"
	"mailercloud-backend/logger"
	"mailercloud-backend/queue"
	"mailercloud-backend/store"
)

func main() {
	// ── Structured logging (must be first) ───────────────────────
	logger.Init()
	slog.Info("starting MailerCloud backend")

	// ── Centralized configuration ───────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// ── Database ────────────────────────────────────────────────
	database, err := db.NewMySQL(cfg.DB)
	if err != nil {
		slog.Error("failed to connect to MySQL", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// ── Redis Queue ─────────────────────────────────────────────
	redisQueue, err := queue.NewRedisQueue(cfg.Redis)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisQueue.Close()

	// ── Repository (Store layer) ────────────────────────────────
	campaignStore := store.NewCampaignStore(database)

	// ── Batcher (consumes from queue → flushes to store) ────────
	// Uses functional options — self-documenting, order-independent.
	batcher := ingestion.NewBatcher(redisQueue, campaignStore,
		ingestion.WithBatchSize(cfg.Batcher.BatchSize),
		ingestion.WithWorkers(cfg.Batcher.NumWorkers),
		ingestion.WithFlushInterval(cfg.Batcher.FlushInterval),
	)
	batcher.Start()

	// ── Router ──────────────────────────────────────────────────
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(handler.RequestLogger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// API endpoints — handlers accept interfaces, not concrete types
	eventsHandler := handler.NewEventsHandler(redisQueue)
	statsHandler := handler.NewStatsHandler(campaignStore)

	r.Post("/events", eventsHandler.ServeHTTP)
	r.Post("/events/batch", eventsHandler.ServeBatchHTTP)
	r.Get("/campaigns/{id}/stats", statsHandler.ServeHTTP)

	// ── HTTP Server ─────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Graceful shutdown
	go func() {
		slog.Info("listening", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.WriteTimeout)
	defer cancel()

	srv.Shutdown(ctx)
	batcher.Stop()

	slog.Info("shutdown complete")
}
