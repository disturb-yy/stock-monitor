# Provider Interface Contract

## Interface: `market.Provider`

```go
type Provider interface {
    Market() Market
    GetMarketStatus(ctx context.Context) (*MarketSession, error)
    GetIndexQuotes(ctx context.Context) ([]IndexQuote, error)
    GetMarketOverview(ctx context.Context) (*MarketOverview, error)
}
```

File: `domain/market/provider.go`（已有，本次不修改）

## TushareProvider 实现契约

### GetMarketStatus

- **输入**: `context.Context`
- **输出**: `*MarketSession` 或 `error`
- **行为**:
  - 基于服务器本地时间（`Asia/Shanghai`）判断市场状态
  - 交易时段 9:30-11:30、13:00-15:00 返回 `trading`
  - 午休 11:30-13:00 返回 `lunch_break`
  - 收盘后/非交易日返回 `closed`
  - 盘前（00:00-9:30）返回 `pre_open`
  - `TradeDate` 默认使用当前日期，回退时使用最新行情日期
  - `IsTrading` 仅在 `Status == MarketStatusTrading` 时为 true

### GetIndexQuotes

- **输入**: `context.Context`
- **输出**: `[]IndexQuote` 或 `error`
- **行为**:
  - 并发查询配置中所有指数的 `index_daily` 数据
  - 取每个指数的最新一条 K 线记录
  - 映射 `IndexDailyItem` → `IndexQuote`（含涨跌幅计算）
  - 需查询前一日数据以获取 `PreClose`
  - 部分指数查询失败时，返回成功的结果 + 汇总错误信息
  - 全部失败时返回 error

### GetMarketOverview

- **输入**: `context.Context`
- **输出**: `*MarketOverview` 或 `error`
- **行为**:
  - 聚合 `GetMarketStatus()` + `GetIndexQuotes()`
  - 计算 `MarketSummary`（涨/跌/平计数 + 总成交额）
  - 任一子调用失败时，返回 error

## 配置契约

### config.yaml 新增段

```yaml
provider:
  type: mock  # mock | tushare

tushare:
# token 字段可通过环境变量 TUSHARE_TOKEN 覆盖（优先级更高）
  # token: ""  # 留空，通过环境变量 TUSHARE_TOKEN 设置，或直接填写
  base_url: ""               # 留空使用默认
  timeout: 30
  indices:
    - symbol: "000001.SH"
      name: "上证指数"
    - symbol: "399001.SZ"
      name: "深证成指"
    - symbol: "399006.SZ"
      name: "创业板指"
    - symbol: "000688.SH"
      name: "科创50"
    - symbol: "000300.SH"
      name: "沪深300"
    - symbol: "000905.SH"
      name: "中证500"
```

### Composition Root 切换逻辑

```text
if config.Provider.Type == "tushare":
    client = tushare.New(config.Tushare.Token, ...)
    provider = market.NewTushareProvider(client)
else:
    provider = market.NewMockProvider()

service = market.NewService(provider)
```

## API 响应契约（不变）

本次不修改 API 响应格式。以下接口契约保持不变：

| 端点 | 方法 | 响应模型 |
|------|------|----------|
| `/api/market/status` | GET | `httputil.Resp{Data: MarketSession}` |
| `/api/market/indices` | GET | `httputil.Resp{Data: []IndexQuote}` |
| `/api/market/overview` | GET | `httputil.Resp{Data: MarketOverview}` |

错误时返回 `httputil.Resp{Code: errcode, Msg: message}`。
