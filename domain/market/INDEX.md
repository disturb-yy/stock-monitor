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
| `collector.go` | `HistoryStore` 内存历史存储 + `Collector` 后台定时采集器 |

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

### 存储与采集

| 类型 | 文件 | 说明 |
| --- | --- | --- |
| `HistoryStore` | `collector.go` | 内存历史行情存储，`map[string][]IndexQuote`，`RWMutex` 保护 |
| `Collector` | `collector.go` | 后台定时采集器，交易时段按 `time.Ticker` 周期拉取 Tushare 行情 |

- `HistoryStore` — 构造函数 `NewHistoryStore(maxDays int) *HistoryStore`
  - `Append(symbol, quote)` — 追加记录（同日去重 + 自动裁剪旧数据）
  - `Query(symbol, start, end) (map[string][]IndexQuote, error)` — 按日期范围查询
- `Collector` — 构造函数 `NewCollector(provider *TushareProvider, store *HistoryStore, calendar *TradingCalendar, intervalMinutes int) *Collector`
  - `Start(ctx)` — 启动后台采集循环（ticker + goroutine）
  - `Stop()` — 优雅关闭（停止 ticker，关闭 done channel）
  - 采集前检查交易日历 + 交易时段，非交易时段跳过
  - 仅 Tushare 模式生效；Mock 模式下不创建

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
    func (h *HTTPHandler) GetHistory(c *gin.Context)
    func (h *HTTPHandler) GetMarketOverview(c *gin.Context)
```

每个方法对应一个 HTTP endpoint，使用 `httputil.Response` 统一响应格式。

**新增端点**: `GET /api/market/history?start=YYYY-MM-DD[&end=YYYY-MM-DD][&symbol=000001.SH]`
- 查询采集器缓存的历史行情数据
- `start` 必填，`end` 和 `symbol` 可选
- 仅在 Tushare 模式且 `collector.enabled=true` 时返回数据

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
