package market

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/disturb-yy/stock-monitor/pkg/httputil"
)

// HTTPHandler 是行情领域的 HTTP 适配层。
type HTTPHandler struct {
	service      *Service       // 行情业务服务
	historyStore *HistoryStore  // 历史行情存储（采集器填充）
	sqliteStore  *SQLiteStore   // SQLite 持久化存储（可为 nil）
}

// NewHTTPHandler 创建行情 HTTP 处理器。
func NewHTTPHandler(service *Service, historyStore *HistoryStore, sqliteStore *SQLiteStore) *HTTPHandler {
	return &HTTPHandler{service: service, historyStore: historyStore, sqliteStore: sqliteStore}
}

// GetMarketStatus 处理 GET /api/market/status 请求。
func (h *HTTPHandler) GetMarketStatus(c *gin.Context) {
	session, err := h.service.GetMarketStatus(c.Request.Context())
	if err != nil {
		httputil.Response(c, http.StatusInternalServerError, httputil.Resp{
			Code: httputil.InternalError,
			Msg:  err.Error(),
		})
		return
	}
	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "success",
		Data: session,
	})
}

// GetMarketIndices 处理 GET /api/market/indices 请求。
func (h *HTTPHandler) GetMarketIndices(c *gin.Context) {
	quotes, err := h.service.GetIndexQuotes(c.Request.Context())
	if err != nil {
		httputil.Response(c, http.StatusInternalServerError, httputil.Resp{
			Code: httputil.InternalError,
			Msg:  err.Error(),
		})
		return
	}
	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "success",
		Data: quotes,
	})
}


// GetHistory 处理 GET /api/market/history 请求。
// 优先查询内存 HistoryStore；如日期范围超出内存覆盖或结果不完整，
// 回退到 SQLite 查询并合并（内存结果优先）。
func (h *HTTPHandler) GetHistory(c *gin.Context) {
	start := c.Query("start")
	symbol := c.Query("symbol")
	end := c.DefaultQuery("end", "")

	if start == "" {
		httputil.Response(c, http.StatusBadRequest, httputil.Resp{
			Code: httputil.ParamError,
			Msg:  "缺少 start 参数（格式: YYYY-MM-DD）",
		})
		return
	}

	if h.historyStore == nil {
		httputil.Response(c, http.StatusOK, httputil.Resp{
			Code: httputil.Success,
			Msg:  "历史存储未启用（仅 Tushare 模式且启用采集器时可用）",
			Data: map[string][]IndexQuote{},
		})
		return
	}

	// 1. 查询内存
	result, err := h.historyStore.Query(symbol, start, end)
	if err != nil {
		httputil.Response(c, http.StatusBadRequest, httputil.Resp{
			Code: httputil.ParamError,
			Msg:  err.Error(),
		})
		return
	}

	// 2. 如内存结果不完整，回退到 SQLite
	if h.sqliteStore != nil && isIncomplete(result) {
		sqliteQuotes, err := h.sqliteStore.Query(symbol, start, end)
		if err == nil {
			mergeFromSQLite(result, sqliteQuotes)
		}
	}

	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "success",
		Data: result,
	})
}

// GetMarketOverview 处理 GET /api/market/overview 请求。
func (h *HTTPHandler) GetMarketOverview(c *gin.Context) {
	overview, err := h.service.GetMarketOverview(c.Request.Context())
	if err != nil {
		httputil.Response(c, http.StatusInternalServerError, httputil.Resp{
			Code: httputil.InternalError,
			Msg:  err.Error(),
		})
		return
	}
	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "success",
		Data: overview,
	})
}

// isIncomplete 判断内存查询结果是否需要回退到 SQLite。
func isIncomplete(result map[string][]IndexQuote) bool {
	if len(result) == 0 {
		return true
	}
	for _, quotes := range result {
		if len(quotes) == 0 {
			return true
		}
	}
	return false
}

// mergeFromSQLite 将 SQLite 查询结果合并到内存结果中。
// 内存中已有的 (symbol, trade_date) 不会覆盖。
func mergeFromSQLite(memResult map[string][]IndexQuote, sqliteQuotes []IndexQuote) {
	// 构建内存已有记录的键集合
	seen := make(map[string]map[string]bool) // symbol -> trade_date
	for sym, quotes := range memResult {
		dates := make(map[string]bool)
		for _, q := range quotes {
			dates[q.UpdatedAt.Format("2006-01-02")] = true
		}
		seen[sym] = dates
	}

	for _, q := range sqliteQuotes {
		dateKey := q.UpdatedAt.Format("2006-01-02")
		if dates, ok := seen[q.Symbol]; ok && dates[dateKey] {
			continue // 内存已有，跳过
		}
		memResult[q.Symbol] = append(memResult[q.Symbol], q)
	}
}
