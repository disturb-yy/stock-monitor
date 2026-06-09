# Data Model: SQLite 持久化

**Feature**: 006-sqlite-persistence

## 实体关系

```text
HistoryStore (内存) ──写──▶ SQLiteStore (磁盘)
        ▲                       │
        │ 启动时加载             │ 按需回退查询
        │ (最近 maxDays)         │ (超期数据)
        │                       ▼
  HTTPHandler.GetHistory ◀── 合并结果
```

## 实体定义

### SQLiteStore（磁盘持久化层）

`domain/market/sqlite_store.go` — 封装 SQLite 连接和操作。

| 方法 | 说明 |
|------|------|
| `Open(path)` | 打开/创建数据库文件，自动建表 |
| `Close()` | 关闭数据库连接 |
| `Flush(quotes []IndexQuote)` | 批量写入行情记录（INSERT OR REPLACE） |
| `LoadRecent(maxDays int) ([]IndexQuote, error)` | 加载最近 maxDays 天数据 |
| `Query(symbol, start, end string) ([]IndexQuote, error)` | 按日期范围查询（用于超出内存范围时回退） |
| `PurgeBefore(days int)` | 删除超过保留天数的过期记录 |

### 数据库表: historical_quotes

| 列 | 类型 | 说明 |
|----|------|------|
| `symbol` | TEXT | 指数代码（如 "000001.SH"），联合主键之一 |
| `trade_date` | TEXT | 交易日期（"2006-01-02"），联合主键之一 |
| `name` | TEXT | 指数中文名称 |
| `open` | REAL | 开盘价 |
| `close` | REAL | 收盘价 |
| `high` | REAL | 最高价 |
| `low` | REAL | 最低价 |
| `volume` | REAL | 成交量 |
| `change_pct` | REAL | 涨跌幅（%） |
| `updated_at` | TEXT | 更新时间（ISO 8601） |

### 配置扩展

在 `pkg/config/config.go` 新增 `PersistenceConfig`：

```go
type PersistenceConfig struct {
    Enabled     bool   `yaml:"enabled"`      // 是否启用 SQLite 持久化
    DBPath      string `yaml:"db_path"`      // SQLite 文件路径
    MaxDays     int    `yaml:"max_days"`      // 最大保留天数
}
```

对应 `config.yaml`：

```yaml
persistence:
  enabled: true
  db_path: "data/market_history.db"
  max_days: 90
```

### HistoryStore 接口扩展

不新增独立接口，直接在 HistoryStore 上增加方法：

```go
// LoadFromSQLite 从 SQLite 加载最近 maxDays 天数据到内存。
func (s *HistoryStore) LoadFromSQLite(ctx context.Context, st *SQLiteStore) error

// FlushToSQLite 将内存中所有数据写入 SQLite（启动时或关闭前全量同步）。
func (s *HistoryStore) FlushToSQLite(ctx context.Context, st *SQLiteStore) error
```
