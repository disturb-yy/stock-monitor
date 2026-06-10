# Implementation Plan: 每日收盘总结推送

**Branch**: `007-daily-summary` | **Date**: 2026-06-11 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/007-daily-summary/spec.md`

## Summary

每个交易日下午 15:15 自动生成当日收盘总结并推送到企业微信。总结包含各指数涨跌幅、涨跌统计、领涨领跌指数、异动事件摘要。提供 GET 查询 API 和 POST 手动触发端点。复用现有 Webhook 推送通道和交易日历。

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: Gin, log/slog, go-sqlite3, 现有 domain/market（HistoryStore, TradingCalendar）、domain/notify（WebhookSender, Dispatcher）
**Storage**: SQLite (market_history.db, persist daily summary records), 内存 HistoryStore (latest data)
**Testing**: go test（标准库），Mock DataProvider, Mock TradingCalendar, MockSender
**Target Platform**: Linux server
**Project Type**: backend web service
**Performance Goals**: 收盘总结生成 < 5s，推送延迟 < 3min（从 15:00 到推送完成）
**Constraints**: 不阻塞异动检测和行情 API；每天仅触发 1 次；进程内 time.Ticker 调度
**Scale/Scope**: 6 个追踪指数，每天 1 次推送，新领域 `domain/daily/`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| 原则 | 状态 | 说明 |
|------|------|------|
| I. DDF Architecture | ✅ | 新领域 `domain/daily/` 扁平结构，自包含模型/服务/HTTP handler |
| II. Strict Dependency Discipline | ✅ | daily 不导入任何其他 domain，通过接口注入（DataProvider, PushFunc） |
| III. INDEX.md Documentation | ✅ | 创建 `domain/daily/INDEX.md` |
| IV. Mock-First Provider Pattern | ✅ | `MockDataProvider` 实现 `DataProvider` 接口，支持无外部依赖测试 |
| V. Phase-Gated Iteration | ✅ | Phase 3 功能，符合 roadmap |
| VI. Frontend-Backend Separation | ✅ | REST API 供前端查询，CORS 在 main.go 管理 |
| VI. 中文注释 | ✅ | 全部 Go 源码使用中文注释 |

**Post-Design Re-check**: ✅ 无违规

## Project Structure

### Documentation (this feature)

```text
specs/007-daily-summary/
├── plan.md              # 本文件
├── research.md          # 技术决策研究
├── data-model.md        # 数据模型
├── quickstart.md        # 验证指南
├── contracts/           # 接口契约
│   └── daily-api.md
└── tasks.md             # 任务分解（/speckit-tasks）
```

### Source Code (repository root)

```text
domain/daily/              # 新增领域
├── INDEX.md               # 领域索引
├── summary.go             # DailySummary 模型 + DataStatus
├── builder.go             # Builder（从 DataProvider 构建总结）
├── builder_test.go        # Builder 单元测试
├── scheduler.go           # Scheduler（定时触发，Ticker + 交易日历）
├── scheduler_test.go      # Scheduler 单元测试
├── config.go              # DailySummaryConfig
├── http_handler.go        # HTTPHandler（GET 查询 + POST 触发）
├── http_handler_test.go   # HTTPHandler 单元测试
├── provider.go            # DataProvider 接口 + MockDataProvider
└── markdown.go            # Markdown 格式化（DailySummary → 企微消息）
```

## Complexity Tracking

无违规项，无需记录。
