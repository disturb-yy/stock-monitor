# Contracts: Webhook 告警推送

**Feature**: 005-webhook-alerts

## 1. HTTP API: 推送历史查询

### GET /api/alerts/history

查询最近 N 条告警推送记录。路由始终注册（即使 webhook 未启用）；webhook 关闭时返回空记录列表。

#### 请求

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `limit` | `int` (query) | 否 | 返回记录数，默认 50，最大 200 |

#### 200 响应

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "records": [
      {
        "id": "d4e5f6a7-b8c9-...",
        "event_ids": ["evt-001", "evt-002"],
        "status": "success",
        "message": "## ⚠️ 异动告警\n\n**上证指数** (000001.SH) ...",
        "error": "",
        "retry_count": 0,
        "delivered_at": "2026-06-09T14:35:00+08:00",
        "completed_at": "2026-06-09T14:35:01+08:00"
      },
      {
        "id": "e5f6a7b8-c9d0-...",
        "event_ids": ["evt-003"],
        "status": "failed",
        "message": "## ⚠️ 异动告警\n\n**创业板指** (399006.SZ) ...",
        "error": "HTTP 500: internal server error",
        "retry_count": 3,
        "delivered_at": "2026-06-09T14:30:00+08:00",
        "completed_at": "2026-06-09T14:31:30+08:00"
      },
      {
        "id": "f6a7b8c9-d0e1-...",
        "event_ids": ["evt-004"],
        "status": "skipped_cooldown",
        "message": "",
        "error": "冷却中 (symbol=000001.SH, type=price_up)",
        "retry_count": 0,
        "delivered_at": "2026-06-09T14:32:00+08:00",
        "completed_at": "2026-06-09T14:32:00+08:00"
      }
    ],
    "total": 3
  }
}
```

#### 路由注册

在 `api/route.go` 中条件注册：

```go
if notifyHandler != nil {
    marketGroup.GET("/alerts/history", notifyHandler.GetHistory)
}
```

## 2. Webhook POST: 企业微信推送

### POST https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key={KEY}

#### 请求体

```json
{
  "msgtype": "markdown",
  "markdown": {
    "content": "## ⚠️ 异动告警\n\n**上证指数** (000001.SH) 触发 **上涨异动**\n> 涨幅：+3.52%（阈值：3.00%）\n> 触发时间：2026-06-09 14:35\n"
  }
}
```

#### 成功响应

```json
{
  "errcode": 0,
  "errmsg": "ok"
}
```

#### 失败响应

```json
{
  "errcode": 45009,
  "errmsg": "api freq limit"
}
```

#### 重试规则

| HTTP 状态码 / errcode | 行为 |
|------------------------|------|
| 200 + errcode=0 | 推送成功，记录 `success` |
| HTTP 4xx（不含 429） | 不可重试，记录 `failed` + 错误信息 |
| HTTP 429 / errcode=45009 / errcode=45033 | 触发限流，退避后重试 |
| HTTP 5xx / 网络超时 / DNS 错误 | 临时故障，按间隔重试 |

## 3. 内部回调契约: anomaly → notify

异动检测引擎通过 Composition Root 注入的回调函数与通知领域通信：

```go
// 在 domain/anomaly/engine.go 中新增
type EventCallback func(ctx context.Context, events []AnomalyEvent)

// Engine 增加 SetCallback 方法
func (e *Engine) SetCallback(cb EventCallback) { ... }
```

- 回调在 `Engine.Detect()` 返回事件列表后同步调用
- 回调内部异步处理推送（不阻塞检测引擎）
- `cmd/server/main.go` 负责将 `notifier.OnAnomalyEvents` 绑定到回调

## 4. YAML 配置契约: configs/config.yaml

```yaml
# Webhook 告警推送配置
webhook:
  enabled: false                    # 是否启用企业微信推送
  webhook_url: ""                   # 企业微信机器人 Webhook URL
  cooldown_minutes: 30              # 同一指数+类型冷却时间（分钟），0 禁用
  retry_count: 3                    # 最大重试次数
  retry_interval_seconds: 30        # 重试间隔（秒）
```
