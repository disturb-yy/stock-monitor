package notify

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ============================================================
// BuildMessage 测试
// ============================================================

func TestBuildMessage_Empty(t *testing.T) {
	if got := BuildMessage(nil); got != "" {
		t.Errorf("BuildMessage(nil) = %q, want empty", got)
	}
	if got := BuildMessage([]AlertEvent{}); got != "" {
		t.Errorf("BuildMessage([]) = %q, want empty", got)
	}
}

func TestBuildMessage_SinglePriceUp(t *testing.T) {
	evt := AlertEvent{
		ID:           "evt-001",
		Type:         AlertPriceUp,
		Symbol:       "000001.SH",
		IndexName:    "上证指数",
		TriggerValue: 3.52,
		Threshold:    3.0,
		TriggeredAt:  time.Date(2026, 6, 10, 14, 35, 0, 0, time.Local),
	}
	msg := BuildMessage([]AlertEvent{evt})

	if !strings.Contains(msg, "⚠️ 异动告警") {
		t.Error("message missing alert header")
	}
	if !strings.Contains(msg, "上证指数") {
		t.Error("message missing index name")
	}
	if !strings.Contains(msg, "(000001.SH)") {
		t.Error("message missing symbol")
	}
	if !strings.Contains(msg, "上涨异动") {
		t.Error("message missing alert type")
	}
	if !strings.Contains(msg, "+3.52%") {
		t.Error("message missing trigger value")
	}
	if !strings.Contains(msg, "3.00%") {
		t.Error("message missing threshold")
	}
	if !strings.Contains(msg, "2026-06-10 14:35:00") {
		t.Error("message missing trigger time")
	}
}

func TestBuildMessage_MultiEvent(t *testing.T) {
	events := []AlertEvent{
		{ID: "e1", Type: AlertPriceUp, Symbol: "000001.SH", IndexName: "上证指数", TriggerValue: 3.5, Threshold: 3.0, TriggeredAt: time.Now()},
		{ID: "e2", Type: AlertVolumeSpike, Symbol: "399001.SZ", IndexName: "深证成指", TriggerValue: 2.5, Threshold: 2.0, TriggeredAt: time.Now()},
	}
	msg := BuildMessage(events)

	if !strings.Contains(msg, "(2条)") {
		t.Error("multi-event message missing count in header")
	}
	if !strings.Contains(msg, "---") {
		t.Error("multi-event message missing separator")
	}
	if !strings.Contains(msg, "放量异动") {
		t.Error("multi-event message missing second event")
	}
}

// ============================================================
// MockSender 测试
// ============================================================

func TestMockSender_Success(t *testing.T) {
	s := NewMockSender()
	code, err := s.Send(context.Background(), "test content")
	if code != 200 || err != nil {
		t.Errorf("default mock should return 200, nil; got %d, %v", code, err)
	}
	if s.CallCount() != 1 {
		t.Errorf("call count = %d, want 1", s.CallCount())
	}
	if s.Calls()[0].Content != "test content" {
		t.Errorf("recorded content = %q, want %q", s.Calls()[0].Content, "test content")
	}
}

func TestMockSender_Failure(t *testing.T) {
	s := NewMockSender()
	s.SetResponse(500, context.DeadlineExceeded)
	code, err := s.Send(context.Background(), "msg")
	if code != 500 {
		t.Errorf("status = %d, want 500", code)
	}
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ============================================================
// CooldownTracker 测试
// ============================================================

func TestCooldownTracker_NoCooldown(t *testing.T) {
	ct := NewCooldownTracker(0)
	if ct.ShouldSkip("000001.SH", AlertPriceUp) {
		t.Error("cooldown=0 should never skip")
	}
}

func TestCooldownTracker_FirstEventAllowed(t *testing.T) {
	ct := NewCooldownTracker(30)
	if ct.ShouldSkip("000001.SH", AlertPriceUp) {
		t.Error("first event should not be skipped")
	}
}

func TestCooldownTracker_SecondEventSkipped(t *testing.T) {
	ct := NewCooldownTracker(30)
	// 记录第一次
	ct.Record("000001.SH", AlertPriceUp)
	if !ct.ShouldSkip("000001.SH", AlertPriceUp) {
		t.Error("second event within cooldown should be skipped")
	}
}

func TestCooldownTracker_DifferentTypeIndependent(t *testing.T) {
	ct := NewCooldownTracker(30)
	ct.Record("000001.SH", AlertPriceUp)
	// 同一指数，不同类型应独立判断
	if ct.ShouldSkip("000001.SH", AlertVolumeSpike) {
		t.Error("different alert type should not be affected by price_up cooldown")
	}
}

func TestCooldownTracker_DifferentSymbolIndependent(t *testing.T) {
	ct := NewCooldownTracker(30)
	ct.Record("000001.SH", AlertPriceUp)
	if ct.ShouldSkip("399001.SZ", AlertPriceUp) {
		t.Error("different symbol should not be affected")
	}
}

// ============================================================
// HistoryBuffer 测试
// ============================================================

func TestHistoryBuffer_AddAndQuery(t *testing.T) {
	h := NewHistoryBuffer(10)
	h.Add(DeliveryRecord{Status: StatusSuccess, EventIDs: []string{"e1"}})
	h.Add(DeliveryRecord{Status: StatusFailed, EventIDs: []string{"e2"}})

	records := h.Query()
	if len(records) != 2 {
		t.Fatalf("len = %d, want 2", len(records))
	}
	if records[0].Status != StatusSuccess {
		t.Error("first record should be success")
	}
	if records[1].Status != StatusFailed {
		t.Error("second record should be failed")
	}
}

func TestHistoryBuffer_EvictOldest(t *testing.T) {
	h := NewHistoryBuffer(3)
	for i := 1; i <= 5; i++ {
		h.Add(DeliveryRecord{Status: StatusSuccess})
	}
	records := h.Query()
	if len(records) != 3 {
		t.Fatalf("len = %d, want 3 (capacity)", len(records))
	}
}

func TestHistoryBuffer_QueryEmpty(t *testing.T) {
	h := NewHistoryBuffer(10)
	records := h.Query()
	if len(records) != 0 {
		t.Errorf("empty buffer should return [], got len=%d", len(records))
	}
}

// ============================================================
// Dispatcher 端到端测试 (MockSender)
// ============================================================

func TestDispatcher_SendSuccess(t *testing.T) {
	sender := NewMockSender()
	history := NewHistoryBuffer(10)
	d := NewDispatcher(sender, nil, history, 0, 0)

	events := []AlertEvent{
		{ID: "evt-001", Type: AlertPriceUp, Symbol: "000001.SH", IndexName: "上证指数", TriggerValue: 3.5, Threshold: 3.0, TriggeredAt: time.Now()},
	}
	d.Dispatch(context.Background(), events)
	d.Wait() // 等待异步推送完成

	if sender.CallCount() != 1 {
		t.Fatalf("sender call count = %d, want 1", sender.CallCount())
	}

	// 验证推送内容
	call := sender.Calls()[0]
	if !strings.Contains(call.Content, "上证指数") {
		t.Error("push content missing index name")
	}
	if !strings.Contains(call.Content, "上涨异动") {
		t.Error("push content missing alert type")
	}

	// 验证历史记录
	records := history.Query()
	if len(records) != 1 {
		t.Fatalf("history len = %d, want 1", len(records))
	}
	if records[0].Status != StatusSuccess {
		t.Errorf("history status = %s, want success", records[0].Status)
	}
}

func TestDispatcher_EmptyEvents(t *testing.T) {
	sender := NewMockSender()
	d := NewDispatcher(sender, nil, nil, 0, 0)
	d.Dispatch(context.Background(), nil)
	d.Dispatch(context.Background(), []AlertEvent{})
	d.Wait()
	if sender.CallCount() != 0 {
		t.Error("empty events should not trigger send")
	}
}

func TestDispatcher_Cooldown(t *testing.T) {
	sender := NewMockSender()
	cooldown := NewCooldownTracker(30)
	history := NewHistoryBuffer(10)
	d := NewDispatcher(sender, cooldown, history, 0, 0)

	now := time.Now()
	event := AlertEvent{ID: "e1", Type: AlertPriceUp, Symbol: "000001.SH", IndexName: "上证指数", TriggerValue: 3.5, Threshold: 3.0, TriggeredAt: now}

	// 第一次推送：通过
	d.Dispatch(context.Background(), []AlertEvent{event})
	d.Wait()
	if sender.CallCount() != 1 {
		t.Fatalf("first dispatch: call count = %d, want 1", sender.CallCount())
	}

	// 第二次推送（立即）：应被冷却跳过
	d.Dispatch(context.Background(), []AlertEvent{event})
	d.Wait()
	if sender.CallCount() != 1 {
		t.Errorf("second dispatch should be skipped by cooldown, got %d calls", sender.CallCount())
	}

	// 验证历史中有一条 skipped 记录
	records := history.Query()
	hasSkipped := false
	for _, r := range records {
		if r.Status == StatusSkippedCooldown {
			hasSkipped = true
		}
	}
	if !hasSkipped {
		t.Error("history should contain a skipped_cooldown record")
	}
}

func TestDispatcher_RetryOn500(t *testing.T) {
	sender := NewMockSender()
	sender.SetResponse(500, context.DeadlineExceeded)
	d := NewDispatcher(sender, nil, NewHistoryBuffer(10), 2, 0)

	events := []AlertEvent{
		{ID: "e1", Type: AlertPriceUp, Symbol: "000001.SH", IndexName: "上证指数", TriggerValue: 3.5, Threshold: 3.0, TriggeredAt: time.Now()},
	}
	d.Dispatch(context.Background(), events)
	d.Wait()

	// 应重试 3 次（初始 + 2 次重试）
	expectedCalls := 3
	if sender.CallCount() != expectedCalls {
		t.Errorf("retry call count = %d, want %d", sender.CallCount(), expectedCalls)
	}
}

func TestDispatcher_NoRetryOn400(t *testing.T) {
	sender := NewMockSender()
	sender.SetResponse(400, context.DeadlineExceeded)
	d := NewDispatcher(sender, nil, NewHistoryBuffer(10), 2, 0)

	events := []AlertEvent{
		{ID: "e1", Type: AlertPriceUp, Symbol: "000001.SH", IndexName: "上证指数", TriggerValue: 3.5, Threshold: 3.0, TriggeredAt: time.Now()},
	}
	d.Dispatch(context.Background(), events)
	d.Wait()

	// 4xx 不可重试，只调一次
	if sender.CallCount() != 1 {
		t.Errorf("4xx should not retry, got %d calls", sender.CallCount())
	}
}

func TestDispatcher_MultiEventMerge(t *testing.T) {
	sender := NewMockSender()
	d := NewDispatcher(sender, nil, nil, 0, 0)

	events := []AlertEvent{
		{ID: "e1", Type: AlertPriceUp, Symbol: "000001.SH", IndexName: "上证指数", TriggerValue: 3.5, Threshold: 3.0, TriggeredAt: time.Now()},
		{ID: "e2", Type: AlertVolumeSpike, Symbol: "399001.SZ", IndexName: "深证成指", TriggerValue: 2.5, Threshold: 2.0, TriggeredAt: time.Now()},
	}
	d.Dispatch(context.Background(), events)
	d.Wait()

	if sender.CallCount() != 1 {
		t.Fatalf("two events should merge into one send, got %d", sender.CallCount())
	}
	call := sender.Calls()[0]
	if !strings.Contains(call.Content, "(2条)") {
		t.Error("merged message missing count")
	}
	if !strings.Contains(call.Content, "---") {
		t.Error("merged message missing separator")
	}
}

// ============================================================
// Service 测试
// ============================================================

func TestService_Disabled(t *testing.T) {
	sender := NewMockSender()
	d := NewDispatcher(sender, nil, nil, 0, 0)
	s := NewService(d, false)

	s.HandleEvents(context.Background(), []AlertEvent{
		{ID: "e1", Type: AlertPriceUp, Symbol: "000001.SH", IndexName: "上证指数", TriggerValue: 3.5, Threshold: 3.0, TriggeredAt: time.Now()},
	})
	d.Wait()

	if sender.CallCount() != 0 {
		t.Error("disabled service should not send")
	}
}

func TestService_EmptyEvents(t *testing.T) {
	sender := NewMockSender()
	d := NewDispatcher(sender, nil, nil, 0, 0)
	s := NewService(d, true)

	s.HandleEvents(context.Background(), nil)
	s.HandleEvents(context.Background(), []AlertEvent{})
	d.Wait()

	if sender.CallCount() != 0 {
		t.Error("empty events should not send")
	}
}

func TestService_HandleEvents(t *testing.T) {
	sender := NewMockSender()
	history := NewHistoryBuffer(10)
	d := NewDispatcher(sender, nil, history, 0, 0)
	s := NewService(d, true)

	events := []AlertEvent{
		{ID: "evt-001", Type: AlertPriceDown, Symbol: "399006.SZ", IndexName: "创业板指", TriggerValue: -2.5, Threshold: -2.0, TriggeredAt: time.Now()},
	}
	s.HandleEvents(context.Background(), events)
	d.Wait()

	if sender.CallCount() != 1 {
		t.Fatalf("call count = %d, want 1", sender.CallCount())
	}
	if !strings.Contains(sender.Calls()[0].Content, "创业板指") {
		t.Error("message missing index")
	}
	if !strings.Contains(sender.Calls()[0].Content, "下跌异动") {
		t.Error("message missing type")
	}
}

// ============================================================
// WeComSender 集成测试（需要真实 Webhook URL）
// ============================================================

func TestWeComSender_Send(t *testing.T) {
	// 仅在有 WEBHOOK_URL 环境变量时运行
	// webhookURL := os.Getenv("WEBHOOK_URL")
	// if webhookURL == "" {
	// 	t.Skip("WEBHOOK_URL not set, skipping integration test")
	// }
	// sender := NewWeComSender(webhookURL)
	// msg := BuildMessage([]AlertEvent{{
	// 	ID: "test", Type: AlertPriceUp, Symbol: "000001.SH", IndexName: "上证指数",
	// 	TriggerValue: 3.52, Threshold: 3.0, TriggeredAt: time.Now(),
	// }})
	// code, err := sender.Send(context.Background(), msg)
	// if err != nil {
	// 	t.Fatalf("send failed: %v", err)
	// }
	// if code != 200 {
	// 	t.Errorf("status = %d, want 200", code)
	// }
	t.Skip("集成测试需要配置 WEBHOOK_URL 环境变量")
}

// ============================================================
// retriable 逻辑测试
// ============================================================

func TestRetriable(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{500, true},
		{502, true},
		{503, true},
		{429, true},
		{400, false},
		{401, false},
		{404, false},
		{200, false},
	}
	for _, tt := range tests {
		if got := retriable(tt.code); got != tt.want {
			t.Errorf("retriable(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}
