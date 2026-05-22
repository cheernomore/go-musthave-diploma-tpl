package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

// BalanceService is the subset of the balance service consumed by handlers.
type BalanceService interface {
	Get(ctx context.Context, userID uuid.UUID) (domain.Balance, error)
	Withdraw(ctx context.Context, userID uuid.UUID, order string, sum decimal.Decimal) error
	ListWithdrawals(ctx context.Context, userID uuid.UUID) ([]domain.Withdrawal, error)
}

// BalanceHandlers groups the balance and withdrawals HTTP handlers.
type BalanceHandlers struct {
	svc BalanceService
	log *slog.Logger
}

// NewBalanceHandlers builds BalanceHandlers backed by svc.
func NewBalanceHandlers(svc BalanceService, log *slog.Logger) *BalanceHandlers {
	return &BalanceHandlers{svc: svc, log: log}
}

type balanceResponse struct {
	Current   JSONDecimal `json:"current"`
	Withdrawn JSONDecimal `json:"withdrawn"`
}

// Get handles GET /api/user/balance.
func (h *BalanceHandlers) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	b, err := h.svc.Get(r.Context(), userID)
	if err != nil {
		h.log.Error("get balance", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(balanceResponse{
		Current:   JSONDecimal(b.Current),
		Withdrawn: JSONDecimal(b.Withdrawn),
	})
}

type withdrawRequest struct {
	Order string          `json:"order"`
	Sum   decimal.Decimal `json:"sum"`
}

// Withdraw handles POST /api/user/balance/withdraw.
func (h *BalanceHandlers) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req withdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	switch err := h.svc.Withdraw(r.Context(), userID, req.Order, req.Sum); {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, domain.ErrInvalidOrderNumber):
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
	case errors.Is(err, domain.ErrInvalidWithdrawalSum):
		http.Error(w, "invalid withdrawal sum", http.StatusBadRequest)
	case errors.Is(err, domain.ErrInsufficientFunds):
		http.Error(w, "insufficient funds", http.StatusPaymentRequired)
	default:
		h.log.Error("withdraw", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

type withdrawalResponse struct {
	Order       string      `json:"order"`
	Sum         JSONDecimal `json:"sum"`
	ProcessedAt string      `json:"processed_at"`
}

// Withdrawals handles GET /api/user/withdrawals.
func (h *BalanceHandlers) Withdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	list, err := h.svc.ListWithdrawals(r.Context(), userID)
	if err != nil {
		h.log.Error("list withdrawals", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if len(list) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	resp := make([]withdrawalResponse, 0, len(list))
	for _, it := range list {
		resp = append(resp, withdrawalResponse{
			Order:       it.OrderNumber,
			Sum:         JSONDecimal(it.Sum),
			ProcessedAt: it.ProcessedAt.Format(time.RFC3339),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
