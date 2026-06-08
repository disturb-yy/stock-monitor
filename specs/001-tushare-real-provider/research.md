# Research: Tushare 真实行情接入

## 1. Tushare `index_daily` API → IndexQuote 映射

**Decision**: 使用 `pkg/tushare` 已有的 `IndexDaily` 方法获取每日 K 线，取最新一条记录映射为 `IndexQuote`。

**Rationale**: `pkg/tushare/client.go` 已实现 `IndexDaily(ctx, tsCode, startDate, endDate, fields...)` 方法，返回 `[]IndexDailyItem`。`IndexDailyItem` 包含 OHLCV 字段。取当天或最近一个交易日的数据，映射到 `IndexQuote` 模型。

**Mapping**:

| IndexDailyItem 字段 | IndexQuote 字段 | 说明 |
|---------------------|-----------------|------|
| `TsCode` | `Symbol` | 指数代码 |
| — | `Name` | Tushare API 不返回名称，需内置映射表 |
| `Close` | `Price` | 最新价 = 收盘价（最新 K 线） |
| `Close - preClose` | `Change` | 涨跌额（需缓存昨收） |
| `(Close - preClose) / preClose * 100` | `ChangePercent` | 涨跌幅 |
| `Open` | `Open` | 直接映射 |
| `High` | `High` | 直接映射 |
| `Low` | `Low` | 直接映射 | 
| 前一日 `Close` | `PreClose` | 需查前一日数据 |
| `Volume` | `Volume` | 需 int64 转换（原始为 float64） |
| `Amount` | `Amount` | 需单位转换（原始为千元） |
| — | `UpdatedAt` | `time.Now()` |
| — | `Market` | 固定 `MarketCNA` |

**Alternatives considered**:
- Tushare `index_basic` + `index_daily` 组合查询：可获得指数名称，但增加一次请求，对 6 指数不划算。选择内置映射表。
- 实时行情接口（如新浪/东方财富）：非 Tushare 方案，超出范围。

## 2. 市场状态判断

**Decision**: 基于服务器本地时间的固定时段判断，不考虑交易日历 API。

**Rationale**: Phase 1 采用固定时段规则，后续可升级。交易时段为标准 A 股时间（东八区）：

| 时间段 | 状态 |
|--------|------|
| 00:00 - 09:15 | `pre_open` |
| 09:15 - 09:25 | `pre_open`（集合竞价） |
| 09:30 - 11:30 | `trading` |
| 11:30 - 13:00 | `lunch_break` |
| 13:00 - 15:00 | `trading` |
| 15:00 - 24:00 | `closed` |
| 周六/周日 | `closed` |
| Tushare 无当日数据时 | 回退为 `closed` |

**TradeDate 确定**:
- 若在交易时段内：`time.Now()` 的日期
- 若收盘后/周末：`IndexDaily` 返回的最新 `TradeDate`

**Alternatives considered**:
- 调用 Tushare `trade_cal` 接口查交易日历：需要额外 Token 权限，且增加请求延迟。Phase 1 不采用。

## 3. 多指数并发查询策略

**Decision**: 使用 goroutine 并发查询 6 个指数，用 ``sync.WaitGroup`` 收集结果，部分失败不影响其余。

**Rationale**: Tushare `IndexDaily` 每次仅查询一个指数代码。6 个指数若串行查询约为 6 × 200ms = 1.2s，并发可将总延迟控制在 ~300ms。使用标准库 `sync.WaitGroup` + goroutine 管理并发，收集成功结果，记录失败指数。

**Alternatives considered**:
- 串行查询：实现简单但延迟高，不符合 <3s 目标
- 批量接口：Tushare 不支持单次多指数查询

## 4. 错误处理策略

**Decision**: 分层错误处理。

| 层级 | 策略 |
|------|------|
| Tushare HTTP 层 | `pkg/tushare` 已有错误返回（网络错误、API 错误码），保持不动 |
| Provider 层 | 包装错误为 `fmt.Errorf("tushare: get index %s: %w", code, err)` |
| Service 层 | 透传错误 |
| HTTP Handler 层 | 已有错误处理（`pkg/httputil`），返回统一错误 JSON |

**Tushare 服务不可用场景**:
- 网络超时 → `pkg/tushare` 返回错误 → Provider 返回错误 → HTTP 500 + 错误描述
- Token 无效 → `pkg/tushare` 返回 API 错误 → Provider 返回错误 → HTTP 500
- 频率限制 → Tushare 返回错误码 → Provider 返回错误

## 5. 配置结构设计

**Decision**: 在 `config.yaml` 中增加 `tushare` 和 `provider` 配置段。

```yaml
provider:
  type: mock  # mock | tushare

tushare:
  token: ""                # Tushare Pro API Token
  base_url: ""             # 可选，默认 http://api.tushare.pro
  timeout: 30              # 请求超时（秒）
  indices:                 # 追踪的指数列表
    - symbol: "000001.SH"
      name: "上证指数"
    - symbol: "399001.SZ"
      name: "深证成指"
    - symbol: "399006.SZ"
      name: "创业板指"
    - symbol: "000688.SH"
      name: "科创50"
    - symbol: "000300.SH"
      name: "沪深300"
    - symbol: "000905.SH"
      name: "中证500"
```

**Rationale**: 配置驱动，符合 FR-005（不硬编码 Token）和 FR-006（配置切换）。指数列表可扩展。

**Alternatives considered**:
- 环境变量：可补充但不应是唯一方式，YAML 更统一
- 硬编码指数列表：不符合可扩展要求
