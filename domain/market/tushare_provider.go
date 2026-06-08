package market

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/disturb-yy/stock-monitor/pkg/tushare"
)

// indexNames maps Tushare index symbols to human-readable names.
var indexNames = map[string]string{
	"000001.SH": "上证指数",
	"399001.SZ": "深证成指",
	"399006.SZ": "创业板指",
	"000688.SH": "科创50",
	"000300.SH": "沪深300",
	"000905.SH": "中证500",
}

type TushareProvider struct {
	client *tushare.Client
}

func NewTushareProvider(client *tushare.Client) *TushareProvider {
	return &TushareProvider{client: client}
}

func (p *TushareProvider) Market() Market {
	return MarketCNA
}

// GetMarketStatus determines the current A-share market session based on
// server local time and the standard trading schedule (Asia/Shanghai).
func (p *TushareProvider) GetMarketStatus(ctx context.Context) (*MarketSession, error) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	now := time.Now().In(loc)

	status, isTrading := determineMarketSession(now)
	tradeDate := now.Format("2006-01-02")

	// On weekends or after close, try to get the latest trade date from API
	if status == MarketStatusClosed {
		if latestDate := p.latestTradeDate(ctx); latestDate != "" {
			tradeDate = latestDate
		}
	}

	return &MarketSession{
		Market:    MarketCNA,
		Status:    status,
		IsTrading: isTrading,
		TradeDate: tradeDate,
		Timezone:  "Asia/Shanghai",
		UpdatedAt: time.Now(),
	}, nil
}

// determineMarketSession returns the market status and whether trading is active
// based on the given time in Asia/Shanghai timezone.
func determineMarketSession(t time.Time) (MarketStatus, bool) {
	weekday := t.Weekday()

	// Weekend: always closed
	if weekday == time.Saturday || weekday == time.Sunday {
		return MarketStatusClosed, false
	}

	hour := t.Hour()
	minute := t.Minute()
	now := hour*60 + minute

	switch {
	case now < 9*60+15:
		return MarketStatusPreOpen, false
	case now < 9*60+30:
		return MarketStatusPreOpen, false
	case now < 11*60+30:
		return MarketStatusTrading, true
	case now < 13*60:
		return MarketStatusLunchBreak, false
	case now < 15*60:
		return MarketStatusTrading, true
	default:
		return MarketStatusClosed, false
	}
}

// latestTradeDate fetches the most recent trade date from the index_daily API.
func (p *TushareProvider) latestTradeDate(ctx context.Context) string {
	items, err := p.client.IndexDaily(ctx, "000001.SH", "", "", "trade_date")
	if err != nil || len(items) == 0 {
		return ""
	}
	return items[0].TradeDate
}

// GetIndexQuotes fetches real index quotes concurrently for all configured
// indices. Partial failures are tolerated: successful results are returned
// alongside aggregated error information.
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

	for i, idx := range indices {
		wg.Add(1)
		go func(i int, symbol, name string) {
			defer wg.Done()
			q, err := p.fetchIndexQuote(ctx, symbol, name)
			results[i] = result{idx: i, quote: q, err: err}
		}(i, idx.Symbol, idx.Name)
	}
	wg.Wait()

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

// fetchIndexQuote fetches a single index quote by querying the latest daily
// K-line and previous day close.
func (p *TushareProvider) fetchIndexQuote(ctx context.Context, symbol, name string) (IndexQuote, error) {
	// Fetch latest trading day data
	items, err := p.client.IndexDaily(ctx, symbol, "", "")
	if err != nil {
		return IndexQuote{}, fmt.Errorf("query %s: %w", symbol, err)
	}
	if len(items) == 0 {
		return IndexQuote{}, fmt.Errorf("query %s: no data returned", symbol)
	}

	latest := items[0]

	// Fetch previous close
	preClose, err := p.fetchPreClose(ctx, symbol, latest.TradeDate)
	if err != nil {
		preClose = latest.Close // fallback: use close as preClose
	}

	return mapToIndexQuote(symbol, name, latest, preClose), nil
}

// fetchPreClose retrieves the closing price of the previous trading day.
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

// mapToIndexQuote converts a Tushare IndexDailyItem into a domain IndexQuote.
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

// GetMarketOverview aggregates market status and all index quotes into a
// comprehensive market overview with summary statistics.
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

// calculateMarketSummary computes rising/falling/flat counts and total amount.
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
