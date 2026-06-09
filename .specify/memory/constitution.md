# stock-monitor Constitution
<!--
Sync Impact Report
==================
Version change: 1.1.0 → 1.1.1
Modified sections:
  - Technical Constraints → Backend: "No database (Phase 1)" 改为 "SQLite 持久化 (Phase 2+)"
Added sections: None
Removed sections: None
Templates requiring updates:
  - .specify/templates/plan-template.md: ✅ aligned (技术上下文已支持 Storage 字段)
  - .specify/templates/spec-template.md: ✅ aligned (no constitution-specific constraints)
  - .specify/templates/tasks-template.md: ✅ aligned (no impact)
  - .specify/templates/checklist-template.md: ✅ aligned (no impact)
Follow-up TODOs: None
Bump rationale: PATCH — 放宽技术约束，允许 Phase 2+ 引入 SQLite 作为轻量持久化方案（不引入新原则，仅修改现有约束条款）。
-->

## Core Principles

### I. DDF Architecture (Domain-Driven Flattening)
Business domains are first-class directories under `domain/`. Each domain is
self-contained: models, interfaces, services, and HTTP handlers live together in
one package. Technical-layer directories (`internal/handler`, `internal/service`,
`internal/repository`) are forbidden. Package names describe the domain, not the
technical role. New domains MUST follow the same flat structure and include an
`INDEX.md`.

### II. Strict Dependency Discipline
Dependencies flow inward. `pkg/*` MUST NOT import `domain/*`, `cmd/*`, or `api`.
Domains MUST NOT import each other (e.g., `domain/market` → `domain/auth` is
forbidden). Cross-domain collaboration occurs exclusively through the Composition
Root (`cmd/server/main.go`). All dependency wiring, including provider injection
and middleware assembly, is centralized in the Composition Root.

### III. INDEX.md Documentation
Every directory that exports code MUST contain an `INDEX.md` describing its
purpose, public API surface, dependency rules, and relationship to other modules.
INDEX.md files MUST be updated whenever code is added, removed, or repackaged.
The root `INDEX.md` serves as the project map for both humans and AI agents.

### IV. Mock-First Provider Pattern
All external data sources MUST be abstracted behind Go interfaces defined in the
consuming domain package. A mock implementation MUST ship before or alongside any
real provider. Real provider implementations (e.g., Tushare, database) MUST
implement the same interface contract. This ensures the system is testable and
runnable without external dependencies from day one.

### V. Phase-Gated Iteration
Development follows a 4-phase roadmap: (1) 看得见 — visible data, (2) 发现异常 —
anomaly detection, (3) 解释原因 — causal analysis, (4) 辅助决策 — decision support.
Each phase has explicit entry and exit criteria defined in the root `INDEX.md`.
Features belonging to future phases MUST NOT be built in the current phase. Scope
creep across phase boundaries requires a constitution amendment.

### VI. Frontend-Backend Separation
The frontend is a standalone application under `frontend/` that communicates with
the Go backend exclusively via REST APIs. Frontend code MUST NOT contain business
logic, data transformation rules, or domain knowledge — those belong to backend
domain packages. The frontend is responsible for presentation, user interaction,
and API data display only. Backend APIs MUST be designed with frontend
consumption in mind: consistent response envelopes, meaningful HTTP status codes,
and CORS configuration managed in `cmd/server/main.go`.

## Technical Constraints

### Backend
- **Language**: Go 1.22+ with module path `github.com/disturb-yy/stock-monitor`.
- **HTTP Framework**: Gin; routing registered in `api/route.go`.
- **Logging**: `log/slog` with `lumberjack` rotation via `pkg/logger`.
- **Auth**: Stateless JWT tokens issued by `domain/auth`; no session storage.
- **Configuration**: YAML-based via `pkg/config`; loaded at startup only.
- **SQLite 持久化 (Phase 2+)**: Phase 2 及之后阶段允许使用 SQLite 作为轻量级本地持久化方案。
  SQLite 作为内存存储的补充（非替代），用于实现重启后数据恢复和历史数据扩展查询。
  禁止引入客户端-服务器型数据库（MySQL、PostgreSQL 等），该决策需单独的 amendment。
- **Observability**: Structured access logs (`pkg/middleware/access_log`) and
  request ID tracing (`pkg/middleware/request_id`) on every request.

### Frontend
- **Language**: TypeScript 5.x with React 18+ or Vue 3+ (team decision required).
- **Build Tool**: Vite or Next.js (aligned with chosen framework).
- **Charts**: ECharts or equivalent for financial data visualization.
- **State Management**: Framework-native (React Context / Vue Composition API) for
  Phase 1; dedicated state library deferred to later phases.
- **HTTP Client**: `fetch` or `axios`; API base URL configurable via environment
  variables.

## Development Workflow

### Backend
- **Build & Run**: Use `Makefile` targets: `make run`, `make build`, `make test`,
  `make tidy`, `make clean`.
- **Code Changes**: Modify or add code in the appropriate domain or pkg directory;
  register new routes in `api/route.go`; wire dependencies in
  `cmd/server/main.go`; update the affected `INDEX.md`(s).
- **Naming**: Package names describe the domain (e.g., `market`, `auth`). File
  names describe the responsibility (e.g., `service.go`, `http_handler.go`).
  Types MUST NOT repeat the package name (use `market.Service`, not
  `market.MarketService`).
- **Spec-Driven Development**: Features are specified via `$speckit-specify`,
  planned via `$speckit-plan`, tasked via `$speckit-tasks`, and implemented via
  `$speckit-implement`. All specs live under `specs/` with the branch naming
  convention `###-feature-name`.

### Frontend
- **Dev Server**: `npm run dev` (or framework equivalent) from `frontend/`.
- **Code Changes**: Components in `frontend/src/components/`, pages in
  `frontend/src/pages/`, API service wrappers in `frontend/src/services/`.
- **Naming**: React components use PascalCase files (e.g., `MarketOverview.tsx`);
  Vue components follow Vue Style Guide conventions.
- **Spec-Driven**: Frontend features follow the same `$speckit-*` workflow.
  A frontend feature spec SHOULD reference the backend API contract it consumes.

### VI. 中文注释（MANDATORY）
所有 Go 源代码必须使用中文注释。包级别、导出的类型/函数/方法、
结构体字段均需有中文说明。关键代码段（计算公式、降级策略、
边界条件处理）需有解释性注释。日志和错误消息也使用中文。


## Governance

This constitution supersedes all other development practices. Amendments require:
1. A documented rationale in the Sync Impact Report at the top of this file.
2. A version bump following semantic versioning (MAJOR for principle
   removal/redefinition, MINOR for new principles/sections, PATCH for
   clarifications).
3. Propagation checks against all `.specify/templates/` files and the root
   `INDEX.md`.

Compliance is verified at the Constitution Check gate in every implementation
plan (`$speckit-plan`). Any deviation from DDF dependency rules or phase
boundaries MUST be explicitly justified in the plan's Complexity Tracking table.

Runtime development guidance is provided by the root `INDEX.md` and per-domain
`INDEX.md` files.

**Version**: 1.1.1 | **Ratified**: 2026-06-07 | **Last Amended**: 2026-06-10
