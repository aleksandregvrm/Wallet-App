-- +goose Up
ALTER TABLE user_auth RENAME COLUMN access_token TO refresh_token;

-- +goose Down
ALTER TABLE user_auth RENAME COLUMN refresh_token TO access_token;
