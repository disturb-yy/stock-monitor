package market

import (
	"net/http"

	"github.com/disturb-yy/stock-monitor/pkg/httputil"
	"github.com/disturb-yy/stock-monitor/pkg/logger"
	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) GetMarketStatus(c *gin.Context) {
	logger.Info(c.Request.Context(), "market status requested")

	session, err := h.service.GetMarketStatus(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "get market status failed", "error", err)
		httputil.Response(c, http.StatusInternalServerError, httputil.Resp{
			Code: httputil.InternalError,
			Msg:  "internal server error",
		})
		return
	}

	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "success",
		Data: session,
	})
}

func (h *HTTPHandler) GetMarketIndices(c *gin.Context) {
	logger.Info(c.Request.Context(), "market indices requested")

	quotes, err := h.service.GetIndexQuotes(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "get market indices failed", "error", err)
		httputil.Response(c, http.StatusInternalServerError, httputil.Resp{
			Code: httputil.InternalError,
			Msg:  "internal server error",
		})
		return
	}

	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "success",
		Data: quotes,
	})
}

func (h *HTTPHandler) GetMarketOverview(c *gin.Context) {
	logger.Info(c.Request.Context(), "market overview requested")

	overview, err := h.service.GetMarketOverview(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "get market overview failed", "error", err)
		httputil.Response(c, http.StatusInternalServerError, httputil.Resp{
			Code: httputil.InternalError,
			Msg:  "internal server error",
		})
		return
	}

	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "success",
		Data: overview,
	})
}
