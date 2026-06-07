package market

import "context"

type Provider interface {
	Market() Market
	GetMarketStatus(ctx context.Context) (*MarketSession, error)
	GetIndexQuotes(ctx context.Context) ([]IndexQuote, error)
	GetMarketOverview(ctx context.Context) (*MarketOverview, error)
}
