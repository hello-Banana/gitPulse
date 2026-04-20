package core

import (
	"math"
	"sort"

	"gitpulse/types"
)

// WorkTimeAnalyzer 工作时间分析器
type WorkTimeAnalyzer struct{}

// NewWorkTimeAnalyzer 创建工作时间分析器
func NewWorkTimeAnalyzer() *WorkTimeAnalyzer {
	return &WorkTimeAnalyzer{}
}

// Analyze 分析工作时间
func (w *WorkTimeAnalyzer) Analyze(commitData *types.GitCommitData) types.WorkTimeData {
	// 构建工作时间分布
	workHourPl := [2]types.TimeCount{
		{Time: "work", Count: 0},
		{Time: "overtime", Count: 0},
	}

	// 统计工作日和周末的提交
	workWeekPl := [2]types.TimeCount{
		{Time: "weekday", Count: 0},
		{Time: "weekend", Count: 0},
	}

	// 默认工作时间（会被后续算法覆盖）
	defaultStartHour := 9.0
	defaultEndHour := 18.0

	// 识别工作时间
	detectResult := w.DetectWorkTime(commitData.DailyFirstCommits, commitData.HourData)

	startHour := detectResult.StartHour
	endHour := detectResult.EndHour

	// 如果没有检测到有效时间，使用默认值
	if startHour <= 0 {
		startHour = defaultStartHour
	}
	if endHour <= 0 {
		endHour = defaultEndHour
	}

	// 计算工作时间分布
	endHourInt := int(math.Ceil(endHour))
	startHourInt := int(math.Floor(startHour))

	for i, hc := range commitData.HourData {
		if i >= startHourInt && i < endHourInt {
			workHourPl[0].Count += hc.Count
		} else {
			workHourPl[1].Count += hc.Count
		}
	}

	// 计算工作日/周末分布
	for weekday, count := range commitData.WeekdayStats {
		if weekday >= 1 && weekday <= 5 {
			workWeekPl[0].Count += count
		} else {
			workWeekPl[1].Count += count
		}
	}

	return types.WorkTimeData{
		HourData:   commitData.HourData,
		WorkHourPl: workHourPl,
		WorkWeekPl: workWeekPl,
	}
}

// DetectWorkTime 检测工作时间（使用分位数和拐点检测）
func (w *WorkTimeAnalyzer) DetectWorkTime(
	dailyFirstCommits []types.DailyFirstCommit,
	hourData []types.TimeCount,
) types.WorkTimeDetectionResult {
	result := types.WorkTimeDetectionResult{
		StartHour:  0,
		EndHour:    0,
		Confidence: "low",
	}

	if len(dailyFirstCommits) == 0 {
		return result
	}

	// 收集每日首次提交时间（分钟）
	// 过滤清晨 (0-5 点) 的提交，这些可能是夜间工作的延续
	var firstCommits []int
	for _, fc := range dailyFirstCommits {
		if fc.MinutesFromMidnight >= 4*60 && fc.MinutesFromMidnight <= 14*60 {
			firstCommits = append(firstCommits, fc.MinutesFromMidnight)
		}
	}

	if len(firstCommits) < 2 {
		// 如果有效样本太少，使用所有数据
		for _, fc := range dailyFirstCommits {
			if fc.MinutesFromMidnight <= 14*60 {
				firstCommits = append(firstCommits, fc.MinutesFromMidnight)
			}
		}
	}

	if len(firstCommits) < 2 {
		return result
	}

	// 排序
	sort.Ints(firstCommits)

	// 取 10%-20% 分位作为上班时间
	idx10 := len(firstCommits) * 10 / 100
	idx20 := len(firstCommits) * 20 / 100

	if idx10 >= len(firstCommits) {
		idx10 = 0
	}
	if idx20 >= len(firstCommits) {
		idx20 = len(firstCommits) - 1
	}

	// 向下取半小时
	startMinutes := (firstCommits[idx10] + firstCommits[idx20]) / 2
	startHour := float64(startMinutes) / 60.0

	// 向下取整到半小时
	startHour = math.Floor(startHour*2) / 2.0

	// 确保上班时间合理
	if startHour < 4 {
		startHour = 9
	}

	result.StartHour = startHour
	result.Confidence = "medium"

	// 检测下班时间（使用晚间拐点）
	endHour := w.detectEndHour(hourData)
	if endHour > 0 {
		result.EndHour = endHour
		result.Confidence = "high"
	} else {
		// 兜底：上班时间 + 9 小时
		result.EndHour = startHour + 9
	}

	return result
}

// detectEndHour 检测下班时间（使用晚间拐点）
func (w *WorkTimeAnalyzer) detectEndHour(hourData []types.TimeCount) float64 {
	if len(hourData) < 12 {
		return 0
	}

	// 从 17 点开始寻找提交数量显著下降的拐点
	maxCount := 0
	maxHour := 17
	for i := 17; i < 24; i++ {
		if hourData[i].Count > maxCount {
			maxCount = hourData[i].Count
			maxHour = i
		}
	}

	// 寻找拐点：提交数量开始显著下降的位置
	for i := maxHour + 1; i < 24; i++ {
		if i+1 < 24 {
			// 如果下一小时的提交数显著减少（减少 30% 以上）
			if hourData[i].Count > 0 && hourData[i+1].Count < hourData[i].Count*7/10 {
				return float64(i + 1)
			}
		}
	}

	// 如果没有找到拐点，使用默认值
	return 0
}
