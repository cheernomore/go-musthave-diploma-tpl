package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/cheernomore/go-musthave-diploma-tpl/internal/domain"
)

func newMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestUserCreateOK(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	u := domain.User{ID: uuid.New(), Login: "alice", PasswordHash: "x", CreatedAt: time.Now()}

	m.ExpectBeginTx(pgx.TxOptions{})
	m.ExpectExec("INSERT INTO users").
		WithArgs(u.ID, u.Login, u.PasswordHash, u.CreatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	m.ExpectExec("INSERT INTO balances").
		WithArgs(u.ID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	m.ExpectCommit()

	repo := NewUserRepository(m)
	if err := repo.Create(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserCreateLoginTaken(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	u := domain.User{ID: uuid.New(), Login: "alice", PasswordHash: "x", CreatedAt: time.Now()}

	m.ExpectBeginTx(pgx.TxOptions{})
	m.ExpectExec("INSERT INTO users").
		WithArgs(u.ID, u.Login, u.PasswordHash, u.CreatedAt).
		WillReturnError(&pgconn.PgError{Code: pgerrcode.UniqueViolation})
	m.ExpectRollback()

	repo := NewUserRepository(m)
	err := repo.Create(context.Background(), u)
	if !errors.Is(err, domain.ErrLoginTaken) {
		t.Fatalf("got %v", err)
	}
}

func TestUserFindByLoginNotFound(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	m.ExpectQuery("SELECT id, login").WithArgs("ghost").
		WillReturnRows(pgxmock.NewRows([]string{"id", "login", "password_hash", "created_at"}))

	repo := NewUserRepository(m)
	if _, err := repo.FindByLogin(context.Background(), "ghost"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestUserFindByLoginOK(t *testing.T) {
	m := newMock(t)
	defer m.Close()

	id := uuid.New()
	created := time.Now()
	m.ExpectQuery("SELECT id, login").WithArgs("alice").
		WillReturnRows(pgxmock.NewRows([]string{"id", "login", "password_hash", "created_at"}).
			AddRow(id, "alice", "hash", created))

	repo := NewUserRepository(m)
	u, err := repo.FindByLogin(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != id || u.Login != "alice" {
		t.Fatalf("got %+v", u)
	}
}
