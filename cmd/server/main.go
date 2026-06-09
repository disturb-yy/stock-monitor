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
	"github.com/disturb-yy/stock-monitor/domain/notify"
	"github.com/disturb-yy/stock-monitor/pkg/config"
	"github.com/disturb-yy/stock-monitor/pkg/logger"
	"github.com/disturb-yy/stock-monitor/pkg/middleware"
	"github.com/disturb-yy/stock-monitor/pkg/tushare"
	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置和日志
	config.Init()
	cfg := config.Get()
	initLogging(cfg)
	defer logger.Sync()

	// 创建后台 context（采集器和优雅关闭共用）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 组装领域模块
	cal := initTradingCalendar(cfg)
	tc := newTushareClient(cfg)
	anomalyEngine, anomalyHandler := setupAnomaly(cfg, tc)
	notifyDispatcher, notifyHandler := setupWebhook(cfg, anomalyEngine)
	marketHandler, collector := setupMarket(ctx, cfg, cal, tc)
	authService, authHandler := setupAuth(cfg)

	// 启动 HTTP 服务
	server := runServer(cfg, marketHandler, authHandler, authService, anomalyHandler, notifyHandler)

	// 等待退出信号并优雅关闭
	gracefulShutdown(server, collector, notifyDispatcher)
}

// initLogging 初始化日志系统。
func initLogging(cfg *config.Config) {
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
}

// newTushareClient 创建 Tushare API 客户端（供多个领域复用）。
func newTushareClient(cfg *config.Config) *tushare.Client {
	return tushare.New(cfg.Tushare.Token,
		tushare.WithBaseURL(cfg.Tushare.BaseURL),
		tushare.WithHTTPClient(&http.Client{
			Timeout: time.Duration(cfg.Tushare.Timeout) * time.Second,
		}),
	)
}

// initTradingCalendar 初始化交易日历（本地缓存 + Tushare 远程拉取）。
// 失败时仅警告，不影响服务启动。
func initTradingCalendar(cfg *config.Config) *market.TradingCalendar {
	cal, err := market.NewTradingCalendar(
		cfg.Calendar.CacheFile,
		cfg.Calendar.UpdateIntervalDays,
		cfg.Calendar.LookbackMonths,
		cfg.Calendar.LookaheadMonths,
		newTushareClient(cfg),
	)
	if err != nil {
		slog.Warn("交易日历初始化失败，回退为周末判断", "error", err)
	}
	return cal
}

// setupAnomaly 创建异动检测引擎和 HTTP 处理器。
// 返回 nil 表示检测功能未启用。
func setupAnomaly(cfg *config.Config, tc *tushare.Client) (*anomaly.Engine, *anomaly.HTTPHandler) {
	if !cfg.Anomaly.Enabled {
		return nil, nil
	}

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
	handler := anomaly.NewHTTPHandler(engine, tc)
	return engine, handler
}

// toAlertEvents 将 domain/anomaly 的事件模型转换为 domain/notify 的告警事件模型。
// 这是 Composition Root 中唯一的跨领域适配器，遵守 DDF 依赖纪律。
func toAlertEvents(events []anomaly.AnomalyEvent) []notify.AlertEvent {
	result := make([]notify.AlertEvent, len(events))
	for i, e := range events {
		result[i] = notify.AlertEvent{
			ID:           e.ID,
			Type:         notify.AlertType(e.Type),
			Symbol:       e.Symbol,
			IndexName:    e.IndexName,
			TriggerValue: e.TriggerValue,
			Threshold:    e.Threshold,
			TriggeredAt:  e.TriggeredAt,
		}
	}
	return result
}

// setupWebhook 创建 Webhook 告警推送相关组件。
// 返回的 Dispatcher 用于优雅关闭，HTTPHandler 用于路由注册。
func setupWebhook(cfg *config.Config, anomalyEngine *anomaly.Engine) (*notify.Dispatcher, *notify.HTTPHandler) {
	if !cfg.Webhook.Enabled || cfg.Webhook.WebhookURL == "" {
		slog.Info("notify: Webhook 告警推送未启用")
		return nil, nil
	}

	slog.Info("notify: Webhook 告警推送已启用", "url", cfg.Webhook.WebhookURL)

	// 企业微信发送器
	sender := notify.NewWeComSender(cfg.Webhook.WebhookURL)

	// 冷却去重（可选）
	var cooldown *notify.CooldownTracker
	if cfg.Webhook.CooldownMinutes > 0 {
		cooldown = notify.NewCooldownTracker(cfg.Webhook.CooldownMinutes)
		cooldown.StartCleanup(5 * time.Minute)
	}

	// 推送历史缓冲区
	history := notify.NewHistoryBuffer(1000)

	// 推送调度器
	dispatcher := notify.NewDispatcher(
		sender,
		cooldown,
		history,
		cfg.Webhook.RetryCount,
		time.Duration(cfg.Webhook.RetryIntervalSeconds)*time.Second,
	)

	// 通知服务 + 回调注册
	service := notify.NewService(dispatcher, true)
	if anomalyEngine != nil {
		anomalyEngine.SetCallback(func(ctx context.Context, events []anomaly.AnomalyEvent) {
			service.HandleEvents(ctx, toAlertEvents(events))
		})
		slog.Info("notify: 已注册异动事件回调")
	}

	// HTTP 处理器（推送历史查询）
	handler := notify.NewHTTPHandler(history)

	return dispatcher, handler
}

// setupMarket 创建行情领域：数据源、采集器、历史存储、HTTP 处理器。
func setupMarket(ctx context.Context, cfg *config.Config, cal *market.TradingCalendar, tc *tushare.Client) (*market.HTTPHandler, *market.Collector) {
	var provider market.Provider
	if cfg.Provider.Type == "tushare" {
		provider = market.NewTushareProvider(tc, cal)
	} else {
		provider = market.NewMockProvider()
	}

	// 历史存储 + 定时采集器（仅 Tushare 模式且启用时）
	var store *market.HistoryStore
	var collector *market.Collector
	var sqliteStore *market.SQLiteStore
	if cfg.Provider.Type == "tushare" && cfg.Collector.Enabled {
		store = market.NewHistoryStore(cfg.Collector.MaxHistoryDays)

		// SQLite 持久化：启动时加载历史数据
		if cfg.Persistence.Enabled {
			var err error
			sqliteStore, err = market.Open(cfg.Persistence.DBPath)
			if err != nil {
				slog.Warn("SQLite 持久化初始化失败，仅使用内存存储", "error", err)
				sqliteStore = nil
			} else {
				if err := store.LoadFromSQLite(sqliteStore); err != nil {
					slog.Warn("从 SQLite 加载历史数据失败", "error", err)
				}
			}
		}

		collector = market.NewCollector(
			provider.(*market.TushareProvider),
			store,
			cal,
			cfg.Collector.IntervalMinutes,
			sqliteStore,
			cfg.Persistence.MaxDays,
		)
		collector.Start(ctx)
	}

	service := market.NewService(provider)
	handler := market.NewHTTPHandler(service, store, sqliteStore)
	return handler, collector
}

// setupAuth 创建认证领域的服务和 HTTP 处理器。
func setupAuth(cfg *config.Config) (*auth.Service, *auth.HTTPHandler) {
	authCfg := cfg.Auth.AuthConfig()
	service := auth.NewService(authCfg)
	handler := auth.NewHTTPHandler(service, cfg.Auth.TokenTTLSeconds)
	return service, handler
}

// runServer 组装 Gin 路由、启动 HTTP 服务并返回 server 实例（用于优雅关闭）。
func runServer(cfg *config.Config, marketHandler *market.HTTPHandler, authHandler *auth.HTTPHandler, authService *auth.Service, anomalyHandler *anomaly.HTTPHandler, notifyHandler *notify.HTTPHandler) *http.Server {
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.Use(gin.Recovery(), middleware.RequestID(), middleware.AccessLog())
	api.RegisterRoutes(router, marketHandler, authHandler, authService, cfg.Auth, anomalyHandler, notifyHandler)

	server := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
	}

	go func() {
		slog.Info("server started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("listen failed", "error", err)
		}
	}()

	return server
}

// gracefulShutdown 等待退出信号，按顺序关闭：Webhook 推送 → 采集器 → HTTP 服务。
func gracefulShutdown(server *http.Server, collector *market.Collector, dispatcher *notify.Dispatcher) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("notify: 正在关闭，等待推送完成...")

	if dispatcher != nil {
		dispatcher.Wait()
		slog.Info("notify: 所有推送已完成")
	}

	if collector != nil {
		collector.Stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("server shutdown failed", "error", err)
	}

	slog.Info("server exited")
}
