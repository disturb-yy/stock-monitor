// Package daily 提供每日收盘总结的生成、调度和查询功能。
// 通过 DataProvider 接口获取行情和异动数据，
// 通过 PushFunc 适配器将总结推送到企业微信。
package daily

import (
	"context"
	"time"
)

// IndexSnapshot 是指数在某个时间点的快照数据。
// DataProvider 返回此结构供 Builder 构建收盘总结。
type IndexSnapshot struct {
	Symbol        string  // 指数代码，如 "000001.SH"
	Name          string  // 指数中文名称
	Price         float64 // 当前价格
	ChangePercent float64 // 涨跌幅（%）
	Change        float64 // 涨跌额
}

// AnomalyItem 是异动事件的简要摘要。
type AnomalyItem struct {
	Symbol       string  // 指数代码
	IndexName    string  // 指数名称
	Type         string  // 异动类型标识（price_up / volume_spike 等）
	TypeName     string  // 异动类型中文名
	TriggerValue float64 // 触发值
}

// DataProvider 是数据获取接口。
// 由 Composition Root 注入 market 领域的数据源实现。
type DataProvider interface {
	// GetLatestQuotes 获取当日最新的指数行情快照。
	GetLatestQuotes(ctx context.Context) ([]IndexSnapshot, error)

	// GetAnomalyEvents 获取当日的异动事件摘要。
	GetAnomalyEvents(ctx context.Context) ([]AnomalyItem, error)

	// IsTradingDay 判断指定日期是否为交易日。
	IsTradingDay(date time.Time) bool
}

// MockDataProvider 是 DataProvider 的测试双。
// 返回预设的模拟数据，用于单元测试。
type MockDataProvider struct {
	Quotes      []IndexSnapshot
	Anomalies   []AnomalyItem
	IsTrading   bool
	QuotesErr   error
	AnomalyErr  error
}

// NewMockDataProvider 创建默认的 MockDataProvider（交易日，6 个指数正常数据）。
func NewMockDataProvider() *MockDataProvider {
	return &MockDataProvider{
		IsTrading: true,
		Quotes: []IndexSnapshot{
			{Symbol: "000001.SH", Name: "上证指数", Price: 3120.50, ChangePercent: 0.39, Change: 12.30},
			{Symbol: "399001.SZ", Name: "深证成指", Price: 9820.18, ChangePercent: -0.19, Change: -18.72},
			{Symbol: "399006.SZ", Name: "创业板指", Price: 1935.42, ChangePercent: -0.54, Change: -10.48},
			{Symbol: "000688.SH", Name: "科创50", Price: 812.36, ChangePercent: 0.32, Change: 2.62},
			{Symbol: "000300.SH", Name: "沪深300", Price: 3688.75, ChangePercent: 0.24, Change: 8.95},
			{Symbol: "000905.SH", Name: "中证500", Price: 5520.16, ChangePercent: -0.11, Change: -6.20},
		},
		Anomalies: []AnomalyItem{
			{Symbol: "000001.SH", IndexName: "上证指数", Type: "price_up", TypeName: "上涨异动", TriggerValue: 3.52},
			{Symbol: "399006.SZ", IndexName: "创业板指", Type: "price_down", TypeName: "下跌异动", TriggerValue: -2.54},
		},
	}
}

// GetLatestQuotes 返回预设的指数行情快照。
func (m *MockDataProvider) GetLatestQuotes(_ context.Context) ([]IndexSnapshot, error) {
	if m.QuotesErr != nil {
		return nil, m.QuotesErr
	}
	return m.Quotes, nil
}

// GetAnomalyEvents 返回预设的异动事件摘要。
func (m *MockDataProvider) GetAnomalyEvents(_ context.Context) ([]AnomalyItem, error) {
	if m.AnomalyErr != nil {
		return nil, m.AnomalyErr
	}
	return m.Anomalies, nil
}

// IsTradingDay 返回预设的交易日判断结果。
func (m *MockDataProvider) IsTradingDay(_ time.Time) bool {
	return m.IsTrading
}
