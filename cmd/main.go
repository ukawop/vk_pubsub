package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"subpub/internal/app"
	"subpub/internal/config"
	"subpub/internal/services/eventbus"
	"subpub/pkg/subpub"
	"syscall"
)

const (
	Dev  = "dev"
	Prod = "prod"
)

func main() {
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	sub := subpub.NewSubPub()
	subService, err := eventbus.NewEventBus(log, sub)
	if err != nil {
		log.Error("sub service failed", "error", err)
	}
	application := app.New(ctx, log, cfg.GRPC.Port, subService)

	go func() {
		log.Info("Starting gRPC server", "port", cfg.GRPC.Port)
		if err := application.Run(); err != nil {
			log.Error("gRPC server failed", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer shutdownCancel()

	application.Stop()

	if err := sub.Close(shutdownCtx); err != nil {
		log.Error("Failed to close subscription service", "error", err)
	}
}

func setupLogger(env string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	if env == Prod {
		logDir := "logs"
		if err := os.MkdirAll(logDir, 0755); err != nil {
			slog.Error("Failed to create log directory", "error", err)
			os.Exit(1)
		}

		logFile, err := os.OpenFile(filepath.Join(logDir, "app.log"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			slog.Error("Failed to open log file", "error", err)
			os.Exit(1)
		}
		opts.Level = slog.LevelInfo
		return slog.New(slog.NewJSONHandler(logFile, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
