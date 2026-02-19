-- 001_initial_schema.sql

CREATE TABLE IF NOT EXISTS bills (
    id               SERIAL PRIMARY KEY,
    name             VARCHAR(255) NOT NULL,
    default_amount   DECIMAL(10,2),
    due_day          INTEGER,
    recurrence       VARCHAR(20) NOT NULL DEFAULT 'monthly',
    recurrence_detail JSONB,
    is_autopay       BOOLEAN NOT NULL DEFAULT FALSE,
    category         VARCHAR(100) NOT NULL DEFAULT '',
    notes            TEXT NOT NULL DEFAULT '',
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order       INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS credit_cards (
    id              SERIAL PRIMARY KEY,
    bill_id         INTEGER NOT NULL REFERENCES bills(id) ON DELETE CASCADE,
    card_label      VARCHAR(100) NOT NULL DEFAULT '',
    statement_day   INTEGER NOT NULL,
    due_day         INTEGER NOT NULL,
    issuer          VARCHAR(100) NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS income_sources (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    pay_schedule    VARCHAR(20) NOT NULL,
    schedule_detail JSONB NOT NULL,
    default_amount  DECIMAL(10,2),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pay_periods (
    id               SERIAL PRIMARY KEY,
    income_source_id INTEGER NOT NULL REFERENCES income_sources(id),
    pay_date         DATE NOT NULL,
    expected_amount  DECIMAL(10,2),
    actual_amount    DECIMAL(10,2),
    notes            TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(income_source_id, pay_date)
);

CREATE INDEX IF NOT EXISTS idx_pay_periods_date ON pay_periods(pay_date);

CREATE TABLE IF NOT EXISTS bill_assignments (
    id              SERIAL PRIMARY KEY,
    bill_id         INTEGER NOT NULL REFERENCES bills(id),
    pay_period_id   INTEGER NOT NULL REFERENCES pay_periods(id),
    planned_amount  DECIMAL(10,2),
    forecast_amount DECIMAL(10,2),
    actual_amount   DECIMAL(10,2),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    deferred_to_id  INTEGER REFERENCES pay_periods(id),
    is_extra        BOOLEAN NOT NULL DEFAULT FALSE,
    extra_name      VARCHAR(255) NOT NULL DEFAULT '',
    notes           TEXT NOT NULL DEFAULT '',
    manually_moved  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(bill_id, pay_period_id)
);

CREATE INDEX IF NOT EXISTS idx_assignments_period ON bill_assignments(pay_period_id);
CREATE INDEX IF NOT EXISTS idx_assignments_bill ON bill_assignments(bill_id);
CREATE INDEX IF NOT EXISTS idx_assignments_status ON bill_assignments(status);

CREATE TABLE IF NOT EXISTS import_history (
    id           SERIAL PRIMARY KEY,
    filename     VARCHAR(255),
    imported_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    row_count    INTEGER,
    period_count INTEGER,
    status       VARCHAR(20) NOT NULL DEFAULT 'completed',
    error_log    TEXT
);

CREATE TABLE IF NOT EXISTS app_settings (
    id            INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    default_view  VARCHAR(20) DEFAULT 'grid',
    periods_ahead INTEGER DEFAULT 8,
    theme         VARCHAR(20) DEFAULT 'light',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO app_settings (id) VALUES (1) ON CONFLICT DO NOTHING;
