package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

type fakeBalance struct {
	get         domain.Balance
	withdrawErr error
	list        []domain.Withdrawal
}

func (f *fakeBalance) Get(context.Context, uuid.UUID) (domain.Balance, error) { return f.get, nil }
func (f *fakeBalance) Withdraw(context.Context, uuid.UUID, string, decimal.Decimal) error {
	return f.withdrawErr
}
func (f *fakeBalance) ListWithdrawals(context.Context, uuid.UUID) ([]domain.Withdrawal, error) {
	return f.list, nil
}

func newBalanceHandlers(s BalanceService) *BalanceHandlers {
	return NewBalanceHandlers(s, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestBalanceGet(t *testing.T) {
	h := newBalanceHandlers(&fakeBalance{get: domain.Balance{
		Current:   decimal.NewFromFloat(500.5),
		Withdrawn: decimal.NewFromInt(42),
	}})
	rr := httptest.NewRecorder()
	h.Get(rr, authedRequest(http.MethodGet, "/", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if _, ok := resp["current"]; !ok {
		t.Fatalf("missing current: %s", rr.Body.String())
	}
}

func TestWithdrawStatuses(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"ok", nil, http.StatusOK},
		{"insufficient", domain.ErrInsufficientFunds, http.StatusPaymentRequired},
		{"bad luhn", domain.ErrInvalidOrderNumber, http.StatusUnprocessableEntity},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newBalanceHandlers(&fakeBalance{withdrawErr: c.err})
			req := authedRequest(http.MethodPost, "/", `{"order":"2377225624","sum":1}`)
			rr := httptest.NewRecorder()
			h.Withdraw(rr, req)
			if rr.Code != c.want {
				t.Fatalf("status = %d, want %d", rr.Code, c.want)
			}
		})
	}
}

func TestWithdrawBadJSON(t *testing.T) {
	h := newBalanceHandlers(&fakeBalance{})
	rr := httptest.NewRecorder()
	h.Withdraw(rr, authedRequest(http.MethodPost, "/", "{"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestWithdrawalsEmpty(t *testing.T) {
	h := newBalanceHandlers(&fakeBalance{})
	rr := httptest.NewRecorder()
	h.Withdrawals(rr, authedRequest(http.MethodGet, "/", ""))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestWithdrawalsJSON(t *testing.T) {
	h := newBalanceHandlers(&fakeBalance{list: []domain.Withdrawal{
		{OrderNumber: "1", Sum: decimal.NewFromInt(50), ProcessedAt: time.Now()},
	}})
	rr := httptest.NewRecorder()
	h.Withdrawals(rr, authedRequest(http.MethodGet, "/", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"order":"1"`) {
		t.Fatalf("body: %s", rr.Body.String())
	}
}
