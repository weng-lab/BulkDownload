# Slice 2: All Supported Sample Tables

## Dependencies

- Slice 1: Import Command And First Sample Table

## Description

Expand the importer to create and populate all currently supported ome-specific sample tables from the hard-coded TSV mapping.

Keep the implementation incremental. Reuse only what is clearly needed after Slice 1, and prefer explicit mappings over premature generalization.

## Expected Behaviors Addressed

- The importer reads the checked-in TSV files without requiring per-run input flags.
- Each supported ome gets its own sample table with columns matching that TSV’s metadata fields.

## Acceptance Criteria

- [ ] All currently supported ome-specific sample tables are created.
- [ ] Each supported table is populated from its checked-in TSV.
- [ ] Assay-specific metadata differences are reflected in the table schemas.
- [ ] The importer still relies on a simple hard-coded TSV-to-table mapping.
- [ ] The implementation only introduces shared code where it clearly reduces repetition without obscuring the import flow.

## QA

1. Run the importer manually.
2. Open the generated SQLite database.
3. Confirm all expected sample tables exist.
4. Confirm each table contains rows.
5. Check that assay-specific columns appear only where expected.

---

*Appended after execution.*

## Completion

Expanded the importer so it now creates and populates all currently supported sample tables: `samples_atac`, `samples_rna`, and `samples_wgbs`.

Key decisions made during implementation:
- Kept ATAC explicit because it has an extra `protocol` column.
- Added a small shared helper only for the RNA and WGBS sample tables because they currently share the same sample-level metadata shape.
- Kept the TSV-to-table mapping hard-coded in `BuildDatabase`.
- Kept the import flow simple: create table, read TSV, dedupe by `sample_id`, insert rows.

Files created or modified:
- `internal/metadata/import.go`

Verification run completed successfully against the checked-in TSVs. The generated database now contains:
- `samples_atac` with 33 rows
- `samples_rna` with 15 rows
- `samples_wgbs` with 15 rows

The table schemas reflect the expected assay differences:
- `samples_atac` includes `protocol`
- `samples_rna` and `samples_wgbs` do not

Anything the next slice should know:
- The importer still only handles sample tables. No `files` table exists yet.
- The output database remains `data/metadata.db`.
- Backup rotation is still not implemented.
