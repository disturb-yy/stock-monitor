// Package notify 提供 Webhook 告警推送功能。
package notify

import "time"

// AlertType 表示告警事件类型。
type AlertType string

const (
	AlertPriceUp         AlertType = "price_up"         // 上涨异动
	AlertPriceDown       AlertType = "price_down"       // 下跌异动
	AlertVolumeSpike     AlertType = "volume_spike"     // 放量异动
	AlertConsecutiveUp   AlertType = "consecutive_up"   // 连续上涨
	AlertConsecutiveDown AlertType = "consecutive_down" // 连续下跌
)

// AlertEvent 是通知领域内的告警事件模型。
// 由 Composition Root 从 anomaly.AnomalyEvent 转换而来，
// 避免 domain/notify 直接依赖 domain/anomaly。
type AlertEvent struct {
	ID           string    // 事件唯一标识
	Type         AlertType // 事件类型
	Symbol       string    // 指数代码，如 "000001.SH"
	IndexName    string    // 指数中文名称
	TriggerValue float64   // 触发值
	Threshold    float64   // 配置阈值
	TriggeredAt  time.Time // 触发时间
}
