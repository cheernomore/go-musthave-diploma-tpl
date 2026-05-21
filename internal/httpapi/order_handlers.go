package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

// OrderService is the subset of the order service consumed by HTTP handlers.
type OrderService interface {
	Upload(ctx context.Context, userID uuid.UUID, number string) error
	List(ctx context.Context, userID uuid.UUID) ([]domain.Order, error)
}

// OrderHandlers groups the orders HTTP handlers.
type OrderHandlers struct {
	svc OrderService
	log *slog.Logger
}

// NewOrderHandlers builds OrderHandlers backed by svc.
func NewOrderHandlers(svc OrderService, log *slog.Logger) *OrderHandlers {
	return &OrderHandlers{svc: svc, log: log}
}

// Upload handles POST /api/user/orders. The request body must be a plain-text
// decimal order number.
func (h *OrderHandlers) Upload(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	number := strings.TrimSpace(string(body))
	if number == "" {
		http.Error(w, "empty order number", http.StatusBadRequest)
		return
	}
	switch err := h.svc.Upload(r.Context(), userID, number); {
	case err == nil:
		w.WriteHeader(http.StatusAccepted)
	case errors.Is(err, domain.ErrInvalidOrderNumber):
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
	case errors.Is(err, domain.ErrOrderAlreadyUploaded):
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, domain.ErrOrderOwnedByAnotherUser):
		http.Error(w, "order owned by another user", http.StatusConflict)
	default:
		h.log.Error("upload order", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

type orderResponse struct {
	Number     string             `json:"number"`
	Status     domain.OrderStatus `json:"status"`
	Accrual    *decimal.Decimal   `json:"accrual,omitempty"`
	UploadedAt string             `json:"uploaded_at"`
}

// List handles GET /api/user/orders.
func (h *OrderHandlers) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	orders, err := h.svc.List(r.Context(), userID)
	if err != nil {
		h.log.Error("list orders", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	resp := make([]orderResponse, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, orderResponse{
			Number:     o.Number,
			Status:     o.Status,
			Accrual:    o.Accrual,
			UploadedAt: o.UploadedAt.Format(time.RFC3339),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.log.Error("encode orders", "err", err)
	}
}
