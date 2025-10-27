# CI/CD Documentation

This document describes the CI/CD workflows configured for DMR-Nexus.

## Overview

The project uses several GitHub Actions workflows to ensure code quality, security, and automated releases.

## Workflows

### 1. CI (`ci.yml`)
**Trigger:** Pull requests and pushes to `main`

Runs tests and linting to ensure code quality:
- Go tests with race detection
- golangci-lint for code quality checks

### 2. Release Please (`release-please.yml`)
**Trigger:** Pushes to `main`

Automatically manages releases and changelogs:
- Creates/updates a release PR when changes are pushed to main
- Follows [Conventional Commits](https://www.conventionalcommits.org/) to determine version bumps
- Generates `CHANGELOG.md` automatically
- When the release PR is merged, creates a GitHub release
- Creates semantic version tags (e.g., `v0.3.0`) without package name prefix

**Current Version:** 0.3.0

### 3. GoReleaser (`goreleaser.yml`)
**Trigger:** When a release is published

Builds multi-platform binaries:
- **Platforms:** Linux, macOS, Windows, FreeBSD
- **Architectures:** amd64, arm64, arm (v6, v7)
- Automatically embeds frontend assets
- Creates archives with LICENSE, README, and sample config
- Uploads artifacts to GitHub Releases

### 4. Pre-release (`prerelease.yml`)
**Trigger:** After successful CI runs on `main`

Creates automatic pre-releases for development:
- Triggers after CI workflow completes successfully
- Creates a timestamped pre-release tag
- Builds and uploads binaries via GoReleaser
- Marks releases as pre-release (unstable)

### 5. Go Vulnerability Check (`govulncheck.yml`)
**Trigger:** Daily at 08:00 UTC, on PRs, and manual

Scans for Go security vulnerabilities:
- Uses `govulncheck` to find known vulnerabilities
- Fails if vulnerabilities are found
- Can be run manually via workflow dispatch

### 6. Frontend Dependency Audit (`frontend-audit.yml`)
**Trigger:** Changes to `frontend/`, daily at 03:00 UTC, and manual

Audits npm dependencies for security issues:
- Runs `npm audit` on frontend dependencies
- Fails on moderate or higher severity vulnerabilities
- Uploads audit report as artifact

## Dependabot

Configured to automatically check for dependency updates:

- **npm (frontend):** Daily checks with grouped minor/patch updates
- **Go modules:** Daily checks with grouped updates
- **GitHub Actions:** Weekly checks

## Release Process

### Creating a Release

1. Make changes following [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` for new features (minor version bump)
   - `fix:` for bug fixes (patch version bump)
   - `feat!:` or `fix!:` for breaking changes (major version bump)

2. Push changes to `main` (or merge a PR)

3. Release Please will automatically:
   - Create/update a release PR with changelog
   - Calculate the next version number

4. Review and merge the release PR

5. GoReleaser will automatically:
   - Build binaries for all platforms
   - Create GitHub release with artifacts

### Pre-releases

Pre-releases are created automatically:
- Every push to `main` that passes CI triggers a pre-release
- Pre-releases include binaries for all platforms
- Tagged using semantic versioning: `vX.Y.Z-pre.YYYYMMDDHHMMSS.SHA`
  - Example: `v0.1.1-pre.20251026184636.e961141`
  - Base version is read from `.release-please-manifest.json` with patch version incremented
  - Includes timestamp and commit SHA for uniqueness
  - Compatible with GoReleaser v2's semantic version requirements
- Marked as "pre-release" on GitHub

## Security Scanning

### Go Vulnerabilities
- Scanned daily via `govulncheck`
- Also runs on every PR

### Frontend Dependencies
- `npm audit` runs daily
- Also runs when frontend files change

### Dependabot
- Monitors for security updates
- Creates PRs for vulnerable dependencies

## Configuration Files

- `.goreleaser.yml` - GoReleaser configuration
- `.release-please-config.json` - Release Please settings
  - `include-component-in-tag: false` ensures tags use semantic version format (e.g., `v0.3.0`) instead of including package name (e.g., `dmr-nexus-v0.3.0`)
- `.release-please-manifest.json` - Current version tracking
- `.github/dependabot.yml` - Dependabot configuration
- `.github/workflows/*.yml` - GitHub Actions workflows

## Manual Triggers

Several workflows can be triggered manually via GitHub UI:
- Go Vulnerability Check
- Frontend Dependency Audit

Navigate to Actions → Select workflow → Run workflow
