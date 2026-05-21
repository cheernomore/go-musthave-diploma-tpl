package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

// BalanceRepository persists balances and withdrawals in PostgreSQL.
type BalanceRepository struct {
	pool Pool
}

// NewBalanceRepository builds a BalanceRepository backed by the given pool.
func NewBalanceRepository(pool Pool) *BalanceRepository {
	return &BalanceRepository{pool: pool}
}

// Get returns the balance of the user, or a zero balance when the row is
// missing (which can only happen for unknown users).
func (r *BalanceRepository) Get(ctx context.Context, userID uuid.UUID) (domain.Balance, error) {
	var b domain.Balance
	err := r.pool.QueryRow(ctx,
		`SELECT current, withdrawn FROM balances WHERE user_id = $1`, userID,
	).Scan(&b.Current, &b.Withdrawn)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Balance{}, nil
		}
		return domain.Balance{}, fmt.Errorf("query balance: %w", err)
	}
	return b, nil
}

// Withdraw performs the balance update and withdrawal insert in a single
// transaction. It uses SELECT ... FOR UPDATE to guard against concurrent
// withdrawals draining the balance.
func (r *BalanceRepository) Withdraw(ctx context.Context, w domain.Withdrawal) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current decimal.Decimal
	err = tx.QueryRow(ctx,
		`SELECT current FROM balances WHERE user_id = $1 FOR UPDATE`,
		w.UserID,
	).Scan(&current)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrInsufficientFunds
		}
		return fmt.Errorf("lock balance: %w", err)
	}
	if current.LessThan(w.Sum) {
		return domain.ErrInsufficientFunds
	}

	_, err = tx.Exec(ctx,
		`UPDATE balances SET current = current - $2, withdrawn = withdrawn + $2 WHERE user_id = $1`,
		w.UserID, w.Sum,
	)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO withdrawals (id, user_id, order_number, sum, processed_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		w.ID, w.UserID, w.OrderNumber, w.Sum, w.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("insert withdrawal: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ListWithdrawals returns the user's withdrawals ordered by processed_at
// descending.
func (r *BalanceRepository) ListWithdrawals(ctx context.Context, userID uuid.UUID) ([]domain.Withdrawal, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, order_number, sum, processed_at
		 FROM withdrawals WHERE user_id = $1
		 ORDER BY processed_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query withdrawals: %w", err)
	}
	defer rows.Close()

	var out []domain.Withdrawal
	for rows.Next() {
		var w domain.Withdrawal
		if err := rows.Scan(&w.ID, &w.UserID, &w.OrderNumber, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}
