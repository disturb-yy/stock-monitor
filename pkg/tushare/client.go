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
	defaultBaseURL = "http://api.tushare.pro"
	defaultTimeout = 30 * time.Second
)

// Client is a lightweight Tushare Pro HTTP API client.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the default Tushare API endpoint.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(url, "/") }
}

// WithHTTPClient replaces the underlying HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.http = hc }
}

// New creates a new Tushare client.
//
//	token is your Tushare Pro API token.
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

// Do send a generic API request and returns the parsed Response.
// An error is returned only for transport / I/O failures; business errors
// (non-zero Code) are surfaced inside the Response object.
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

// IndexDaily returns daily K-line data for stock indices.
//
//	tsCode:  index code, e.g. "000001.SH", "399001.SZ"
//	start / end date: optional, format "YYYYMMDD". Pass empty strings to omit.
//	fields: optional field names to limit the response columns.
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

// toFloat64 coerces a json.Number / float64 / int to float64.
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}
