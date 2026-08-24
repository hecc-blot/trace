# hecc-blot-trace

基于 OpenTelemetry 的链路追踪：OTLP 导出 Jaeger，自动关联日志与 HTTP/SSE。

## 安装

```bash
go get github.com/hecc-blot/trace
```

## 配置

```yaml
trace:
  service_name: Hecc-Blot            # 服务名称
  endpoint: 127.0.0.1:4318           # OTLP 接收端点 (HTTP)
  sampler:
    type: always                     # always / never / probability
    ratio: 0.5                       # 采样比例 (probability 模式使用)
```

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `service_name` | string | 服务名称，用于在追踪系统中标识 |
| `endpoint` | string | OTLP HTTP 接收端点地址 |
| `sampler.type` | string | 采样类型：`always` / `never` / `probability` |
| `sampler.ratio` | float | 采样比例，仅 probability 模式生效 (0-1) |

| 采样类型 | 说明 |
|------|------|
| `always` | 采样所有请求，适用于开发环境 |
| `never` | 不采样任何请求 |
| `probability` | 按比例采样，`ratio` 指定采样率 |

## 初始化

```go
import (
    trace "github.com/hecc-blot/trace/service"
)

traceSvc, traceClearUp, err := trace.NewTraceSvc(&config.Trace)
if err != nil {
    panic(err)
}
defer traceClearUp()

container.Set(new(traceContract.ITrace), traceSvc)
```

## 在业务中使用

```go
import traceContract "github.com/hecc-blot/trace/contract"

type YourApi struct {
    TraceSvc traceContract.ITrace `inject:""`
}

func (y YourApi) Call(ctx *gin.Context) (interface{}, error) {
    // 从 Context 获取当前活跃的 Span
    currentSpan := y.TraceSvc.FromContext(ctx)

    // 添加自定义属性
    currentSpan.SetAttribute("user.id", 12345)
    currentSpan.SetAttribute("operation.type", "query")

    // 记录业务错误
    if err != nil {
        currentSpan.RecordError(err)
    }

    // 开启子 Span 追踪子操作
    subCtx, subSpan := y.TraceSvc.Start(ctx, "sub-operation",
        "sub.key", "sub-value",
    )
    defer subSpan.End()

    result := doSomething(subCtx)
    return result, nil
}
```

### Span 操作

| 方法 | 说明 |
|------|------|
| `SetAttribute(key, value)` | 设置 Span 属性，支持 string/int/int64/bool/float64 |
| `RecordError(err)` | 记录错误信息到当前 Span |
| `Name()` | 获取 Span 名称 |
| `End()` | 结束当前 Span |

## 中间件实现自动追踪

框架提供 `HttpTraceMiddleware` 与 `SseTraceMiddleware`，分别通过 `trace.NewHttpMiddleware(traceSvc)` / `trace.NewSseMiddleware(traceSvc)` 在组装阶段显式注册：

```go
apiHandle := httpSvc.NewApiSvc(&config.Server, responseSvc, container)
sseHandle := sse.NewSseSvc(apiHandle.Engine(), container)

apiHandle.Middleware(trace.NewHttpMiddleware(traceSvc))
sseHandle.Middleware(trace.NewSseMiddleware(traceSvc))
```

### 自动行为

`HttpTraceMiddleware` 自动执行：

1. **链路上下文提取**：从请求头 `traceparent` 提取分布式追踪上下文
2. **创建请求 Span**：为每个 HTTP 请求创建 `http.request` Span，路径记录在 `http.url`
3. **响应头注入**：`X-Trace-Id`（当前 Trace ID）与 `traceparent`（W3C Trace Context）

SSE 连接由 `SseTraceMiddleware` 以 `sse.connection` 为 span 名称追踪，行为与 HTTP 中间件一致。

## 日志集成

追踪服务与日志服务深度集成，自动将 TraceId 关联到日志：

```go
y.LogSvc.Info(ctx, "执行查询", "table", "users")
// 输出: {"level":"info","msg":"执行查询","table":"users","traceId":"4bf92f35..."}
```

## 上下文传递

跨 HTTP 服务间通过 `traceparent` 头传递追踪上下文：

```go
// 发送方
req, _ := http.NewRequest("POST", "http://service-b/api", body)
req.Header.Set("traceparent", c.GetHeader("traceparent"))

// 接收方自动通过中间件提取
```

## 相关模块

| 模块 | 说明 |
|------|------|
| [framework](https://github.com/hecc-blot/framework) | 日志 TraceId 关联、`IMiddleware` 接口 |
| [db](https://github.com/hecc-blot/db) | SQL 自动生成 span（依赖全局 TracerProvider） |
| [cache](https://github.com/hecc-blot/cache) | 缓存操作 span |
| [sse](https://github.com/hecc-blot/sse) | SSE 连接 span |
