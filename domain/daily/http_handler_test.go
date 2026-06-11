package daily

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter(h *HTTPHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/market/daily-summary", h.GetSummary)
	r.POST("/api/market/daily-summary", h.PostSummary)
	return r
}

func TestGetSummary_Success(t *testing.T) {
	p := NewMockDataProvider()
	b := NewBuilder(p, cstLocation)
	h := NewHTTPHandler(b, nil, cstLocation)
	router := setupTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/market/daily-summary", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	if resp.Data["is_trading_day"] != true {
		t.Error("should be trading day")
	}
	if up, ok := resp.Data["up_count"].(float64); !ok || up != 3 {
		t.Errorf("up_count = %v, want 3", resp.Data["up_count"])
	}
	if resp.Data["trade_date"] == "" || resp.Data["trade_date"] == nil {
		t.Error("trade_date missing")
	}
	anomaly, _ := resp.Data["anomaly_summary"].(map[string]any)
	if anomaly == nil {
		t.Error("anomaly_summary missing")
	}
}

func TestGetSummary_NonTradingDay(t *testing.T) {
	p := NewMockDataProvider()
	p.IsTrading = false
	b := NewBuilder(p, cstLocation)
	h := NewHTTPHandler(b, nil, cstLocation)
	router := setupTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/market/daily-summary", nil)
	router.ServeHTTP(w, req)

	var resp struct {
		Data struct {
			IsTradingDay bool `json:"is_trading_day"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Data.IsTradingDay {
		t.Error("should not be trading day")
	}
}

func TestPostSummary_Success(t *testing.T) {
	p := NewMockDataProvider()
	b := NewBuilder(p, cstLocation)

	push := func(s DailySummary) error {
		return nil
	}

	h := NewHTTPHandler(b, push, cstLocation)
	router := setupTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/market/daily-summary", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp struct {
		Msg  string `json:"msg"`
		Data struct {
			Triggered bool `json:"triggered"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.Data.Triggered {
		t.Error("should be triggered")
	}
}

func TestPostSummary_NonTradingDay(t *testing.T) {
	p := NewMockDataProvider()
	p.IsTrading = false
	b := NewBuilder(p, cstLocation)

	h := NewHTTPHandler(b, nil, cstLocation)
	router := setupTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/market/daily-summary", nil)
	router.ServeHTTP(w, req)

	var resp struct {
		Data struct {
			Triggered bool `json:"triggered"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Data.Triggered {
		t.Error("should not trigger on non-trading day")
	}
}

func TestPostSummary_WebhookDisabled(t *testing.T) {
	p := NewMockDataProvider()
	b := NewBuilder(p, cstLocation)

	// push 为 nil 表示 Webhook 未启用
	h := NewHTTPHandler(b, nil, cstLocation)
	router := setupTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/market/daily-summary", nil)
	router.ServeHTTP(w, req)

	// Webhook 未启用时应返回 503
	if w.Code < 400 {
		t.Errorf("status = %d, want error for disabled webhook", w.Code)
	}
}
