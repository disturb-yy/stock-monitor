# Contracts: 每日收盘总结 API

**Feature**: 007-daily-summary

## 1. GET /api/market/daily-summary — 查询当日收盘总结

查询当日（或指定日期）的收盘总结 JSON 数据。

### 请求

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `date` | `string` (query) | 否 | 查询日期（YYYY-MM-DD），默认当日 |

### 200 响应

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "trade_date": "2026-06-11",
    "is_trading_day": true,
    "up_count": 4,
    "down_count": 2,
    "flat_count": 0,
    "total_indices": 6,
    "top_gainer": {
      "symbol": "000001.SH",
      "name": "上证指数",
      "price": 3120.50,
      "change_percent": 0.39,
      "change": 12.30
    },
    "top_loser": {
      "symbol": "399006.SZ",
      "name": "创业板指",
      "price": 1935.42,
      "change_percent": -0.54,
      "change": -10.48
    },
    "indices": [
      {"symbol": "000001.SH", "name": "上证指数", "price": 3120.50, "change_percent": 0.39, "change": 12.30},
      {"symbol": "399001.SZ", "name": "深证成指", "price": 9820.18, "change_percent": -0.19, "change": -18.72}
    ],
    "anomaly_summary": {
      "total_count": 2,
      "items": [
        {"symbol": "000001.SH", "index_name": "上证指数", "type": "price_up", "type_name": "上涨异动", "trigger_value": 3.52}
      ]
    },
    "data_status": "ok",
    "generated_at": "2026-06-11T15:15:00+08:00"
  }
}
```

### 非交易日响应

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "trade_date": "2026-06-13",
    "is_trading_day": false,
    "data_status": "ok",
    "generated_at": "2026-06-13T15:15:00+08:00"
  }
}
```

## 2. POST /api/market/daily-summary — 手动触发收盘总结推送

异步触发当日收盘总结生成并推送到企业微信。立即返回确认，推送在后台完成。

### 请求

无请求体。

### 200 响应

```json
{
  "code": 0,
  "msg": "收盘总结推送已触发",
  "data": {"triggered": true}
}
```

### 非交易日响应

```json
{
  "code": 0,
  "msg": "非交易日，不推送",
  "data": {"triggered": false}
}
```

### Webhook 未启用响应

```json
{
  "code": 50000,
  "msg": "Webhook 推送未启用，请先配置 webhook.enabled=true",
  "data": {"triggered": false}
}
```

## 3. 企业微信推送格式 (Markdown)

自动或手动触发的收盘总结，以 Markdown 格式推送到企业微信群机器人：

```markdown
## 📊 收盘总结

**交易日期**：2026-06-11

**指数表现**：
上证指数 +0.39%、深证成指 -0.19%、创业板指 -0.54%
科创50 +0.32%、沪深300 +0.24%、中证500 -0.11%

> 上涨 4 只，下跌 2 只
> 领涨：上证指数 (+0.39%)
> 领跌：创业板指 (-0.54%)

**异动事件**（共 2 条）：
- 上证指数 上涨异动 (+3.52%)
- 创业板指 下跌异动 (-2.54%)

---
*stock-monitor 自动推送 · 2026-06-11 15:15*
```

### 数据异常时的推送格式

```markdown
## 📊 收盘总结

**交易日期**：2026-06-11

⚠️ **数据异常：部分数据获取失败**

**指数表现**：
上证指数 +0.39%、深证成指 数据缺失、...
> 可用数据：上涨 3 只，下跌 1 只
> 领涨：上证指数 (+0.39%)
> 4 个指数数据不可用

**异动事件**（共 1 条）：
- 上证指数 上涨异动 (+3.52%)

---
*stock-monitor 自动推送 · 2026-06-11 15:15 · ⚠️ 数据异常*
```

## 4. 内部接口契约: daily → notify

daily 领域不直接依赖 notify。Composition Root（`cmd/server/main.go`）负责：

1. 生成 `DailySummary` → 构建 Markdown 文本 → 调用 `notify.Dispatcher.Dispatch()`
2. daily 领域定义 `PushCallback` 接口，main.go 注入 notify 实现的适配器

```go
// domain/daily/summary.go
type PushFunc func(ctx context.Context, summary DailySummary) error
```
