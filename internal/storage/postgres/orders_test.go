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

func TestOrderInsertNew(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	o := domain.Order{Number: "12345678903", UserID: uuid.New(), Status: domain.OrderStatusNew, UploadedAt: time.Now()}
	m.ExpectExec("INSERT INTO orders").
		WithArgs(o.Number, o.UserID, string(o.Status), o.UploadedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewOrderRepository(m)
	if err := repo.Insert(context.Background(), o); err != nil {
		t.Fatal(err)
	}
}

func TestOrderInsertSameUser(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	userID := uuid.New()
	o := domain.Order{Number: "1", UserID: userID, Status: domain.OrderStatusNew, UploadedAt: time.Now()}
	m.ExpectExec("INSERT INTO orders").
		WithArgs(o.Number, o.UserID, string(o.Status), o.UploadedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 0))
	m.ExpectQuery("SELECT user_id FROM orders").WithArgs(o.Number).
		WillReturnRows(pgxmock.NewRows([]string{"user_id"}).AddRow(userID))

	repo := NewOrderRepository(m)
	err := repo.Insert(context.Background(), o)
	if !errors.Is(err, domain.ErrOrderAlreadyUploaded) {
		t.Fatalf("got %v", err)
	}
}

func TestOrderInsertOtherUser(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	o := domain.Order{Number: "1", UserID: uuid.New(), Status: domain.OrderStatusNew, UploadedAt: time.Now()}
	other := uuid.New()
	m.ExpectExec("INSERT INTO orders").
		WithArgs(o.Number, o.UserID, string(o.Status), o.UploadedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 0))
	m.ExpectQuery("SELECT user_id FROM orders").WithArgs(o.Number).
		WillReturnRows(pgxmock.NewRows([]string{"user_id"}).AddRow(other))

	repo := NewOrderRepository(m)
	err := repo.Insert(context.Background(), o)
	if !errors.Is(err, domain.ErrOrderOwnedByAnotherUser) {
		t.Fatalf("got %v", err)
	}
}

func TestOrderListByUser(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	userID := uuid.New()
	accrual := decimal.NewFromInt(50)
	now := time.Now()
	m.ExpectQuery("SELECT number, user_id, status, accrual, uploaded_at").
		WithArgs(userID).
		WillReturnRows(pgxmock.NewRows([]string{"number", "user_id", "status", "accrual", "uploaded_at"}).
			AddRow("1", userID, "PROCESSED", &accrual, now).
			AddRow("2", userID, "NEW", (*decimal.Decimal)(nil), now))

	repo := NewOrderRepository(m)
	got, err := repo.ListByUser(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
}

func TestOrderResetStatus(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	m.ExpectExec("UPDATE orders SET status = 'NEW'").WithArgs("1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewOrderRepository(m)
	if err := repo.ResetStatus(context.Background(), "1"); err != nil {
		t.Fatal(err)
	}
}

func TestOrderClaimPending(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	uid := uuid.New()
	m.ExpectQuery("UPDATE orders").WithArgs(10).
		WillReturnRows(pgxmock.NewRows([]string{"number", "user_id"}).AddRow("1", uid))

	repo := NewOrderRepository(m)
	got, err := repo.ClaimPending(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].UserID != uid {
		t.Fatalf("got %+v", got)
	}
}

func TestOrderApplyAccrualProcessed(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	uid := uuid.New()
	accrual := decimal.NewFromInt(100)
	m.ExpectBeginTx(pgx.TxOptions{})
	m.ExpectExec("UPDATE orders SET status").WithArgs("1", "PROCESSED", &accrual).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	m.ExpectExec("UPDATE balances SET current").WithArgs(uid, &accrual).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	m.ExpectCommit()

	repo := NewOrderRepository(m)
	if err := repo.ApplyAccrualResult(context.Background(), "1", uid, "PROCESSED", &accrual); err != nil {
		t.Fatal(err)
	}
}

func TestOrderApplyAccrualInvalid(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	uid := uuid.New()
	m.ExpectBeginTx(pgx.TxOptions{})
	m.ExpectExec("UPDATE orders SET status").WithArgs("1", "INVALID", (*decimal.Decimal)(nil)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	m.ExpectCommit()

	repo := NewOrderRepository(m)
	if err := repo.ApplyAccrualResult(context.Background(), "1", uid, "INVALID", nil); err != nil {
		t.Fatal(err)
	}
}
