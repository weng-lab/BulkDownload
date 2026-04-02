-- +goose Up
CREATE TABLE files (
    ome TEXT NOT NULL,
    sample_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    file_type TEXT NOT NULL,
    size INTEGER NOT NULL,
    open_access BOOLEAN NOT NULL,
    PRIMARY KEY (ome, sample_id, filename)
);

-- +goose Down
DROP TABLE files;
