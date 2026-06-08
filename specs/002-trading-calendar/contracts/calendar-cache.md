# Calendar Cache Contract

## 文件格式

文件路径: `data/trading_calendar.json`（可通过配置修改）

```json
{
  "updated_at": "2026-06-08T00:00:00+08:00",
  "start_date": "2026-01-01",
  "end_date": "2026-12-31",
  "days": {
    "2026-06-08": true,
    "2026-06-07": false
  }
}
```

## 内存结构

```go
type TradingCalendar struct {
    Days      map[string]bool  // "2006-01-02" → isTradingDay
    StartDate string
    EndDate   string
    UpdatedAt time.Time
}
```

## 核心方法

```go
// IsTradingDay checks if the given date is a trading day.
func (c *TradingCalendar) IsTradingDay(date time.Time) bool

// LastTradeDate returns the most recent trading day on or before the given date.
func (c *TradingCalendar) LastTradeDate(date time.Time) (string, bool)

// NeedsRefresh checks if the calendar needs re-fetching.
func (c *TradingCalendar) NeedsRefresh() bool
```

## 更新流程

```
启动时:
  1. 尝试从 cache_file 加载
  2. 加载失败或 NeedsRefresh() → 从 Tushare 拉取
  3. 拉取成功 → 写入 cache_file
  4. 拉取失败 → 使用旧缓存 + 告警日志

拉取参数:
  exchange: "SSE"
  start_date: today - lookback_months
  end_date: today + lookahead_months
```
