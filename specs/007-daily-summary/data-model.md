# Data Model: 每日收盘总结推送

**Feature**: 007-daily-summary

## 实体

### DailySummary（收盘总结）

| 字段 | 类型 | 说明 |
|------|------|------|
| `TradeDate` | `string` | 交易日期（YYYY-MM-DD） |
| `IsTradingDay` | `bool` | 是否为交易日 |
| `UpCount` | `int` | 上涨指数数量 |
| `DownCount` | `int` | 下跌指数数量 |
| `FlatCount` | `int` | 持平指数数量 |
| `TotalIndices` | `int` | 追踪指数总数 |
| `TopGainer` | `*IndexSnapshot` | 领涨指数（可为 nil） |
| `TopLoser` | `*IndexSnapshot` | 领跌指数（可为 nil） |
| `Indices` | `[]IndexSnapshot` | 所有指数快照列表 |
| `AnomalySummary` | `AnomalySummary` | 异动事件摘要 |
| `DataStatus` | `DataStatus` | 数据获取状态 |
| `GeneratedAt` | `time.Time` | 总结生成时间 |

### IndexSnapshot（指数快照）

| 字段 | 类型 | 说明 |
|------|------|------|
| `Symbol` | `string` | 指数代码，如 "000001.SH" |
| `Name` | `string` | 指数中文名称 |
| `Price` | `float64` | 当前价格 |
| `ChangePercent` | `float64` | 涨跌幅（%） |
| `Change` | `float64` | 涨跌额 |

### AnomalySummary（异动摘要）

| 字段 | 类型 | 说明 |
|------|------|------|
| `TotalCount` | `int` | 异动事件总数 |
| `Items` | `[]AnomalyItem` | 异动条目列表（最多 5 条） |

### AnomalyItem（异动条目）

| 字段 | 类型 | 说明 |
|------|------|------|
| `Symbol` | `string` | 指数代码 |
| `IndexName` | `string` | 指数名称 |
| `Type` | `string` | 异动类型（price_up / price_down / volume_spike / consecutive_up / consecutive_down） |
| `TypeName` | `string` | 异动类型中文名 |
| `TriggerValue` | `float64` | 触发值 |

### DataStatus（数据状态）

```go
type DataStatus string

const (
    DataStatusOK       DataStatus = "ok"        // 数据正常
    DataStatusPartial  DataStatus = "partial"   // 部分数据获取失败
    DataStatusFailed   DataStatus = "failed"    // 完全无法获取数据
)
```

## 关系

```
DailySummary
├── TopGainer → IndexSnapshot (可选)
├── TopLoser  → IndexSnapshot (可选)
├── Indices   → []IndexSnapshot
└── AnomalySummary
    └── Items → []AnomalyItem (≤5)
```

## 配置实体

### DailySummaryConfig

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Enabled` | `bool` | `false` | 启用开关 |
| `PushTime` | `string` | `"15:15"` | 推送时间（HH:MM） |
