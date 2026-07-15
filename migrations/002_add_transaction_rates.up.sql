ALTER TABLE finance.transactions
ADD COLUMN IF NOT EXISTS input_in_bs BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE finance.transactions
ADD COLUMN IF NOT EXISTS rate_type TEXT NULL;

ALTER TABLE finance.transactions
ADD COLUMN IF NOT EXISTS rate_value NUMERIC(14,4) NOT NULL DEFAULT 0;

ALTER TABLE finance.transactions
DROP CONSTRAINT IF EXISTS transactions_rate_type_check;

ALTER TABLE finance.transactions
ADD CONSTRAINT transactions_rate_type_check CHECK (rate_type IN ('usd', 'eur', 'usdt'));
