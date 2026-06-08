# domain/market — 市场行情领域

## 包名

`market`

## 文件地图

| 文件 | 内容 |
| --- | --- |
| `quote.go` | 领域模型（Market、MarketStatus、MarketSession、IndexQuote、MarketOverview、MarketSummary） |
| `provider.go` | `Provider` 接口：行情数据源契约 |
| `mock_provider.go` | `MockProvider`：A 股模拟数据实现 |
| `tushare_provider.go` | `TushareProvider`：Tushare 真实数据源（已实现） |
| `calendar.go` | `TradingCalendar`：A 股交易日历缓存与查询 |
| `service.go` | `Service`：业务服务，委托 Provider 查询 |
| `http_handler.go` | `HTTPHandler`：Gin HTTP 适配层 |

## 公开类型

### 领域模型

| 类型 | 文件 | 说明 |
| --- | --- | --- |
| `Market` (`string`) | `quote.go` | 市场标识，常量 `MarketCNA` = `"CN_A"` |
| `MarketStatus` (`string`) | `quote.go` | 市场状态：`MarketStatusUnknown`、`MarketStatusPreOpen`、`MarketStatusTrading`、`MarketStatusLunchBreak`、`MarketStatusClosed` |
| `MarketSession` | `quote.go` | 市场会话：Market、Status、IsTrading、TradeDate、Timezone、UpdatedAt |
| `IndexQuote` | `quote.go` | 指数行情：Market、Symbol、Name、Price、Change、ChangePercent、Open、High、Low、PreClose、Volume、Amount、UpdatedAt |
| `MarketOverview` | `quote.go` | 大盘总览：Market、Status、TradeDate、UpdatedAt、Summary (MarketSummary)、Indices ([]IndexQuote) |
| `MarketSummary` | `quote.go` | 大盘摘要：RisingIndexCount、FallingIndexCount、FlatIndexCount、TotalAmount |

### 接口

```go
type Provider interface {
    Market() Market
    GetMarketStatus(ctx context.Context) (*MarketSession, error)
    GetIndexQuotes(ctx context.Context) ([]IndexQuote, error)
    GetMarketOverview(ctx context.Context) (*MarketOverview, error)
}
```

定义在 `domain/market/provider.go`。所有 Provider 实现（Mock、Tushare）均实现在 market 包内，不向外暴露独立接口包。

### Provider 实现

- `MockProvider` — 构造函数 `NewMockProvider() *MockProvider`，返回 A 股模拟数据
- `TushareProvider` — 构造函数 `NewTushareProvider(client *tushare.Client) *TushareProvider`
  - **已实现**：`GetMarketStatus`、`GetIndexQuotes`、`GetMarketOverview`
  - 通过 `pkg/tushare.Client` 调用 Tushare Pro API 获取真实行情
  - 市场状态基于 Asia/Shanghai 时区时间判断（9:30-11:30, 13:00-15:00 为交易时段）
  - 指数行情并发查询（errgroup），部分失败不影响其余
  - 配置切换：`config.yaml` 中 `provider.type` 设为 `"tushare"`，并通过 `TUSHARE_TOKEN` 环境变量提供 Token

### 服务

```go
type Service
    func NewService(provider Provider) *Service
    func (s *Service) GetMarketStatus(ctx context.Context) (*MarketSession, error)
    func (s *Service) GetIndexQuotes(ctx context.Context) ([]IndexQuote, error)
    func (s *Service) GetMarketOverview(ctx context.Context) (*MarketOverview, error)
```

### HTTP 适配

```go
type HTTPHandler
    func NewHTTPHandler(service *Service) *HTTPHandler
    func (h *HTTPHandler) GetMarketStatus(c *gin.Context)
    func (h *HTTPHandler) GetMarketIndices(c *gin.Context)
    func (h *HTTPHandler) GetMarketOverview(c *gin.Context)
```

每个方法对应一个 HTTP endpoint，使用 `httputil.Response` 统一响应格式。

## 依赖规则

| 依赖方向 | 说明 |
| --- | --- |
| `domain/market` → `pkg/httputil` | HTTP 响应封装 |
| `domain/market` → `pkg/logger` | 日志记录 |
| `domain/market` → `pkg/tushare` | Tushare 客户端（TushareProvider） |
| `domain/market` → `gin` | HTTP 框架 |

**禁止**：`domain/market` → `domain/auth`（跨领域依赖须通过 `cmd/server/main.go` Composition Root 组装）

## 新增 Provider 指南

1. 在 `domain/market/` 下创建新文件，实现 `Provider` 接口
2. 提供构造函数（如 `NewXxxProvider(...) *XxxProvider`）
3. 在 `cmd/server/main.go` 中创建实例并注入 `NewService(provider)`
4. 不要将 Provider 接口或实现放在 `pkg/` 下
