# Implementation Plan: Webhook 告警推送

**Branch**: `005-webhook-alerts` | **Date**: 2026-06-09 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/005-webhook-alerts/spec.md`

## Summary

当异动检测引擎生成异动事件后，系统自动通过企业微信机器人 Webhook 将告警消息推送到指定群聊。消息采用 Markdown 格式，具备去重冷却、失败重试、推送历史追溯能力。推送流程与检测引擎异步解耦，不阻塞现有 API 响应。

## Technical Context

**Language/Version**: Go 1.25

**Primary Dependencies**: `net/http`（标准库，Webhook POST 不引入第三方 HTTP 客户端）

**Storage**: 内存（推送历史记录存储在内存环形缓冲区，Phase 2 不引入数据库）

**Testing**: `go test ./...`（Go 标准测试框架 + `net/http/httptest` 模拟 Webhook 端点）

**Target Platform**: Linux server（与现有后端一致）

**Project Type**: web-service（后端 API 服务 + 后台异步任务）

**Performance Goals**: 单个 Webhook POST 请求 3 秒内完成；推送不阻塞异动检测 API 响应

**Constraints**: 推送异步执行；内存占用 <10MB（推送历史 + 冷却追踪）；企业微信 Markdown 不支持表格/列表

**Scale/Scope**: 6 个指数、1 个推送渠道、1000 条推送历史上限

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| **I. DDF Architecture** | ✅ PASS | 新增 `domain/notify/` 领域包，自包含模型、接口、服务、HTTP handler，不创建技术分层目录 |
| **II. Strict Dependency Discipline** | ✅ PASS | `domain/notify` 不导入其他 domain；与 anomaly 的协作通过 Composition Root 注入回调函数；`pkg/*` 不导入 `domain/notify` |
| **III. INDEX.md Documentation** | ✅ PASS | `domain/notify/INDEX.md` 将在实现阶段创建，描述公开 API 入口 |
| **IV. Mock-First Provider Pattern** | ✅ PASS | 企业微信 Webhook 调用抽象为 `WebhookSender` 接口；提供 `MockSender` 和 `WeComSender` 两个实现 |
| **V. Phase-Gated Iteration** | ✅ PASS | 本功能属于 Phase 2（发现异常→告警通知），符合阶段目标；不涉及 Phase 3/4 的内容 |
| **VI. Frontend-Backend Separation** | ✅ PASS | 后端新增推送历史查询 API；前端可展示但不含推送逻辑 |
| **VI. 中文注释（MANDATORY）** | ✅ PASS | 所有新增 Go 代码使用中文注释 |

**Gate Result**: ALL PASS — 无需填写 Complexity Tracking

## Project Structure

### Documentation (this feature)

```text
specs/005-webhook-alerts/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── webhook-api.md   # Webhook 推送 API 契约
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
domain/
├── anomaly/                    # 现有：异动检测引擎
│   └── ...                     # 新增：事件回调注入点
├── notify/                     # 新增：Webhook 告警领域
│   ├── INDEX.md                # 领域索引
│   ├── config.go               # 告警推送配置结构
│   ├── sender.go               # WebhookSender 接口 + 企业微信实现
│   ├── mock_sender.go          # Mock 发送器（测试用）
│   ├── message.go              # Markdown 消息构建
│   ├── dedup.go                # 冷却去重追踪器
│   ├── dispatcher.go           # 推送调度器（异步、重试、合并）
│   ├── history.go              # 推送历史记录（内存环形缓冲区）
│   ├── service.go              # 通知服务（编排调度 + 历史查询）
│   └── http_handler.go         # HTTP 处理器（推送历史查询接口）
pkg/
└── config/
    └── config.go               # 新增 WebhookConfig 配置结构
api/
└── route.go                    # 新增推送历史查询路由
cmd/server/
└── main.go                     # 新增 notify 领域组装 + 注入 anomaly 回调
configs/
└── config.yaml                 # 新增 webhook 配置段
```

**Structure Decision**: 遵循 DDF 架构，新增 `domain/notify/` 作为独立的告警推送领域。跨领域协作通过 Composition Root 注入回调实现。

## Complexity Tracking

无违规项，无需填写。
