# gitPulse 需求规格说明书

## 1. 项目定位

一款基于 Golang 开发的高性能命令行工具（CLI），专门用于解析 Git 仓库提交记录，结合灵活的中国特色工作制配置（如大小周、报税期、调休等），输出多维度的研发加班统计报告。

### 1.1 设计目标

| 目标 | 说明 |
|------|------|
| **高性能** | 纯 Go 实现，支持百万级提交记录秒级分析 |
| **零依赖** | 使用 go-git，无需安装 git 客户端 |
| **灵活配置** | YAML 配置文件 + 命令行参数双重支持 |
| **中国特色** | 原生支持调休、大小周、报税期等场景 |
| **隐私安全** | 纯本地运行，数据不出机器 |

---

## 2. 核心功能需求

### 2.1 灵活的工作制配置引擎 (Scheduling Engine)

#### 2.1.1 基础模式切换

```yaml
work_settings:
  mode: "DoubleRest"  # 可选值见下表
```

| 模式 | 说明 | 休息日 |
|------|------|--------|
| `DoubleRest` | 标准双休 | 周六、周日 |
| `BigSmallWeek` | 大小周 | 单休周的周日 / 双休周的周六日 |
| `Fixed6Days` | 固定单休 (996) | 仅周日 |
| `Custom` | 自定义 | 用户指定工作日集合 |

**大小周配置示例：**
```yaml
work_settings:
  mode: "BigSmallWeek"
  big_week_start: "2026-01-05"  # 第一个大周的起始日期（周一）
  # 系统会自动推算后续大小周规律
```

#### 2.1.2 动态时期配置 (Special Periods)

**报税期模式：**
```yaml
tax_period:
  enabled: true
  monthly_range: [1, 15]           # 每月 1-15 号
  work_mode_override: "Fixed6Days" # 报税期内强制单休
  # 或指定特定月份
  months: [1, 4, 7, 10]            # 季度报税
```

**项目冲刺期：**
```yaml
special_periods:
  - name: "Q4 冲刺"
    start: "2026-10-01"
    end: "2026-12-31"
    work_mode_override: "Fixed6Days"
    label: "🔥"
```

#### 2.1.3 法定节假日与调休

```yaml
holidays:
  auto_sync: true           # 自动同步官方数据
  country: "CN"             # 国家代码
  offline_mode: false       # 纯离线模式（使用内置数据）
  
  # 手动指定补班日（调休产生的工作日）
  extra_workdays: 
    - "2026-01-26"  # 周日补班
    - "2026-02-08"  # 周日补班
  
  # 手动指定休息日（非周末）
  extra_holidays:
    - "2026-01-27"  # 春节调休
```

**内置节假日数据源：**
- 优先：`timor.tech/api/holiday` (免费 API)
- 备选：内置 `holidays_cn.yaml` (每年更新)
- 降级：基础周末判断 (周一~周五工作)

#### 2.1.4 弹性工时设定

```yaml
work_settings:
  standard_hours:
    start: "09:30"
    end: "18:30"
  grace_period: 30  # 宽限期 (分钟)，18:30-19:00 不计为加班
  overtime_threshold: 21:00  # 深夜加班判定线
```

**工时判定规则：**
```
早到：commit_time < standard_start - grace_period
正常：standard_start - grace_period <= commit_time <= standard_end + grace_period
加班：commit_time > standard_end + grace_period
深夜：commit_time >= overtime_threshold
```

---

### 2.2 数据采集与过滤

#### 2.2.1 仓库解析

```go
// 使用 go-git 直接读取 .git 目录
repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{
    DetectDotGit: true,
})
```

#### 2.2.2 过滤参数

| 参数 | 简写 | 说明 | 示例 |
|------|------|------|------|
| `--since` | `-s` | 开始日期 | `2025-01-01` |
| `--until` | `-u` | 结束日期 | `2025-12-31` |
| `--branch` | `-b` | 指定分支 | `main` |
| `--author` | `-a` | 作者过滤 | `bob@example.com` |
| `--exclude-author` | `-x` | 排除作者 | `bot@*` |
| `--ignore-msg` | `-m` | 排除提交信息 | `^Merge` |

#### 2.2.3 多仓库合并分析

```bash
# 同时分析多个仓库，汇总同一开发者的提交
gitpulse multi /path/to/repo1 /path/to/repo2 ~/projects/*

# 从配置文件读取仓库列表
gitpulse multi --from-config .git-ot.yaml
```

**配置文件仓库列表：**
```yaml
repositories:
  - path: ~/work/project-a
    label: "项目 A"
  - path: ~/work/project-b
    label: "项目 B"
  - path: ~/github/open-source
    label: "开源贡献"
    exclude: true  # 不计入加班统计
```

---

### 2.3 统计分析指标

#### 2.3.1 加班频次统计

| 指标 | 判定条件 | 输出 |
|------|----------|------|
| **早到** | commit_time < 09:00 | 次数、最早时间 |
| **晚走** | commit_time > 18:30 | 次数、最晚时间 |
| **深夜提交** | commit_time >= 21:00 | 次数、代码量 |
| **周末提交** | is_rest_day AND commit_count > 0 | 天数、提交数 |
| **节假日提交** | is_holiday AND commit_count > 0 | 次数、详情 |

#### 2.3.2 疲劳度模型

```yaml
fatigue_alert:
  enabled: true
  thresholds:
    consecutive_overtime_days: 5   # 连续加班天数预警
    weekly_overtime_hours: 10      # 周加班时长预警
    monthly_overtime_days: 15      # 月加班天数预警
```

**疲劳等级：**
```
🟢 健康：连续加班 < 3 天
🟡 注意：连续加班 3-5 天
🟠 疲劳：连续加班 6-10 天
🔴 危险：连续加班 > 10 天
```

#### 2.3.3 节奏一致性分析

```go
type RhythmAnalysis struct {
    Pattern       string  // "规律型" | "爆发型" | "随机型"
    PeakHour      int     // 提交高峰时段
    Consistency   float64 // 一致性得分 0-100
    BurstDetected bool    // 是否检测到深夜爆发
}
```

#### 2.3.4 代码量统计

```bash
# 加班时段提交的代码量
git log --since="18:30" --shortstat
```

| 指标 | 说明 |
|------|------|
| 加班提交数 | 非工作时间 commit 数量 |
| 加班代码行 | 加班提交的 +/- 行数 |
| 人均加班代码量 | 团队对比参考 |

---

### 2.4 CLI 交互与输出

#### 2.4.1 配置优先级

```
命令行参数 > .git-ot.yaml > 内置默认值
```

#### 2.4.2 可视化报表

**ASCII 表格输出：**
```
┌─────────────────────────────────────────────────────────────┐
│  gitPulse 工作强度分析报告                                  │
├─────────────────────────────────────────────────────────────┤
│  工作强度指数：177  [████████████████████] 非常严重         │
│  加班比例：59%        总提交数：37                          │
│  工作时间：09:30-18:30 (约 9.0 小时)                          │
├─────────────────────────────────────────────────────────────┤
│  📈 加班情况                                                │
│    工作日加班：9 次 (一=0 二=4 三=2 四=0 五=3)                │
│    周末加班：2 天 (真正加班：1 天)                           │
│    深夜加班：3 天 (25.0%)                                   │
└─────────────────────────────────────────────────────────────┘
```

**24 小时热力图：**
```
     周一  周二  周三  周四  周五  周六  周日
00:00  ░░    ░░    ░░    ░░    ░░    █░    ░░
01:00  ░░    ░░    ░░    ░░    ░░    █░    ░░
...
18:00  ██    ███   ██    ███   ██    ░░    ░░
19:00  ███   ████  ███   ███   ████  ░░    ░░
20:00  ██    ███   ██    ██    ███   ░░    ░░
21:00  █░    ██    █░    █░    ██    ░░    ░░
```

#### 2.4.3 数据导出

```bash
# JSON 格式
gitpulse --export json --output report.json

# CSV 格式 (适合 Excel)
gitpulse --export csv --output report.csv

# 同时导出
gitpulse --export json,csv
```

**JSON 输出示例：**
```json
{
  "report_time": "2026-04-18T14:30:00+08:00",
  "repo": "project-a",
  "period": {
    "since": "2025-01-01",
    "until": "2025-12-31"
  },
  "summary": {
    "total_commits": 350,
    "overtime_commits": 120,
    "overtime_ratio": 34.3,
    "work_intensity_index": 103
  },
  "details": {
    "weekday_overtime": [...],
    "weekend_overtime": [...],
    "late_night_overtime": [...]
  }
}
```

---

## 3. 技术实现

### 3.1 项目结构

```
gitpulse/
├── cmd/
│   └── gitpulse/
│       ├── main.go           # 入口
│       ├── root.go           # 根命令
│       ├── analyze.go        # 分析命令
│       └── multi.go          # 多仓库命令
├── internal/
│   ├── collector/            # 数据采集
│   │   ├── git_collector.go
│   │   └── holiday_collector.go
│   ├── engine/               # 核心引擎
│   │   ├── schedule.go       # 排班引擎
│   │   ├── analyzer.go       # 分析器
│   │   └── fatigue.go        # 疲劳度模型
│   ├── config/               # 配置管理
│   │   ├── config.go
│   │   └── loader.go
│   ├── printer/              # 输出模块
│   │   ├── table.go
│   │   ├── heatmap.go
│   │   └── export.go
│   └── utils/                # 工具函数
│       ├── time.go
│       └── filter.go
├── types/                    # 类型定义
│   └── types.go
├── data/                     # 内置数据
│   └── holidays_cn.yaml
├── .git-ot.yaml              # 配置模板
├── go.mod
└── README.md
```

### 3.2 核心依赖

```go
require (
    github.com/spf13/cobra v1.8.0       // CLI 框架
    github.com/spf13/viper v1.18.0      // 配置管理
    github.com/go-git/go-git/v5 v5.11.0 // Git 解析
    github.com/olekukonko/tablewriter v0.0.5  // 表格输出
    github.com/fatih/color v1.16.0      // 彩色输出
    github.com/rickar/cal/v2 v2.1.21    // 假期计算
)
```

### 3.3 配置文件 Schema

```yaml
# .git-ot.yaml 完整示例

# 工作设置
work_settings:
  standard_hours:
    start: "09:30"
    end: "18:30"
  mode: "DoubleRest"          # 基础模式
  grace_period: 30            # 宽限期 (分钟)
  overtime_threshold: "21:00" # 深夜判定
  
# 大小周配置 (mode=BigSmallWeek 时生效)
big_small_week:
  enabled: false
  reference_monday: "2026-01-05"  # 第一个大周的周一
  
# 报税期配置
tax_period:
  enabled: false
  monthly_range: [1, 15]
  months: [1, 4, 7, 10]
  work_mode_override: "Fixed6Days"
  
# 特殊时期
special_periods: []

# 节假日配置
holidays:
  auto_sync: true
  country: "CN"
  offline_mode: false
  extra_workdays: []
  extra_holidays: []
  
# 疲劳度预警
fatigue_alert:
  enabled: true
  consecutive_days: 5
  weekly_hours: 10
  monthly_days: 15
  
# 仓库配置
repositories:
  - path: "."
    label: "当前仓库"
    exclude: false
    
# 输出配置
output:
  format: "table"       # table|json|csv
  show_heatmap: true
  verbose: false
```

---

## 4. 核心算法

### 4.1 工作日判定链

```go
// IsWorkDay 判定某一天是否为工作日
func IsWorkDay(date time.Time, cfg *Config) bool {
    // 1. 优先：手动指定的补班日 (调休)
    if contains(cfg.Holidays.ExtraWorkdays, date) {
        return true
    }
    
    // 2. 优先：手动指定的休息日
    if contains(cfg.Holidays.ExtraHolidays, date) {
        return false
    }
    
    // 3. 法定节假日
    if isNationalHoliday(date, cfg) {
        return false
    }
    
    // 4. 特殊时期 (报税期、冲刺期等)
    if period := getSpecialPeriod(date, cfg); period != nil {
        return checkPeriodSchedule(date, period)
    }
    
    // 5. 基础模式判定
    return checkNormalSchedule(date, cfg.WorkSettings.Mode)
}
```

### 4.2 大小周推算

```go
// IsBigWeek 判断给定日期所在周是否为大周 (双休)
func IsBigWeek(date time.Time, reference time.Time) bool {
    // 计算与参考日期的周数差
    weeksDiff := int(date.Sub(reference).Hours() / 24 / 7)
    
    // 根据周数差奇偶性判断
    return weeksDiff % 2 == 0  // 偶数周为大周
}

// GetRestDays 获取某日的休息日判定
func GetRestDays(date time.Time, reference time.Time) []time.Weekday {
    if IsBigWeek(date, reference) {
        return []time.Weekday{time.Saturday, time.Sunday}
    }
    return []time.Weekday{time.Sunday}
}
```

### 4.3 疲劳度计算

```go
// CalculateFatigue 计算疲劳度得分
func CalculateFatigue(commits []Commit, cfg *Config) *FatigueResult {
    // 连续加班天数
    maxConsecutive := 0
    currentConsecutive := 0
    
    // 周加班时长
    weeklyHours := make(map[int]float64)
    
    // 遍历 commits 计算
    for _, c := range commits {
        if isOvertime(c.Time, cfg) {
            currentConsecutive++
            if currentConsecutive > maxConsecutive {
                maxConsecutive = currentConsecutive
            }
        } else {
            currentConsecutive = 0
        }
        
        // 累加周加班时长
        _, week := c.Time.ISOWeek()
        weeklyHours[week] += calcOvertimeHours(c.Time, cfg)
    }
    
    // 计算疲劳等级
    level := FatigueLevelHealthy
    if maxConsecutive >= cfg.Fatigue.ConsecutiveDays {
        level = FatigueLevelWarning
    }
    
    return &FatigueResult{
        MaxConsecutiveDays: maxConsecutive,
        MaxWeeklyHours:     max(weeklyHours),
        Level:              level,
    }
}
```

### 4.4 节奏一致性分析

```go
// AnalyzeRhythm 分析提交节奏
func AnalyzeRhythm(commits []Commit) *RhythmAnalysis {
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
            peakHour = h
            peakCount = cnt
        }
    }
    
    // 计算标准差 (一致性)
    variance := calculateVariance(hourDist)
    consistency := 100 - math.Min(100, variance*10)
    
    // 判断模式
    pattern := "规律型"
    if variance > 2.0 {
        pattern = "爆发型"
    } else if variance > 1.0 {
        pattern = "随机型"
    }
    
    return &RhythmAnalysis{
        Pattern:     pattern,
        PeakHour:    peakHour,
        Consistency: consistency,
    }
}
```

---

## 5. 跨平台支持

### 5.1 Windows 适配

```go
// 路径处理
path := filepath.Clean(filepath.Join(baseDir, ".git"))

// 颜色支持 (Windows Terminal)
color.NoColor = false  // 强制启用 ANSI 颜色

// 文件锁 (Windows 独占模式)
file, err := os.OpenFile(path, os.O_RDONLY|os.O_SYNC, 0644)
```

### 5.2 PowerShell 集成

```powershell
# 添加 PowerShell 别名
Set-Alias gitpulse "C:\path\to\gitpulse.exe"

# 添加到 profile
Add-Content $PROFILE "Set-Alias gitpulse 'C:\path\to\gitpulse.exe'"
```

---

## 6. 开发计划

### Phase 1 - 核心功能 (Week 1-2)
- [ ] 基础 CLI 框架搭建
- [ ] go-git 数据采集
- [ ] 基础工作制配置 (双休/单休)
- [ ] 简单统计输出

### Phase 2 - 高级配置 (Week 3-4)
- [ ] 大小周推算
- [ ] 报税期配置
- [ ] 节假日 API 对接
- [ ] YAML 配置加载

### Phase 3 - 分析增强 (Week 5-6)
- [ ] 疲劳度模型
- [ ] 节奏一致性分析
- [ ] 代码量统计
- [ ] 多仓库合并

### Phase 4 - 输出优化 (Week 7-8)
- [ ] ASCII 热力图
- [ ] JSON/CSV 导出
- [ ] 配置模板
- [ ] 文档完善

---

## 7. 测试计划

### 7.1 单元测试

```go
// 工作日判定测试
func TestIsWorkDay(t *testing.T) {
    tests := []struct {
        date   string
        expect bool
    }{
        {"2026-01-01", false},  // 元旦
        {"2026-01-26", true},   // 补班日
        {"2026-01-27", false},  // 春节调休
    }
    // ...
}
```

### 7.2 集成测试

```bash
# 使用真实仓库测试
./gitpulse /path/to/real-repo --all-time --export json
```

---

## 8. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 节假日 API 不稳定 | 无法准确识别调休 | 内置离线数据 + 降级方案 |
| 大型仓库性能问题 | 分析时间长 | 增量采集 + 并发处理 |
| Windows 颜色显示 | 输出乱码 | 检测终端能力 + 降级方案 |
| go-git 兼容性 | 某些仓库无法解析 | 回退到 git 命令执行 |

---

## 附录 A: 内置节假日数据示例

```yaml
# data/holidays_cn.yaml
2026:
  - name: "元旦"
    date: "2026-01-01"
    days: 1
  - name: "春节"
    date: "2026-02-17"
    days: 7
    adjust_workdays:
      - "2026-02-08"  # 周日补班
      - "2026-02-21"  # 周六补班
  - name: "清明节"
    date: "2026-04-05"
    days: 3
  # ...
```

## 附录 B: 命令行参数速查

```bash
# 基础分析
gitpulse                           # 当前仓库，最近一年
gitpulse -y 2025                   # 指定年份
gitpulse --all-time                # 全量历史

# 过滤
gitpulse -a bob@example.com        # 指定作者
gitpulse -s 2025-01-01 -u 2025-06-30  # 时间范围
gitpulse -m "^Merge"               # 排除合并提交

# 输出
gitpulse --export json -o report.json
gitpulse -v                        # 显示详细信息

# 多仓库
gitpulse multi ~/work/*            # 扫描目录
gitpulse multi --from-config       # 从配置读取
```
