package tushare

// package tushare 封装 Tushare Pro HTTP API 的客户端。
// 提供通用的请求/响应类型和具体 API（如指数日线、交易日历）的数据结构。

import (
	"encoding/json"
)

// Request 为发送到 Tushare Pro API 的 JSON 请求体。
type Request struct {
	APIName string         `json:"api_name"` // API 名称
	Token   string         `json:"token"`    // 用户 Token
	Params  map[string]any `json:"params"`   // 查询参数
	Fields  string         `json:"fields"`   // 返回字段，逗号分隔
}

// Response 为 Tushare Pro API 的顶层响应信封。
type Response struct {
	Code int    `json:"code"`           // 状态码，0 表示成功
	Msg  string `json:"msg"`            // 返回消息
	Data *Data  `json:"data,omitempty"` // 表格数据
}

// Data 封装 Tushare 返回的表格形式数据。
type Data struct {
	Fields []string `json:"fields"` // 列名
	Items  [][]any  `json:"items"`  // 行数据
}

// IndexDailyItem 表示指数日线 K 线的一行数据。
type IndexDailyItem struct {
	TsCode    string  // 指数代码
	TradeDate string  // 交易日 YYYYMMDD
	Open      float64 // 开盘价
	High      float64 // 最高价
	Low       float64 // 最低价
	Close     float64 // 收盘价
	Volume    float64 // 成交量（手）
	Amount    float64 // 成交额（千元）
}

// TradeCalItem 表示交易日历的一行数据。
type TradeCalItem struct {
	CalDate      string // 日期 YYYYMMDD
	IsOpen       int    // 是否开市，1=是 0=否
	PretradeDate string // 前一交易日 YYYYMMDD
}

// indexDailyFields 为 IndexDaily API 默认返回的字段列表。
var indexDailyFields = []string{
	"ts_code", "trade_date", "open", "high",
	"low", "close", "vol", "amount",
}

// tradeCalFields 为 TradeCal API 默认返回的字段列表。
var tradeCalFields = []string{"cal_date", "is_open", "pretrade_date"}

// parseIndexDailyItem 根据动态字段列表将行数据解析为 IndexDailyItem。
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

// parseTradeCalItem 根据动态字段列表将行数据解析为 TradeCalItem。
func parseTradeCalItem(fields []string, row []any) TradeCalItem {
	var item TradeCalItem
	for i, f := range fields {
		if i >= len(row) {
			break
		}
		val := row[i]
		switch f {
		case "cal_date":
			item.CalDate, _ = val.(string)
		case "is_open":
			if v, ok := toFloat64(val); ok {
				item.IsOpen = int(v)
			}
		case "pretrade_date":
			item.PretradeDate, _ = val.(string)
		}
	}
	return item
}

// toFloat64 将 json.Number / float64 / int 统一转为 float64。
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}
