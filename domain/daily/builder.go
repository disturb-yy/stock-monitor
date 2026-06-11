package daily

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"
)

// Builder 从 DataProvider 获取数据并构建 DailySummary。
type Builder struct {
	provider DataProvider  // 数据源
	location *time.Location // 时区
}

// NewBuilder 创建收盘总结构建器。
// location 为 A 股时区（Asia/Shanghai）。
func NewBuilder(provider DataProvider, location *time.Location) *Builder {
	if location == nil {
		location = time.Local
	}
	return &Builder{provider: provider, location: location}
}

// Build 构建当日的收盘总结。
// 先尝试获取行情数据，再获取异动事件，最后汇总统计。
func (b *Builder) Build(ctx context.Context) (*DailySummary, error) {
	now := time.Now().In(b.location)
	tradeDate := now.Format(time.DateOnly)

	summary := &DailySummary{
		TradeDate:    tradeDate,
		IsTradingDay: b.provider.IsTradingDay(now),
		GeneratedAt:  now,
		DataStatus:   DataStatusOK,
	}

	// 非交易日直接返回
	if !summary.IsTradingDay {
		return summary, nil
	}

	// 获取行情数据
	quotes, quotesErr := b.provider.GetLatestQuotes(ctx)
	if quotesErr != nil {
		slog.Error("daily: 获取行情数据失败", "error", quotesErr)
		summary.DataStatus = DataStatusFailed
		return summary, nil
	}
	if len(quotes) == 0 {
		summary.DataStatus = DataStatusFailed
		return summary, nil
	}

	// 统计涨跌
	b.calcStats(summary, quotes)

	// 获取异动事件
	anomalies, anomalyErr := b.provider.GetAnomalyEvents(ctx)
	if anomalyErr != nil {
		slog.Warn("daily: 获取异动事件失败", "error", anomalyErr)
		summary.DataStatus = DataStatusPartial
	}
	b.fillAnomalySummary(summary, anomalies)

	return summary, nil
}

// calcStats 计算涨跌统计并识别领涨领跌指数。
func (b *Builder) calcStats(summary *DailySummary, quotes []IndexSnapshot) {
	summary.TotalIndices = len(quotes)
	summary.Indices = quotes

	// 按涨跌幅排序副本，用于找领涨/领跌
	sorted := make([]IndexSnapshot, len(quotes))
	copy(sorted, quotes)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ChangePercent > sorted[j].ChangePercent
	})

	for _, q := range quotes {
		switch {
		case q.ChangePercent > 0:
			summary.UpCount++
		case q.ChangePercent < 0:
			summary.DownCount++
		default:
			summary.FlatCount++
		}
	}

	if len(sorted) > 0 {
		top := sorted[0]
		bottom := sorted[len(sorted)-1]

		if top.ChangePercent > 0 {
			summary.TopGainer = &top
		}
		if bottom.ChangePercent < 0 {
			summary.TopLoser = &bottom
		}
	}
}

// fillAnomalySummary 填充异动事件摘要。
// 最多保留 5 条，超出部分仅记录总数。
func (b *Builder) fillAnomalySummary(summary *DailySummary, items []AnomalyItem) {
	summary.AnomalySummary.TotalCount = len(items)
	if len(items) > 5 {
		summary.AnomalySummary.Items = items[:5]
	} else {
		summary.AnomalySummary.Items = items
	}
}

// ErrNotTradingDay 表示非交易日。
var ErrNotTradingDay = errors.New("非交易日")

// BuildAndPush 构建总结并通过 PushFunc 推送。
// 非交易日返回 ErrNotTradingDay。
func (b *Builder) BuildAndPush(ctx context.Context, push PushFunc) error {
	summary, err := b.Build(ctx)
	if err != nil {
		return fmt.Errorf("构建收盘总结失败: %w", err)
	}

	if !summary.IsTradingDay {
		return ErrNotTradingDay
	}

	if push != nil {
		if err := push(*summary); err != nil {
			return fmt.Errorf("推送收盘总结失败: %w", err)
		}
	}

	return nil
}
