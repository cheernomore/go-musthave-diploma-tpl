CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY,
    login         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS orders (
    number       TEXT PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       TEXT NOT NULL,
    accrual      NUMERIC(20, 2),
    uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS orders_user_uploaded_idx
    ON orders (user_id, uploaded_at DESC);

CREATE INDEX IF NOT EXISTS orders_status_idx
    ON orders (status)
    WHERE status IN ('NEW', 'PROCESSING');

CREATE TABLE IF NOT EXISTS balances (
    user_id   UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current   NUMERIC(20, 2) NOT NULL DEFAULT 0,
    withdrawn NUMERIC(20, 2) NOT NULL DEFAULT 0,
    CONSTRAINT balances_non_negative CHECK (current >= 0 AND withdrawn >= 0)
);

CREATE TABLE IF NOT EXISTS withdrawals (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_number TEXT NOT NULL,
    sum          NUMERIC(20, 2) NOT NULL CHECK (sum > 0),
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS withdrawals_user_processed_idx
    ON withdrawals (user_id, processed_at DESC);
