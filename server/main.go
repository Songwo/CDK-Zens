package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"cdk-airdrop-station/server/internal/app"
	"cdk-airdrop-station/server/internal/config"
	"cdk-airdrop-station/server/internal/store"
)

func main() {
	godotenv.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config failed", "error", err)
		os.Exit(1)
	}

	stateStore, err := store.NewWithDatabase(cfg.DataFile, cfg.PostgresDSN, cfg.MySQLDSN, cfg.RedisURL, cfg.RabbitMQURL)
	if err != nil {
		logger.Error("init store failed", "error", err)
		os.Exit(1)
	}

	server := app.New(cfg, stateStore, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("cdk airdrop server starting", "addr", cfg.Addr, "dataFile", cfg.DataFile)
		if err := server.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
			logger.Error("server exited unexpectedly", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// 关闭领取 worker
	stateStore.Shutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
