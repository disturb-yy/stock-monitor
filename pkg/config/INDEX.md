# pkg/config — 配置加载

## 包名

`config`

## 文件

`config.go`

## 公开类型

### 顶层配置

```go
type Config struct {
    Server  ServerConfig
    Log     LogConfig
    Auth    AuthConfig
    Tushare TushareConfig
}
```

### 子配置

```go
type ServerConfig struct {
    Addr                string  // 监听地址，默认 ":30080"
    Mode                string  // gin mode，默认 "debug"
    ReadTimeoutSeconds  int     // 读超时（秒），默认 10
    WriteTimeoutSeconds int     // 写超时（秒），默认 10
}

type LogConfig struct {
    Level      string  // 日志级别，默认 "info"
    Filename   string  // 日志文件路径，默认 "./logs/app.log"
    MaxSize    int     // 单文件最大 MB，默认 100
    MaxBackups int     // 最大备份数，默认 10
    MaxAge     int     // 最大保留天数，默认 30
    Compress   bool    // 是否压缩，默认 true
    Console    bool    // 是否输出到控制台，默认 true
    Format     string  // 格式 "json" 或 "text"，默认 "json"
}

type AuthConfig struct {
    Enabled         bool    // 是否启用认证，默认 false
    Secret          string  // HMAC 签名密钥
    Issuer          string  // JWT issuer
    Audience        string  // JWT audience
    TokenTTLSeconds int     // Token 有效期（秒），默认 3600
}

type TushareConfig struct {
    Token   string  // Tushare API token（可通过 TUSHARE_TOKEN 环境变量覆盖）
    BaseURL string  // Tushare API 地址，默认 "http://api.tushare.pro"
}
```

### 函数

```go
func Init()              // 从 CONFIG_PATH 或 "configs/config.yaml" 加载
func Get() *Config       // 获取全局配置（未 Init 时返回默认值）
```

### 方法

```go
func (c AuthConfig) AuthConfig() auth.Config  // 转换为 domain/auth.Config
```

## 配置来源

- 默认路径：`configs/config.yaml`
- 环境变量覆盖：`CONFIG_PATH`（指定配置文件路径）、`TUSHARE_TOKEN`（覆盖 tushare.token）、`WEBHOOK_URL`（覆盖 webhook.webhook_url）
- 环境变量优先级高于 YAML 文件值
- 默认值：`defaultConfig()` 内部函数

## 依赖

| 依赖 | 用途 |
| --- | --- |
| `domain/auth` | `AuthConfig()` 返回 `auth.Config` |
| `gopkg.in/yaml.v3` | YAML 解析 |
