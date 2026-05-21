package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Status mirrors the accrual system order statuses. Values are kept as the
// raw strings used by the API.
type Status string

// Statuses reported by the accrual system.
const (
	// StatusRegistered means the order is known but not yet processed.
	StatusRegistered Status = "REGISTERED"
	// StatusInvalid is a terminal failure state.
	StatusInvalid Status = "INVALID"
	// StatusProcessing means the calculation is in progress.
	StatusProcessing Status = "PROCESSING"
	// StatusProcessed is the terminal success state.
	StatusProcessed Status = "PROCESSED"
)

// Result is the parsed response from the accrual service.
// Accrual is nil when the API omits the field.
type Result struct {
	Order   string
	Status  Status
	Accrual *decimal.Decimal
}

// ErrNotRegistered is returned when the accrual system replies with 204.
var ErrNotRegistered = errors.New("order not registered in accrual system")

// RateLimitedError is returned when the accrual system replies with 429.
// RetryAfter carries the cool-down hint announced by the server.
type RateLimitedError struct {
	RetryAfter time.Duration
}

// Error implements the error interface.
func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("accrual: rate limited, retry after %s", e.RetryAfter)
}

// Client talks to the external accrual HTTP API.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient builds an accrual Client targeting baseURL. The base URL must
// not contain a trailing slash.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetOrder retrieves the accrual status for a single order number.
// It returns ErrNotRegistered for HTTP 204 and *RateLimitedError for HTTP 429.
func (c *Client) GetOrder(ctx context.Context, number string) (Result, error) {
	url := c.baseURL + "/api/orders/" + number
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{}, fmt.Errorf("build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var body struct {
			Order   string           `json:"order"`
			Status  Status           `json:"status"`
			Accrual *decimal.Decimal `json:"accrual"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return Result{}, fmt.Errorf("decode: %w", err)
		}
		return Result{Order: body.Order, Status: body.Status, Accrual: body.Accrual}, nil
	case http.StatusNoContent:
		return Result{}, ErrNotRegistered
	case http.StatusTooManyRequests:
		retry := parseRetryAfter(resp.Header.Get("Retry-After"))
		return Result{}, &RateLimitedError{RetryAfter: retry}
	default:
		return Result{}, fmt.Errorf("accrual: unexpected status %d", resp.StatusCode)
	}
}

func parseRetryAfter(h string) time.Duration {
	h = strings.TrimSpace(h)
	if h == "" {
		return 60 * time.Second
	}
	if secs, err := strconv.Atoi(h); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 60 * time.Second
}
