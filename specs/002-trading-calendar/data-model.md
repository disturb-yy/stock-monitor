# Data Model: 交易日历接入

## 新增实体

### TradingCalendar（交易日历缓存）

内存结构，从 JSON 文件加载。

| 字段 | 类型 | 说明 |
|------|------|------|
| `Days` | `map[string]bool` | key=日期 "2006-01-02"，value=是否为交易日 |
| `StartDate` | `string` | 覆盖起始日期 "2006-01-02" |
| `EndDate` | `string` | 覆盖截止日期 "2006-01-02" |
| `UpdatedAt` | `time.Time` | 上次更新时间 |

**文件格式** (`data/trading_calendar.json`):
```json
{
  "updated_at": "2026-06-08T00:00:00Z",
  "start_date": "2026-01-01",
  "end_date": "2026-12-31",
  "days": {
    "2026-06-08": true,
    "2026-06-07": false,
    "2026-06-06": false
  }
}
```

### TradeCalItem（Tushare API 返回）

| 字段 | 类型 | 来源 | 说明 |
|------|------|------|------|
| `CalDate` | `string` | API `cal_date` | 日期 YYYYMMDD |
| `IsOpen` | `int` | API `is_open` | 1=交易日 0=非交易日 |
| `PretradeDate` | `string` | API `pretrade_date` | 前一交易日 YYYYMMDD |

## 已有模型修改

### MarketSession（不变）

| 字段 | 修改 |
|------|------|
| `Status` | 计算逻辑改为：日历判断 + 时段判断 |
| `TradeDate` | 计算逻辑改为：日历 pretrade_date 或当天日期 |
| 其余字段 | 不变 |

### TushareProvider

新增字段：
```go
type TushareProvider struct {
    client   *tushare.Client
    calendar *TradingCalendar  // 新增
}
```

## 配置新增

```yaml
calendar:
  cache_file: "data/trading_calendar.json"
  update_interval_days: 7   # 缓存有效期
  lookback_months: 6        # 向前覆盖月数
  lookahead_months: 6       # 向后覆盖月数
```
