# Data Model: Webhook 告警推送

**Feature**: 005-webhook-alerts

## 实体关系

```text
WebhookConfig ──1:1──▶ Notifier ──produces──▶ DeliveryRecord (0..N)
                                  │
                                  ├──uses──▶ CooldownTracker (1:1, 内部)
                                  └──calls──▶ WebhookSender (interface)
                                                    │
                                                    ├──▶ WeComSender (企业微信实现)
                                                    └──▶ MockSender (测试用)
```

## 实体定义

### WebhookConfig（推送配置）

从 YAML 配置文件加载的静态配置，运行期间不可变。

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `enabled` | `bool` | 是否启用 Webhook 推送 | `false` |
| `webhook_url` | `string` | 企业微信机器人 Webhook 完整 URL | `""` |
| `cooldown_minutes` | `int` | 冷却时间（分钟），0 表示禁用冷却 | `30` |
| `retry_count` | `int` | 最大重试次数 | `3` |
| `retry_interval_seconds` | `int` | 重试间隔（秒） | `30` |

### Notifier（推送调度器）— 运行时对象

由 Composition Root 从 `WebhookConfig` 创建，注册到异动检测引擎作为事件回调。

| 职责 | 说明 |
|------|------|
| 接收异动事件批次 | `OnAnomalyEvents([]AnomalyEvent)` 回调入口 |
| 冷却过滤 | 通过 CooldownTracker 判定每条事件是否应推送 |
| 消息构建 | 将有效事件合并为一条 Markdown 消息 |
| 异步发送 | 在新 goroutine 中调用 WebhookSender.Send() |
| 重试 | 失败时按 retry_count 和 retry_interval 重试 |
| 记录历史 | 每次推送（含最终成功/失败/去重）写入 DeliveryRecord |

### DeliveryRecord（推送记录）

存储在环形缓冲区中，通过 HTTP API 查询。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 记录唯一标识（UUID） |
| `event_ids` | `[]string` | 本条推送包含的异动事件 ID 列表（合并模式下可能多个） |
| `status` | `DeliveryStatus` | `success` / `failed` / `skipped_cooldown` / `retrying` |
| `message` | `string` | 消息内容摘要（截取前 200 字符） |
| `error` | `string` | 失败时的错误信息 |
| `retry_count` | `int` | 已重试次数 |
| `delivered_at` | `time.Time` | 首次推送尝试时间 |
| `completed_at` | `time.Time` | 最终完成时间（成功或最终失败） |

#### DeliveryStatus 枚举

| 值 | 含义 |
|----|------|
| `success` | 推送成功（企业微信返回 errcode=0） |
| `failed` | 最终失败（重试耗尽或 4xx 不可重试） |
| `skipped_cooldown` | 被冷却机制跳过（未实际发送） |
| `retrying` | 重试中（中间状态，仅最新一次推送有此状态） |

### CooldownTracker（冷却追踪）— 运行时对象

内存键值存储，Notifier 内部使用。

| Key | Value |
|-----|-------|
| `"symbol:anomaly_type"`（如 `"000001.SH:price_up"`） | `time.Time`（最近一次成功推送的时间） |

- 写入：推送成功后更新
- 查询：推送前检查 `now - value < cooldown`，若成立则跳过
- 清理：定时扫描（每 5 分钟），删除超过冷却时间 2 倍的过期记录

### 消息结构（Markdown）

发送到企业微信的消息体：

```json
{
  "msgtype": "markdown",
  "markdown": {
    "content": "## ⚠️ 异动告警\n\n**上证指数** (000001.SH) 触发 **上涨异动**\n> 涨幅：+3.52%（阈值：3.00%）\n> 触发时间：2026-06-09 14:35\n\n---\n\n**创业板指** (399006.SZ) 触发 **放量异动**\n> 成交量倍数：2.3x（阈值：2.0x）\n> 触发时间：2026-06-09 14:35"
  }
}
```

消息字段映射（从 `AnomalyEvent` 转换）：

| AnomalyEvent 字段 | Markdown 位置 | 转换规则 |
|-------------------|---------------|----------|
| `IndexName` + `Symbol` | 标题行 | `**{IndexName}** ({Symbol})` |
| `Type` | 副标题 | 映射为中文：`price_up→上涨异动`、`volume_spike→放量异动` 等 |
| `TriggerValue` | 触发值行 | 涨跌幅加 `%`，成交量加 `x` |
| `Threshold` | 阈值行 | 与 TriggerValue 同单位 |
| `TriggeredAt` | 时间行 | 格式化为 `YYYY-MM-DD HH:MM` |

### WebhookSender 接口（定义在 domain/notify 内）

```go
// WebhookSender 是 Webhook 消息发送的抽象接口。
// 提供企业微信真实发送器和 Mock 测试发送器两种实现。
type WebhookSender interface {
    // Send 发送 Markdown 消息到 Webhook 端点。
    // content 为已构建好的 Markdown 文本（不含外层 JSON 包装）。
    // 返回 HTTP 状态码和可能的错误。
    Send(ctx context.Context, content string) (statusCode int, err error)
}
```
