-- Add effective_from column to income_sources table
-- This column determines the earliest date from which pay periods should be generated
ALTER TABLE income_sources ADD COLUMN IF NOT EXISTS effective_from DATE;
