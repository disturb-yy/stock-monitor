// Package market 是市场行情领域，负责 A 股指数行情数据的获取和展示。
// 提供 Provider 接口抽象数据源，支持 Mock（模拟）和 Tushare（真实）两种实现。
package market

import "context"

// Provider 是行情数据源的统一接口。
// 所有数据源实现（MockProvider、TushareProvider）必须满足此契约。
type Provider interface {
	// Market 返回当前数据源服务的市场标识。
	Market() Market
	// GetMarketStatus 返回当前市场交易状态。
	GetMarketStatus(ctx context.Context) (*MarketSession, error)
	// GetIndexQuotes 返回所有追踪指数的实时行情数据。
	GetIndexQuotes(ctx context.Context) ([]IndexQuote, error)
	// GetMarketOverview 返回市场状态和所有指数行情的概览快照。
	GetMarketOverview(ctx context.Context) (*MarketOverview, error)
}
