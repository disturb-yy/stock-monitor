# domain/anomaly — 异动检测领域

## 包名

`anomaly`

## 文件地图

| 文件 | 内容 |
| --- | --- |
| `event.go` | `AnomalyEvent` 模型和 `AnomalyType` 常量 |
| `rule.go` | `Detector` 接口、`IndexData`、`HistoricalDay` 类型 |
| `engine.go` | `Engine`：注册检测器、聚合执行 |
| `price_change.go` | `PriceChangeDetector`：涨跌幅超阈值检测 |
| `volume_spike.go` | `VolumeSpikeDetector`：成交量突增检测 |
| `consecutive.go` | `ConsecutiveDetector`：连续涨跌检测 |
| `http_handler.go` | `HTTPHandler`：异动事件查询接口 |

## 公开类型

### 事件模型

```go
type AnomalyType string  // price_up, price_down, volume_spike, consecutive_up, consecutive_down

type AnomalyEvent struct {
    ID, Type, Symbol, IndexName, Description string
    TriggerValue, Threshold float64
    TriggeredAt time.Time
}
```

### 检测器接口

```go
type Detector interface {
    Type() AnomalyType
    Detect(ctx, current map[string]IndexData, historical map[string][]HistoricalDay) ([]AnomalyEvent, error)
}
```

### 引擎

```go
type Engine
    func NewEngine(detectors ...Detector) *Engine
    func (e *Engine) Detect(ctx, current, historical) []AnomalyEvent
```

### HTTP 适配

```go
type HTTPHandler
    func NewHTTPHandler(engine *Engine, client *tushare.Client) *HTTPHandler
    func (h *HTTPHandler) GetAnomalies(c *gin.Context)
```

## API

`GET /api/market/anomalies` — 返回当前检测到的所有异动事件。

## 配置

```yaml
anomaly:
  enabled: true
  rules:
    - type: price_change
      upper_threshold: 3.0
      lower_threshold: -2.0
    - type: volume_spike
      lookback_days: 5
      spike_multiplier: 2.0
    - type: consecutive
      days: 5
```

## DDF 依赖规则

| 依赖方向 | 说明 |
| --- | --- |
| `domain/anomaly` → `pkg/tushare` | Tushare 客户端（行情数据获取） |
| `domain/anomaly` → `pkg/httputil` | HTTP 响应封装 |
| `domain/anomaly` → `gin` | HTTP 框架 |

**禁止**：`domain/anomaly` → `domain/market`，`domain/anomaly` → `domain/auth`
（跨领域协作通过 `cmd/server/main.go` Composition Root 完成引擎创建和注入）
