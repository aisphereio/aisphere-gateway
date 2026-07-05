# Gateway AuthN: Casdoor JWT + JWKS

Gateway is the external authentication boundary.

## Runtime flow

```text
1. Client sends Authorization: Bearer <casdoor_access_token> or aisphere_access_token cookie.
2. Gateway matches the route.
3. If the route requires authn, Gateway verifies the token locally.
4. Verification uses configured Casdoor discovery/JWKS public keys.
5. Gateway validates iss/aud/exp/owner/alg.
6. Gateway strips all inbound X-Aisphere-* headers.
7. Gateway injects verified Principal headers.
8. Gateway dispatches to the upstream service.
```

## Why Gateway verifies locally

JWT is designed for local resource-server verification. Calling IAM on every request would make IAM a global authn bottleneck and defeat the purpose of signed JWTs. IAM remains responsible for user/group management, authz control plane and Casdoor M2M calls, not every-request token verification.

## Security rules

- `discovery_url` / `jwks_url` must be configured, not taken from the token.
- JWKS is public key material, not a secret.
- Only configured `issuer` and `audience` are accepted.
- Client-supplied `X-Aisphere-*` headers are always removed before trusted headers are injected.
- Internal services must not be reachable directly from the public network.
