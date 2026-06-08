# Data Model: 异动检测规则引擎

## 新增实体

### AnomalyEvent（异动事件）

| 字段 | 类型 | 说明 |
|------|------|------|
| `ID` | `string` | 事件唯一标识（`{ruleType}_{symbol}_{timestamp}` 哈希） |
| `Type` | `AnomalyType` | 枚举：`price_up`、`price_down`、`volume_spike`、`consecutive_up`、`consecutive_down` |
| `Symbol` | `string` | 指数代码，如 "000001.SH" |
| `IndexName` | `string` | 指数名称 |
| `TriggerValue` | `float64` | 触发值（涨跌幅%、成交量倍数、连续天数） |
| `Threshold` | `float64` | 配置阈值 |
| `Description` | `string` | 人类可读描述 |
| `TriggeredAt` | `time.Time` | 触发时间 |

### AnomalyType（事件类型枚举）

```go
const (
    AnomalyPriceUp        = "price_up"
    AnomalyPriceDown      = "price_down"
    AnomalyVolumeSpike    = "volume_spike"
    AnomalyConsecutiveUp  = "consecutive_up"
    AnomalyConsecutiveDown = "consecutive_down"
)
```

### DetectionRule 接口

```go
type Detector interface {
    Type() string
    Detect(ctx context.Context, quotes []IndexQuote, historical map[string][]IndexDailyItem) ([]AnomalyEvent, error)
}
```

### 具体规则结构

| 规则 | 结构体 | 关键参数 |
|------|--------|----------|
| 涨跌幅 | `PriceChangeRule` | `UpperThreshold`、`LowerThreshold` |
| 成交量 | `VolumeSpikeRule` | `LookbackDays`、`SpikeMultiplier` |
| 连续涨跌 | `ConsecutiveRule` | `Days`（连续天数） |

### 引擎

```go
type Engine struct {
    detectors []Detector
}

func (e *Engine) Detect(ctx context.Context, quotes []IndexQuote, historical map[string][]IndexDailyItem) []AnomalyEvent
```

## 已有模型关联

| 已有模型 | 关联方式 |
|----------|----------|
| `IndexQuote` | 检测输入：`Price`、`ChangePercent`、`Volume` |
| `IndexDailyItem` | 历史数据：`Close`（连续涨跌）、`Volume`（成交量均值） |

## 配置结构

```yaml
anomaly:
  enabled: true
  rules:
    - type: price_change
      upper_threshold: 3.0
      lower_threshold: -2.0
      enabled: true
    - type: volume_spike
      lookback_days: 5
      spike_multiplier: 2.0
      enabled: true
    - type: consecutive
      days: 5
      enabled: true
```
