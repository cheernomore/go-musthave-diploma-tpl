package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/shopspring/decimal"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

func TestBalanceGet(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	uid := uuid.New()
	m.ExpectQuery("SELECT current, withdrawn").WithArgs(uid).
		WillReturnRows(pgxmock.NewRows([]string{"current", "withdrawn"}).
			AddRow(decimal.NewFromInt(100), decimal.NewFromInt(50)))

	repo := NewBalanceRepository(m)
	b, err := repo.Get(context.Background(), uid)
	if err != nil {
		t.Fatal(err)
	}
	if !b.Current.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("current = %v", b.Current)
	}
}

func TestBalanceGetNoRows(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	uid := uuid.New()
	m.ExpectQuery("SELECT current, withdrawn").WithArgs(uid).
		WillReturnRows(pgxmock.NewRows([]string{"current", "withdrawn"}))

	repo := NewBalanceRepository(m)
	b, err := repo.Get(context.Background(), uid)
	if err != nil {
		t.Fatal(err)
	}
	if !b.Current.IsZero() {
		t.Fatalf("expected zero balance")
	}
}

func TestBalanceWithdrawOK(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	w := domain.Withdrawal{
		ID: uuid.New(), UserID: uuid.New(), OrderNumber: "1",
		Sum: decimal.NewFromInt(40), ProcessedAt: time.Now(),
	}
	m.ExpectBeginTx(pgx.TxOptions{})
	m.ExpectQuery("SELECT current FROM balances").WithArgs(w.UserID).
		WillReturnRows(pgxmock.NewRows([]string{"current"}).AddRow(decimal.NewFromInt(100)))
	m.ExpectExec("UPDATE balances SET current").WithArgs(w.UserID, w.Sum).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	m.ExpectExec("INSERT INTO withdrawals").
		WithArgs(w.ID, w.UserID, w.OrderNumber, w.Sum, w.ProcessedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	m.ExpectCommit()

	repo := NewBalanceRepository(m)
	if err := repo.Withdraw(context.Background(), w); err != nil {
		t.Fatal(err)
	}
}

func TestBalanceWithdrawInsufficient(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	w := domain.Withdrawal{UserID: uuid.New(), Sum: decimal.NewFromInt(200)}
	m.ExpectBeginTx(pgx.TxOptions{})
	m.ExpectQuery("SELECT current FROM balances").WithArgs(w.UserID).
		WillReturnRows(pgxmock.NewRows([]string{"current"}).AddRow(decimal.NewFromInt(100)))
	m.ExpectRollback()

	repo := NewBalanceRepository(m)
	err := repo.Withdraw(context.Background(), w)
	if !errors.Is(err, domain.ErrInsufficientFunds) {
		t.Fatalf("got %v", err)
	}
}

func TestBalanceListWithdrawals(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	uid := uuid.New()
	m.ExpectQuery("SELECT id, user_id, order_number").WithArgs(uid).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "order_number", "sum", "processed_at"}).
			AddRow(uuid.New(), uid, "1", decimal.NewFromInt(10), time.Now()))

	repo := NewBalanceRepository(m)
	got, err := repo.ListWithdrawals(context.Background(), uid)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
}
