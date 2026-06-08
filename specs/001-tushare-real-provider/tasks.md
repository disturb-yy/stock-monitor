# Tasks: Tushare 真实行情接入

**Input**: Design documents from `specs/001-tushare-real-provider/`

**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Not explicitly requested in spec — test tasks omitted.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

DDF 结构: `domain/{domain}/`, `pkg/`, `configs/`, `cmd/server/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 配置结构和依赖准备

- [x] T001 Create TushareConfig and IndexConfig structs in `domain/market/config.go`
- [x] T002 Create ProviderConfig struct with type field in `domain/market/config.go`
- [x] T003 [P] Update `pkg/config/config.go` to parse `provider` and `tushare` YAML sections
- [x] T004 [P] Add default tushare configuration block to `configs/config.yaml`
- [x] T005 Add sync.WaitGroup-based concurrent query support (stdlib only, no external dependency)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Composition Root 改造，支持 Mock/Tushare 切换

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T006 Update `cmd/server/main.go` to create Tushare client and TushareProvider based on config
- [x] T007 Update `cmd/server/main.go` to wire the selected provider (mock or tushare) into market.Service
- [x] T008 Verify existing MockProvider path still works after Composition Root refactor (`make run` with `provider.type: mock`)

**Checkpoint**: Foundation ready — provider switching works, mock path unchanged

---

## Phase 3: User Story 1 - 查询真实指数行情 (Priority: P1) 🎯 MVP

**Goal**: GetIndexQuotes 返回来自 Tushare 的真实 A 股指数行情数据

**Independent Test**: `GET /api/market/indices` 返回 6 个指数完整字段，值与公开行情一致

### Implementation for User Story 1

- [x] T009 [P] [US1] Implement index name lookup table (Symbol → Name) in `domain/market/tushare_provider.go`
- [x] T010 [P] [US1] Implement IndexDailyItem → IndexQuote mapping function in `domain/market/tushare_provider.go`
- [x] T011 [P] [US1] Implement helper to fetch previous day close for PreClose in `domain/market/tushare_provider.go`
- [x] T012 [US1] Implement GetIndexQuotes with goroutine concurrent queries via errgroup in `domain/market/tushare_provider.go`
- [x] T013 [US1] Add partial-failure error aggregation for GetIndexQuotes in `domain/market/tushare_provider.go` (return successful results + errors for failed indices)

**Checkpoint**: At this point, User Story 1 should be fully functional — real index data via API

---

## Phase 4: User Story 2 - 查询真实市场状态 (Priority: P1) 🎯 MVP

**Goal**: GetMarketStatus 基于服务器时间返回准确的 A 股交易状态

**Independent Test**: `GET /api/market/status` 在不同时段返回正确的 MarketStatus

### Implementation for User Story 2

- [x] T014 [US2] Implement market session time-based determination logic in `domain/market/tushare_provider.go`
- [x] T015 [US2] Implement GetMarketStatus with timezone-aware time check (Asia/Shanghai) in `domain/market/tushare_provider.go`
- [x] T016 [US2] Handle weekend detection and fallback trade_date from latest IndexDaily in `domain/market/tushare_provider.go`

**Checkpoint**: At this point, Users Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - 查询大盘总览 (Priority: P2)

**Goal**: GetMarketOverview 聚合市场状态 + 指数行情 + 涨跌统计

**Independent Test**: `GET /api/market/overview` 返回包含 status, indices, summary 的完整快照

### Implementation for User Story 3

- [x] T017 [P] [US3] Implement MarketSummary calculation (rising/falling/flat counts + total amount) in `domain/market/tushare_provider.go`
- [x] T018 [US3] Implement GetMarketOverview by composing GetMarketStatus + GetIndexQuotes in `domain/market/tushare_provider.go`

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 文档同步和最终验证

- [x] T019 [P] Update `domain/market/INDEX.md` with TushareProvider implementation status and usage
- [x] T020 [P] Update `pkg/tushare/INDEX.md` with any new methods or usage notes
- [x] T021 Run `make build` and `make test` to verify compilation and existing tests pass
- [x] T022 Run quickstart.md validation scenarios (mock → tushare → mock switch)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (config structs needed for Composition Root)
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - US1 and US2 can proceed in parallel (different methods in same file but independent logic)
  - US3 depends on US1 and US2 (aggregates their results)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational — No dependencies on US2 or US3
- **User Story 2 (P1)**: Can start after Foundational — No dependencies on US1 or US3
- **User Story 3 (P2)**: Depends on US1 and US2 (calls GetMarketStatus and GetIndexQuotes internally)

### Within Each User Story

- Mapping helpers before main implementation
- Core implementation before error handling
- Story complete before moving to next priority

### Parallel Opportunities

- T003 and T004 can run in parallel (different files)
- T009, T010, T011 can run in parallel (different functions in same file)
- US1 and US2 can be worked on in parallel (different methods, same file)
- T017 can run in parallel once US1 is complete
- T019 and T020 can run in parallel (different INDEX.md files)

---

## Parallel Example: User Story 1

```bash
# Launch all helper functions for US1 in parallel:
Task: "Implement index name lookup table in domain/market/tushare_provider.go"
Task: "Implement IndexDailyItem → IndexQuote mapping in domain/market/tushare_provider.go"
Task: "Implement PreClose fetcher in domain/market/tushare_provider.go"

# Then implement:
Task: "Implement GetIndexQuotes with errgroup concurrency in domain/market/tushare_provider.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 + 2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (real index data)
4. Complete Phase 4: User Story 2 (real market status)
5. **STOP and VALIDATE**: Test US1 and US2 independently
6. Deploy/demo if ready — real market data is already valuable

### Incremental Delivery

1. Setup + Foundational → provider switching works
2. Add User Story 1 → Test independently → Real index quotes (MVP!)
3. Add User Story 2 → Test independently → Real market status
4. Add User Story 3 → Test independently → Full market overview
5. Polish → docs + build validation

### Parallel Implementation

With US1 and US2 being independent (different methods), they can be implemented in parallel:
- Developer A: User Story 1 (GetIndexQuotes)
- Developer B: User Story 2 (GetMarketStatus)
- Developer C: User Story 3 (GetMarketOverview, after US1+US2)

---

## Notes

- [P] tasks = different concerns, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- All tasks operate within existing Provider interface — no interface changes
- MockProvider is preserved untouched — only referenced by Composition Root
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
