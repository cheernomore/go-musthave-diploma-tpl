package order

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
	"github.com/cheernomore/go-musthave-diploma-tpl/internal/luhn"
)

// Repository abstracts persistence operations for orders.
type Repository interface {
	// Insert stores a new order. It must return domain.ErrOrderAlreadyUploaded
	// when the same user re-uploads an existing number, and
	// domain.ErrOrderOwnedByAnotherUser when another user already owns it.
	Insert(ctx context.Context, o domain.Order) error
	// ListByUser returns the user's orders ordered by upload time descending.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Order, error)
}

// Service implements the order use cases.
type Service struct {
	repo Repository
	now  func() time.Time
}

// New constructs an order Service.
func New(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Upload validates the supplied number with the Luhn algorithm and records a
// new NEW-status order for the given user. It returns
// domain.ErrInvalidOrderNumber, domain.ErrOrderAlreadyUploaded or
// domain.ErrOrderOwnedByAnotherUser as appropriate.
func (s *Service) Upload(ctx context.Context, userID uuid.UUID, number string) error {
	if !luhn.Valid(number) {
		return domain.ErrInvalidOrderNumber
	}
	o := domain.Order{
		Number:     number,
		UserID:     userID,
		Status:     domain.OrderStatusNew,
		UploadedAt: s.now(),
	}
	if err := s.repo.Insert(ctx, o); err != nil {
		return err
	}
	return nil
}

// List returns the orders uploaded by the user, newest first.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]domain.Order, error) {
	orders, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	return orders, nil
}
