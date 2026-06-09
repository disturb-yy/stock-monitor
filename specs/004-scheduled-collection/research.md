# Research: 定时采集

## 1. 调度机制

**Decision**: 使用 Go 标准库 `time.Ticker` 实现周期性采集，辅以 context 控制启停。

**Rationale**: Ticker 是 Go 原生定时器，无需引入第三方调度库（如 cron）。采集间隔为分钟级，Ticker 精度足够。通过 context.WithCancel 控制优雅关闭——收到退出信号时，等待当前采集周期完成后退出。

**Alternatives considered**:
- `robfig/cron`: 功能强大但引入外部依赖，分钟级间隔用 cron 过度
- `time.AfterFunc` 递归调度: 可避免 goroutine 泄漏但代码不如 Ticker 直观

## 2. 交易时段判断

**Decision**: 复用现有 `TradingCalendar` + 时段判断逻辑。采集器持有 `*TradingCalendar`，每个 tick 到达时先调用 `IsTradingDay(now)` + `determineMarketSession(now)`，非交易时段跳过。

**Rationale**: 避免重复实现。交易日历已覆盖节假日/调休，时段判断已覆盖盘前/午休/收盘。

**采集窗口**: 交易时段 9:30-11:30 和 13:00-15:00。盘前、午休、收盘后均不采集。

## 3. 内存存储结构

**Decision**: 使用 `map[string][]IndexQuote`（key=指数代码，value=按日期降序的行情切片）。

**淘汰策略**:
- 每次采集后检查每个指数的切片长度
- 超过 `maxDays`（默认 30 天）时截断尾部
- 日期去重：同一指数同一交易日仅保留最新一条

**线程安全**: 使用 `sync.RWMutex` 保护，采集器写锁、API 查询读锁。

**Alternatives considered**:
- 文件存储（JSON/CSV）: 读写性能差，查询不便
- SQLite: 超出 Phase 2 范围，留到数据库存储阶段
- 环形缓冲区: 查询灵活性不足（需要按日期范围筛选）

## 4. 历史查询接口

**Decision**: `GET /api/market/history?start=2026-06-01&end=2026-06-09&symbol=000001.SH`

**参数**:
- `start`: 起始日期（含），YYYY-MM-DD，必填
- `end`: 截止日期（含），YYYY-MM-DD，可选（默认今天）
- `symbol`: 指数代码，可选（默认全部指数）

**响应格式**: 与 `GET /api/market/indices` 相似，items 按 date 降序排列。

## 5. 优雅关闭

**Decision**: 采集器实现 `Start(ctx)` 和 `Stop()` 方法。`main.go` 在收到 SIGINT/SIGTERM 时调用 `Stop()`，等待当前采集完成（最多 30 秒超时）。

```go
type Collector struct {
    ticker   *time.Ticker
    done     chan struct{}
    // ...
}

func (c *Collector) Start(ctx context.Context) {
    go func() {
        for {
            select {
            case <-c.ticker.C:
                c.collect(ctx)
            case <-c.done:
                return
            }
        }
    }()
}

func (c *Collector) Stop() {
    c.ticker.Stop()
    close(c.done)
}
```

## 6. 配置结构

```yaml
collector:
  enabled: true
  interval_minutes: 5    # 采集间隔（分钟）
  max_history_days: 30    # 内存最多保留天数
```
