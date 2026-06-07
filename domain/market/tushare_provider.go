package market

import (
	"context"

	"github.com/disturb-yy/stock-monitor/pkg/tushare"
)

type TushareProvider struct {
	client *tushare.Client
}

func NewTushareProvider(client *tushare.Client) *TushareProvider {
	return &TushareProvider{client: client}
}

func (p *TushareProvider) Market() Market {
	return MarketCNA
}

func (p *TushareProvider) GetMarketStatus(ctx context.Context) (*MarketSession, error) {
	// TODO: implement real market status via Tushare API
	return nil, nil
}

func (p *TushareProvider) GetIndexQuotes(ctx context.Context) ([]IndexQuote, error) {
	// TODO: implement real index quotes via Tushare API
	return nil, nil
}

func (p *TushareProvider) GetMarketOverview(ctx context.Context) (*MarketOverview, error) {
	// TODO: implement real market overview via Tushare API
	return nil, nil
}
