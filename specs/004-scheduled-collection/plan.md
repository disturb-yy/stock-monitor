# Implementation Plan: 定时采集

**Branch**: `004-scheduled-collection` | **Date**: 2026-06-09 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/004-scheduled-collection/spec.md`

## Summary

在现有按需查询基础上新增后台定时采集器，交易时段自动周期性拉取 Tushare 行情数据并缓存到内存，同时提供历史行情查询接口。采集间隔可配置，复用交易日历模块判断采集时机。

## Technical Context

**Language/Version**: Go 1.22+

**Primary Dependencies**: Gin, log/slog, `pkg/tushare`, `domain/market/TradingCalendar`

**Storage**: 内存缓存（map），历史数据上限 30 天/指数

**Testing**: `go test ./...`

**Target Platform**: Linux server

**Project Type**: web-service (REST API + 后台采集器)

**Performance Goals**: 采集器不阻塞 API；历史查询 < 100ms

**Constraints**: Tushare 免费版频率限制 ~200 次/分钟；采集间隔 ≥ 1 分钟

**Scale/Scope**: 6 个指数，内存存储 < 10MB

## Constitution Check

*GATE: Must pass before Phase 0 research.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. DDF Architecture | ✅ PASS | 采集器放 `domain/market/collector.go`，属行情领域 |
| II. Strict Dependency Discipline | ✅ PASS | `domain/market` → `pkg/tushare` 允许；无跨领域导入 |
| III. INDEX.md Documentation | ✅ PASS | 实施后更新 `domain/market/INDEX.md` |
| IV. Mock-First Provider Pattern | ✅ PASS | 采集器仅对 TushareProvider 有效，Mock 模式跳过 |
| V. Phase-Gated Iteration | ✅ PASS | 属 Phase 2（发现异常的基础设施），不含告警 |
| VI. 中文注释 | ✅ PASS | 新文件使用中文注释 |

## Project Structure

### Documentation

```text
specs/004-scheduled-collection/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md
```

### Source Code

```text
domain/market/
├── collector.go         # 新建：定时采集器（ticker + 内存存储）
├── tushare_provider.go  # 已有，不动
├── calendar.go          # 已有，复用
├── quote.go             # 已有，不动
├── http_handler.go      # 修改：新增 GetHistory 端点
└── INDEX.md             # 更新

api/
└── route.go             # 注册 GET /api/market/history

configs/
└── config.yaml          # 新增 collector 配置段

cmd/server/
└── main.go              # 启动/停止采集器
```

**Structure Decision**: 采集器是行情领域的内部组件，放 `domain/market/collector.go`。历史数据存储为 `map[string][]HistoricalQuote`（key=指数代码），由 collector 写入、handler 读取。采集器生命周期由 Composition Root 控制。
