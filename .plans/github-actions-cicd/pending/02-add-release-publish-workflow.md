# Slice 2: Add release publish workflow

## Dependencies

Slice 1: Add baseline CI workflow.

## Description

Add a GitHub Actions release workflow that runs when a GitHub Release is published from the UI, reruns the required validation, then builds the Docker image from the existing Dockerfile for `linux/amd64` and pushes it to Docker Hub as both the semantic version tag and `latest`. This delivers the full end-to-end shipping path without introducing prerelease handling or additional artifact types.

## Expected Behaviors Addressed

- When I publish a GitHub Release like `v0.3.0`, GitHub Actions reruns validation, builds the Docker image from the repo's Dockerfile, and pushes `jaiir320/bulkdownload:v0.3.0` and `jaiir320/bulkdownload:latest`.
- If tests or build validation fail during release, the image is not published.
- The release process stays centered on GitHub's Releases UI rather than requiring a manual tag-push workflow.
- The Docker image remains the only release artifact that matters for running this service.

## Acceptance Criteria

- [ ] A GitHub Actions workflow exists for published GitHub Releases.
- [ ] The workflow reruns the required validation before any Docker publish step runs.
- [ ] The workflow logs into Docker Hub using GitHub Actions secrets.
- [ ] The workflow builds the existing Dockerfile for `linux/amd64` and pushes `jaiir320/bulkdownload:vX.Y.Z` and `jaiir320/bulkdownload:latest`.
- [ ] If validation fails, the Docker publish steps do not run.

## QA

1. Review the release workflow file and confirm it triggers on published GitHub Releases.
2. Review the workflow steps and confirm validation runs before Docker login and image push steps.
3. Confirm the workflow uses Docker Hub credentials from GitHub Actions secrets rather than hardcoded values.
4. Publish a GitHub Release such as `v0.3.0` and confirm the workflow starts automatically.
5. Confirm Docker Hub receives both `jaiir320/bulkdownload:v0.3.0` and `jaiir320/bulkdownload:latest`.
6. Intentionally break validation in a temporary test scenario and confirm the release workflow fails without publishing an image.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
