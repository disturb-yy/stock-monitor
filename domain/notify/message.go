package notify

import (
	"fmt"
	"strings"
	"time"
)

// 异动类型到中文描述的映射。
var alertTypeNames = map[AlertType]string{
	AlertPriceUp:         "上涨异动",
	AlertPriceDown:       "下跌异动",
	AlertVolumeSpike:     "放量异动",
	AlertConsecutiveUp:   "连续上涨",
	AlertConsecutiveDown: "连续下跌",
}

// BuildMessage 将告警事件列表构建为一条企业微信 Markdown 消息。
// 多事件之间用 "---" 分隔线区分。
// 返回空字符串表示无有效事件。
func BuildMessage(events []AlertEvent) string {
	if len(events) == 0 {
		return ""
	}

	var parts []string
	for _, e := range events {
		parts = append(parts, formatEvent(e))
	}

	header := "## ⚠️ 异动告警\n"
	if len(events) > 1 {
		header = fmt.Sprintf("## ⚠️ 异动告警 (%d条)\n", len(events))
	}

	return header + "\n" + strings.Join(parts, "\n---\n")
}

// formatEvent 格式化单条告警事件为 Markdown 段落。
func formatEvent(e AlertEvent) string {
	typeName := alertTypeNames[e.Type]
	if typeName == "" {
		typeName = string(e.Type)
	}

	var valueStr string
	switch e.Type {
	case AlertPriceUp, AlertPriceDown:
		valueStr = fmt.Sprintf("%+.2f%%", e.TriggerValue)
	case AlertVolumeSpike:
		valueStr = fmt.Sprintf("%.1fx", e.TriggerValue)
	case AlertConsecutiveUp, AlertConsecutiveDown:
		valueStr = fmt.Sprintf("%.0f天", e.TriggerValue)
	default:
		valueStr = fmt.Sprintf("%.2f", e.TriggerValue)
	}

	var thresholdStr string
	switch e.Type {
	case AlertPriceUp, AlertPriceDown:
		thresholdStr = fmt.Sprintf("%.2f%%", e.Threshold)
	case AlertVolumeSpike:
		thresholdStr = fmt.Sprintf("%.1fx", e.Threshold)
	case AlertConsecutiveUp, AlertConsecutiveDown:
		thresholdStr = fmt.Sprintf("%.0f天", e.Threshold)
	default:
		thresholdStr = fmt.Sprintf("%.2f", e.Threshold)
	}

	timeStr := e.TriggeredAt.Format(time.DateTime)

	return fmt.Sprintf("**%s** (%s) 触发 **%s**\n> 触发值：%s（阈值：%s）\n> 触发时间：%s",
		e.IndexName, e.Symbol, typeName, valueStr, thresholdStr, timeStr)
}
