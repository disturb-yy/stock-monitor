# pkg/router — 路由组辅助工具

## 包名

`router`

## 文件

`group.go`

## 公开类型

```go
type Group struct { ... }
    func NewGroup(group *gin.RouterGroup) *Group
    func (g *Group) Use(handlers ...gin.HandlerFunc) *Group
    func (g *Group) UseIf(enabled bool, handlers ...gin.HandlerFunc) *Group
    func (g *Group) GET(relativePath string, handlers ...gin.HandlerFunc) *Group
    func (g *Group) Native() *gin.RouterGroup
```

`Group` 是 `gin.RouterGroup` 的轻量包装，提供链式路由注册。

- `Use`：添加中间件
- `UseIf`：仅在 `enabled` 为 true 时添加中间件
- `GET`：注册 GET 路由
- `Native`：返回底层 `*gin.RouterGroup`

## 使用示例

```go
router.NewGroup(api.Group("/market")).
    UseIf(authCfg.Enabled, auth.JWTMiddleware(authService)).
    GET("/status", marketHandler.GetMarketStatus).
    GET("/indices", marketHandler.GetMarketIndices)
```

## 依赖

| 依赖 | 用途 |
| --- | --- |
| `github.com/gin-gonic/gin` | 路由框架 |
