# Gateway 本地运行指南

## 目录布局

推荐把三个仓库放在同一层：

```text
aisphereio/
  kernel/
  aisphere-iam/
  aisphere-gateway/
```

使用 `go.work`：

```powershell
cd aisphereio
go work init .\kernel .\aisphere-iam .\aisphere-gateway
```

## 生成代码

如果你正在联调 Kernel generator：

```powershell
cd aisphere-gateway
make tools-local KERNEL_LOCAL=..\kernel
make api
make proto-check
```

如果只使用已发布 Kernel：

```powershell
make tools
make api
make proto-check
```

## 启动顺序

1. 启动 etcd。
2. 启动 IAM，让 IAM 发布 Gateway Manifest。
3. 启动 Gateway，让 Gateway 从 route registry 加载路由。

```powershell
cd aisphere-iam
make run

cd ..\aisphere-gateway
make run
```

默认 route registry：

```text
provider: etcd
prefix: /aisphere/kernel/routes/dev
```

## 默认端口

| 端口 | 用途 |
|---:|---|
| `18000` | Gateway HTTP |
| `19000` | Gateway gRPC admin |
| `19100` | Prometheus metrics |

## 验证

```powershell
curl "http://127.0.0.1:18000/healthz"
curl "http://127.0.0.1:18000/readyz"
curl "http://127.0.0.1:18000/v1/gateway/routes"
curl "http://127.0.0.1:18000/v1/iam/login-url?redirect_uri=http://localhost:3000/callback&state=dev&scope=read"
```

## 当前过渡限制

Gateway 目前编译期绑定了 IAM generated gRPC invoker。后续新增 Hub/Runtime/Skill 服务时，不应继续在 Gateway `main.go` 手写 operation map；应该把 module/factory 注册能力推进到 Kernel generator。
