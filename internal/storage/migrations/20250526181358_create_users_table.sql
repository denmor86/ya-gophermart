-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS USERS (
   id TEXT PRIMARY KEY NOT NUll,
   login TEXT UNIQUE  NOT NUll,
   password TEXT NOT NUll
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_login ON USERS (login);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_users_login;
DROP TABLE USERS;
-- +goose StatementEnd