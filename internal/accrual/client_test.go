package accrual

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientProcessed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/orders/12345" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"order":"12345","status":"PROCESSED","accrual":500}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	res, err := c.GetOrder(context.Background(), "12345")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusProcessed {
		t.Fatalf("status = %v", res.Status)
	}
	if res.Accrual == nil || !res.Accrual.Equal(res.Accrual.Truncate(2)) {
		t.Fatalf("accrual = %v", res.Accrual)
	}
}

func TestClientNotRegistered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, time.Second)
	_, err := c.GetOrder(context.Background(), "x")
	if !errors.Is(err, ErrNotRegistered) {
		t.Fatalf("got %v", err)
	}
}

func TestClientRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, time.Second)
	_, err := c.GetOrder(context.Background(), "x")
	var rl *RateLimitedError
	if !errors.As(err, &rl) {
		t.Fatalf("got %v", err)
	}
	if rl.RetryAfter != 5*time.Second {
		t.Fatalf("retry = %v", rl.RetryAfter)
	}
}
