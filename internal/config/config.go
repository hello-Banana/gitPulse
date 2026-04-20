package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// WorkMode 工作模式
type WorkMode string

const (
	ModeDoubleRest   WorkMode = "DoubleRest"    // 标准双休
	ModeBigSmallWeek WorkMode = "BigSmallWeek"  // 大小周
	ModeFixed6Days   WorkMode = "Fixed6Days"    // 固定单休 (996)
	ModeCustom       WorkMode = "Custom"        // 自定义
)

// Config 配置结构
type Config struct {
	WorkSettings   WorkSettings   `yaml:"work_settings" mapstructure:"work_settings"`
	BigSmallWeek   BigSmallWeek   `yaml:"big_small_week" mapstructure:"big_small_week"`
	TaxPeriod      TaxPeriod      `yaml:"tax_period" mapstructure:"tax_period"`
	SpecialPeriods []SpecialPeriod `yaml:"special_periods" mapstructure:"special_periods"`
	Holidays       HolidayConfig  `yaml:"holidays" mapstructure:"holidays"`
	FatigueAlert   FatigueAlert   `yaml:"fatigue_alert" mapstructure:"fatigue_alert"`
	Repositories   []RepoConfig   `yaml:"repositories" mapstructure:"repositories"`
	Output         OutputConfig   `yaml:"output" mapstructure:"output"`
}

// WorkSettings 工作设置
type WorkSettings struct {
	StandardHours   TimeRange `yaml:"standard_hours" mapstructure:"standard_hours"`
	Mode            WorkMode  `yaml:"mode" mapstructure:"mode"`
	GracePeriod     int       `yaml:"grace_period" mapstructure:"grace_period"`     // 宽限期 (分钟)
	OvertimeThreshold string  `yaml:"overtime_threshold" mapstructure:"overtime_threshold"` // 深夜加班判定线
}

// TimeRange 时间范围
type TimeRange struct {
	Start string `yaml:"start" mapstructure:"start"`
	End   string `yaml:"end" mapstructure:"end"`
}

// BigSmallWeek 大小周配置
type BigSmallWeek struct {
	Enabled         bool   `yaml:"enabled" mapstructure:"enabled"`
	ReferenceMonday string `yaml:"reference_monday" mapstructure:"reference_monday"`
}

// TaxPeriod 报税期配置
type TaxPeriod struct {
	Enabled          bool     `yaml:"enabled" mapstructure:"enabled"`
	MonthlyRange     []int    `yaml:"monthly_range" mapstructure:"monthly_range"`
	Months           []int    `yaml:"months" mapstructure:"months"`
	WorkModeOverride WorkMode `yaml:"work_mode_override" mapstructure:"work_mode_override"`
}

// SpecialPeriod 特殊时期
type SpecialPeriod struct {
	Name            string   `yaml:"name" mapstructure:"name"`
	Start           string   `yaml:"start" mapstructure:"start"`
	End             string   `yaml:"end" mapstructure:"end"`
	WorkModeOverride WorkMode `yaml:"work_mode_override" mapstructure:"work_mode_override"`
	Label           string   `yaml:"label" mapstructure:"label"`
}

// HolidayConfig 节假日配置
type HolidayConfig struct {
	AutoSync       bool     `yaml:"auto_sync" mapstructure:"auto_sync"`
	Country        string   `yaml:"country" mapstructure:"country"`
	OfflineMode    bool     `yaml:"offline_mode" mapstructure:"offline_mode"`
	ExtraWorkdays  []string `yaml:"extra_workdays" mapstructure:"extra_workdays"`
	ExtraHolidays  []string `yaml:"extra_holidays" mapstructure:"extra_holidays"`
}

// FatigueAlert 疲劳度预警
type FatigueAlert struct {
	Enabled         bool `yaml:"enabled" mapstructure:"enabled"`
	ConsecutiveDays int  `yaml:"consecutive_days" mapstructure:"consecutive_days"`
	WeeklyHours     int  `yaml:"weekly_hours" mapstructure:"weekly_hours"`
	MonthlyDays     int  `yaml:"monthly_days" mapstructure:"monthly_days"`
}

// RepoConfig 仓库配置
type RepoConfig struct {
	Path    string `yaml:"path" mapstructure:"path"`
	Label   string `yaml:"label" mapstructure:"label"`
	Exclude bool   `yaml:"exclude" mapstructure:"exclude"`
}

// OutputConfig 输出配置
type OutputConfig struct {
	Format      string `yaml:"format" mapstructure:"format"`
	ShowHeatmap bool   `yaml:"show_heatmap" mapstructure:"show_heatmap"`
	Verbose     bool   `yaml:"verbose" mapstructure:"verbose"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		WorkSettings: WorkSettings{
			StandardHours: TimeRange{
				Start: "09:30",
				End:   "18:30",
			},
			Mode:            ModeDoubleRest,
			GracePeriod:     30,
			OvertimeThreshold: "21:00",
		},
		Holidays: HolidayConfig{
			AutoSync:    true,
			Country:     "CN",
			OfflineMode: false,
		},
		FatigueAlert: FatigueAlert{
			Enabled:         true,
			ConsecutiveDays: 5,
			WeeklyHours:     10,
			MonthlyDays:     15,
		},
		Output: OutputConfig{
			Format:      "table",
			ShowHeatmap: true,
			Verbose:     false,
		},
		Repositories: []RepoConfig{},
	}
}

// LoadConfig 加载配置
func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// 如果指定了配置文件路径
	if configPath != "" {
		if err := loadFromFile(cfg, configPath); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	// 尝试从当前目录加载
	cwd, err := os.Getwd()
	if err != nil {
		return cfg, nil
	}

	configFiles := []string{
		".git-ot.yaml",
		".git-ot.yml",
		".gitpulse.yaml",
		".gitpulse.yml",
	}

	for _, file := range configFiles {
		path := filepath.Join(cwd, file)
		if _, err := os.Stat(path); err == nil {
			if err := loadFromFile(cfg, path); err != nil {
				return nil, err
			}
			return cfg, nil
		}
	}

	return cfg, nil
}

// loadFromFile 从文件加载配置
func loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return err
	}

	// 使用 viper 支持 mapstructure 标签
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return err
	}

	return nil
}

// ParseTime 解析时间字符串为 time.Time
func ParseTime(timeStr string) (time.Time, error) {
	return time.Parse("15:04", timeStr)
}

// ParseDate 解析日期字符串为 time.Time
func ParseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

// SaveConfig 保存配置到文件
func SaveConfig(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// InitConfig 初始化默认配置文件
func InitConfig(path string) error {
	cfg := DefaultConfig()
	return SaveConfig(cfg, path)
}
