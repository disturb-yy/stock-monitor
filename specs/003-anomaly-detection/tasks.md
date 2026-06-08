# Tasks: 异动检测规则引擎

**Input**: Design documents from `specs/003-anomaly-detection/`

**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Not explicitly requested — test tasks omitted.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 配置、依赖、领域脚手架

- [x] T001 Create `domain/anomaly/` directory structure
- [x] T002 Add `anomaly` configuration structs to `pkg/config/config.go`
- [x] T003 [P] Add `anomaly` section to `configs/config.yaml` with default rules

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 核心引擎和事件模型，所有 User Story 的前置依赖

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Create `AnomalyEvent` struct and `AnomalyType` constants in `domain/anomaly/event.go`
- [x] T005 Create `Detector` interface in `domain/anomaly/rule.go`
- [x] T006 Create `Engine` struct with detector registration and `Detect` method in `domain/anomaly/engine.go`
- [x] T007 Create `HTTPHandler` with `GetAnomalies` endpoint skeleton in `domain/anomaly/http_handler.go`

**Checkpoint**: Engine + event model ready, detectors can be plugged in

---

## Phase 3: User Story 1 - 涨跌幅超阈值 (Priority: P1) 🎯 MVP

**Goal**: 指数涨跌幅超过配置阈值时自动生成异动事件

**Independent Test**: 调低阈值至 0.01%，请求 anomalies 接口验证返回 price_up/price_down 事件

### Implementation for User Story 1

- [x] T008 [US1] Implement `PriceChangeDetector` in `domain/anomaly/price_change.go`
- [x] T009 [US1] Register `PriceChangeDetector` in `Engine` constructor in `domain/anomaly/engine.go`
- [x] T010 [US1] Implement `GetAnomalies` handler: query market indices, run engine, return events in `domain/anomaly/http_handler.go`

**Checkpoint**: US1 complete — 涨跌幅异动可检测并查询

---

## Phase 4: User Story 2 - 成交量突增 (Priority: P1) 🎯 MVP

**Goal**: 成交量超过近 N 日均值 M 倍时生成放量异动事件

**Independent Test**: 某指数当日成交量远超均值，验证返回 volume_spike 事件

### Implementation for User Story 2

- [x] T011 [US2] Implement `VolumeSpikeDetector` with historical volume query via Tushare in `domain/anomaly/volume_spike.go`
- [x] T012 [US2] Register `VolumeSpikeDetector` in `Engine` constructor in `domain/anomaly/engine.go`
- [x] T013 [US2] Handle edge case: insufficient historical data (< N days) → skip detection in `domain/anomaly/volume_spike.go`

**Checkpoint**: US1 + US2 complete — 涨跌幅 + 成交量双重检测

---

## Phase 5: User Story 3 - 连续涨跌 (Priority: P2)

**Goal**: 指数连续 N 日同向变动时生成连涨/连跌异动事件

**Independent Test**: 某指数连续多日上涨，验证返回 consecutive_up 事件

### Implementation for User Story 3

- [x] T014 [US3] Implement `ConsecutiveDetector` with historical close price query via Tushare in `domain/anomaly/consecutive.go`
- [x] T015 [US3] Register `ConsecutiveDetector` in `Engine` constructor in `domain/anomaly/engine.go`
- [x] T016 [US3] Handle edge case: holiday gap → skip missing days, detect consecutive TRADING days in `domain/anomaly/consecutive.go`

**Checkpoint**: US1-3 complete — 三种检测规则全部可用

---

## Phase 6: User Story 4 - 异动事件查询 (Priority: P2)

**Goal**: 独立 API 端点返回当前所有异动事件列表

**Independent Test**: `GET /api/market/anomalies` 返回结构化事件 JSON 数组

### Implementation for User Story 4

- [x] T017 [US4] Register `GET /api/market/anomalies` route in `api/route.go`
- [x] T018 [US4] Wire `Engine` into Composition Root (`cmd/server/main.go`) and inject into `anomaly.HTTPHandler`
- [x] T019 [US4] Implement handler response formatting per `contracts/anomalies-api.md` in `domain/anomaly/http_handler.go`

**Checkpoint**: All stories complete — 独立接口可用，三种规则运行中

---

## Phase 7: Polish & Cross-Cutting Concerns

- [x] T020 [P] Create `domain/anomaly/INDEX.md` with module documentation
- [x] T021 [P] Add `data/trading_calendar.json` to `.gitignore` (if not already)
- [x] T022 Run `make build` to verify compilation
- [x] T023 Run quickstart.md validation scenarios (adjust thresholds, verify events)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Phase 1 (config structs for rule creation)
- **US1 (Phase 3)**: Depends on Phase 2 (engine + event model)
- **US2 (Phase 4)**: Depends on Phase 2 (independent of US1, but same file engine.go)
- **US3 (Phase 5)**: Depends on Phase 2 (same as US2)
- **US4 (Phase 6)**: Depends on US1 (endpoint wiring needs engine to work)
- **Polish (Phase 7)**: Depends on all phases

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational — needs engine.go ready for registration
- **US2 (P1)**: Can start after Foundational — independent detector implementation
- **US3 (P2)**: Can start after Foundational — independent detector implementation
- **US4 (P2)**: Depends on at least US1 completed (engine functional before endpoint exposure)

### Within Each User Story

- Detector implementation before engine registration
- Core logic before edge case handling
- Story complete before moving to next

### Parallel Opportunities

- T002 and T003 can run in parallel (different files)
- US1, US2, US3 detectors are independent files → can be implemented in parallel
- T020 and T021 can run in parallel (different files)

---

## Parallel Example: Detectors (US1 + US2 + US3)

```bash
# All three detectors are independent files — implement in parallel:
Task: "Implement PriceChangeDetector in domain/anomaly/price_change.go"
Task: "Implement VolumeSpikeDetector in domain/anomaly/volume_spike.go"
Task: "Implement ConsecutiveDetector in domain/anomaly/consecutive.go"

# Then register all in engine.go:
Task: "Register PriceChangeDetector in Engine constructor"
Task: "Register VolumeSpikeDetector in Engine constructor"
Task: "Register ConsecutiveDetector in Engine constructor"
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Phase 1: Setup → config ready
2. Phase 2: Foundational → engine skeleton
3. Phase 3: US1 → 涨跌幅检测 working
4. Phase 6 (partial): Wire endpoint → anomalies API live
5. **STOP and VALIDATE**: `curl /api/market/anomalies`
6. Deploy — 涨跌幅异动检测已可用

### Incremental Delivery

1. Setup + Foundational → engine framework
2. Add US1 → Price change detection (MVP!)
3. Add US2 → Volume spike detection
4. Add US3 → Consecutive days detection
5. Add US4 → Polished endpoint + full wiring
6. Polish → Docs + build

---

## Notes

- DDF 关键约束: `domain/anomaly` 不导入 `domain/market`，不导入 `domain/auth`
- Engine 依赖 Tushare client 注入（由 Composition Root 提供），与 market package 独立
- HTTP handler 内部直接调用 `tushare.Client` 获取行情数据（不通过 market.Service）
- 每次请求实时检测，不缓存事件结果
