# Tasks: Webhook 告警推送

**Input**: Design documents from `specs/005-webhook-alerts/`

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Not explicitly requested — test tasks provided for critical components only.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: New domain scaffolding and configuration structure

- [x] T001 Create `domain/notify/` directory with INDEX.md per DDF conventions
- [x] T002 Add `WebhookConfig` struct to `pkg/config/config.go` and update `Config` struct and `defaultConfig()`
- [x] T003 Add `webhook` section with defaults to `configs/config.yaml`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core interfaces and contracts that ALL user stories depend on

**⚠️ CRITICAL**: No user story work until this phase is complete

- [x] T004 Define `WebhookSender` interface and `DeliveryStatus` type in `domain/notify/sender.go`
- [x] T005 [P] Implement `MockSender` (records calls for testing) in `domain/notify/mock_sender.go`
- [x] T006 [P] Add `EventCallback` type and `SetCallback` method to `domain/anomaly/engine.go`

**Checkpoint**: Foundation ready — interfaces and config in place, user stories can begin

---

## Phase 3: User Story 1 - 异动事件自动推送到企业微信 (Priority: P1) 🎯 MVP

**Goal**: 异动事件产生后自动通过企业微信 Webhook 推送 Markdown 告警消息

**Independent Test**: 配置企业微信 Webhook URL，触发异动检测，验证企微群收到 Markdown 消息

### Implementation for User Story 1

- [x] T007 [US1] Implement `WeComSender` (企业微信机器人 POST) in `domain/notify/sender.go`
- [x] T008 [P] [US1] Implement Markdown message builder (`BuildMessage`) in `domain/notify/message.go`
- [x] T009 [US1] Implement `Dispatcher` (异步推送、重试、合并多事件) in `domain/notify/dispatcher.go`
- [x] T010 [US1] Implement `Service` (Notifier 编排：接收回调 → 构建消息 → 调度推送) in `domain/notify/service.go`
- [x] T011 [US1] Wire notify domain in `cmd/server/main.go`: create WebhookSender from config, create Notifier, register callback to anomaly Engine
- [x] T012 [US1] Add startup/shutdown logs for notify domain in `cmd/server/main.go`

**Checkpoint**: US1 complete — 异动事件触发后企业微信群能收到 Markdown 告警

---

## Phase 4: User Story 2 - 告警去重与冷却 (Priority: P2)

**Goal**: 同一指数+同类型在冷却时间内仅推送一次，避免重复骚扰

**Independent Test**: 连续 3 次触发同一异动（间隔小于冷却时间），仅推送第 1 次

### Implementation for User Story 2

- [x] T013 [US2] Implement `CooldownTracker` (sync.Map 存储冷却记录) in `domain/notify/dedup.go`
- [x] T014 [US2] Integrate CooldownTracker into Dispatcher in `domain/notify/dispatcher.go`: 推送前检查冷却，成功后更新记录
- [x] T015 [US2] Add cooldown cleanup goroutine (定时清理过期记录) in `domain/notify/dedup.go`

**Checkpoint**: US2 complete — 冷却时间内重复告警被抑制

---

## Phase 5: User Story 3 - 推送状态追溯 (Priority: P3)

**Goal**: 提供推送历史查询 API，记录每次推送状态（成功/失败/去重）

**Independent Test**: 请求 `GET /api/market/alerts/history`，验证返回推送记录列表

### Implementation for User Story 3

- [x] T016 [US3] Implement `DeliveryRecord` type and `HistoryBuffer` (环形缓冲区) in `domain/notify/history.go`
- [x] T017 [US3] Integrate HistoryBuffer into Dispatcher in `domain/notify/dispatcher.go`: 每次推送（含成功/失败/去重）写入记录
- [x] T018 [P] [US3] Implement `HTTPHandler` with `GetHistory` endpoint in `domain/notify/http_handler.go`
- [x] T019 [US3] Register `GET /api/market/alerts/history` route in `api/route.go`
- [x] T020 [US3] Wire HTTPHandler in `cmd/server/main.go` and pass to route registration

**Checkpoint**: US3 complete — 推送历史可查询，前端/运维可通过 API 追溯所有告警状态

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, cleanup, and complete validation

- [x] T021 [P] Write `domain/notify/INDEX.md` describing public API surface (Sender, Service, HTTPHandler)
- [x] T022 [P] Update root `INDEX.md` to add `domain/notify/` to domain listing and Phase 2 status
- [x] T023 Run `make test ./domain/notify/...` and ensure all unit tests pass
- [x] T024 Validate full flow per `specs/005-webhook-alerts/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 — MVP delivery
- **US2 (Phase 4)**: Depends on US1 (needs Dispatcher to be working)
- **US3 (Phase 5)**: Depends on US1 (needs Dispatcher to record history)
- **Polish (Phase 6)**: Depends on all desired phases complete

### User Story Dependencies

- **US1 (P1)**: Standalone MVP — no other stories needed
- **US2 (P2)**: Enhances US1 Dispatcher — needs US1 complete
- **US3 (P3)**: Adds query API on top of US1 — needs US1 complete
- **US2 and US3 are independent of each other**: can be developed in parallel after US1

### Within Each User Story

- Models/entities first → services → integration
- Dispatcher before CooldownTracker and HistoryBuffer (both hook into Dispatcher)

### Parallel Opportunities

- T004 + T005 + T006: All foundational, different files, can run in parallel
- T013 + T016: CooldownTracker and HistoryBuffer are independent, can run in parallel after US1
- T008: Message builder is independent of WeComSender, can parallel with T007
- T018 + T021: HTTP handler and INDEX.md are independent

---

## Parallel Example: Phase 2 Foundational

```bash
# Run all foundational interface/type definitions in parallel:
Task: "Define WebhookSender interface in domain/notify/sender.go" (T004)
Task: "Implement MockSender in domain/notify/mock_sender.go" (T005)
Task: "Add EventCallback to domain/anomaly/engine.go" (T006)
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: 配置企业微信 Webhook，触发异动，验证群消息
5. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 → 企业微信推送可工作 → **MVP!**
3. Add US2 → 去重冷却生效 → 减少骚扰验证
4. Add US3 → 推送历史可查询 → 完整功能

### Parallel After US1

Once US1's Dispatcher is in place (T009), US2 (T013-T015) and US3 (T016-T020) can proceed in parallel by different developers.

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to user story for traceability
- Each user story is independently completable and testable
- US2 and US3 both hook into Dispatcher — ensure Dispatcher has clear extension points (pre-send hook for cooldown, post-send hook for history)
- All Go source with 中文注释 per constitution requirement VI
- Commit after each task or logical group
