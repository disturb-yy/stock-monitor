# Feature Specification: Tushare 真实行情接入

**Feature Branch**: `001-tushare-real-provider`

**Created**: 2026-06-08

**Status**: Draft

**Input**: User description: "接入 Tushare 真实行情数据，替换 mock provider"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 查询真实指数行情 (Priority: P1)

作为 API 使用者，当我调用指数行情接口时，我希望获得来自 Tushare 的真实 A 股指数数据（上证指数、深证成指、创业板指等），而非模拟数据，以便基于真实市场数据做决策。

**Why this priority**: 指数行情是市场监控的核心数据，没有真实数据整个系统失去实用价值。

**Independent Test**: 调用指数行情接口，验证返回的指数价格、涨跌幅与公开市场数据一致（允许秒级延迟）。

**Acceptance Scenarios**:

1. **Given** Tushare 服务正常可用，**When** 用户在交易日交易时段请求指数行情，**Then** 返回包含上证指数、深证成指、创业板指的最新行情，含价格、涨跌幅、成交量等字段。
2. **Given** Tushare 服务正常可用，**When** 用户在非交易日（周末/节假日）请求指数行情，**Then** 返回最近一个交易日的收盘行情数据。
3. **Given** Tushare 服务不可用（网络故障/配额耗尽），**When** 用户请求指数行情，**Then** 返回明确的错误信息，说明数据源暂时不可用，而非返回空数据或假数据。

---

### User Story 2 - 查询真实市场状态 (Priority: P1)

作为 API 使用者，当我调用市场状态接口时，我希望获得准确的 A 股市场当前交易状态（盘前/交易中/午休/已收盘），而非固定模拟值，以便判断当前是否可以交易。

**Why this priority**: 市场状态决定所有其他数据的时效性解释，是基础能力。

**Independent Test**: 在不同时段调用市场状态接口，验证返回的状态与实际 A 股交易时段一致。

**Acceptance Scenarios**:

1. **Given** 当前时间为交易日 9:30-11:30，**When** 用户请求市场状态，**Then** 返回 `trading` 状态，`isTrading` 为 true。
2. **Given** 当前时间为交易日 11:30-13:00，**When** 用户请求市场状态，**Then** 返回 `lunch_break` 状态，`isTrading` 为 false。
3. **Given** 当前时间为非交易日（周末），**When** 用户请求市场状态，**Then** 返回 `closed` 状态，`isTrading` 为 false，`tradeDate` 为最近一个交易日。

---

### User Story 3 - 查询大盘总览 (Priority: P2)

作为 API 使用者，当我调用大盘总览接口时，我希望一次性获得市场状态与所有关键指数的综合快照，减少请求次数，便于快速掌握市场全貌。

**Why this priority**: 总览是前两个接口的组合增强，提供一站式查询体验。

**Independent Test**: 调用大盘总览接口，验证返回数据同时包含正确的市场状态、指数行情列表、以及涨跌统计汇总。

**Acceptance Scenarios**:

1. **Given** Tushare 服务正常且市场在交易中，**When** 用户请求大盘总览，**Then** 返回市场状态 + 所有追踪指数行情 + 上涨/下跌/平盘指数计数 + 总成交额。
2. **Given** 部分指数行情获取失败（如某指数代码无效），**When** 用户请求大盘总览，**Then** 返回可获取的指数数据，并在错误信息中标明哪些指数获取失败。

---

### Edge Cases

- Tushare 返回空数据（新上市指数尚无历史数据）时如何处理？
- Tushare 频率限制触发（免费版每分钟有限额）时，连续请求如何表现？
- 交易日历变更（如临时休市）时，市场状态判断是否仍然准确？
- Tushare token 未配置或过期时，系统启动阶段如何反馈？

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 通过 Tushare Pro 获取 A 股主要指数的实时/最新行情数据。
- **FR-002**: 系统 MUST 根据当前服务器时间和 A 股交易日历判断市场状态（盘前/交易中/午休/已收盘）。
- **FR-003**: 系统 MUST 支持至少以下 A 股关键指数：上证指数（000001.SH）、深证成指（399001.SZ）、创业板指（399006.SZ）、科创50（000688.SH）、沪深300（000300.SH）、中证500（000905.SH）。
- **FR-004**: 系统 MUST 在 Tushare 服务不可用时返回明确的错误信息，不返回空数据或过期数据。
- **FR-005**: 系统 MUST 通过配置读取 Tushare API Token，不在代码中硬编码。
- **FR-006**: 系统 MUST 保留模拟数据源实现，允许通过配置切换模拟/真实数据源。
- **FR-007**: 系统 MUST 在市场总览中计算上涨、下跌、平盘指数数量及总成交额汇总。

### Key Entities *(include if feature involves data)*

- **Tushare 连接配置**: API 访问凭证、服务地址、超时时间等连接参数。
- **指数行情 (IndexQuote)**: 已有数据模型，本次接入填充真实数值 — 指数代码、名称、最新价、涨跌额、涨跌幅、开盘价、最高价、最低价、昨收、成交量、成交额。
- **市场状态 (MarketSession)**: 已有数据模型，本次接入基于真实时间/交易日历计算 — 市场标识、状态、是否交易中、交易日、时区。
- **大盘总览 (MarketOverview)**: 已有数据模型，本次接入填充真实汇总数据 — 市场状态 + 指数列表 + 涨跌统计 + 总成交额。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 指数行情数据与公开市场数据（如上交所/深交所官网）偏差不超过 0.1%。
- **SC-002**: 市场状态在 A 股交易时段切换点（9:30、11:30、13:00、15:00）前后 1 分钟内准确反映当前状态。
- **SC-003**: 大盘总览接口在数据源正常时 3 秒内完成响应（含多指数查询）。
- **SC-004**: 运维人员可通过配置项在模拟数据模式和真实数据模式之间切换，无需重新部署。

## Assumptions

- Tushare Pro API Token 由使用者自行注册获取，系统不提供 Token。
- A 股交易日历遵循中国沪深交易所官方日历，暂时采用固定时段判断（9:30-11:30, 13:00-15:00），后续可升级为交易日历查询。
- 默认追踪指数列表以上述 6 个指数为准，可通过配置扩展。
- Tushare 免费版的频率限制（约 200 次/分钟）对当前访问量足够，暂不需要客户端限流。
- 指数行情数据使用 Tushare 提供的最新交易日行情数据，代表当前行情。
