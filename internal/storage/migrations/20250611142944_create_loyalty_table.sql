-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS LOYALTY (
    number SERIAL PRIMARY KEY,         
    user_id TEXT NOT NULL,             
    order_number TEXT UNIQUE NOT NULL, 
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
    amount DECIMAL(10, 2) NOT NULL 
);

CREATE INDEX IF NOT EXISTS idx_user_id ON LOYALTY (user_id);
CREATE INDEX IF NOT EXISTS idx_order_number ON LOYALTY (order_number);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_user_id;
DROP INDEX idx_order_number;
DROP TABLE LOYALTY;
-- +goose StatementEnd