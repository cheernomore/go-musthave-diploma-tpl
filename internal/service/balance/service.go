package balance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
	"github.com/cheernomore/go-musthave-diploma-tpl/internal/luhn"
)

// Repository abstracts persistence operations for balances and withdrawals.
type Repository interface {
	// Get returns the current balance of the user.
	Get(ctx context.Context, userID uuid.UUID) (domain.Balance, error)
	// Withdraw deducts sum from the user's current balance, increments the
	// withdrawn total and records a withdrawal. It must return
	// domain.ErrInsufficientFunds when current < sum.
	Withdraw(ctx context.Context, w domain.Withdrawal) error
	// ListWithdrawals returns the user's withdrawals ordered by processed_at
	// descending.
	ListWithdrawals(ctx context.Context, userID uuid.UUID) ([]domain.Withdrawal, error)
}

// Service implements the balance use cases.
type Service struct {
	repo Repository
	now  func() time.Time
}

// New constructs a balance Service.
func New(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Get returns the user's current balance.
func (s *Service) Get(ctx context.Context, userID uuid.UUID) (domain.Balance, error) {
	b, err := s.repo.Get(ctx, userID)
	if err != nil {
		return domain.Balance{}, fmt.Errorf("get balance: %w", err)
	}
	return b, nil
}

// Withdraw validates the order number, checks the balance and records a
// withdrawal atomically.
func (s *Service) Withdraw(ctx context.Context, userID uuid.UUID, order string, sum decimal.Decimal) error {
	if !luhn.Valid(order) {
		return domain.ErrInvalidOrderNumber
	}
	if sum.Sign() <= 0 {
		return fmt.Errorf("sum must be positive")
	}
	w := domain.Withdrawal{
		ID:          uuid.New(),
		UserID:      userID,
		OrderNumber: order,
		Sum:         sum,
		ProcessedAt: s.now(),
	}
	return s.repo.Withdraw(ctx, w)
}

// ListWithdrawals returns user's withdrawals, newest first.
func (s *Service) ListWithdrawals(ctx context.Context, userID uuid.UUID) ([]domain.Withdrawal, error) {
	ws, err := s.repo.ListWithdrawals(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list withdrawals: %w", err)
	}
	return ws, nil
}
