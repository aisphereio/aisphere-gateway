# Gateway Kernel 合规说明

## 目标链路

```text
service proto contract
  -> generated Gateway Manifest
  -> route registry
  -> Gateway route matcher
  -> generated upstream invoker
  -> backend service access middleware
```

## Gateway 应该做什么

- 读取 route registry。
- 匹配 HTTP method/path。
- 根据 GatewayPolicy 做 PUBLIC/INTERNAL/AUTHENTICATED 的边界准入。
- 转发 Authorization、request id、trace headers。
- 调用 generated upstream invoker。

## Gateway 不应该做什么

- 不做资源级最终授权。
- 不手写业务路由表。
- 不把 IAM/Hub/Runtime 的 query/body 绑定逻辑长期维护在 Gateway 业务代码里。
- 不直接暴露 INTERNAL 路由。

## 当前技术债

当前 `main.go` 里存在 IAM operation/message factory 的编译期绑定。这是为了验证 Kernel Gateway generated invoker 的过渡实现。

后续应在 Kernel 中补齐：

```text
protoc-gen-go-gateway
  -> generated Gateway module
  -> generated message factory registry
  -> generated query/path/body binder
```

然后 Gateway 只注册 generated modules，不再关心具体业务 RPC。

## P0 检查清单

- [ ] `go.mod` 只引用完整 GitHub module path。
- [ ] proto `go_package` 是完整 GitHub module path。
- [ ] `make api` 后不存在 undefined generated resolver。
- [ ] Gateway admin API 的 proto policy 能生成 request/access resolver。
- [x] README 写清楚 IAM/Gateway 启动顺序（见 `docs/run-local.md`）。

## 不允许的回退

```text
require aisphere-iam v0.0.0
import "aisphere-iam/..."
手写所有服务的 route manifest
复制 IAM operation map 给其他服务
把 Gateway 当成唯一鉴权防线
```
EOF
