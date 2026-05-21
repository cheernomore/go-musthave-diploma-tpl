package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

type fakeUsers struct {
	byLogin   map[string]domain.User
	created   []domain.User
	createErr error
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byLogin: map[string]domain.User{}}
}

func (f *fakeUsers) Create(_ context.Context, u domain.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	if _, ok := f.byLogin[u.Login]; ok {
		return domain.ErrLoginTaken
	}
	f.byLogin[u.Login] = u
	f.created = append(f.created, u)
	return nil
}

func (f *fakeUsers) FindByLogin(_ context.Context, login string) (domain.User, error) {
	u, ok := f.byLogin[login]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func TestRegisterLoginRoundTrip(t *testing.T) {
	users := newFakeUsers()
	svc := New(users, "test-secret", time.Hour)

	token, err := svc.Register(context.Background(), "alice", "p4ssw0rd")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if token == "" {
		t.Fatal("empty token")
	}

	id, err := svc.VerifyToken(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if id != users.created[0].ID {
		t.Fatalf("subject mismatch")
	}

	loginToken, err := svc.Login(context.Background(), "alice", "p4ssw0rd")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loginToken == "" {
		t.Fatal("empty login token")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	users := newFakeUsers()
	svc := New(users, "test-secret", time.Hour)

	if _, err := svc.Register(context.Background(), "a", "b"); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Register(context.Background(), "a", "b")
	if !errors.Is(err, domain.ErrLoginTaken) {
		t.Fatalf("want ErrLoginTaken, got %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	users := newFakeUsers()
	svc := New(users, "test-secret", time.Hour)
	if _, err := svc.Register(context.Background(), "a", "b"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(context.Background(), "a", "x"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	svc := New(newFakeUsers(), "test-secret", time.Hour)
	if _, err := svc.Login(context.Background(), "ghost", "x"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestVerifyTokenBadSignature(t *testing.T) {
	usersA := newFakeUsers()
	a := New(usersA, "secret-a", time.Hour)
	tok, err := a.Register(context.Background(), "u", "p")
	if err != nil {
		t.Fatal(err)
	}
	b := New(newFakeUsers(), "secret-b", time.Hour)
	if _, err := b.VerifyToken(tok); err == nil {
		t.Fatal("expected error verifying foreign token")
	}
}
