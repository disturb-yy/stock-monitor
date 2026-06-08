<!-- SPECKIT START -->
当前实施计划: specs/003-anomaly-detection/plan.md
<!-- SPECKIT END -->

## 代码规范

### 注释语言

所有 Go 代码 **必须** 使用中文注释：

- 包级别注释：解释包的职责和用途
- 导出类型/函数/方法注释：说明功能和关键参数含义
- 结构体字段注释：说明字段含义和数据来源
- 关键代码段注释：解释非显而易见的逻辑（如计算公式、降级策略）
- 日志和错误消息：使用中文以便运维排查

示例：

```go
// GetMarketStatus 获取当前 A 股市场状态。
// 判断逻辑：先通过交易日历判断是否为交易日，
// 再通过服务器本地时间判断当前交易时段。
func (p *TushareProvider) GetMarketStatus(ctx context.Context) (*MarketSession, error) {
```

### 命名规范

- 包名描述领域，文件名描述职责
- 不使用 `models`、`services`、`repositories` 技术角色包名
- 包名已表达领域时，类型名不重复领域名（如 `market.Service` 而非 `market.MarketService`）

### DDF 架构规范

- 业务领域代码放在 `domain/{领域名}/` 下，每个领域自包含
- 公共组件放在 `pkg/` 下
- 跨领域协作通过 `cmd/server/main.go`（Composition Root）组装
- `pkg/*` 不得导入 `domain/*`
- 领域包之间不得互相导入

### INDEX.md 规范

- 每个导出代码的目录必须包含 `INDEX.md`
- 描述目录用途、公开 API、依赖关系
- 代码变更时同步更新
