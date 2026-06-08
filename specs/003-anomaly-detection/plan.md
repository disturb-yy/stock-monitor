# Implementation Plan: 异动检测规则引擎

**Branch**: `003-anomaly-detection` | **Date**: 2026-06-08 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/003-anomaly-detection/spec.md`

## Summary

在现有行情查询基础上增加可配置的异动检测规则引擎，支持涨跌幅超阈值、成交量突增、连续涨跌三种检测类型。检测结果通过独立 API 接口暴露为异动事件列表。规则全部配置化，无需改代码即可调整参数。

## Technical Context

**Language/Version**: Go 1.22+

**Primary Dependencies**: Gin, log/slog, `pkg/tushare`（历史数据查询）

**Storage**: N/A（Phase 2 无数据库，成交量均值与连续涨跌依赖 Tushare API 历史数据）

**Testing**: `go test ./...`

**Target Platform**: Linux server

**Project Type**: web-service (REST API)

**Performance Goals**: 异动检测叠加于行情查询，不增加明显延迟（<500ms 额外开销）

**Constraints**: 连续涨跌检测需额外 Tushare API 调用（历史 K 线），注意频率限制

**Scale/Scope**: 当前 6 个指数，规则 ≤ 10 条

## Constitution Check

*GATE: Must pass before Phase 0 research.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. DDF Architecture | ✅ PASS | 新增 `domain/anomaly/` 领域包，自包含模型+引擎+HTTP handler |
| II. Strict Dependency Discipline | ✅ PASS | `domain/anomaly` → `domain/market`（市场数据模型）允许；不导出到 pkg |
| III. INDEX.md Documentation | ✅ PASS | 实施后创建 `domain/anomaly/INDEX.md` |
| IV. Mock-First Provider Pattern | ✅ PASS | 规则引擎纯计算，无外部依赖 |
| V. Phase-Gated Iteration | ✅ PASS | 属 Phase 2（发现异常）；不含告警通知（Phase 2 后续） |

## Project Structure

### Documentation

```text
specs/003-anomaly-detection/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md
```

### Source Code

```text
domain/anomaly/
├── event.go           # AnomalyEvent 模型
├── rule.go            # DetectionRule 接口 + 配置结构
├── engine.go          # 规则引擎：注册规则 + 执行检测
├── price_change.go    # 涨跌幅检测器
├── volume_spike.go    # 成交量突增检测器
├── consecutive.go     # 连续涨跌检测器
├── http_handler.go    # 异动事件查询接口
└── INDEX.md           # 新建

domain/market/
├── http_handler.go    # 修改：注入 anomaly 引擎，扩增大盘总览返回异动标记

api/
└── route.go           # 注册 /api/market/anomalies

configs/
└── config.yaml        # 新增 rules 配置段

cmd/server/
└── main.go            # 创建规则引擎，注入 handler
```

**Structure Decision**: 异动检测作为独立领域 `domain/anomaly/`，遵循 DDF。通过 Composition Root 将引擎注入 market handler，market handler 在查询行情后触发检测。

## Complexity Tracking

> 无违规项。
