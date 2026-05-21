package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

type fakeAuth struct {
	registerToken string
	registerErr   error
	loginToken    string
	loginErr      error
}

func (f *fakeAuth) Register(context.Context, string, string) (string, error) {
	return f.registerToken, f.registerErr
}
func (f *fakeAuth) Login(context.Context, string, string) (string, error) {
	return f.loginToken, f.loginErr
}

func newHandlers(a AuthService) *AuthHandlers {
	return NewAuthHandlers(a, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestRegisterOK(t *testing.T) {
	h := newHandlers(&fakeAuth{registerToken: "tok"})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"login":"a","password":"b"}`))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.HasPrefix(rr.Header().Get("Authorization"), "Bearer ") {
		t.Fatalf("Authorization header = %q", rr.Header().Get("Authorization"))
	}
}

func TestRegisterBadJSON(t *testing.T) {
	h := newHandlers(&fakeAuth{})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("{")))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestRegisterConflict(t *testing.T) {
	h := newHandlers(&fakeAuth{registerErr: domain.ErrLoginTaken})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"login":"a","password":"b"}`))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestLoginUnauthorized(t *testing.T) {
	h := newHandlers(&fakeAuth{loginErr: domain.ErrInvalidCredentials})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"login":"a","password":"b"}`))
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestLoginInternal(t *testing.T) {
	h := newHandlers(&fakeAuth{loginErr: errors.New("boom")})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"login":"a","password":"b"}`))
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rr.Code)
	}
}
