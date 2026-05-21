package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/storage/postgres"
	"github.com/cheernomore/go-musthave-diploma-tpl/internal/worker"
)

// workerStore adapts *postgres.OrderRepository to the worker.Store interface,
// translating the storage-layer PendingOrder to the worker-layer one.
type workerStore struct {
	repo *postgres.OrderRepository
}

func (s workerStore) ClaimPending(ctx context.Context, limit int) ([]worker.PendingOrder, error) {
	got, err := s.repo.ClaimPending(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]worker.PendingOrder, len(got))
	for i, p := range got {
		out[i] = worker.PendingOrder{Number: p.Number, UserID: p.UserID}
	}
	return out, nil
}

func (s workerStore) ApplyAccrualResult(ctx context.Context, number string, userID uuid.UUID, status string, accrual *decimal.Decimal) error {
	return s.repo.ApplyAccrualResult(ctx, number, userID, status, accrual)
}

func (s workerStore) ResetStatus(ctx context.Context, number string) error {
	return s.repo.ResetStatus(ctx, number)
}
