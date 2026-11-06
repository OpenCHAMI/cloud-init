<!-- SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC -->
<!-- SPDX-License-Identifier: MIT -->

# JWT authentication for administrative APIs

Every registered route under `/admin/` requires an `Authorization: Bearer <JWT>`
header. This includes cluster defaults, instance overrides, group reads/writes,
enabled impersonation routes and enabled fake-SMD administration. No admin route
is mounted anonymously. Cookie and query-string tokens are not accepted by this API.

Configure the inbound verification keys using the existing `--jwks-url` option or
`JWKS_URL` environment variable. `TOKEN_URL` and `ACCESS_TOKEN` configure outbound
SMD access; they do not configure authentication of incoming administrative requests.

| Condition | Administrative response |
| --- | --- |
| JWKS is unset or initial loading fails, including an empty key set | 503 Service Unavailable; no admin handler executes |
| Verifier is available but the bearer JWT is missing, malformed, invalidly signed, expired, not yet valid or missing a required claim | 401 Unauthorized |
| JWT passes the existing signature/time/required-claim checks | Existing admin operation runs with its previous response/status behavior |

JWKS loading is attempted once during startup with a ten-second HTTP client timeout.
Failure does not turn on anonymous administration or stop node bootstrap routes from
being registered. Correct the JWKS configuration/trust and restart the service to
enable administration after initialization failure. The loaded keys are a snapshot;
coordinate key overlap and restart the service when rotating verification keys.

## Token profile and trust boundary

This change applies the service's existing JWT verification and requires the `sub`,
`iss` and `aud` claims to be present. It does **not** introduce administrator roles,
scopes or configured expected issuer/audience values. The inherited validator rejects
expired/not-yet-valid claims when provided, but does not require an `exp` claim.
Presence of `iss`/`aud` does not verify their values against this application's identity.

Consequently, a successfully verified token with these claims grants access to all
enabled admin operations, including impersonation. Trust keys/issuance policy suitable
for that authority. Do not point this admin verifier at broadly shared signing keys
and assume unrelated service or ordinary-user tokens are excluded by their audience
or role. Restrict token issuance and use short-lived administrative credentials. A
separate reviewed token-policy change is needed for per-role, issuer/audience or scope
enforcement; this patch must not be described as implementing those controls or the
whole cross-service authorization model.

The existing JWKS fetcher supports system-trusted HTTPS and internal HTTP; redirects
and normal environment proxy handling retain their existing behavior. Protect the
configured key-distribution path. Use HTTPS with the system CA trust for a remote
issuer, or an explicitly trusted VM-local/internal path under the deployment's trust
model. This change does not establish a new proxy, TLS or WireGuard trust policy.

## Upgrade and client guidance

Previously `/admin` was outside the JWT middleware even when JWKS was configured;
JWKS only enabled a separate empty `/secure` router. Anonymous admin requests now
fail deliberately. Existing routes, payloads and successful handler responses are
unchanged, so clients must add a bearer header rather than move to `/secure/admin`.

Before deploying, obtain an administrative JWT from the site's trusted issuer and
configure its verification keys. Use the existing ochami token configuration for
token-aware cloud-init admin commands, and qualify the specific commands used at the
site. This patch does not independently qualify a released ochami binary end to end.
For a direct local development request with a provisioned token in the environment:

```bash
JWKS_URL=https://issuer.example/jwks.json cloud-init-server --cluster-name example
curl --fail-with-body -H "Authorization: Bearer ${CLOUD_INIT_ADMIN_TOKEN}" \
  http://127.0.0.1:27777/admin/groups
```

Use the site's HTTPS ingress for requests outside the trusted VM. A reverse proxy may
add a path prefix, but must preserve the bearer header, method and body. Do not put
tokens in URLs. Keep credentials out of shell history and retained diagnostics.

Node `/user-data`, `/meta-data`, `/vendor-data`, `/{group}.yaml`, `/wg-init` and
`/phone-home/{id}` remain outside the admin JWT middleware. Their existing WireGuard,
source-identity, payload validation and callback behavior are unchanged; this does
not mean every request to those routes succeeds or that their independent trust
issues are resolved. No new operator token is needed for node bootstrap.

For upgrade qualification, check anonymous/invalid admin requests return 401, valid
administrative reads and a disposable write succeed, impersonation remains gated,
and actual node bootstrap still works. If JWKS is unavailable, expect 503 for admin
and normal existing bootstrap behavior. The release endpoint validator can use a
reviewed site policy with `jwt: required` for a selected admin GET. It does not test
mutations, token roles or WireGuard enrollment. Maintain a separate bootstrap check.
