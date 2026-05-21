package httpapi

import (
	"log/slog"
	"net/http"
)

// Deps bundles the dependencies required to build the HTTP router.
type Deps struct {
	Logger   *slog.Logger
	Auth     *AuthHandlers
	Orders   *OrderHandlers
	Balance  *BalanceHandlers
	Verifier TokenVerifier
}

// NewRouter assembles the gophermart HTTP router with the standard middleware
// chain (recover, logging, gzip) applied to every route. The /api/user/*
// endpoints that require authentication are additionally wrapped by the
// AuthMiddleware.
func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("POST /api/user/register", d.Auth.Register)
	mux.HandleFunc("POST /api/user/login", d.Auth.Login)

	authMW := AuthMiddleware(d.Verifier)
	mux.Handle("POST /api/user/orders", authMW(http.HandlerFunc(d.Orders.Upload)))
	mux.Handle("GET /api/user/orders", authMW(http.HandlerFunc(d.Orders.List)))
	mux.Handle("GET /api/user/balance", authMW(http.HandlerFunc(d.Balance.Get)))
	mux.Handle("POST /api/user/balance/withdraw", authMW(http.HandlerFunc(d.Balance.Withdraw)))
	mux.Handle("GET /api/user/withdrawals", authMW(http.HandlerFunc(d.Balance.Withdrawals)))

	return Chain(mux,
		Recover(d.Logger),
		Logging(d.Logger),
		Gzip(),
	)
}
