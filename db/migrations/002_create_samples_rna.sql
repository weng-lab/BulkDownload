-- +goose Up
CREATE TABLE samples_rna (
    sample_id TEXT PRIMARY KEY,
    site TEXT NOT NULL,
    sex TEXT NOT NULL,
    status TEXT NOT NULL
);

-- +goose Down
DROP TABLE samples_rna;
