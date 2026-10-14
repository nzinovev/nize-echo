-- Baseline migration: it creates goose's version table on an empty database
-- and gives later migrations a starting point. The v1 schema starts in F-05.

-- +goose Up
SELECT 1;

-- +goose Down
SELECT 1;
