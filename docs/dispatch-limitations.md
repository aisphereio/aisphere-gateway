# Gateway Dispatch 当前边界

Gateway 当前已经使用 Kernel `gatewayx` 的 route registry、matcher、dispatcher 和 generated IAM gRPC invoker，但仍有过渡性限制。

## 当前状态

- `cmd/aisphere-gateway/main.go` 编译期引入 IAM API，用于注册 IAM generated invoker。
- `internal/dispatch/JSONBodyInvoker` 仍维护 IAM request message factory 和少量 query binding。
- Gateway 只做边界路由、`INTERNAL` 屏蔽、Authorization header 转发和上游分发。
- 资源级授权必须由 IAM/Hub/Runtime 等后端服务自己的 Kernel access middleware 执行。

## 禁止继续扩散

新增业务服务时，不要复制 IAM 的 operation map，也不要在 Gateway 中继续手写 request type switch。

正确方向：

```text
proto contract
  -> protoc-gen-go-gateway
  -> generated manifest + invoker + binder
  -> gatewayx registry/dispatcher
```

如果生成器不能表达新服务的 binding，先修 Kernel generator，再接入业务服务。

## 本地多仓联调

推荐使用 `go.work`：

```powershell
cd E:\coding\aisphereio
go work init .\kernel .\aisphere-iam .\aisphere-gateway
go work use .\kernel .\aisphere-iam .\aisphere-gateway
```

`replace github.com/aisphereio/aisphere-iam => ../aisphere-iam` 只适合本地验证，不应作为发布分支的长期依赖形态。
