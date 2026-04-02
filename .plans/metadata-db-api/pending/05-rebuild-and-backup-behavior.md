# Slice 5: Rebuild And Backup Behavior

## Dependencies

- Slice 4: Files Migration And File Import

## Description

Make the importer operationally usable by always building a fresh DB and moving any existing target DB to `.bak` before replacement.

Keep the workflow simple and explicit. This slice is about safe manual rebuilds, not automation. The importer should continue to run as a single command through `go run ./cmd/importer`.

## Expected Behaviors Addressed

- If the output DB already exists, it is moved to a `.bak` file before the new DB is put in place.
- I can run one Go command manually to build a fresh metadata database.

## Acceptance Criteria

- [ ] Re-running the importer against the same output path creates a fresh database.
- [ ] An existing output database is moved to `.bak` before replacement.
- [ ] The importer workflow remains manual and separate from service startup.
- [ ] The implementation stays small and does not add extra retention or rotation policy beyond `.bak`.

## QA

1. Run `go run ./cmd/importer` once and confirm the database file is created.
2. Run it again against the same target path.
3. Confirm the previous database file was moved to `.bak`.
4. Confirm the new database file exists.
5. Confirm the importer still runs as a standalone manual command.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
