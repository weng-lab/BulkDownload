# GitHub Actions CI/CD for BulkDownload

## Problem

I want a professional but simple GitHub Actions setup for this repo so that normal code changes are validated automatically, and publishing a GitHub Release from the UI automatically tests the project and pushes the Docker image to Docker Hub. I do not want a complicated release process or binary packaging workflow, because this service is only meant to run from its Docker image.

## Solution

Add two focused GitHub Actions workflows.

The first workflow handles continuous integration for pull requests and pushes to `main`. It runs the Go validation steps that reflect how this repo is developed, so code quality is checked automatically before release work happens.

The second workflow handles releases. It is triggered by publishing a GitHub Release in the GitHub UI, reruns the required validation, then builds the Docker image from the existing Dockerfile and pushes it to Docker Hub under `jaiir320/bulkdownload`.

This keeps the day-to-day developer flow simple while giving releases a clear, professional path: publish release in GitHub, let automation validate and ship the container.

## Expected Behavior

- When a pull request is opened or updated, GitHub Actions runs CI automatically and reports whether the Go project passes validation.
- When code is pushed to `main`, GitHub Actions reruns the same CI checks so the main branch stays healthy.
- When I publish a normal GitHub Release like `v0.3.0`, GitHub Actions reruns validation, builds the Docker image from the repo's Dockerfile, and pushes `jaiir320/bulkdownload:v0.3.0` and `jaiir320/bulkdownload:latest`.
- When I publish a prerelease, GitHub Actions reruns validation, builds the Docker image, and pushes only the prerelease version tag, without moving `latest`.
- If tests or build validation fail during release, the image is not published.
- The release process stays centered on GitHub's Releases UI rather than requiring a manual tag-push workflow.

## Implementation Decisions

- Use a two-workflow design instead of one all-purpose workflow so CI and release concerns stay easy to understand and maintain.
- Treat GitHub Releases as the source of truth for shipping, rather than git tag pushes as the primary release trigger.
- Keep CI focused on the repo's real validation points: Go tests and successful buildability of the application.
- Use the existing Dockerfile as the release artifact definition rather than introducing a second packaging path.
- Publish to Docker Hub only, using the existing image name `jaiir320/bulkdownload`, to avoid multi-registry complexity.
- Publish `linux/amd64` images only for now, since this is internal-use and simplicity is preferred over multi-arch support.
- Distinguish stable releases from prereleases so only stable releases update `latest`.
- Store Docker Hub credentials in GitHub Actions secrets and use standard marketplace actions for Docker login, metadata/tag generation, and image builds so the workflow stays conventional and easy to maintain.
- Keep permissions and workflow scope minimal, with only the release workflow receiving what it needs to publish images.

## Testing Approach

- Verify CI by confirming pull request and `main` push events run the expected Go validation steps successfully.
- Validate that the workflow runs the full Go test suite already present in the repo, including package tests and the end-to-end tests that currently exercise the service behavior.
- Confirm the app can still build cleanly in CI so source-level regressions are caught even outside Docker publishing.
- Validate the release workflow with a GitHub prerelease first, ensuring it reruns checks and publishes only the versioned Docker tag.
- Validate a stable release afterward, confirming both the semantic version tag and `latest` appear in Docker Hub.
- Confirm release failure behavior by ensuring no Docker publish step runs when validation fails.
- Check that published image tags line up exactly with the GitHub Release tag to keep release provenance obvious.

## Out of Scope

- Standalone binary artifacts attached to GitHub Releases.
- Multi-architecture image publishing such as `arm64`.
- Publishing to GHCR or any registry other than Docker Hub.
- Automated semantic version calculation, changelog generation, or release note generation.
- Deployment automation beyond building and publishing the Docker image.
- Advanced CI expansion such as lint pipelines, code coverage reporting, or matrix testing across multiple Go versions unless added later.
