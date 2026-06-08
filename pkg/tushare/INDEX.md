# pkg/tushare — Tushare Pro API 客户端

## 包名

`tushare`

## 文件

| 文件 | 内容 |
| --- | --- |
| `client.go` | `Client` 结构体、`Do`、`IndexDaily` |
| `types.go` | 请求/响应类型、`IndexDailyItem` |

## 公开类型

### 客户端

```go
type Client struct { ... }
    func New(token string, opts ...Option) *Client
    func (c *Client) Do(ctx context.Context, apiName string, params map[string]any, fields ...string) (*Response, error)
    func (c *Client) IndexDaily(ctx context.Context, tsCode string, startDate, endDate string, fields ...string) ([]IndexDailyItem, error)
```

- `New`：使用 Tushare Pro API token 创建客户端
- `Do`：通用 API 请求，返回 `*Response`。传输/IO 错误返回 error；业务错误（Code != 0）同时返回 error 和 Response
- `IndexDaily`：查询指数日线数据

### 配置选项

```go
type Option func(*Client)
    func WithBaseURL(url string) Option       // 覆盖 API 地址
    func WithHTTPClient(hc *http.Client) Option  // 替换 HTTP 客户端
```

### 请求/响应

```go
type Request struct {
    APIName string         `json:"api_name"`
    Token   string         `json:"token"`
    Params  map[string]any `json:"params"`
    Fields  string         `json:"fields"`
}

type Response struct {
    Code int    `json:"code"`
    Msg  string `json:"msg"`
    Data *Data  `json:"data,omitempty"`
}

type Data struct {
    Fields []string `json:"fields"`
    Items  [][]any  `json:"items"`
}
```

### 指数日线

```go
type IndexDailyItem struct {
    TsCode    string   // 指数代码
    TradeDate string   // 交易日
    Open      float64  // 开盘价
    High      float64  // 最高价
    Low       float64  // 最低价
    Close     float64  // 收盘价
    Volume    float64  // 成交量（手）
    Amount    float64  // 成交额（千元）
}
```

## 使用示例

```go
client := tushare.New(token, tushare.WithBaseURL("http://api.tushare.pro"))
items, err := client.IndexDaily(ctx, "000001.SH", "20240101", "20240131")
```

## 注入指南

本包提供的 `Client` 由 Composition Root（`cmd/server/main.go`）根据配置创建并注入：

```go
tushareClient := tushare.New(cfg.Tushare.Token,
    tushare.WithBaseURL(cfg.Tushare.BaseURL),
    tushare.WithHTTPClient(&http.Client{
        Timeout: time.Duration(cfg.Tushare.Timeout) * time.Second,
    }),
)
marketProvider = market.NewTushareProvider(tushareClient)
```

Token 通过 `TUSHARE_TOKEN` 环境变量提供（优先级高于 `config.yaml`）。

## 指南

- 本包只封装 Tushare HTTP API 调用，不含业务模型转换
- 将 `IndexDailyItem` 转换为 `market.IndexQuote` 的逻辑在 `domain/market/tushare_provider.go` 中实现
- 新增 Tushare API 时，在 `client.go` 中添加方法，在 `types.go` 中添加对应的 Item 类型

## 依赖

| 依赖 | 用途 |
| --- | --- |
| `net/http` | HTTP 客户端 |
