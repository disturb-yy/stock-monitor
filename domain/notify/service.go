package notify

import (
	"context"
	"log/slog"
)

// Service 是通知服务。
// 通过 EventCallback 接收告警事件并分发给 Dispatcher。
type Service struct {
	dispatcher *Dispatcher
	enabled    bool
}

// NewService 创建通知服务。
// enabled 为 false 时，HandleEvents 不执行任何操作。
func NewService(dispatcher *Dispatcher, enabled bool) *Service {
	return &Service{
		dispatcher: dispatcher,
		enabled:    enabled,
	}
}

// HandleEvents 处理一批告警事件，转发给 Dispatcher 进行异步推送。
func (s *Service) HandleEvents(ctx context.Context, events []AlertEvent) {
	if !s.enabled {
		return
	}
	if len(events) == 0 {
		return
	}
	slog.Info("notify: 收到告警事件", "count", len(events))
	s.dispatcher.Dispatch(ctx, events)
}

// Enabled 返回通知服务是否已启用。
func (s *Service) Enabled() bool {
	return s.enabled
}
