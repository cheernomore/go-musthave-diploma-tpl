package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

type fakeVerifier struct {
	id  uuid.UUID
	err error
}

func (f fakeVerifier) VerifyToken(string) (uuid.UUID, error) { return f.id, f.err }

func TestAuthMiddlewareMissingToken(t *testing.T) {
	mw := AuthMiddleware(fakeVerifier{err: errors.New("nope")})
	h := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not be called")
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestAuthMiddlewareValidBearer(t *testing.T) {
	want := uuid.New()
	mw := AuthMiddleware(fakeVerifier{id: want})

	var got uuid.UUID
	h := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		id, ok := UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("missing user id")
		}
		got = id
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer xxx")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if got != want {
		t.Fatalf("user id mismatch")
	}
}

func TestAuthMiddlewareCookie(t *testing.T) {
	want := uuid.New()
	mw := AuthMiddleware(fakeVerifier{id: want})

	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "Authorization", Value: "xxx"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
}
