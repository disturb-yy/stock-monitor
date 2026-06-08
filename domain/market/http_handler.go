package market

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/disturb-yy/stock-monitor/pkg/httputil"
)

// HTTPHandler 是行情领域的 HTTP 适配层。
// 将 Gin 的 HTTP 请求转换为对 Service 的调用。
type HTTPHandler struct {
	service *Service
}

// NewHTTPHandler 创建行情 HTTP 处理器。
func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
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
