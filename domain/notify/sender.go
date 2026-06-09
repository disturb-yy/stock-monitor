// Package notify 提供 Webhook 告警推送功能。
// 将异动检测引擎产生的异动事件通过企业微信机器人 Webhook 推送到指定群聊。
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// DeliveryStatus 表示单次推送的最终状态。
type DeliveryStatus string

const (
	StatusSuccess         DeliveryStatus = "success"          // 推送成功
	StatusFailed          DeliveryStatus = "failed"           // 最终失败（重试耗尽或不可重试）
	StatusSkippedCooldown DeliveryStatus = "skipped_cooldown" // 被冷却机制跳过
	StatusRetrying        DeliveryStatus = "retrying"         // 重试中（中间状态）
)

// WebhookSender 是 Webhook 消息发送的抽象接口。
// 提供企业微信真实发送器和 Mock 测试发送器两种实现。
type WebhookSender interface {
	// Send 发送 Markdown 消息到 Webhook 端点。
	// content 为已构建好的 Markdown 文本（不含外层 JSON 包装）。
	// 返回 HTTP 状态码和可能的错误。
	Send(ctx context.Context, content string) (statusCode int, err error)
}

// wecomMsg 是企业微信机器人 Markdown 消息的 JSON 结构。
type wecomMsg struct {
	MsgType  string         `json:"msgtype"`
	Markdown wecomMdContent `json:"markdown"`
}

// wecomMdContent 是企业微信 Markdown 消息内容。
type wecomMdContent struct {
	Content string `json:"content"`
}

// wecomResp 是企业微信 Webhook API 响应。
type wecomResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// WeComSender 是企业微信机器人 Webhook 发送器。
type WeComSender struct {
	webhookURL string
	httpClient *http.Client
}

// NewWeComSender 创建企业微信 Webhook 发送器。
func NewWeComSender(webhookURL string) *WeComSender {
	return &WeComSender{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send 将 Markdown 内容包装为企业微信消息格式并 POST 到 Webhook URL。
// 返回 HTTP 状态码和可能的错误。
func (s *WeComSender) Send(ctx context.Context, content string) (int, error) {
	msg := wecomMsg{
		MsgType: "markdown",
		Markdown: wecomMdContent{
			Content: content,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("序列化企业微信消息失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	slog.Debug("notify: 发送企业微信消息", "content_len", len(content))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("企业微信返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var wr wecomResp
	if err := json.Unmarshal(respBody, &wr); err != nil {
		return resp.StatusCode, fmt.Errorf("解析企业微信响应失败: %w", err)
	}

	if wr.ErrCode != 0 {
		return resp.StatusCode, fmt.Errorf("企业微信 API 错误 (errcode=%d): %s", wr.ErrCode, wr.ErrMsg)
	}

	slog.Info("notify: 企业微信消息发送成功", "errcode", wr.ErrCode)
	return resp.StatusCode, nil
}
