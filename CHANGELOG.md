# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2024-07-09

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

## [0.0.4] - 2024-01-17

### Added

- Initial release
- Created SMD client
- Added memory-based store
- Able to provide cloud-init payloads that work with newly booted nodes
- Build and release with goreleaser
