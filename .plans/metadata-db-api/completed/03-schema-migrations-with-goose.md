# Slice 3: Schema Migrations With Goose

## Dependencies

- Slice 2: All Supported Sample Tables

## Description

Add `goose`-managed SQL migrations for the current sample tables and refactor the importer so schema creation no longer lives in Go.

Keep this pass focused on moving schema responsibility into migrations while preserving the current sample import behavior.

## Expected Behaviors Addressed

- I can run one Go command manually to build a fresh metadata database.
- The importer applies `goose` migrations before loading data.
- Each supported ome gets its own sample table with columns matching that TSV’s metadata fields.

## Acceptance Criteria

- [ ] `goose` is added for migration management.
- [ ] SQL migration files exist for `samples_atac`, `samples_rna`, and `samples_wgbs`.
- [ ] The importer applies migrations before importing sample TSV data.
- [ ] Table creation logic is removed from the Go importer code.
- [ ] Running the importer still produces the same sample tables and sample data as before.

## QA

1. Run the importer manually.
2. Open the generated SQLite database.
3. Confirm the sample tables exist.
4. Confirm the schema came from applied migrations rather than in-code `CREATE TABLE` statements.
5. Confirm sample data still imports correctly for all currently supported omes.

---

*Appended after execution.*

## Completion

Added `goose`-managed SQL migrations for the current sample tables and refactored the importer so schema creation no longer lives in Go.

What was built:
- Added `github.com/pressly/goose/v3` as a dependency.
- Added migration files for `samples_atac`, `samples_rna`, and `samples_wgbs` under `db/migrations/`.
- Updated the importer to apply migrations before loading sample TSV data.
- Removed the in-code table creation helpers from `internal/metadata/import.go`.

Key decisions made during implementation:
- Kept migrations as plain SQL files, one file per table.
- Kept the importer entrypoint unchanged at `go run ./cmd/importer`.
- Used `goose.Up` from the importer so schema application stays part of the single manual command.
- Left the sample import logic largely unchanged so this slice only moved schema responsibility, not data-loading behavior.

Files created or modified:
- `go.mod`
- `go.sum`
- `db/migrations/001_create_samples_atac.sql`
- `db/migrations/002_create_samples_rna.sql`
- `db/migrations/003_create_samples_wgbs.sql`
- `internal/metadata/import.go`

Verification run completed successfully. Running `go run ./cmd/importer` applied the three migrations and produced the same sample tables and row counts as before:
- `samples_atac`: 33
- `samples_rna`: 15
- `samples_wgbs`: 15

Anything the next slice should know:
- Schema creation now belongs in `db/migrations/`.
- The next table addition should be done as another migration, not with Go DDL.
- The importer still only handles sample tables; the `files` table does not exist yet.
