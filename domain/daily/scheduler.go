package daily

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Scheduler 按时触发每日收盘总结的生成与推送。
// 使用 time.Ticker 每分钟检查一次是否到达推送时间。
type Scheduler struct {
	builder  *Builder      // 收盘总结构建器
	push     PushFunc      // 推送回调
	pushTime string        // 推送时间（HH:MM，如 "15:15"）
	location *time.Location // 时区
	stopCh   chan struct{} // 停止信号
	wg       sync.WaitGroup
}

// NewScheduler 创建收盘总结定时调度器。
// pushTime 格式为 HH:MM（如 "15:15"）。
func NewScheduler(builder *Builder, push PushFunc, pushTime string, location *time.Location) *Scheduler {
	if location == nil {
		location = time.Local
	}
	return &Scheduler{
		builder:  builder,
		push:     push,
		pushTime: pushTime,
		location: location,
		stopCh:   make(chan struct{}),
	}
}

// Start 启动定时调度（每分钟检查一次，非阻塞）。
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		slog.Info("daily: 收盘总结调度器已启动", "push_time", s.pushTime)

		// 立即检查一次，避免启动后需等待第一个 tick 间隔
		s.tick()

		for {
			select {
			case <-ticker.C:
				s.tick()
			case <-s.stopCh:
				slog.Info("daily: 收盘总结调度器已停止")
				return
			}
		}
	}()
}

// Stop 停止调度器并等待当前推送完成。
func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

// tick 检查当前时间是否匹配推送时间，并触发推送。
func (s *Scheduler) tick() {
	now := time.Now().In(s.location)
	targetHHMM := now.Format("15:04")

	if targetHHMM != s.pushTime {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	slog.Info("daily: 到达推送时间，开始生成收盘总结", "time", targetHHMM)

	err := s.builder.BuildAndPush(ctx, s.push)
	if err != nil {
		if err == ErrNotTradingDay {
			slog.Info("daily: 非交易日，跳过推送")
		} else {
			slog.Error("daily: 收盘总结推送失败", "error", err)
		}
	}
}
