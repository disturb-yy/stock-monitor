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
	"github.com/disturb-yy/stock-monitor/domain/daily"
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
	marketHandler, collector, historyStore := setupMarket(ctx, cfg, cal, tc)
	authService, authHandler := setupAuth(cfg)
	dailyHandler, dailyScheduler := setupDaily(ctx, cfg, cal, historyStore, anomalyEngine, notifyDispatcher)

	// 启动 HTTP 服务
	server := runServer(cfg, marketHandler, authHandler, authService, anomalyHandler, notifyHandler, dailyHandler)

	// 等待退出信号并优雅关闭
	gracefulShutdown(server, collector, notifyDispatcher, dailyScheduler)
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
		// 即使推送未启用，也要提供 HTTPHandler 用于查询推送历史（空记录）
		history := notify.NewHistoryBuffer(1000)
		return nil, notify.NewHTTPHandler(history)
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
func setupMarket(ctx context.Context, cfg *config.Config, cal *market.TradingCalendar, tc *tushare.Client) (*market.HTTPHandler, *market.Collector, *market.HistoryStore) {
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
	return handler, collector, store
}

// setupDaily 创建每日收盘总结领域的组件。
// HTTPHandler 始终返回（路由始终注册），Scheduler 仅在启用且有 HistoryStore 时有效。
func setupDaily(ctx context.Context, cfg *config.Config, cal *market.TradingCalendar, historyStore *market.HistoryStore, anomalyEngine *anomaly.Engine, dispatcher *notify.Dispatcher) (*daily.HTTPHandler, *daily.Scheduler) {
	// 时区
	location, _ := time.LoadLocation("Asia/Shanghai")
	if location == nil {
		location = time.FixedZone("CST", 8*60*60)
	}

	// 创建数据提供者（如果 HistoryStore 可用用真实数据，否则用 mock）
	var provider daily.DataProvider
	if historyStore != nil && cfg.DailySummary.Enabled {
		provider = &dailyDataProvider{
			historyStore:  historyStore,
			tradeCalendar: cal,
			anomalyEngine: anomalyEngine,
		}
	} else {
		provider = daily.NewMockDataProvider()
		if !cfg.DailySummary.Enabled {
			slog.Info("daily: 每日收盘总结未启用（使用 mock 数据提供查询）")
		} else {
			slog.Warn("daily: HistoryStore 不可用（仅 Tushare + 采集器模式），使用 mock 数据")
		}
	}

	// 创建 Builder
	builder := daily.NewBuilder(provider, location)

	// 创建 PushFunc 适配器（仅启用时有效推送）
	var push daily.PushFunc
	var scheduler *daily.Scheduler
	if cfg.DailySummary.Enabled && dispatcher != nil && historyStore != nil {
		push = func(s daily.DailySummary) error {
			content := daily.FormatMarkdown(&s)
			dispatcher.PushContent(content)
			return nil
		}
		scheduler = daily.NewScheduler(builder, push, cfg.DailySummary.PushTime, location)
		scheduler.Start()
		slog.Info("daily: 收盘总结调度器已启动", "push_time", cfg.DailySummary.PushTime)
	}

	// 创建 HTTPHandler（始终返回，路由始终注册）
	handler := daily.NewHTTPHandler(builder, push, location)

	return handler, scheduler
}

// dailyDataProvider 实现 daily.DataProvider 接口。
// 将 market 领域的 HistoryStore、TradingCalendar 和 anomaly.Engine
// 适配为 daily 领域的数据模型。
type dailyDataProvider struct {
	historyStore  *market.HistoryStore
	tradeCalendar *market.TradingCalendar
	anomalyEngine *anomaly.Engine
}

// GetLatestQuotes 从 HistoryStore 获取当日最新行情快照。
func (p *dailyDataProvider) GetLatestQuotes(ctx context.Context) ([]daily.IndexSnapshot, error) {
	today := time.Now().Format("2006-01-02")
	data, err := p.historyStore.Query("", today, today)
	if err != nil {
		return nil, err
	}

	var snapshots []daily.IndexSnapshot
	for _, quotes := range data {
		if len(quotes) == 0 {
			continue
		}
		latest := quotes[len(quotes)-1]
		snapshots = append(snapshots, daily.IndexSnapshot{
			Symbol:        latest.Symbol,
			Name:          latest.Name,
			Price:         latest.Price,
			ChangePercent: latest.ChangePercent,
			Change:        latest.Change,
		})
	}

	if len(snapshots) == 0 {
		return nil, errors.New("当日无行情数据")
	}
	return snapshots, nil
}

// GetAnomalyEvents 从异动引擎获取当日异动摘要。
func (p *dailyDataProvider) GetAnomalyEvents(ctx context.Context) ([]daily.AnomalyItem, error) {
	if p.anomalyEngine == nil {
		return nil, nil
	}

	// 通过异常引擎获取事件列表
	events := p.anomalyEngine.GetLastEvents()
	items := make([]daily.AnomalyItem, 0, len(events))
	for _, e := range events {
		typeName := mapAnomalyType(e.Type)
		items = append(items, daily.AnomalyItem{
			Symbol:       e.Symbol,
			IndexName:    e.IndexName,
			Type:         string(e.Type),
			TypeName:     typeName,
			TriggerValue: e.TriggerValue,
		})
	}
	return items, nil
}

// IsTradingDay 通过交易日历判断指定日期是否为交易日。
func (p *dailyDataProvider) IsTradingDay(date time.Time) bool {
	if p.tradeCalendar == nil {
		// 降级：周末判断
		wd := date.Weekday()
		return wd != time.Saturday && wd != time.Sunday
	}
	return p.tradeCalendar.IsTradingDay(date)
}

// mapAnomalyType 将 anomaly 异动类型映射为中文名称。
func mapAnomalyType(t anomaly.AnomalyType) string {
	switch t {
	case anomaly.AnomalyPriceUp:
		return "上涨异动"
	case anomaly.AnomalyPriceDown:
		return "下跌异动"
	case anomaly.AnomalyVolumeSpike:
		return "放量异动"
	case anomaly.AnomalyConsecutiveUp:
		return "连续上涨"
	case anomaly.AnomalyConsecutiveDown:
		return "连续下跌"
	default:
		return string(t)
	}
}

// setupAuth 创建认证领域的服务和 HTTP 处理器。
func setupAuth(cfg *config.Config) (*auth.Service, *auth.HTTPHandler) {
	authCfg := cfg.Auth.AuthConfig()
	service := auth.NewService(authCfg)
	handler := auth.NewHTTPHandler(service, cfg.Auth.TokenTTLSeconds)
	return service, handler
}

// runServer 组装 Gin 路由、启动 HTTP 服务并返回 server 实例（用于优雅关闭）。
func runServer(cfg *config.Config, marketHandler *market.HTTPHandler, authHandler *auth.HTTPHandler, authService *auth.Service, anomalyHandler *anomaly.HTTPHandler, notifyHandler *notify.HTTPHandler, dailyHandler *daily.HTTPHandler) *http.Server {
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.Use(gin.Recovery(), middleware.RequestID(), middleware.AccessLog())
	api.RegisterRoutes(router, marketHandler, authHandler, authService, cfg.Auth, anomalyHandler, notifyHandler, dailyHandler)

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
func gracefulShutdown(server *http.Server, collector *market.Collector, dispatcher *notify.Dispatcher, dailyScheduler *daily.Scheduler) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("notify: 正在关闭，等待推送完成...")

	if dispatcher != nil {
		dispatcher.Wait()
		slog.Info("notify: 所有推送已完成")
	}

	if dailyScheduler != nil {
		dailyScheduler.Stop()
		slog.Info("daily: 收盘总结调度器已停止")
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
