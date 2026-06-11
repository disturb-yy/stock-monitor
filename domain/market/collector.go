// Package market 是市场行情领域，负责 A 股指数行情数据的获取和展示。
package market

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

)

// HistoryStore 是采集行情数据的内存存储。
// 以指数代码为键，存储按日期降序排列的历史行情切片。
// 使用 sync.RWMutex 保证并发安全：采集侧持写锁，查询侧持读锁。
type HistoryStore struct {
	mu      sync.RWMutex
	data    map[string][]IndexQuote // key=指数代码，value=按日期降序的行情记录
	maxDays int                     // 每个指数最多保留天数
}

// NewHistoryStore 创建历史行情内存存储。
// maxDays 指定每个指数最多保留的天数，超出部分在每次追加时自动裁剪。
func NewHistoryStore(maxDays int) *HistoryStore {
	return &HistoryStore{
		data:    make(map[string][]IndexQuote),
		maxDays: maxDays,
	}
}

// Append 向指定指数追加一条行情记录。
// 同一交易日已有记录时采用新值覆盖（去重），追加后自动裁剪超出天数的旧数据。
func (s *HistoryStore) Append(symbol string, quote IndexQuote) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries := s.data[symbol]
	dateKey := quote.UpdatedAt.Format("2006-01-02")

	// 同日期去重：查找并替换已有记录
	found := false
	for i, entry := range entries {
		if entry.UpdatedAt.Format("2006-01-02") == dateKey {
			entries[i] = quote
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, quote)
	}

	// 按日期降序排列
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].UpdatedAt.After(entries[j].UpdatedAt)
	})

	// 裁剪超出天数的旧数据
	if len(entries) > s.maxDays {
		entries = entries[:s.maxDays]
	}

	s.data[symbol] = entries
}

// Query 按指数代码和日期范围查询历史行情。
// symbol 为空时返回所有指数数据；start/end 格式为 "2006-01-02"。
// 返回的数据按日期降序排列。
func (s *HistoryStore) Query(symbol, start, end string) (map[string][]IndexQuote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	startTime, err := time.Parse("2006-01-02", start)
	if err != nil {
		return nil, fmt.Errorf("start 日期格式错误: %w", err)
	}

	endTime := time.Now()
	if end != "" {
		endTime, err = time.Parse("2006-01-02", end)
		if err != nil {
			return nil, fmt.Errorf("end 日期格式错误: %w", err)
		}
		// end 包含当天全天
		endTime = endTime.Add(24*time.Hour - time.Second)
	}

	result := make(map[string][]IndexQuote)
	symbols := []string{symbol}
	if symbol == "" {
		symbols = nil
		for sym := range s.data {
			symbols = append(symbols, sym)
		}
		sort.Strings(symbols)
	}

	for _, sym := range symbols {
		entries, ok := s.data[sym]
		if !ok {
			continue
		}
		var filtered []IndexQuote
		for _, entry := range entries {
			if !entry.UpdatedAt.Before(startTime) && !entry.UpdatedAt.After(endTime) {
				filtered = append(filtered, entry)
			}
		}
		if len(filtered) > 0 {
			result[sym] = filtered
		}
	}

	return result, nil
}

// LoadFromSQLite 从 SQLite 加载最近 maxDays 天的行情数据到内存。
func (s *HistoryStore) LoadFromSQLite(st *SQLiteStore) error {
	if st == nil {
		return nil
	}
	quotes, err := st.LoadRecent(s.maxDays)
	if err != nil {
		return err
	}
	for _, q := range quotes {
		s.Append(q.Symbol, q)
	}
	slog.Info("market: 已从 SQLite 加载历史数据", "days", s.maxDays, "records", len(quotes))
	return nil
}

// Len 返回当前存储的指数总数。
func (s *HistoryStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// Collector 是后台定时行情采集器。
// 在交易时段内按配置的间隔周期（time.Ticker）自动拉取所有追踪指数的行情数据，
// 并写入 HistoryStore。非交易时段休眠等待，支持优雅关闭。
type Collector struct {
	provider    *TushareProvider  // 行情数据源（仅 Tushare 模式有效）
	store       *HistoryStore     // 内存历史存储
	calendar    *TradingCalendar  // 交易日历（判断采集时机）
	intervalMin int               // 采集间隔（分钟）
	sqlite       *SQLiteStore     // SQLite 持久化存储（可为 nil）
	maxPurgeDays int              // SQLite 最大保留天数

	ticker *time.Ticker
	done   chan struct{}
}

// NewCollector 创建定时采集器。
// provider 为 TushareProvider 实例，store 为历史存储，
// calendar 用于交易日判断，intervalMinutes 为采集间隔（分钟）。
func NewCollector(provider *TushareProvider, store *HistoryStore, calendar *TradingCalendar, intervalMinutes int, sqlite *SQLiteStore, maxPurgeDays int) *Collector {
	return &Collector{
		provider:     provider,
		store:        store,
		calendar:     calendar,
		intervalMin:  intervalMinutes,
		sqlite:       sqlite,
		maxPurgeDays: maxPurgeDays,
		done:         make(chan struct{}),
	}
}

// Start 启动后台采集循环。
// 通过传入的 context 启动一个 goroutine，按配置的间隔执行采集。
// 每次 tick 到达时先判断当前是否为交易时段，非交易时段则跳过。
func (c *Collector) Start(ctx context.Context) {
	c.ticker = time.NewTicker(time.Duration(c.intervalMin) * time.Minute)

	slog.Info("定时采集器已启动",
		"interval_minutes", c.intervalMin,
		"max_history_days", c.store.maxDays,
	)

	go func() {
		// 启动后立即执行一次采集
		c.collectIfTrading(ctx)

		for {
			select {
			case <-c.ticker.C:
				c.collectIfTrading(ctx)
			case <-c.done:
				slog.Info("定时采集器已停止")
				return
			}
		}
	}()
}

// Stop 优雅关闭采集器。
// 停止 ticker，关闭 done channel，等待当前采集周期结束（最多 30s）。
func (c *Collector) Stop() {
	if c.ticker != nil {
		c.ticker.Stop()
	}
	close(c.done)
}

// collectIfTrading 在交易时段才执行采集，否则记录跳过原因。
func (c *Collector) collectIfTrading(ctx context.Context) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	now := time.Now().In(loc)

	// 第一步：交易日判断
	if c.calendar != nil && !c.calendar.IsTradingDay(now) {
		slog.Debug("非交易日，跳过采集", "date", now.Format("2006-01-02"))
		return
	}

	// 第二步：时段判断
	status, isTrading := determineMarketSession(now)
	if !isTrading {
		slog.Debug("非交易时段，跳过采集",
			"status", status,
			"at", now.Format("15:04:05"),
		)
		return
	}

	slog.Info("开始采集行情数据", "at", now.Format("2006-01-02 15:04:05"))
	c.collect(ctx)
}

// collect 执行一次完整采集：拉取所有追踪指数行情并写入 HistoryStore。
func (c *Collector) collect(ctx context.Context) {
	quotes, err := c.provider.GetIndexQuotes(ctx)
	if err != nil {
		slog.Warn("采集行情失败", "error", err)
		return
	}

	for _, q := range quotes {
		c.store.Append(q.Symbol, q)
	}

	// SQLite 持久化写入
	if c.sqlite != nil {
		if err := c.sqlite.Flush(quotes); err != nil {
			slog.Warn("SQLite 持久化写入失败", "error", err)
		}
		// 清理过期数据
		if err := c.sqlite.PurgeBefore(c.maxPurgeDays); err != nil {
			slog.Warn("SQLite 过期数据清理失败", "error", err)
		}
	}

	slog.Info("采集完成", "count", len(quotes))
}
