# 📈 stock-monitor

A 股大盘实时监控系统，支持行情查询、异动检测、企业微信告警推送和每日收盘总结。

## 特性

- **行情查询** — 市场状态、指数行情、大盘总览、历史数据
- **异动检测** — 涨跌幅、成交量突增、连续涨跌三维度规则引擎
- **企业微信告警** — 异动事件自动推送到企微群机器人，支持冷却去重/失败重试/推送历史
- **定时采集** — 后台定时拉取 Tushare 行情数据，内存 + SQLite 双存储
- **收盘总结** — 交易日下午自动推送当日市场总结到企微，支持手动触发和 JSON 查询
- **交易日历** — 本地缓存，周末/节假日智能判断

## 快速开始

### 前置条件

- Go 1.25+
- （可选）[Tushare Pro](https://tushare.pro) Token（用于真实行情）

### 安装与运行

```bash
# 克隆仓库
git clone https://github.com/disturb-yy/stock-monitor.git
cd stock-monitor

# 安装依赖
go mod tidy

# Mock 模式启动（无需外部依赖）
make run
```

服务默认监听 `http://localhost:30083`。

### 配置模式切换

| 模式 | 配置 | 说明 |
|------|------|------|
| Mock | `provider.type: "mock"` | 返回模拟数据，零外部依赖 |
| Tushare | `provider.type: "tushare"` | 真实 A 股行情，需 Token |

## API 接口

### 公开端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/index` | 服务基本信息 |
| GET | `/v1/ready` | 就绪检查 |
| GET | `/v1/heart` | 心跳检查 |

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/token` | 生成 JWT Token |

> 认证默认关闭，可通过 `auth.enabled: true` 启用。

### 行情

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/market/status` | 市场交易状态 |
| GET | `/api/market/indices` | 指数行情列表 |
| GET | `/api/market/overview` | 大盘总览 |
| GET | `/api/market/history?start=YYYY-MM-DD` | 历史行情查询 |

### 异动检测

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/market/anomalies` | 实时异动检测事件 |

### 告警推送

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/market/alerts/history` | 推送历史记录 |

### 每日收盘总结

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/market/daily-summary` | 查询收盘总结 JSON |
| POST | `/api/market/daily-summary` | 手动触发推送 |

## 配置

编辑 `configs/config.yaml`：

```yaml
server:
  addr: ":30083"

provider:
  type: "mock"            # mock / tushare

tushare:
  token: ""               # 或通过 TUSHARE_TOKEN 环境变量设置

anomaly:
  enabled: true
  rules:
    - type: price_change
      upper_threshold: 3.0
      lower_threshold: -2.0

webhook:
  enabled: false
  webhook_url: ""         # 或通过 WEBHOOK_URL 环境变量设置

daily_summary:
  enabled: false
  push_time: "15:15"
```

### 环境变量

| 变量 | 说明 |
|------|------|
| `TUSHARE_TOKEN` | Tushare API Token（覆盖配置文件） |
| `WEBHOOK_URL` | 企业微信 Webhook 地址（覆盖配置文件） |
| `CONFIG_PATH` | 配置文件路径（默认 `configs/config.yaml`） |

## 项目结构

```text
stock-monitor/
├── cmd/server/           # 启动入口，Composition Root
├── api/route.go          # 路由注册
├── domain/               # 业务领域（DDF 架构）
│   ├── market/           #   行情领域
│   ├── auth/             #   认证领域
│   ├── anomaly/          #   异动检测领域
│   ├── notify/           #   Webhook 告警推送领域
│   └── daily/            #   每日收盘总结领域
├── pkg/                  # 公共组件
│   ├── config/           #   配置加载
│   ├── httputil/         #   响应格式
│   ├── logger/           #   日志封装
│   ├── middleware/       #   中间件
│   ├── router/           #   路由辅助
│   └── tushare/          #   Tushare API 客户端
├── configs/config.yaml   # 默认配置
├── specs/                # 功能规约（Spec Kit）
├── Makefile
└── go.mod
```

### 架构原则

采用 **DDF（Domain-Driven Flattening）** 架构：

- 每个领域自包含：模型、接口、服务、HTTP handler 同在一个 package
- 领域之间禁止直接依赖，跨域协作通过 `cmd/server/main.go` Composition Root 组装
- Mock-First：所有外部数据源先有 Mock 实现，再有真实实现
- 详见各领域 `INDEX.md` 和 [项目宪法](.specify/memory/constitution.md)

## 开发

```bash
make run          # 启动服务
make build        # 编译二进制
make test         # 运行全部测试
make tidy         # 整理依赖
make clean        # 清理产物
```

### Spec Kit 工作流

```bash
$speckit-specify <功能描述>   # 创建功能规约
$speckit-clarify               # 澄清需求
$speckit-plan                  # 生成实施计划
$speckit-tasks                 # 分解任务
$speckit-implement             # 执行实现
```

## 路线图

| 阶段 | 状态 | 说明 |
|------|------|------|
| Phase 1 · 看得见 | ✅ | 行情查询、指数数据、大盘总览 |
| Phase 2 · 发现异常 | ✅ | 异动检测、企微告警、SQLite 持久化 |
| Phase 3 · 解释原因 | 🚧 | 每日收盘总结已交付，板块分析待开发 |
| Phase 4 · 辅助决策 | 📋 | 市场评分、自定义规则、多市场扩展 |

## License

MIT
