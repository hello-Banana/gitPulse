package core

import (
	"math"
	"sort"
	"time"

	"gitpulse/types"
)

// OvertimeAnalyzer 加班分析器
type OvertimeAnalyzer struct{}

// NewOvertimeAnalyzer 创建加班分析器
func NewOvertimeAnalyzer() *OvertimeAnalyzer {
	return &OvertimeAnalyzer{}
}

// AnalyzeWeekdayOvertime 分析工作日加班分布
func (o *OvertimeAnalyzer) AnalyzeWeekdayOvertime(
	dayHourCommits []types.DayHourCommit,
	workTime types.WorkTimeDetectionResult,
) types.WeekdayOvertimeDistribution {
	endHour := int(math.Ceil(workTime.EndHour))

	// 初始化周一到周五的加班计数
	overtimeCounts := map[string]int{
		"monday":    0,
		"tuesday":   0,
		"wednesday": 0,
		"thursday":  0,
		"friday":    0,
	}

	dayNames := map[int]string{
		1: "monday",
		2: "tuesday",
		3: "wednesday",
		4: "thursday",
		5: "friday",
	}

	// 统计每个工作日下班后的提交数
	for _, commit := range dayHourCommits {
		// 只统计工作日（周一到周五：1-5）
		if dayName, ok := dayNames[commit.Weekday]; ok {
			// 只统计下班时间之后的提交
			if commit.Hour >= endHour {
				overtimeCounts[dayName] += commit.Count
			}
		}
	}

	// 找出加班最多的一天
	peakDay := "monday"
	peakCount := 0
	for day, count := range overtimeCounts {
		if count > peakCount {
			peakCount = count
			peakDay = day
		}
	}

	dayNameMap := map[string]string{
		"monday":    "周一",
		"tuesday":   "周二",
		"wednesday": "周三",
		"thursday":  "周四",
		"friday":    "周五",
	}

	return types.WeekdayOvertimeDistribution{
		Monday:    overtimeCounts["monday"],
		Tuesday:   overtimeCounts["tuesday"],
		Wednesday: overtimeCounts["wednesday"],
		Thursday:  overtimeCounts["thursday"],
		Friday:    overtimeCounts["friday"],
		PeakDay:   dayNameMap[peakDay],
		PeakCount: peakCount,
	}
}

// AnalyzeWeekendOvertime 分析周末加班分布
func (o *OvertimeAnalyzer) AnalyzeWeekendOvertime(
	dailyCommitHours []types.DailyCommitHours,
) types.WeekendOvertimeDistribution {
	const REAL_OVERTIME_THRESHOLD = 3 // 提交时间跨度>=3 小时才算真正加班

	var saturdayDays int
	var sundayDays int
	var casualFixDays int
	var realOvertimeDays int

	for _, dch := range dailyCommitHours {
		commitDate, err := time.Parse("2006-01-02", dch.Date)
		if err != nil {
			continue
		}

		dayOfWeek := int(commitDate.Weekday())
		commitHours := len(dch.Hours)

		// 只统计周末
		if dayOfWeek != 0 && dayOfWeek != 6 {
			continue
		}

		// 判断是否为真正加班
		isRealOvertime := commitHours >= REAL_OVERTIME_THRESHOLD

		if dayOfWeek == 6 {
			// 周六
			saturdayDays++
			if isRealOvertime {
				realOvertimeDays++
			} else {
				casualFixDays++
			}
		} else if dayOfWeek == 0 {
			// 周日
			sundayDays++
			if isRealOvertime {
				realOvertimeDays++
			} else {
				casualFixDays++
			}
		}
	}

	return types.WeekendOvertimeDistribution{
		SaturdayDays:     saturdayDays,
		SundayDays:       sundayDays,
		CasualFixDays:    casualFixDays,
		RealOvertimeDays: realOvertimeDays,
	}
}

// AnalyzeLateNight 分析深夜加班情况
func (o *OvertimeAnalyzer) AnalyzeLateNight(
	dailyLatestCommits []types.DailyLatestCommit,
	dailyFirstCommits []types.DailyFirstCommit,
	workTime types.WorkTimeDetectionResult,
	since, until string,
) types.LateNightAnalysis {
	endHour := int(math.Ceil(workTime.EndHour))

	// 统计不同时段的天数
	var evening int    // 下班后 -21:00
	var lateNight int  // 21:00-23:00
	var midnight int   // 23:00-02:00
	var dawn int       // 02:00-06:00

	// 统计有深夜/凌晨提交的天数
	midnightDaysSet := make(map[string]bool)

	// 按照每天的最晚提交时间来统计
	for _, commit := range dailyLatestCommits {
		latestHour := commit.MinutesFromMidnight / 60

		if latestHour >= endHour && latestHour < 21 {
			evening++
		} else if latestHour >= 21 && latestHour < 23 {
			lateNight++
		} else if latestHour >= 23 {
			// 23:00-23:59 算作深夜
			midnight++
			midnightDaysSet[commit.Date] = true
		} else if latestHour < 6 {
			// 00:00-05:59 算作凌晨
			dawn++
			midnightDaysSet[commit.Date] = true
		}
	}

	// 统计总工作日天数
	workDaysSet := make(map[string]bool)
	for _, commit := range dailyFirstCommits {
		commitDate, err := time.Parse("2006-01-02", commit.Date)
		if err != nil {
			continue
		}
		dayOfWeek := int(commitDate.Weekday())
		if dayOfWeek >= 1 && dayOfWeek <= 5 {
			workDaysSet[commit.Date] = true
		}
	}

	midnightDays := len(midnightDaysSet)
	totalWorkDays := len(workDaysSet)
	if totalWorkDays == 0 {
		totalWorkDays = 1
	}
	midnightRate := float64(midnightDays) / float64(totalWorkDays) * 100

	// 计算总周数和月数
	var totalWeeks int
	var totalMonths int

	if since != "" && until != "" {
		sinceDate, _ := time.Parse("2006-01-02", since)
		untilDate, _ := time.Parse("2006-01-02", until)
		diffTime := untilDate.Sub(sinceDate)
		diffDays := int(diffTime.Hours() / 24)

		totalWeeks = max(1, diffDays/7)
		totalMonths = max(1, diffDays/30)
	} else {
		totalWeeks = max(1, totalWorkDays/5)
		totalMonths = max(1, totalWorkDays/22)
	}

	return types.LateNightAnalysis{
		Evening:       evening,
		LateNight:     lateNight,
		Midnight:      midnight,
		Dawn:          dawn,
		MidnightDays:  midnightDays,
		TotalWorkDays: totalWorkDays,
		MidnightRate:  midnightRate,
		TotalWeeks:    totalWeeks,
		TotalMonths:   totalMonths,
	}
}

// AnalyzeWorkSpan 分析工作跨度（每日工作时长）
func (o *OvertimeAnalyzer) AnalyzeWorkSpan(dailyCommits []types.DailyCommitHours) []float64 {
	var workSpans []float64

	for _, dch := range dailyCommits {
		if len(dch.Hours) == 0 {
			continue
		}

		// 收集所有小时
		hours := make([]int, 0, len(dch.Hours))
		for h := range dch.Hours {
			hours = append(hours, h)
		}

		if len(hours) == 0 {
			continue
		}

		// 排序
		sort.Ints(hours)

		// 计算跨度
		span := float64(hours[len(hours)-1] - hours[0] + 1)
		workSpans = append(workSpans, span)
	}

	return workSpans
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
