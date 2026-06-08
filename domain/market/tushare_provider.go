package market

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/disturb-yy/stock-monitor/pkg/tushare"
)

// indexNames 为 Tushare 指数代码到中文名称的映射表。
var indexNames = map[string]string{
	"000001.SH": "上证指数",
	"399001.SZ": "深证成指",
	"399006.SZ": "创业板指",
	"000688.SH": "科创50",
	"000300.SH": "沪深300",
	"000905.SH": "中证500",
}

// TushareProvider 使用 Tushare Pro API 提供真实 A 股行情数据。
// 实现 market.Provider 接口。
type TushareProvider struct {
	client   *tushare.Client   // Tushare HTTP 客户端
	calendar *TradingCalendar  // 交易日历（用于节假日/调休判断）
}

// NewTushareProvider 创建 Tushare 行情数据源。
// calendar 可为 nil，此时回退为简单周末判断。
func NewTushareProvider(client *tushare.Client, calendar *TradingCalendar) *TushareProvider {
	return &TushareProvider{
		client:   client,
		calendar: calendar,
	}
}

// Market 返回当前服务的市场标识。
func (p *TushareProvider) Market() Market {
	return MarketCNA
}

// GetMarketStatus 获取当前 A 股市场状态。
// 判断逻辑：先通过交易日历判断是否为交易日，
// 再通过服务器本地时间判断当前交易时段。
func (p *TushareProvider) GetMarketStatus(ctx context.Context) (*MarketSession, error) {
	// 加载上海时区
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	now := time.Now().In(loc)

	// 第一步：交易日历判断
	if p.calendar != nil && !p.calendar.IsTradingDay(now) {
		tradeDate := now.Format("2006-01-02")
		if td, ok := p.calendar.LastTradeDate(now); ok {
			tradeDate = td
		}
		return &MarketSession{
			Market:    MarketCNA,
			Status:    MarketStatusClosed,
			IsTrading: false,
			TradeDate: tradeDate,
			Timezone:  "Asia/Shanghai",
			UpdatedAt: time.Now(),
		}, nil
	}

	// 第二步：时段判断
	status, isTrading := determineMarketSession(now)

	return &MarketSession{
		Market:    MarketCNA,
		Status:    status,
		IsTrading: isTrading,
		TradeDate: now.Format("2006-01-02"),
		Timezone:  "Asia/Shanghai",
		UpdatedAt: time.Now(),
	}, nil
}

// determineMarketSession 根据上海时间判断当前所处的交易时段。
// 交易日判断由调用方完成（日历或周末判断）。
func determineMarketSession(t time.Time) (MarketStatus, bool) {
	hour := t.Hour()
	minute := t.Minute()
	now := hour*60 + minute

	switch {
	case now < 9*60+15:
		return MarketStatusPreOpen, false // 凌晨至开盘前
	case now < 9*60+30:
		return MarketStatusPreOpen, false // 集合竞价时段
	case now < 11*60+30:
		return MarketStatusTrading, true  // 上午交易时段
	case now < 13*60:
		return MarketStatusLunchBreak, false // 午间休市
	case now < 15*60:
		return MarketStatusTrading, true  // 下午交易时段
	default:
		return MarketStatusClosed, false  // 收盘后
	}
}

// GetIndexQuotes 并发获取所有追踪指数的实时行情。
// 单个指数查询失败不影响其余指数，返回成功结果 + 汇总错误信息。
func (p *TushareProvider) GetIndexQuotes(ctx context.Context) ([]IndexQuote, error) {
	type indexEntry struct{ Symbol, Name string }
	var indices []indexEntry
	for sym, name := range indexNames {
		indices = append(indices, indexEntry{Symbol: sym, Name: name})
	}

	type result struct {
		idx   int
		quote IndexQuote
		err   error
	}

	results := make([]result, len(indices))
	var wg sync.WaitGroup

	// 并发查询各指数
	for i, idx := range indices {
		wg.Add(1)
		go func(i int, symbol, name string) {
			defer wg.Done()
			q, err := p.fetchIndexQuote(ctx, symbol, name)
			results[i] = result{idx: i, quote: q, err: err}
		}(i, idx.Symbol, idx.Name)
	}
	wg.Wait()

	// 收集成功与失败结果
	var quotes []IndexQuote
	var errs []string
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, r.err.Error())
		} else {
			quotes = append(quotes, r.quote)
		}
	}

	if len(quotes) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("tushare: all index queries failed: %s", strings.Join(errs, "; "))
	}

	var retErr error
	if len(errs) > 0 {
		retErr = fmt.Errorf("tushare: partial failure (%d/%d indices): %s",
			len(errs), len(indices), strings.Join(errs, "; "))
	}

	return quotes, retErr
}

// fetchIndexQuote 获取单个指数的当日行情（含昨收价）。
func (p *TushareProvider) fetchIndexQuote(ctx context.Context, symbol, name string) (IndexQuote, error) {
	// 获取最新交易日数据
	items, err := p.client.IndexDaily(ctx, symbol, "", "")
	if err != nil {
		return IndexQuote{}, fmt.Errorf("query %s: %w", symbol, err)
	}
	if len(items) == 0 {
		return IndexQuote{}, fmt.Errorf("query %s: no data returned", symbol)
	}

	latest := items[0]

	// 获取昨收价
	preClose, err := p.fetchPreClose(ctx, symbol, latest.TradeDate)
	if err != nil {
		preClose = latest.Close // 降级：用当日收盘价
	}

	return mapToIndexQuote(symbol, name, latest, preClose), nil
}

// fetchPreClose 获取指定交易日前一个交易日的收盘价。
func (p *TushareProvider) fetchPreClose(ctx context.Context, symbol, currentTradeDate string) (float64, error) {
	items, err := p.client.IndexDaily(ctx, symbol, "", currentTradeDate, "close")
	if err != nil {
		return 0, fmt.Errorf("fetch preclose %s: %w", symbol, err)
	}
	if len(items) < 2 {
		return 0, fmt.Errorf("fetch preclose %s: not enough data (need 2, got %d)", symbol, len(items))
	}
	return items[1].Close, nil
}

// mapToIndexQuote 将 Tushare 日线数据映射为领域 IndexQuote 模型。
func mapToIndexQuote(symbol, name string, item tushare.IndexDailyItem, preClose float64) IndexQuote {
	change := item.Close - preClose
	changePercent := 0.0
	if preClose != 0 {
		changePercent = change / preClose * 100
	}

	return IndexQuote{
		Market:        MarketCNA,
		Symbol:        symbol,
		Name:          name,
		Price:         item.Close,
		Change:        change,
		ChangePercent: changePercent,
		Open:          item.Open,
		High:          item.High,
		Low:           item.Low,
		PreClose:      preClose,
		Volume:        int64(item.Volume),
		Amount:        item.Amount,
		UpdatedAt:     time.Now(),
	}
}

// GetMarketOverview 聚合市场状态和指数行情，生成大盘总览。
func (p *TushareProvider) GetMarketOverview(ctx context.Context) (*MarketOverview, error) {
	session, err := p.GetMarketStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("get market status: %w", err)
	}

	quotes, err := p.GetIndexQuotes(ctx)
	if err != nil && len(quotes) == 0 {
		return nil, fmt.Errorf("get index quotes: %w", err)
	}

	summary := calculateMarketSummary(quotes)

	return &MarketOverview{
		Market:    MarketCNA,
		Status:    session.Status,
		TradeDate: session.TradeDate,
		UpdatedAt: time.Now(),
		Summary:   summary,
		Indices:   quotes,
	}, nil
}

// calculateMarketSummary 计算涨/跌/平盘指数数量和总成交额。
func calculateMarketSummary(quotes []IndexQuote) MarketSummary {
	var s MarketSummary
	var totalAmount float64
	for _, q := range quotes {
		totalAmount += q.Amount
		switch {
		case q.ChangePercent > 0:
			s.RisingIndexCount++
		case q.ChangePercent < 0:
			s.FallingIndexCount++
		default:
			s.FlatIndexCount++
		}
	}
	s.TotalAmount = totalAmount
	return s
}
