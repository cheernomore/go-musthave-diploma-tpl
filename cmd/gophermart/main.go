// Command gophermart is the entry point of the loyalty service.
// It loads configuration from flags and environment variables and runs the
// HTTP server until a termination signal is received.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/shopspring/decimal"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/app"
	"github.com/cheernomore/go-musthave-diploma-tpl/internal/config"
	"github.com/cheernomore/go-musthave-diploma-tpl/internal/logger"
)

func init() {
	decimal.MarshalJSONWithoutQuotes = true
}

func main() {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(2)
	}

	log := logger.New(os.Getenv("LOG_LEVEL"))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, log); err != nil {
		log.Error("service stopped with error", "err", err)
		os.Exit(1)
	}
	log.Info("service stopped")
}
