# Data Model: 定时采集

## 新增实体

### HistoricalQuote（历史行情记录）

与 `IndexQuote` 结构相同，增加采集时间戳元数据。

| 字段 | 类型 | 说明 |
|------|------|------|
| `IndexQuote` | 嵌入 | 复用 `market.IndexQuote` 全部字段 |
| `CollectedAt` | `time.Time` | 采集时间 |

### 内存存储结构

```go
type HistoryStore struct {
    mu       sync.RWMutex
    data     map[string][]IndexQuote  // key=指数代码
    maxDays  int                       // 每个指数最多保留天数
}
```

**方法**:
- `Append(symbol string, quote IndexQuote)` — 追加记录，自动去重同日期
- `Query(symbol, start, end string) []IndexQuote` — 按日期范围查询
- `Trim()` — 裁剪超出 maxDays 的旧数据

## 已有模型关联

| 已有模型 | 关联方式 |
|----------|----------|
| `IndexQuote` | 采集器生成，直接存入 |
| `TradingCalendar` | 采集器依赖，判断是否执行采集 |
| `TushareProvider` | 采集器复用其 `fetchIndexQuote` 逻辑 |

## 配置新增

```yaml
collector:
  enabled: true
  interval_minutes: 5
  max_history_days: 30
```

## 新增 Config 类型

```go
type CollectorConfig struct {
    Enabled         bool `yaml:"enabled"`
    IntervalMinutes int  `yaml:"interval_minutes"`
    MaxHistoryDays  int  `yaml:"max_history_days"`
}
```
