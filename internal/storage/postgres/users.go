package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

// UserRepository persists users in PostgreSQL.
type UserRepository struct {
	pool Pool
}

// NewUserRepository builds a UserRepository backed by the given pool.
func NewUserRepository(pool Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user and creates an empty balance row in the same
// transaction. It returns domain.ErrLoginTaken on a unique-violation conflict
// over the login column.
func (r *UserRepository) Create(ctx context.Context, u domain.User) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx,
		`INSERT INTO users (id, login, password_hash, created_at) VALUES ($1, $2, $3, $4)`,
		u.ID, u.Login, u.PasswordHash, u.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return domain.ErrLoginTaken
		}
		return fmt.Errorf("insert user: %w", err)
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO balances (user_id, current, withdrawn) VALUES ($1, 0, 0)`,
		u.ID,
	)
	if err != nil {
		return fmt.Errorf("insert balance: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// FindByLogin loads a user record by login, returning domain.ErrUserNotFound
// if no matching row exists.
func (r *UserRepository) FindByLogin(ctx context.Context, login string) (domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, login, password_hash, created_at FROM users WHERE login = $1`,
		login,
	).Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("query user: %w", err)
	}
	return u, nil
}
