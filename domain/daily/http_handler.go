package daily

import (
	"context"
	"net/http"
	"time"

	"github.com/disturb-yy/stock-monitor/pkg/httputil"
	"github.com/gin-gonic/gin"
)

// HTTPHandler 处理每日收盘总结相关的 HTTP 请求。
type HTTPHandler struct {
	builder  *Builder  // 收盘总结构建器
	push     PushFunc  // 推送回调
	location *time.Location // 时区
}

// NewHTTPHandler 创建收盘总结 HTTP 处理器。
func NewHTTPHandler(builder *Builder, push PushFunc, location *time.Location) *HTTPHandler {
	if location == nil {
		location = time.Local
	}
	return &HTTPHandler{builder: builder, push: push, location: location}
}

// GetSummary 处理 GET /api/market/daily-summary 请求。
// 查询当日或指定日期的收盘总结 JSON 数据。
func (h *HTTPHandler) GetSummary(c *gin.Context) {
	dateStr := c.Query("date")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	summary, err := h.builder.Build(ctx)
	if err != nil {
		httputil.Response(c, http.StatusInternalServerError, httputil.Resp{
			Code: httputil.InternalError,
			Msg:  "生成收盘总结失败: " + err.Error(),
		})
		return
	}

	// 如果指定了日期且不同于今日，返回空总结（暂不支持历史日期查询）
	if dateStr != "" && dateStr != summary.TradeDate {
		httputil.Response(c, http.StatusOK, httputil.Resp{
			Code: httputil.Success,
			Msg:  "success",
			Data: gin.H{
				"trade_date":     dateStr,
				"is_trading_day": false,
				"data_status":    "ok",
				"generated_at":   time.Now().In(h.location).Format(time.DateTime),
			},
		})
		return
	}

	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "success",
		Data: summaryToMap(summary),
	})
}

// PostSummary 处理 POST /api/market/daily-summary 请求。
// 异步触发收盘总结的生成与推送，立即返回确认。
func (h *HTTPHandler) PostSummary(c *gin.Context) {
	// 先判断是否为交易日
	now := time.Now().In(h.location)
	if !h.builder.provider.IsTradingDay(now) {
		httputil.Response(c, http.StatusOK, httputil.Resp{
			Code: httputil.Success,
			Msg:  "非交易日，不推送",
			Data: gin.H{"triggered": false},
		})
		return
	}

	if h.push == nil {
		httputil.Response(c, http.StatusServiceUnavailable, httputil.Resp{
			Code: httputil.InternalError,
			Msg:  "Webhook 推送未启用，请先配置 webhook.enabled=true",
			Data: gin.H{"triggered": false},
		})
		return
	}

	// 异步推送
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := h.builder.BuildAndPush(ctx, h.push); err != nil {
			// 错误在 Builder/Markdown/Push 内部已记录，不阻塞
			_ = err
		}
	}()

	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "收盘总结推送已触发",
		Data: gin.H{"triggered": true},
	})
}

// summaryToMap 将 DailySummary 转为 API 响应的 map 结构。
// 避免 JSON 序列化循环引用，按 contracts/daily-api.md 格式输出。
func summaryToMap(s *DailySummary) gin.H {
	m := gin.H{
		"trade_date":    s.TradeDate,
		"is_trading_day": s.IsTradingDay,
		"data_status":   string(s.DataStatus),
		"generated_at":  s.GeneratedAt.Format(time.DateTime),
	}

	if s.IsTradingDay {
		m["up_count"] = s.UpCount
		m["down_count"] = s.DownCount
		m["flat_count"] = s.FlatCount
		m["total_indices"] = s.TotalIndices

		if s.TopGainer != nil {
			m["top_gainer"] = gin.H{
				"symbol":         s.TopGainer.Symbol,
				"name":           s.TopGainer.Name,
				"price":          s.TopGainer.Price,
				"change_percent": s.TopGainer.ChangePercent,
				"change":         s.TopGainer.Change,
			}
		}
		if s.TopLoser != nil {
			m["top_loser"] = gin.H{
				"symbol":         s.TopLoser.Symbol,
				"name":           s.TopLoser.Name,
				"price":          s.TopLoser.Price,
				"change_percent": s.TopLoser.ChangePercent,
				"change":         s.TopLoser.Change,
			}
		}

		indices := make([]gin.H, 0, len(s.Indices))
		for _, idx := range s.Indices {
			indices = append(indices, gin.H{
				"symbol":         idx.Symbol,
				"name":           idx.Name,
				"price":          idx.Price,
				"change_percent": idx.ChangePercent,
				"change":         idx.Change,
			})
		}
		m["indices"] = indices

		items := make([]gin.H, 0, len(s.AnomalySummary.Items))
		for _, item := range s.AnomalySummary.Items {
			items = append(items, gin.H{
				"symbol":        item.Symbol,
				"index_name":    item.IndexName,
				"type":          item.Type,
				"type_name":     item.TypeName,
				"trigger_value": item.TriggerValue,
			})
		}
		m["anomaly_summary"] = gin.H{
			"total_count": s.AnomalySummary.TotalCount,
			"items":       items,
		}
	}

	return m
}
