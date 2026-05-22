package balance

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

type fakeRepo struct {
	balance         domain.Balance
	withdrawErr     error
	withdrawn       []domain.Withdrawal
	listWithdrawals []domain.Withdrawal
}

func (f *fakeRepo) Get(context.Context, uuid.UUID) (domain.Balance, error) {
	return f.balance, nil
}

func (f *fakeRepo) Withdraw(_ context.Context, w domain.Withdrawal) error {
	if f.withdrawErr != nil {
		return f.withdrawErr
	}
	f.withdrawn = append(f.withdrawn, w)
	return nil
}

func (f *fakeRepo) ListWithdrawals(context.Context, uuid.UUID) ([]domain.Withdrawal, error) {
	return f.listWithdrawals, nil
}

func TestWithdrawInvalidLuhn(t *testing.T) {
	svc := New(&fakeRepo{})
	err := svc.Withdraw(context.Background(), uuid.New(), "123", decimal.NewFromInt(10))
	if !errors.Is(err, domain.ErrInvalidOrderNumber) {
		t.Fatalf("got %v", err)
	}
}

func TestWithdrawZeroSum(t *testing.T) {
	svc := New(&fakeRepo{})
	err := svc.Withdraw(context.Background(), uuid.New(), "12345678903", decimal.Zero)
	if !errors.Is(err, domain.ErrInvalidWithdrawalSum) {
		t.Fatalf("want ErrInvalidWithdrawalSum, got %v", err)
	}
}

func TestWithdrawNegativeSum(t *testing.T) {
	svc := New(&fakeRepo{})
	err := svc.Withdraw(context.Background(), uuid.New(), "12345678903", decimal.NewFromInt(-1))
	if !errors.Is(err, domain.ErrInvalidWithdrawalSum) {
		t.Fatalf("want ErrInvalidWithdrawalSum, got %v", err)
	}
}

func TestWithdrawInsufficient(t *testing.T) {
	svc := New(&fakeRepo{withdrawErr: domain.ErrInsufficientFunds})
	err := svc.Withdraw(context.Background(), uuid.New(), "12345678903", decimal.NewFromInt(10))
	if !errors.Is(err, domain.ErrInsufficientFunds) {
		t.Fatalf("got %v", err)
	}
}

func TestGet(t *testing.T) {
	svc := New(&fakeRepo{balance: domain.Balance{Current: decimal.NewFromInt(100)}})
	b, err := svc.Get(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if !b.Current.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("current = %v", b.Current)
	}
}

func TestListWithdrawals(t *testing.T) {
	svc := New(&fakeRepo{listWithdrawals: []domain.Withdrawal{{OrderNumber: "x"}}})
	got, err := svc.ListWithdrawals(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
}

func TestWithdrawOK(t *testing.T) {
	repo := &fakeRepo{}
	svc := New(repo)
	if err := svc.Withdraw(context.Background(), uuid.New(), "12345678903", decimal.NewFromInt(50)); err != nil {
		t.Fatal(err)
	}
	if len(repo.withdrawn) != 1 {
		t.Fatalf("withdrawn = %d", len(repo.withdrawn))
	}
}
