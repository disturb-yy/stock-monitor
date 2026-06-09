# Tasks: 定时采集

**Input**: Design documents from `specs/004-scheduled-collection/`

**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Not explicitly requested — test tasks omitted.

## Format: `[ID] [P?] [Story] Description`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 配置结构、依赖准备

- [ ] T001 Add `CollectorConfig` struct to `pkg/config/config.go`
- [ ] T002 [P] Add `collector` section to `configs/config.yaml`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 内存存储 + 采集器核心循环，所有 User Story 前置依赖

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T003 Create `HistoryStore` struct with `sync.RWMutex` map in `domain/market/collector.go`
- [ ] T004 Implement `Append(symbol, quote)` with same-date dedup in `domain/market/collector.go`
- [ ] T005 Implement `Trim(maxDays)` eviction in `domain/market/collector.go`
- [ ] T006 Create `Collector` struct with ticker + done channel in `domain/market/collector.go`
- [ ] T007 Implement `Start(ctx)` background loop in `domain/market/collector.go`

**Checkpoint**: Store + Collector ready — can be started in background

---

## Phase 3: User Story 1 - 交易时段自动采集 (Priority: P1) 🎯 MVP

**Goal**: 交易时段自动定时拉取行情，非交易时段休眠

**Independent Test**: 交易时段启动，等待一个周期后 history 接口返回数据

### Implementation for User Story 1

- [ ] T008 [US1] Implement `collect(ctx)` — fetch all indices, store to HistoryStore in `domain/market/collector.go`
- [ ] T009 [US1] Add trading-day check via `TradingCalendar` before collect in `domain/market/collector.go`
- [ ] T010 [US1] Add session-time check (skip pre_open/lunch_break/closed) in `domain/market/collector.go`
- [ ] T011 [US1] Implement `Stop()` graceful shutdown in `domain/market/collector.go`

**Checkpoint**: US1 complete — collector runs autonomously during trading hours

---

## Phase 4: User Story 2 - 采集间隔可配置 (Priority: P1) 🎯 MVP

**Goal**: 采集间隔通过 config.yaml 调整，无需改代码

**Independent Test**: 修改 interval_minutes 为 1，观察两次采集时间差

### Implementation for User Story 2

- [ ] T012 [US2] Read `CollectorConfig` fields (Enabled, IntervalMinutes, MaxHistoryDays) in `domain/market/collector.go`
- [ ] T013 [US2] Handle disabled state (Enabled=false → skip Start) in `domain/market/collector.go`
- [ ] T014 [US2] Wire collector creation + lifecycle in `cmd/server/main.go`

**Checkpoint**: US1+US2 complete — collector configurable, auto-starts on server boot

---

## Phase 5: User Story 3 - 历史行情查询 (Priority: P2)

**Goal**: API 接口按日期范围查询已采集的历史数据

**Independent Test**: 采集运行后请求 `/api/market/history?start=...` 返回数据

### Implementation for User Story 3

- [ ] T015 [P] [US3] Implement `Query(symbol, start, end)` in `HistoryStore` in `domain/market/collector.go`
- [ ] T016 [US3] Add `GetHistory` handler to `market.HTTPHandler` in `domain/market/http_handler.go`
- [ ] T017 [US3] Register `GET /api/market/history` route in `api/route.go`
- [ ] T018 [US3] Pass `HistoryStore` to `HTTPHandler` (inject via constructor) in `domain/market/http_handler.go`

**Checkpoint**: All stories complete — collector + history API fully operational

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T019 Update `domain/market/INDEX.md` with collector + history store docs
- [ ] T020 Run `make build` to verify compilation
- [ ] T021 Run quickstart.md validation scenarios

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Phase 1 (CollectorConfig needed)
- **US1 (Phase 3)**: Depends on Phase 2 (Collector + Store ready)
- **US2 (Phase 4)**: Depends on Phase 2 (Collector configured) + US1 (collect logic done)
- **US3 (Phase 5)**: Depends on Phase 2 (Store ready) — can run parallel with US1/US2
- **Polish (Phase 6)**: Depends on all phases

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational
- **US2 (P1)**: Depends on US1 (needs Collector structure before wiring config + lifecycle)
- **US3 (P2)**: Can start after Foundational — independent of US1/US2 (needs Store, not Collector)

### Parallel Opportunities

- T001 and T002 (different files)
- US3 can run in parallel with US1+US2 (different concerns: query vs collect)
- T015 can start once Store (T003-T005) is done, before Collector

---

## Implementation Strategy

### MVP First (US1 + US2)

1. Phase 1: Setup → config ready
2. Phase 2: Foundational → Store + Collector skeleton
3. Phase 3: US1 → trading-hour auto-collect
4. Phase 4: US2 → config wiring + lifecycle
5. **STOP and VALIDATE**: check logs for auto-collect, query history via API
6. Deploy — autonomous data collection active

### Incremental Delivery

1. Setup + Foundational → Store + Collector skeleton
2. Add US1 → Auto-collect during trading hours
3. Add US2 → Config-driven intervals
4. Add US3 → History query API
5. Polish → Docs + build

---

## Notes

- Collector 仅 Tushare 模式有效；Mock 模式下 Start 直接 return
- HistoryStore 使用 RWMutex：collect 持写锁、API 查询持读锁
- 采集间隔最小 1 分钟（Tushare 免费版频率限制考量）
- 优雅关闭：Stop() 关闭 done channel → 等待当前采集完成（最多 30s）
