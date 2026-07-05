# Gateway AuthN Full Flow

## Goal

Gateway is the external authentication boundary. It verifies Casdoor-issued JWTs locally and forwards only verified requests to gRPC backends.

```text
Client -> Gateway -> gRPC backend
```

## Runtime path

1. Client sends `Authorization: Bearer <casdoor_access_token>`.
2. Gateway reads configured OIDC issuer/discovery/JWKS.
3. Gateway verifies JWT signature with JWKS public key.
4. Gateway validates `iss`, `aud`, `exp`, `nbf`, `iat`, `alg`, and Casdoor `owner`.
5. Gateway strips inbound `X-Aisphere-*` headers to prevent spoofing.
6. Gateway injects trusted identity headers and forwards to the backend gRPC invoker.
7. Backend can either trust the injected Principal or re-verify the same Bearer token.

AuthZ can be disabled or short-circuited during this test; this document only validates authn.

## Cache behavior

Gateway can use Redis through Kernel `cachex` for token verification-result cache. It stores:

```text
key   = sha256(raw_token)
value = normalized Principal
ttl   = min(configured cache_ttl, token_exp-now)
```

Gateway does **not** cache private keys or client secrets. JWKS itself is public-key material and is also cached in-process by Kernel.

## Config

```yaml
data:
  cache:
    enabled: true
    config:
      driver: redis
      addrs: [redis:6379]
      key_prefix: gateway

security:
  authn:
    enabled: true
    provider: casdoor
    cache_ttl_ns: 300000000000
    oidc:
      provider: casdoor
      issuer: https://casdoor.example.com
      discovery_url: https://casdoor.example.com/.well-known/openid-configuration
      jwks_url: https://casdoor.example.com/.well-known/jwks
      audience: [aisphere-web]
      allowed_owners: [aisphere]
      allowed_algs: [RS256]
      jwks_cache_ttl_ns: 600000000000
      clock_skew_ns: 60000000000
```
