# Tasks: SQLite 持久化

**Input**: Design documents from `specs/006-sqlite-persistence/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Not explicitly requested. Core logic validated via quickstart manual tests.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Dependency and configuration scaffolding

- [x] T001 Add `github.com/mattn/go-sqlite3` dependency to `go.mod` via `go get`
- [x] T002 Add `PersistenceConfig` struct to `pkg/config/config.go` and wire into `Config` + `defaultConfig()`
- [x] T003 Add `persistence` section with defaults to `configs/config.yaml`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: SQLite 核心存储层 — 建表、读写基础操作

**⚠️ CRITICAL**: US1 依赖此阶段

- [x] T004 Create `SQLiteStore` struct with `Open`, `Close`, `createTable` in `domain/market/sqlite_store.go`
- [x] T005 Add `LoadRecent(maxDays int)` method to SQLiteStore in `domain/market/sqlite_store.go`
- [x] T006 [P] Add `Query(symbol, start, end string)` method to SQLiteStore in `domain/market/sqlite_store.go`
- [x] T007 [P] Add `Flush(quotes []IndexQuote)` method (INSERT OR REPLACE batch write) to SQLiteStore in `domain/market/sqlite_store.go`

**Checkpoint**: SQLite 表已创建，能写入和查询

---

## Phase 3: User Story 1 — 重启后数据恢复 (Priority: P1) 🎯 MVP

**Goal**: 启动时从 SQLite 加载最近 maxDays 天数据到内存；采集后写入 SQLite

**Independent Test**: 采集一批数据 → 重启 → `/api/market/history` 返回重启前数据

### Implementation for User Story 1

- [x] T008 [US1] Integrate `Flush` call into `Collector.collect()` after `Append` loop in `domain/market/collector.go`
- [x] T009 [US1] Add `LoadFromSQLite(ctx, st *SQLiteStore) error` method to `HistoryStore` in `domain/market/collector.go`
- [x] T010 [US1] Wire `SQLiteStore` init + `LoadFromSQLite` call into startup sequence in `cmd/server/main.go`

**Checkpoint**: US1 complete — 重启后数据可恢复

---

## Phase 4: User Story 2 — 超内存范围 SQLite 回退查询 (Priority: P2)

**Goal**: 请求超出内存覆盖范围时，自动回退到 SQLite 查询并合并结果（内存优先）

**Independent Test**: 内存保留 7 天，查询 10 天前数据，验证从 SQLite 返回

### Implementation for User Story 2

- [x] T011 [US2] Modify `HTTPHandler.GetHistory` to call SQLiteStore.Query when request range exceeds memory in `domain/market/http_handler.go`
- [x] T012 [US2] Add SQLiteStore field to `HTTPHandler` struct and update `NewHTTPHandler` constructor in `domain/market/http_handler.go`
- [x] T013 [US2] Wire `SQLiteStore` into `HTTPHandler` in `cmd/server/main.go`

**Checkpoint**: US2 complete — 跨边界查询正常合并返回

---

## Phase 5: User Story 3 — 文件管理与损坏恢复 (Priority: P3)

**Goal**: 过期数据自动清理；文件损坏时自动重建不影响服务

**Independent Test**: 损坏 SQLite 文件 → 重启 → 服务正常启动

### Implementation for User Story 3

- [x] T014 [US3] Add `PurgeBefore(maxDays int)` method to SQLiteStore in `domain/market/sqlite_store.go`
- [x] T015 [US3] Integrate `PurgeBefore` call into `Collector.collect()` after `Flush` in `domain/market/collector.go`
- [x] T016 [US3] Add file corruption detection in `Open()` (PRAGMA integrity_check + auto-rebuild) in `domain/market/sqlite_store.go`

**Checkpoint**: US3 complete — 过期清理 + 损坏重建均正常工作

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and validation

- [x] T017 [P] Update `domain/market/INDEX.md` to document `SQLiteStore` and persistence config
- [x] T018 [P] Update root `INDEX.md` to note SQLite persistence under Phase 2 status
- [x] T019 Run `go test ./domain/market/...` and validate per `quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖
- **Foundational (Phase 2)**: 依赖 Phase 1 — BLOCKS 所有 user stories
- **US1 (Phase 3)**: 依赖 Phase 2 — MVP
- **US2 (Phase 4)**: 依赖 US1（需要 SQLiteStore 和 Load 逻辑就绪）
- **US3 (Phase 5)**: 依赖 US1（需要 Flush 集成就绪）
- **Polish (Phase 6)**: 依赖所有阶段

### User Story Dependencies

- **US1**: Standalone MVP
- **US2**: 依赖 US1（SQLiteStore 已创建并注入）
- **US3**: 依赖 US1（Flush 已集成到 Collector）
- **US2 和 US3 可并行开发**（修改不同文件）

### Within Each Phase

- Phase 2: T004 必须最先（创建 struct），T005-T007 可并行
- Phase 3: T008 → T009 → T010（顺序依赖）
- Phase 4: T011 → T012 → T013（顺序依赖）

### Parallel Opportunities

- T005 + T006 + T007：三个 SQLiteStore 方法，不同函数，可并行
- T017 + T018：两个 INDEX.md 不同文件，可并行
- US2 + US3：修改不同文件，US1 完成后可并行

---

## Parallel Example: Phase 2 Foundational

```bash
# SQLiteStore 方法可并行开发：
Task T005: "Add LoadRecent to domain/market/sqlite_store.go"
Task T006: "Add Query to domain/market/sqlite_store.go"
Task T007: "Add Flush to domain/market/sqlite_store.go"
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Phase 1: Setup → go-sqlite3 + config
2. Phase 2: Foundational → SQLiteStore 基本读写
3. Phase 3: US1 → 重启恢复
4. **STOP**: 验证重启后 `/api/market/history` 返回数据

### Incremental Delivery

1. Setup + Foundational + US1 → **MVP!** 重启不丢数据
2. +US2 → 能查超出内存范围的历史数据
3. +US3 → 过期清理 + 损坏自愈

---

## Notes

- 所有 Go 源码使用中文注释（constitution VI）
- SQLiteStore 创建在 `domain/market/` 内，不新增 domain 包
- `database/sql` + `github.com/mattn/go-sqlite3`，无额外第三方 ORM
- 单测可使用 `:memory:` DSN，无需文件系统依赖
