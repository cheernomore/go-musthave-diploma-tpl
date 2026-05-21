package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderStatus is the processing status of an order in the loyalty system.
type OrderStatus string

// Order status constants as exposed by the public API.
const (
	// OrderStatusNew means the order is accepted but not yet processed.
	OrderStatusNew OrderStatus = "NEW"
	// OrderStatusProcessing means the accrual is being calculated.
	OrderStatusProcessing OrderStatus = "PROCESSING"
	// OrderStatusInvalid means the accrual system refused calculation.
	OrderStatusInvalid OrderStatus = "INVALID"
	// OrderStatusProcessed is the terminal success state.
	OrderStatusProcessed OrderStatus = "PROCESSED"
)

// Order represents a loyalty system order tied to a user.
// Accrual is nil until the accrual system reports a final value.
type Order struct {
	Number     string
	UserID     uuid.UUID
	Status     OrderStatus
	Accrual    *decimal.Decimal
	UploadedAt time.Time
}

// PendingOrder is a projection of an order awaiting accrual processing,
// carrying only the fields the worker needs.
type PendingOrder struct {
	Number string
	UserID uuid.UUID
}
