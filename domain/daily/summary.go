package daily

import "time"

// DataStatus 表示收盘总结的数据获取状态。
type DataStatus string

const (
	DataStatusOK      DataStatus = "ok"      // 数据正常
	DataStatusPartial DataStatus = "partial" // 部分数据获取失败
	DataStatusFailed  DataStatus = "failed"  // 完全无法获取数据
)

// AnomalySummary 是异动事件的摘要统计。
type AnomalySummary struct {
	TotalCount int           // 异动事件总数
	Items      []AnomalyItem // 异动条目列表（最多 5 条）
}

// DailySummary 是每日收盘总结的完整数据。
type DailySummary struct {
	TradeDate      string          // 交易日期（YYYY-MM-DD）
	IsTradingDay   bool            // 是否为交易日
	UpCount        int             // 上涨指数数量
	DownCount      int             // 下跌指数数量
	FlatCount      int             // 持平指数数量
	TotalIndices   int             // 追踪指数总数
	TopGainer      *IndexSnapshot  // 领涨指数（可为 nil）
	TopLoser       *IndexSnapshot  // 领跌指数（可为 nil）
	Indices        []IndexSnapshot // 所有指数快照列表
	AnomalySummary AnomalySummary  // 异动事件摘要
	DataStatus     DataStatus      // 数据获取状态
	GeneratedAt    time.Time       // 总结生成时间
}

// PushFunc 是推送回调函数类型。
// daily 领域不直接依赖 notify，由 Composition Root 注入适配器。
type PushFunc func(summary DailySummary) error
