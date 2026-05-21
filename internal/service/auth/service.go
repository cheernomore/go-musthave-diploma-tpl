package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

// UserRepository abstracts persistence operations for users. It is consumed
// by the auth service and implemented by storage adapters.
type UserRepository interface {
	// Create stores a new user. It must return domain.ErrLoginTaken if the
	// login is not unique.
	Create(ctx context.Context, u domain.User) error
	// FindByLogin loads a user by login. It returns domain.ErrUserNotFound if
	// no such user exists.
	FindByLogin(ctx context.Context, login string) (domain.User, error)
}

// Service performs registration, authentication and token verification.
// It is safe for concurrent use.
type Service struct {
	users  UserRepository
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// New constructs an auth Service. The secret must be non-empty and the ttl
// must be positive; otherwise New panics, since these are programmer errors.
func New(users UserRepository, secret string, ttl time.Duration) *Service {
	if secret == "" {
		panic("auth: empty secret")
	}
	if ttl <= 0 {
		panic("auth: non-positive token TTL")
	}
	return &Service{
		users:  users,
		secret: []byte(secret),
		ttl:    ttl,
		now:    time.Now,
	}
}

// Register creates a new user with the given login and password and returns
// a signed JWT token authenticating the new session. The login must not be
// empty; the password must contain at least one character.
func (s *Service) Register(ctx context.Context, login, password string) (string, error) {
	if login == "" || password == "" {
		return "", fmt.Errorf("login and password required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	u := domain.User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: string(hash),
		CreatedAt:    s.now(),
	}
	if err := s.users.Create(ctx, u); err != nil {
		return "", err
	}
	return s.issueToken(u.ID)
}

// Login authenticates an existing user and returns a signed JWT token.
// It returns domain.ErrInvalidCredentials when login/password do not match.
func (s *Service) Login(ctx context.Context, login, password string) (string, error) {
	if login == "" || password == "" {
		return "", domain.ErrInvalidCredentials
	}
	u, err := s.users.FindByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", domain.ErrInvalidCredentials
		}
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", domain.ErrInvalidCredentials
	}
	return s.issueToken(u.ID)
}

// VerifyToken parses and validates a signed token, returning the user ID
// stored in its subject claim.
func (s *Service) VerifyToken(token string) (uuid.UUID, error) {
	parsed, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok || !parsed.Valid {
		return uuid.Nil, fmt.Errorf("invalid token")
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid subject: %w", err)
	}
	return id, nil
}

func (s *Service) issueToken(userID uuid.UUID) (string, error) {
	now := s.now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
