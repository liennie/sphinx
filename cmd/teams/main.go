package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sphinx/internal/ctxlog"
	"sphinx/internal/db"
	"sphinx/internal/rec"
	"syscall"
)

func run(ctx context.Context, config string) (err error) {
	defer rec.Error(&err)

	logger := ctxlog.Get(ctx)

	c, err := LoadConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	logger.Info("opening db")
	db.Open(c.DB)
	defer ctxlog.Close(ctx, "db", db.Closer())

	for team, progress := range db.AllTeams() {
		// TODO sort by puzzle order
		// TODO combine all and sort by time
		logger.Info("team progress", "team", team, "progress", progress)
	}

	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ctx = ctxlog.Setup(ctx, "teams")

	logger := ctxlog.Get(ctx)

	config := "config.yaml"
	if len(os.Args) > 1 {
		config = os.Args[1]
	}

	err := run(ctx, config)
	if err != nil {
		logger.Error("stopped unexpectedly", "error", err)
	}
}
