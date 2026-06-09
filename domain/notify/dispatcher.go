package notify

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// retriable 判断 HTTP 状态码是否可重试。
// 仅 5xx 和 429 需要重试；4xx 属于客户端错误，重试无意义。
func retriable(statusCode int) bool {
	return statusCode >= 500 || statusCode == http.StatusTooManyRequests
}

// Dispatcher 是推送调度器，负责异步发送、重试、去重过滤和推送记录。
type Dispatcher struct {
	sender        WebhookSender
	cooldown      *CooldownTracker // 冷却去重（可为 nil）
	history       *HistoryBuffer   // 推送历史（可为 nil）
	retryCount    int
	retryInterval time.Duration
	wg            sync.WaitGroup
}

// NewDispatcher 创建推送调度器。
// cooldown 和 history 可为 nil（分别禁用冷却和推送历史）。
func NewDispatcher(sender WebhookSender, cooldown *CooldownTracker, history *HistoryBuffer, retryCount int, retryInterval time.Duration) *Dispatcher {
	return &Dispatcher{
		sender:        sender,
		cooldown:      cooldown,
		history:       history,
		retryCount:    retryCount,
		retryInterval: retryInterval,
	}
}

// SetCooldown 设置冷却追踪器（用于延迟注入场景）。
func (d *Dispatcher) SetCooldown(c *CooldownTracker) {
	d.cooldown = c
}

// SetHistory 设置推送历史缓冲区（用于延迟注入场景）。
func (d *Dispatcher) SetHistory(h *HistoryBuffer) {
	d.history = h
}

// Dispatch 处理一批告警事件：冷却过滤 → 合并消息 → 异步发送。
// 调用立即返回，实际推送在后台 goroutine 中执行。
func (d *Dispatcher) Dispatch(ctx context.Context, events []AlertEvent) {
	if len(events) == 0 {
		return
	}

	// 1. 冷却过滤
	validEvents, skippedEvents := d.applyCooldown(events)

	// 2. 将跳过的冷却事件记录到历史
	for _, e := range skippedEvents {
		d.recordSkipped(e)
	}

	// 3. 如果没有可通过冷却的事件，直接返回
	if len(validEvents) == 0 {
		slog.Debug("notify: 全部事件被冷却过滤，不推送", "total", len(events), "skipped", len(skippedEvents))
		return
	}

	// 4. 构建合并消息
	content := BuildMessage(validEvents)

	// 5. 异步发送（使用 background context 避免父 ctx 取消中断推送）
	eventIDs := extractIDs(validEvents)
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.sendWithRetry(context.Background(), content, eventIDs)
	}()
}

// Wait 等待所有进行中的推送完成（用于优雅关闭）。
func (d *Dispatcher) Wait() {
	d.wg.Wait()
}

// applyCooldown 应用冷却过滤，返回（有效事件, 被跳过事件）。
func (d *Dispatcher) applyCooldown(events []AlertEvent) (valid, skipped []AlertEvent) {
	if d.cooldown == nil {
		return events, nil
	}
	for _, e := range events {
		if d.cooldown.ShouldSkip(e.Symbol, e.Type) {
			skipped = append(skipped, e)
		} else {
			valid = append(valid, e)
		}
	}
	return
}

// recordSkipped 记录被冷却跳过的事件。
func (d *Dispatcher) recordSkipped(e AlertEvent) {
	if d.history == nil {
		return
	}
	d.history.Add(DeliveryRecord{
		EventIDs:    []string{e.ID},
		Status:      StatusSkippedCooldown,
		Error:       fmt.Sprintf("冷却中 (symbol=%s, type=%s)", e.Symbol, e.Type),
		DeliveredAt: time.Now(),
		CompletedAt: time.Now(),
	})
}

// sendWithRetry 发送消息并在失败时按策略重试。
func (d *Dispatcher) sendWithRetry(ctx context.Context, content string, eventIDs []string) {
	now := time.Now()

	for attempt := 0; attempt <= d.retryCount; attempt++ {
		statusCode, err := d.sender.Send(ctx, content)

		if err == nil {
			d.recordSuccess(content, eventIDs, now, attempt)
			return
		}

		// 不可重试的错误（4xx 非 429）直接标记失败
		if statusCode > 0 && !retriable(statusCode) {
			slog.Error("notify: 推送永久失败（不可重试）", "status", statusCode, "error", err)
			d.recordFailure(content, eventIDs, now, attempt, err)
			return
		}

		// 最后一次尝试仍失败
		if attempt >= d.retryCount {
			slog.Error("notify: 推送最终失败（重试耗尽）", "attempts", attempt+1, "error", err)
			d.recordFailure(content, eventIDs, now, attempt, err)
			return
		}

		// 临时故障，等待后重试
		slog.Warn("notify: 推送失败，准备重试", "attempt", attempt+1, "status", statusCode, "error", err)
		select {
		case <-ctx.Done():
			d.recordFailure(content, eventIDs, now, attempt, ctx.Err())
			return
		case <-time.After(d.retryInterval):
		}
	}
}

// recordSuccess 记录成功推送。
func (d *Dispatcher) recordSuccess(content string, eventIDs []string, started time.Time, retries int) {
	if d.history == nil {
		return
	}
	d.history.Add(DeliveryRecord{
		EventIDs:    eventIDs,
		Status:      StatusSuccess,
		Message:     truncateContent(content, 200),
		RetryCount:  retries,
		DeliveredAt: started,
		CompletedAt: time.Now(),
	})
}

// recordFailure 记录失败推送。
func (d *Dispatcher) recordFailure(content string, eventIDs []string, started time.Time, retries int, err error) {
	if d.history == nil {
		return
	}
	d.history.Add(DeliveryRecord{
		EventIDs:    eventIDs,
		Status:      StatusFailed,
		Message:     truncateContent(content, 200),
		Error:       err.Error(),
		RetryCount:  retries,
		DeliveredAt: started,
		CompletedAt: time.Now(),
	})
}

// extractIDs 提取事件 ID 列表。
func extractIDs(events []AlertEvent) []string {
	ids := make([]string, 0, len(events))
	for _, e := range events {
		if e.ID != "" {
			ids = append(ids, e.ID)
		}
	}
	return ids
}

// truncateContent 截断内容到指定长度。
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}
