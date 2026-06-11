# Tasks: 每日收盘总结推送

**Input**: Design documents from `specs/007-daily-summary/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/daily-api.md ✅, quickstart.md ✅

**Tests**: 已列入（Builder、Scheduler、HTTPHandler 的单元测试），遵循 quickstart.md 中定义的测试场景。

**Organization**: 按 User Story 分组，支持增量交付。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行（不同文件，无依赖）
- **[Story]**: 归属 User Story（US1/US2/US3）

---

## Phase 1: Setup（项目初始化）

**Purpose**: 创建领域目录和基础骨架

- [x] T001 创建 `domain/daily/` 目录结构（空文件骨架）

---

## Phase 2: Foundational（基础设施，阻塞所有 User Story）

**Purpose**: 配置、数据模型、接口定义 — 所有 US 的前置依赖

**⚠️ CRITICAL**: 必须全部完成后才能开始 Phase 3+

- [x] T002 [P] 新增 `DailySummaryConfig` 到 `pkg/config/config.go`（Enabled bool + PushTime string）
- [x] T003 [P] 定义 `DataProvider` 接口和 `MockDataProvider` 在 `domain/daily/provider.go`
- [x] T004 定义 `DailySummary`、`IndexSnapshot`、`AnomalySummary`、`AnomalyItem`、`DataStatus` 模型在 `domain/daily/summary.go`

**Checkpoint**: 模型和接口就绪 — 可以开始各 User Story

---

## Phase 3: User Story 1 — 收盘后自动收到当日总结 (Priority: P1) 🎯 MVP

**Goal**: 每个交易日下午 15:15 自动生成收盘总结并推送到企业微信

**Independent Test**: Mock 模式下等待定时触发（或手动加速时钟），验证 Builder 生成正确总结、Scheduler 触发推送、Markdown 格式化、POST 手动触发端点可用。

### US1 实现

- [x] T005 [US1] 实现 `Builder`（从 DataProvider 构建 DailySummary）在 `domain/daily/builder.go`
  - 从 DataProvider 获取当日行情数据
  - 计算涨/跌/平数量统计
  - 识别领涨和领跌指数
  - 处理数据异常（标注 DataStatus）
- [x] T006 [US1] 实现 `Builder` 单元测试在 `domain/daily/builder_test.go`
  - `TestBuilder_BuildSummary_Success` — 正常数据
  - `TestBuilder_BuildSummary_PartialData` — 部分缺失
  - `TestBuilder_BuildSummary_NoData` — 全部缺失
- [x] T007 [US1] 实现 Markdown 格式化（DailySummary → 企微消息文本）在 `domain/daily/markdown.go`
  - 逐个列出指数名称和涨跌幅
  - 涨跌统计汇总行 + 领涨/领跌
  - 异动事件摘要（≤5 条，超出显示"等 N 条"）
  - 数据异常时添加 ⚠️ 标注
- [x] T008 [US1] 实现 Markdown 格式化单元测试在 `domain/daily/markdown_test.go`
- [x] T009 [US1] 实现 `Scheduler`（time.Ticker + 交易日历）在 `domain/daily/scheduler.go`
  - 每 60s 检查是否到达推送时间
  - 调用交易日历判断是否为交易日
  - 到达时调用 Builder → Markdown → PushFunc
  - 推送失败记录日志但不影响下次调度
- [x] T010 [US1] 实现 `Scheduler` 单元测试在 `domain/daily/scheduler_test.go`
  - `TestScheduler_TradingDay` — 交易日触发
  - `TestScheduler_NonTradingDay` — 非交易日跳过
  - `TestScheduler_PushFailure` — 推送失败不影响调度
- [x] T011 [US1] 实现 `HTTPHandler.PostSummary`（手动触发推送）在 `domain/daily/http_handler.go`
  - 接受 POST 请求，异步触发 Builder → Markdown → PushFunc
  - 立即返回 `{"triggered": true}`
  - 非交易日返回 `{"triggered": false}`
  - 推送未启用时返回错误
- [x] T012 [US1] 实现 `HTTPHandler` 单元测试在 `domain/daily/http_handler_test.go`
  - `TestPostSummary_Success` — 正常触发
  - `TestPostSummary_NonTradingDay` — 非交易日
  - `TestPostSummary_WebhookDisabled` — 推送未启用
- [x] T013 [US1] 创建 `domain/daily/INDEX.md` 领域索引文档

**Checkpoint**: Mock 模式下 POST 手动触发成功，Scheduler 单元测试全部通过 — MVP 可交付

---

## Phase 4: User Story 2 — 总结内容清晰完整 (Priority: P2)

**Goal**: 收盘总结 Markdown 包含完整的涨跌统计、领涨/领跌、异动摘要信息

**Independent Test**: 查询推送历史中的收盘总结记录，验证 Markdown 内容包含各指数涨跌幅列表、涨跌统计、领涨/领跌指数、异动事件摘要。

> **注**: US2 的内容格式化逻辑已在 US1 的 `markdown.go` 中实现（T007）。本阶段为验证和增强。

- [x] T014 [US2] 验证 Markdown 输出包含所有必需要素：逐指数涨跌幅、涨跌统计、领涨/领跌、异动摘要、时间戳
- [x] T015 [US2] 增强异动摘要截断逻辑：超过 5 条时仅展示前 5 条 + "等 N 条"

**Checkpoint**: 手动触发推送后企业微信收到的 Markdown 消息内容完整

---

## Phase 5: User Story 3 — 推送记录可追溯 (Priority: P3)

**Goal**: 提供 GET 查询接口返回当日收盘总结 JSON，推送历史可通过现有 `alerts/history` 接口查询

**Independent Test**: 调用 GET /api/market/daily-summary 验证返回完整 JSON 结构（含涨跌统计、指数列表、异动摘要），调用 GET /api/market/alerts/history 验证收盘总结推送记录存在。

- [x] T016 [US3] 实现 `HTTPHandler.GetSummary`（GET 查询）在 `domain/daily/http_handler.go`
  - 支持 `?date=YYYY-MM-DD` 参数，默认当日
  - 返回 DailySummary JSON（或空）给非交易日返回 `is_trading_day: false`
  - 返回结构符合 `contracts/daily-api.md` 定义
- [x] T017 [US3] 实现 GET 查询单元测试在 `domain/daily/http_handler_test.go`
  - `TestGetSummary_Success` — 正常返回
  - `TestGetSummary_NonTradingDay` — 非交易日
  - `TestGetSummary_SpecificDate` — 指定日期查询
- [x] T018 [US3] 确认收盘总结推送记录正确写入现有 `notify.HistoryBuffer`（通过 PushFunc 适配器）

**Checkpoint**: GET 查询返回正确 JSON，推送历史中可查到收盘总结记录

---

## Phase 6: Integration & Polish（集成与收尾）

**Purpose**: 将 daily 领域接入 Composition Root，注册路由，更新配置

- [x] T019 在 `cmd/server/main.go` 中组装 daily 领域依赖（`setupDaily` 函数）
  - 注入 `market.HistoryStore`（实现 DataProvider）
  - 注入 `notify.Dispatcher`（实现 PushFunc）
  - 创建 `daily.Scheduler` 并启动
  - 优雅关闭时停止 Scheduler
- [x] T020 在 `api/route.go` 中注册 daily 路由
  - `GET /api/market/daily-summary` → dailyHandler.GetSummary
  - `POST /api/market/daily-summary` → dailyHandler.PostSummary
- [x] T021 在 `configs/config.yaml` 中新增 `daily_summary` 配置段
  ```yaml
  daily_summary:
    enabled: false
    push_time: "15:15"
  ```
- [x] T022 更新根目录 `INDEX.md`，新增 `domain/daily/` 索引引用和 API 表格条目
- [x] T023 运行 `quickstart.md` 全部验证步骤

**Checkpoint**: 完整功能端到端可运行

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: 无依赖 — 立即开始
- **Phase 2 (Foundational)**: 依赖 Phase 1 — 阻塞所有 User Story
- **Phase 3 (US1 MVP)**: 依赖 Phase 2 — 🎯 MVP 目标
- **Phase 4 (US2 内容)**: 依赖 Phase 3（US1 已实现 markdown.go）
- **Phase 5 (US3 查询)**: 依赖 Phase 2 + Phase 3（HTTPHandler 已在 US1 创建）
- **Phase 6 (集成)**: 依赖 Phase 3-5 全部完成

### User Story Dependencies

- **US1 (P1)**: Phase 2 → 开始 — 无其他 US 依赖
- **US2 (P2)**: Phase 3 → 开始 — 增强 US1 的 markdown.go
- **US3 (P3)**: Phase 2 + Phase 3 → 开始 — 扩展 US1 的 HTTPHandler

### Parallel Opportunities

- Phase 2 内：T002 和 T003 可并行（config vs provider，不同文件）
- Phase 3 内：T005+T007（Builder + Markdown）可并行，它们依赖相同接口但实现独立
- Phase 3 内：T006+T008+T010+T012（所有测试）完成后可并行
- Phase 6 内：T019+T020+T021（main.go + route.go + config.yaml）可并行

---

## Parallel Example: Phase 3 (US1 Core)

```bash
# Step 1: 并行创建 Builder 和 Markdown（独立模块）
Task T005: "实现 Builder 在 domain/daily/builder.go"
Task T007: "实现 Markdown 格式化 在 domain/daily/markdown.go"

# Step 2: 并行编写所有测试
Task T006: "Builder 单元测试 在 domain/daily/builder_test.go"
Task T008: "Markdown 单元测试 在 domain/daily/markdown_test.go"

# Step 3: Scheduler 和 HTTPHandler（依赖 Builder）
Task T009: "实现 Scheduler 在 domain/daily/scheduler.go"
Task T011: "实现 HTTPHandler.PostSummary 在 domain/daily/http_handler.go"

# Step 4: Scheduler 和 HTTPHandler 测试
Task T010: "Scheduler 单元测试 在 domain/daily/scheduler_test.go"
Task T012: "HTTPHandler 单元测试 在 domain/daily/http_handler_test.go"
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Phase 1: Setup (T001)
2. Phase 2: Foundational (T002-T004)
3. Phase 3: US1 (T005-T013)
4. **STOP & VALIDATE**: Mock 模式下 POST 手动触发成功 ✅
5. 可交付 MVP

### Incremental Delivery

1. Setup + Foundational → 基础就绪
2. US1 → 自动推送 + 手动触发 → **MVP 交付** 🎯
3. US2 → 内容完整性增强 → 交付
4. US3 → 查询 API → 交付
5. Phase 6 → 完整集成 → 上线

---

## Notes

- 共 23 个任务，分布于 6 个 Phase
- US1 (Phase 3) 占 9 个任务，为核心 MVP
- `daily` 领域不导入其他 domain，跨域依赖通过 main.go 注入接口
- PushFunc 适配器在 main.go 中将 DailySummary → Markdown → notify.Dispatcher.Dispatch
- Scheduler 使用 `time.Ticker`（60s），与现有 Collector 模式一致
- 时区统一使用 `Asia/Shanghai`
