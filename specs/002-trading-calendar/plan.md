# Implementation Plan: 交易日历接入

**Branch**: `002-trading-calendar` | **Date**: 2026-06-08 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/002-trading-calendar/spec.md`

## Summary

通过 Tushare `trade_cal` 接口获取 A 股交易日历，替换当前 `TushareProvider` 中仅基于时间（周末 + 时段）的简易状态判断。实现日历本地缓存、自动更新、过期告警，确保市场状态和 `tradeDate` 字段在法定节假日/调休场景下准确。

## Technical Context

**Language/Version**: Go 1.22+

**Primary Dependencies**: Gin, log/slog + lumberjack, `pkg/tushare`（扩展 `TradeCal` 方法）

**Storage**: 本地 JSON 文件缓存交易日历

**Testing**: `go test ./...`

**Target Platform**: Linux server

**Project Type**: web-service (REST API)

**Performance Goals**: 日历查询为内存操作，不增加接口延迟

**Constraints**: Tushare `trade_cal` 接口每月有调用频率限制；日历数据覆盖范围前后各 6 个月

**Scale/Scope**: Phase 1 增强，不涉及新 API 端点

## Constitution Check

*GATE: Must pass before Phase 0 research.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. DDF Architecture | ✅ PASS | 日历逻辑在 `domain/market/` 内；Tushare 客户端扩展在 `pkg/tushare/` |
| II. Strict Dependency Discipline | ✅ PASS | `domain/market` → `pkg/tushare` 允许；无跨领域导入 |
| III. INDEX.md Documentation | ✅ PASS | 实施后更新 `domain/market/INDEX.md`、`pkg/tushare/INDEX.md` |
| IV. Mock-First Provider Pattern | ✅ PASS | MockProvider 的 `GetMarketStatus` 不受影响（独立实现） |
| V. Phase-Gated Iteration | ✅ PASS | 属 Phase 1 增强（交易日历是"看得见"的基础） |

## Project Structure

### Documentation (this feature)

```text
specs/002-trading-calendar/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md             # (/speckit-tasks)
```

### Source Code (repository root)

```text
pkg/tushare/
├── client.go            # 新增 TradeCal 方法
├── types.go             # 新增 TradeCalItem 类型
└── INDEX.md             # 更新

domain/market/
├── calendar.go          # 新增：交易日历缓存与查询
├── tushare_provider.go  # 修改：替换 determineMarketSession
├── quote.go             # 已有，不动
└── INDEX.md             # 更新

configs/
└── config.yaml          # 新增 calendar 配置段

data/
└── trading_calendar.json  # 运行时生成的缓存文件（.gitignore）
```

**Structure Decision**: 日历逻辑封装在 `domain/market/calendar.go` 新建文件，与 Provider 解耦。Tushare `trade_cal` API 封装在 `pkg/tushare/` 中扩展。

## Complexity Tracking

> 无违规项。
