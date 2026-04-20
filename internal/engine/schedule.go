package engine

import (
	"time"

	"gitpulse/internal/config"
)

// ScheduleEngine 排班引擎
type ScheduleEngine struct {
	config *config.Config
}

// NewScheduleEngine 创建排班引擎
func NewScheduleEngine(cfg *config.Config) *ScheduleEngine {
	return &ScheduleEngine{
		config: cfg,
	}
}

// IsWorkDay 判定某一天是否为工作日
func (s *ScheduleEngine) IsWorkDay(date time.Time) bool {
	// 1. 优先：手动指定的补班日 (调休)
	for _, d := range s.config.Holidays.ExtraWorkdays {
		if parseDateMatch(d, date) {
			return true
		}
	}

	// 2. 优先：手动指定的休息日
	for _, d := range s.config.Holidays.ExtraHolidays {
		if parseDateMatch(d, date) {
			return false
		}
	}

	// 3. 法定节假日 (简化版：使用内置数据)
	if isNationalHoliday(date, s.config) {
		return false
	}

	// 4. 特殊时期 (报税期、冲刺期等)
	if period := s.getSpecialPeriod(date); period != nil {
		return s.checkPeriodSchedule(date, period)
	}

	// 5. 基础模式判定
	return s.checkNormalSchedule(date)
}

// IsRestDay 判定某一天是否为休息日
func (s *ScheduleEngine) IsRestDay(date time.Time) bool {
	return !s.IsWorkDay(date)
}

// checkNormalSchedule 基础模式判定
func (s *ScheduleEngine) checkNormalSchedule(date time.Time) bool {
	weekday := date.Weekday()

	switch s.config.WorkSettings.Mode {
	case config.ModeDoubleRest:
		// 双休：周六、周日休息
		return weekday != time.Saturday && weekday != time.Sunday

	case config.ModeFixed6Days:
		// 单休：仅周日休息
		return weekday != time.Sunday

	case config.ModeBigSmallWeek:
		// 大小周
		return s.checkBigSmallWeek(date)

	case config.ModeCustom:
		// 自定义模式（暂不实现）
		return weekday != time.Saturday && weekday != time.Sunday

	default:
		return weekday != time.Saturday && weekday != time.Sunday
	}
}

// checkBigSmallWeek 大小周判定
func (s *ScheduleEngine) checkBigSmallWeek(date time.Time) bool {
	if !s.config.BigSmallWeek.Enabled {
		// 未启用大小周，降级为双休
		weekday := date.Weekday()
		return weekday != time.Saturday && weekday != time.Sunday
	}

	// 解析参考日期
	refDate, err := time.Parse("2006-01-02", s.config.BigSmallWeek.ReferenceMonday)
	if err != nil {
		// 参考日期无效，降级为双休
		weekday := date.Weekday()
		return weekday != time.Saturday && weekday != time.Sunday
	}

	// 计算与参考日期的周数差
	refMonday := refDate
	if refMonday.Weekday() != time.Monday {
		// 调整为最近的周一
		daysSinceMonday := int(refMonday.Weekday() - time.Monday)
		refMonday = refMonday.AddDate(0, 0, -daysSinceMonday)
	}

	// 计算目标日期所在周的周一
	targetMonday := date
	daysSinceMonday := int(targetMonday.Weekday() - time.Monday)
	if targetMonday.Weekday() < time.Monday {
		daysSinceMonday = int(targetMonday.Weekday()) + 6
	}
	targetMonday = date.AddDate(0, 0, -daysSinceMonday)

	// 计算周数差
	weeksDiff := int(targetMonday.Sub(refMonday).Hours() / 24 / 7)

	// 偶数周为大周（双休），奇数周为小周（单休）
	isBigWeek := weeksDiff%2 == 0
	weekday := date.Weekday()

	if isBigWeek {
		// 大周：周六、周日休息
		return weekday != time.Saturday && weekday != time.Sunday
	}
	// 小周：仅周日休息
	return weekday != time.Sunday
}

// checkPeriodSchedule 特殊时期判定
func (s *ScheduleEngine) checkPeriodSchedule(date time.Time, period *config.SpecialPeriod) bool {
	switch period.WorkModeOverride {
	case config.ModeFixed6Days:
		return date.Weekday() != time.Sunday
	case config.ModeDoubleRest:
		return date.Weekday() != time.Saturday && date.Weekday() != time.Sunday
	case config.ModeBigSmallWeek:
		// 特殊时期内的大小周
		return s.checkBigSmallWeek(date)
	default:
		return date.Weekday() != time.Saturday && date.Weekday() != time.Sunday
	}
}

// getSpecialPeriod 获取日期所在的特殊时期
func (s *ScheduleEngine) getSpecialPeriod(date time.Time) *config.SpecialPeriod {
	// 检查报税期
	if s.config.TaxPeriod.Enabled {
		if s.isInTaxPeriod(date) {
			return &config.SpecialPeriod{
				Name:            "TaxPeriod",
				WorkModeOverride: s.config.TaxPeriod.WorkModeOverride,
			}
		}
	}

	// 检查自定义特殊时期
	for i := range s.config.SpecialPeriods {
		period := &s.config.SpecialPeriods[i]
		start, err1 := time.Parse("2006-01-02", period.Start)
		end, err2 := time.Parse("2006-01-02", period.End)
		if err1 != nil || err2 != nil {
			continue
		}

		if (date.Equal(start) || date.After(start)) && (date.Equal(end) || date.Before(end)) {
			return period
		}
	}

	return nil
}

// isInTaxPeriod 是否在报税期内
func (s *ScheduleEngine) isInTaxPeriod(date time.Time) bool {
	day := date.Day()
	month := int(date.Month())

	// 检查月份
	if len(s.config.TaxPeriod.Months) > 0 {
		inMonth := false
		for _, m := range s.config.TaxPeriod.Months {
			if month == m {
				inMonth = true
				break
			}
		}
		if !inMonth {
			return false
		}
	}

	// 检查日期范围
	if len(s.config.TaxPeriod.MonthlyRange) == 2 {
		return day >= s.config.TaxPeriod.MonthlyRange[0] && day <= s.config.TaxPeriod.MonthlyRange[1]
	}

	return false
}

// IsOvertime 判定是否为加班时间
func (s *ScheduleEngine) IsOvertime(commitTime time.Time) bool {
	// 首先判断是否为休息日
	if s.IsRestDay(commitTime) {
		return true
	}

	// 解析标准工作时间
	startTime, err := time.Parse("15:04", s.config.WorkSettings.StandardHours.Start)
	if err != nil {
		startTime, _ = time.Parse("15:04", "09:30")
	}

	endTime, err := time.Parse("15:04", s.config.WorkSettings.StandardHours.End)
	if err != nil {
		endTime, _ = time.Parse("15:04", "18:30")
	}

	// 解析宽限期
	graceMinutes := s.config.WorkSettings.GracePeriod
	if graceMinutes < 0 {
		graceMinutes = 0
	}

	// 创建同一日期的工作时间
	commitHourMin := commitTime.Hour()*60 + commitTime.Minute()
	startHourMin := startTime.Hour()*60 + startTime.Minute()
	endHourMin := endTime.Hour()*60 + endTime.Minute()

	// 考虑宽限期
	startWithGrace := startHourMin - graceMinutes
	endWithGrace := endHourMin + graceMinutes

	// 判定是否为加班
	return commitHourMin < startWithGrace || commitHourMin > endWithGrace
}

// IsLateNight 判定是否为深夜提交
func (s *ScheduleEngine) IsLateNight(commitTime time.Time) bool {
	thresholdTime, err := time.Parse("15:04", s.config.WorkSettings.OvertimeThreshold)
	if err != nil {
		thresholdTime, _ = time.Parse("15:04", "21:00")
	}

	commitHourMin := commitTime.Hour()*60 + commitTime.Minute()
	thresholdHourMin := thresholdTime.Hour()*60 + thresholdTime.Minute()

	return commitHourMin >= thresholdHourMin
}

// parseDateMatch 解析日期并匹配
func parseDateMatch(dateStr string, target time.Time) bool {
	// 支持完整日期 "2006-01-02"
	if d, err := time.Parse("2006-01-02", dateStr); err == nil {
		return d.Year() == target.Year() && d.Month() == target.Month() && d.Day() == target.Day()
	}

	// 支持月 - 日 "01-02"
	if len(dateStr) == 5 {
		if d, err := time.Parse("2006-"+dateStr, "2006-01-02"); err == nil {
			return d.Month() == target.Month() && d.Day() == target.Day()
		}
	}

	return false
}

// isNationalHoliday 判定是否为法定节假日（简化版）
func isNationalHoliday(date time.Time, cfg *config.Config) bool {
	// 简化的节假日判断（实际需要完整数据）
	holidays := map[string]bool{
		// 元旦
		"01-01": true,
		// 春节（简化，实际需要每年调整）
		"01-21": true,
		"01-22": true,
		"01-23": true,
		"01-24": true,
		"01-25": true,
		"01-26": true,
		"01-27": true,
		// 清明节
		"04-04": true,
		"04-05": true,
		// 劳动节
		"05-01": true,
		// 端午节
		"06-10": true,
		// 中秋节
		"09-17": true,
		// 国庆节
		"10-01": true,
		"10-02": true,
		"10-03": true,
		"10-04": true,
		"10-05": true,
		"10-06": true,
		"10-07": true,
	}

	key := date.Format("01-02")
	return holidays[key]
}

// GetWorkHours 获取标准工作时长（小时）
func (s *ScheduleEngine) GetWorkHours() float64 {
	startTime, err := time.Parse("15:04", s.config.WorkSettings.StandardHours.Start)
	if err != nil {
		startTime, _ = time.Parse("15:04", "09:30")
	}

	endTime, err := time.Parse("15:04", s.config.WorkSettings.StandardHours.End)
	if err != nil {
		endTime, _ = time.Parse("15:04", "18:30")
	}

	return endTime.Sub(startTime).Hours()
}
