# pkg/httputil — HTTP 响应工具

## 包名

`httputil`

## 文件

| 文件 | 内容 |
| --- | --- |
| `resp.go` | `Resp` 结构体、`Response` 函数 |
| `errcode.go` | 统一错误码常量 |

## 公开类型

### 响应体

```go
type Resp struct {
    Code int    `json:"code"`
    Msg  string `json:"msg"`
    Data any    `json:"data,omitempty"`
}
```

### 错误码

| 常量 | 值 | 说明 |
| --- | --- | --- |
| `Success` | `0` | 成功 |
| `ParamError` | `40001` | 参数错误 |
| `InternalError` | `50000` | 内部错误 |

### 函数

```go
func Response(c *gin.Context, status int, resp Resp)
```

向 `gin.Context` 写入 JSON 响应，同时设置 HTTP 状态码。

## 使用示例

```go
// 成功
httputil.Response(c, http.StatusOK, httputil.Resp{
    Code: httputil.Success,
    Msg:  "success",
    Data: result,
})

// 参数错误
httputil.Response(c, http.StatusBadRequest, httputil.Resp{
    Code: httputil.ParamError,
    Msg:  "subject is required",
})

// 内部错误
httputil.Response(c, http.StatusInternalServerError, httputil.Resp{
    Code: httputil.InternalError,
    Msg:  "internal server error",
})
```

## 依赖

| 依赖 | 用途 |
| --- | --- |
| `github.com/gin-gonic/gin` | `c.JSON()` |
