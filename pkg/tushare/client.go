package tushare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL = "http://api.tushare.pro" // 默认 API 地址
	defaultTimeout = 30 * time.Second          // 默认请求超时
)

// Client 是 Tushare Pro HTTP API 的轻量客户端。
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// Option 为 Client 的函数式配置选项。
type Option func(*Client)

// WithBaseURL 覆盖默认的 API 地址。
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(url, "/") }
}

// WithHTTPClient 替换底层 HTTP 客户端。
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.http = hc }
}

// New 创建 Tushare 客户端。
// token 为 Tushare Pro API Token。
func New(token string, opts ...Option) *Client {
	c := &Client{
		baseURL: defaultBaseURL,
		token:   token,
		http: &http.Client{
			Timeout: defaultTimeout,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Do 发送通用 API 请求并返回解析后的 Response。
// 传输/IO 错误返回 error；业务错误（Code != 0）同时返回 error 和 Response。
func (c *Client) Do(ctx context.Context, apiName string, params map[string]any, fields ...string) (*Response, error) {
	body := Request{
		APIName: apiName,
		Token:   c.token,
		Params:  params,
		Fields:  strings.Join(fields, ","),
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("tushare: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("tushare: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tushare: http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tushare: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tushare: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var result Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("tushare: unmarshal response: %w", err)
	}

	if result.Code != 0 {
		return &result, fmt.Errorf("tushare: api error [%d]: %s", result.Code, result.Msg)
	}

	return &result, nil
}

// IndexDaily 查询指数日线 K 线数据。
// tsCode: 指数代码，如 "000001.SH"
// startDate / endDate: 日期范围 YYYYMMDD，空字符串表示不限制
// fields: 可选字段名列表，不传使用默认全部字段
func (c *Client) IndexDaily(ctx context.Context, tsCode string, startDate, endDate string, fields ...string) ([]IndexDailyItem, error) {
	params := map[string]any{"ts_code": tsCode}
	if startDate != "" {
		params["start_date"] = startDate
	}
	if endDate != "" {
		params["end_date"] = endDate
	}

	apiFields := fields
	if len(apiFields) == 0 {
		apiFields = indexDailyFields
	}

	resp, err := c.Do(ctx, "index_daily", params, apiFields...)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil || len(resp.Data.Items) == 0 {
		return nil, nil
	}

	items := make([]IndexDailyItem, 0, len(resp.Data.Items))
	for _, row := range resp.Data.Items {
		items = append(items, parseIndexDailyItem(resp.Data.Fields, row))
	}
	return items, nil
}

// TradeCal 查询交易日历数据。
// exchange: 交易所代码，如 "SSE"（上交所）、"SZSE"（深交所）
// startDate / endDate: 日期范围 YYYYMMDD
func (c *Client) TradeCal(ctx context.Context, exchange, startDate, endDate string) ([]TradeCalItem, error) {
	params := map[string]any{
		"exchange":   exchange,
		"start_date": startDate,
		"end_date":   endDate,
	}

	resp, err := c.Do(ctx, "trade_cal", params, tradeCalFields...)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil || len(resp.Data.Items) == 0 {
		return nil, nil
	}

	items := make([]TradeCalItem, 0, len(resp.Data.Items))
	for _, row := range resp.Data.Items {
		items = append(items, parseTradeCalItem(resp.Data.Fields, row))
	}
	return items, nil
}
