# Data Model: Tushare 真实行情接入

## 实体总览

本次接入**不新增领域模型**，复用 `domain/market/quote.go` 中已有的模型。新增**配置结构体**用于读取 Tushare 连接参数和指数列表。

## 已有模型（只读，不修改）

### IndexQuote（指数行情）

| 字段 | 类型 | 来源 | 说明 |
|------|------|------|------|
| `Market` | `Market` | 常量 `MarketCNA` | 所属市场 |
| `Symbol` | `string` | Tushare `ts_code` | 指数代码 |
| `Name` | `string` | 配置映射表 | 指数中文名 |
| `Price` | `float64` | Tushare `close` | 最新价 |
| `Change` | `float64` | 计算：`close - pre_close` | 涨跌额 |
| `ChangePercent` | `float64` | 计算：`(close - pre_close) / pre_close * 100` | 涨跌幅 |
| `Open` | `float64` | Tushare `open` | 开盘价 |
| `High` | `float64` | Tushare `high` | 最高价 |
| `Low` | `float64` | Tushare `low` | 最低价 |
| `PreClose` | `float64` | 前一日 Tushare `close` | 昨收价 |
| `Volume` | `int64` | Tushare `vol`（float64→int64） | 成交量（手） |
| `Amount` | `float64` | Tushare `amount`（千元） | 成交额 |
| `UpdatedAt` | `time.Time` | `time.Now()` | 数据更新时间 |

**验证规则**:
- `Symbol` 不为空
- `Price` >= 0
- `ChangePercent` 在合理范围（±20%）

### MarketSession（市场状态）

| 字段 | 类型 | 来源 | 说明 |
|------|------|------|------|
| `Market` | `Market` | 常量 `MarketCNA` | 所属市场 |
| `Status` | `MarketStatus` | 时段判断逻辑 | 当前状态 |
| `IsTrading` | `bool` | `Status == MarketStatusTrading` | 是否交易中 |
| `TradeDate` | `string` | `time.Now()` 或 Tushare `trade_date` | 交易日 |
| `Timezone` | `string` | 固定 `"Asia/Shanghai"` | 时区 |
| `UpdatedAt` | `time.Time` | `time.Now()` | 更新时间 |

**状态转换**（时间驱动）:

```
[00:00] → pre_open → [09:30] → trading → [11:30] → lunch_break
    → [13:00] → trading → [15:00] → closed → [00:00] → ...
周末全天 → closed
```

### MarketOverview（大盘总览）

| 字段 | 类型 | 来源 | 说明 |
|------|------|------|------|
| `Market` | `Market` | 同 MarketSession | 所属市场 |
| `Status` | `MarketStatus` | 同 MarketSession | 市场状态 |
| `TradeDate` | `string` | 同 MarketSession | 交易日 |
| `UpdatedAt` | `time.Time` | `time.Now()` | 更新时间 |
| `Summary` | `MarketSummary` | 计算 | 涨跌统计汇总 |
| `Indices` | `[]IndexQuote` | 聚合 `GetIndexQuotes` | 指数行情列表 |

**MarketSummary 计算规则**:
- `RisingIndexCount`: `ChangePercent > 0` 的指数数量
- `FallingIndexCount`: `ChangePercent < 0` 的指数数量
- `FlatIndexCount`: `ChangePercent == 0` 的指数数量
- `TotalAmount`: 所有指数 `Amount` 之和

## 新增配置结构

### TushareConfig

```go
type TushareConfig struct {
    Token   string        // Tushare Pro API Token
    BaseURL string        // API 地址（默认 http://api.tushare.pro）
    Timeout time.Duration // 请求超时
    Indices []IndexConfig // 追踪指数列表
}

type IndexConfig struct {
    Symbol string // 指数代码，如 "000001.SH"
    Name   string // 指数名称，如 "上证指数"
}
```

### ProviderConfig

```go
type ProviderConfig struct {
    Type string // "mock" 或 "tushare"
}
```

## 实体关系

```
ProviderConfig ──type──→ 选择 Provider 实现
TushareConfig ──token──→ pkg/tushare.Client
TushareConfig ──indices──→ TushareProvider 查询列表

TushareClient.IndexDaily(tsCode) → IndexDailyItem → 映射 → IndexQuote
TushareProvider.GetMarketStatus() → 时间判断 → MarketSession
TushareProvider.GetMarketOverview() → MarketSession + []IndexQuote → MarketOverview
```
