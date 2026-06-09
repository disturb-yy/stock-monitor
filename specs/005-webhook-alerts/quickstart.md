# Quickstart: Webhook 告警推送

**Feature**: 005-webhook-alerts

本文档提供 Webhook 告警推送功能的手动验证步骤，无需依赖完整的异动检测链路。

## 前置条件

1. 企业微信群机器人 Webhook URL（在企业微信群聊 → 群设置 → 群机器人 → 添加 → 复制 Webhook 地址）
2. 本地 Go 环境（`go 1.25+`）
3. 项目已构建：`make build`

## 验证步骤

### Step 1: 配置 Webhook

编辑 `configs/config.yaml`，添加 webhook 段：

```yaml
webhook:
  enabled: true
  webhook_url: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY"
  cooldown_minutes: 30
  retry_count: 3
  retry_interval_seconds: 30
```

### Step 2: Mock 模式验证（不触发 Webhook 调用）

使用 Mock 数据源启动服务，Webhook 配置启用但不会自动推送（Mock 下无实际异动）。

```bash
make run
```

1. 确认服务启动日志中有类似 `notify: webhook enabled` 的日志
2. 调用推送历史接口验证路由注册：

```bash
curl -s http://localhost:30080/api/market/alerts/history | jq
```

预期响应：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "records": [],
    "total": 0
  }
}
```

### Step 3: 单元测试验证核心逻辑

```bash
go test ./domain/notify/... -v
```

预期覆盖：

- `TestWeComSender_Send_Success` — 模拟 HTTP 200 + errcode=0
- `TestWeComSender_Send_Failure` — 模拟 HTTP 500，验证重试
- `TestWeComSender_Send_RateLimited` — 模拟 HTTP 429，验证重试后成功
- `TestCooldownTracker_Skip` — 冷却时间内重复事件跳过
- `TestCooldownTracker_Allow` — 冷却时间过后允许推送
- `TestMergeMessage` — 多事件合并 Markdown 格式正确
- `TestHistoryBuffer_EvictOldest` — 超出容量后淘汰最旧记录
- `TestMockSender_RecordsCall` — MockSender 记录每次调用内容

### Step 4: 集成验证（需企业微信 Webhook URL）

1. 使用 Tushare 数据源 + 极低触发阈值，确保异动能够产生：

```yaml
provider:
  type: tushare
tushare:
  token: "${TUSHARE_TOKEN}"
anomaly:
  enabled: true
  rules:
    - type: price_change
      upper_threshold: 0.01    # 极低阈值，容易触发
      lower_threshold: -0.01
      enabled: true
webhook:
  enabled: true
  webhook_url: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY"
  cooldown_minutes: 0           # 禁用冷却以便多次验证
  retry_count: 1                # 减少重试加速验证
```

2. 启动服务并调用异动检测接口：

```bash
make run
# 另一个终端
curl -s http://localhost:30080/api/market/anomalies | jq '.events | length'
```

3. 检查企业微信群聊是否收到告警消息：

- 群内应显示一条 Markdown 格式消息，标题为 `⚠️ 异动告警`
- 内容包含触发异动的指数名称、异动类型、触发值和时间

4. 查询推送历史验证记录：

```bash
curl -s http://localhost:30080/api/market/alerts/history | jq '.data.total'
```

预期：`> 0`（表示至少有一条推送记录）

### Step 5: 验证未配置时的零影响

1. 删除或禁用 webhook 配置：

```yaml
webhook:
  enabled: false
```

2. 重启服务，确认：

```bash
# 异动检测接口仍正常工作
curl -s http://localhost:30080/api/market/anomalies | jq '.code'
# 预期: 0

# 推送历史路由不注册
curl -s http://localhost:30080/api/market/alerts/history | jq '.code'
# 预期: 404
```
