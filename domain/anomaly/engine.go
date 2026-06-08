package anomaly

import "context"

// Engine 是异动检测引擎，负责协调多个 Detector 的执行。
// 采用管道模式：注册一组检测器，依次执行并聚合结果。
type Engine struct {
	detectors []Detector
}

// NewEngine 创建检测引擎并注册给定的检测器。
func NewEngine(detectors ...Detector) *Engine {
	return &Engine{detectors: detectors}
}

// Detect 依次运行所有已注册的检测器，收集并返回全部事件。
// 单个检测器失败不影响其他检测器继续执行。
func (e *Engine) Detect(ctx context.Context, current map[string]IndexData, historical map[string][]HistoricalDay) []AnomalyEvent {
	var allEvents []AnomalyEvent
	for _, d := range e.detectors {
		events, err := d.Detect(ctx, current, historical)
		if err != nil {
			// 单个检测器失败仅跳过，不阻塞其余
			continue
		}
		allEvents = append(allEvents, events...)
	}
	return allEvents
}
