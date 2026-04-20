# gitPulse

[![Go Report Card](https://goreportcard.com/badge/github.com/hello-Banana/gitPulse)](https://goreportcard.com/report/github.com/hello-Banana/gitPulse)
[![Go Version](https://img.shields.io/github/go-mod/go-version/hello-Banana/gitPulse)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**gitPulse** 是一款 Git 仓库工作强度分析工具，通过统计 commit 时间分布，智能识别团队的工作模式和加班文化。

> 🎯 **核心理念**：用数据了解团队节奏，拒绝无效内卷

---

## 🚀 快速开始

### 安装

```bash
# 方式 1: 从源码编译
git clone https://github.com/hello-Banana/gitPulse.git
cd gitPulse
go build -o gitpulse ./cmd/gitpulse

# 方式 2: 使用 go install
go install github.com/hello-Banana/gitPulse/cmd/gitpulse@latest
```

### 基础使用

```bash
# 分析当前仓库
gitpulse

# 分析指定仓库
gitpulse /path/to/repo

# 分析 2025 年数据
gitpulse -y 2025

# 分析全部历史
gitpulse --all-time

# 只看自己的提交
gitpulse --self
```

---

## 📊 输出示例

```
🔍 分析仓库：/workspace/my-project
📅 时间范围：2025-01-01 至 2025-12-31

╔══════════════════╤═════════════════════════════════════════════════════════╗
║ 996 指数           │ 75.0                                                      ║
╟──────────────────┼─────────────────────────────────────────────────────────╢
║ 整体评价           │ 较差，加班文化比较严重                                                    ║
╟──────────────────┼─────────────────────────────────────────────────────────╢
║ 加班比例           │ 35.0%                                                   ║
╟──────────────────┼─────────────────────────────────────────────────────────╢
║ 总提交数           │ 1250                                                    ║
╚══════════════════╧═════════════════════════════════════════════════════════╝

📋 详细分析:
  ⚠️  较差，加班文化比较严重（加班比例 35.0%）
  ⚠️  工作日加班频繁，周二是加班高峰（120 次提交）
  🌃 存在深夜加班情况（25 天），需注意休息

⌛ 工作时间推测:
  上班时间：09:30  下班时间：20:00
```

---

## 🛠️ 命令参数

### 时间范围

| 参数 | 简写 | 说明 | 示例 |
|------|------|------|------|
| `--year` | `-y` | 指定年份或年份范围 | `-y 2025` / `-y 2023-2025` |
| `--since` | `-s` | 自定义开始日期 | `-s 2025-01-01` |
| `--until` | `-u` | 自定义结束日期 | `-u 2025-12-31` |
| `--all-time` | | 分析全部历史 | `--all-time` |

### 作者过滤

| 参数 | 简写 | 说明 | 示例 |
|------|------|------|------|
| `--self` | | 仅统计自己的提交 | `--self` |
| `--author` | `-a` | 指定作者（邮箱或姓名） | `-a user@example.com` |
| `--exclude-author` | `-x` | 排除作者（支持通配符） | `-x bot` |

### 其他过滤

| 参数 | 简写 | 说明 | 示例 |
|------|------|------|------|
| `--branch` | `-b` | 指定分支 | `-b main` |
| `--ignore-msg` | `-m` | 排除匹配正则的提交信息 | `-m "WIP"` |

### 工作制配置

| 参数 | 说明 | 示例 |
|------|------|------|
| `--hours` | 手动指定标准工作时间 | `--hours 9-18` / `--hours 9.5-18.5` |
| `--timezone` | 指定时区进行分析 | `--timezone +0800` |
| `--cn` | 强制开启中国节假日调休模式 | `--cn` |

### 输出选项

| 参数 | 简写 | 说明 | 示例 |
|------|------|------|------|
| `--verbose` | `-v` | 显示详细信息（包含热力图） | `-v` |
| `--export` | | 导出格式（json/csv） | `--export json` |
| `--output` | `-o` | 输出文件路径 | `-o result.json` |
| `--heatmap` | | 显示热力图 | `--heatmap` |
| `--init` | | 初始化配置文件 | `--init` |

---

## 📈 核心指标说明

### 996 指数（工作强度指数）

| 指数范围 | 评价 | 说明 |
|----------|------|------|
| 0-21 | 🟢 非常健康 | 几乎不加班，理想状态 |
| 22-48 | 🟡 健康 | 偶尔加班，可以接受 |
| 49-63 | 🟠 一般 | 有点卷，需注意 |
| 64-100 | 🔴 较差 | 加班文化严重 |
| 100+ | ⚫ 很差 | 接近或超过 996 |

### 疲劳度等级

- 🟢 **健康**：连续加班 ≤ 2 天
- 🟡 **警告**：连续加班 3-5 天
- 🟠 **疲劳**：连续加班 6-9 天
- 🔴 **危险**：连续加班 ≥ 10 天

### 节奏模式

- 📅 **规律型**：提交时间稳定，工作节奏好
- 💥 **爆发型**：某些天集中提交，可能是赶工
- 🎲 **随机型**：提交时间分散，节奏不稳定

---

## 📁 配置文件

运行 `gitpulse --init` 生成 `.git-ot.yaml` 配置文件：

```yaml
# 工作设置
workSettings:
  workMode: "DoubleRest"    # DoubleRest | BigSmallWeek | Fixed6Days | Custom
  standardHours:
    start: "09:30"
    end: "18:30"
  timezone: "+0800"

# 大小周设置
bigSmallWeek:
  enabled: false
  referenceMonday: "2025-01-06"

# 报税期设置（适用于财务项目）
taxPeriod:
  enabled: false
  periodType: "monthly"
  startDay: 1
  endDay: 15

# 特殊时期
specialPeriods:
  - name: "年终冲刺"
    startDate: "2025-12-01"
    endDate: "2025-12-31"
    workMode: "Fixed6Days"

# 节假日
holidays:
  enableCNHoliday: true
  enableAutoFetch: true

# 疲劳度告警
fatigueAlert:
  enabled: true
  consecutiveDays: 5
```

---

## 🎯 典型使用场景

### 1. 入职前背调
```bash
# 分析目标公司的开源项目，了解团队工作强度
gitpulse /path/to/cloned-repo --all-time
```

### 2. 团队健康检查
```bash
# 查看最近一年的加班情况
gitpulse -y 2025 --export json -o report.json
```

### 3. 个人工作复盘
```bash
# 只看自己的提交，回顾工作状态
gitpulse --self -y 2025 -v
```

### 4. 生成报告
```bash
# 导出 JSON 格式用于进一步分析
gitpulse --export json -o team-analysis.json

# 导出 CSV 格式用于 Excel 处理
gitpulse --export csv -o team-analysis.csv
```

---

## ⚠️ 注意事项

1. **数据准确性**：仅统计 commit 提交时间，不包含会议、学习、文档编写等活动
2. **仅供参考**：分析结果不能完全反映实际工作强度，请理性看待
3. **隐私保护**：所有分析在本地完成，不会上传任何数据
4. **使用限制**：请勿将分析结果用于不当用途

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

```bash
# Fork 项目
git clone https://github.com/hello-Banana/gitPulse.git

# 创建功能分支
git checkout -b feature/amazing-feature

# 提交更改
git commit -m "Add amazing feature"

# 推送到分支
git push origin feature/amazing-feature
```

---

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

## 🙏 致谢

- 灵感来源于 [code996](https://github.com/hellodigua/code996)
- 使用 [go-git](https://github.com/go-git/go-git) 进行 Git 仓库解析
- 使用 [cobra](https://github.com/spf13/cobra) 构建 CLI

---

**📦 项目地址**: https://github.com/hello-Banana/gitPulse
