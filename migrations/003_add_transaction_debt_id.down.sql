DROP INDEX IF EXISTS finance.idx_tx_debt_id;

ALTER TABLE finance.transactions
DROP COLUMN IF EXISTS debt_id;
