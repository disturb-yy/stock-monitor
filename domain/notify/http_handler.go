package notify

import (
	"net/http"

	"github.com/disturb-yy/stock-monitor/pkg/httputil"
	"github.com/gin-gonic/gin"
)

// HTTPHandler 处理 Webhook 告警推送相关的 HTTP 请求。
type HTTPHandler struct {
	history *HistoryBuffer
}

// NewHTTPHandler 创建 Webhook 告警推送 HTTP 处理器。
func NewHTTPHandler(history *HistoryBuffer) *HTTPHandler {
	return &HTTPHandler{history: history}
}

// GetHistory 处理 GET /api/market/alerts/history 请求。
// 返回推送历史记录列表。
func (h *HTTPHandler) GetHistory(c *gin.Context) {
	records := h.history.Query()

	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "success",
		Data: gin.H{
			"records": records,
			"total":   len(records),
		},
	})
}
