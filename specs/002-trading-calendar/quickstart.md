# Quickstart: 交易日历接入

## 前置条件

1. 已完成 001-tushare-real-provider 功能（Tushare Provider 可正常工作）
2. `provider.type` 为 `tushare`

## 验证步骤

### 1. 启动服务（含交易日历初始化）

```bash
make run
```

预期日志包含日历加载信息（首次启动从 Tushare 拉取）。

### 2. 验证节假日识别

```bash
# 查一个已知节假日（如元旦 2026-01-01，当时是周四）
# 由于系统判断当前日期，测试时可通过系统时间模拟

# 正常请求市场状态
curl -s http://localhost:8080/api/market/status | jq .
```

### 3. 验证 tradeDate 准确性

```bash
curl -s http://localhost:8080/api/market/status | jq '.data.tradeDate'

# 预期: 返回最近交易日（今天或前一交易日）
```

### 4. 验证缓存持久化

```bash
# 检查缓存文件是否生成
cat data/trading_calendar.json | jq '.start_date, .end_date, (.days | length)'

# 预期: 包含前后各 6 个月数据，数百条记录
```

### 5. 重启验证（使用缓存）

```bash
# 重启服务
make run

# 日志应显示"using cached calendar"，而非重新拉取
```

### 6. 验证过期警告

修改 `data/trading_calendar.json` 中的 `end_date` 为过去日期，重启：

```bash
# 预期日志: "calendar expired, needs refresh"
```
