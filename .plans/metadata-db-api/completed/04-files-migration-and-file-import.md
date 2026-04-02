# Slice 4: Files Migration And File Import

## Dependencies

- Slice 3: Schema Migrations With Goose

## Description

Add the `files` table as a migration and import file metadata from the master files TSV, skipping unsupported omes silently.

Keep this pass focused on the core file metadata columns and the planned supported-ome filtering. Keep `cmd/importer` as the entrypoint and add the file import logic under `internal/importer/` alongside the existing ome-specific import files.

## Expected Behaviors Addressed

- The global `files` table contains file metadata keyed by `ome` and `sample_id`.
- File rows for unsupported omes are skipped silently.

## Acceptance Criteria

- [ ] A migration exists for the `files` table.
- [ ] The importer loads file metadata from `tsv/mohd_phase_0_download_files.tsv`.
- [ ] `size` is stored as an integer.
- [ ] `open_access` is stored as a boolean.
- [ ] Rows for unsupported omes are skipped without failing the import.

## QA

1. Run `go run ./cmd/importer` and confirm the database file is created.
2. Open the generated SQLite database.
3. Confirm the `files` table exists.
4. Confirm file rows are present for supported omes.
5. Confirm unsupported ome rows from the master TSV are not imported.

---

*Appended after execution.*

## Completion

Added the `files` table as a migration and imported file metadata from `tsv/mohd_phase_0_download_files.tsv` into the SQLite database.

What was built:
- Added `db/migrations/004_create_files.sql`.
- Added `internal/importer/files.go` for file TSV parsing and insertion.
- Updated `internal/importer/import.go` so the importer now loads file metadata after the sample tables.

Key decisions made during implementation:
- Kept the `files` table minimal with columns: `ome`, `sample_id`, `filename`, `file_type`, `size`, and `open_access`.
- Used a primary key on `(ome, sample_id, filename)`.
- Added an explicit supported mapping from master TSV `file_ome` values to internal ome values:
  - `ATAC_SEQ` -> `atac`
  - `RNA_SEQ` -> `rna`
  - `WGBS` -> `wgbs`
- Rows for unsupported omes are skipped silently as planned.
- `size` is parsed as `int64` and `open_access` is parsed from the source `True`/`False` strings.

Files created or modified:
- `db/migrations/004_create_files.sql`
- `internal/importer/files.go`
- `internal/importer/import.go`

Verification run completed successfully. Running `go run ./cmd/importer` applied the new migration and imported supported file rows into `files` with these counts:
- `atac`: 298
- `rna`: 166
- `wgbs`: 196
- total: 660

Anything the next slice should know:
- The database now contains `files` alongside the three sample tables.
- The master files TSV includes aggregate rows such as `ATAC_allsamples`, and they are currently imported if their `file_ome` maps to a supported ome.
- Backup rotation is still not implemented yet.
