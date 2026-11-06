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

Administrative verification requires the `sub`, `iss` and `aud` claims to be
present. Administrator roles, scopes and expected issuer/audience values are not
enforced. The inherited validator rejects expired/not-yet-valid claims when provided, but does not require an `exp` claim.
Presence of `iss`/`aud` does not verify their values against this application's identity.

Consequently, a successfully verified token with these claims grants access to all
enabled admin operations, including impersonation. Trust keys/issuance policy suitable
for that authority. Do not point this admin verifier at broadly shared signing keys
and assume unrelated service or ordinary-user tokens are excluded by their audience
or role. Restrict token issuance and use short-lived administrative credentials. Deployments needing per-role, issuer/audience or scope enforcement
require additional token-policy controls.

The existing JWKS fetcher supports system-trusted HTTPS and internal HTTP; redirects
and normal environment proxy handling retain their existing behavior. Protect the
configured key-distribution path. Use HTTPS with the system CA trust for a remote
issuer, or an explicitly trusted VM-local/internal path under the deployment's trust
model. Proxy, TLS and WireGuard configuration remain separate deployment controls.

## Upgrade and client guidance

Previously `/admin` was outside the JWT middleware even when JWKS was configured;
JWKS only enabled a separate empty `/secure` router. Anonymous admin requests now
fail deliberately. Existing routes, payloads and successful handler responses are
unchanged, so clients must add a bearer header rather than move to `/secure/admin`.

Before deploying, obtain an administrative JWT from the site's trusted issuer and
configure its verification keys. Use the existing ochami token configuration for
token-aware cloud-init admin commands, and qualify the specific commands used at the
site, using the token option supported by the installed CLI version. Include both
a read and a disposable write in that qualification.
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

## Upgrade qualification

Set `CLOUD_INIT_URL` to the direct backend or ingress base URL, including any proxy
prefix. With a working verifier, these probes should each print `401`:

```bash
curl --silent --show-error --output /dev/null --write-out '%{http_code}\n' \
  "${CLOUD_INIT_URL%/}/admin/groups"
curl --silent --show-error --output /dev/null --write-out '%{http_code}\n' \
  -H 'Authorization: Bearer invalid' "${CLOUD_INIT_URL%/}/admin/groups"
```

After obtaining a token from the trusted administrative issuer, check the read:

```bash
curl --fail-with-body -H "Authorization: Bearer ${CLOUD_INIT_ADMIN_TOKEN}" \
  "${CLOUD_INIT_URL%/}/admin/groups"
```

In a disposable instance, exercise a representative write with that token and
confirm the resulting configuration. Repeat without a token and with an invalid
token: both must return 401 without changing stored state. When impersonation or
the SMD simulator is enabled, repeat the checks on those admin routes. In a separate
instance with missing or unreachable JWKS at startup, admin requests must return
503; correct configuration and restart before expecting successful administration.

Run these checks directly and through the site's proxy. Confirm actual node boot
still fetches its expected metadata and configuration without an operator JWT;
HTTP reachability alone does not establish successful bootstrap. Preserve the
site's existing WireGuard enrollment and phone-home checks. These checks do not
establish issuer, audience or role isolation, which this token profile does not
enforce.
