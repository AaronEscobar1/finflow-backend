ALTER TABLE finance.transactions
ADD COLUMN IF NOT EXISTS debt_id BIGINT NULL REFERENCES finance.debts(id) ON DELETE CASCADE;

DROP INDEX IF EXISTS finance.idx_tx_debt_id;
CREATE INDEX idx_tx_debt_id ON finance.transactions(debt_id);
