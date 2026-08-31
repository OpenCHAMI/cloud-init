<!--
SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC

SPDX-License-Identifier: MIT
-->

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Fixed
- Fixed race condition in PeerRemovalQueue where stale removal could happen after a node reconnects. Queue now stores peer key and validates before removal. Added tests for queue behavior.
- Added timeout to SMD token refresh to avoid deadlocks; introduced context with 10s timeout and refreshed token logic. Added benchmark for token refresh.
- Filtered component fetch endpoint to only return Node components, fixing component information cache miss. Added test for component cache behavior.

### Changed
- Updated StopCacheRefresh to safely close channel using safeClose pattern.

## [1.4.10] - 2026-08-25

### Added

- Added 10‑second timeout for SMD token refresh and benchmark (PR #130).
- Fixed race condition in PeerRemovalQueue (PR #132).
- Filtered component fetch to Node components and safe channel close (PR #131).

### Changed

- Updated StopCacheRefresh to safely close channel.
- Various dependency updates and build improvements.

## [1.4.9] - 2026-08-25

### Fixed

- Addressing a race condition in the Wireguard Tunnel manager (PR #127).

## [1.4.8] - 2026-08-10

### Refactor

- Normalize Go module paths (refactor! #126).

## [1.4.7] - 2026-08-03

### Features

- Refactor SMD cache (PR #123).
- Replace if‑else with switch for URL path handling in performance tests.
- Enhance SMD client with membership listing and improve tests for node population.

## [1.4.6] - 2026-08-03

### Fixed

- Avoid blocking SMD cache during update (PR #122).
- Dependency updates: golang.org/x/crypto to 0.52.0, golang.org/x/net to 0.55.0.

## [1.4.5] - 2026-07-09

### Fixed

- Implement retry logic for ComponentInformation to handle transient errors (PR #118).

## [1.4.4] - 2026-07-08

### Fixed

- Change mutex usage from read to write lock in MemStore methods (PR #116).
- Remove logic error that applies Wireguard middleware globally (PR #113).
- Dependency bump for go‑chi/chi/v5 to 5.2.4 (PR #111).
- Improve WireGuard middleware to enforce policies based on client IP and interface (PR #110).

## [1.4.3] - 2026-07-08

### Fixed

- Exclusive map access fixes (PR #117).
- Suppress errcheck lint warning in TestConcurrentInstanceAccess.
- Change mutex usage from read to write lock in RemoveGroupData and DeleteInstanceInfo methods.
- Add concurrent access tests for instance information in MemStore.

## [1.4.2] - 2026-06-11

### Changed

- No changes since v1.4.1.

## [1.4.1] - 2026-06-11

### Added

- Enhance SMDClient with group membership caching and reverse indexing (PR #109).
- Add basic mem store boot‑time initialization (PR #103).
- Add Delve Dockerfile for debugging (PR #95).

## [1.4.0] - 2025-10-03

### Added

- Introduced WireGuard support with client management.
- Added cache refresh mechanism in SMDClient.
- Added version endpoint.

### Fixed

- Fixed race conditions in WireGuard tunnel manager.

## [1.3.0] - 2025-10-02

### Added

- Enhanced SMD client with group membership caching and reverse indexing.
- Added basic mem store boot‑time initialization.

## [1.2.0] - 2025-04-02

### Added

- Implemented cache refresh mechanism in SMDClient.
- Added WireGuard IP management methods.

## [1.1.0] - 2025-02-26

### Added

- Added version endpoint and version information handler.
- Refactored cloud‑init‑server command with cobra and viper.

## [1.0.0] - 2025-01-15

### Added

- Initial release of v1.0 series with core functionality.

## [0.1.1] - 2024-07-19

### Added

- Supports `/cloud-init[-secure]/{user,meta,vendor}-data` endpoints, which auto-detect the querying node's IP address and look up the corresponding xname in SMD
  - This is in contrast to the existing MAC-based endpoints, which remain functional

## [0.1.0] - 2024-07-17

### Added

- Added an additional URL endpoint (`/cloud-init-secure`) which requires JWT authentication for access
  - At the Docker level, if the `JWKS_URL` env var is set, this server will attempt to load the corresponding JSON Web Key Set at startup.
    If this succeeds, the secure route will be enabled, with tokens validated against the JWKS keyset.
- During a query, if no xnames are found for the given input name (usually a MAC address), the input name is used directly.
  This enables users to query an xname (i.e. without needing to look up its MAC first and query using that), or a group name.

### Changed

- Switched from [Gin](https://github.com/gin-gonic/gin) HTTP router to [Chi](https://github.com/go-chi/chi)
- When adding entries to the internal datastore, names are no longer "slug-ified" (via the `gosimple/slug` package).
  This means that when a user requests data for a node, the name they query should be a standard colon-separated MAC address, as opposed to using dashes.
- Rather than requiring a single static JWT on launch, we now accept an OIDC token endpoint. New JWTs are requested from the endpoint as necessary, allowing us to run for longer than the lifetime of a single token.

## [0.0.4] - 2024-01-17

### Added

- Initial release
- Created SMD client
- Added memory-based store
- Able to provide cloud-init payloads that work with newly booted nodes
- Build and release with goreleaser
