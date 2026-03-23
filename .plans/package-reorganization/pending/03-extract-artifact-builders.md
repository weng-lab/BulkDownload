# Slice 3: Extract Artifact Builders

## Dependencies

Slice 1: Extract Job Model And Store.

## Description

Move zip, tarball, and script generation helpers into `internal/artifacts`, including progress tracking and artifact file helpers, while keeping the existing dispatch flows and generated outputs unchanged.

## Expected Behaviors Addressed

- Archive and script generation live in one obvious place.
- The service keeps the same external behavior: same routes, same job lifecycle, same artifact outputs.

## Acceptance Criteria

- [ ] Zip, tarball, and script generation logic lives in `internal/artifacts`.
- [ ] Progress tracking and artifact cleanup helpers move with the artifact generation code.
- [ ] Existing archive and script outputs remain unchanged.

## QA

1. Run archive and script focused tests and confirm they still pass.
2. Run API or end-to-end flows for zip, tarball, and script jobs.
3. Confirm artifact-building code is no longer mixed into the old catch-all package.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
