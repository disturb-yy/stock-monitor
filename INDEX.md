# stock-monitor 项目索引

本文档用于告诉 AI：这个项目是什么、当前要做哪一步、项目有几层、每层负责什么。需要查看某一层细节时，再进入对应目录读取该层的 `INDEX.md`。

## 项目定位

`stock-monitor` 是股票大盘监控系统，长期目标是支持多个市场的大盘监控、异动发现、告警通知和市场分析。

当前 MVP 只支持 A 股，第一阶段目标是完成 **数据底座**。

## 项目结构（DDF）

项目采用 DDF（Domain-Driven Flattening，领域驱动平铺）结构，按业务领域拆分，不再使用 `internal/handler`、`internal/service`、`internal/repository`、`internal/model` 的技术分层。

```plaintext
domain/
  market/    # 市场行情领域 → domain/market/INDEX.md
  auth/      # 认证领域 → domain/auth/INDEX.md
```

转换规则参见 `../stock-market-monitor/docs/ddf_migration_rules.md`。

## 当前阶段

| 阶段 | 名称 | 状态 | 目标 |
| --- | --- | --- | --- |
| Phase 1 | 看得见 | ✅ 已完成 | 返回 A 股市场状态、主要指数行情、大盘总览 |
| Phase 2 | 发现异常 | ✅ 已完成 | 规则引擎、异动事件、企业微信告警通知 |
| Phase 3 | 解释原因 | 未开始 | 市场结构分析、板块贡献、收盘总结 |
| Phase 4 | 辅助决策 | 未开始 | 市场评分、自定义规则、多市场扩展 |

## 已实现功能

### Phase 1: 看得见



- 定义 A 股市场常量，例如 `CN_A`。（已完成：`domain/market/quote.go`）
- 定义市场状态，例如 `pre_open`、`trading`、`lunch_break`、`closed`。（已完成）
- 定义指数行情模型，例如 `IndexQuote`。（已完成）
- 定义大盘总览模型，例如 `MarketOverview`。（已完成）
- 定义行情数据源接口，例如 `Provider`。（已完成：`domain/market/provider.go`）
- 先用 mock 数据实现 A 股 provider。（已完成：`domain/market/mock_provider.go`）
- 预留 Tushare provider。（已完成：`domain/market/tushare_provider.go`）
- 定义市场业务服务，例如 `Service`。（已完成：`domain/market/service.go`）
- 提供以下 API：
  - `POST /api/auth/token`
  - `GET /api/market/status`
  - `GET /api/market/indices`
  - `GET /api/market/overview`

### Phase 2: 发现异常

- Tushare 真实行情源接入（`specs/001-tushare-real-provider/`）
- 交易日历缓存与查询（`specs/002-trading-calendar/`）
- 多维度异动检测引擎：涨跌幅 / 成交量 / 连续涨跌（`specs/003-anomaly-detection/`）
- 定时采集 + 历史行情查询 + SQLite 持久化（`specs/004-scheduled-collection/`, `specs/006-sqlite-persistence/`）
- 企业微信 Webhook 告警推送：冷却去重 / 重试 / 推送历史（`specs/005-webhook-alerts/`）

## 项目分层

| 层级 | 路径 | 职责 | 详情 |
| --- | --- | --- | --- |
| 启动层 | `cmd/server/main.go` | Composition Root：集中加载配置、创建依赖、注入、启动 HTTP Server、优雅关闭 | 见本文档下方 |
| 路由层 | `api/route.go` | 注册 Gin 路由，接收领域 handler 实例，不创建业务依赖 | 见本文档下方 |
| 领域层 | `domain/` | 按业务领域拆分，每个领域包含模型、接口、实现、服务、HTTP 适配 | `domain/market/INDEX.md`、`domain/auth/INDEX.md` |
| 配置层 | `configs/` | YAML 运行配置文件 | 见本文档下方 |
| 公共组件层 | `pkg/` | 跨领域复用的基础设施能力，不含业务模型或接口 | 各子包均有 `INDEX.md` |

## 领域详情

### domain/market — 市场行情领域

详见 [`domain/market/INDEX.md`](domain/market/INDEX.md)

**包名**：`market`

**公开入口：**

| 类型 | 构造函数 | 用途 |
| --- | --- | --- |
| `Provider` (interface) | — | 行情数据源契约 |
| `MockProvider` | `NewMockProvider()` | A 股模拟数据 |
| `TushareProvider` | `NewTushareProvider(client)` | Tushare 真实数据源（待实现） |
| `Service` | `NewService(provider)` | 业务服务 |
| `HTTPHandler` | `NewHTTPHandler(service)` | HTTP 适配 |

### domain/auth — 认证领域

详见 [`domain/auth/INDEX.md`](domain/auth/INDEX.md)

**包名**：`auth`

**公开入口：**

| 类型 | 构造函数 | 用途 |
| --- | --- | --- |
| `Authenticator` (interface) | — | 认证接口 |
| `Service` | `NewService(config)` | Token 生成/验证服务 |
| `HTTPHandler` | `NewHTTPHandler(service, tokenTTL)` | Token API handler |
| `JWTMiddleware` | `JWTMiddleware(authenticator)` | Gin JWT 中间件 |
| `Config` | — | 认证配置 |

## pkg 基础设施

| 路径 | 职责 | 详情 |
| --- | --- | --- |
| `pkg/config/` | 加载和解析 YAML 配置，提供全局配置读取 | [`pkg/config/INDEX.md`](pkg/config/INDEX.md) |
| `pkg/httputil/` | 统一响应结构和错误码 | [`pkg/httputil/INDEX.md`](pkg/httputil/INDEX.md) |
| `pkg/logger/` | 基于 `slog` 和 `lumberjack` 的日志封装 | [`pkg/logger/INDEX.md`](pkg/logger/INDEX.md) |
| `pkg/middleware/` | 通用 Gin 中间件（RequestID、AccessLog） | [`pkg/middleware/INDEX.md`](pkg/middleware/INDEX.md) |
| `pkg/router/` | Gin 路由组辅助工具 | [`pkg/router/INDEX.md`](pkg/router/INDEX.md) |
| `pkg/tushare/` | Tushare Pro HTTP API 客户端 | [`pkg/tushare/INDEX.md`](pkg/tushare/INDEX.md) |

## DDF 依赖规则

**允许：**

```plaintext
cmd/server -> api
cmd/server -> domain/*
cmd/server -> pkg/*

api -> domain/*
api -> pkg/router
api -> gin

domain/* -> pkg/httputil
domain/* -> pkg/logger
domain/market -> pkg/tushare
domain/auth -> pkg/config
```

**禁止：**

```plaintext
domain/market -> domain/auth
domain/auth -> domain/market
domain/* -> cmd/server
domain/* -> api
pkg/* -> domain/*
```

跨领域协作通过 `cmd/server/main.go`（Composition Root）组装。

## 请求链路

```plaintext
HTTP Request
  -> cmd/server/main.go 创建 gin.Engine 并挂载中间件
  -> api/route.go 注册路由，接收 marketHandler、authHandler 实例
  -> domain/market/http_handler.go 处理 market API 请求
  -> domain/market/service.go 组织 market 查询逻辑
  -> domain/market/mock_provider.go 返回 mock 数据
  -> domain/market/quote.go 承载 MarketSession、IndexQuote、MarketOverview
  -> pkg/httputil 统一响应格式
```

## 当前 API 契约

| 接口 | 返回核心数据 | 用途 |
| --- | --- | --- |
| `POST /api/auth/token` | JWT Token | 根据 subject 生成访问 Token |
| `GET /api/market/status` | `MarketSession` | 判断当前市场是否交易、处于什么阶段 |
| `GET /api/market/indices` | `[]IndexQuote` | 返回 A 股主要指数行情 |
| `GET /api/market/overview` | `MarketOverview` | 返回市场状态 + 指数行情的大盘总览 |
| `GET /api/market/history` | `map[string][]IndexQuote` | 历史行情查询（支持日期范围、symbol 过滤） |
| `GET /api/market/anomalies` | `{events, count}` | 实时异动检测事件列表 |
| `GET /api/market/alerts/history` | `{records, total}` | Webhook 告警推送历史（始终可用，webhook 关闭时返回空） |
| `GET /api/market/daily-summary` | `DailySummary` | 查询当日收盘总结 JSON（含涨跌统计、异动摘要） |
| `POST /api/market/daily-summary` | — | 手动触发收盘总结推送 |

公开端点（无需认证）：

| 接口 | 用途 |
| --- | --- |
| `GET /v1/index` | 服务基本信息 |
| `GET /v1/ready` | 就绪检查 |
| `GET /v1/heart` | 心跳检查 |

## 命名规范

- 包名描述领域，文件名描述职责。
- 不使用 `models`、`services`、`repositories` 技术角色包名。
- 包名已表达领域时，类型名不重复领域名（如 `market.Service` 而非 `market.MarketService`）。

## 修改建议

- 新增领域时，在 `domain/` 下创建新目录，包含该领域的模型、接口、服务、HTTP handler，并创建对应的 `INDEX.md`。
- 新增 Provider 实现时，接口定义在使用方领域包内，实现在同一领域包内。
- 新增 API 时，先在领域包实现 handler/service，再到 `api/route.go` 注册。
- 不要在 `pkg/` 中放置业务接口（如 `MarketProvider`、`OrderRepository`）。
- 所有依赖组装集中在 `cmd/server/main.go`。
- DDF 迁移规则详见 `../stock-market-monitor/docs/ddf_migration_rules.md`。
- 修改代码后，更新对应目录的 `INDEX.md` 以保持文档同步。

## 部署与运行文件

- `Makefile`：常用开发命令入口，包含 `run`、`build`、`tidy`、`test`、`clean`。
- `go.mod`：Go 模块定义，module path 为 `github.com/disturb-yy/stock-monitor`。
- `configs/config.yaml`：默认配置文件。

## 子索引

| 目录 | 索引文件 |
| --- | --- |
| `domain/market/` | [`domain/market/INDEX.md`](domain/market/INDEX.md) |
| `domain/auth/` | [`domain/auth/INDEX.md`](domain/auth/INDEX.md) |
| `pkg/config/` | [`pkg/config/INDEX.md`](pkg/config/INDEX.md) |
| `pkg/httputil/` | [`pkg/httputil/INDEX.md`](pkg/httputil/INDEX.md) |
| `pkg/logger/` | [`pkg/logger/INDEX.md`](pkg/logger/INDEX.md) |
| `pkg/middleware/` | [`pkg/middleware/INDEX.md`](pkg/middleware/INDEX.md) |
| `pkg/router/` | [`pkg/router/INDEX.md`](pkg/router/INDEX.md) |
| `pkg/tushare/` | [`pkg/tushare/INDEX.md`](pkg/tushare/INDEX.md) |
| `domain/daily/` | [`domain/daily/INDEX.md`](domain/daily/INDEX.md) |

## notify 领域

`domain/notify/` — Webhook 告警推送，支持企业微信群机器人。
详见 [`domain/notify/INDEX.md`](domain/notify/INDEX.md)

## daily 领域

`domain/daily/` — 每日收盘总结，自动生成并推送市场总结。
详见 [`domain/daily/INDEX.md`](domain/daily/INDEX.md)
