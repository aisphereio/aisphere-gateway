# Gateway AuthN Auto Wiring

Gateway is the only public AuthN boundary.

```text
Client Casdoor JWT
  -> Gateway OIDC/JWKS verifier
  -> strip spoofable X-Aisphere-* headers
  -> inject trusted Principal headers
  -> inject X-Aisphere-Internal-Token
  -> dispatch to gRPC upstream
```

## Config

```yaml
security:
  authn:
    enabled: true
    mode: casdoor_jwt
    provider: casdoor
    oidc:
      issuer: http://casdoor.example.com
      discovery_url: http://casdoor.example.com/.well-known/openid-configuration
      jwks_url: http://casdoor.example.com/.well-known/jwks
      audience: [hub-web]
      allowed_owners: [aisphere]
      allowed_algs: [RS256]
  internal_call:
    enabled: true
    header: X-Aisphere-Internal-Token
    token: "${GATEWAY_TO_BACKEND_INTERNAL_TOKEN}"
```

`internal/data.NewResources` uses `securityx.NewAuthnBoundaryRuntime` to build
the verifier. `gatewayx.Dispatcher` then automatically performs strip/inject; no
route handler should parse or inject identity headers manually.
