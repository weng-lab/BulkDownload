# Slice 6: Minimal Query Helpers

## Dependencies

- Slice 5: Rebuild And Backup Behavior

## Description

Add a small helper layer for opening the DB and querying sample rows and file rows so the imported DB is easy to inspect and later reuse in the service.

Keep this helper layer minimal. It should support straightforward inspection and future reuse, without designing a full metadata service API ahead of need. It should fit around the existing `internal/importer/` layout rather than pulling importer and query responsibilities back together.

## Expected Behaviors Addressed

- Minimal helper/query code exists so we can verify imported data manually and later reuse it in the service.

## Acceptance Criteria

- [ ] There is a small reusable way to open the metadata database.
- [ ] There is a simple way to fetch sample metadata for a given supported ome and `sample_id`.
- [ ] There is a simple way to fetch file rows for a given supported ome and `sample_id`.
- [ ] The helper code remains intentionally narrow and avoids premature abstractions.

## QA

1. Build the database with `go run ./cmd/importer`.
2. Use the helper code to open the database.
3. Fetch a known sample row for a supported ome.
4. Fetch the associated file rows for that sample.
5. Confirm the helper layer stays small and focused on direct database access.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
