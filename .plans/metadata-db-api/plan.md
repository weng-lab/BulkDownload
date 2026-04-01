# Metadata Database Import

## Problem

I want to move the metadata source of truth into this repo and make it reliable before worrying about endpoints. The current need is not a running metadata API. The current need is a clean way to build and rebuild a SQLite database from the checked-in TSV files so the service can use that data later.

The data is relatively stable. Updates happen manually when new TSVs arrive or when source rows need correction. Because of that, the import process should be simple, explicit, and separate from normal service startup.

## Solution

Add a separate manual Go importer command that reads the checked-in TSV files, creates a fresh SQLite database, and loads the metadata into a small relational schema.

The database shape is:
- one sample table per supported ome
- one shared `files` table across all omes

The importer will:
- read the current TSV files from `tsv/`
- create a brand new SQLite database
- create the supported ome-specific sample tables first
- populate those sample tables from their assay TSVs
- create and populate the global `files` table from the master files TSV
- fail fast on malformed source rows
- move any existing target database to `.bak` before replacing it

This keeps import logic separate from the service and produces a stable DB artifact the service can open later.

## Expected Behavior

- I can run one Go command manually to build a fresh metadata database.
- The importer reads the checked-in TSV files without requiring per-run input flags.
- Each supported ome gets its own sample table with columns matching that TSV’s metadata fields.
- The global `files` table contains file metadata keyed by `ome` and `sample_id`.
- If the output DB already exists, it is moved to a `.bak` file before the new DB is put in place.
- If a TSV row is malformed, the importer fails immediately with a useful error so I can fix the source data.
- File rows for unsupported omes are skipped silently.
- Minimal helper/query code exists so we can verify imported data in tests and later reuse it in the service.

## Implementation Decisions

- Use SQLite as the metadata store.
- Keep this work database-only. No metadata endpoints yet.
- Use a separate manual Go command for import rather than tying import to service startup.
- Default the importer to the checked-in `tsv/` directory.
- Hard-code the current TSV-to-table mapping for now.
- Use one sample table per ome instead of a single generic sample table.
- Use one global `files` table with an `ome` column.
- Join files to sample metadata by `sample_id`.
- Build sample tables before building the `files` table.
- Skip file rows whose ome is not currently supported.
- Always rebuild from scratch rather than incrementally updating rows.
- On rebuild, rotate any existing output database to `.bak`.
- Fail fast on invalid source rows such as malformed integer or boolean fields.
- Keep a small helper/query layer for opening the DB, creating schema, importing rows, and validating/querying results.

## Testing Approach

Test the importer and database layer directly with temporary SQLite databases and small fixture TSV inputs.

Key test coverage:
- schema creation for supported ome-specific sample tables and the global `files` table
- importing sample metadata from current assay TSV shapes
- deduping sample rows down to one row per `sample_id` in each sample table
- importing file rows from the master files TSV
- parsing `size` as an integer
- parsing `open_access` as a boolean
- skipping unsupported ome file rows
- failing on malformed source rows with clear errors
- rotating an existing target DB to `.bak`
- minimal helper queries returning expected sample rows and file rows after import

Good verification for the completed work is:
- run the importer against the checked-in TSVs
- inspect row counts by table
- query a known sample from at least two omes
- query the matching files for that sample
- confirm the imported values match the source TSVs

## Out of Scope

- HTTP endpoints
- frontend contract work
- service startup integration
- automatic refresh or scheduled rebuilds
- backup retention policy beyond a simple `.bak`
- generalized dynamic discovery of new TSVs or omes
- redesigning download job flows around metadata selection yet
