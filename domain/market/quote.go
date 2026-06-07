package market

import "time"

type Market string

const (
	MarketCNA Market = "CN_A"
)

type MarketStatus string

const (
	MarketStatusUnknown    MarketStatus = "unknown"
	MarketStatusPreOpen    MarketStatus = "pre_open"
	MarketStatusTrading    MarketStatus = "trading"
	MarketStatusLunchBreak MarketStatus = "lunch_break"
	MarketStatusClosed     MarketStatus = "closed"
)

type MarketSession struct {
	Market    Market       `json:"market"`
	Status    MarketStatus `json:"status"`
	IsTrading bool         `json:"isTrading"`
	TradeDate string       `json:"tradeDate"`
	Timezone  string       `json:"timezone"`
	UpdatedAt time.Time    `json:"updatedAt"`
}

type IndexQuote struct {
	Market        Market    `json:"market"`
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name"`
	Price         float64   `json:"price"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"changePercent"`
	Open          float64   `json:"open"`
	High          float64   `json:"high"`
	Low           float64   `json:"low"`
	PreClose      float64   `json:"preClose"`
	Volume        int64     `json:"volume"`
	Amount        float64   `json:"amount"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type MarketOverview struct {
	Market    Market        `json:"market"`
	Status    MarketStatus  `json:"status"`
	TradeDate string        `json:"tradeDate"`
	UpdatedAt time.Time     `json:"updatedAt"`
	Summary   MarketSummary `json:"summary"`
	Indices   []IndexQuote  `json:"indices"`
}

type MarketSummary struct {
	RisingIndexCount  int     `json:"risingIndexCount"`
	FallingIndexCount int     `json:"fallingIndexCount"`
	FlatIndexCount    int     `json:"flatIndexCount"`
	TotalAmount       float64 `json:"totalAmount"`
}
