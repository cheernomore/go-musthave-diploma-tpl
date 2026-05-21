package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/accrual"
)

type fakeStore struct {
	mu      sync.Mutex
	pending []PendingOrder
	applied []appliedCall
	resets  []string
}

type appliedCall struct {
	number  string
	status  string
	accrual *decimal.Decimal
}

func (s *fakeStore) ClaimPending(_ context.Context, _ int) ([]PendingOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.pending
	s.pending = nil
	return out, nil
}

func (s *fakeStore) ApplyAccrualResult(_ context.Context, number string, _ uuid.UUID, status string, a *decimal.Decimal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applied = append(s.applied, appliedCall{number: number, status: status, accrual: a})
	return nil
}

func (s *fakeStore) ResetStatus(_ context.Context, number string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resets = append(s.resets, number)
	return nil
}

type fakeClient struct {
	result accrual.Result
	err    error
}

func (f fakeClient) GetOrder(context.Context, string) (accrual.Result, error) {
	return f.result, f.err
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestWorkerProcessesProcessed(t *testing.T) {
	accrualValue := decimal.NewFromInt(100)
	store := &fakeStore{pending: []PendingOrder{{Number: "1", UserID: uuid.New()}}}
	client := fakeClient{result: accrual.Result{Status: accrual.StatusProcessed, Accrual: &accrualValue}}
	w := New(store, client, discardLogger(), 1, 5*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = w.Run(ctx)

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.applied) != 1 || store.applied[0].status != "PROCESSED" {
		t.Fatalf("applied = %+v", store.applied)
	}
}

func TestWorkerResetsOnNotRegistered(t *testing.T) {
	store := &fakeStore{pending: []PendingOrder{{Number: "1", UserID: uuid.New()}}}
	client := fakeClient{err: accrual.ErrNotRegistered}
	w := New(store, client, discardLogger(), 1, 5*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = w.Run(ctx)

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.resets) == 0 {
		t.Fatal("expected reset call")
	}
}

func TestWorkerPausesOnRateLimit(t *testing.T) {
	store := &fakeStore{pending: []PendingOrder{{Number: "1", UserID: uuid.New()}}}
	client := fakeClient{err: &accrual.RateLimitedError{RetryAfter: time.Hour}}
	w := New(store, client, discardLogger(), 1, 5*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	_ = w.Run(ctx)

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.applied) != 0 {
		t.Fatal("must not apply when rate limited")
	}
	if !errors.Is(client.err, client.err) {
		t.Fatal("sanity")
	}
}

func TestWorkerInvalidStatus(t *testing.T) {
	store := &fakeStore{pending: []PendingOrder{{Number: "1", UserID: uuid.New()}}}
	client := fakeClient{result: accrual.Result{Status: accrual.StatusInvalid}}
	w := New(store, client, discardLogger(), 1, 5*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	_ = w.Run(ctx)

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.applied) != 1 || store.applied[0].status != "INVALID" {
		t.Fatalf("applied = %+v", store.applied)
	}
}
