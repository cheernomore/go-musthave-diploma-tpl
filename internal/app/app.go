package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/accrual"
	"github.com/cheernomore/go-musthave-diploma-tpl/internal/config"
	"github.com/cheernomore/go-musthave-diploma-tpl/internal/httpapi"
	"github.com/cheernomore/go-musthave-diploma-tpl/internal/service/auth"
	"github.com/cheernomore/go-musthave-diploma-tpl/internal/service/balance"
	"github.com/cheernomore/go-musthave-diploma-tpl/internal/service/order"
	"github.com/cheernomore/go-musthave-diploma-tpl/internal/storage/postgres"
	"github.com/cheernomore/go-musthave-diploma-tpl/internal/worker"
)

// Run starts the gophermart service with the supplied configuration and
// logger. It returns when ctx is cancelled or any component fails. The first
// non-nil error from a component is returned.
func Run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	db, err := postgres.New(ctx, cfg.DatabaseURI)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()
	log.Info("database connected and migrated")

	orderRepo := db.Orders()

	authSvc := auth.New(db.Users(), cfg.JWTSecret, cfg.JWTTTL)
	orderSvc := order.New(orderRepo)
	balanceSvc := balance.New(db.Balances())

	router := httpapi.NewRouter(httpapi.Deps{
		Logger:   log,
		Auth:     httpapi.NewAuthHandlers(authSvc, log),
		Orders:   httpapi.NewOrderHandlers(orderSvc, log),
		Balance:  httpapi.NewBalanceHandlers(balanceSvc, log),
		Verifier: authSvc,
	})

	srv := &http.Server{
		Addr:              cfg.RunAddress,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	accrualClient := accrual.NewClient(cfg.AccrualSystemAddress, 5*time.Second)
	wrk := worker.New(orderRepo, accrualClient, log, cfg.AccrualWorkers, cfg.AccrualPollInterval)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Info("accrual worker starting", "workers", cfg.AccrualWorkers)
		return wrk.Run(gctx)
	})

	g.Go(func() error {
		log.Info("http server starting", "addr", cfg.RunAddress)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		<-gctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		log.Info("http server shutting down")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}
