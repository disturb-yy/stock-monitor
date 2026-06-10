package anomaly

import "context"

// EventCallback 是异动事件的回调函数类型。
// 每次 Detect 完成后，将检测到的事件列表传递给注册的回调。
// 回调内部应异步处理，不得长时间阻塞检测引擎。
type EventCallback func(ctx context.Context, events []AnomalyEvent)

// Engine 是异动检测引擎，负责协调多个 Detector 的执行。
// 采用管道模式：注册一组检测器，依次执行并聚合结果。
type Engine struct {
	detectors  []Detector
	callback   EventCallback // 异动事件回调（可选）
	lastEvents []AnomalyEvent // 最近一次检测的事件（供 daily 领域查询）
}

// NewEngine 创建检测引擎并注册给定的检测器。
func NewEngine(detectors ...Detector) *Engine {
	return &Engine{detectors: detectors}
}

// SetCallback 注册异动事件回调。
// 每次 Detect 返回事件后，自动调用此回调。
// 设为 nil 可移除回调。
func (e *Engine) SetCallback(cb EventCallback) {
	e.callback = cb
}

// Detect 依次运行所有已注册的检测器，收集并返回全部事件。
// 单个检测器失败不影响其他检测器继续执行。
// 若注册了回调，检测完成后将事件异步传递给回调。
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

	// 存储最近一次检测结果（供 daily 领域查询）
	e.lastEvents = allEvents

	// 异步调用回调，不阻塞检测流程
	if e.callback != nil && len(allEvents) > 0 {
		go e.callback(context.Background(), allEvents)
	}

	return allEvents
}

// GetLastEvents 返回最近一次检测的异动事件列表。
// 用于 daily 领域获取当日异动摘要（不重复执行检测）。
func (e *Engine) GetLastEvents() []AnomalyEvent {
	return e.lastEvents
}
