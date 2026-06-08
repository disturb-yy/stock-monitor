# Implementation Plan: Tushare 真实行情接入

**Branch**: `001-tushare-real-provider` | **Date**: 2026-06-08 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/001-tushare-real-provider/spec.md`

## Summary

实现 TushareProvider 的三个方法（`GetMarketStatus`、`GetIndexQuotes`、`GetMarketOverview`），使其通过 `pkg/tushare` 客户端调用 Tushare Pro API 获取 A 股真实行情数据，替换当前返回 nil 的桩实现。同时扩展配置，支持在 mock 和真实数据源之间切换。

## Technical Context

**Language/Version**: Go 1.22+

**Primary Dependencies**: Gin (HTTP), log/slog + lumberjack (logging), `pkg/tushare` (Tushare Pro HTTP client)

**Storage**: N/A (Phase 1 — 无数据库，内存运行)

**Testing**: `go test ./...`

**Target Platform**: Linux server

**Project Type**: web-service (REST API)

**Performance Goals**: 大盘总览 < 3 秒（含多指数聚合查询）

**Constraints**: Tushare 免费版频率限制 ~200 次/分钟；无缓存层；单次请求最多查询 6 个指数

**Scale/Scope**: Phase 1 单机部署，内部使用

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. DDF Architecture | ✅ PASS | TushareProvider 在 `domain/market/`，与模型/接口/服务同包；Tushare 客户端在 `pkg/tushare/` |
| II. Strict Dependency Discipline | ✅ PASS | `domain/market` → `pkg/tushare` 允许；无跨领域导入；Composition Root 注入 |
| III. INDEX.md Documentation | ✅ PASS | 实施后更新 `domain/market/INDEX.md`、`pkg/tushare/INDEX.md` |
| IV. Mock-First Provider Pattern | ✅ PASS | MockProvider 保留；TushareProvider 实现同一 Provider 接口；配置切换 |
| V. Phase-Gated Iteration | ✅ PASS | 仅限 Phase 1（看得见）范围；不含异常检测/告警/分析等后续阶段功能 |

## Project Structure

### Documentation (this feature)

```text
specs/001-tushare-real-provider/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
domain/market/
├── provider.go              # Provider interface (已有)
├── tushare_provider.go      # TushareProvider 实现 ← 本次填充
├── mock_provider.go         # MockProvider (已有，保留)
├── quote.go                 # 数据模型 (已有)
├── service.go               # Service (已有)
├── http_handler.go          # HTTP handler (已有)
└── INDEX.md                 # 更新

pkg/tushare/
├── client.go                # Tushare HTTP 客户端 (已有)
├── types.go                 # 类型定义 (已有，可能扩展)
└── INDEX.md                 # 更新

configs/
└── config.yaml              # 添加 tushare 配置段 + provider 开关

cmd/server/
└── main.go                  # Composition Root: 根据配置选择 provider 注入
```

**Structure Decision**: 沿用项目 DDF 结构。`domain/market/tushare_provider.go` 填充实现逻辑，`configs/config.yaml` 增加 Tushare 配置项，`cmd/server/main.go` 根据配置开关注入 MockProvider 或 TushareProvider。

## Complexity Tracking

> 无违规项，无需填写。
