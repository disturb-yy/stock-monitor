# Research: 异动检测规则引擎

## 1. 规则引擎架构

**Decision**: 采用管道模式（Pipeline/Chain），每个规则实现统一 `Detector` 接口，引擎按配置顺序执行。

**Rationale**: 规则之间互不依赖，各自独立判断。管道模式简洁，新增规则只需实现接口并注册。

```go
type Detector interface {
    Detect(ctx context.Context, quotes []IndexQuote) ([]AnomalyEvent, error)
}
```

**Alternatives considered**:
- 事件驱动（规则订阅行情数据）：过度设计，当前规模不需要
- 硬编码 if-else 分支：不符合 FR-005 可配置原则

## 2. 检测执行时机

**Decision**: 在 `GetMarketOverview` 和 `GetIndexQuotes` 的 HTTP handler 中，行情查询完成后同步执行检测。同时提供独立 `GET /api/market/anomalies` 端点做主动查询。

**Rationale**: 前端查询大盘总览时一併获得异动信息，减少请求次数。独立端点用于轮询和调试。

**Alternatives considered**:
- 定时后台任务：需要调度器，Phase 2 先不做
- 仅在独立端点执行：需要前端额外调用，体验差

## 3. 成交量均值和连续涨跌的历史数据获取

**Decision**: 使用 Tushare `IndexDaily` 接口获取近 N 个交易日数据，每次检测时实时查询。

**成交量均值计算**: 查询近 5 个交易日（不含当日），取 `vol` 均值。
**连续涨跌计算**: 查询近 N 个交易日（含今日），逐日比较涨跌方向。

**Performance concern**: 成交量检测需要额外 6 次 API 调用（每个指数查历史），连续涨跌同理。可通过 `IndexDaily` 一次调用返回多日数据来减少请求数——只需传入 `start_date` 和 `end_date`，而非逐日查询。

**Alternatives considered**:
- 本地缓存历史数据：需要数据库，超出 Phase 2 范围
- 内存缓存最近 30 天数据：可行，但本次先不做

## 4. 配置结构

**Decision**: YAML 规则列表，每种规则类型有独立参数段。

```yaml
anomaly:
  enabled: true
  rules:
    - type: price_change
      upper_threshold: 3.0    # 涨超 3%
      lower_threshold: -2.0   # 跌超 2%
      enabled: true
    - type: volume_spike
      lookback_days: 5        # 近 5 日均值
      spike_multiplier: 2.0   # 超过 2 倍
      enabled: true
    - type: consecutive
      days: 5                 # 连续 5 天
      enabled: true
```

**Alternatives considered**:
- 每个规则独立命名的 YAML key：结构松散，不如列表直观
- 环境变量配置：复杂结构不适合

## 5. 异常事件去重

**Decision**: 每次检测全量重新计算，不持久化，无去重。

**Rationale**: 无数据库，无法追踪历史事件。每次请求时基于最新行情数据生成当前事件快照。告警去重留到 Phase 2 告警通知功能中处理。

## 6. 服务注入方式

**Decision**: 通过 `market.HTTPHandler` 持有 `*anomaly.Engine`，在行情 handler 中调用 `engine.Detect(quotes)`。

**Rationale**: DDF 禁止 `domain/anomaly` → `domain/market` 的反向依赖，但 `domain/market` → `domain/anomaly` 是允许的（market 依赖 anomaly）。引擎由 Composition Root 创建并注入 handler。

实际上这会违反 "domains MUST NOT import each other"，所以改为：market handler 返回行情数据后，由 Composition Root 层（api 或 cmd）协调调用 anomaly engine，而不是在 market handler 内部直接依赖 anomaly。

**最终决策**: 在 `api/route.go` 或新的中间层协调，anomaly engine 由 `cmd/server/main.go` 创建，注入到 wrapper handler。
