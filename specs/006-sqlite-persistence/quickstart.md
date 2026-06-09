# Quickstart: SQLite 持久化

**Feature**: 006-sqlite-persistence

验证 SQLite 持久化功能的端到端流程。

## 前置条件

- Go 1.25+ + CGo（`gcc`）
- `make build` 已通过

## Step 1: 启用配置

编辑 `configs/config.yaml`：

```yaml
persistence:
  enabled: true
  db_path: "data/market_history.db"
  max_days: 90
```

## Step 2: 首次启动验证

```bash
make run
```

预期日志：
```
market: SQLite 数据库已创建, path=data/market_history.db
```

检查文件：
```bash
ls -l data/market_history.db   # 应存在
```

## Step 3: 采集后验证数据写入

1. 等待一次采集完成（日志中出现 "采集完成"）
2. 检查 SQLite 数据：

```bash
sqlite3 data/market_history.db "SELECT symbol, trade_date, close FROM historical_quotes ORDER BY trade_date DESC LIMIT 5;"
```

预期：返回最近采集的行情记录。

## Step 4: 重启后数据恢复验证

1. 记录当前数据条数：

```bash
sqlite3 data/market_history.db "SELECT COUNT(*) FROM historical_quotes;"
```

2. 重启服务：

```bash
# Ctrl+C 停止，然后
make run
```

3. 立即查询历史：

```bash
curl -s http://localhost:30083/api/market/history?symbol=000001.SH | jq '.data | length'
```

预期：返回重启前采集的数据（条数 > 0）。

## Step 5: 超内存范围查询

1. 修改内存保留为 7 天、SQLite 保留为 90 天（`collector.max_history_days: 7`）
2. 确保 SQLite 中有 30 天数据（已提前采集或手动插入测试数据）
3. 查询 20 天前的数据：

```bash
START=$(date -d '20 days ago' +%Y-%m-%d)
END=$(date -d '19 days ago' +%Y-%m-%d)
curl -s "http://localhost:30083/api/market/history?symbol=000001.SH&start=$START&end=$END" | jq '.data | length'
```

预期：返回 1 条数据（从 SQLite 回退查询）。

## Step 6: 异常场景验证

文件损坏：

```bash
# 停止服务，损坏文件
echo "corrupted" > data/market_history.db
# 重启
make run
```

预期日志：
```
market: SQLite 文件已损坏，自动重建
```

## Step 7: 单测验证

```bash
go test ./domain/market/... -v -run SQLite
```

预期覆盖：建表、写入、查询、加载、清理、损坏重建。
