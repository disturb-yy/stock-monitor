package anomaly

import "context"

// IndexData 包含单个指数在当前时刻的关键行情数据，
// 供检测器进行规则判断。
type IndexData struct {
	Symbol    string  // 指数代码
	Name      string  // 指数名称
	Price     float64 // 最新价
	ChangePct float64 // 涨跌幅百分比
	Open      float64 // 开盘价
	High      float64 // 最高价
	Low       float64 // 最低价
	PreClose  float64 // 昨收价
	Volume    float64 // 当前成交量
	Close     float64 // 当前收盘价
	TradeDate string  // 交易日 YYYYMMDD
}

// HistoricalDay 保存单个历史交易日的关键指标。
type HistoricalDay struct {
	TradeDate string  // 交易日 YYYYMMDD
	Close     float64 // 收盘价
	Volume    float64 // 成交量
}

// Detector 是异动检测规则的统一接口。
// 每个具体规则（涨跌幅、成交量、连续涨跌）均实现此接口。
type Detector interface {
	// Type 返回检测器对应的事件类型。
	Type() AnomalyType

	// Detect 对当前行情和历史数据执行检测，返回触发的事件列表。
	Detect(ctx context.Context, current map[string]IndexData, historical map[string][]HistoricalDay) ([]AnomalyEvent, error)
}
