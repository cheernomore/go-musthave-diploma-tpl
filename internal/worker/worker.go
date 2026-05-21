package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/accrual"
)

// PendingOrder is a single order awaiting accrual processing.
type PendingOrder struct {
	Number string
	UserID uuid.UUID
}

// Store is the persistence layer required by the worker.
type Store interface {
	ClaimPending(ctx context.Context, limit int) ([]PendingOrder, error)
	ApplyAccrualResult(ctx context.Context, number string, userID uuid.UUID, status string, accrual *decimal.Decimal) error
	ResetStatus(ctx context.Context, number string) error
}

// AccrualClient is the subset of the accrual HTTP client consumed by the
// worker.
type AccrualClient interface {
	GetOrder(ctx context.Context, number string) (accrual.Result, error)
}

// Worker polls the accrual system and applies its verdicts to the storage.
type Worker struct {
	store        Store
	client       AccrualClient
	log          *slog.Logger
	workers      int
	pollInterval time.Duration
	batchSize    int

	pauseMu    sync.Mutex
	pauseUntil time.Time
}

// New returns a configured Worker. The number of goroutines and the polling
// interval must both be positive; otherwise New panics.
func New(store Store, client AccrualClient, log *slog.Logger, workers int, pollInterval time.Duration) *Worker {
	if workers <= 0 {
		panic("worker: non-positive worker count")
	}
	if pollInterval <= 0 {
		panic("worker: non-positive poll interval")
	}
	return &Worker{
		store:        store,
		client:       client,
		log:          log,
		workers:      workers,
		pollInterval: pollInterval,
		batchSize:    workers * 4,
	}
}

// Run starts the worker pool and returns when ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	jobs := make(chan PendingOrder)

	var wg sync.WaitGroup
	wg.Add(w.workers)
	for i := 0; i < w.workers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					w.process(ctx, job)
				}
			}
		}()
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil
		case <-ticker.C:
			if !w.canPoll() {
				continue
			}
			batch, err := w.store.ClaimPending(ctx, w.batchSize)
			if err != nil {
				w.log.Error("claim pending", "err", err)
				continue
			}
			for _, p := range batch {
				select {
				case <-ctx.Done():
					close(jobs)
					wg.Wait()
					return nil
				case jobs <- p:
				}
			}
		}
	}
}

func (w *Worker) process(ctx context.Context, p PendingOrder) {
	res, err := w.client.GetOrder(ctx, p.Number)
	if err != nil {
		var rl *accrual.RateLimitedError
		switch {
		case errors.As(err, &rl):
			w.pause(rl.RetryAfter)
			_ = w.store.ResetStatus(ctx, p.Number)
		case errors.Is(err, accrual.ErrNotRegistered):
			_ = w.store.ResetStatus(ctx, p.Number)
		case errors.Is(err, context.Canceled):
			return
		default:
			w.log.Warn("accrual fetch failed", "number", p.Number, "err", err)
			_ = w.store.ResetStatus(ctx, p.Number)
		}
		return
	}

	switch res.Status {
	case accrual.StatusProcessed:
		if err := w.store.ApplyAccrualResult(ctx, p.Number, p.UserID, "PROCESSED", res.Accrual); err != nil {
			w.log.Error("apply processed", "number", p.Number, "err", err)
		}
	case accrual.StatusInvalid:
		if err := w.store.ApplyAccrualResult(ctx, p.Number, p.UserID, "INVALID", nil); err != nil {
			w.log.Error("apply invalid", "number", p.Number, "err", err)
		}
	default:
		_ = w.store.ResetStatus(ctx, p.Number)
	}
}

func (w *Worker) pause(d time.Duration) {
	w.pauseMu.Lock()
	defer w.pauseMu.Unlock()
	until := time.Now().Add(d)
	if until.After(w.pauseUntil) {
		w.pauseUntil = until
		w.log.Warn("accrual rate limited", "retry_after", d.String())
	}
}

func (w *Worker) canPoll() bool {
	w.pauseMu.Lock()
	defer w.pauseMu.Unlock()
	return time.Now().After(w.pauseUntil)
}
