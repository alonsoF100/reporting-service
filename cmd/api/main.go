package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alonsoF100/reporting-service/internal/config"
	"github.com/alonsoF100/reporting-service/internal/logger"
	"github.com/alonsoF100/reporting-service/internal/repository/postgres"
	"github.com/alonsoF100/reporting-service/internal/service"
	"github.com/alonsoF100/reporting-service/internal/transport/handler"
	"github.com/alonsoF100/reporting-service/internal/transport/server"
	_ "github.com/alonsoF100/reporting-service/migrations/postgres" // миграции
)

func main() {
	cfg := config.Load()

	log := logger.Setup(cfg)
	slog.SetDefault(log)

	slog.Info("starting reporting service",
		"version", "1.0.0",
		"input_dir", cfg.Application.Input,
		"output_dir", cfg.Application.Output,
	)

	pool, err := postgres.NewPool(cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := postgres.New(pool)
	slog.Info("database connected")

	deviceService := service.NewDeviceService(repo)

	scanner := service.NewScanner(cfg, repo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go scanner.Start(ctx)
	slog.Info("scanner started",
		"interval", cfg.Application.Period,
		"workers", cfg.Application.Workers)

	h := handler.New(deviceService)
	srv := server.New(cfg, h, log)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		slog.Info("shutting down gracefully...")
		cancel()

		ctx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := srv.Server.Shutdown(ctx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	if err := srv.Start(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
