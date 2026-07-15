ALTER TABLE finance.transactions
ADD COLUMN debt_id BIGINT NULL REFERENCES finance.debts(id) ON DELETE CASCADE;

CREATE INDEX idx_tx_debt_id ON finance.transactions(debt_id);
