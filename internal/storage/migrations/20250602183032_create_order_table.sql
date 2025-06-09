-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ORDERS (
   number TEXT PRIMARY KEY UNIQUE NOT NUll,
   user_uuid TEXT NOT NUll,
   status TEXT NOT NUll,
   retry_count NUMERIC DEFAULT 0,
   updated_at TIMESTAMP NOT NUll,
   created_at  TIMESTAMP NOT NUll,
   accrual NUMERIC DEFAULT 0,
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_uuid ON ORDERS (user_uuid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_created_at  ON ORDERS (created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_users_uuid;
DROP INDEX idx_created_at;
DROP TABLE ORDERS;
-- +goose StatementEnd