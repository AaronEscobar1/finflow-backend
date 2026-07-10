-- Extensiones requeridas
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Esquemas
CREATE SCHEMA IF NOT EXISTS security;
CREATE SCHEMA IF NOT EXISTS finance;
CREATE SCHEMA IF NOT EXISTS notifications;
CREATE SCHEMA IF NOT EXISTS telemetry;

-- Función compartida para mantener updated_at
CREATE OR REPLACE FUNCTION security.set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================ SECURITY ============================
CREATE TABLE security.users (
    id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email              CITEXT UNIQUE NOT NULL,
    password_hash      TEXT NULL,
    full_name          TEXT NULL,
    avatar_url         TEXT NULL,
    auth_provider      TEXT NOT NULL DEFAULT 'email' CHECK (auth_provider IN ('email','google')),
    google_sub         TEXT UNIQUE NULL,
    email_verified     BOOLEAN NOT NULL DEFAULT false,
    preferred_currency TEXT NOT NULL DEFAULT 'USD',
    preferred_language TEXT NOT NULL DEFAULT 'es',
    theme_preference   TEXT NOT NULL DEFAULT 'system' CHECK (theme_preference IN ('system','light','dark')),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_users_updated BEFORE UPDATE ON security.users
    FOR EACH ROW EXECUTE FUNCTION security.set_updated_at();

CREATE TABLE security.sessions (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES security.users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ NULL,
    ip          TEXT NULL,
    user_agent  TEXT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessions_token_hash ON security.sessions(token_hash);
CREATE INDEX idx_sessions_user ON security.sessions(user_id);

CREATE TABLE security.user_otp (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email       CITEXT NOT NULL,
    otp_code    TEXT NOT NULL,
    action      TEXT NOT NULL CHECK (action IN ('REGISTER','RECOVER')),
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_user_otp_lookup ON security.user_otp(email, action, created_at DESC);

-- ============================ FINANCE ============================
CREATE TABLE finance.accounts (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES security.users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'cash' CHECK (type IN ('cash','bank','card','investment','other')),
    currency   TEXT NOT NULL DEFAULT 'USD',
    color      TEXT NULL,
    icon_key   TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_accounts_user ON finance.accounts(user_id) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_accounts_updated BEFORE UPDATE ON finance.accounts
    FOR EACH ROW EXECUTE FUNCTION security.set_updated_at();

CREATE TABLE finance.categories (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES security.users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL CHECK (type IN ('income','expense')),
    icon_key   TEXT NULL,
    color      TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX uq_categories_user_name_type ON finance.categories(user_id, name, type) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_categories_updated BEFORE UPDATE ON finance.categories
    FOR EACH ROW EXECUTE FUNCTION security.set_updated_at();

CREATE TABLE finance.transactions (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES security.users(id) ON DELETE CASCADE,
    account_id  BIGINT NOT NULL REFERENCES finance.accounts(id),
    amount      NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    type        TEXT NOT NULL CHECK (type IN ('income','expense')),
    category_id BIGINT NOT NULL REFERENCES finance.categories(id),
    currency    TEXT NOT NULL DEFAULT 'USD',
    description TEXT NOT NULL DEFAULT '',
    date        TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ NULL
);
CREATE INDEX idx_tx_user_date ON finance.transactions(user_id, date DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_tx_user_category ON finance.transactions(user_id, category_id);
CREATE INDEX idx_tx_user_account ON finance.transactions(user_id, account_id);
CREATE INDEX idx_tx_description_trgm ON finance.transactions USING gin (description gin_trgm_ops);
CREATE TRIGGER trg_tx_updated BEFORE UPDATE ON finance.transactions
    FOR EACH ROW EXECUTE FUNCTION security.set_updated_at();

CREATE TABLE finance.debts (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES security.users(id) ON DELETE CASCADE,
    amount      NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    direction   TEXT NOT NULL CHECK (direction IN ('receivable','payable')),
    person_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    currency    TEXT NOT NULL DEFAULT 'USD',
    date        TIMESTAMPTZ NOT NULL,
    due_date    TIMESTAMPTZ NULL,
    is_paid     BOOLEAN NOT NULL DEFAULT false,
    paid_at     TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ NULL
);
CREATE INDEX idx_debts_user_dir ON finance.debts(user_id, direction) WHERE deleted_at IS NULL;
CREATE INDEX idx_debts_due ON finance.debts(due_date) WHERE deleted_at IS NULL AND is_paid = false;
CREATE TRIGGER trg_debts_updated BEFORE UPDATE ON finance.debts
    FOR EACH ROW EXECUTE FUNCTION security.set_updated_at();

CREATE TABLE finance.budgets (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES security.users(id) ON DELETE CASCADE,
    category_id BIGINT NULL REFERENCES finance.categories(id),
    amount      NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    period      TEXT NOT NULL DEFAULT 'monthly' CHECK (period IN ('weekly','monthly','yearly')),
    currency    TEXT NOT NULL DEFAULT 'USD',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX uq_budgets_user_cat_period ON finance.budgets(user_id, COALESCE(category_id, 0), period) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_budgets_updated BEFORE UPDATE ON finance.budgets
    FOR EACH ROW EXECUTE FUNCTION security.set_updated_at();

-- ========================== NOTIFICATIONS ==========================
CREATE TABLE notifications.devices (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES security.users(id) ON DELETE CASCADE,
    token      TEXT UNIQUE NOT NULL,
    platform   TEXT NOT NULL CHECK (platform IN ('android','ios','web')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_devices_user ON notifications.devices(user_id);
CREATE TRIGGER trg_devices_updated BEFORE UPDATE ON notifications.devices
    FOR EACH ROW EXECUTE FUNCTION security.set_updated_at();

CREATE TABLE notifications.notifications (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES security.users(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'system' CHECK (type IN ('debt_due','budget_exceeded','system')),
    payload    JSONB NULL,
    read_at    TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notifications_user ON notifications.notifications(user_id, created_at DESC);

-- ============================ TELEMETRY ============================
CREATE TABLE telemetry.api_logs (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    track_code    TEXT NOT NULL,
    user_id       BIGINT NULL,
    method        TEXT NOT NULL,
    path          TEXT NOT NULL,
    http_status   INT NOT NULL,
    response_code TEXT NOT NULL,
    latency_ms    BIGINT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_api_logs_created ON telemetry.api_logs(created_at DESC);
CREATE INDEX idx_api_logs_user ON telemetry.api_logs(user_id);
