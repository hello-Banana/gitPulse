package printer

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gitpulse/types"
	"github.com/fatih/color"
)

// formatTime 格式化时间 (将 9.5 转为 09:30)
func formatTime(hourFloat float64) string {
	hour := int(hourFloat)
	minute := int((hourFloat - float64(hour)) * 60)
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

// getLevelColor 根据指数获取颜色
func getLevelColor(index int) *color.Color {
	if index <= 21 {
		return color.New(color.FgGreen)
	} else if index <= 48 {
		return color.New(color.FgHiGreen)
	} else if index <= 63 {
		return color.New(color.FgYellow)
	} else if index <= 100 {
		return color.New(color.FgRed)
	}
	return color.New(color.FgHiRed)
}

// getLevelEmoji 根据指数获取表情
func getLevelEmoji(index int) string {
	if index <= 21 {
		return "✓"
	} else if index <= 48 {
		return "○"
	} else if index <= 63 {
		return "△"
	}
	return "×"
}

// PrintReport 打印完整报告
func PrintReport(result types.AnalysisResult) {
	fmt.Println()

	// 顶部标题
	fmt.Println(color.CyanString("🔍 分析仓库：%s", result.RepoPath))
	fmt.Printf("📅 时间范围：%s 至 %s（按最后提交回溯 365 天）\n\n",
		result.CommitData.Since, result.CommitData.Until)

	// 核心结果表格
	printCoreResult(result)
	fmt.Println()

	// 详细分析
	printDetailedAnalysis(result)
	fmt.Println()

	// 工作时间推测
	printWorkTimeDetect(result)
	fmt.Println()

	// 24 小时分布
	print24HourDistribution(result)
	fmt.Println()

	// 星期分布
	printWeekdayDistribution(result)
	fmt.Println()

	// 工作日加班分布
	printWeekdayOvertime(result)
	fmt.Println()

	// 周末加班分析
	printWeekendOvertime(result)
	fmt.Println()

	// 深夜加班分析
	printLateNightOvertime(result)
	fmt.Println()

	// 疲劳度和节奏
	printAdvancedAnalysis(result)
	fmt.Println()

	// 底部提示
	printFooter()
}

// printCoreResult 打印核心结果
func printCoreResult(result types.AnalysisResult) {
	index := result.WorkIntensity.Index
	indexStr := result.WorkIntensity.IndexStr
	ratio := result.WorkIntensity.OverTimeRadio
	total := result.CommitData.TotalCommits

	fmt.Println(color.CyanString(strings.Repeat("─", 78)))
	fmt.Println()

	// 指数框
	fmt.Printf("╔══════════════════╤═════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║ %-16s │ %-57s ║\n", "996 指数", fmt.Sprintf("%.1f", float64(index)/10*10))
	fmt.Printf("╟──────────────────┼─────────────────────────────────────────────────────────╢\n")
	fmt.Printf("║ %-16s │ %-57s ║\n", "整体评价", indexStr)
	fmt.Printf("╟──────────────────┼─────────────────────────────────────────────────────────╢\n")
	fmt.Printf("║ %-16s │ %-57s ║\n", "分析时段", fmt.Sprintf("%s 至 %s", result.CommitData.Since, result.CommitData.Until))
	fmt.Printf("╟──────────────────┼─────────────────────────────────────────────────────────╢\n")
	fmt.Printf("║ %-16s │ %-57s ║\n", "加班比例", fmt.Sprintf("%.1f%%", float64(ratio)))
	fmt.Printf("╟──────────────────┼─────────────────────────────────────────────────────────╢\n")
	fmt.Printf("║ %-16s │ %-57s ║\n", "总提交数", fmt.Sprintf("%d", total))
	fmt.Printf("╚══════════════════╧═════════════════════════════════════════════════════════╝\n")
	fmt.Println()
	fmt.Println("  * 996 指数：为 0 则不加班，值越大代表加班越严重，996 工作制对应的值为 100。")
}

// printDetailedAnalysis 打印详细分析
func printDetailedAnalysis(result types.AnalysisResult) {
	index := result.WorkIntensity.Index
	ratio := result.WorkIntensity.OverTimeRadio
	weekday := result.WeekdayOvertime
	lateNight := result.LateNight

	fmt.Println(color.CyanString("📋 详细分析:"))
	fmt.Println()

	// 评价
	var evalEmoji, evalText string
	if index <= 21 {
		evalEmoji, evalText = "🌟", "很好，几乎不加班"
	} else if index <= 48 {
		evalEmoji, evalText = "✓", "不错，加班较少"
	} else if index <= 63 {
		evalEmoji, evalText = "△", "还行，偶尔加班"
	} else if index <= 100 {
		evalEmoji, evalText = "⚠️", "较差，加班较多"
	} else {
		evalEmoji, evalText = "🚨", "很差，接近 996 的程度"
	}

	fmt.Printf("  %s  %s（加班比例 %.1f%%）\n", evalEmoji, evalText, float64(ratio))

	// 工作日加班评价
	totalWeekday := weekday.Monday + weekday.Tuesday + weekday.Wednesday + weekday.Thursday + weekday.Friday
	if totalWeekday > 0 {
		peakDay := weekday.PeakDay
		peakCount := weekday.PeakCount
		fmt.Printf("  ⚠️  工作日加班频繁，%s是加班高峰（%d次提交）\n", peakDay, peakCount)
	}

	// 周末加班评价
	weekend := result.WeekendOvertime
	if weekend.RealOvertimeDays > 5 {
		fmt.Printf("  ⚠️  周末加班严重（%d天真正加班），工作侵占休息时间\n", weekend.RealOvertimeDays)
	}

	// 深夜加班评价
	if lateNight.MidnightDays > 0 {
		fmt.Printf("  🌃 存在深夜加班情况（%d天），需注意休息\n", lateNight.MidnightDays)
	}
}

// printWorkTimeDetect 打印工作时间推测
func printWorkTimeDetect(result types.AnalysisResult) {
	detect := result.WorkTimeDetect
	workHours := detect.EndHour - detect.StartHour

	fmt.Println(color.CyanString("⌛ 工作时间推测: （自动推断）"))
	fmt.Println()

	fmt.Println("╔══════════════════╤═════════════════════════════════════════════════════════╗")
	fmt.Printf("║ %-16s │ %-57s ║\n", "上班时间", fmt.Sprintf("%s（推测上班区间：%s-%s）",
		formatTime(detect.StartHour),
		formatTime(detect.StartHour),
		formatTime(detect.StartHour+0.5)))
	fmt.Println("╟──────────────────┼─────────────────────────────────────────────────────────╢")
	fmt.Printf("║ %-16s │ %-57s ║\n", "下班时间", fmt.Sprintf("%s（推测下班区间：%s-%s）",
		formatTime(detect.EndHour),
		formatTime(detect.EndHour),
		formatTime(detect.EndHour+1)))
	fmt.Println("╟──────────────────┼─────────────────────────────────────────────────────────╢")
	fmt.Printf("║ %-16s │ %-57s ║\n", "置信度", fmt.Sprintf("%.0f%%（样本天数：%d）",
		map[string]float64{"high": 90, "medium": 74, "low": 50}[detect.Confidence]*100/90,
		result.CommitData.TotalCommits/30))
	fmt.Println("╚══════════════════╧═════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("⚠️  加班判定说明：推测的平均工作时长约为 %.1f 小时，指数计算仅将前 9 小时视为正常工时，超出时段已按加班统计。\n", workHours)
}

// print24HourDistribution 打印 24 小时分布
func print24HourDistribution(result types.AnalysisResult) {
	fmt.Println(color.CyanString("🕐 24 小时分布:"))
	fmt.Println()

	// 找出最大值用于比例计算
	maxCount := 0
	for _, tc := range result.CommitData.HourData {
		if tc.Count > maxCount {
			maxCount = tc.Count
		}
	}

	// 只打印有数据的时段
	for i, tc := range result.CommitData.HourData {
		if tc.Count > 0 {
			barLen := tc.Count * 20 / maxCount
			if barLen < 1 {
				barLen = 1
			}
			bar := strings.Repeat("█", barLen)
			fmt.Printf("%02d: %-20s %5d\n", i, bar, tc.Count)
		}
	}
}

// printWeekdayDistribution 打印星期分布
func printWeekdayDistribution(result types.AnalysisResult) {
	fmt.Println()
	fmt.Println(color.CyanString("📅 星期分布:"))
	fmt.Println()

	weekdayNames := map[int]string{
		1: "周一", 2: "周二", 3: "周三", 4: "周四", 5: "周五", 6: "周六", 7: "周日",
	}

	// 找出最大值
	maxCount := 0
	for _, count := range result.CommitData.WeekdayStats {
		if count > maxCount {
			maxCount = count
		}
	}

	total := result.CommitData.TotalCommits

	for i := 1; i <= 7; i++ {
		count := result.CommitData.WeekdayStats[i]
		if count > 0 {
			barLen := count * 20 / maxCount
			if barLen < 1 {
				barLen = 1
			}
			bar := strings.Repeat("█", barLen)
			pct := float64(count) * 100 / float64(total)
			fmt.Printf("%s: %-20s %5d (%.1f%%)\n", weekdayNames[i], bar, count, pct)
		}
	}
}

// printWeekdayOvertime 打印工作日加班分布
func printWeekdayOvertime(result types.AnalysisResult) {
	fmt.Println()
	fmt.Println(color.CyanString("💼 工作日加班分布:"))
	fmt.Println()

	weekdayNames := map[int]string{
		1: "周一", 2: "周二", 3: "周三", 4: "周四", 5: "周五",
	}
	counts := map[string]int{
		"周一": result.WeekdayOvertime.Monday,
		"周二": result.WeekdayOvertime.Tuesday,
		"周三": result.WeekdayOvertime.Wednesday,
		"周四": result.WeekdayOvertime.Thursday,
		"周五": result.WeekdayOvertime.Friday,
	}

	maxCount := 0
	for _, count := range counts {
		if count > maxCount {
			maxCount = count
		}
	}

	for i := 1; i <= 5; i++ {
		name := weekdayNames[i]
		count := counts[name]
		if maxCount > 0 {
			barLen := count * 20 / maxCount
			if barLen < 1 && count > 0 {
				barLen = 1
			}
			bar := strings.Repeat("█", barLen)
			mark := ""
			if name == result.WeekdayOvertime.PeakDay {
				mark = " ⚠️ 加班高峰"
			}
			fmt.Printf("%s: %-20s %3d次%s\n", name, bar, count, mark)
		}
	}
}

// printWeekendOvertime 打印周末加班分析
func printWeekendOvertime(result types.AnalysisResult) {
	fmt.Println()
	fmt.Println(color.CyanString("📅 周末加班分析:"))
	fmt.Println()

	weekend := result.WeekendOvertime
	total := weekend.SaturdayDays + weekend.SundayDays

	var satPct, sunPct float64
	if total > 0 {
		satPct = float64(weekend.SaturdayDays) * 100 / float64(total)
		sunPct = float64(weekend.SundayDays) * 100 / float64(total)
	}

	max := weekend.SaturdayDays
	if weekend.SundayDays > max {
		max = weekend.SundayDays
	}

	satBar := ""
	sunBar := ""
	if max > 0 {
		satLen := weekend.SaturdayDays * 20 / max
		sunLen := weekend.SundayDays * 20 / max
		if satLen < 1 && weekend.SaturdayDays > 0 {
			satLen = 1
		}
		if sunLen < 1 && weekend.SundayDays > 0 {
			sunLen = 1
		}
		satBar = strings.Repeat("█", satLen)
		sunBar = strings.Repeat("█", sunLen)
	}

	fmt.Printf("周六：%-20s %3d天 (%.1f%%)\n", satBar, weekend.SaturdayDays, satPct)
	fmt.Printf("周日：%-20s %3d天 (%.1f%%)\n", sunBar, weekend.SundayDays, sunPct)
	fmt.Println()
	fmt.Println("加班类型:")
	fmt.Printf("  真正加班：%d天 (提交时间跨度>=3 小时)\n", weekend.RealOvertimeDays)
	fmt.Printf("  临时修复：%d天 (提交时间跨度<3 小时)\n", weekend.CasualFixDays)
	if total > 0 {
		fmt.Printf("  加班占比：%.1f%%\n", float64(weekend.RealOvertimeDays)*100/float64(total))
	}
}

// printLateNightOvertime 打印深夜加班分析
func printLateNightOvertime(result types.AnalysisResult) {
	fmt.Println()
	fmt.Println(color.CyanString("🌙 深夜加班分析:"))
	fmt.Println()

	lateNight := result.LateNight

	// 计算总工作日数（估算）
	totalWorkDays := result.CommitData.TotalCommits / 5

	fmt.Printf("20:00-21:00: %-20s %3d天 (晚间提交) 平均每周%.1f天 每月%.1f天\n",
		strings.Repeat("█", lateNight.Evening*20/100), lateNight.Evening,
		float64(lateNight.Evening)/52, float64(lateNight.Evening)/12)
	fmt.Printf("21:00-23:00: %-20s %3d天 (加班晚期) 平均每周%.1f天 每月%.1f天\n",
		strings.Repeat("█", lateNight.LateNight*20/100), lateNight.LateNight,
		float64(lateNight.LateNight)/52, float64(lateNight.LateNight)/12)
	fmt.Printf("23:00-02:00: %-20s %3d天 (深夜加班) ⚠️ 平均每周%.1f天 每月%.1f天\n",
		strings.Repeat("█", lateNight.Midnight*20/100), lateNight.Midnight,
		float64(lateNight.Midnight)/52, float64(lateNight.Midnight)/12)
	fmt.Println()
	if totalWorkDays > 0 {
		fmt.Printf("深夜/凌晨加班天数：%d 天 / %d天工作日 (%.1f%%)\n",
			lateNight.MidnightDays, totalWorkDays,
			float64(lateNight.MidnightDays)*100/float64(totalWorkDays))
	}
}

// printAdvancedAnalysis 打印高级分析
func printAdvancedAnalysis(result types.AnalysisResult) {
	fmt.Println()
	fmt.Println(color.CyanString("🧠 疲劳度与节奏分析:"))
	fmt.Println()

	fatigue := result.Fatigue
	rhythm := result.Rhythm

	// 疲劳度
	var fatigueIcon string
	switch fatigue.LevelStr {
	case "健康":
		fatigueIcon = "🟢"
	case "警告":
		fatigueIcon = "🟡"
	case "疲劳":
		fatigueIcon = "🟠"
	case "危险":
		fatigueIcon = "🔴"
	default:
		fatigueIcon = "⚪"
	}

	fmt.Printf("疲劳度：%s %s\n", fatigueIcon, fatigue.LevelStr)
	fmt.Printf("  连续加班最多 %d 天\n", fatigue.MaxConsecutiveDays)
	fmt.Println()

	// 节奏
	var rhythmIcon string
	switch rhythm.Pattern {
	case "规律型":
		rhythmIcon = "📅"
	case "爆发型":
		rhythmIcon = "💥"
	case "随机型":
		rhythmIcon = "🎲"
	default:
		rhythmIcon = "📊"
	}

	fmt.Printf("节奏：%s %s\n", rhythmIcon, rhythm.Pattern)
	fmt.Printf("  一致性：%.0f%%\n", rhythm.Consistency)
	if rhythm.PeakHour > 0 {
		fmt.Printf("  高峰时段：%02d:00\n", rhythm.PeakHour)
	}
}

// printFooter 打印底部提示
func printFooter() {
	fmt.Println()
	fmt.Println(color.CyanString(strings.Repeat("─", 78)))
	fmt.Println()
	fmt.Println("ℹ️  使用提示:")
	fmt.Println()
	fmt.Println("  ● 隐私保护：所有 Git 数据分析均在本地完成，不会上传任何数据。")
	fmt.Println("  ● 分析局限性：仅统计 commit 提交时间，不包含会议、学习、调试等活动。")
	fmt.Println("  ● 使用限制：分析结果仅供参考，请勿用于不当用途。")
	fmt.Println()
	fmt.Println("  📖 常用命令:")
	fmt.Println("     gitpulse                           分析当前仓库")
	fmt.Println("     gitpulse /path/to/repo             分析指定仓库")
	fmt.Println("     gitpulse -y 2025                   分析 2025 年数据")
	fmt.Println("     gitpulse -y 2023-2025              分析 2023-2025 年数据")
	fmt.Println("     gitpulse -s 2025-01-01             指定开始日期")
	fmt.Println("     gitpulse -u 2025-12-31             指定结束日期")
	fmt.Println("     gitpulse --all-time                分析全部历史")
	fmt.Println("     gitpulse --self                    只看自己的提交")
	fmt.Println("     gitpulse -a user@example.com       指定作者")
	fmt.Println("     gitpulse -x bot                    排除作者")
	fmt.Println("     gitpulse -b main                   指定分支")
	fmt.Println("     gitpulse --hours 9-18              指定工作时间")
	fmt.Println("     gitpulse --timezone +0800          指定时区")
	fmt.Println("     gitpulse --cn                      启用中国节假日调休")
	fmt.Println("     gitpulse --export json -o out.json 导出为 JSON")
	fmt.Println("     gitpulse --export csv -o out.csv   导出为 CSV")
	fmt.Println("     gitpulse -v                        显示热力图")
	fmt.Println("     gitpulse --init                    初始化配置文件")
	fmt.Println()
	fmt.Println("  📦 项目地址：https://github.com/hello-Banana/gitPulse")
}

// PrintHeatmap 打印热力图
func PrintHeatmap(commitData types.GitCommitData) {
	fmt.Println()
	fmt.Println(color.CyanString(strings.Repeat("═", 70)))
	fmt.Println("  📅 24 小时提交热力图")
	fmt.Println(color.CyanString(strings.Repeat("═", 70)))
	fmt.Println()

	// 按星期几和小时聚合数据
	heatmap := make(map[string]int)
	for _, dhc := range commitData.DayHourCommits {
		key := fmt.Sprintf("%d-%d", dhc.Weekday, dhc.Hour)
		heatmap[key] = dhc.Count
	}

	// 找最大值用于颜色分级
	maxCount := 0
	for _, count := range heatmap {
		if count > maxCount {
			maxCount = count
		}
	}

	// 打印表头 - 修正对齐
	fmt.Print("        ") // 8 个空格，对齐时间列
	weekdays := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	for _, day := range weekdays {
		fmt.Printf(" %s ", day) // 每个星期占 3 个字符宽度
	}
	fmt.Println()

	// 打印每一行
	for hour := 0; hour < 24; hour++ {
		fmt.Printf("%02d:00  ", hour) // 时间列占 6 个字符
		for weekday := 1; weekday <= 7; weekday++ {
			key := fmt.Sprintf("%d-%d", weekday, hour)
			count := heatmap[key]

			// 根据数量选择颜色块
			block := "░░"
			if count > 0 {
				ratio := float64(count) / float64(maxCount)
				if ratio > 0.75 {
					block = color.New(color.FgRed).Sprintf("██")
				} else if ratio > 0.5 {
					block = color.New(color.FgYellow).Sprintf("██")
				} else if ratio > 0.25 {
					block = color.New(color.FgGreen).Sprintf("██")
				} else {
					block = color.New(color.FgHiBlack).Sprintf("▒▒")
				}
			}
			fmt.Printf(" %s ", block) // 每个色块占 3 个字符宽度
		}
		fmt.Println()
	}
	fmt.Println()
}

// ExportJSON 导出为 JSON
func ExportJSON(result types.AnalysisResult, outputPath string) error {
	exportData := buildExportData(result)

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return err
	}

	var writer io.Writer
	if outputPath == "" || outputPath == "-" {
		writer = os.Stdout
	} else {
		file, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	}

	_, err = writer.Write(data)
	return err
}

// ExportCSV 导出为 CSV
func ExportCSV(result types.AnalysisResult, outputPath string) error {
	var writer io.Writer
	if outputPath == "" || outputPath == "-" {
		writer = os.Stdout
	} else {
		file, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	}

	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// 写入表头
	header := []string{
		"指标", "值",
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	// 写入数据
	rows := [][]string{
		{"总提交数", fmt.Sprintf("%d", result.CommitData.TotalCommits)},
		{"加班比例", fmt.Sprintf("%d%%", result.WorkIntensity.OverTimeRadio)},
		{"工作强度指数", fmt.Sprintf("%d", result.WorkIntensity.Index)},
		{"指数评价", result.WorkIntensity.IndexStr},
		{"上班时间", formatTime(result.WorkTimeDetect.StartHour)},
		{"下班时间", formatTime(result.WorkTimeDetect.EndHour)},
		{"工作日加班", fmt.Sprintf("%d", result.WeekdayOvertime.Monday+result.WeekdayOvertime.Tuesday+result.WeekdayOvertime.Wednesday+result.WeekdayOvertime.Thursday+result.WeekdayOvertime.Friday)},
		{"周末加班", fmt.Sprintf("%d", result.WeekendOvertime.SaturdayDays+result.WeekendOvertime.SundayDays)},
		{"深夜加班", fmt.Sprintf("%d 天 (%.1f%%)", result.LateNight.MidnightDays, result.LateNight.MidnightRate)},
		{"疲劳度", fmt.Sprintf("%s (%d 天)", result.Fatigue.LevelStr, result.Fatigue.MaxConsecutiveDays)},
		{"节奏模式", result.Rhythm.Pattern},
	}

	for _, row := range rows {
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// buildExportData 构建导出数据
func buildExportData(result types.AnalysisResult) types.ExportData {
	return types.ExportData{
		ReportTime: time.Now().Format(time.RFC3339),
		Repo:       result.RepoName,
		Period: types.ExportPeriod{
			Since: result.CommitData.Since,
			Until: result.CommitData.Until,
		},
		Summary: types.ExportSummary{
			TotalCommits:       result.CommitData.TotalCommits,
			OvertimeCommits:    result.WorkIntensity.OverTimeRadio * result.CommitData.TotalCommits / 100,
			OvertimeRatio:      float64(result.WorkIntensity.OverTimeRadio) / 100,
			WorkIntensityIndex: result.WorkIntensity.Index,
			WorkHours:          result.WorkTimeDetect.EndHour - result.WorkTimeDetect.StartHour,
		},
		Details: types.ExportDetails{
			WeekdayOvertime: []map[string]interface{}{
				{"周一": result.WeekdayOvertime.Monday},
				{"周二": result.WeekdayOvertime.Tuesday},
				{"周三": result.WeekdayOvertime.Wednesday},
				{"周四": result.WeekdayOvertime.Thursday},
				{"周五": result.WeekdayOvertime.Friday},
			},
			WeekendOvertime: map[string]interface{}{
				"saturday_days":      result.WeekendOvertime.SaturdayDays,
				"sunday_days":        result.WeekendOvertime.SundayDays,
				"real_overtime_days": result.WeekendOvertime.RealOvertimeDays,
			},
			LateNightOvertime: map[string]interface{}{
				"midnight_days": result.LateNight.MidnightDays,
				"midnight_rate": result.LateNight.MidnightRate,
			},
			Fatigue: map[string]interface{}{
				"level":           result.Fatigue.LevelStr,
				"max_consecutive": result.Fatigue.MaxConsecutiveDays,
			},
			Rhythm: map[string]interface{}{
				"pattern":     result.Rhythm.Pattern,
				"peak_hour":   result.Rhythm.PeakHour,
				"consistency": result.Rhythm.Consistency,
			},
		},
	}
}

// PrintTable 使用表格打印（简化版）
func PrintTable(result types.AnalysisResult) {
	fmt.Println()
	fmt.Printf("%-20s %s\n", "📊 工作强度指数:", color.CyanString("%d", result.WorkIntensity.Index))
	fmt.Printf("%-20s %s\n", "评价:", result.WorkIntensity.IndexStr)
	fmt.Printf("%-20s %s\n", "加班比例:", color.YellowString("%d%%", result.WorkIntensity.OverTimeRadio))
	fmt.Printf("%-20s %s\n", "总提交数:", color.GreenString("%d", result.CommitData.TotalCommits))
	fmt.Printf("%-20s %s - %s\n", "工作时间:", formatTime(result.WorkTimeDetect.StartHour), formatTime(result.WorkTimeDetect.EndHour))
	fmt.Printf("%-20s %s\n", "疲劳度:", result.Fatigue.LevelStr)
	fmt.Printf("%-20s %s\n", "节奏:", result.Rhythm.Pattern)
}
