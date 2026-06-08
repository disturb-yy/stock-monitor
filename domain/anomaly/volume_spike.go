package anomaly

import (
	"context"
	"fmt"
	"time"
)

// VolumeSpikeDetector 是成交量突增检测器。
// 当日成交量超过近 N 日均值的 M 倍时触发"放量异动"。
type VolumeSpikeDetector struct {
	LookbackDays    int     // 回溯天数，如 5 表示计算近 5 日均值
	SpikeMultiplier float64 // 突增倍数阈值，如 2.0 表示超过 2 倍触发
}

// NewVolumeSpikeDetector 创建成交量突增检测器。
func NewVolumeSpikeDetector(lookbackDays int, spikeMultiplier float64) *VolumeSpikeDetector {
	return &VolumeSpikeDetector{LookbackDays: lookbackDays, SpikeMultiplier: spikeMultiplier}
}

func (d *VolumeSpikeDetector) Type() AnomalyType {
	return AnomalyVolumeSpike
}

// Detect 遍历所有指数，比较当日成交量与近 N 日均值。
// 历史数据不足 N 天时自动跳过该指数。
func (d *VolumeSpikeDetector) Detect(ctx context.Context, current map[string]IndexData, historical map[string][]HistoricalDay) ([]AnomalyEvent, error) {
	var events []AnomalyEvent
	now := time.Now()

	for _, idx := range current {
		hist, ok := historical[idx.Symbol]
		if !ok || len(hist) < d.LookbackDays {
			continue // 历史数据不足，跳过
		}

		// 计算近 N 日均成交量
		var sum float64
		count := 0
		for i := 0; i < d.LookbackDays && i < len(hist); i++ {
			if hist[i].Volume > 0 {
				sum += hist[i].Volume
				count++
			}
		}
		if count == 0 {
			continue
		}
		avg := sum / float64(count)

		// 比较当前量与均值的比值
		if avg > 0 && idx.Volume > avg*d.SpikeMultiplier {
			ratio := idx.Volume / avg
			events = append(events, AnomalyEvent{
				ID:           fmt.Sprintf("volume_spike_%s_%s", idx.Symbol, now.Format("20060102_150405")),
				Type:         AnomalyVolumeSpike,
				Symbol:       idx.Symbol,
				IndexName:    idx.Name,
				TriggerValue: ratio,
				Threshold:    d.SpikeMultiplier,
				Description:  fmt.Sprintf("%s 成交量 %.2f 倍于近 %d 日均值，超过阈值 %.2f", idx.Name, ratio, d.LookbackDays, d.SpikeMultiplier),
				TriggeredAt:  now,
			})
		}
	}
	return events, nil
}
