package order

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

type fakeRepo struct {
	inserted  []domain.Order
	insertErr error
	list      []domain.Order
}

func (f *fakeRepo) Insert(_ context.Context, o domain.Order) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	f.inserted = append(f.inserted, o)
	return nil
}

func (f *fakeRepo) ListByUser(context.Context, uuid.UUID) ([]domain.Order, error) {
	return f.list, nil
}

func TestUploadInvalidLuhn(t *testing.T) {
	svc := New(&fakeRepo{})
	err := svc.Upload(context.Background(), uuid.New(), "12345")
	if !errors.Is(err, domain.ErrInvalidOrderNumber) {
		t.Fatalf("want ErrInvalidOrderNumber, got %v", err)
	}
}

func TestUploadOK(t *testing.T) {
	repo := &fakeRepo{}
	svc := New(repo)
	if err := svc.Upload(context.Background(), uuid.New(), "12345678903"); err != nil {
		t.Fatalf("upload: %v", err)
	}
	if len(repo.inserted) != 1 {
		t.Fatalf("inserted = %d", len(repo.inserted))
	}
	if repo.inserted[0].Status != domain.OrderStatusNew {
		t.Fatalf("status = %v", repo.inserted[0].Status)
	}
}

func TestList(t *testing.T) {
	repo := &fakeRepo{list: []domain.Order{{Number: "1"}, {Number: "2"}}}
	svc := New(repo)
	got, err := svc.List(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
}

func TestUploadPropagatesRepoError(t *testing.T) {
	repo := &fakeRepo{insertErr: domain.ErrOrderOwnedByAnotherUser}
	svc := New(repo)
	err := svc.Upload(context.Background(), uuid.New(), "12345678903")
	if !errors.Is(err, domain.ErrOrderOwnedByAnotherUser) {
		t.Fatalf("got %v", err)
	}
}
