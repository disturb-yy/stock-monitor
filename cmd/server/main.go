package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/disturb-yy/stock-monitor/api"
	"github.com/disturb-yy/stock-monitor/domain/auth"
	"github.com/disturb-yy/stock-monitor/domain/market"
	"github.com/disturb-yy/stock-monitor/pkg/config"
	"github.com/disturb-yy/stock-monitor/pkg/logger"
	"github.com/disturb-yy/stock-monitor/pkg/middleware"
	"github.com/disturb-yy/stock-monitor/pkg/tushare"
	"github.com/gin-gonic/gin"
)

func main() {
	config.Init()

	cfg := config.Get()
	logger.Init(logger.Config{
		Level:      cfg.Log.Level,
		Filename:   cfg.Log.Filename,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
		Console:    cfg.Log.Console,
		Format:     cfg.Log.Format,
	})
	defer logger.Sync()

	// --- Domain: market ---
	var marketProvider market.Provider
	if cfg.Provider.Type == "tushare" {
		tushareClient := tushare.New(cfg.Tushare.Token,
			tushare.WithBaseURL(cfg.Tushare.BaseURL),
			tushare.WithHTTPClient(&http.Client{
				Timeout: time.Duration(cfg.Tushare.Timeout) * time.Second,
			}),
		)
		marketProvider = market.NewTushareProvider(tushareClient)
	} else {
		marketProvider = market.NewMockProvider()
	}
	marketService := market.NewService(marketProvider)
	marketHandler := market.NewHTTPHandler(marketService)

	// --- Domain: auth ---
	authConfig := cfg.Auth.AuthConfig()
	authService := auth.NewService(authConfig)
	authHandler := auth.NewHTTPHandler(authService, cfg.Auth.TokenTTLSeconds)

	// --- HTTP server ---
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.Use(gin.Recovery(), middleware.RequestID(), middleware.AccessLog())
	api.RegisterRoutes(router, marketHandler, authHandler, authService, cfg.Auth)

	server := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
	}

	go func() {
		logger.Info(context.Background(), "server started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("listen failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("server shutdown failed", "error", err)
	}

	logger.Info(context.Background(), "server exited")
}
