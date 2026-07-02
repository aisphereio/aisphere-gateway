# Aisphere Gateway Agent 规范

本仓库是 Kernel Gateway 业务仓库，不是 Kernel layout 模板仓库。AI Agent 和人类开发者必须遵守以下约束。

## 1. 模块路径

- `go.mod` 必须使用 `module github.com/aisphereio/aisphere-gateway`。
- 引用 IAM 必须使用 `github.com/aisphereio/aisphere-iam/api/...`。
- 本地联调优先使用 `go.work`；`replace github.com/aisphereio/aisphere-iam => ../aisphere-iam` 只允许作为临时本地方案。
- 禁止重新引入 `aisphere-iam` 短模块路径。

## 2. Gateway 路由来源

- Gateway 不允许长期手写业务路由表。
- 业务服务必须通过 proto 的 `google.api.http` 和 `aisphere.access.v1.policy` 生成 Gateway Manifest。
- Gateway 只能消费 `gatewayx.RouteRegistry` 中的 Manifest，不应在主逻辑中维护业务 HTTP path 清单。

## 3. 当前 IAM 编译期绑定的处理规则

当前版本为了验证 generated gRPC invoker，Gateway 编译期引入 IAM API。这个绑定是过渡方案。

新增服务时不要复制 `main.go` 里的 IAM operation map。必须先检查 `docs/dispatch-limitations.md`，应优先扩展 Kernel `protoc-gen-go-gateway`，生成统一的 Gateway module/factory 注册入口。

## 4. 边界职责

- Gateway 负责 route match、PUBLIC/INTERNAL 边界策略、Authorization header 转发和上游分发。
- Gateway 不负责最终资源级授权。
- IAM/Hub/Runtime 等后端服务必须自己接入 Kernel access middleware。
- `INTERNAL` 路由默认不对公网暴露。

## 5. 生成代码

- 修改 `api/gateway/v1/gateway.proto` 后必须运行 `make api && make proto-check`。
- 如果同时修改 Kernel generator，先应用 Kernel 补丁，再运行：

```powershell
make tools-local KERNEL_LOCAL=../kernel
make api
make proto-check
make test
```

## 6. 文档门禁

以下变化必须同步 README 或 `docs/*.md`：route registry、upstream target、启动顺序、编译期绑定服务列表、Kernel generator 使用方式。

## 7. 提交前检查

```powershell
make tools-local KERNEL_LOCAL=../kernel
make api
make proto-check
make test
make build
```
