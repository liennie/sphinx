package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sphinx/internal/ctxlog"
	"sphinx/internal/rec"
	"sphinx/internal/server"
	"syscall"
)

func run(ctx context.Context, config string) (err error) {
	defer rec.Error(&err)

	logger := ctxlog.Get(ctx)

	c, err := LoadConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	logger.Info("starting server")
	srv := server.New(c.Server)

	return srv.Run(ctx)
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ctx = ctxlog.Setup(ctx)

	logger := ctxlog.Get(ctx)

	config := "config.yaml"
	if len(os.Args) > 1 {
		config = os.Args[1]
	}

	err := run(ctx, config)
	if err != nil {
		logger.Error("server stopped unexpectedly", "error", err)
	} else {
		logger.Info("server gracefully stopped")
	}
}
