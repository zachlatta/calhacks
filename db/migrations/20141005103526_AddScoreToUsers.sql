
-- +goose Up
ALTER TABLE users
  ADD COLUMN score integer not null;


-- +goose Down
ALTER TABLE users
  DROP COLUMN score;

