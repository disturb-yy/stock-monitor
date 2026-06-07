package tushare

// Request is the JSON payload sent to the Tushare Pro API.
type Request struct {
	APIName string         `json:"api_name"`
	Token   string         `json:"token"`
	Params  map[string]any `json:"params"`
	Fields  string         `json:"fields"`
}

// Response is the top-level envelope returned by the Tushare Pro API.
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data *Data  `json:"data,omitempty"`
}

// Data wraps the tabular result from Tushare.
type Data struct {
	Fields []string `json:"fields"`
	Items  [][]any  `json:"items"`
}

// IndexDailyItem represents a row of daily index K-line data.
type IndexDailyItem struct {
	TsCode    string  // 指数代码
	TradeDate string  // 交易日
	Open      float64 // 开盘价
	High      float64 // 最高价
	Low       float64 // 最低价
	Close     float64 // 收盘价
	Volume    float64 // 成交量（手）
	Amount    float64 // 成交额（千元）
}

// fieldIndices maps IndexDailyItem fields by name for dynamic row parsing.
// The keys are the field names returned by Tushare's index_daily API.
var indexDailyFields = []string{
	"ts_code", "trade_date", "open", "high",
	"low", "close", "vol", "amount",
}

func parseIndexDailyItem(fields []string, row []any) IndexDailyItem {
	var item IndexDailyItem
	for i, f := range fields {
		if i >= len(row) {
			break
		}
		val := row[i]
		switch f {
		case "ts_code":
			item.TsCode, _ = val.(string)
		case "trade_date":
			item.TradeDate, _ = val.(string)
		case "open":
			item.Open, _ = toFloat64(val)
		case "high":
			item.High, _ = toFloat64(val)
		case "low":
			item.Low, _ = toFloat64(val)
		case "close":
			item.Close, _ = toFloat64(val)
		case "vol":
			item.Volume, _ = toFloat64(val)
		case "amount":
			item.Amount, _ = toFloat64(val)
		}
	}
	return item
}
