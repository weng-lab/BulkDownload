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

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
