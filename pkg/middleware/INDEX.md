# pkg/middleware — 通用 Gin 中间件

## 包名

`middleware`

## 文件

| 文件 | 内容 |
| --- | --- |
| `request_id.go` | RequestID 中间件 |
| `access_log.go` | 访问日志中间件 |

## 公开函数

### RequestID

```go
func RequestID() gin.HandlerFunc
```

为每个请求生成唯一的 `X-Request-Id`。若请求已携带该 header，则复用。同时将 request_id 注入 `context.Context` 和 logger context。

```go
func RequestIDFromContext(ctx context.Context) string
```

从 context 中读取 request_id。

**常量：**

| 常量 | 值 |
| --- | --- |
| `HeaderRequestID` | `"X-Request-Id"` |

### AccessLog

```go
func AccessLog() gin.HandlerFunc
```

记录每个 HTTP 请求的 method、path、status、latency_ms、client_ip。

## 不包含

> JWT 认证中间件位于 `domain/auth/middleware.go`（`JWTMiddleware`），不在本包。

## 依赖

| 依赖 | 用途 |
| --- | --- |
| `pkg/logger` | 日志输出 |
| `gin` | 中间件框架 |
