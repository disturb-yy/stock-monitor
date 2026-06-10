package daily

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// cstLocation 为 A 股时区，测试复用。
var cstLocation = time.FixedZone("CST", 8*60*60)

// ============================================================
// Builder 测试
// ============================================================

func TestBuilder_BuildSummary_Success(t *testing.T) {
	p := NewMockDataProvider()
	b := NewBuilder(p, cstLocation)

	s, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if !s.IsTradingDay {
		t.Error("should be trading day")
	}
	if s.UpCount != 3 {
		t.Errorf("UpCount = %d, want 3", s.UpCount)
	}
	if s.DownCount != 3 {
		t.Errorf("DownCount = %d, want 3", s.DownCount)
	}
	if s.TotalIndices != 6 {
		t.Errorf("TotalIndices = %d, want 6", s.TotalIndices)
	}
	if s.TopGainer == nil {
		t.Error("TopGainer should not be nil")
	}
	if s.TopLoser == nil {
		t.Error("TopLoser should not be nil")
	}
	if s.AnomalySummary.TotalCount != 2 {
		t.Errorf("AnomalySummary.TotalCount = %d, want 2", s.AnomalySummary.TotalCount)
	}
	if s.DataStatus != DataStatusOK {
		t.Errorf("DataStatus = %s, want ok", s.DataStatus)
	}
	if s.TradeDate == "" {
		t.Error("TradeDate should not be empty")
	}
}

func TestBuilder_BuildSummary_NonTradingDay(t *testing.T) {
	p := NewMockDataProvider()
	p.IsTrading = false
	b := NewBuilder(p, cstLocation)

	s, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if s.IsTradingDay {
		t.Error("should not be trading day")
	}
	if s.UpCount != 0 {
		t.Error("non-trading day should have 0 up")
	}
}

func TestBuilder_BuildSummary_PartialData(t *testing.T) {
	p := NewMockDataProvider()
	p.AnomalyErr = fmt.Errorf("anomaly service unavailable")
	b := NewBuilder(p, cstLocation)

	s, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if s.DataStatus != DataStatusPartial {
		t.Errorf("DataStatus = %s, want partial", s.DataStatus)
	}
	if s.AnomalySummary.TotalCount != 0 {
		t.Error("anomaly total should be 0 on error")
	}
}

func TestBuilder_BuildSummary_NoData(t *testing.T) {
	p := NewMockDataProvider()
	p.QuotesErr = fmt.Errorf("data source unavailable")
	b := NewBuilder(p, cstLocation)

	s, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if s.DataStatus != DataStatusFailed {
		t.Errorf("DataStatus = %s, want failed", s.DataStatus)
	}
}

func TestBuilder_BuildAndPush_TradingDay(t *testing.T) {
	p := NewMockDataProvider()
	b := NewBuilder(p, cstLocation)

	pushed := false
	push := func(s DailySummary) error {
		pushed = true
		if s.UpCount != 3 {
			return fmt.Errorf("unexpected up count")
		}
		return nil
	}

	err := b.BuildAndPush(context.Background(), push)
	if err != nil {
		t.Fatalf("BuildAndPush failed: %v", err)
	}
	if !pushed {
		t.Error("push should have been called")
	}
}

func TestBuilder_BuildAndPush_NonTradingDay(t *testing.T) {
	p := NewMockDataProvider()
	p.IsTrading = false
	b := NewBuilder(p, cstLocation)

	pushed := false
	err := b.BuildAndPush(context.Background(), func(s DailySummary) error {
		pushed = true
		return nil
	})

	if err != ErrNotTradingDay {
		t.Errorf("expected ErrNotTradingDay, got %v", err)
	}
	if pushed {
		t.Error("should not push on non-trading day")
	}
}

func TestBuilder_TopGainerLoser(t *testing.T) {
	p := NewMockDataProvider()
	b := NewBuilder(p, cstLocation)

	s, _ := b.Build(context.Background())

	// 上证指数 +0.39% 应领涨，创业板指 -0.54% 应领跌
	if s.TopGainer == nil || s.TopGainer.Symbol != "000001.SH" {
		t.Errorf("TopGainer should be 000001.SH (+0.39%%), got %v", s.TopGainer)
	}
	if s.TopLoser == nil || s.TopLoser.Symbol != "399006.SZ" {
		t.Errorf("TopLoser should be 399006.SZ (-0.54%%), got %v", s.TopLoser)
	}
}

// ============================================================
// Markdown 测试
// ============================================================

func TestFormatMarkdown_Full(t *testing.T) {
	s := &DailySummary{
		TradeDate:    "2026-06-11",
		IsTradingDay: true,
		UpCount:      3,
		DownCount:    2,
		FlatCount:    0,
		TotalIndices: 6,
		TopGainer:    &IndexSnapshot{Symbol: "000001.SH", Name: "上证指数", ChangePercent: 0.39},
		TopLoser:     &IndexSnapshot{Symbol: "399006.SZ", Name: "创业板指", ChangePercent: -0.54},
		Indices: []IndexSnapshot{
			{Symbol: "000001.SH", Name: "上证指数", ChangePercent: 0.39},
			{Symbol: "399006.SZ", Name: "创业板指", ChangePercent: -0.54},
		},
		AnomalySummary: AnomalySummary{
			TotalCount: 2,
			Items: []AnomalyItem{
				{Symbol: "000001.SH", IndexName: "上证指数", Type: "price_up", TypeName: "上涨异动", TriggerValue: 3.52},
				{Symbol: "399006.SZ", IndexName: "创业板指", Type: "price_down", TypeName: "下跌异动", TriggerValue: -2.54},
			},
		},
		DataStatus:  DataStatusOK,
		GeneratedAt: time.Date(2026, 6, 11, 15, 15, 0, 0, cstLocation),
	}

	md := FormatMarkdown(s)

	checks := []string{
		"📊 收盘总结",
		"2026-06-11",
		"上证指数 +0.39%",
		"创业板指 -0.54%",
		"上涨 3 只，下跌 2 只",
		"领涨：上证指数 (+0.39%)",
		"领跌：创业板指 (-0.54%)",
		"共 2 条",
		"上涨异动",
		"下跌异动",
		"stock-monitor 自动推送",
	}

	for _, c := range checks {
		if !strings.Contains(md, c) {
			t.Errorf("markdown missing: %q", c)
		}
	}
}

func TestFormatMarkdown_NoAnomaly(t *testing.T) {
	s := &DailySummary{
		TradeDate:    "2026-06-11",
		IsTradingDay: true,
		UpCount:      4,
		DownCount:    2,
		TotalIndices: 6,
		Indices: []IndexSnapshot{
			{Symbol: "000001.SH", Name: "上证指数", ChangePercent: 0.39},
		},
		DataStatus:  DataStatusOK,
		GeneratedAt: time.Now(),
	}

	md := FormatMarkdown(s)
	if !strings.Contains(md, "今日无异常") {
		t.Error("should say 今日无异常 when no anomalies")
	}
}

func TestFormatMarkdown_PartialData(t *testing.T) {
	s := &DailySummary{
		TradeDate:    "2026-06-11",
		IsTradingDay: true,
		UpCount:      2,
		DownCount:    1,
		TotalIndices: 6,
		Indices: []IndexSnapshot{
			{Symbol: "000001.SH", Name: "上证指数", ChangePercent: 0.39},
		},
		DataStatus:  DataStatusPartial,
		GeneratedAt: time.Now(),
	}

	md := FormatMarkdown(s)
	if !strings.Contains(md, "数据异常") {
		t.Error("should have data anomaly warning")
	}
}

func TestFormatMarkdown_FailedData(t *testing.T) {
	s := &DailySummary{
		TradeDate:    "2026-06-11",
		IsTradingDay: true,
		DataStatus:   DataStatusFailed,
		GeneratedAt:  time.Now(),
	}

	md := FormatMarkdown(s)
	if !strings.Contains(md, "数据获取完全失败") {
		t.Error("should indicate complete failure")
	}
}
