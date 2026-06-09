package market

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/disturb-yy/stock-monitor/pkg/httputil"
)

// HTTPHandler 是行情领域的 HTTP 适配层。
// 将 Gin 的 HTTP 请求转换为对 Service 的调用。
type HTTPHandler struct {
	service      *Service       // 行情业务服务
	historyStore *HistoryStore  // 历史行情存储（采集器填充）
}

// NewHTTPHandler 创建行情 HTTP 处理器。
func NewHTTPHandler(service *Service, historyStore *HistoryStore) *HTTPHandler {
	return &HTTPHandler{service: service, historyStore: historyStore}
}

// GetMarketStatus 处理 GET /api/market/status 请求。
// 返回当前 A 股市场的交易状态。
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
// 返回所有追踪指数的实时行情数据。
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
// 查询指定指数在日期范围内的历史行情数据（采集器自动填充）。
// 参数：start（起始日期，必填）、end（截止日期，可选）、symbol（指数代码，可选）。
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

	result, err := h.historyStore.Query(symbol, start, end)
	if err != nil {
		httputil.Response(c, http.StatusBadRequest, httputil.Resp{
			Code: httputil.ParamError,
			Msg:  err.Error(),
		})
		return
	}

	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "success",
		Data: result,
	})
}

// GetMarketOverview 处理 GET /api/market/overview 请求。
// 返回市场状态 + 所有指数行情 + 涨跌统计的总览快照。
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
