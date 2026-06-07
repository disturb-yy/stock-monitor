package config

import (
	"os"
	"time"

	"github.com/disturb-yy/stock-monitor/domain/auth"
	"gopkg.in/yaml.v3"
)

const defaultPath = "configs/config.yaml"

var conf *Config

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Log     LogConfig     `yaml:"log"`
	Auth    AuthConfig    `yaml:"auth"`
	Tushare TushareConfig `yaml:"tushare"`
}

type ServerConfig struct {
	Addr                string `yaml:"addr"`
	Mode                string `yaml:"mode"`
	ReadTimeoutSeconds  int    `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `yaml:"write_timeout_seconds"`
}

type LogConfig struct {
	Level      string `yaml:"level"`
	Filename   string `yaml:"filename"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
	Console    bool   `yaml:"console"`
	Format     string `yaml:"format"`
}

type AuthConfig struct {
	Enabled         bool   `yaml:"enabled"`
	Secret          string `yaml:"secret"`
	Issuer          string `yaml:"issuer"`
	Audience        string `yaml:"audience"`
	TokenTTLSeconds int    `yaml:"token_ttl_seconds"`
}

type TushareConfig struct {
	Token   string `yaml:"token"`
	BaseURL string `yaml:"base_url"`
}

func (c AuthConfig) AuthConfig() auth.Config {
	return auth.Config{
		Enabled:  c.Enabled,
		Secret:   c.Secret,
		Issuer:   c.Issuer,
		Audience: c.Audience,
		TokenTTL: time.Duration(c.TokenTTLSeconds) * time.Second,
	}
}

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
}

func Get() *Config {
	if conf == nil {
		cfg := defaultConfig()
		conf = &cfg
	}
	return conf
}

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
		Tushare: TushareConfig{
			BaseURL: "http://api.tushare.pro",
		},
	}
}
