-- +goose Up
ALTER TYPE outbox_status ADD VALUE 'claimed';

-- +goose Down
ALTER TYPE outbox_status RENAME TO outbox_status_old;