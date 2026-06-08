package market

import "context"

// Service 是市场行情的业务服务层。
// 委托 Provider 接口执行实际查询，可在未来添加缓存、聚合等业务逻辑。
type Service struct {
	provider Provider
}

// NewService 使用给定的数据源创建业务服务。
func NewService(provider Provider) *Service {
	return &Service{provider: provider}
}

// GetMarketStatus 返回当前市场交易状态。
func (s *Service) GetMarketStatus(ctx context.Context) (*MarketSession, error) {
	return s.provider.GetMarketStatus(ctx)
}

// GetIndexQuotes 返回所有追踪指数的行情数据。
func (s *Service) GetIndexQuotes(ctx context.Context) ([]IndexQuote, error) {
	return s.provider.GetIndexQuotes(ctx)
}

// GetMarketOverview 返回市场大盘总览。
func (s *Service) GetMarketOverview(ctx context.Context) (*MarketOverview, error) {
	return s.provider.GetMarketOverview(ctx)
}
