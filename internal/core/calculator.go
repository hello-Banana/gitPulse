package core

import (
	"gitpulse/types"
)

// CalculateWorkIntensityIndex 计算工作强度指数
func CalculateWorkIntensityIndex(data types.WorkTimeData) types.WorkIntensityResult {
	workHourPl := data.WorkHourPl
	workWeekPl := data.WorkWeekPl
	hourData := data.HourData

	// y: 正常工作时间的 commit 数量
	y := workHourPl[0].Count

	// x: 加班时间的 commit 数量
	x := workHourPl[1].Count

	// m: 工作日的 commit 数量
	m := workWeekPl[0].Count

	// n: 周末的 commit 数量
	n := workWeekPl[1].Count

	// 修正后的加班 commit 数量
	// 公式：x + (y * n) / (m + n)
	var overTimeAmendCount int
	if m+n > 0 {
		overTimeAmendCount = int(float64(x) + float64(y*n)/float64(m+n))
	} else {
		overTimeAmendCount = x
	}

	// 总 commit 数
	totalCount := y + x

	// 加班 commit 百分比
	overTimeRadio := 0
	if totalCount > 0 {
		overTimeRadio = int(float64(overTimeAmendCount) / float64(totalCount) * 100)
	}

	// 针对低加班且数据量不足的情况进行特殊处理
	if overTimeRadio == 0 && len(hourData) < 9 {
		overTimeRadio = getUn996Radio(hourData, totalCount)
	}

	// 工作强度指数 = 加班比例 * 3
	indexWork := overTimeRadio * 3

	// 生成分析文字
	indexWorkStr := generateDescription(indexWork)

	return types.WorkIntensityResult{
		Index:      indexWork,
		IndexStr:   indexWorkStr,
		OverTimeRadio: overTimeRadio,
	}
}

// generateDescription 生成工作强度指数分析文字
func generateDescription(index int) string {
	if index <= 0 {
		return "非常健康，是理想的项目情况"
	}
	if index <= 21 {
		return "很健康，加班非常少"
	}
	if index <= 48 {
		return "还行，偶尔加班，能接受"
	}
	if index <= 63 {
		return "较差，加班文化比较严重"
	}
	if index <= 100 {
		return "很差，接近 996 的程度"
	}
	if index <= 130 {
		return "非常差，加班文化严重"
	}
	return "加班文化非常严重，福报已经修满了"
}

// getUn996Radio 计算不加班比例（用于处理工作量较少的项目）
func getUn996Radio(hourData []types.TimeCount, totalCount int) int {
	// 计算每小时平均 commit 数
	averageCommit := float64(totalCount) / float64(len(hourData))

	// 模拟标准工作日（9 小时）的 commit 总数
	mockTotalCount := averageCommit * 9

	// 计算工作饱和度（返回负值）
	radio := int(float64(totalCount)/mockTotalCount*100) - 100

	return radio
}
