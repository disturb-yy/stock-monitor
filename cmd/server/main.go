// Package main 是 stock-monitor 服务的启动入口。
// 负责加载配置、创建依赖、组装领域模块并启动 HTTP 服务器。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/disturb-yy/stock-monitor/api"
	"github.com/disturb-yy/stock-monitor/domain/anomaly"
	"github.com/disturb-yy/stock-monitor/domain/auth"
	"github.com/disturb-yy/stock-monitor/domain/market"
	"github.com/disturb-yy/stock-monitor/pkg/config"
	"github.com/disturb-yy/stock-monitor/pkg/logger"
	"github.com/disturb-yy/stock-monitor/pkg/middleware"
	"github.com/disturb-yy/stock-monitor/pkg/tushare"
	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	config.Init()
	cfg := config.Get()

	// 初始化日志
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

	// --- 交易日历（本地缓存 + Tushare 远程拉取） ---
	cal, err := market.NewTradingCalendar(
		cfg.Calendar.CacheFile,
		cfg.Calendar.UpdateIntervalDays,
		cfg.Calendar.LookbackMonths,
		cfg.Calendar.LookaheadMonths,
		tushare.New(cfg.Tushare.Token,
			tushare.WithBaseURL(cfg.Tushare.BaseURL),
			tushare.WithHTTPClient(&http.Client{
				Timeout: time.Duration(cfg.Tushare.Timeout) * time.Second,
			}),
		),
	)
	if err != nil {
		slog.Warn("交易日历初始化失败，回退为周末判断", "error", err)
	}

	// --- 异动检测引擎（从配置创建检测器列表） ---
	var anomalyHandler *anomaly.HTTPHandler
	if cfg.Anomaly.Enabled {
		var detectors []anomaly.Detector
		for _, rule := range cfg.Anomaly.Rules {
			if !rule.Enabled {
				continue
			}
			switch rule.Type {
			case "price_change":
				detectors = append(detectors, anomaly.NewPriceChangeDetector(rule.UpperThreshold, rule.LowerThreshold))
			case "volume_spike":
				detectors = append(detectors, anomaly.NewVolumeSpikeDetector(rule.LookbackDays, rule.SpikeMultiplier))
			case "consecutive":
				detectors = append(detectors, anomaly.NewConsecutiveDetector(rule.Days))
			}
		}
		engine := anomaly.NewEngine(detectors...)
		anomalyClient := tushare.New(cfg.Tushare.Token,
			tushare.WithBaseURL(cfg.Tushare.BaseURL),
			tushare.WithHTTPClient(&http.Client{
				Timeout: time.Duration(cfg.Tushare.Timeout) * time.Second,
			}),
		)
		anomalyHandler = anomaly.NewHTTPHandler(engine, anomalyClient)
	}

	// --- 行情领域 ---
	// 根据配置选择 Mock / Tushare 数据源
	var marketProvider market.Provider
	if cfg.Provider.Type == "tushare" {
		tushareClient := tushare.New(cfg.Tushare.Token,
			tushare.WithBaseURL(cfg.Tushare.BaseURL),
			tushare.WithHTTPClient(&http.Client{
				Timeout: time.Duration(cfg.Tushare.Timeout) * time.Second,
			}),
		)
		marketProvider = market.NewTushareProvider(tushareClient, cal)
	} else {
		marketProvider = market.NewMockProvider()
	}
	// 创建后台 context（用于采集器生命周期）
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	// --- 历史行情存储（采集器填充） ---
	var historyStore *market.HistoryStore
	var collector *market.Collector

	// 定时采集器仅在 Tushare 模式且启用时创建
	if cfg.Provider.Type == "tushare" && cfg.Collector.Enabled {
		historyStore = market.NewHistoryStore(cfg.Collector.MaxHistoryDays)

		collector = market.NewCollector(
			marketProvider.(*market.TushareProvider),
			historyStore,
			cal,
			cfg.Collector.IntervalMinutes,
		)
		collector.Start(ctx)
	}

	marketService := market.NewService(marketProvider)
	marketHandler := market.NewHTTPHandler(marketService, historyStore)

	// --- 认证领域 ---
	authConfig := cfg.Auth.AuthConfig()
	authService := auth.NewService(authConfig)
	authHandler := auth.NewHTTPHandler(authService, cfg.Auth.TokenTTLSeconds)

	// --- HTTP 服务器 ---
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.Use(gin.Recovery(), middleware.RequestID(), middleware.AccessLog())
	api.RegisterRoutes(router, marketHandler, authHandler, authService, cfg.Auth, anomalyHandler)

	server := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
	}

	// 启动 HTTP 服务
	go func() {
		logger.Info(ctx, "server started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("listen failed", "error", err)
		}
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 优雅关闭
	// 先停止采集器，再关闭 HTTP 服务
	if collector != nil {
		collector.Stop()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("server shutdown failed", "error", err)
	}

	logger.Info(context.Background(), "server exited")
}
