
-- +goose Up
ALTER TABLE challenges
  ADD COLUMN expected_output text not null;


-- +goose Down
ALTER TABLE challenges
  DROP COLUMN expected_output;

