# Metadata Database Import

## Problem

I want to move the metadata source of truth into this repo and make it reliable before worrying about endpoints. The current need is not a running metadata API. The current need is a clean way to build and rebuild a SQLite database from the checked-in TSV files so the service can use that data later.

The data is relatively stable. Updates happen manually when new TSVs arrive or when source rows need correction. Because of that, the import process should be simple, explicit, and separate from normal service startup.

## Solution

Add a separate manual Go importer command at `go run ./cmd/importer` that reads the checked-in TSV files, creates a fresh SQLite database, applies `goose` migrations, and loads the metadata into the migrated schema.

The database shape is:
- one sample table per supported ome
- one shared `files` table across all omes

The importer will:
- read the current TSV files from `tsv/`
- create a brand new SQLite database
- apply SQL migrations to create the schema
- populate the supported ome-specific sample tables from their assay TSVs
- populate the global `files` table from the master files TSV
- fail fast on malformed source rows
- move any existing target database to `.bak` before replacing it

This keeps schema management in SQL, keeps import logic in Go under `internal/importer/`, and produces a stable DB artifact the service can open later.

## Expected Behavior

- I can run one Go command manually to build a fresh metadata database.
- The importer reads the checked-in TSV files without requiring per-run input flags.
- The importer applies `goose` migrations before loading data.
- Each supported ome gets its own sample table with columns matching that TSV’s metadata fields.
- The global `files` table contains file metadata keyed by `ome` and `sample_id`.
- If the output DB already exists, it is moved to a `.bak` file before the new DB is put in place.
- If a TSV row is malformed, the importer fails immediately with a useful error so I can fix the source data.
- File rows for unsupported omes are skipped silently.
- Minimal helper/query code exists so we can verify imported data in tests and later reuse it in the service.

## Implementation Decisions

- Use SQLite as the metadata store.
- Use `goose` SQL migrations for schema management.
- Keep this work database-only. No metadata endpoints yet.
- Use a separate manual Go command for import rather than tying import or migrations to service startup.
- Keep `cmd/importer` as the entrypoint and keep the importer implementation under `internal/importer/`.
- Default the importer to the checked-in `tsv/` directory.
- Hard-code the current TSV-to-table mapping for now.
- Use one sample table per ome instead of a single generic sample table.
- Use one global `files` table with an `ome` column.
- Join files to sample metadata by `sample_id`.
- Keep schema in SQL migrations and keep TSV parsing/loading in Go.
- Skip file rows whose ome is not currently supported.
- Always rebuild from scratch rather than incrementally updating rows.
- On rebuild, rotate any existing output database to `.bak`.
- Fail fast on invalid source rows such as malformed integer or boolean fields.
- Keep any helper/query layer small and separate from migration and import responsibilities.

Implementation note:
- Slice 1 and Slice 2 established the initial import path with schema created in Go.
- Going forward, that path is being refined so schema creation moves into `goose` migrations before additional import work continues.

## Testing Approach

Keep verification practical and manual for now.

Good verification for the completed work is:
- run the importer against the checked-in TSVs
- confirm `goose` migrations apply successfully to a fresh SQLite database
- inspect the created tables and columns
- inspect row counts by table
- query a known sample from at least two omes
- later, query the matching files for that sample
- confirm the imported values match the source TSVs

Formal automated test coverage can be added later if the importer logic becomes complex enough to justify it.

## Out of Scope

- HTTP endpoints
- frontend contract work
- service startup integration
- automatic refresh or scheduled rebuilds
- backup retention policy beyond a simple `.bak`
- generalized dynamic discovery of new TSVs or omes
- introducing a larger ORM or schema framework beyond `goose` + raw SQL
- redesigning download job flows around metadata selection yet
