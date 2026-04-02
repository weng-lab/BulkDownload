# Slice 1: Import Command And First Sample Table

## Dependencies

None

## Description

Add the manual Go importer command, create a fresh SQLite DB, and import one ome-specific sample table from its checked-in TSV.

Keep this first pass minimal. The goal is to prove the import flow end-to-end with the smallest useful slice before expanding to the rest of the schema.

## Expected Behaviors Addressed

- I can run one Go command manually to build a fresh metadata database.
- The importer reads the checked-in TSV files without requiring per-run input flags.
- Each supported ome gets its own sample table with columns matching that TSV’s metadata fields.

## Acceptance Criteria

- [ ] A manual Go importer command exists and can create a fresh SQLite database.
- [ ] One ome-specific sample table is created from the hard-coded TSV mapping.
- [ ] Sample rows are imported into that table.
- [ ] Duplicate file-level rows in the TSV collapse into one sample row per `sample_id`.
- [ ] The implementation stays intentionally small and does not pre-build generalized abstractions for later omes.

## QA

1. Run the importer manually.
2. Open the generated SQLite database.
3. Confirm one ome-specific sample table exists.
4. Confirm known sample rows are present once per `sample_id`.
5. Confirm the importer completes without requiring custom TSV input arguments.

---

*Appended after execution.*

## Completion

Built a minimal manual importer command at `go run ./cmd/importer`.

The importer currently creates `data/metadata.db`, creates a single `samples_atac` table, and imports deduplicated ATAC sample metadata from the checked-in `tsv/atac.tsv` file. It intentionally only supports ATAC in this first pass and keeps the TSV-to-table mapping hard-coded.

Key decisions made during implementation:
- Used a small `internal/metadata` package to hold the importer logic so the command stays thin and the code can be reused later.
- Used SQLite via `modernc.org/sqlite` to avoid external system dependencies.
- Dedupe is done by `sample_id`, and the importer fails if duplicate ATAC rows disagree on sample metadata.
- Blank trailing TSV rows are ignored, but any non-blank row missing `sample_id` fails the import.

Files created or modified:
- `cmd/importer/main.go`
- `internal/metadata/import.go`
- `go.mod`
- `go.sum`

Verification run completed successfully against the checked-in ATAC TSV. The generated database contains a single `samples_atac` table with 33 imported sample rows, which matches the non-empty unique `sample_id` values in the source TSV.

Anything the next slice should know:
- The current output path is `data/metadata.db`.
- Backup rotation is not implemented yet.
- Only the ATAC sample schema exists so far.
