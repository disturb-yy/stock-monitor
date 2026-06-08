# Research: 交易日历接入

## 1. Tushare `trade_cal` API

**Decision**: 使用 Tushare `trade_cal` 接口获取 A 股交易日历，参数 `exchange='SSE'`。

**Rationale**: `trade_cal` 返回每日期 + 是否开市（`is_open`）+ 盘前/盘后状态（`pretrade_date`）。一次查询可覆盖数月范围，完全满足需求。

**API 请求格式**:
```
api_name: trade_cal
params: { exchange: "SSE", start_date: "20260101", end_date: "20261231" }
fields: cal_date, is_open, pretrade_date
```

**返回示例**:
```
items: [
  { cal_date: "20260608", is_open: 1, pretrade_date: "20260605" },
  { cal_date: "20260607", is_open: 0, pretrade_date: "20260605" },
  ...
]
```

- `is_open`: 1=交易日, 0=非交易日（含周末和节假日）
- `pretrade_date`: 前一交易日（无论当前是否为交易日都返回）

**Alternatives considered**:
- 本地硬编码节假日列表：维护成本高，需每次手动更新
- 新浪/东方财富日历接口：多一个外部依赖，不如统一用 Tushare

## 2. 日历缓存策略

**Decision**: JSON 文件缓存到 `data/trading_calendar.json`，内容为日期→交易日布尔值的 map。每次启动时检查缓存有效期，过期则异步拉取更新。

**缓存格式**:
```json
{
  "updated_at": "2026-06-08T00:00:00Z",
  "start_date": "2026-01-01",
  "end_date": "2026-12-31",
  "days": {
    "20260608": true,
    "20260607": false
  }
}
```

**更新策略**:
- 启动时：缓存不存在或已过期（当前日期 > `end_date` 或 `updated_at` 超过 7 天）→ 全量拉取
- 拉取失败：继续使用过期缓存 + 告警日志
- 拉取成功：覆盖写入 JSON 文件

**Alternatives considered**:
- 内存 map 无持久化：每次重启都需重新拉取，浪费 API 额度
- SQLite/数据库：过度设计，Phase 1 不需要

## 3. 与现有时间判断的集成

**Decision**: 在 `GetMarketStatus` 中先查日历判断是否为交易日，再叠加时段判断当前状态。

**新逻辑流程**:
```
1. 查日历: today 是否为交易日？
   NO → 返回 closed（tradeDate = 最近交易日）
   YES → 继续
2. 时段判断: 当前时间在哪个交易时段？
   → 返回 pre_open / trading / lunch_break / closed
```

**Alternatives considered**:
- 完全依赖 Tushare `trade_cal` 的 `is_open` 不做时段细分：丢失盘前/午休细节
- 保留原函数不修改，在外层加 wrap：增加复杂度，不如直接整合

## 4. tradeDate 计算

**Decision**: 利用 Tushare `pretrade_date` 字段获取最近交易日。

**逻辑**:
- 如果今天是交易日 → tradeDate = today
- 如果今天非交易日 → tradeDate = 日历中返回的 pretrade_date
- 日历不可用 → 回退到 `latestTradeDate`（通过 API 查最新行情日期）

**Alternatives considered**:
- 向后遍历找最近交易日：需要遍历 map，效率低
- Tushare `pretrade_date` 字段直接提供，最优

## 5. pkg/tushare 扩展

**Decision**: 在 `pkg/tushare/client.go` 新增 `TradeCal` 方法，在 `types.go` 新增 `TradeCalItem` 类型。

**方法签名**:
```go
func (c *Client) TradeCal(ctx context.Context, exchange, startDate, endDate string) ([]TradeCalItem, error)
```

**TradeCalItem**:
```go
type TradeCalItem struct {
    CalDate     string // 日期 YYYYMMDD
    IsOpen      int    // 1=开市 0=休市
    PretradeDate string // 前一交易日
}
```
