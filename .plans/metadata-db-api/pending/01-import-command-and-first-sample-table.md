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

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
