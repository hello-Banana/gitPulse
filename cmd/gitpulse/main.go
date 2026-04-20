package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitpulse/internal/collector"
	"gitpulse/internal/config"
	"gitpulse/internal/engine"
	"gitpulse/internal/printer"
	"gitpulse/types"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	version = "0.2.0"

	// CLI 参数
	repoPath      string
	configPath    string
	year          string
	since         string
	until         string
	allTime       bool
	self          bool
	author        string
	excludeAuthor string
	branch        string
	ignoreMsg     string
	hours         string
	halfHour      bool
	timezone      string
	cnHoliday     bool
	verbose       bool
	exportFormat  string
	output        string
	showHeatmap   bool
	initConfig    bool
	listAuthors   bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gitpulse [repo-path]",
		Short: "Git 仓库工作强度分析工具",
		Long: `gitPulse - Git 仓库工作强度分析工具

统计 Git 项目的 commit 时间分布，结合灵活的工作制配置，
输出多维度的研发加班统计报告。

使用方法:
  gitpulse [repo-path] [flags]

示例:
  gitpulse                          # 分析当前仓库
  gitpulse /path/to/repo            # 分析指定仓库
  gitpulse -y 2025                  # 分析 2025 年
  gitpulse --self                   # 只分析自己的提交
  gitpulse --export json -o out.json # 导出 JSON
`,
		Version: version,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runAnalysis,
	}

	// 路径参数
	rootCmd.Flags().StringVarP(&repoPath, "path", "p", "", "指定仓库路径 (默认当前目录)")
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "配置文件路径 (.git-ot.yaml)")

	// 时间范围
	rootCmd.Flags().StringVarP(&year, "year", "y", "", "指定年份或年份范围 (如 2025 或 2023-2025)")
	rootCmd.Flags().StringVarP(&since, "since", "s", "", "自定义开始日期 (YYYY-MM-DD)")
	rootCmd.Flags().StringVarP(&until, "until", "u", "", "自定义结束日期 (YYYY-MM-DD)")
	rootCmd.Flags().BoolVar(&allTime, "all-time", false, "覆盖整个仓库历史数据")

	// 作者过滤
	rootCmd.Flags().BoolVar(&self, "self", false, "仅统计当前 Git 用户的提交记录")
	rootCmd.Flags().StringVarP(&author, "author", "a", "", "指定作者 (邮箱或姓名)")
	rootCmd.Flags().StringVarP(&excludeAuthor, "exclude-author", "x", "", "排除作者 (支持通配符)")
	rootCmd.Flags().BoolVar(&listAuthors, "authors", false, "列出所有作者")

	// 其他过滤
	rootCmd.Flags().StringVarP(&branch, "branch", "b", "", "指定分支")
	rootCmd.Flags().StringVarP(&ignoreMsg, "ignore-msg", "m", "", "排除匹配正则的提交信息")

	// 工作制配置
	rootCmd.Flags().StringVar(&hours, "hours", "", "手动指定标准工作时间 (如 9-18 或 9.5-18.5)")
	rootCmd.Flags().StringVar(&timezone, "timezone", "", "指定时区进行分析 (如 +0800)")
	rootCmd.Flags().BoolVar(&cnHoliday, "cn", false, "强制开启中国节假日调休模式")

	// 输出选项
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "显示详细信息 (包含热力图)")
	rootCmd.Flags().StringVar(&exportFormat, "export", "", "导出格式 (json/csv)")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "输出文件路径")
	rootCmd.Flags().BoolVar(&showHeatmap, "heatmap", false, "显示热力图")
	rootCmd.Flags().BoolVar(&initConfig, "init", false, "初始化配置文件")

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(initConfigCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("gitPulse version %s\n", version)
		},
	}
}

func initConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "初始化配置文件",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.DefaultConfig()
			path := ".git-ot.yaml"
			if len(args) > 0 {
				path = args[0]
			}
			if err := config.SaveConfig(cfg, path); err != nil {
				return err
			}
			fmt.Printf("配置文件已创建：%s\n", path)
			return nil
		},
	}
}

// runAnalysis 执行分析
func runAnalysis(cmd *cobra.Command, args []string) error {
	// 处理初始化配置
	if initConfig {
		cfg := config.DefaultConfig()
		path := ".git-ot.yaml"
		if configPath != "" {
			path = configPath
		}
		if err := config.SaveConfig(cfg, path); err != nil {
			return err
		}
		fmt.Printf("配置文件已创建：%s\n", path)
		return nil
	}

	// 处理 repo 路径
	if repoPath == "" {
		if len(args) > 0 {
			repoPath = args[0]
		} else {
			var err error
			repoPath, err = os.Getwd()
			if err != nil {
				return err
			}
		}
	}

	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败：%v", err)
	}

	// 命令行参数覆盖配置
	if hours != "" {
		parts := strings.Split(hours, "-")
		if len(parts) == 2 {
			cfg.WorkSettings.StandardHours.Start = parts[0]
			cfg.WorkSettings.StandardHours.End = parts[1]
		}
	}
	if self {
		// 获取当前用户邮箱
		email, _ := getCurrentUserEmail(repoPath)
		if email != "" {
			author = email
		}
	}

	// 检查是否为 git 仓库
	if !isGitRepo(repoPath) {
		return fmt.Errorf("%s 不是 Git 仓库", repoPath)
	}

	// 创建采集器
	gitCollector, err := collector.NewGitCollector(repoPath)
	if err != nil {
		return fmt.Errorf("打开仓库失败：%v", err)
	}

	// 列出所有作者
	if listAuthors {
		authors, err := gitCollector.GetAuthors()
		if err != nil {
			return fmt.Errorf("获取作者列表失败：%v", err)
		}
		fmt.Println(color.CyanString("仓库作者列表:"))
		fmt.Println()
		for i, author := range authors {
			fmt.Printf("  %d. %s\n", i+1, author)
		}
		fmt.Printf("\n共 %d 位作者\n", len(authors))
		fmt.Println()
		fmt.Println("使用 -a 或 --author 参数查看特定作者的统计:")
		fmt.Println("  gitpulse -a \"author@example.com\"")
		return nil
	}

	// 处理时间范围
	timeSince, timeUntil := parseTimeRange(year, since, until, allTime)

	fmt.Println(color.CyanString("正在分析仓库：%s", repoPath))
	if !timeSince.IsZero() {
		fmt.Printf("时间范围：%s - %s\n", timeSince.Format("2006-01-02"), timeUntil.Format("2006-01-02"))
	}
	fmt.Println()

	// 采集数据（包含变更统计）
	fmt.Println("正在采集提交数据...")
	commits, err := gitCollector.CollectCommitsWithStats(&collector.CollectOptions{
		Since:         timeSince,
		Until:         timeUntil,
		Author:        author,
		ExcludeAuthor: excludeAuthor,
		Branch:        branch,
		IgnoreMsg:     ignoreMsg,
	})
	if err != nil {
		return fmt.Errorf("采集提交失败：%v", err)
	}

	if len(commits) == 0 {
		return fmt.Errorf("在指定时间范围内没有找到提交记录")
	}

	fmt.Printf("采集到 %d 条提交记录\n", len(commits))

	// 转换为分析用的 CommitRecord
	records := convertCommits(commits, cfg)

	// 分析数据
	result := analyzeData(records, cfg, repoPath)

	// 导出
	if exportFormat != "" {
		return exportResult(result, exportFormat, output)
	}

	// 打印报告
	printer.PrintReport(result)

	// 热力图
	if verbose || showHeatmap {
		printer.PrintHeatmap(result.CommitData)
	}

	return nil
}

// convertCommits 转换为分析记录
func convertCommits(commits []collector.CommitRecord, cfg *config.Config) []engine.CommitRecord {
	schedule := engine.NewScheduleEngine(cfg)
	var records []engine.CommitRecord

	for _, c := range commits {
		isOvertime := schedule.IsOvertime(c.Time)
		isLateNight := schedule.IsLateNight(c.Time)
		isWeekend := schedule.IsRestDay(c.Time)

		records = append(records, engine.CommitRecord{
			Time:      c.Time,
			IsOvertime: isOvertime,
			IsLateNight: isLateNight,
			IsWeekend:  isWeekend,
			Additions:  c.Additions,
			Deletions:  c.Deletions,
		})
	}

	return records
}

// analyzeData 分析数据
func analyzeData(records []engine.CommitRecord, cfg *config.Config, repoPath string) types.AnalysisResult {
	schedule := engine.NewScheduleEngine(cfg)

	// 构建提交数据
	commitData := buildCommitData(records)
	commitData.Since = records[0].Time.Format("2006-01-02")
	commitData.Until = records[len(records)-1].Time.Format("2006-01-02")

	// 工作时间分析
	workTimeData := buildWorkTimeData(records, schedule)
	workTimeDetect := detectWorkTime(records)

	// 计算工作强度指数
	workIntensity := calculateWorkIntensity(workTimeData)

	// 加班分析
	weekdayOvertime := analyzeWeekdayOvertime(records, workTimeDetect)
	weekendOvertime := analyzeWeekendOvertime(records, schedule)
	lateNight := analyzeLateNight(records, workTimeDetect)

	// 疲劳度分析
	fatigue := engine.AnalyzeFatigue(records, cfg)

	// 节奏分析
	rhythm := engine.AnalyzeRhythm(records)

	// 获取仓库名称
	repoName := filepath.Base(repoPath)

	return types.AnalysisResult{
		CommitData:      commitData,
		WorkTimeData:    workTimeData,
		WorkIntensity:   workIntensity,
		WorkTimeDetect:  workTimeDetect,
		WeekdayOvertime: weekdayOvertime,
		WeekendOvertime: weekendOvertime,
		LateNight:       lateNight,
		Fatigue: types.FatigueInfo{
			MaxConsecutiveDays: fatigue.MaxConsecutiveDays,
			Level:              fmt.Sprintf("%d", fatigue.Level),
			LevelStr:           fatigue.LevelStr,
			Emoji:              engine.GetFatigueEmoji(fatigue.Level),
		},
		Rhythm: types.RhythmInfo{
			Pattern:       rhythm.Pattern,
			PeakHour:      rhythm.PeakHour,
			Consistency:   rhythm.Consistency,
			BurstDetected: rhythm.BurstDetected,
			Emoji:         engine.GetPatternEmoji(rhythm.Pattern),
		},
		RepoPath: repoPath,
		RepoName: repoName,
	}
}

// buildCommitData 构建提交数据
func buildCommitData(records []engine.CommitRecord) types.GitCommitData {
	hourData := make([]types.TimeCount, 24)
	hourDataHalf := make([]types.TimeCount, 48)
	weekdayStats := make(map[int]int)
	dayHourCommits := make(map[string]*types.DayHourCommit)
	dailyHours := make(map[string]map[int]bool)
	dailyFirst := make(map[string]int)
	dailyLatest := make(map[string]int)

	for _, r := range records {
		hour := r.Time.Hour()
		minutes := r.Time.Hour()*60 + r.Time.Minute()
		halfHour := hour*2 + minutes/30
		weekday := int(r.Time.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		dateStr := r.Time.Format("2006-01-02")

		// 小时统计
		hourData[hour].Count++

		// 半小时统计
		if halfHour >= 0 && halfHour < 48 {
			hourDataHalf[halfHour].Count++
		}

		// 星期统计
		weekdayStats[weekday]++

		// 星期几 + 小时统计
		dhKey := fmt.Sprintf("%d-%d", weekday, hour)
		if _, exists := dayHourCommits[dhKey]; !exists {
			dayHourCommits[dhKey] = &types.DayHourCommit{
				Weekday: weekday,
				Hour:    hour,
			}
		}
		dayHourCommits[dhKey].Count++

		// 每日小时统计
		if _, exists := dailyHours[dateStr]; !exists {
			dailyHours[dateStr] = make(map[int]bool)
		}
		dailyHours[dateStr][hour] = true

		// 每日首次/最晚提交
		if _, exists := dailyFirst[dateStr]; !exists {
			dailyFirst[dateStr] = minutes
			dailyLatest[dateStr] = minutes
		} else {
			if minutes < dailyFirst[dateStr] {
				dailyFirst[dateStr] = minutes
			}
			if minutes > dailyLatest[dateStr] {
				dailyLatest[dateStr] = minutes
			}
		}
	}

	// 转换为切片
	var dhcList []types.DayHourCommit
	for _, dhc := range dayHourCommits {
		dhcList = append(dhcList, *dhc)
	}

	var dailyFirstList []types.DailyFirstCommit
	for date, minutes := range dailyFirst {
		dailyFirstList = append(dailyFirstList, types.DailyFirstCommit{
			Date:              date,
			MinutesFromMidnight: minutes,
		})
	}

	var dailyLatestList []types.DailyLatestCommit
	for date, minutes := range dailyLatest {
		dailyLatestList = append(dailyLatestList, types.DailyLatestCommit{
			Date:              date,
			MinutesFromMidnight: minutes,
		})
	}

	var dailyCommitHours []types.DailyCommitHours
	for date, hours := range dailyHours {
		dailyCommitHours = append(dailyCommitHours, types.DailyCommitHours{
			Date:  date,
			Hours: hours,
			Size:  len(hours),
		})
	}

	// 初始化 TimeCount 的 Time 字段
	for i := range hourData {
		hourData[i].Time = fmt.Sprintf("%02d:00", i)
	}
	for i := range hourDataHalf {
		hour := i / 2
		minute := (i % 2) * 30
		hourDataHalf[i].Time = fmt.Sprintf("%02d:%02d", hour, minute)
	}

	return types.GitCommitData{
		TotalCommits:      len(records),
		HourData:          hourData,
		HourDataHalf:      hourDataHalf,
		DayHourCommits:    dhcList,
		WeekdayStats:      weekdayStats,
		DailyFirstCommits: dailyFirstList,
		DailyLatestCommits: dailyLatestList,
		DailyCommitHours:  dailyCommitHours,
	}
}

// buildWorkTimeData 构建工作时间数据
func buildWorkTimeData(records []engine.CommitRecord, schedule *engine.ScheduleEngine) types.WorkTimeData {
	workHourPl := [2]types.TimeCount{
		{Time: "work", Count: 0},
		{Time: "overtime", Count: 0},
	}
	workWeekPl := [2]types.TimeCount{
		{Time: "weekday", Count: 0},
		{Time: "weekend", Count: 0},
	}

	for _, r := range records {
		if r.IsOvertime {
			workHourPl[1].Count++
		} else {
			workHourPl[0].Count++
		}

		if schedule.IsRestDay(r.Time) {
			workWeekPl[1].Count++
		} else {
			workWeekPl[0].Count++
		}
	}

	return types.WorkTimeData{
		WorkHourPl: workHourPl,
		WorkWeekPl: workWeekPl,
	}
}

// detectWorkTime 检测工作时间
func detectWorkTime(records []engine.CommitRecord) types.WorkTimeDetectionResult {
	result := types.WorkTimeDetectionResult{
		StartHour:  9.5,
		EndHour:    18.5,
		Confidence: "medium",
	}

	// 收集每日首次提交
	var firstCommits []int
	dayFirst := make(map[string]int)

	for _, r := range records {
		dateStr := r.Time.Format("2006-01-02")
		minutes := r.Time.Hour()*60 + r.Time.Minute()

		if _, exists := dayFirst[dateStr]; !exists {
			dayFirst[dateStr] = minutes
		} else if minutes < dayFirst[dateStr] {
			dayFirst[dateStr] = minutes
		}
	}

	for _, minutes := range dayFirst {
		if minutes >= 4*60 && minutes <= 14*60 {
			firstCommits = append(firstCommits, minutes)
		}
	}

	if len(firstCommits) >= 2 {
		// 排序取 10-20% 分位
		for i := 0; i < len(firstCommits)-1; i++ {
			for j := i + 1; j < len(firstCommits); j++ {
				if firstCommits[i] > firstCommits[j] {
					firstCommits[i], firstCommits[j] = firstCommits[j], firstCommits[i]
				}
			}
		}

		idx := len(firstCommits) * 15 / 100
		if idx >= len(firstCommits) {
			idx = len(firstCommits) - 1
		}

		startMinutes := firstCommits[idx]
		startHour := float64(startMinutes) / 60.0
		startHour = float64(int(startHour*2)) / 2.0

		if startHour >= 4 {
			result.StartHour = startHour
		}

		// 估算下班时间
		result.EndHour = result.StartHour + 9
	}

	return result
}

// calculateWorkIntensity 计算工作强度指数
func calculateWorkIntensity(data types.WorkTimeData) types.WorkIntensityResult {
	y := data.WorkHourPl[0].Count
	x := data.WorkHourPl[1].Count
	m := data.WorkWeekPl[0].Count
	n := data.WorkWeekPl[1].Count

	var overTimeAmendCount int
	if m+n > 0 {
		overTimeAmendCount = x + (y*n)/(m+n)
	} else {
		overTimeAmendCount = x
	}

	totalCount := y + x
	overTimeRadio := 0
	if totalCount > 0 {
		overTimeRadio = overTimeAmendCount * 100 / totalCount
	}

	index := overTimeRadio * 3

	indexStr := "非常健康，是理想的项目情况"
	if index > 21 {
		indexStr = "很健康，加班非常少"
	}
	if index > 48 {
		indexStr = "还行，偶尔加班，能接受"
	}
	if index > 63 {
		indexStr = "较差，加班文化比较严重"
	}
	if index > 100 {
		indexStr = "很差，接近 996 的程度"
	}
	if index > 130 {
		indexStr = "加班文化非常严重，福报已经修满了"
	}

	return types.WorkIntensityResult{
		Index:         index,
		IndexStr:      indexStr,
		OverTimeRadio: overTimeRadio,
	}
}

// analyzeWeekdayOvertime 分析工作日加班
func analyzeWeekdayOvertime(records []engine.CommitRecord, workTime types.WorkTimeDetectionResult) types.WeekdayOvertimeDistribution {
	counts := map[string]int{
		"monday": 0, "tuesday": 0, "wednesday": 0, "thursday": 0, "friday": 0,
	}
	endHour := int(workTime.EndHour)

	for _, r := range records {
		weekday := r.Time.Weekday()
		if weekday >= 1 && weekday <= 5 && r.Time.Hour() >= endHour {
			days := []string{"", "monday", "tuesday", "wednesday", "thursday", "friday"}
			counts[days[weekday]]++
		}
	}

	peakDay := "周一"
	peakCount := counts["monday"]
	dayNames := map[string]string{
		"monday": "周一", "tuesday": "周二", "wednesday": "周三",
		"thursday": "周四", "friday": "周五",
	}

	for day, count := range counts {
		if count > peakCount {
			peakCount = count
			peakDay = dayNames[day]
		}
	}

	return types.WeekdayOvertimeDistribution{
		Monday:    counts["monday"],
		Tuesday:   counts["tuesday"],
		Wednesday: counts["wednesday"],
		Thursday:  counts["thursday"],
		Friday:    counts["friday"],
		PeakDay:   peakDay,
		PeakCount: peakCount,
	}
}

// analyzeWeekendOvertime 分析周末加班
func analyzeWeekendOvertime(records []engine.CommitRecord, schedule *engine.ScheduleEngine) types.WeekendOvertimeDistribution {
	dayHours := make(map[string]map[int]bool)

	for _, r := range records {
		if r.IsWeekend {
			dateStr := r.Time.Format("2006-01-02")
			if _, exists := dayHours[dateStr]; !exists {
				dayHours[dateStr] = make(map[int]bool)
			}
			dayHours[dateStr][r.Time.Hour()] = true
		}
	}

	var saturdayDays, sundayDays, realOvertimeDays int
	for dateStr, hours := range dayHours {
		date, _ := time.Parse("2006-01-02", dateStr)
		weekday := date.Weekday()

		if len(hours) >= 3 {
			realOvertimeDays++
		}

		if weekday == time.Saturday {
			saturdayDays++
		} else if weekday == time.Sunday {
			sundayDays++
		}
	}

	return types.WeekendOvertimeDistribution{
		SaturdayDays:     saturdayDays,
		SundayDays:       sundayDays,
		RealOvertimeDays: realOvertimeDays,
	}
}

// analyzeLateNight 分析深夜加班
func analyzeLateNight(records []engine.CommitRecord, workTime types.WorkTimeDetectionResult) types.LateNightAnalysis {
	endHour := int(workTime.EndHour)
	dayLatest := make(map[string]int)

	for _, r := range records {
		dateStr := r.Time.Format("2006-01-02")
		minutes := r.Time.Hour()*60 + r.Time.Minute()

		if _, exists := dayLatest[dateStr]; !exists {
			dayLatest[dateStr] = minutes
		} else if minutes > dayLatest[dateStr] {
			dayLatest[dateStr] = minutes
		}
	}

	var evening, lateNight, midnight, dawn, midnightDays int
	midnightDaySet := make(map[string]bool)

	for _, minutes := range dayLatest {
		hour := minutes / 60

		if hour >= endHour && hour < 21 {
			evening++
		} else if hour >= 21 && hour < 23 {
			lateNight++
		} else if hour >= 23 || hour < 6 {
			midnight++
			if hour >= 23 || hour < 6 {
				midnightDaySet[fmt.Sprintf("%d", minutes)] = true
			}
		}
	}

	midnightDays = len(midnightDaySet)

	return types.LateNightAnalysis{
		Evening:      evening,
		LateNight:    lateNight,
		Midnight:     midnight,
		Dawn:         dawn,
		MidnightDays: midnightDays,
	}
}

// exportResult 导出结果
func exportResult(result types.AnalysisResult, format, output string) error {
	switch format {
	case "json":
		return printer.ExportJSON(result, output)
	case "csv":
		return printer.ExportCSV(result, output)
	default:
		return fmt.Errorf("不支持的导出格式：%s", format)
	}
}

// 辅助函数
func isGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func getCurrentUserEmail(repoPath string) (string, error) {
	collector, err := collector.NewGitCollector(repoPath)
	if err != nil {
		return "", err
	}
	return collector.GetCurrentEmail()
}

func parseTimeRange(year, since, until string, allTime bool) (time.Time, time.Time) {
	var timeSince, timeUntil time.Time

	if allTime {
		return timeSince, timeUntil
	}

	if year != "" {
		if len(year) == 4 {
			timeSince, _ = time.Parse("2006-01-02", year+"-01-01")
			timeUntil, _ = time.Parse("2006-01-02", year+"-12-31")
		} else if strings.Contains(year, "-") {
			parts := strings.Split(year, "-")
			if len(parts) == 2 {
				timeSince, _ = time.Parse("2006-01-02", parts[0]+"-01-01")
				timeUntil, _ = time.Parse("2006-01-02", parts[1]+"-12-31")
			}
		}
		return timeSince, timeUntil
	}

	if since != "" {
		timeSince, _ = time.Parse("2006-01-02", since)
	}
	if until != "" {
		timeUntil, _ = time.Parse("2006-01-02", until)
	}

	if timeSince.IsZero() && timeUntil.IsZero() {
		// 默认最近一年
		timeUntil = time.Now()
		timeSince = time.Now().AddDate(-1, 0, 0)
	}

	return timeSince, timeUntil
}
