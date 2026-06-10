# Research: 每日收盘总结推送

**Feature**: 007-daily-summary | **Date**: 2026-06-11

## 1. 领域归属决策

**Decision**: 创建新领域 `domain/daily/`（收盘总结领域）

**Rationale**:
- 收盘总结涉及独立的业务逻辑：总结生成、定时调度、手动触发、查询接口
- 与 `domain/notify/` 职责不同（notify 负责推送通道，daily 负责内容聚合与调度）
- 符合 DDF 扁平化原则：每个领域自包含
- 跨领域依赖通过 Composition Root 注入解决（与 anomaly→notify 模式一致）

**Alternatives considered**:
- 放入 `domain/market/`：market 已承担行情查询、采集、存储 3 类职责，再加入总结生成会导致职责过重
- 放入 `domain/notify/`：notify 是推送通道，不应包含业务内容生成逻辑

## 2. 定时调度方案

**Decision**: 使用 `time.Ticker` + 交易日历在进程内调度

**Rationale**:
- 功能只需每日触发 1 次，无需外部调度框架（cron、temporal）
- `time.Ticker` 每隔 60s 检查"当前时间是否匹配推送配置时间 + 今日是否交易日"
- 已经在 Collector 中有相同模式（`domain/market/collector.go` 使用 `time.Ticker`）
- 零外部依赖，符合宪法 SQLite-only 约束

**Alternatives considered**:
- `robfig/cron`: 外部依赖，增加了复杂度，每天 1 次的任务杀鸡用牛刀
- `time.AfterFunc`: 需精确计算到 15:15 的间隔，重启后需要重算

## 3. 数据获取策略

**Decision**: 通过接口注入 `SummaryDataProvider`，由 Composition Root 注入 `market.HistoryStore`

**Rationale**:
- 遵循 DDF 依赖规则：daily 定义接口，main.go 注入 market 实现
- `HistoryStore.Query()` 返回内存中的当日数据，性能足够
- Mock 实现可在 UT 中独立测试 daily 领域

**Alternatives considered**:
- 直接查询 SQLite：跨越了 market 领域封装，违反 DDF 规则
- 通过 HTTP 调用 market API：进程内调用增加网络开销，无意义

## 4. 总结内容推送方式

**Decision**: 复用 `domain/notify/` 的 WebhookSender，通过 Composition Root 转换

**Rationale**:
- 已有完善的推送通道（企业微信 Markdown、重试、历史记录）
- daily 定义 `Summary` 模型 → main.go 转为 Markdown → notify.Dispatcher
- 与 anomaly→notify 回调模式一致（FR-005 明确要求复用）

**Alternatives considered**:
- daily 直接调用 WebhookSender：daily 依赖 notify，违反 DDF 跨域禁止
- 新建独立推送通道：代码重复，维护两套重试/历史逻辑

## 5. 配置设计

**Decision**: 新增 `daily_summary:` 配置段，独立于 `webhook:` 段

```yaml
daily_summary:
  enabled: false
  push_time: "15:15"
```

**Rationale**:
- FR-008 要求独立开关，与 webhook 告警解耦
- 最小配置项数（2 项），遵循 YAGNI
- push_time 为 HH:MM 字符串，解析简单

## 6. 时区处理

**Decision**: 统一使用 `Asia/Shanghai` 时区（A 股）

**Rationale**:
- 项目已定义 `aStockTimezone = "Asia/Shanghai"`（`domain/market/mock_provider.go`）
- 推送时间比较、交易日判断、时间戳格式化均使用该时区
- 服务器时区不影响行为

## 7. 手动触发端点路径

**Decision**: `POST /api/market/daily-summary`（触发推送）+ `GET /api/market/daily-summary`（查询）

**Rationale**:
- 挂载在 `/api/market/` 下，与现有 API 分组一致
- POST 触发异步推送，GET 返回 JSON 总结
- 路径语义清晰

**Alternatives considered**:
- `/api/daily/summary`：需要新的路由 Group，当前 api/route.go 按 market 分组更自然
