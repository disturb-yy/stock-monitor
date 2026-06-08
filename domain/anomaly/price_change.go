package anomaly

import (
	"context"
	"fmt"
	"time"
)

// PriceChangeDetector 是涨跌幅异动检测器。
// 当日涨跌幅超过配置的上限阈值时触发"上涨异动"，
// 低于下限阈值时触发"下跌异动"。
type PriceChangeDetector struct {
	UpperThreshold float64 // 上限阈值，如 3.0 表示涨超 3% 触发
	LowerThreshold float64 // 下限阈值，如 -2.0 表示跌超 2% 触发
}

// NewPriceChangeDetector 创建涨跌幅检测器。
func NewPriceChangeDetector(upper, lower float64) *PriceChangeDetector {
	return &PriceChangeDetector{UpperThreshold: upper, LowerThreshold: lower}
}

func (d *PriceChangeDetector) Type() AnomalyType {
	return AnomalyPriceUp // 作为事件基类使用，实际事件携带具体 type
}

// Detect 遍历所有指数，检查涨跌幅是否超过阈值。
func (d *PriceChangeDetector) Detect(ctx context.Context, current map[string]IndexData, _ map[string][]HistoricalDay) ([]AnomalyEvent, error) {
	var events []AnomalyEvent
	now := time.Now()

	for _, idx := range current {
		if idx.ChangePct > d.UpperThreshold {
			events = append(events, AnomalyEvent{
				ID:           fmt.Sprintf("price_up_%s_%s", idx.Symbol, now.Format("20060102_150405")),
				Type:         AnomalyPriceUp,
				Symbol:       idx.Symbol,
				IndexName:    idx.Name,
				TriggerValue: idx.ChangePct,
				Threshold:    d.UpperThreshold,
				Description:  fmt.Sprintf("%s 涨幅 %.2f%% 超过阈值 %.2f%%", idx.Name, idx.ChangePct, d.UpperThreshold),
				TriggeredAt:  now,
			})
		}
		if idx.ChangePct < d.LowerThreshold {
			events = append(events, AnomalyEvent{
				ID:           fmt.Sprintf("price_down_%s_%s", idx.Symbol, now.Format("20060102_150405")),
				Type:         AnomalyPriceDown,
				Symbol:       idx.Symbol,
				IndexName:    idx.Name,
				TriggerValue: idx.ChangePct,
				Threshold:    d.LowerThreshold,
				Description:  fmt.Sprintf("%s 跌幅 %.2f%% 超过阈值 %.2f%%", idx.Name, idx.ChangePct, d.LowerThreshold),
				TriggeredAt:  now,
			})
		}
	}
	return events, nil
}
