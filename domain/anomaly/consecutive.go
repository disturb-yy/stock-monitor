package anomaly

import (
	"context"
	"fmt"
	"time"
)

// ConsecutiveDetector 是连续涨跌检测器。
// 指数连续 N 个交易日同向变动时触发"连续上涨"或"连续下跌"异动。
type ConsecutiveDetector struct {
	Days int // 连续天数阈值，如 5 表示连续 5 天触发
}

// NewConsecutiveDetector 创建连续涨跌检测器。
func NewConsecutiveDetector(days int) *ConsecutiveDetector {
	return &ConsecutiveDetector{Days: days}
}

func (d *ConsecutiveDetector) Type() AnomalyType {
	return AnomalyConsecutiveUp
}

// Detect 遍历所有指数，检查当日与前 N-1 日是否全部同向（涨或跌）。
// 节假日导致的日期断档不影响判断——只管交易日序号。
func (d *ConsecutiveDetector) Detect(ctx context.Context, current map[string]IndexData, historical map[string][]HistoricalDay) ([]AnomalyEvent, error) {
	var events []AnomalyEvent
	now := time.Now()

	for _, idx := range current {
		hist, ok := historical[idx.Symbol]
		if !ok || len(hist) < d.Days-1 {
			continue // 历史数据不足，跳过
		}

		// 构建涨跌方向序列：当日 + 近 N-1 日
		directions := make([]int, 0, d.Days)
		todayDir := sign(idx.Close - idx.PreClose)
		directions = append(directions, todayDir)
		for i := 0; i < d.Days-1 && i < len(hist); i++ {
			if i+1 < len(hist) {
				dir := sign(hist[i].Close - hist[i+1].Close)
				directions = append(directions, dir)
			}
		}

		if len(directions) < d.Days {
			continue
		}

		// 检查是否全部同向（全部 >0 或全部 <0）
		allSame := true
		ref := directions[0]
		for _, dir := range directions[1:] {
			if dir != ref || dir == 0 {
				allSame = false
				break
			}
		}

		if allSame && ref > 0 {
			events = append(events, AnomalyEvent{
				ID:           fmt.Sprintf("consecutive_up_%s_%s", idx.Symbol, now.Format("20060102_150405")),
				Type:         AnomalyConsecutiveUp,
				Symbol:       idx.Symbol,
				IndexName:    idx.Name,
				TriggerValue: float64(d.Days),
				Threshold:    float64(d.Days),
				Description:  fmt.Sprintf("%s 连续 %d 个交易日上涨", idx.Name, d.Days),
				TriggeredAt:  now,
			})
		} else if allSame && ref < 0 {
			events = append(events, AnomalyEvent{
				ID:           fmt.Sprintf("consecutive_down_%s_%s", idx.Symbol, now.Format("20060102_150405")),
				Type:         AnomalyConsecutiveDown,
				Symbol:       idx.Symbol,
				IndexName:    idx.Name,
				TriggerValue: float64(d.Days),
				Threshold:    float64(d.Days),
				Description:  fmt.Sprintf("%s 连续 %d 个交易日下跌", idx.Name, d.Days),
				TriggeredAt:  now,
			})
		}
	}
	return events, nil
}

// sign 返回数值的符号：正数→1，负数→-1，零→0。
func sign(v float64) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}
