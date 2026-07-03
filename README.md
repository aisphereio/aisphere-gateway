# Aisphere Gateway

Aisphere Gateway 是基于 `github.com/aisphereio/kernel` 的边界网关服务。它读取 route registry，将外部 HTTP 请求分发到后端业务服务（IAM、Hub 等），并处理边界准入（PUBLIC/INTERNAL/AUTHENTICATED）。

## 架构

```text
外部请求
  -> Gateway HTTP server
  -> route matcher (读取 route registry)
  -> 边界准入 (PUBLIC / INTERNAL / AUTHENTICATED)
  -> generated upstream invoker (gRPC 转发到后端)
  -> 后端服务
```

Gateway **不做** 资源级最终授权，不手写业务路由表，不直接暴露 INTERNAL 路由。

## 本地开发

```powershell
# 安装工具链（使用本地 kernel 开发时）
make tools-local KERNEL_LOCAL=../kernel

# 生成 API 代码
make api

# 检查 proto
make proto-check

# 运行测试
make test

# 启动服务
make run
```

## 本地运行

```powershell
go run ./cmd/aisphere-gateway -conf ./configs/config.local.yaml
```

默认端口：

- HTTP: `0.0.0.0:18000`
- gRPC admin: `0.0.0.0:19000`
- Metrics: `127.0.0.1:19100`

## Layout

```text
cmd/aisphere-gateway/    Application entrypoint
configs/                 Local config files
internal/conf/           Config DTOs scanned by configx
internal/data/           Kernel resource initialization (DB, Cache, Authn, Authz)
internal/dispatch/       JSON body invoker + IAM message factory
internal/registry/       Route registry client (etcd)
internal/server/         Kernel HTTP and gRPC server construction
internal/service/        Gateway admin service (route snapshot, reload, health, version)
```

## 当前技术债

当前 `main.go` 编译期绑定了 IAM 的 gRPC invoker 和 message factory。这是过渡实现，后续 Kernel 的 `protoc-gen-go-gateway` 补齐后，Gateway 只注册 generated modules，不再关心具体业务 RPC。

详见 `docs/kernel-compliance.md` 和 `docs/dispatch-limitations.md`。

## 验证

```bash
curl http://127.0.0.1:18000/healthz
curl http://127.0.0.1:18000/readyz
curl http://127.0.0.1:18000/v1/gateway/routes
```

## 依赖

- `github.com/aisphereio/kernel` — 核心框架
- `github.com/aisphereio/aisphere-iam` — IAM 服务（编译期绑定 gRPC invoker）
- etcd — route registry 存储