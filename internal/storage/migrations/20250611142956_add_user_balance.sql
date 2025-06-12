-- +goose Up
-- +goose StatementBegin
ALTER TABLE USERS
ADD balance DECIMAL(10, 2) DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE USERS
DROP COLUMN balance;
-- +goose StatementEnd
