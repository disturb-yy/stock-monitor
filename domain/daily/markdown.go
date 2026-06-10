package daily

import (
	"fmt"
	"strings"
	"time"
)

// FormatMarkdown 将 DailySummary 格式化为企业微信 Markdown 消息文本。
// 格式符合 contracts/daily-api.md 中定义的推送格式。
func FormatMarkdown(s *DailySummary) string {
	var b strings.Builder

	// 标题
	b.WriteString("## 📊 收盘总结\n\n")

	// 交易日期
	b.WriteString(fmt.Sprintf("**交易日期**：%s\n\n", s.TradeDate))

	// 数据异常标注
	switch s.DataStatus {
	case DataStatusPartial:
		b.WriteString("⚠️ **数据异常：部分数据获取失败**\n\n")
	case DataStatusFailed:
		b.WriteString("⚠️ **数据异常：数据获取完全失败**\n\n")
		b.WriteString(fmt.Sprintf("---\n*stock-monitor 自动推送 · %s · ⚠️ 数据异常*\n",
			s.GeneratedAt.Format(time.DateTime)))
		return b.String()
	}

	// 指数表现
	b.WriteString("**指数表现**：\n")
	names := make([]string, 0, len(s.Indices))
	missingCount := 0
	for _, idx := range s.Indices {
		if idx.ChangePercent == 0 && idx.Price == 0 {
			names = append(names, fmt.Sprintf("%s 数据缺失", idx.Name))
			missingCount++
		} else {
			names = append(names, fmt.Sprintf("%s %+.2f%%", idx.Name, idx.ChangePercent))
		}
	}
	b.WriteString(strings.Join(names, "、"))
	b.WriteString("\n\n")

	// 涨跌统计
	b.WriteString(fmt.Sprintf("> 上涨 %d 只，下跌 %d 只", s.UpCount, s.DownCount))
	if s.FlatCount > 0 {
		b.WriteString(fmt.Sprintf("，持平 %d 只", s.FlatCount))
	}
	b.WriteString("\n")

	// 领涨/领跌
	if s.TopGainer != nil {
		b.WriteString(fmt.Sprintf("> 领涨：%s (%+.2f%%)\n", s.TopGainer.Name, s.TopGainer.ChangePercent))
	}
	if s.TopLoser != nil {
		b.WriteString(fmt.Sprintf("> 领跌：%s (%+.2f%%)\n", s.TopLoser.Name, s.TopLoser.ChangePercent))
	}

	// 异动事件摘要
	b.WriteString("\n**异动事件**")
	if s.AnomalySummary.TotalCount == 0 {
		b.WriteString("：今日无异常\n")
	} else {
		b.WriteString(fmt.Sprintf("（共 %d 条）：\n", s.AnomalySummary.TotalCount))
		for _, item := range s.AnomalySummary.Items {
			b.WriteString(fmt.Sprintf("- %s %s (%+.2f%%)\n", item.IndexName, item.TypeName, item.TriggerValue))
		}
		if s.AnomalySummary.TotalCount > 5 {
			b.WriteString(fmt.Sprintf("- 等 %d 条\n", s.AnomalySummary.TotalCount-5))
		}
	}

	// 页脚
	footer := fmt.Sprintf("---\n*stock-monitor 自动推送 · %s*", s.GeneratedAt.Format(time.DateTime))
	if s.DataStatus == DataStatusPartial {
		footer += " · ⚠️ 数据异常"
	}
	b.WriteString(footer)

	return b.String()
}
