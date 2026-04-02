-- +goose Up
CREATE TABLE samples_atac (
    sample_id TEXT PRIMARY KEY,
    site TEXT NOT NULL,
    status TEXT NOT NULL,
    sex TEXT NOT NULL,
    protocol TEXT NOT NULL
);

-- +goose Down
DROP TABLE samples_atac;
