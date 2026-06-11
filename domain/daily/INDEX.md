# domain/daily — 每日收盘总结领域

## 职责

负责在每个交易日下午收盘后自动生成当日市场收盘总结，并通过企业微信 Webhook 推送到指定群聊。总结内容包含各指数涨跌幅、涨跌统计、领涨/领跌指数、异动事件摘要。

同时提供 REST API 查询收盘总结 JSON 数据（GET）和手动触发推送（POST）。

本领域不导入任何其他 domain 包。数据获取通过 `DataProvider` 接口注入，推送通过 `PushFunc` 回调注入，由 Composition Root（`cmd/server/main.go`）完成跨领域组装。

## 公开入口

| 类型 | 构造函数 | 用途 |
|------|----------|------|
| `DataProvider` (interface) | — | 数据获取接口（行情快照 + 异动事件 + 交易日判断） |
| `MockDataProvider` | `NewMockDataProvider()` | 测试用数据提供者 |
| `Builder` | `NewBuilder(provider, location)` | 收盘总结构建器 |
| `Scheduler` | `NewScheduler(builder, push, pushTime, location)` | 定时调度器（Ticker 驱动） |
| `HTTPHandler` | `NewHTTPHandler(builder, push, location)` | HTTP 处理器（GET 查询 + POST 触发） |
| `FormatMarkdown` | — | 将 DailySummary 格式化为企业微信 Markdown |
| `PushFunc` | — | 推送回调函数类型 |

## 依赖规则

- **不导入**任何其他 domain 包
- 可被 `cmd/server/main.go`（Composition Root）和 `api/route.go` 引用
- 通过接口注入获取数据和推送能力

## 文件清单

| 文件 | 职责 |
|------|------|
| `summary.go` | `DailySummary`、`IndexSnapshot`、`AnomalyItem`、`DataStatus` 模型 + `PushFunc` 类型 |
| `provider.go` | `DataProvider` 接口 + `MockDataProvider` 测试双 |
| `builder.go` | `Builder` 从 DataProvider 构建 DailySummary |
| `markdown.go` | `FormatMarkdown` 将总结格式化为企微 Markdown |
| `scheduler.go` | `Scheduler` 定时调度（Ticker + 交易日历） |
| `http_handler.go` | `HTTPHandler` GET 查询 + POST 手动触发 |
| `builder_test.go` | Builder 和 Markdown 单元测试 |
| `scheduler_test.go` | Scheduler 单元测试 |
| `http_handler_test.go` | HTTPHandler 单元测试 |
