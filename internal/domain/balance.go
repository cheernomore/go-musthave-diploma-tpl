package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Balance is the loyalty point balance of a single user.
// Current is the amount currently available for spending; Withdrawn is the
// total amount the user has spent so far.
type Balance struct {
	Current   decimal.Decimal
	Withdrawn decimal.Decimal
}

// Withdrawal is a single point withdrawal recorded against a user.
type Withdrawal struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	OrderNumber string
	Sum         decimal.Decimal
	ProcessedAt time.Time
}
