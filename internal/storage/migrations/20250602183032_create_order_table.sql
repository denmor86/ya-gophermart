-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ORDERS (
   number TEXT PRIMARY KEY UNIQUE NOT NUll,
   user_uuid TEXT NOT NUll,
   status TEXT NOT NUll,
   uploaded_at TIMESTAMP NOT NUll,
   accrual NUMERIC DEFAULT 0,
   is_processing BOOLEAN DEFAULT FALSE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_uuid ON ORDERS (user_uuid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_uploaded_at ON ORDERS (uploaded_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_users_uuid;
DROP INDEX idx_uploaded_at;
DROP TABLE ORDERS;
-- +goose StatementEnd