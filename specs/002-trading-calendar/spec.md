# Feature Specification: 交易日历接入

**Feature Branch**: `002-trading-calendar`

**Created**: 2026-06-08

**Status**: Draft

**Input**: User description: "交易日历接入"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 节假日准确识别为非交易日 (Priority: P1)

作为 API 使用者，当我在法定节假日（如春节、国庆）请求市场状态时，系统应返回 `closed` 状态（非 `trading` 或 `pre_open`），而非错误地将工作日判断为交易日。

**Why this priority**: 当前实现仅按周末判断，法定节假日会误报为交易日，这是基础正确性问题。

**Independent Test**: 在元旦（1月1日）请求市场状态，验证返回 `closed`，`isTrading` 为 false。

**Acceptance Scenarios**:

1. **Given** 今天是法定节假日（如春节初一），**When** 用户请求市场状态，**Then** 返回 `closed` 状态，非 `trading` 或 `pre_open`。
2. **Given** 今天是调休工作日（如国庆调休周六上班），**When** 用户请求市场状态，**Then** 正常按交易时段判断（`trading`/`lunch_break`），而非直接判为周末 `closed`。
3. **Given** 今天是普通周末（非法定假期附近），**When** 用户请求市场状态，**Then** 返回 `closed`。

---

### User Story 2 - 市场状态返回准确交易日 (Priority: P1)

作为 API 使用者，当我查询市场状态时，`tradeDate` 字段应返回真实的最近交易日（如遇节假日则向前追溯），而非简单的当前日期。

**Why this priority**: `tradeDate` 是下游数据展示和查询的关键字段，不准确会导致数据错乱。

**Independent Test**: 在周一请求市场状态（盘前），验证 `tradeDate` 为上周末而非当前日期。

**Acceptance Scenarios**:

1. **Given** 今天是周一且为交易日（9:00），**When** 用户请求市场状态，**Then** `tradeDate` 为当天日期。
2. **Given** 今天是周一但为节假日，**When** 用户请求市场状态，**Then** `tradeDate` 为上周五（最近交易日）。
3. **Given** 今天是节后首个交易日，**When** 用户请求市场状态，**Then** `tradeDate` 为当天日期。

---

### User Story 3 - 交易日历可配置更新 (Priority: P2)

作为运维人员，我希望交易日历数据可以自动或手动更新，以确保临时休市等特殊安排能及时反映到系统中。

**Why this priority**: 临时休市虽然不常见但影响重大，需有更新机制。

**Independent Test**: 手动更新日历数据后，请求市场状态验证新日历生效。

**Acceptance Scenarios**:

1. **Given** 交易日历数据过期（超过预设有效期），**When** 系统启动或定时检查，**Then** 自动从数据源拉取最新日历，或发出过期警告。
2. **Given** 数据源不可用（网络故障），**When** 系统需要更新日历，**Then** 继续使用本地缓存的日历数据，并记录告警日志。

---

### Edge Cases

- 半年末、年末最后交易日日历是否正确？
- 数据源返回的日历数据格式异常（字段缺失）如何处理？
- 日历时区与服务器时区不一致时如何对齐？
- 日历数据覆盖的时间范围不足（如只覆盖到年底），过期后如何处理？

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 能够判断任意日期是否为 A 股交易日（交易日 = 非周末 + 非法定节假日 + 非临时休市）。
- **FR-002**: 系统 MUST 能够处理调休场景：周末调休为工作日时识别为交易日。
- **FR-003**: 系统 MUST 在查询市场状态时返回准确的最近交易日日期（`tradeDate`）。
- **FR-004**: 系统 MUST 支持交易日历数据的本地缓存，数据源不可用时使用缓存。
- **FR-005**: 系统 MUST 在交易日历数据过期时产生明确的告警信息。
- **FR-006**: 系统 MUST 替换当前 `TushareProvider` 中仅基于时间的简单状态判断逻辑，改由交易日历 + 时段联合判断。

### Key Entities

- **交易日历项 (TradingDay)**: 日期、是否为交易日、是否为调休日、市场类型（A股）。
- **交易日历缓存**: 一组日期的交易日历数据，含数据覆盖起始日期和截止日期。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 2026 年全年交易日判断准确率达到 100%（与上交所官方日历对比）。
- **SC-002**: 市场状态接口在交易日历查询场景下响应时间不超过当前实现（新增日历查询不增加明显延迟）。
- **SC-003**: 数据源不可用时，系统可连续使用本地缓存日历至少 30 天不报错（仅告警）。

## Assumptions

- 交易日历数据源优先使用 Tushare `trade_cal` 接口，该接口返回每日期是否开市。
- 日历数据覆盖范围至少为前后各 6 个月，每次更新拉取完整范围。
- 交易日历缓存以文件形式存储在本地。
- 当前已实现的基于时间的市场时段判断（9:30-11:30, 13:00-15:00）继续保留，叠加日历判断使用。
- 默认时区为 `Asia/Shanghai`，日历日期与时区无关（仅日期粒度）。
