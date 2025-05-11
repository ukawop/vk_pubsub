package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"subpub/internal/subpubserver"
	"subpub/pkg/subpub"
	"sync"

	"google.golang.org/grpc"
)

type App struct {
	log         *slog.Logger
	gRPCServer  *grpc.Server
	port        int
	shutdownCtx context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func New(ctx context.Context, log *slog.Logger, port int, pubsub subpub.SubPub) *App {
	ctx, cancel := context.WithCancel(context.Background())

	gRPCServer := grpc.NewServer()
	subpubserver.Register(ctx, gRPCServer, pubsub)

	return &App{
		log:         log,
		gRPCServer:  gRPCServer,
		port:        port,
		shutdownCtx: ctx,
		cancel:      cancel,
	}
}

func (a *App) Run() error {
	const op = "app.Run"
	log := a.log.With(slog.String("op", op))

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()

		log.Info("starting gRPC server", slog.Int("port", a.port))
		if err := a.gRPCServer.Serve(l); err != nil {
			log.Error("gRPC server failed", slog.String("error", err.Error()))
			a.cancel()
		}
	}()

	return nil
}

func (a *App) Stop() {
	const op = "app.Stop"

	a.log.With(slog.String("op", op)).Info("stopping gRPC server")

	a.gRPCServer.GracefulStop()
}

func (a *App) Wait() {
	a.wg.Wait()
}
