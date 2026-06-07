# pkg/logger — 日志封装

## 包名

`logger`

## 文件

`logger.go`

## 公开类型

### 配置

```go
type Config struct {
    Level      string   // debug / info / warn / error
    Filename   string   // 日志文件路径
    MaxSize    int      // MB
    MaxBackups int      // 最大备份数
    MaxAge     int      // 最大保留天数
    Compress   bool     // 是否压缩
    Console    bool     // 是否同时输出到控制台
    Format     string   // "json" 或 "text"
}
```

### 常量

| 常量 | 值 |
| --- | --- |
| `LogTimeFormat` | `"2006-01-02 15:04:05.000"` |
| `FormatJSON` | `"json"` |
| `FormatText` | `"text"` |

### 函数

```go
func Init(cfg Config)                                                 // 初始化全局 logger
func Sync()                                                           // 刷新缓冲（当前为空操作）
func Info(ctx context.Context, msg string, args ...any)               // Info 级别
func Error(ctx context.Context, msg string, args ...any)              // Error 级别
func Debug(ctx context.Context, msg string, args ...any)              // Debug 级别
func Warn(ctx context.Context, msg string, args ...any)               // Warn 级别
func Fatal(msg string, args ...any)                                   // 记录 Fatal 并 os.Exit(1)
func WithContext(ctx context.Context, args ...any) context.Context    // 返回带 slog 属性的 context
```

## 底层

基于 `log/slog` + `gopkg.in/natefinch/lumberjack.v2`（日志轮转）。

- 文件轮转：`lumberjack` 管理 `Filename`、`MaxSize`、`MaxBackups`、`MaxAge`、`Compress`
- 格式：`FormatJSON` 输出 JSON handler，其他输出 Text handler
- `AddSource: true`：每条日志带源文件位置
- 时间格式为 `LogTimeFormat`

## 依赖

| 依赖 | 用途 |
| --- | --- |
| `log/slog` | 结构化日志 |
| `gopkg.in/natefinch/lumberjack.v2` | 日志轮转 |
