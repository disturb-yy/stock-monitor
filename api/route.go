package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/disturb-yy/stock-monitor/domain/auth"
	"github.com/disturb-yy/stock-monitor/domain/market"
	"github.com/disturb-yy/stock-monitor/pkg/config"
	"github.com/disturb-yy/stock-monitor/pkg/httputil"
	"github.com/disturb-yy/stock-monitor/pkg/logger"
	"github.com/disturb-yy/stock-monitor/pkg/router"
)

func RegisterRoutes(r *gin.Engine, marketHandler *market.HTTPHandler, authHandler *auth.HTTPHandler, authService *auth.Service, authCfg config.AuthConfig) {
	v1 := r.Group("/v1")
	{
		v1.GET("/ready", ready)
		v1.GET("/heart", heart)
		v1.GET("/index", index)
	}

	api := r.Group("/api")
	{
		api.POST("/auth/token", authHandler.GenerateToken)

		router.NewGroup(api.Group("/market")).
			UseIf(authCfg.Enabled, auth.JWTMiddleware(authService)).
			GET("/status", marketHandler.GetMarketStatus).
			GET("/indices", marketHandler.GetMarketIndices).
			GET("/overview", marketHandler.GetMarketOverview)
	}
}

func index(c *gin.Context) {
	logger.Info(c.Request.Context(), "index requested")

	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "stock-monitor started",
		Data: gin.H{"service": "stock-monitor"},
	})
}

func ready(c *gin.Context) {
	logger.Info(c.Request.Context(), "ready check")

	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "service ready",
	})
}

func heart(c *gin.Context) {
	logger.Info(c.Request.Context(), "heartbeat check")

	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "service running",
	})
}
