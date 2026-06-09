package notify

import (
	"sync"
	"time"
)

// DeliveryRecord 表示一条推送记录。
type DeliveryRecord struct {
	ID          string         `json:"id"`           // 记录唯一标识（UUID）
	EventIDs    []string       `json:"event_ids"`    // 关联的异动事件 ID 列表
	Status      DeliveryStatus `json:"status"`       // 推送状态
	Message     string         `json:"message"`      // 消息内容摘要（截断）
	Error       string         `json:"error"`        // 失败时的错误信息
	RetryCount  int            `json:"retry_count"`  // 已重试次数
	DeliveredAt time.Time      `json:"delivered_at"` // 首次推送时间
	CompletedAt time.Time      `json:"completed_at"` // 最终完成时间
}

// HistoryBuffer 是推送历史的环形缓冲区（固定容量，线程安全）。
type HistoryBuffer struct {
	mu       sync.RWMutex
	buf      []DeliveryRecord
	capacity int
	writeIdx int
	count    int
}

// NewHistoryBuffer 创建固定容量的推送历史缓冲区。
func NewHistoryBuffer(capacity int) *HistoryBuffer {
	return &HistoryBuffer{
		buf:      make([]DeliveryRecord, capacity),
		capacity: capacity,
	}
}

// Add 写入一条推送记录。超出容量时覆盖最旧记录。
func (h *HistoryBuffer) Add(record DeliveryRecord) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.buf[h.writeIdx] = record
	h.writeIdx = (h.writeIdx + 1) % h.capacity
	if h.count < h.capacity {
		h.count++
	}
}

// Query 返回所有推送记录的副本，按时间降序排列（最新在前）。
func (h *HistoryBuffer) Query() []DeliveryRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.count == 0 {
		return nil
	}

	result := make([]DeliveryRecord, h.count)
	// 从最新写入位置反向遍历
	idx := (h.writeIdx - 1 + h.capacity) % h.capacity
	for i := 0; i < h.count; i++ {
		result[i] = h.buf[idx]
		idx = (idx - 1 + h.capacity) % h.capacity
	}
	return result
}

// Len 返回已存储的记录数。
func (h *HistoryBuffer) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.count
}
