package market

import (
	"context"
	"time"
)

const aStockTimezone = "Asia/Shanghai"

type MockProvider struct {
	location *time.Location
}

func NewMockProvider() *MockProvider {
	location, err := time.LoadLocation(aStockTimezone)
	if err != nil {
		location = time.FixedZone("CST", 8*60*60)
	}

	return &MockProvider{location: location}
}

func (p *MockProvider) Market() Market {
	return MarketCNA
}

func (p *MockProvider) GetMarketStatus(ctx context.Context) (*MarketSession, error) {
	now := time.Now().In(p.location)
	status := p.resolveMarketStatus(now)

	return &MarketSession{
		Market:    p.Market(),
		Status:    status,
		IsTrading: status == MarketStatusTrading,
		TradeDate: now.Format(time.DateOnly),
		Timezone:  aStockTimezone,
		UpdatedAt: now,
	}, nil
}

func (p *MockProvider) GetIndexQuotes(ctx context.Context) ([]IndexQuote, error) {
	now := time.Now().In(p.location)

	return []IndexQuote{
		{
			Market:        p.Market(),
			Symbol:        "000001.SH",
			Name:          "上证指数",
			Price:         3120.50,
			Change:        12.30,
			ChangePercent: 0.39,
			Open:          3108.12,
			High:          3128.66,
			Low:           3098.21,
			PreClose:      3108.20,
			Volume:        331200000,
			Amount:        420000000000,
			UpdatedAt:     now,
		},
		{
			Market:        p.Market(),
			Symbol:        "399001.SZ",
			Name:          "深证成指",
			Price:         9820.18,
			Change:        -18.72,
			ChangePercent: -0.19,
			Open:          9842.61,
			High:          9878.33,
			Low:           9788.45,
			PreClose:      9838.90,
			Volume:        486500000,
			Amount:        558000000000,
			UpdatedAt:     now,
		},
		{
			Market:        p.Market(),
			Symbol:        "399006.SZ",
			Name:          "创业板指",
			Price:         1935.42,
			Change:        -10.48,
			ChangePercent: -0.54,
			Open:          1948.10,
			High:          1952.76,
			Low:           1928.30,
			PreClose:      1945.90,
			Volume:        153400000,
			Amount:        198000000000,
			UpdatedAt:     now,
		},
		{
			Market:        p.Market(),
			Symbol:        "000688.SH",
			Name:          "科创50",
			Price:         812.36,
			Change:        2.62,
			ChangePercent: 0.32,
			Open:          809.30,
			High:          816.18,
			Low:           806.77,
			PreClose:      809.74,
			Volume:        62400000,
			Amount:        83000000000,
			UpdatedAt:     now,
		},
		{
			Market:        p.Market(),
			Symbol:        "000300.SH",
			Name:          "沪深300",
			Price:         3688.75,
			Change:        8.95,
			ChangePercent: 0.24,
			Open:          3679.22,
			High:          3696.80,
			Low:           3668.64,
			PreClose:      3679.80,
			Volume:        212600000,
			Amount:        316000000000,
			UpdatedAt:     now,
		},
		{
			Market:        p.Market(),
			Symbol:        "000905.SH",
			Name:          "中证500",
			Price:         5520.16,
			Change:        -6.20,
			ChangePercent: -0.11,
			Open:          5528.80,
			High:          5550.42,
			Low:           5498.72,
			PreClose:      5526.36,
			Volume:        168900000,
			Amount:        236000000000,
			UpdatedAt:     now,
		},
		{
			Market:        p.Market(),
			Symbol:        "000852.SH",
			Name:          "中证1000",
			Price:         5906.28,
			Change:        0,
			ChangePercent: 0,
			Open:          5902.14,
			High:          5940.10,
			Low:           5870.55,
			PreClose:      5906.28,
			Volume:        129800000,
			Amount:        176000000000,
			UpdatedAt:     now,
		},
	}, nil
}

func (p *MockProvider) GetMarketOverview(ctx context.Context) (*MarketOverview, error) {
	session, err := p.GetMarketStatus(ctx)
	if err != nil {
		return nil, err
	}

	quotes, err := p.GetIndexQuotes(ctx)
	if err != nil {
		return nil, err
	}

	return &MarketOverview{
		Market:    session.Market,
		Status:    session.Status,
		TradeDate: session.TradeDate,
		UpdatedAt: session.UpdatedAt,
		Summary:   summarizeIndexQuotes(quotes),
		Indices:   quotes,
	}, nil
}

func (p *MockProvider) resolveMarketStatus(now time.Time) MarketStatus {
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return MarketStatusClosed
	}

	hour, minute := now.Hour(), now.Minute()
	minutes := hour*60 + minute

	switch {
	case minutes < 9*60+30:
		return MarketStatusPreOpen
	case minutes >= 9*60+30 && minutes <= 11*60+30:
		return MarketStatusTrading
	case minutes > 11*60+30 && minutes < 13*60:
		return MarketStatusLunchBreak
	case minutes >= 13*60 && minutes <= 15*60:
		return MarketStatusTrading
	default:
		return MarketStatusClosed
	}
}

func summarizeIndexQuotes(quotes []IndexQuote) MarketSummary {
	var summary MarketSummary
	for _, quote := range quotes {
		switch {
		case quote.ChangePercent > 0:
			summary.RisingIndexCount++
		case quote.ChangePercent < 0:
			summary.FallingIndexCount++
		default:
			summary.FlatIndexCount++
		}
		summary.TotalAmount += quote.Amount
	}
	return summary
}
