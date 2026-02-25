-- 004_add_deleted_bill_periods.sql
-- Track bill+period combinations that were explicitly deleted by the user
-- to prevent auto-assign from recreating them.

CREATE TABLE IF NOT EXISTS deleted_bill_periods (
    id            SERIAL PRIMARY KEY,
    bill_id       INTEGER NOT NULL REFERENCES bills(id) ON DELETE CASCADE,
    pay_period_id INTEGER NOT NULL REFERENCES pay_periods(id) ON DELETE CASCADE,
    deleted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(bill_id, pay_period_id)
);

CREATE INDEX IF NOT EXISTS idx_deleted_bill_periods_bill ON deleted_bill_periods(bill_id);
CREATE INDEX IF NOT EXISTS idx_deleted_bill_periods_period ON deleted_bill_periods(pay_period_id);
