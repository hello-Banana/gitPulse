package engine

import (
	"math"
	"sort"
	"time"

	"gitpulse/internal/config"
)

// FatigueLevel 疲劳等级
type FatigueLevel int

const (
	FatigueLevelHealthy FatigueLevel = iota // 健康
	FatigueLevelWarning                     // 注意
	FatigueLevelFatigue                     // 疲劳
	FatigueLevelDanger                      // 危险
)

// FatigueResult 疲劳度分析结果
type FatigueResult struct {
	MaxConsecutiveDays int          `json:"max_consecutive_days"`
	MaxWeeklyHours     float64      `json:"max_weekly_hours"`
	Level              FatigueLevel `json:"level"`
	LevelStr           string       `json:"level_str"`
}

// RhythmAnalysis 节奏分析结果
type RhythmAnalysis struct {
	Pattern       string  `json:"pattern"`        // 规律型 | 爆发型 | 随机型
	PeakHour      int     `json:"peak_hour"`      // 提交高峰时段
	Consistency   float64 `json:"consistency"`    // 一致性得分 0-100
	BurstDetected bool    `json:"burst_detected"` // 是否检测到深夜爆发
}

// CommitRecord 提交记录（用于分析）
type CommitRecord struct {
	Time      time.Time
	IsOvertime bool
	IsLateNight bool
	IsWeekend bool
	Additions int
	Deletions int
}

// AnalyzeFatigue 分析疲劳度
func AnalyzeFatigue(commits []CommitRecord, cfg *config.Config) *FatigueResult {
	if len(commits) == 0 {
		return &FatigueResult{
			Level:    FatigueLevelHealthy,
			LevelStr: "无数据",
		}
	}

	// 按时间排序
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Time.Before(commits[j].Time)
	})

	// 计算连续加班天数
	maxConsecutive := 0
	currentConsecutive := 0
	lastDate := time.Time{}

	// 周加班时长统计
	weeklyHours := make(map[string]float64)

	for _, c := range commits {
		if !c.IsOvertime {
			currentConsecutive = 0
			continue
		}

		// 判断是否为新的一天
		currentDate := time.Date(c.Time.Year(), c.Time.Month(), c.Time.Day(), 0, 0, 0, 0, c.Time.Location())
		if !currentDate.Equal(lastDate) {
			if !lastDate.IsZero() && currentDate.Sub(lastDate).Hours() < 30 {
				// 连续的一天（考虑跨天）
				currentConsecutive++
			} else {
				currentConsecutive = 1
			}
			lastDate = currentDate
		}

		if currentConsecutive > maxConsecutive {
			maxConsecutive = currentConsecutive
		}

		// 计算周加班时长
		year, week := c.Time.ISOWeek()
		weekKey := string(rune(year)) + "-" + string(rune(week))
		weeklyHours[weekKey] += 1.0 // 每次加班提交计为 1 小时（简化）
	}

	// 找最大周加班时长
	maxWeeklyHours := 0.0
	for _, hours := range weeklyHours {
		if hours > maxWeeklyHours {
			maxWeeklyHours = hours
		}
	}

	// 计算疲劳等级
	level := FatigueLevelHealthy
	levelStr := "健康"

	threshold := cfg.FatigueAlert.ConsecutiveDays
	if threshold == 0 {
		threshold = 5
	}

	if maxConsecutive >= threshold*2 {
		level = FatigueLevelDanger
		levelStr = "危险"
	} else if maxConsecutive >= threshold+3 {
		level = FatigueLevelFatigue
		levelStr = "疲劳"
	} else if maxConsecutive >= threshold {
		level = FatigueLevelWarning
		levelStr = "注意"
	}

	return &FatigueResult{
		MaxConsecutiveDays: maxConsecutive,
		MaxWeeklyHours:     maxWeeklyHours,
		Level:              level,
		LevelStr:           levelStr,
	}
}

// AnalyzeRhythm 分析提交节奏
func AnalyzeRhythm(commits []CommitRecord) *RhythmAnalysis {
	if len(commits) == 0 {
		return &RhythmAnalysis{
			Pattern:     "无数据",
			PeakHour:    0,
			Consistency: 0,
		}
	}

	// 按小时聚合
	hourDist := make(map[int]int)
	for _, c := range commits {
		hourDist[c.Time.Hour()]++
	}

	// 找高峰时段
	peakHour := 0
	peakCount := 0
	for h, cnt := range hourDist {
		if cnt > peakCount {
			peakCount = cnt
			peakHour = h
		}
	}

	// 计算方差（一致性）
	variance := calculateVariance(hourDist)
	consistency := 100 - math.Min(100, variance*10)

	// 判断模式
	pattern := "规律型"
	if variance > 2.0 {
		pattern = "爆发型"
	} else if variance > 1.0 {
		pattern = "随机型"
	}

	// 检测深夜爆发
	burstDetected := detectBurst(hourDist)

	return &RhythmAnalysis{
		Pattern:       pattern,
		PeakHour:      peakHour,
		Consistency:   consistency,
		BurstDetected: burstDetected,
	}
}

// calculateVariance 计算方差
func calculateVariance(data map[int]int) float64 {
	if len(data) == 0 {
		return 0
	}

	// 计算平均值
	sum := 0
	for _, v := range data {
		sum += v
	}
	mean := float64(sum) / float64(len(data))

	// 计算方差
	variance := 0.0
	for _, v := range data {
		diff := float64(v) - mean
		variance += diff * diff
	}
	variance /= float64(len(data))

	return math.Sqrt(variance)
}

// detectBurst 检测深夜爆发
func detectBurst(hourDist map[int]int) bool {
	// 检查 21 点后的提交是否显著高于平均水平
	lateNightCount := 0
	totalCount := 0

	for h, cnt := range hourDist {
		totalCount += cnt
		if h >= 21 {
			lateNightCount += cnt
		}
	}

	if totalCount == 0 {
		return false
	}

	// 深夜提交占比超过 20% 视为爆发
	return float64(lateNightCount)/float64(totalCount) > 0.2
}

// GetFatigueEmoji 获取疲劳等级表情
func GetFatigueEmoji(level FatigueLevel) string {
	switch level {
	case FatigueLevelHealthy:
		return "🟢"
	case FatigueLevelWarning:
		return "🟡"
	case FatigueLevelFatigue:
		return "🟠"
	case FatigueLevelDanger:
		return "🔴"
	default:
		return "⚪"
	}
}

// GetPatternEmoji 获取节奏模式表情
func GetPatternEmoji(pattern string) string {
	switch pattern {
	case "规律型":
		return "📅"
	case "爆发型":
		return "💥"
	case "随机型":
		return "🎲"
	default:
		return "📊"
	}
}
