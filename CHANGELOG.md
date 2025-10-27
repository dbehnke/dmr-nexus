# Changelog

## [0.5.0](https://github.com/dbehnke/dmr-nexus/compare/v0.4.0...v0.5.0) (2025-10-27)


### Features

* **frontend:** Replace Tailwind CSS v4 with Quasar Framework + Enable SQLite WAL Mode and WebSocket Push Updates ([#64](https://github.com/dbehnke/dmr-nexus/issues/64)) ([cad3b51](https://github.com/dbehnke/dmr-nexus/commit/cad3b511176f084f3b02b4980bdad742109a7487))

## [0.4.0](https://github.com/dbehnke/dmr-nexus/compare/v0.3.0...v0.4.0) (2025-10-27)


### Features

* Add CI/CD workflows: release-please, goreleaser, govulncheck, dependabot ([#38](https://github.com/dbehnke/dmr-nexus/issues/38)) ([a132793](https://github.com/dbehnke/dmr-nexus/commit/a1327937c7c4dc3b5623743a34dd983b1492c22f))
* Add Private Call (Unit-to-Unit) Routing Support ([#26](https://github.com/dbehnke/dmr-nexus/issues/26)) ([99e4599](https://github.com/dbehnke/dmr-nexus/commit/99e4599a74c7a27787b08cff835f35edbcdf27f3))
* **frontend:** add minimal Vite+Vue scaffold so make frontend can build ([cd60baf](https://github.com/dbehnke/dmr-nexus/commit/cd60baf8944c8dd9cca17c9383dbaf530813ad99))
* **frontend:** Tailwind + Pinia + WS client; feat(web): serve heartbeat WS events and upgrade /ws ([7c503ab](https://github.com/dbehnke/dmr-nexus/commit/7c503ab9460c78967adff295f6f2374e2a3c03ea))
* **frontend:** wire existing src into a minimal SPA (router, views, API calls) ([2704e1e](https://github.com/dbehnke/dmr-nexus/commit/2704e1eeb5a88f37db2fc063a62f523664af2122))
* Implement Phase 5 - Web Dashboard Backend ([3f1b38d](https://github.com/dbehnke/dmr-nexus/commit/3f1b38dda56679af13a9ecb4c57c7ea2e3a58884))
* **network:** add RPTO (OPTIONS) packet handler ([2f9367f](https://github.com/dbehnke/dmr-nexus/commit/2f9367f2a10328761886459ff891a4ad3b2f5069))


### Bug Fixes

* code formatting ([f8c47bd](https://github.com/dbehnke/dmr-nexus/commit/f8c47bda43049afa9c20283b07cd0e0de2026140))
* golanglint-ci run fixes ([6e9a0e9](https://github.com/dbehnke/dmr-nexus/commit/6e9a0e92f49cb968bdbced0c739c786f517a457f))
* golanglint-ci run fixes ([2ae5b2f](https://github.com/dbehnke/dmr-nexus/commit/2ae5b2fa2f1d5c21c919dfab038364cbcef43fcc))
* golanglint-ci run fixes ([5758a88](https://github.com/dbehnke/dmr-nexus/commit/5758a8882aba222f158789c22f86f6502761844f))
* ignore dmr-nexus.yaml in root dir ([508c6bf](https://github.com/dbehnke/dmr-nexus/commit/508c6bf7be08c291532d09762d90d1c9b667822e))
* **lint/tests:** resolve golangci-lint failures and test issues ([effd4a3](https://github.com/dbehnke/dmr-nexus/commit/effd4a37d8b86cfb31d11bb23dc998fc6bcbfe22))

## [0.3.0](https://github.com/dbehnke/dmr-nexus/compare/dmr-nexus-v0.2.0...dmr-nexus-v0.3.0) (2025-10-27)


### Features

* Add Private Call (Unit-to-Unit) Routing Support ([#26](https://github.com/dbehnke/dmr-nexus/issues/26)) ([99e4599](https://github.com/dbehnke/dmr-nexus/commit/99e4599a74c7a27787b08cff835f35edbcdf27f3))

## [0.2.0](https://github.com/dbehnke/dmr-nexus/compare/dmr-nexus-v0.1.0...dmr-nexus-v0.2.0) (2025-10-26)


### Features

* Add CI/CD workflows: release-please, goreleaser, govulncheck, dependabot ([#38](https://github.com/dbehnke/dmr-nexus/issues/38)) ([a132793](https://github.com/dbehnke/dmr-nexus/commit/a1327937c7c4dc3b5623743a34dd983b1492c22f))
* **frontend:** add minimal Vite+Vue scaffold so make frontend can build ([cd60baf](https://github.com/dbehnke/dmr-nexus/commit/cd60baf8944c8dd9cca17c9383dbaf530813ad99))
* **frontend:** Tailwind + Pinia + WS client; feat(web): serve heartbeat WS events and upgrade /ws ([7c503ab](https://github.com/dbehnke/dmr-nexus/commit/7c503ab9460c78967adff295f6f2374e2a3c03ea))
* **frontend:** wire existing src into a minimal SPA (router, views, API calls) ([2704e1e](https://github.com/dbehnke/dmr-nexus/commit/2704e1eeb5a88f37db2fc063a62f523664af2122))
* Implement Phase 5 - Web Dashboard Backend ([3f1b38d](https://github.com/dbehnke/dmr-nexus/commit/3f1b38dda56679af13a9ecb4c57c7ea2e3a58884))
* **network:** add RPTO (OPTIONS) packet handler ([2f9367f](https://github.com/dbehnke/dmr-nexus/commit/2f9367f2a10328761886459ff891a4ad3b2f5069))


### Bug Fixes

* code formatting ([f8c47bd](https://github.com/dbehnke/dmr-nexus/commit/f8c47bda43049afa9c20283b07cd0e0de2026140))
* golanglint-ci run fixes ([6e9a0e9](https://github.com/dbehnke/dmr-nexus/commit/6e9a0e92f49cb968bdbced0c739c786f517a457f))
* golanglint-ci run fixes ([2ae5b2f](https://github.com/dbehnke/dmr-nexus/commit/2ae5b2fa2f1d5c21c919dfab038364cbcef43fcc))
* golanglint-ci run fixes ([5758a88](https://github.com/dbehnke/dmr-nexus/commit/5758a8882aba222f158789c22f86f6502761844f))
* ignore dmr-nexus.yaml in root dir ([508c6bf](https://github.com/dbehnke/dmr-nexus/commit/508c6bf7be08c291532d09762d90d1c9b667822e))
* **lint/tests:** resolve golangci-lint failures and test issues ([effd4a3](https://github.com/dbehnke/dmr-nexus/commit/effd4a37d8b86cfb31d11bb23dc998fc6bcbfe22))
