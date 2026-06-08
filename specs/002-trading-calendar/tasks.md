# Tasks: 交易日历接入

**Input**: Design documents from `specs/002-trading-calendar/`

**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Not explicitly requested — test tasks omitted.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Tushare API 扩展和配置准备

- [x] T001 Add `TradeCalItem` struct in `pkg/tushare/types.go`
- [x] T002 Add `TradeCal` method to Client in `pkg/tushare/client.go`
- [x] T003 [P] Add `calendar` configuration section to `pkg/config/config.go`
- [x] T004 [P] Add calendar defaults to `configs/config.yaml`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 交易日历缓存核心模块，所有 User Story 的前置依赖

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 Create `TradingCalendar` struct with `Days`, `StartDate`, `EndDate`, `UpdatedAt` in `domain/market/calendar.go`
- [x] T006 Implement `LoadFromFile` / `SaveToFile` JSON persistence in `domain/market/calendar.go`
- [x] T007 Implement `IsTradingDay(date) bool` lookup method in `domain/market/calendar.go`
- [x] T008 Implement `NeedsRefresh() bool` expiry check in `domain/market/calendar.go`
- [x] T009 Implement `FetchFromTushare` to populate calendar via `trade_cal` API in `domain/market/calendar.go`

**Checkpoint**: Calendar module self-contained — can load, query, and refresh independently

---

## Phase 3: User Story 1 - 节假日准确识别 (Priority: P1) 🎯 MVP

**Goal**: 法定节假日和调休日正确判为非交易日/交易日，替换纯时间判断

**Independent Test**: 查询一个已知节假日的日期，验证 `IsTradingDay` 返回 false

### Implementation for User Story 1

- [x] T010 [US1] Initialize `TradingCalendar` in `TushareProvider` constructor (`NewTushareProvider`) in `domain/market/tushare_provider.go`
- [x] T011 [US1] Replace weekday-only check in `determineMarketSession` with `calendar.IsTradingDay` in `domain/market/tushare_provider.go`
- [x] T012 [US1] Handle non-trading day: return `closed` status with `isTrading=false` in `domain/market/tushare_provider.go`

**Checkpoint**: US1 complete — holidays returned as closed, trading days correctly identified

---

## Phase 4: User Story 2 - 准确的 tradeDate (Priority: P1) 🎯 MVP

**Goal**: `tradeDate` 字段遇节假日向前追溯至最近交易日

**Independent Test**: 请求市场状态，验证 `tradeDate` 为最近交易日（非过去非交易日）

### Implementation for User Story 2

- [x] T013 [US2] Implement `LastTradeDate(date) (string, bool)` in `domain/market/calendar.go`
- [x] T014 [US2] Update `GetMarketStatus` to use `calendar.LastTradeDate` for `tradeDate` field in `domain/market/tushare_provider.go`
- [x] T015 [US2] Add fallback: when calendar unavailable, use `latestTradeDate` (existing API query) in `domain/market/tushare_provider.go`

**Checkpoint**: US1 + US2 complete — accurate market status + tradeDate for all calendar scenarios

---

## Phase 5: User Story 3 - 日历可配置更新 (Priority: P2)

**Goal**: 日历自动拉取更新 + 本地缓存降级

**Independent Test**: 修改缓存过期时间，重启验证触发自动更新

### Implementation for User Story 3

- [x] T016 [US3] Implement auto-refresh on startup when `NeedsRefresh()` in `domain/market/calendar.go`
- [x] T017 [US3] Add graceful degradation: use stale cache + log warning when Tushare unreachable in `domain/market/calendar.go`
- [x] T018 [US3] Add expiry warning log when cache is older than `update_interval_days` in `domain/market/calendar.go`

**Checkpoint**: All stories complete — calendar auto-updates with fallback

---

## Phase 6: Polish & Cross-Cutting Concerns

- [x] T019 [P] Update `domain/market/INDEX.md` with calendar module docs
- [x] T020 [P] Update `pkg/tushare/INDEX.md` with `TradeCal` method docs
- [x] T021 Add `data/trading_calendar.json` to `.gitignore`
- [x] T022 Run `make build` to verify compilation
- [x] T023 Run quickstart.md validation scenarios

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Phase 1 (TradeCal API needed for FetchFromTushare)
- **US1 (Phase 3)**: Depends on Phase 2 (calendar module)
- **US2 (Phase 4)**: Depends on Phase 3 (uses initialized calendar + US1 integration)
- **US3 (Phase 5)**: Depends on Phase 2 (extends calendar module)
- **Polish (Phase 6)**: Depends on all phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational
- **User Story 2 (P1)**: Depends on US1 (needs calendar initialized in TushareProvider)
- **User Story 3 (P2)**: Can start after Foundational (independent of US1/US2, extends calendar.go)

### Parallel Opportunities

- T003 and T004 can run in parallel (different files)
- T019 and T020 can run in parallel (different INDEX.md files)
- US3 can run in parallel with US1+US2 (different concerns in calendar.go)

---

## Implementation Strategy

### MVP First (US1 + US2)

1. Phase 1: Setup → Tushare trade_cal API ready
2. Phase 2: Foundational → Calendar module works
3. Phase 3: US1 → Holidays correctly identified
4. Phase 4: US2 → Accurate tradeDate
5. **STOP and VALIDATE**: Test with known holidays
6. Deploy — market status now accurate year-round

### Incremental Delivery

1. Setup + Foundational → Calendar loads from Tushare
2. Add US1 → Holiday detection works
3. Add US2 → tradeDate always correct
4. Add US3 → Auto-refresh + cache fallback
5. Polish → Docs + validation

---

## Notes

- Calendar JSON file size: ~365 entries per year, < 10KB
- Tushare `trade_cal` API free tier: same 200/min limit as `index_daily`
- MockProvider not affected — has its own independent GetMarketStatus implementation
- `determineMarketSession` retains time-of-day logic; only weekday check replaced
