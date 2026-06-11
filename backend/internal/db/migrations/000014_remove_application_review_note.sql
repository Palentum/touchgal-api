-- +goose Up
ALTER TABLE api_applications DROP COLUMN review_note;

-- +goose Down
ALTER TABLE api_applications ADD COLUMN review_note text NOT NULL DEFAULT '';
