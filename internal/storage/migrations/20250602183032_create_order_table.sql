-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ORDERS (
   number TEXT PRIMARY KEY UNIQUE NOT NULL,
   user_id TEXT NOT NULL,
   status TEXT NOT NULL,
   retry_count NUMERIC DEFAULT 0,
   updated_at TIMESTAMP NOT NULL,
   created_at TIMESTAMP NOT NULL,
   accrual DECIMAL(10, 2) DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_user_id ON ORDERS (user_id);
CREATE INDEX IF NOT EXISTS idx_created_at ON ORDERS (created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_user_id;
DROP INDEX idx_created_at;
DROP TABLE ORDERS;
-- +goose StatementEnd