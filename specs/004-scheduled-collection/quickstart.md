# Quickstart: 定时采集

## 前置条件

- Tushare Provider 正常工作
- 交易日历可用

## 配置

```yaml
collector:
  enabled: true
  interval_minutes: 5
  max_history_days: 30
```

## 验证步骤

### 1. 启动服务

```bash
make run
```

日志应包含 "采集器已启动" 及间隔信息。

### 2. 验证自动采集

等待第一个采集周期（≤ 5 分钟），日志应包含采集记录。

### 3. 查询历史数据

```bash
# 查上证指数今天的数据
curl -s "http://localhost:30083/api/market/history?symbol=000001.SH&start=$(date +%Y-%m-%d)" | jq .

# 预期: quotes 数组包含至少一条记录
```

### 4. 验证多指数查询

```bash
curl -s "http://localhost:30083/api/market/history?start=$(date -d '3 days ago' +%Y-%m-%d)" | jq '.data.indices | keys'

# 预期: 6 个指数代码
```

### 5. 验证非交易时段跳过

非交易时段启动服务，检查无采集日志。

### 6. 验证间隔配置

修改 `interval_minutes: 1`，重启后观察两次采集时间差 ≈ 1 分钟。
