-- Bills: opt-in flag + how many periods to spread over
ALTER TABLE bills ADD COLUMN IF NOT EXISTS sinking_fund_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE bills ADD COLUMN IF NOT EXISTS sinking_fund_periods INTEGER; -- null = user picks at generate time

-- Assignments: mark installments and link them back to the target period
ALTER TABLE bill_assignments ADD COLUMN IF NOT EXISTS is_sinking_fund BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE bill_assignments ADD COLUMN IF NOT EXISTS sinking_fund_for_period_id INTEGER REFERENCES pay_periods(id) ON DELETE SET NULL;
