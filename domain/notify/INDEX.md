# domain/notify — Webhook 告警推送领域

## 职责

负责将异动事件通过企业微信机器人 Webhook 推送到指定群聊。支持 Markdown 消息格式化、冷却去重、失败重试和推送历史追溯。

本领域不导入任何其他 domain 包，事件模型由 Composition Root（`cmd/server/main.go`）完成跨领域转换。

## 公开入口

| 类型 | 构造函数 | 用途 |
|------|----------|------|
| `AlertEvent` | — | 告警事件模型（通知领域内部类型） |
| `WebhookSender` (interface) | — | Webhook 消息发送契约 |
| `MockSender` | `NewMockSender()` | 测试用发送器，记录每次调用 |
| `WeComSender` | `NewWeComSender(webhookURL)` | 企业微信机器人 HTTP 发送器 |
| `CooldownTracker` | `NewCooldownTracker(minutes)` | 冷却去重追踪器 |
| `HistoryBuffer` | `NewHistoryBuffer(capacity)` | 推送历史环形缓冲区 |
| `Dispatcher` | `NewDispatcher(sender, cooldown, history, retryCount, retryInterval)` | 推送调度器 |
| `Service` | `NewService(dispatcher, enabled)` | 通知服务，通过 `HandleEvents(ctx, []AlertEvent)` 接收告警 |
| `HTTPHandler` | `NewHTTPHandler(history)` | HTTP 处理器（推送历史查询） |

## 依赖规则

- **不导入**任何其他 domain 包
- 可被 `cmd/server/main.go`（Composition Root）和 `api/route.go` 引用
- 使用 `net/http` 标准库，无第三方 HTTP 依赖

## 文件清单

| 文件 | 职责 |
|------|------|
| `event.go` | `AlertEvent` 告警事件模型 |
| `sender.go` | `WebhookSender` 接口 + `WeComSender` 实现 |
| `mock_sender.go` | `MockSender` 测试双 |
| `message.go` | Markdown 告警消息构建 |
| `dedup.go` | `CooldownTracker` 冷却去重 |
| `dispatcher.go` | `Dispatcher` 异步推送调度器 |
| `history.go` | `HistoryBuffer` 推送历史记录 |
| `service.go` | `Service` 通知服务编排 |
| `http_handler.go` | `HTTPHandler` 推送历史查询 API |
