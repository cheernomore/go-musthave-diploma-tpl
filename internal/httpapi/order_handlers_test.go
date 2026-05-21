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

type fakeOrders struct {
	uploadErr error
	list      []domain.Order
	listErr   error
}

func (f *fakeOrders) Upload(context.Context, uuid.UUID, string) error { return f.uploadErr }
func (f *fakeOrders) List(context.Context, uuid.UUID) ([]domain.Order, error) {
	return f.list, f.listErr
}

func newOrderHandlers(s OrderService) *OrderHandlers {
	return NewOrderHandlers(s, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func authedRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := WithUserID(req.Context(), uuid.New())
	return req.WithContext(ctx)
}

func TestOrderUploadStatuses(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"accepted", nil, http.StatusAccepted},
		{"same user", domain.ErrOrderAlreadyUploaded, http.StatusOK},
		{"other user", domain.ErrOrderOwnedByAnotherUser, http.StatusConflict},
		{"invalid luhn", domain.ErrInvalidOrderNumber, http.StatusUnprocessableEntity},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newOrderHandlers(&fakeOrders{uploadErr: c.err})
			rr := httptest.NewRecorder()
			h.Upload(rr, authedRequest(http.MethodPost, "/", "12345678903"))
			if rr.Code != c.want {
				t.Fatalf("status = %d, want %d", rr.Code, c.want)
			}
		})
	}
}

func TestOrderUploadUnauthorized(t *testing.T) {
	h := newOrderHandlers(&fakeOrders{})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("12345678903"))
	rr := httptest.NewRecorder()
	h.Upload(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestOrderListEmpty(t *testing.T) {
	h := newOrderHandlers(&fakeOrders{})
	rr := httptest.NewRecorder()
	h.List(rr, authedRequest(http.MethodGet, "/", ""))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestOrderListJSON(t *testing.T) {
	accrual := decimal.NewFromInt(500)
	orders := []domain.Order{
		{Number: "1", Status: domain.OrderStatusProcessed, Accrual: &accrual, UploadedAt: time.Now()},
		{Number: "2", Status: domain.OrderStatusNew, UploadedAt: time.Now()},
	}
	h := newOrderHandlers(&fakeOrders{list: orders})
	rr := httptest.NewRecorder()
	h.List(rr, authedRequest(http.MethodGet, "/", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var resp []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("len = %d", len(resp))
	}
	if _, has := resp[1]["accrual"]; has {
		t.Fatalf("accrual must be omitted for NEW order")
	}
}
