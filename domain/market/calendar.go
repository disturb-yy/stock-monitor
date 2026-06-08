package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/disturb-yy/stock-monitor/pkg/tushare"
)

// TradingCalendar 缓存 A 股交易日历数据。
// 数据来源为 Tushare trade_cal API，本地以 JSON 文件持久化。
type TradingCalendar struct {
	Days      map[string]bool `json:"days"`       // "2006-01-02" → 是否为交易日
	StartDate string          `json:"start_date"` // 覆盖起始日期
	EndDate   string          `json:"end_date"`   // 覆盖截止日期
	UpdatedAt time.Time       `json:"updated_at"` // 上次更新时间

	cacheFile          string // 缓存文件路径
	updateIntervalDays int    // 缓存有效期（天）
	lookbackMonths     int    // 向前覆盖月数
	lookaheadMonths    int    // 向后覆盖月数
}

// NewTradingCalendar 从配置创建交易日历实例。
// 优先加载本地缓存文件；缓存不存在或过期时从 Tushare API 拉取。
func NewTradingCalendar(cacheFile string, updateIntervalDays, lookbackMonths, lookaheadMonths int, client *tushare.Client) (*TradingCalendar, error) {
	cal := &TradingCalendar{
		Days:               make(map[string]bool),
		cacheFile:          cacheFile,
		updateIntervalDays: updateIntervalDays,
		lookbackMonths:     lookbackMonths,
		lookaheadMonths:    lookaheadMonths,
	}

	// 优先加载本地缓存
	if err := cal.LoadFromFile(); err == nil && !cal.NeedsRefresh() {
		slog.Info("交易日历从缓存加载", "start", cal.StartDate, "end", cal.EndDate, "days", len(cal.Days))
		return cal, nil
	}

	// 拉取最新数据
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := cal.FetchFromTushare(ctx, client); err != nil {
		if len(cal.Days) > 0 {
			slog.Warn("交易日历刷新失败，使用过期缓存", "error", err)
			return cal, nil
		}
		return nil, fmt.Errorf("calendar: 首次拉取失败（无缓存可用）: %w", err)
	}

	return cal, nil
}

// LoadFromFile 从 JSON 文件加载缓存。
func (c *TradingCalendar) LoadFromFile() error {
	dir := filepath.Dir(c.cacheFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}

	data, err := os.ReadFile(c.cacheFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, c)
}

// SaveToFile 将当前日历数据持久化到 JSON 文件。
func (c *TradingCalendar) SaveToFile() error {
	dir := filepath.Dir(c.cacheFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}

	c.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化日历失败: %w", err)
	}
	return os.WriteFile(c.cacheFile, data, 0644)
}

// IsTradingDay 判断给定日期是否为 A 股交易日。
// 日期超出缓存范围时回退为周末判断。
func (c *TradingCalendar) IsTradingDay(date time.Time) bool {
	key := date.Format("2006-01-02")
	isTrading, ok := c.Days[key]
	if !ok {
		wd := date.Weekday()
		return wd != time.Saturday && wd != time.Sunday
	}
	return isTrading
}

// LastTradeDate 返回给定日期当日或之前的最近交易日。
// 向前最多查找 14 天，找不到返回日期为当天。
func (c *TradingCalendar) LastTradeDate(date time.Time) (string, bool) {
	for i := 0; i < 14; i++ {
		d := date.AddDate(0, 0, -i)
		key := d.Format("2006-01-02")
		if isTrading, ok := c.Days[key]; ok && isTrading {
			return key, true
		}
		if _, ok := c.Days[key]; !ok {
			wd := d.Weekday()
			if wd != time.Saturday && wd != time.Sunday {
				return key, true
			}
		}
	}
	return date.Format("2006-01-02"), false
}

// NeedsRefresh 判断缓存是否需要刷新。
// 条件：缓存为空、超过更新间隔、或当前日期超出覆盖范围。
func (c *TradingCalendar) NeedsRefresh() bool {
	if len(c.Days) == 0 {
		return true
	}
	if time.Since(c.UpdatedAt) > time.Duration(c.updateIntervalDays)*24*time.Hour {
		return true
	}
	endDate, err := time.Parse("2006-01-02", c.EndDate)
	if err != nil {
		return true
	}
	return time.Now().After(endDate)
}

// FetchFromTushare 从 Tushare trade_cal API 拉取交易日历并保存缓存。
func (c *TradingCalendar) FetchFromTushare(ctx context.Context, client *tushare.Client) error {
	now := time.Now()
	startDate := now.AddDate(0, -c.lookbackMonths, 0).Format("20060102")
	endDate := now.AddDate(0, c.lookaheadMonths, 0).Format("20060102")

	slog.Info("从 Tushare 拉取交易日历", "start", startDate, "end", endDate)
	items, err := client.TradeCal(ctx, "SSE", startDate, endDate)
	if err != nil {
		return fmt.Errorf("拉取 trade_cal 失败: %w", err)
	}
	if len(items) == 0 {
		return fmt.Errorf("trade_cal 返回空数据")
	}

	// 构建日期映射（YYYYMMDD → YYYY-MM-DD）
	days := make(map[string]bool, len(items))
	var minDate, maxDate string

	for _, item := range items {
		dateStr := item.CalDate[0:4] + "-" + item.CalDate[4:6] + "-" + item.CalDate[6:8]
		days[dateStr] = item.IsOpen == 1

		if minDate == "" || item.CalDate < minDate {
			minDate = item.CalDate
		}
		if maxDate == "" || item.CalDate > maxDate {
			maxDate = item.CalDate
		}
	}

	if len(minDate) == 8 {
		minDate = minDate[0:4] + "-" + minDate[4:6] + "-" + minDate[6:8]
	}
	if len(maxDate) == 8 {
		maxDate = maxDate[0:4] + "-" + maxDate[4:6] + "-" + maxDate[6:8]
	}

	c.Days = days
	c.StartDate = minDate
	c.EndDate = maxDate

	if err := c.SaveToFile(); err != nil {
		slog.Warn("交易日历缓存保存失败", "error", err)
	}

	slog.Info("交易日历刷新完成", "start", c.StartDate, "end", c.EndDate, "days", len(c.Days))
	return nil
}
