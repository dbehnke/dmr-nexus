Title
Add dashboard enhancements with GORM/SQLite, transmission logging, and embedded frontend

Body

## Summary

This PR implements the dashboard enhancements and related infra requested in the issue. Key goals met:

- Active bridges view (dynamic bridges) and recent transmissions/talk log in the UI.
- Persistent transmission logging using GORM + pure-Go SQLite (modernc.org/sqlite).
- Transmission logging in the router/bridge layer (start, packets, terminator handling, cleanup).
- Version wiring from build-time ldflags -> exposed via /api/status and shown in the frontend footer.
- Optionally embed the built Vue3 frontend into the Go binary using go:embed (Makefile + Docker support).
- Make Docker/compose builds compute or accept git-based version metadata (VERSION, GIT_COMMIT, BUILD_TIME).
- Tests and linter fixes across packages; updated Makefile, scripts, Dockerfile for robust builds.

## What changed (high level)

- Backend
  - Added `pkg/database`:
    - GORM models and repository for `Transmission`.
    - Uses `modernc.org/sqlite` (pure go) dialector to avoid CGO.
    - Migration + repository methods: Create, GetRecent, GetRecentPaginated, GetByRadioID, GetByTalkgroup, DeleteOlderThan.
  - Added `pkg/bridge/transmission_logger.go`:
    - Tracks active streams, accumulates packet counts and timing, saves transmissions on terminator or cleanup.
    - Periodic cleanup (cleanup routine started in main).
  - Router updates:
    - Dynamic bridge behavior and active status, timeslot-agnostic bridge model.
    - Hook to call transmission logger on packets.
  - API (`pkg/web`):
    - `/api/status` includes version, git commit, build time.
    - `/api/transmissions` endpoint with pagination.
    - `/api/bridges` returns dynamic bridges with subscriber info and `active`/`active_radio_id`.
    - `pkg/web/version.go` stores version info and is set from `main` using ldflags.
    - Static file serving tries embedded FS (build tag `embed`) first, then falls back to `frontend/dist` on disk.
  - `cmd/dmr-nexus/main.go`:
    - Wires `web.SetVersionInfo(version, commit, buildTime)` using ldflags.
    - Initializes database, transmission repository, transmission logger and sets router->logger hook.
    - Starts cleanup goroutine for stale streams.

- Frontend
  - `frontend/src/stores/app.js`:
    - New `version` state populated from `/api/status`.
    - Fetch and store transmissions (pagination).
  - `frontend/src/App.vue`:
    - Footer now displays `store.version`.
  - `frontend/src/views/Dashboard.vue`:
    - Active bridges grid (cards, idle/active colors, speaker icon).
    - Recent transmissions table with formatted duration and relative time, auto-refresh every 5s.

- Build & Dev tooling
  - `Makefile`:
    - `build-embed` target builds frontend, copies `frontend/dist` → `pkg/web/frontend/dist` and builds Go binary with `-tags=embed`.
    - `prepare-frontend-embed` target copies frontend artifacts for embedding.
    - `docker`, `compose-build`, and `docker-compose-*` targets accept/propagate VERSION/GIT_COMMIT/BUILD_TIME build args.
  - `Dockerfile`:
    - Multi-stage: `frontend-builder` builds SPA; `backend-builder` copies SPA into build context and runs a helper script to compute version info and build with `-tags=embed`.
    - Diagnostic checks to fail early if `frontend/dist` is missing in the builder stage.
  - Scripts:
    - `scripts/docker-build-embed.sh` — runs inside Docker builder to compute version metadata and execute the embedded `go build`.
    - `scripts/compose-build` — creates a temporary `.env` using git info and runs `docker compose build`.
  - `.dockerignore` / build context:
    - Adjusted so `.git` is available in the build context if you want the builder to compute git metadata. (Alternative approach: keep `.git` excluded and always pass VERSION/GIT_COMMIT/BUILD_TIME from host/CI — scripts already support that.)

- Tests & lint
  - Fixed lint issues (errcheck/unused).
  - Added tests for database, transmission logger, API endpoints and updated existing tests to properly ignore deferred cleanup errors.
  - Ran full test suite locally; tests pass.

## API additions

- GET /api/status
  - Response includes: status, service, version, commit, build_time
- GET /api/transmissions?page=1&per_page=50
  - Returns paginated transmissions and total count.
- GET /api/bridges
  - Returns dynamic bridges with subscriber timeslot info and active state.

## How to build & run locally

- Regular (non-embedded) build:
  - make deps
  - make build
  - The server will serve frontend from `frontend/dist` if present.

- Embedded frontend (single binary):
  - make build-embed
  - Result: `./bin/dmr-nexus` (contains the SPA via go:embed)

- Docker:
  - Build with auto-detected version (requires `.git` in build context) or pass args:
    - ./scripts/build-docker.sh (helper)
    - OR docker build --build-arg VERSION=$(git describe --tags --always --dirty) --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) -t dmr-nexus:local .
  - For compose:
    - scripts/compose-build
    - docker compose up -d

## Notes, rationale and decisions

- SQLite driver:
  - Chose `modernc.org/sqlite` (pure Go) so we can build static binaries without CGO, which simplifies Docker builds and cross-compilation.
- .git in context:
  - To compute build-time git metadata inside the Docker builder, `.git` must be present in the build context. That was made optional but the repository currently contains changes to allow it (or you can use `scripts/compose-build` which creates a temporary `.env` with git metadata and avoids including `.git`).
- Version info:
  - Version/commit/build_time are injected at build time using ldflags (Makefile / Docker build helper).
  - Exposed via `/api/status` and shown in the frontend footer.
- Transmission logging:
  - TransmissionLogger ensures we only persist transmissions that are meaningfully long (>= 0.5s), to reduce noise/duplicates.
  - A periodic cleanup saves stale streams older than a configurable threshold (default: 60s cleanup checks every 30s).
- Single-stream enforcement:
  - When a stream header arrives for a talkgroup and another stream is active, the router will reject the new stream to avoid cross-talk duplicates. This behavior is enforced in the router logic; it is conservative but avoids duplicate audio.

## Files touched (selected)
- cmd/dmr-nexus/main.go — wire version info, DB init, tx logger
- pkg/web/* — static embed support, API changes, version storage
- pkg/database/* — DB driver, model, repository, migrations, tests
- pkg/bridge/* — router updates, transmission logger, dynamic bridge model and tests
- frontend/src/* — footer, store, dashboard UI updates
- Makefile, Dockerfile, scripts/compose-build, scripts/docker-build-embed.sh, docker-compose.yml
- .dockerignore updated (see note above)

## How I validated
- Ran go test ./... locally (all tests pass).
- Confirmed `make build-embed` produces a binary and `./bin/dmr-nexus --version` prints version/commit/build_time when built with ldflags.
- Built Docker image with `scripts/build-docker.sh` and validated the container startup logs contain the embedded git metadata.
- Frontend updated to read version from the API and display it in the footer; dashboard views auto-refresh and show dynamic bridges and transmissions.

## Remaining / follow-ups
- Decide on policy for `.git` in Docker build context vs. always passing build args from CI (scripts already support both).
- Optional: add CI step to run `scripts/compose-build` or set build args in pipeline to ensure images created in CI have correct version metadata.
- Optional: Add rate-limiting or retention policy UI for transmissions (e.g., deletion by age via API).
- Optional: Add more frontend tests (Cypress / Playwright) to cover the dashboard UI interactions.

Closes #20
