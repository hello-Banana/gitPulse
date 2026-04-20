package types

// TimeCount 时间点的提交数量
type TimeCount struct {
	Time  string `json:"time"`  // 时间点 (如 "09:00" 或 "09:30")
	Count int    `json:"count"` // 提交数量
}

// DayHourCommit 按星期几和小时的提交数据
type DayHourCommit struct {
	Weekday int    `json:"weekday"` // 1-7 (周一到周日)
	Hour    int    `json:"hour"`    // 0-23
	Count   int    `json:"count"`   // 提交数量
}

// DailyCommitHours 每日提交小时数
type DailyCommitHours struct {
	Date  string       `json:"date"`      // 日期 (YYYY-MM-DD)
	Hours map[int]bool `json:"hours"`     // 有提交的小时集合
	Size  int          `json:"size"`      // 提交小时跨度
}

// DailyFirstCommit 每日首次提交
type DailyFirstCommit struct {
	Date              string `json:"date"`                // 日期
	MinutesFromMidnight int  `json:"minutes_from_midnight"` // 距离午夜的分钟数
}

// DailyLatestCommit 每日最晚提交
type DailyLatestCommit struct {
	Date              string `json:"date"`                // 日期
	MinutesFromMidnight int  `json:"minutes_from_midnight"` // 距离午夜的分钟数
}

// WorkTimeData 工作时间数据
type WorkTimeData struct {
	HourData     []TimeCount  `json:"hour_data"`      // 小时分布数据
	WorkHourPl   [2]TimeCount `json:"work_hour_pl"`   // 工作时间分布 [0]=正常，[1]=加班
	WorkWeekPl   [2]TimeCount `json:"work_week_pl"`   // 工作周分布 [0]=工作日，[1]=周末
	HourDataHalf []TimeCount  `json:"hour_data_half"` // 半小时粒度数据 (可选)
}

// WorkIntensityResult 工作强度指数结果
type WorkIntensityResult struct {
	Index         int    `json:"index"`           // 工作强度指数
	IndexStr      string `json:"index_str"`       // 工作强度指数描述
	OverTimeRadio int    `json:"over_time_radio"` // 加班比例
}

// WorkTimeDetectionResult 工作时间识别结果
type WorkTimeDetectionResult struct {
	StartHour  float64 `json:"start_hour"`  // 上班时间
	EndHour    float64 `json:"end_hour"`    // 下班时间
	Confidence string  `json:"confidence"`  // 置信度 (high/medium/low)
}

// WeekdayOvertimeDistribution 工作日加班分布
type WeekdayOvertimeDistribution struct {
	Monday    int    `json:"monday"`
	Tuesday   int    `json:"tuesday"`
	Wednesday int    `json:"wednesday"`
	Thursday  int    `json:"thursday"`
	Friday    int    `json:"friday"`
	PeakDay   string `json:"peak_day"`   // 加班最多的一天
	PeakCount int    `json:"peak_count"` // 加班最多的数量
}

// WeekendOvertimeDistribution 周末加班分布
type WeekendOvertimeDistribution struct {
	SaturdayDays     int `json:"saturday_days"`   // 周六天数
	SundayDays       int `json:"sunday_days"`     // 周日天数
	CasualFixDays    int `json:"casual_fix_days"` // 临时修复天数
	RealOvertimeDays int `json:"real_overtime_days"` // 真正加班天数
}

// LateNightAnalysis 深夜加班分析
type LateNightAnalysis struct {
	Evening       int     `json:"evening"`         // 下班后 -21:00 的天数
	LateNight     int     `json:"late_night"`      // 21:00-23:00 的天数
	Midnight      int     `json:"midnight"`        // 23:00-02:00 的天数
	Dawn          int     `json:"dawn"`            // 02:00-06:00 的天数
	MidnightDays  int     `json:"midnight_days"`   // 有深夜/凌晨提交的天数
	TotalWorkDays int     `json:"total_work_days"` // 总工作日天数
	MidnightRate  float64 `json:"midnight_rate"`   // 深夜加班率 (%)
	TotalWeeks    int     `json:"total_weeks"`     // 总周数
	TotalMonths   int     `json:"total_months"`    // 总月数
}

// GitCommitData Git 提交数据
type GitCommitData struct {
	TotalCommits     int                `json:"total_commits"`      // 总提交数
	HourData         []TimeCount        `json:"hour_data"`          // 小时分布 (24 点)
	HourDataHalf     []TimeCount        `json:"hour_data_half"`     // 半小时分布 (48 点)
	DayHourCommits   []DayHourCommit    `json:"day_hour_commits"`   // 按星期几和小时
	WeekdayStats     map[int]int        `json:"weekday_stats"`      // 星期统计
	DailyFirstCommits []DailyFirstCommit  `json:"daily_first_commits"`  // 每日首次提交
	DailyLatestCommits []DailyLatestCommit `json:"daily_latest_commits"` // 每日最晚提交
	DailyCommitHours []DailyCommitHours `json:"daily_commit_hours"` // 每日提交小时
	TimezoneStats    map[string]int     `json:"timezone_stats"`     // 时区统计
	AuthorCommits    map[string]int     `json:"author_commits"`     // 作者提交统计
	Since            string             `json:"since"`              // 开始日期
	Until            string             `json:"until"`              // 结束日期
}

// FatigueInfo 疲劳度信息
type FatigueInfo struct {
	MaxConsecutiveDays int    `json:"max_consecutive_days"`
	Level              string `json:"level"`
	LevelStr           string `json:"level_str"`
	Emoji              string `json:"emoji"`
}

// RhythmInfo 节奏分析信息
type RhythmInfo struct {
	Pattern       string  `json:"pattern"`
	PeakHour      int     `json:"peak_hour"`
	Consistency   float64 `json:"consistency"`
	BurstDetected bool    `json:"burst_detected"`
	Emoji         string  `json:"emoji"`
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	CommitData       GitCommitData                `json:"commit_data"`
	WorkTimeData     WorkTimeData                 `json:"work_time_data"`
	WorkIntensity    WorkIntensityResult          `json:"work_intensity"` // 工作强度指数
	WorkTimeDetect   WorkTimeDetectionResult      `json:"work_time_detect"`
	WeekdayOvertime  WeekdayOvertimeDistribution  `json:"weekday_overtime"`
	WeekendOvertime  WeekendOvertimeDistribution  `json:"weekend_overtime"`
	LateNight        LateNightAnalysis            `json:"late_night"`
	Fatigue          FatigueInfo                  `json:"fatigue"`        // 疲劳度
	Rhythm           RhythmInfo                   `json:"rhythm"`         // 节奏分析
	ProjectType      string                       `json:"project_type"`   // 项目类型
	ProjectTypeConf  string                       `json:"project_type_conf"` // 项目类型置信度
	IsCrossTimezone  bool                         `json:"is_cross_timezone"` // 是否跨时区
	TimezoneWarning  string                       `json:"timezone_warning"`  // 时区警告
	RepoPath         string                       `json:"repo_path"`      // 仓库路径
	RepoName         string                       `json:"repo_name"`      // 仓库名称
}

// RepoAnalysis 仓库分析结果 (多仓库模式)
type RepoAnalysis struct {
	RepoPath string         `json:"repo_path"`
	RepoName string         `json:"repo_name"`
	Result   AnalysisResult `json:"result"`
}

// MultiRepoResult 多仓库分析结果
type MultiRepoResult struct {
	Repos       []RepoAnalysis `json:"repos"`
	MergedResult AnalysisResult `json:"merged_result"`
}

// ExportData 导出数据结构
type ExportData struct {
	ReportTime string         `json:"report_time"`
	Repo       string         `json:"repo"`
	Period     ExportPeriod   `json:"period"`
	Summary    ExportSummary  `json:"summary"`
	Details    ExportDetails  `json:"details"`
}

// ExportPeriod 导出时间段
type ExportPeriod struct {
	Since string `json:"since"`
	Until string `json:"until"`
}

// ExportSummary 导出摘要
type ExportSummary struct {
	TotalCommits       int     `json:"total_commits"`
	OvertimeCommits    int     `json:"overtime_commits"`
	OvertimeRatio      float64 `json:"overtime_ratio"`
	WorkIntensityIndex int     `json:"work_intensity_index"`
	WorkHours          float64 `json:"work_hours"`
}

// ExportDetails 导出详情
type ExportDetails struct {
	WeekdayOvertime  []map[string]interface{} `json:"weekday_overtime"`
	WeekendOvertime  map[string]interface{}   `json:"weekend_overtime"`
	LateNightOvertime map[string]interface{}  `json:"late_night_overtime"`
	Fatigue           map[string]interface{}  `json:"fatigue"`
	Rhythm            map[string]interface{}  `json:"rhythm"`
}
