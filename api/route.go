// Package api 负责 HTTP 路由的注册和分发。
// 将各领域的 HTTPHandler 挂载到 Gin 路由器上，
// 并根据配置决定是否启用认证中间件。
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/disturb-yy/stock-monitor/domain/anomaly"
	"github.com/disturb-yy/stock-monitor/domain/auth"
	"github.com/disturb-yy/stock-monitor/domain/market"
	"github.com/disturb-yy/stock-monitor/pkg/config"
	"github.com/disturb-yy/stock-monitor/pkg/httputil"
	"github.com/disturb-yy/stock-monitor/pkg/logger"
	"github.com/disturb-yy/stock-monitor/pkg/router"
)

// RegisterRoutes 一次性注册所有路由。
// 参数由 Composition Root（cmd/server/main.go）注入。
func RegisterRoutes(r *gin.Engine, marketHandler *market.HTTPHandler, authHandler *auth.HTTPHandler, authService *auth.Service, authCfg config.AuthConfig, anomalyHandler *anomaly.HTTPHandler) {
	// 健康检查路由组
	v1 := r.Group("/v1")
	{
		v1.GET("/ready", ready)
		v1.GET("/heart", heart)
		v1.GET("/index", index)
	}

	// 业务 API 路由组
	api := r.Group("/api")
	{
		// 认证接口
		api.POST("/auth/token", authHandler.GenerateToken)

		// 行情接口（可选 JWT 认证）
		marketGroup := router.NewGroup(api.Group("/market")).
			UseIf(authCfg.Enabled, auth.JWTMiddleware(authService)).
			GET("/status", marketHandler.GetMarketStatus).
			GET("/indices", marketHandler.GetMarketIndices).
			GET("/overview", marketHandler.GetMarketOverview)

		// 异动检测接口（条件注册，仅当 anomalyHandler 不为 nil）
		if anomalyHandler != nil {
			marketGroup.GET("/anomalies", anomalyHandler.GetAnomalies)
		}
	}
}

// index 返回服务基本信息。
func index(c *gin.Context) {
	logger.Info(c.Request.Context(), "index requested")
	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "stock-monitor started",
		Data: gin.H{"service": "stock-monitor"},
	})
}

// ready 就绪检查端点。
func ready(c *gin.Context) {
	logger.Info(c.Request.Context(), "ready check")
	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "service ready",
	})
}

// heart 心跳检查端点。
func heart(c *gin.Context) {
	logger.Info(c.Request.Context(), "heartbeat check")
	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "service running",
	})
}
