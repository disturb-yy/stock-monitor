# Quickstart: 每日收盘总结推送

**Feature**: 007-daily-summary

## 前置条件

1. 服务已启动，Webhook 已配置企业微信 URL
2. 交易日历已初始化（`data/trading_calendar.json` 存在）
3. 采集器已运行过（至少有一条当日行情数据在 HistoryStore 中）

## 验证步骤

### Step 1: 启用收盘总结功能

编辑 `configs/config.yaml`：

```yaml
daily_summary:
  enabled: true
  push_time: "15:15"
```

重启服务。

### Step 2: Mock 模式验证（不触发 Webhook 真实推送）

使用 Mock 数据源启动服务，Webhook 配置启用但不会自动推送（Mock 下使用 MockSender）。

```bash
make run
```

### Step 3: 查询收盘总结 JSON

```bash
# 查询当日收盘总结 JSON
curl -s http://localhost:30083/api/market/daily-summary | jq

# 预期：返回 data.trade_date、data.up_count/down_count、data.indices 等字段
# 非交易日返回 data.is_trading_day = false
```

验证 JSON 结构包含：
- `up_count / down_count / flat_count / total_indices`
- `top_gainer` 和 `top_loser` (含 symbol/name/change_percent)
- `indices` 数组，每项含 symbol/name/change_percent
- `anomaly_summary` 含 total_count 和 items 数组

### Step 4: 手动触发收盘总结推送

```bash
# 手动触发（异步，立即返回）
curl -s -X POST http://localhost:30083/api/market/daily-summary | jq

# 预期：{"code":0, "msg":"收盘总结推送已触发", "data":{"triggered":true}}
```

等待 2-3 秒后查询推送历史：

```bash
curl -s http://localhost:30083/api/market/alerts/history | jq '.data.records[] | select(.message | contains("收盘总结"))'
```

### Step 5: 单元测试

```bash
go test ./domain/daily/... -v
```

预期覆盖：
- `TestBuilder_BuildSummary_Success` — 正常生成总结
- `TestBuilder_BuildSummary_PartialData` — 部分数据缺失，标注 partial
- `TestBuilder_BuildSummary_NoAnomalies` — 无异动时 anomaly_summary.total_count = 0
- `TestScheduler_TradingDay` — 交易日按时触发
- `TestScheduler_NonTradingDay` — 非交易日不触发
- `TestSummaryToMarkdown` — Markdown 格式完整

### Step 6: 集成验证（需企业微信 Webhook + 真实数据）

1. 配置 Tushare Token 和企业微信 Webhook URL
2. 在交易日下午 15:15 等待自动推送，或手动触发验证

```bash
# 手动触发（生产环境）
curl -s -X POST http://localhost:30083/api/market/daily-summary | jq

# 检查企业微信群是否收到 📊 收盘总结 Markdown 消息
# 消息应包含：指数涨跌幅列表、涨跌统计、异动摘要
```

### Step 7: 数据异常场景验证

关闭 Tushare 网络或使用 invalid token，手动触发推送：

```bash
# 预期消息结尾包含 ⚠️ 数据异常 标记
curl -s -X POST http://localhost:30083/api/market/daily-summary | jq
```

查询 JSON 确认 `data_status: "partial"` 或 `"failed"`。
