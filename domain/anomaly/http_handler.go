package anomaly

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/disturb-yy/stock-monitor/pkg/httputil"
	"github.com/disturb-yy/stock-monitor/pkg/tushare"
)

// indexNames 为 Tushare 指数代码到中文名称的映射。
var indexNames = map[string]string{
	"000001.SH": "上证指数",
	"399001.SZ": "深证成指",
	"399006.SZ": "创业板指",
	"000688.SH": "科创50",
	"000300.SH": "沪深300",
	"000905.SH": "中证500",
}

// DefaultMaxHistoryDays 是历史数据查询的最大回溯天数。
const DefaultMaxHistoryDays = 10

// HTTPHandler 处理异动检测相关的 HTTP 请求。
type HTTPHandler struct {
	engine      *Engine         // 检测引擎
	client      *tushare.Client // Tushare API 客户端
	maxLookback int             // 历史数据最大回溯天数
}

// NewHTTPHandler 创建异动检测 HTTP 处理器。
func NewHTTPHandler(engine *Engine, client *tushare.Client) *HTTPHandler {
	return &HTTPHandler{
		engine:      engine,
		client:      client,
		maxLookback: DefaultMaxHistoryDays,
	}
}

// GetAnomaliesResponse 是异动事件查询接口的响应体。
type GetAnomaliesResponse struct {
	Events []AnomalyEvent `json:"events"` // 事件列表
	Count  int            `json:"count"`  // 事件总数
}

// GetAnomalies 处理 GET /api/market/anomalies 请求。
// 获取实时行情数据，运行检测引擎，返回异动事件列表。
func (h *HTTPHandler) GetAnomalies(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// 1. 获取所有指数的当前行情
	current, err := h.fetchCurrentQuotes(ctx)
	if err != nil {
		slog.Error("anomaly: 获取当前行情失败", "error", err)
		httputil.Response(c, http.StatusInternalServerError, httputil.Resp{
			Code: httputil.InternalError,
			Msg:  "failed to fetch market data",
		})
		return
	}

	// 2. 获取历史数据（用于成交量和连续涨跌检测）
	historical := h.fetchHistorical(ctx, current)

	// 3. 运行检测引擎
	events := h.engine.Detect(ctx, current, historical)

	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "success",
		Data: GetAnomaliesResponse{Events: events, Count: len(events)},
	})
}

// fetchCurrentQuotes 并发获取所有追踪指数的当日行情数据。
func (h *HTTPHandler) fetchCurrentQuotes(ctx context.Context) (map[string]IndexData, error) {
	type entry struct{ Symbol, Name string }
	var indices []entry
	for sym, name := range indexNames {
		indices = append(indices, entry{Symbol: sym, Name: name})
	}

	type result struct {
		symbol string
		name   string
		data   IndexData
		err    error
	}

	results := make([]result, len(indices))
	var wg sync.WaitGroup

	// 并发查询每个指数
	for i, idx := range indices {
		wg.Add(1)
		go func(i int, symbol, name string) {
			defer wg.Done()
			items, err := h.client.IndexDaily(ctx, symbol, "", "")
			if err != nil {
				results[i] = result{symbol: symbol, name: name, err: err}
				return
			}
			if len(items) == 0 {
				results[i] = result{symbol: symbol, name: name, err: nil}
				return
			}
			latest := items[0]
			results[i] = result{
				symbol: symbol,
				name:   name,
				data: IndexData{
					Symbol:    symbol,
					Name:      name,
					Price:     latest.Close,
					ChangePct: 0, // 稍后通过 PreClose 计算
					Open:      latest.Open,
					High:      latest.High,
					Low:       latest.Low,
					Volume:    latest.Volume,
					Close:     latest.Close,
					TradeDate: latest.TradeDate,
				},
			}
		}(i, idx.Symbol, idx.Name)
	}
	wg.Wait()

	// 收集结果并计算涨跌幅
	current := make(map[string]IndexData)
	for _, r := range results {
		if r.err != nil {
			slog.Warn("anomaly: 查询指数失败", "symbol", r.symbol, "error", r.err)
			continue
		}
		d := r.data

		// 获取昨收价以计算涨跌幅
		preItems, err := h.client.IndexDaily(ctx, r.symbol, "", r.data.TradeDate, "close")
		if err == nil && len(preItems) >= 2 {
			d.PreClose = preItems[1].Close
			if d.PreClose != 0 {
				d.ChangePct = (d.Close - d.PreClose) / d.PreClose * 100
			}
		}

		current[r.symbol] = d
	}
	return current, nil
}

// fetchHistorical 获取所有指数近 N 日的历史 OHLCV 数据。
// 用于成交量均值计算和连续涨跌判断。
func (h *HTTPHandler) fetchHistorical(ctx context.Context, current map[string]IndexData) map[string][]HistoricalDay {
	historical := make(map[string][]HistoricalDay)

	// 查询截止日期为明天，确保包含当日
	endDate := time.Now().AddDate(0, 0, 1).Format("20060102")

	for symbol := range current {
		items, err := h.client.IndexDaily(ctx, symbol, "", endDate, "close", "vol", "trade_date")
		if err != nil {
			slog.Warn("anomaly: 查询历史数据失败", "symbol", symbol, "error", err)
			continue
		}
		days := make([]HistoricalDay, 0, len(items))
		for _, item := range items {
			days = append(days, HistoricalDay{
				TradeDate: item.TradeDate,
				Close:     item.Close,
				Volume:    item.Volume,
			})
			if len(days) >= h.maxLookback {
				break
			}
		}
		historical[symbol] = days
	}
	return historical
}
