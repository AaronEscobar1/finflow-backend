ALTER TABLE finance.transactions
DROP COLUMN IF EXISTS input_in_bs,
DROP COLUMN IF EXISTS rate_type,
DROP COLUMN IF EXISTS rate_value;
