<!-- SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC -->
<!-- SPDX-License-Identifier: MIT -->

# Dependency security update

This update changes dependency versions without changing cloud-init routes,
configuration flags, token requirements or storage settings. Deploy a rebuilt
service artifact to use the patched dependencies; updating source manifests alone
does not update running installations.

| Dependency | Previous | Updated | Findings addressed |
| --- | --- | --- | --- |
| `github.com/go-jose/go-jose/v4` | 4.1.4 | 4.1.6 | [RSA JWK denial of service](https://github.com/advisories/GHSA-mgcc-cww5-fp2q), [JSON parsing panic](https://github.com/advisories/GHSA-vvm8-gf78-28g4), [CBC-HMAC panic](https://github.com/advisories/GHSA-4w3g-jfj2-cw2q) |
| `github.com/go-chi/chi/v5` | 5.3.1 | 5.4.0 | [ClientIPFromXFF peer validation](https://github.com/advisories/GHSA-j6qf-3m9c-vwj3) |
| `github.com/klauspost/compress` | 1.18.0 | 1.20.2 | [S2 integer overflow](https://github.com/advisories/GHSA-259r-337f-4rfw), plus the [reachable compression panic reported by govulncheck](https://pkg.go.dev/vuln/GO-2026-6865) |
| `github.com/go-viper/mapstructure/v2` | 2.3.0 | 2.4.0 | [Sensitive values in malformed-data errors](https://github.com/advisories/GHSA-2464-8j7c-4cjm) |

These versions address the six open GitHub dependency alerts inspected on
2026-11-06. A dependency alert establishes the presence of an affected version;
it does not establish that every advisory's vulnerable function is reachable in
this service. The initial govulncheck scan found a reachable compression panic,
an imported mapstructure issue and additional module-only advisories.

## Compatibility and qualification

Go JOSE 4.1.6 deliberately caps parsed RSA JWK moduli at 4096 bits by default;
see its [release notes](https://github.com/go-jose/go-jose/releases/tag/v4.1.6).
Here it is an indirect dependency through SMD, HMS certificate/secure-storage
helpers and the Vault API. Sites using that path must qualify their certificate,
Vault authentication and renewal flow, especially if it uses larger RSA keys.
This patch does not relax that upstream security limit. Incoming cloud-init JWT
verification continues to use the separate existing jwtauth/jwx implementation.

Chi 5.4.0 fixes its newer ClientIPFrom* helpers. Cloud-init currently uses the
legacy RealIP middleware, whose code and deployment behavior are unchanged by
this update. Do not treat this dependency bump as a solution to cloud-init's
forwarding-header trust or WireGuard transport-peer policy; those need separate
application/deployment changes and compatibility review.

Compression changes affect the Arrow/DuckDB and Quack-backed data paths. Before
production rollout, qualify a copy of the site's existing persistent store with
the rebuilt artifact and perform normal read/write and restart checks. Preserve
a backup and the previous artifact for rollback. This patch introduces no
intentional schema migration. Malformed configuration errors may omit sensitive
values after the mapstructure update; do not rely on the prior diagnostic text.

Run the repository checks and test the site's actual bootstrap/admin/client flow
before rollout. GitHub closes alerts after it scans the updated default branch;
local passing checks do not close repository alerts or prove deployment.
