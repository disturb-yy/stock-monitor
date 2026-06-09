# Implementation Plan: SQLite 持久化

**Branch**: `006-sqlite-persistence` | **Date**: 2026-06-10 | **Spec**: [spec.md](./spec.md)

## Summary

为 HistoryStore 内存数据增加 SQLite 持久化层：采集后立即写入 SQLite，启动时从 SQLite 恢复最近 maxDays 天数据到内存，更早数据保留在 SQLite 按需查询。实现服务重启后历史数据零丢失。

## Technical Context

**Language/Version**: Go 1.25

**Primary Dependencies**: `github.com/mattn/go-sqlite3`（CGo SQLite 驱动）+ `database/sql`（标准库）

**Storage**: SQLite 文件（`data/market_history.db`）+ 现有内存 HistoryStore（不做替换，作为缓存层）

**Testing**: `go test ./...` + `net/http/httptest` + SQLite 内存模式（`:memory:` DSN）

**Target Platform**: Linux server（需 CGo 编译支持）

**Project Type**: web-service 基础设施增强

**Performance Goals**: SQLite 写入 < 5ms/条（6 个指数单次采集 < 30ms）；启动加载 < 3s

**Constraints**: 写入不能阻塞采集周期；不能影响现有 API 响应；内存优先于 SQLite

**Scale/Scope**: 6 个指数 × 90 天 ≈ 540 条记录（行数极小，SQLite 性能无需优化）

## Constitution Check

*GATE: Must pass before Phase 0 research.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| **I. DDF Architecture** | ✅ PASS | 持久化逻辑作为 `market` 领域内部组件（`sqlite_store.go`），不新增 domain 包 |
| **II. Strict Dependency Discipline** | ✅ PASS | 新增代码在 `domain/market/` 内，不引入跨领域导入 |
| **III. INDEX.md Documentation** | ✅ PASS | 更新 `domain/market/INDEX.md` 添加 SQLite 持久化说明 |
| **IV. Mock-First Provider Pattern** | ✅ PASS | SQLite 层封装在 `HistoryStore` 内，通过 `database/sql` 接口天然可 mock（`:memory:` 模式） |
| **V. Phase-Gated Iteration** | ✅ PASS | 本功能属于 Phase 2 基础设施增强 |
| **VI. Frontend-Backend Separation** | ✅ PASS | 纯后端变更，前端无感知 |
| **VI. 中文注释** | ✅ PASS | 所有新增 Go 代码使用中文注释 |

**⚠️ Constitution Amendment Required**: 当前 constitution 技术约束规定 "No database (Phase 1)"，需通过 amendment 添加 SQLite 存储原则。Amendment 已完成（constitution v1.1.1）。

**Gate Result**: ALL PASS（amendment 为前置条件）

## Project Structure

### Documentation

```text
specs/006-sqlite-persistence/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── tasks.md
```

### Source Code

```text
domain/market/
├── collector.go          # 现有：采集器 + HistoryStore
├── sqlite_store.go       # 新增：SQLite 持久化层（建表/写入/查询/清理）
├── http_handler.go       # 修改：GetHistory 查询路径加入 SQLite 回退
├── INDEX.md              # 更新：添加 SQLite 持久化说明
pkg/config/
└── config.go             # 修改：新增 PersistenceConfig
cmd/server/
└── main.go               # 修改：初始化 SQLite 连接，注入 HistoryStore
configs/
└── config.yaml           # 修改：新增 persistence 配置段
```

**Structure Decision**: SQLite 持久化作为 `domain/market` 内部组件（`sqlite_store.go`），不创建独立 domain 包。持久化逻辑通过扩展 HistoryStore 对外接口（添加 Load/Flush）暴露，Composition Root 负责初始化 SQLite 连接并注入。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 引入 SQLite 数据库 | 重启后数据持久性是刚需，纯内存方案重启必丢数据 | 文件序列化（JSON/CSV）无法支持按日期范围高效查询；mmap 文件方案复杂度更高 |

