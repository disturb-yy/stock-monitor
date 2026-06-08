// Package anomaly 提供异动检测规则引擎。
// 支持涨跌幅、成交量突增、连续涨跌三种检测规则，
// 通过可配置的检测器管道对实时行情数据进行分析。
package anomaly

import "time"

// AnomalyType 表示异动事件的类型分类。
type AnomalyType string

const (
	AnomalyPriceUp         AnomalyType = "price_up"          // 上涨异动
	AnomalyPriceDown       AnomalyType = "price_down"        // 下跌异动
	AnomalyVolumeSpike     AnomalyType = "volume_spike"      // 放量异动
	AnomalyConsecutiveUp   AnomalyType = "consecutive_up"    // 连续上涨
	AnomalyConsecutiveDown AnomalyType = "consecutive_down"  // 连续下跌
)

// AnomalyEvent 表示一条异动检测结果。
type AnomalyEvent struct {
	ID           string      `json:"id"`           // 事件唯一标识
	Type         AnomalyType `json:"type"`         // 事件类型
	Symbol       string      `json:"symbol"`       // 指数代码，如 "000001.SH"
	IndexName    string      `json:"indexName"`    // 指数中文名称
	TriggerValue float64     `json:"triggerValue"` // 触发值（涨跌幅%、成交量倍数、连续天数）
	Threshold    float64     `json:"threshold"`    // 配置阈值
	Description  string      `json:"description"`  // 人类可读描述
	TriggeredAt  time.Time   `json:"triggeredAt"`  // 触发时间
}
