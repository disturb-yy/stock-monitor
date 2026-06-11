package daily

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestScheduler_TradingDay(t *testing.T) {
	p := NewMockDataProvider()
	b := NewBuilder(p, cstLocation)

	var mu sync.Mutex
	pushed := false
	push := func(s DailySummary) error {
		mu.Lock()
		pushed = true
		mu.Unlock()
		return nil
	}

	// 设置推送时间为当前时间，立即 tick 触发
	now := time.Now().In(cstLocation)
	pushTime := now.Format("15:04")

	sched := NewScheduler(b, push, pushTime, cstLocation)
	sched.Start()
	sched.Stop() // Stop 会等待当前推送完成（包括异步 PushFunc）

	mu.Lock()
	defer mu.Unlock()
	if !pushed {
		t.Error("should have pushed on trading day")
	}
}

func TestScheduler_NonTradingDay(t *testing.T) {
	p := NewMockDataProvider()
	p.IsTrading = false
	b := NewBuilder(p, cstLocation)

	pushed := false
	push := func(s DailySummary) error {
		pushed = true
		return nil
	}

	now := time.Now().In(cstLocation)
	sched := NewScheduler(b, push, now.Format("15:04"), cstLocation)
	sched.Start()

	time.Sleep(200 * time.Millisecond)
	sched.Stop()

	if pushed {
		t.Error("should not push on non-trading day")
	}
}

func TestScheduler_PushFailure(t *testing.T) {
	p := NewMockDataProvider()
	b := NewBuilder(p, cstLocation)

	// push 返回错误，但不应影响调度器
	push := func(s DailySummary) error {
		return errors.New("push failed") // 任意的 error
	}

	now := time.Now().In(cstLocation)
	sched := NewScheduler(b, push, now.Format("15:04"), cstLocation)
	sched.Start()

	time.Sleep(200 * time.Millisecond)
	// 调度器不应该 panic 或阻塞
	sched.Stop()
}

func TestScheduler_DifferentTime(t *testing.T) {
	p := NewMockDataProvider()
	b := NewBuilder(p, cstLocation)

	pushed := false
	push := func(s DailySummary) error {
		pushed = true
		return nil
	}

	// 设置为一个不会匹配的时间
	sched := NewScheduler(b, push, "99:99", cstLocation)
	sched.Start()

	time.Sleep(200 * time.Millisecond)
	sched.Stop()

	if pushed {
		t.Error("should not push when time doesn't match")
	}
}
