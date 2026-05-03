package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"keyloop-service-scheduler/internal/config"
	"keyloop-service-scheduler/internal/database"
	apphttp "keyloop-service-scheduler/internal/http"
	"keyloop-service-scheduler/internal/http/middleware"
	"keyloop-service-scheduler/internal/repository"
	"keyloop-service-scheduler/internal/scheduling"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	var referenceRepo repository.ReferenceRepository
	var availabilityService *scheduling.AvailabilityService
	var appointmentService *scheduling.AppointmentService
	if cfg.DatabaseURL != "" {
		dbPool, err := database.NewPool(context.Background(), cfg.DatabaseURL)
		if err != nil {
			logger.Error("invalid database configuration", "error", err)
			os.Exit(1)
		}
		defer dbPool.Close()
		postgresRepo := repository.NewPostgresReferenceRepository(dbPool)
		referenceRepo = postgresRepo
		availabilityService = scheduling.NewAvailabilityService(postgresRepo)
		appointmentService = scheduling.NewAppointmentService(postgresRepo)
		logger.Info("database pool configured")
	}

	router := apphttp.NewRouter(referenceRepo, availabilityService, appointmentService, logger, middleware.JSONLogger(logger))

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("api server starting", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", "error", err)
			os.Exit(1)
		}
	}()

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-shutdownCtx.Done()
	logger.Info("api server shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("api server shutdown failed", "error", err)
		os.Exit(1)
	}
}
