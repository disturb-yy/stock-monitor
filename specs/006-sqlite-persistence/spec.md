# Feature Specification: SQLite 持久化

**Feature Branch**: `006-sqlite-persistence`

**Created**: 2026-06-09

**Status**: Draft

**Input**: User description: "SQLite 持久化 — 将 HistoryStore 内存数据定期刷写到 SQLite，重启不丢数据"

## Clarifications

### Session 2026-06-10

- Q: 启动时数据加载策略？ → A: 启动仅加载最近 maxDays 天数据到内存，更早数据保留在 SQLite 按需查询。
- Q: 内存与 SQLite 数据冲突时的优先级？ → A: 内存优先；查询先用内存，仅在内存无覆盖的日期范围才查 SQLite。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 服务重启后历史数据可恢复 (Priority: P1)

作为系统运维者，当服务因升级、宕机或手动重启而重新启动后，我希望之前采集的历史行情数据能从 SQLite 自动加载回内存，无需等待数小时重新采集才能使用历史查询和异动检测功能。

**Why this priority**: 历史数据是成交量均值计算和连续涨跌检测的基础依赖。重启丢数据意味着重启后相当长一段时间内异动检测的成交量/连续涨跌规则无法正常工作。

**Independent Test**: 服务运行一段时间采集到数据后，重启服务，立即请求历史行情查询接口，验证返回重启前采集的数据。

**Acceptance Scenarios**:

1. **Given** 服务已运行 1 小时并采集了 60 条历史行情记录，**When** 服务重启，**Then** 历史查询接口能返回重启前采集的最近 maxDays 内的数据，条数一致。
2. **Given** SQLite 文件尚不存在（首次启动），**When** 服务启动，**Then** 系统自动创建数据库文件并初始化表结构，不报错。
3. **Given** 服务已运行，**When** 新数据被采集到内存 HistoryStore，**Then** 系统同步将新数据写入 SQLite。

---

### User Story 2 - 历史查询不依赖内存 (Priority: P2)

作为 API 使用者，当请求的历史数据超出内存保留天数（如请求 30 天前的数据）时，我希望系统能从 SQLite 中查询并返回，而非仅返回内存中有限的数据。

**Why this priority**: 内存存储受限于容量（默认 30 天），SQLite 可按需扩展存储更多历史数据，提升查询能力。

**Independent Test**: 配置内存保留 7 天、SQLite 保留 90 天，写入 30 天历史数据到 SQLite，请求 20 天前的数据，验证能从 SQLite 返回。

**Acceptance Scenarios**:

1. **Given** 内存保留 7 天、SQLite 中存有 30 天数据，**When** 用户请求 10 天前的历史行情，**Then** 系统从 SQLite 查询并返回数据。
2. **Given** 请求的日期范围跨越内存和 SQLite 边界（部分在内存、部分仅在 SQLite），**When** 用户请求历史行情，**Then** 系统合并两个来源返回完整结果。

---

### User Story 3 - 数据库文件管理 (Priority: P3)

作为运维者，我希望 SQLite 数据库文件的位置、保留策略可配置，且文件损坏时系统能自动重建而不影响服务运行。

**Why this priority**: 数据库文件管理是生产环境运维的基本需求，但优先级低于核心的读写功能。

**Independent Test**: 手动删除 SQLite 文件后重启服务，验证服务正常启动并自动创建新数据库文件。

**Acceptance Scenarios**:

1. **Given** SQLite 文件不存在或被删除，**When** 服务启动，**Then** 系统自动创建新数据库文件和表结构，记录警告日志。
2. **Given** SQLite 文件已存在且包含数据，**When** 服务启动，**Then** 系统加载最近 maxDays 内的数据到内存，不覆盖已有数据。
3. **Given** 配置文件指定了 SQLite 文件路径和最大保留天数，**When** 超过保留期的数据存在，**Then** 系统在定时清理中自动删除过期数据。

---

### Edge Cases

- SQLite 写入失败（磁盘满、权限不足）时，如何保证内存数据不丢失且服务不崩溃？
- 数据库文件在服务运行中被外部修改或损坏，如何处理？
- 服务启动时加载大量历史数据（如 90 天 × 6 个指数），启动时间如何控制？
- 内存和 SQLite 数据不一致时（如写入 SQLite 中途崩溃），以哪个为准？

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 在服务启动时自动检测 SQLite 数据库文件是否存在；不存在时自动创建文件和表结构。
- **FR-002**: 系统 MUST 在服务启动时从 SQLite 加载最近 maxDays 天（与 HistoryStore 内存保留期一致）的历史数据到内存；更早数据保留在 SQLite 中按需查询。
- **FR-003**: 系统 MUST 在每次定时采集写入内存后，将新增的行情记录同步写入 SQLite（采集后立即写入）。
- **FR-004**: 系统 MUST 支持通过配置指定 SQLite 文件路径、最大数据保留天数和是否启用。
- **FR-005**: 系统 MUST 定期清理 SQLite 中超过保留天数的过期数据。
- **FR-006: 历史查询接口 MUST 优先使用内存数据；当请求范围超出内存保留期时回退到 SQLite 查询，合并两个来源返回完整结果（内存优先于 SQLite）。
- **FR-007**: SQLite 写入失败 MUST NOT 导致服务崩溃或影响数据采集和 API 的正常运行；仅记录错误日志并跳过本次写入。
- **FR-008**: SQLite 文件损坏时 MUST 自动重建（删除旧文件、创建新文件），服务继续正常运行。

### Key Entities

- **历史行情记录 (HistoricalQuote)**: 指数代码、交易日期、开盘价、收盘价、最高价、最低价、成交量、涨跌幅——与现有 `IndexQuote` 结构一致。
- **持久化配置 (PersistenceConfig)**: SQLite 文件路径、最大保留天数、是否启用。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 服务重启后，内存保留期内（默认 30 天）的历史数据在 3 秒内可查询（含 SQLite 加载时间）。
- **SC-002**: 服务重启后数据丢失率为 0%（所有在重启前已写入 SQLite 的数据均可恢复）。
- **SC-003**: SQLite 写入操作不影响采集周期的稳定性（写入耗时 < 采集间隔的 10%）。
- **SC-004**: SQLite 文件损坏后服务仍正常启动，历史查询返回内存中可用数据或从重建的空库返回空结果。

## Assumptions

- SQLite 文件默认路径为 `data/market_history.db`，可通过配置调整。
- 默认最大保留天数为 90 天（SQLite 层），内存仅加载最近 30 天（与 HistoryStore maxDays 对齐）。
- 数据刷写策略采用"采集后立即写入"，简化一致性保证。
- SQLite 数据格式与 `IndexQuote` 结构体字段一一对应，以交易日期 + 指数代码作为联合唯一键。
- 本功能属于 Phase 2 延续，需要先完成 constitution amendment 来添加存储原则（当前 constitution 禁止数据库依赖）。
- 仅持久化 HistoryStore（市场行情历史数据），不涉及 notify 推送历史等其他内存数据。
- `github.com/mattn/go-sqlite3` 已作为项目间接依赖存在（go.mod 中可见），可直接使用。
