package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

// OrderRepository persists orders in PostgreSQL.
type OrderRepository struct {
	pool Pool
}

// NewOrderRepository builds an OrderRepository backed by the given pool.
func NewOrderRepository(pool Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

// Insert attempts to record a new order. If the number already exists, it
// returns domain.ErrOrderAlreadyUploaded when the owner is the same user, and
// domain.ErrOrderOwnedByAnotherUser otherwise.
func (r *OrderRepository) Insert(ctx context.Context, o domain.Order) error {
	tag, err := r.pool.Exec(ctx,
		`INSERT INTO orders (number, user_id, status, uploaded_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (number) DO NOTHING`,
		o.Number, o.UserID, string(o.Status), o.UploadedAt,
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var owner uuid.UUID
	if err := r.pool.QueryRow(ctx,
		`SELECT user_id FROM orders WHERE number = $1`, o.Number,
	).Scan(&owner); err != nil {
		return fmt.Errorf("lookup order owner: %w", err)
	}
	if owner == o.UserID {
		return domain.ErrOrderAlreadyUploaded
	}
	return domain.ErrOrderOwnedByAnotherUser
}

// ListByUser returns the orders for a user ordered by uploaded_at descending.
func (r *OrderRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Order, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT number, user_id, status, accrual, uploaded_at
		 FROM orders
		 WHERE user_id = $1
		 ORDER BY uploaded_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var out []domain.Order
	for rows.Next() {
		var (
			o      domain.Order
			status string
		)
		if err := rows.Scan(&o.Number, &o.UserID, &status, &o.Accrual, &o.UploadedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		o.Status = domain.OrderStatus(status)
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}
