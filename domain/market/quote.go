package market

import "time"

// Market 代表市场标识，使用常量如 MarketCNA。
type Market string

const (
	MarketCNA Market = "CN_A" // 中国 A 股市场
)

// MarketStatus 表示市场当前的状态阶段。
type MarketStatus string

const (
	MarketStatusUnknown    MarketStatus = "unknown"     // 未知状态
	MarketStatusPreOpen    MarketStatus = "pre_open"    // 盘前（开盘前集合竞价）
	MarketStatusTrading    MarketStatus = "trading"     // 交易中
	MarketStatusLunchBreak MarketStatus = "lunch_break" // 午间休市
	MarketStatusClosed     MarketStatus = "closed"      // 已收盘
)

// MarketSession 描述当前市场的会话状态。
type MarketSession struct {
	Market    Market       `json:"market"`    // 市场标识
	Status    MarketStatus `json:"status"`    // 当前状态
	IsTrading bool         `json:"isTrading"` // 是否在交易时段
	TradeDate string       `json:"tradeDate"` // 交易日 YYYY-MM-DD
	Timezone  string       `json:"timezone"`  // 时区
	UpdatedAt time.Time    `json:"updatedAt"` // 数据更新时间
}

// IndexQuote 表示一条指数实时行情数据。
type IndexQuote struct {
	Market        Market    `json:"market"`        // 所属市场
	Symbol        string    `json:"symbol"`        // 指数代码
	Name          string    `json:"name"`          // 指数名称
	Price         float64   `json:"price"`         // 最新价
	Change        float64   `json:"change"`        // 涨跌额
	ChangePercent float64   `json:"changePercent"` // 涨跌幅（百分比）
	Open          float64   `json:"open"`          // 开盘价
	High          float64   `json:"high"`          // 最高价
	Low           float64   `json:"low"`           // 最低价
	PreClose      float64   `json:"preClose"`      // 昨收价
	Volume        int64     `json:"volume"`        // 成交量（手）
	Amount        float64   `json:"amount"`        // 成交额（千元）
	UpdatedAt     time.Time `json:"updatedAt"`     // 数据更新时间
}

// MarketOverview 是市场大盘一览。
type MarketOverview struct {
	Market    Market        `json:"market"`    // 市场标识
	Status    MarketStatus  `json:"status"`    // 市场状态
	TradeDate string        `json:"tradeDate"` // 交易日
	UpdatedAt time.Time     `json:"updatedAt"` // 更新时间
	Summary   MarketSummary `json:"summary"`   // 涨跌汇总
	Indices   []IndexQuote  `json:"indices"`   // 各指数行情
}

// MarketSummary 是市场的涨跌统计汇总。
type MarketSummary struct {
	RisingIndexCount  int     `json:"risingIndexCount"`  // 上涨指数数
	FallingIndexCount int     `json:"fallingIndexCount"` // 下跌指数数
	FlatIndexCount    int     `json:"flatIndexCount"`    // 平盘指数数
	TotalAmount       float64 `json:"totalAmount"`       // 总成交额
}
