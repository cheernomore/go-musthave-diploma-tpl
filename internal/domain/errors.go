package domain

import "errors"

// Sentinel errors returned by the service and storage layers. Callers should
// compare against them with errors.Is rather than by string equality.
var (
	// ErrLoginTaken is returned when registration fails because the login is
	// already used by another user.
	ErrLoginTaken = errors.New("login already taken")

	// ErrInvalidCredentials is returned when login/password do not match a
	// registered user.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserNotFound is returned when a lookup by user identifier fails.
	ErrUserNotFound = errors.New("user not found")

	// ErrOrderAlreadyUploaded is returned when the same user re-uploads a
	// previously accepted order number.
	ErrOrderAlreadyUploaded = errors.New("order already uploaded by the same user")

	// ErrOrderOwnedByAnotherUser is returned when an order number was already
	// uploaded by a different user.
	ErrOrderOwnedByAnotherUser = errors.New("order owned by another user")

	// ErrInvalidOrderNumber is returned when the order number does not pass
	// the Luhn check.
	ErrInvalidOrderNumber = errors.New("invalid order number")

	// ErrInsufficientFunds is returned when a withdrawal exceeds the current
	// balance.
	ErrInsufficientFunds = errors.New("insufficient funds")
)
