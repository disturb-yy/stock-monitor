package market

import "context"

type Service struct {
	provider Provider
}

func NewService(provider Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) GetMarketStatus(ctx context.Context) (*MarketSession, error) {
	return s.provider.GetMarketStatus(ctx)
}

func (s *Service) GetIndexQuotes(ctx context.Context) ([]IndexQuote, error) {
	return s.provider.GetIndexQuotes(ctx)
}

func (s *Service) GetMarketOverview(ctx context.Context) (*MarketOverview, error) {
	return s.provider.GetMarketOverview(ctx)
}
