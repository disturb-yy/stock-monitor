package notify

import (
	"context"
	"sync"
)

// MockCall 记录 MockSender 的每次调用。
type MockCall struct {
	Content    string // 发送的消息内容
	StatusCode int    // 模拟返回的 HTTP 状态码
	Error      error  // 模拟返回的错误
}

// MockSender 是 WebhookSender 的测试双，记录每次调用而非真实发送。
// 可通过 SetResponse 预设下一次调用的返回值。
type MockSender struct {
	mu       sync.Mutex
	calls    []MockCall
	respCode int
	respErr  error
}

// NewMockSender 创建 MockSender，默认返回 200 成功。
func NewMockSender() *MockSender {
	return &MockSender{respCode: 200}
}

// SetResponse 设置下一次 Send 调用的模拟返回值。
func (m *MockSender) SetResponse(statusCode int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.respCode = statusCode
	m.respErr = err
}

// Send 记录调用并返回预设的模拟响应。
func (m *MockSender) Send(_ context.Context, content string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	call := MockCall{Content: content, StatusCode: m.respCode, Error: m.respErr}
	m.calls = append(m.calls, call)
	return m.respCode, m.respErr
}

// Calls 返回所有已记录的调用列表。
func (m *MockSender) Calls() []MockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	cpy := make([]MockCall, len(m.calls))
	copy(cpy, m.calls)
	return cpy
}

// CallCount 返回已记录的调用次数。
func (m *MockSender) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}
