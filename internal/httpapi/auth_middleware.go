package httpapi

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// TokenVerifier is the subset of the auth service used by the auth middleware
// to validate session tokens.
type TokenVerifier interface {
	VerifyToken(token string) (uuid.UUID, error)
}

// AuthMiddleware returns an HTTP middleware that authenticates requests via a
// Bearer token in the Authorization header or a token cookie named
// "Authorization". Unauthorised requests receive HTTP 401.
func AuthMiddleware(v TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			userID, err := v.VerifyToken(token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if strings.HasPrefix(h, "Bearer ") {
			return strings.TrimPrefix(h, "Bearer ")
		}
		return h
	}
	if c, err := r.Cookie("Authorization"); err == nil {
		return c.Value
	}
	return ""
}
