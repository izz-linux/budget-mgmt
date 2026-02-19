-- 002_add_manually_moved.sql
-- Track whether a bill assignment was manually placed/moved by the user.
-- Auto-assign and optimizer use this to avoid overwriting user decisions.

ALTER TABLE bill_assignments ADD COLUMN IF NOT EXISTS manually_moved BOOLEAN NOT NULL DEFAULT FALSE;
