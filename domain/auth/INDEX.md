# domain/auth — 认证领域

## 包名

`auth`

## 文件地图

| 文件 | 内容 |
| --- | --- |
| `config.go` | `Config`：认证配置结构（Enabled、Secret、Issuer、Audience、TokenTTL） |
| `principal.go` | `Principal`：认证主体模型 |
| `jwt.go` | JWT 生成、验证、解析（HS256） |
| `service.go` | `Service`：Token 生成/验证服务、`Authenticator` 接口 |
| `http_handler.go` | `HTTPHandler`：Token 签发 API handler |
| `middleware.go` | `JWTMiddleware`：Gin JWT 认证中间件 |

## 公开类型

### 配置

```go
type Config struct {
    Enabled  bool
    Secret   string
    Issuer   string
    Audience string
    TokenTTL time.Duration
}
```

由 `pkg/config.AuthConfig.AuthConfig()` 方法转换生成。

### 领域模型

| 类型 | 文件 | 说明 |
| --- | --- | --- |
| `Principal` | `principal.go` | 认证主体：Subject、Claims (\*JWTClaims)、Extra (map) |
| `JWTClaims` | `jwt.go` | JWT 声明：Subject、Issuer、Audience、ExpiresAt、NotBefore、IssuedAt、Extra |
| `JWTOptions` | `jwt.go` | JWT 验证选项：Secret、Issuer、Audience |

构造函数：`NewPrincipal(claims *JWTClaims) *Principal`

### 错误变量

| 变量 | 说明 |
| --- | --- |
| `ErrMissingToken` | 缺少 Bearer token |
| `ErrInvalidTokenFormat` | Token 格式无效 |
| `ErrUnsupportedAlg` | 不支持的签名算法 |
| `ErrInvalidSignature` | 签名无效 |
| `ErrExpiredToken` | Token 已过期 |
| `ErrTokenNotValidYet` | Token 尚未生效 |
| `ErrInvalidIssuer` | 签发者不匹配 |
| `ErrInvalidAudience` | 受众不匹配 |
| `ErrEmptySigningSecret` | 签名密钥为空 |
| `ErrInvalidTokenPayload` | Token 负载无效 |
| `ErrReservedClaim` | 保留声明不能覆盖 |

### JWT 底层函数

```go
func GenerateJWT(subject string, extra map[string]any, config Config) (string, error)
func GenerateJWTAt(subject string, extra map[string]any, config Config, now time.Time) (string, error)
func ValidateJWT(token string, options JWTOptions) (*JWTClaims, error)
func ValidateJWTAt(token string, options JWTOptions, now time.Time) (*JWTClaims, error)
func ValidateBearerToken(headerValue string, options JWTOptions) (*JWTClaims, error)
func ValidateBearerTokenAt(headerValue string, options JWTOptions, now time.Time) (*JWTClaims, error)
```

算法固定为 HS256。`ValidateBearerToken*` 自动剥离 `"Bearer "` 前缀。

### 接口

```go
type Authenticator interface {
    ValidateBearerToken(headerValue string) (*Principal, error)
}
```

定义在 `domain/auth/service.go`。`Service` 实现了 `Authenticator`，同时供给 `JWTMiddleware`。

### 服务

```go
type Service
    func NewService(config Config) *Service
    func (s *Service) GenerateToken(subject string, extra map[string]any) (string, error)
    func (s *Service) GenerateTokenAt(subject string, extra map[string]any, now time.Time) (string, error)
    func (s *Service) ValidateToken(token string) (*Principal, error)
    func (s *Service) ValidateTokenAt(token string, now time.Time) (*Principal, error)
    func (s *Service) ValidateBearerToken(headerValue string) (*Principal, error)
    func (s *Service) ValidateBearerTokenAt(headerValue string, now time.Time) (*Principal, error)
```

所有带 `At` 后缀的方法接收显式的 `time.Time` 参数，用于测试。

### HTTP 适配

```go
type HTTPHandler
    func NewHTTPHandler(service *Service, tokenTTL int) *HTTPHandler
    func (h *HTTPHandler) GenerateToken(c *gin.Context)
```

- 接收 JSON：`{"subject": "...", "extra": {...}}`（extra 可选）
- 返回 JSON：`{"token": "...", "tokenType": "Bearer", "expiresIn": 3600}`
- `GenerateTokenRequest` 和 `GenerateTokenResponse` 为公开类型，可直接引用

### 中间件

```go
func JWTMiddleware(authenticator Authenticator) gin.HandlerFunc
```

从 `Authorization` header 提取 Bearer token，验证后将 `Principal`、`jwt_subject`、`jwt_claims` 写入 Gin context。

**常量：**

| 常量 | 值 |
| --- | --- |
| `AuthorizationHeader` | `"Authorization"` |
| `ContextPrincipal` | `"auth_principal"` |
| `ContextJWTSubject` | `"jwt_subject"` |
| `ContextJWTClaims` | `"jwt_claims"` |
| `BearerPrefix` | `"Bearer "` |

## 依赖规则

| 依赖方向 | 说明 |
| --- | --- |
| `domain/auth` → `pkg/httputil` | HTTP 响应封装 |
| `domain/auth` → `pkg/logger` | 日志记录 |
| `domain/auth` → `pkg/config` | 配置类型（AuthConfig.AuthConfig() 方法） |
| `domain/auth` → `gin` | HTTP 框架 |

**禁止**：`domain/auth` → `domain/market`

## 新增认证能力指南

1. 需要新的 JWT 声明字段时，在 `jwt.go` 的 `JWTClaims` 中添加
2. 需要新的 Token API 时，在 `http_handler.go` 的 `HTTPHandler` 中添加方法
3. 所有依赖组装在 `cmd/server/main.go` 中完成
4. 不要将 `Authenticator` 接口移到 `pkg/` 下
