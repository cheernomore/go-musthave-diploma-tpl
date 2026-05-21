package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

// AuthService is the subset of the auth service consumed by HTTP handlers.
type AuthService interface {
	Register(ctx context.Context, login, password string) (string, error)
	Login(ctx context.Context, login, password string) (string, error)
}

// AuthHandlers groups the register/login HTTP handlers.
type AuthHandlers struct {
	svc AuthService
	log *slog.Logger
}

// NewAuthHandlers builds AuthHandlers backed by svc.
func NewAuthHandlers(svc AuthService, log *slog.Logger) *AuthHandlers {
	return &AuthHandlers{svc: svc, log: log}
}

type credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Register handles POST /api/user/register.
func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	c, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	token, err := h.svc.Register(r.Context(), c.Login, c.Password)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrLoginTaken):
			http.Error(w, "login already taken", http.StatusConflict)
		default:
			h.log.Error("register failed", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}
	writeToken(w, token)
}

// Login handles POST /api/user/login.
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	c, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	token, err := h.svc.Login(r.Context(), c.Login, c.Password)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
		default:
			h.log.Error("login failed", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}
	writeToken(w, token)
}

func decodeCredentials(w http.ResponseWriter, r *http.Request) (credentials, bool) {
	var c credentials
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return credentials{}, false
	}
	if c.Login == "" || c.Password == "" {
		http.Error(w, "login and password required", http.StatusBadRequest)
		return credentials{}, false
	}
	return c, true
}

func writeToken(w http.ResponseWriter, token string) {
	w.Header().Set("Authorization", "Bearer "+token)
	http.SetCookie(w, &http.Cookie{
		Name:     "Authorization",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusOK)
}
