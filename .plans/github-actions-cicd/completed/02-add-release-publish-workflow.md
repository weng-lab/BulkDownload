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

**Built:** Added `.github/workflows/release.yml` to run on published GitHub Releases, rerun `go test ./...` and `go build .`, then build and push the Docker image to Docker Hub as both the release tag and `latest`.

**Decisions:** Reused the same Go setup and validation commands as the CI workflow for consistency, used `docker/metadata-action` to derive exact tags from `github.event.release.tag_name`, and used the conventional `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` GitHub Actions secrets for Docker Hub authentication.

**Deviations:** No functional deviations from the slice plan. Local QA also included a `docker build --platform linux/amd64` run to verify the existing Dockerfile still builds successfully before the GitHub-hosted publish path is exercised.

**Files:** Added `.github/workflows/release.yml`; updated and moved this slice record to `.plans/github-actions-cicd/completed/02-add-release-publish-workflow.md`.

**Notes for next slice:** Plan work is complete. Before the first release publish, set the `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` repository secrets in GitHub so the workflow can authenticate and push `jaiir320/bulkdownload`.
