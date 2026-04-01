# Slice 3: Global Files Table

## Dependencies

- Slice 2: All Supported Sample Tables

## Description

Add the shared `files` table and import file metadata from the master files TSV, skipping unsupported omes silently.

Keep this pass focused on getting the core file rows into SQLite with the planned columns. Avoid extra derived data or service-facing concerns.

## Expected Behaviors Addressed

- The global `files` table contains file metadata keyed by `ome` and `sample_id`.
- File rows for unsupported omes are skipped silently.

## Acceptance Criteria

- [ ] A shared `files` table exists in the SQLite database.
- [ ] File metadata from the master TSV is imported into `files`.
- [ ] `size` is stored as an integer.
- [ ] `open_access` is stored as a boolean.
- [ ] Rows for unsupported omes are skipped without failing the import.

## QA

1. Run the importer manually.
2. Open the generated SQLite database.
3. Confirm the `files` table exists.
4. Confirm file rows are present for supported omes.
5. Confirm unsupported ome rows from the master TSV are not imported.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
