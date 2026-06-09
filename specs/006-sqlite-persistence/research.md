# Research: SQLite 持久化

**Feature**: 006-sqlite-persistence
**Date**: 2026-06-10

## 1. SQLite 驱动选择

### Decision

使用 `github.com/mattn/go-sqlite3`（CGo 驱动），配合 `database/sql` 标准接口。

### Rationale

- 最成熟的 Go SQLite 驱动，94k+ GitHub stars，生产验证充分
- 标准 `database/sql` 接口，代码与未来可能的数据库迁移兼容
- 支持 `:memory:` 模式用于单元测试
- Linux CGo 编译无额外依赖（只需 gcc）

### Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| `modernc.org/sqlite`（纯 Go） | 编译速度慢、二进制体积大，本项目无交叉编译需求 |
| 直接文件 I/O（JSON/CSV） | 无法按日期范围高效查询，需全量加载解析 |
| BoltDB / bbolt | 额外依赖，K-V 模型需自行实现日期范围查询 |

## 2. 表结构设计

### Decision

单表 `historical_quotes`，字段与 `IndexQuote` 一一对应，`(symbol, trade_date)` 作为联合唯一键。

```sql
CREATE TABLE IF NOT EXISTS historical_quotes (
    symbol      TEXT NOT NULL,
    trade_date  TEXT NOT NULL,  -- YYYY-MM-DD
    name        TEXT NOT NULL,
    open        REAL NOT NULL,
    close       REAL NOT NULL,
    high        REAL NOT NULL,
    low         REAL NOT NULL,
    volume      REAL NOT NULL,
    change_pct  REAL NOT NULL,
    updated_at  TEXT NOT NULL,  -- ISO 8601
    PRIMARY KEY (symbol, trade_date)
);
CREATE INDEX IF NOT EXISTS idx_trade_date ON historical_quotes(trade_date);
```

### Rationale

- 单表足够，6 指数 × 90 天 = 540 行，无需分表或分区
- `(symbol, trade_date)` 唯一键天然去重，INSERT OR REPLACE 无需手动判断
- `trade_date` 索引支持按日期范围高效查询
- 字段使用 `REAL`（SQLite 浮点），与 Go `float64` 直接映射

## 3. 写入策略

### Decision

采集后立即写入（per-collect flush），使用 `INSERT OR REPLACE`。不在每次 `Append` 时写入，而是在 `collect()` 完成后批量写入。

### Rationale

- 采集周期为 5 分钟，每次 6 条记录，批量写入开销可忽略（<5ms）
- `INSERT OR REPLACE` 处理同日期覆盖，无需应用层判断
- 避免每次 `Append` 都触发 I/O（`Append` 调用频率可能更高）

### Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 每条 Append 立即写入 | 频繁 I/O，且 Append 调用方不止采集器 |
| 定时批量刷写（如每 5 分钟） | 增加崩溃丢失窗口，复杂度换不来收益 |
| WAL 模式 + 异步写入 | 数据量太小，WAL 无意义 |

## 4. 启动加载策略

### Decision

启动时执行 `SELECT * FROM historical_quotes WHERE trade_date >= ? ORDER BY trade_date DESC`，加载最近 maxDays 天数据到内存 HistoryStore。

### Rationale

- 与 clarify 阶段决策一致：仅加载最近 maxDays 天
- 使用参数化查询防止 SQL 注入
- 按日期降序返回，直接匹配 HistoryStore 内部顺序

## 5. 过期数据清理

### Decision

在采集器每次写入后检查：执行 `DELETE FROM historical_quotes WHERE trade_date < date('now', '-N days')`，N 为最大保留天数。

### Rationale

- 附加在批量写入后，频率低（5 分钟一次），开销可忽略
- 使用 SQLite 内置 `date()` 函数，无需应用层计算日期
- 清理失败仅记录日志，不阻塞采集

## 6. 历史查询合并策略

### Decision

查询时先走内存 HistoryStore.Query()，若返回数据不完整（日期范围超出内存覆盖），再补查 SQLite。两个结果集合并去重时，内存优先。

### Rationale

- 内存查询零 I/O，命中率高（最近 30 天查询占绝大多数）
- SQLite 仅在"查更早数据"时触发，频率低
- 去重以 (symbol, trade_date) 为键，内存记录覆盖 SQLite 记录
