# Research: Webhook 告警推送

**Feature**: 005-webhook-alerts
**Date**: 2026-06-09

## 1. 企业微信机器人 Webhook API

### Decision

使用企业微信群机器人 Webhook，POST JSON 消息体到 `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=KEY`。

### Rationale

- 企业微信群机器人是官方支持的免费接入方式，无需企业认证或复杂 OAuth
- 支持 text 和 markdown 两种消息类型
- 消息格式简单：POST JSON，`{"msgtype":"markdown","markdown":{"content":"..."}}`
- 无需加签（企业微信机器人 Webhook 仅靠 URL 中的 key 做鉴权）

### Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 企业微信应用消息 API | 需要企业认证、CorpID/CorpSecret、access_token 管理，复杂度远超需求 |
| 飞书/钉钉 | 用户明确仅需企业微信（已在 clarify 阶段确认） |
| 微信服务号模板消息 | 需服务号认证、用户关注、模板审批，不适用于内部运维告警 |

### API 关键约束

- 每个机器人每分钟最多发送 20 条消息
- Markdown 支持：标题（# ~ ######）、加粗、链接、引用、字体颜色（仅限内置颜色）
- Markdown 不支持：表格、有序/无序列表、图片
- `content` 字段最大 4096 字节
- HTTP 响应：`{"errcode":0,"errmsg":"ok"}` 表示成功；非 0 errcode 或 HTTP 非 200 表示失败

### 参考

- [企业微信群机器人配置说明](https://developer.work.weixin.qq.com/document/path/91770)

## 2. HTTP 重试策略

### Decision

使用指数退避重试，仅对临时故障（5xx、429、网络超时）重试，4xx 直接失败。默认重试 3 次，间隔 30 秒（固定间隔）。

### Rationale

- 企业微信的 429（限流）和 500（服务错误）是临时性故障，重试有恢复可能性
- 400/401/404 表示客户端错误（URL 无效、权限不足），重试无意义，立即暴露问题帮助运维排查
- 采用固定间隔而非指数退避：告警量小（每分钟最多几个事件），指数退避的小延迟收益不明显，固定间隔更简单且便于预测
- 标准库 `net/http` 即可实现，无需引入 `go-retryablehttp` 等第三方库

### Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 指数退避随机抖动 | 告警量太小，不需要复杂退避算法 |
| 不重试 | 限流等临时故障会丢失告警 |
| 统一重试所有状态码 | 4xx 重试浪费资源且延迟暴露配置错误 |

## 3. 内存冷却去重

### Decision

使用 `sync.Map` 存储冷却记录，key 为 `"symbol:anomaly_type"`，value 为最近推送时间戳。推送前检查：若 `now - lastPushTime < cooldown` 则跳过。

### Rationale

- `sync.Map` 天然并发安全，无需额外加锁
- 6 个指数 × 5 种异动类型 = 最多 30 个 key，内存占用可忽略
- 定时清理过期记录（每 5 分钟扫描一次），避免无限增长
- 冷却时间可配置（默认 30 分钟）

### Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| `map[string]time.Time` + `sync.RWMutex` | 需要手动管理锁，代码更复杂 |
| Redis 去重 | Phase 2 无数据库依赖，引入 Redis 违反阶段约束 |
| 无冷却 | 重复告警骚扰用户，破坏信任度 |

## 4. 推送历史环形缓冲区

### Decision

使用固定容量的切片 + 读写索引实现线程安全的环形缓冲区。容量默认 1000 条，超出后覆盖最旧记录。提供查询接口返回按时间降序的切片。

### Rationale

- 固定容量保证内存不无限增长
- 环形缓冲区 O(1) 写入和 O(n) 查询（n=1000，可接受）
- 无需排序——写入选代顺序自然按时间递增
- 查询时快照当前缓冲区内容，拷贝返回避免并发问题

### Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| `container/list` 双向链表 | 需要遍历才能淘汰，O(n) 写入 |
| 无限增长切片 | 内存失控风险 |
| Channel + goroutine | 增加复杂性无显著收益 |

## 5. 多事件合并推送

### Decision

异动检测完成后收集所有事件，按冷却过滤后合并为一条 Markdown 消息：使用 `\n---\n` 分隔线区分各条异动。仅在至少有一条符合推送条件的事件时才发送。

### Rationale

- 每轮检测可能有 0~N 个事件，合并发送减少 HTTP 调用次数（缓解每分钟 20 条限流）
- 分隔线在 Markdown 中渲染为水平线，视觉区分清晰
- 若全部被冷却过滤，本轮不发送（避免发送空消息或仅含分隔线的消息）

### Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 逐条独立发送 | 触发 20 条/分钟限流风险更高 |
| 仅推送第一条 | 丢失其他重要异动信息（已在 clarify 阶段选择 B） |
