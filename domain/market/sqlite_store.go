// Package market 是市场行情领域。
// 本文件提供 SQLite 持久化层，用于将 HistoryStore 内存数据定期刷写到磁盘。
package market

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore 封装 SQLite 数据库连接和行情数据持久化操作。
// 作为 HistoryStore 的磁盘补充，不替代内存存储。
type SQLiteStore struct {
	db   *sql.DB
	path string // 数据库文件路径
}

// Open 打开或创建 SQLite 数据库文件并初始化表结构。
// path 为数据库文件路径，文件不存在时自动创建。
// 数据库损坏时自动重建。
func Open(path string) (*SQLiteStore, error) {
	// 确保目录存在
	if err := os.MkdirAll(dirOf(path), 0755); err != nil {
		return nil, fmt.Errorf("创建 SQLite 数据目录失败: %w", err)
	}

	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 数据库失败: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite 单写者
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	store := &SQLiteStore{db: db, path: path}

	// 检测数据库是否损坏
	if store.IsCorrupt() {
		slog.Warn("market: SQLite 文件已损坏，自动重建", "path", path)
		db.Close()
		os.Remove(path)

		// 重新打开
		db2, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
		if err != nil {
			return nil, fmt.Errorf("重建 SQLite 数据库失败: %w", err)
		}
		db2.SetMaxOpenConns(1)
		db2.SetMaxIdleConns(1)
		db2.SetConnMaxLifetime(time.Hour)
		store = &SQLiteStore{db: db2, path: path}
	}

	if err := store.createTable(); err != nil {
		store.db.Close()
		return nil, fmt.Errorf("创建 SQLite 表失败: %w", err)
	}

	slog.Info("market: SQLite 持久化已启用", "path", path)
	return store, nil
}

// dirOf 返回路径的目录部分。
func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return ""
}

// Close 关闭数据库连接。
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// createTable 创建行情数据表（如不存在）。
func (s *SQLiteStore) createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS historical_quotes (
		symbol      TEXT NOT NULL,
		trade_date  TEXT NOT NULL,
		name        TEXT NOT NULL,
		open        REAL NOT NULL,
		price       REAL NOT NULL,
		high        REAL NOT NULL,
		low         REAL NOT NULL,
		volume      INTEGER NOT NULL,
		change_percent REAL NOT NULL,
		updated_at  TEXT NOT NULL,
		PRIMARY KEY (symbol, trade_date)
	);
	CREATE INDEX IF NOT EXISTS idx_trade_date ON historical_quotes(trade_date);
	`
	_, err := s.db.Exec(query)
	return err
}

// scanQuote 从数据库行扫描出 IndexQuote。
func scanQuote(scanner interface {
	Scan(dest ...interface{}) error
}) (IndexQuote, error) {
	var q IndexQuote
	var tradeDate, updatedStr string
	err := scanner.Scan(
		&q.Symbol, &tradeDate, &q.Name,
		&q.Open, &q.Price, &q.High, &q.Low, &q.Volume, &q.ChangePercent,
		&updatedStr,
	)
	if err != nil {
		return q, err
	}
	q.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	if q.UpdatedAt.IsZero() {
		q.UpdatedAt, _ = time.Parse("2006-01-02", tradeDate)
	}
	return q, nil
}

// Flush 批量写入行情记录到 SQLite（INSERT OR REPLACE 去重）。
func (s *SQLiteStore) Flush(quotes []IndexQuote) error {
	if len(quotes) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO historical_quotes
		(symbol, trade_date, name, open, price, high, low, volume, change_percent, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("准备 SQL 语句失败: %w", err)
	}
	defer stmt.Close()

	for _, q := range quotes {
		tradeDate := q.UpdatedAt.Format("2006-01-02")
		_, err := stmt.Exec(
			q.Symbol, tradeDate, q.Name,
			q.Open, q.Price, q.High, q.Low, q.Volume, q.ChangePercent,
			q.UpdatedAt.Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("写入行情记录失败 (symbol=%s, date=%s): %w", q.Symbol, tradeDate, err)
		}
	}

	return tx.Commit()
}

// LoadRecent 从 SQLite 加载最近 maxDays 天的行情数据。
func (s *SQLiteStore) LoadRecent(maxDays int) ([]IndexQuote, error) {
	cutoff := time.Now().AddDate(0, 0, -maxDays).Format("2006-01-02")

	rows, err := s.db.Query(`
		SELECT symbol, trade_date, name, open, price, high, low, volume, change_percent, updated_at
		FROM historical_quotes
		WHERE trade_date >= ?
		ORDER BY trade_date DESC, symbol ASC
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("查询最近行情数据失败: %w", err)
	}
	defer rows.Close()

	var quotes []IndexQuote
	for rows.Next() {
		q, err := scanQuote(rows)
		if err != nil {
			return nil, fmt.Errorf("解析行情记录失败: %w", err)
		}
		quotes = append(quotes, q)
	}
	return quotes, rows.Err()
}

// Query 按指数代码和日期范围查询 SQLite 中的历史数据。
func (s *SQLiteStore) Query(symbol, start, end string) ([]IndexQuote, error) {
	query := `SELECT symbol, trade_date, name, open, price, high, low, volume, change_percent, updated_at
		FROM historical_quotes WHERE 1=1`
	args := []interface{}{}

	if symbol != "" {
		query += " AND symbol = ?"
		args = append(args, symbol)
	}
	if start != "" {
		query += " AND trade_date >= ?"
		args = append(args, start)
	}
	if end != "" {
		query += " AND trade_date <= ?"
		args = append(args, end)
	}
	query += " ORDER BY trade_date DESC, symbol ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询 SQLite 历史数据失败: %w", err)
	}
	defer rows.Close()

	var quotes []IndexQuote
	for rows.Next() {
		q, err := scanQuote(rows)
		if err != nil {
			return nil, fmt.Errorf("解析 SQLite 记录失败: %w", err)
		}
		quotes = append(quotes, q)
	}
	return quotes, rows.Err()
}

// PurgeBefore 删除超过 maxDays 天的过期记录。
func (s *SQLiteStore) PurgeBefore(maxDays int) error {
	cutoff := time.Now().AddDate(0, 0, -maxDays).Format("2006-01-02")
	result, err := s.db.Exec(`DELETE FROM historical_quotes WHERE trade_date < ?`, cutoff)
	if err != nil {
		return fmt.Errorf("清理过期数据失败: %w", err)
	}
	n, _ := result.RowsAffected()
	if n > 0 {
		slog.Debug("market: 已清理 SQLite 过期记录", "count", n, "before", cutoff)
	}
	return nil
}

// IsCorrupt 通过 PRAGMA integrity_check 检测数据库是否损坏。
func (s *SQLiteStore) IsCorrupt() bool {
	var result string
	err := s.db.QueryRow("PRAGMA integrity_check").Scan(&result)
	if err != nil || result != "ok" {
		slog.Warn("market: SQLite 完整性检查失败", "result", result, "error", err)
		return true
	}
	return false
}
