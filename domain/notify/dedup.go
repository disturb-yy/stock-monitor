package notify

import (
	"sync"
	"time"
)

// CooldownTracker 追踪每个 (指数, 异动类型) 组合的最近推送时间。
// 用于推送前的冷却去重判断。
type CooldownTracker struct {
	mu              sync.Mutex
	records         map[string]time.Time
	cooldownMinutes int
	stopCh          chan struct{}
}

// NewCooldownTracker 创建冷却追踪器。
// cooldownMinutes 为冷却时间（分钟），0 表示禁用冷却。
func NewCooldownTracker(cooldownMinutes int) *CooldownTracker {
	return &CooldownTracker{
		records:         make(map[string]time.Time),
		cooldownMinutes: cooldownMinutes,
		stopCh:          make(chan struct{}),
	}
}

// ShouldSkip 判断该事件是否应被冷却跳过。
// key 格式为 "symbol:alert_type"。
func (c *CooldownTracker) ShouldSkip(symbol string, at AlertType) bool {
	if c == nil || c.cooldownMinutes <= 0 {
		return false
	}
	key := symbol + ":" + string(at)
	c.mu.Lock()
	defer c.mu.Unlock()
	lastTime, exists := c.records[key]
	if !exists {
		return false
	}
	return time.Since(lastTime) < time.Duration(c.cooldownMinutes)*time.Minute
}

// Record 记录一次成功推送。
func (c *CooldownTracker) Record(symbol string, at AlertType) {
	if c == nil || c.cooldownMinutes <= 0 {
		return
	}
	key := symbol + ":" + string(at)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records[key] = time.Now()
}

// StartCleanup 启动定时清理过期记录的 goroutine。
// interval 为清理间隔。
func (c *CooldownTracker) StartCleanup(interval time.Duration) {
	if c == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.cleanup()
			case <-c.stopCh:
				return
			}
		}
	}()
}

// Stop 停止清理 goroutine。
func (c *CooldownTracker) Stop() {
	if c == nil {
		return
	}
	close(c.stopCh)
}

// cleanup 删除超过冷却时间 2 倍的过期记录。
func (c *CooldownTracker) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	threshold := time.Duration(c.cooldownMinutes*2) * time.Minute
	now := time.Now()
	for key, t := range c.records {
		if now.Sub(t) > threshold {
			delete(c.records, key)
		}
	}
}
