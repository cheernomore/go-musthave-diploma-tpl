package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// PendingOrder is a minimal projection of an order that needs accrual
// processing, carrying just the data the worker requires.
type PendingOrder struct {
	Number string
	UserID uuid.UUID
}

// ClaimPending atomically marks up to limit unfinished orders as PROCESSING
// and returns them. Concurrent workers will not see overlapping batches.
func (r *OrderRepository) ClaimPending(ctx context.Context, limit int) ([]PendingOrder, error) {
	rows, err := r.pool.Query(ctx,
		`UPDATE orders
		 SET status = 'PROCESSING'
		 WHERE number IN (
		     SELECT number FROM orders
		     WHERE status IN ('NEW', 'PROCESSING')
		     ORDER BY uploaded_at
		     LIMIT $1
		     FOR UPDATE SKIP LOCKED
		 )
		 RETURNING number, user_id`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("claim pending: %w", err)
	}
	defer rows.Close()

	var out []PendingOrder
	for rows.Next() {
		var p PendingOrder
		if err := rows.Scan(&p.Number, &p.UserID); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

// ApplyAccrualResult records the outcome of an accrual lookup. If accrual is
// non-nil and status is PROCESSED, the user's balance is incremented inside
// the same transaction.
func (r *OrderRepository) ApplyAccrualResult(ctx context.Context, number string, userID uuid.UUID, status string, accrual *decimal.Decimal) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE orders SET status = $2, accrual = $3 WHERE number = $1`,
		number, status, accrual,
	); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	if accrual != nil && accrual.Sign() > 0 && status == "PROCESSED" {
		if _, err := tx.Exec(ctx,
			`UPDATE balances SET current = current + $2 WHERE user_id = $1`,
			userID, accrual,
		); err != nil {
			return fmt.Errorf("update balance: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ResetStatus rolls an order back to NEW; the worker calls it when accrual
// returned a retryable error so the order is reconsidered later.
func (r *OrderRepository) ResetStatus(ctx context.Context, number string) error {
	if _, err := r.pool.Exec(ctx,
		`UPDATE orders SET status = 'NEW' WHERE number = $1 AND status = 'PROCESSING'`,
		number,
	); err != nil {
		return fmt.Errorf("reset status: %w", err)
	}
	return nil
}
