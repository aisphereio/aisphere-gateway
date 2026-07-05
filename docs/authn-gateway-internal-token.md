# Gateway AuthN + Internal Service Token

本仓库的 Gateway 是外部认证边界：

1. 从 `Authorization: Bearer` 或 `aisphere_access_token` Cookie 读取 Casdoor JWT。
2. 使用 Kernel `authn/oidcx` 从 `issuer/discovery_url/jwks_url` 拉取 JWKS 公钥。
3. 本地校验 JWT：签名、`iss`、`aud`、`exp`、`nbf`、`iat`、`alg`、`owner`。
4. 删除所有客户端伪造的 `X-Aisphere-*` 和 `X-Aisphere-Internal-Token`。
5. 注入可信 Principal headers。
6. 注入 `X-Aisphere-Internal-Token`，转发到后端 gRPC 服务。

`X-Aisphere-Internal-Token` 只证明“请求来自 Gateway”，不代表用户身份。用户身份来自 Gateway 验证 Casdoor JWT 后注入的 Principal。

## 配置

```yaml
security:
  authn:
    enabled: true
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

第一版使用共享 token 即可；生产建议叠加 Kubernetes NetworkPolicy，限制只有 Gateway Pod 能访问后端服务。后续可以升级到 mTLS/SPIFFE。
