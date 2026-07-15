ALTER TABLE finance.transactions
ADD COLUMN input_in_bs BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN rate_type TEXT NULL CHECK (rate_type IN ('usd', 'eur', 'usdt')),
ADD COLUMN rate_value NUMERIC(14,4) NOT NULL DEFAULT 0;
