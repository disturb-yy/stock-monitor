// Package config 负责加载和提供全局配置。
// 配置来源为 YAML 文件（默认 configs/config.yaml），
// 敏感字段（如 Tushare Token）可通过环境变量覆盖。
package config

import (
	"os"
	"time"

	"github.com/disturb-yy/stock-monitor/domain/auth"
	"gopkg.in/yaml.v3"
)

const defaultPath = "configs/config.yaml"

var conf *Config

// Config 为全局配置的根结构。
type Config struct {
	Server   ServerConfig   `yaml:"server"`   // HTTP 服务器配置
	Log      LogConfig      `yaml:"log"`      // 日志配置
	Auth     AuthConfig     `yaml:"auth"`     // 认证配置
	Provider ProviderConfig `yaml:"provider"` // 数据源配置
	Tushare  TushareConfig  `yaml:"tushare"`  // Tushare API 配置
	Calendar CalendarConfig `yaml:"calendar"` // 交易日历配置
	Anomaly   AnomalyConfig   `yaml:"anomaly"`   // 异动检测配置
	Webhook   WebhookConfig   `yaml:"webhook"`   // Webhook 告警推送配置
	Collector CollectorConfig `yaml:"collector"` // 定时采集配置
}

// ServerConfig 为 HTTP 服务器配置。
type ServerConfig struct {
	Addr                string `yaml:"addr"`                  // 监听地址，默认 ":30080"
	Mode                string `yaml:"mode"`                  // gin 运行模式
	ReadTimeoutSeconds  int    `yaml:"read_timeout_seconds"`  // 读超时
	WriteTimeoutSeconds int    `yaml:"write_timeout_seconds"` // 写超时
}

// LogConfig 为日志配置。
type LogConfig struct {
	Level      string `yaml:"level"`       // 日志级别
	Filename   string `yaml:"filename"`    // 日志文件路径
	MaxSize    int    `yaml:"max_size"`    // 单文件最大 MB
	MaxBackups int    `yaml:"max_backups"` // 最大备份数
	MaxAge     int    `yaml:"max_age"`     // 最大保留天数
	Compress   bool   `yaml:"compress"`    // 是否压缩
	Console    bool   `yaml:"console"`     // 是否输出到控制台
	Format     string `yaml:"format"`      // 日志格式 json/text
}

// AuthConfig 为 JWT 认证配置。
type AuthConfig struct {
	Enabled         bool   `yaml:"enabled"`           // 是否启用认证
	Secret          string `yaml:"secret"`            // 签名密钥
	Issuer          string `yaml:"issuer"`            // JWT 签发者
	Audience        string `yaml:"audience"`          // JWT 受众
	TokenTTLSeconds int    `yaml:"token_ttl_seconds"` // Token 有效期（秒）
}

// ProviderConfig 为数据源选择配置。
type ProviderConfig struct {
	Type string `yaml:"type"` // "mock" 或 "tushare"
}

// IndexConfig 为单个指数追踪配置。
type IndexConfig struct {
	Symbol string `yaml:"symbol"` // 指数代码如 "000001.SH"
	Name   string `yaml:"name"`   // 中文名称
}

// TushareConfig 为 Tushare Pro API 连接配置。
type TushareConfig struct {
	Token   string        `yaml:"token"`    // API Token（可通过 TUSHARE_TOKEN 环境变量覆盖）
	BaseURL string        `yaml:"base_url"` // API 地址
	Timeout int           `yaml:"timeout"`  // 请求超时（秒）
	Indices []IndexConfig `yaml:"indices"`  // 追踪的指数列表
}

// CalendarConfig 为交易日历配置。
type CalendarConfig struct {
	CacheFile          string `yaml:"cache_file"`           // 本地缓存文件路径
	UpdateIntervalDays int    `yaml:"update_interval_days"` // 缓存刷新间隔（天）
	LookbackMonths     int    `yaml:"lookback_months"`      // 向前覆盖月数
	LookaheadMonths    int    `yaml:"lookahead_months"`     // 向后覆盖月数
}

// AnomalyRuleConfig 为单条异动检测规则配置。
type AnomalyRuleConfig struct {
	Type            string  `yaml:"type"`             // price_change / volume_spike / consecutive
	UpperThreshold  float64 `yaml:"upper_threshold"`  // 涨跌幅上阈值
	LowerThreshold  float64 `yaml:"lower_threshold"`  // 涨跌幅下阈值
	LookbackDays    int     `yaml:"lookback_days"`    // 成交量均线回溯天数
	SpikeMultiplier float64 `yaml:"spike_multiplier"` // 成交量突增倍数
	Days            int     `yaml:"days"`             // 连续涨跌天数
	Enabled         bool    `yaml:"enabled"`          // 是否启用
}

// AnomalyConfig 为异动检测引擎配置。
type AnomalyConfig struct {
	Enabled bool                `yaml:"enabled"` // 是否启用异动检测
	Rules   []AnomalyRuleConfig `yaml:"rules"`   // 检测规则列表
}


// CollectorConfig 为后台定时采集器配置。
type CollectorConfig struct {
	Enabled         bool `yaml:"enabled"`          // 是否启用定时采集
	IntervalMinutes int  `yaml:"interval_minutes"` // 采集间隔（分钟）
	MaxHistoryDays  int  `yaml:"max_history_days"` // 每个指数内存最多保留天数
}

// WebhookConfig 为 Webhook 告警推送配置。
type WebhookConfig struct {
	Enabled              bool   `yaml:"enabled"`                 // 是否启用 Webhook 推送
	WebhookURL           string `yaml:"webhook_url"`             // 企业微信机器人 Webhook URL
	CooldownMinutes      int    `yaml:"cooldown_minutes"`        // 冷却时间（分钟），0 禁用
	RetryCount           int    `yaml:"retry_count"`             // 最大重试次数
	RetryIntervalSeconds int    `yaml:"retry_interval_seconds"`  // 重试间隔（秒）
}

// AuthConfig 转换为 domain/auth.Config 格式。
func (c AuthConfig) AuthConfig() auth.Config {
	return auth.Config{
		Enabled:  c.Enabled,
		Secret:   c.Secret,
		Issuer:   c.Issuer,
		Audience: c.Audience,
		TokenTTL: time.Duration(c.TokenTTLSeconds) * time.Second,
	}
}

// Init 加载全局配置。优先从 CONFIG_PATH 环境变量指定的路径加载。
func Init() {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = defaultPath
	}

	cfg := defaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		panic("read config err: " + err.Error())
	}
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		panic("parse config err: " + err.Error())
	}

	conf = &cfg

	// 环境变量覆盖 Tushare Token（避免敏感信息写入配置文件）
	if token := os.Getenv("TUSHARE_TOKEN"); token != "" {
		conf.Tushare.Token = token
	}
}

// Get 返回全局配置。未初始化时返回默认配置。
func Get() *Config {
	if conf == nil {
		cfg := defaultConfig()
		conf = &cfg
	}
	return conf
}

// defaultConfig 提供所有配置的默认值。
func defaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Addr:                ":30080",
			Mode:                "debug",
			ReadTimeoutSeconds:  10,
			WriteTimeoutSeconds: 10,
		},
		Log: LogConfig{
			Level:      "info",
			Filename:   "./logs/app.log",
			MaxSize:    100,
			MaxBackups: 10,
			MaxAge:     30,
			Compress:   true,
			Console:    true,
			Format:     "json",
		},
		Auth: AuthConfig{
			Enabled:         false,
			TokenTTLSeconds: 3600,
		},
		Provider: ProviderConfig{
			Type: "mock",
		},
		Tushare: TushareConfig{
			BaseURL: "http://api.tushare.pro",
			Timeout: 30,
			Indices: []IndexConfig{
				{Symbol: "000001.SH", Name: "上证指数"},
				{Symbol: "399001.SZ", Name: "深证成指"},
				{Symbol: "399006.SZ", Name: "创业板指"},
				{Symbol: "000688.SH", Name: "科创50"},
				{Symbol: "000300.SH", Name: "沪深300"},
				{Symbol: "000905.SH", Name: "中证500"},
			},
		},
		Calendar: CalendarConfig{
			CacheFile:          "data/trading_calendar.json",
			UpdateIntervalDays: 7,
			LookbackMonths:     6,
			LookaheadMonths:    6,
		},
		Collector: CollectorConfig{
			Enabled:         true,
			IntervalMinutes: 5,
			MaxHistoryDays:  30,
		},
		Anomaly: AnomalyConfig{
			Enabled: true,
			Rules: []AnomalyRuleConfig{
				{Type: "price_change", UpperThreshold: 3.0, LowerThreshold: -2.0, Enabled: true},
				{Type: "volume_spike", LookbackDays: 5, SpikeMultiplier: 2.0, Enabled: true},
				{Type: "consecutive", Days: 5, Enabled: true},
			},
		},
	}
}
