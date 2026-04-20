package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HolidayAPI 节假日 API 客户端
type HolidayAPI struct {
	client  *http.Client
	cache   map[int]*HolidayYear
	timeout time.Duration
}

// HolidayYear 年度节假日数据
type HolidayYear struct {
	Year    int       `json:"year"`
	Holidays []Holiday `json:"holidays"`
}

// Holiday 单个节假日
type Holiday struct {
	Name    string   `json:"name"`     // 节假日名称
	Date    string   `json:"date"`     // 日期 (YYYY-MM-DD)
	Days    int      `json:"days"`     // 放假天数
	RestDays []string `json:"rest"`   // 休息日 (YYYY-MM-DD)
	WorkDays []string `json:"work"`   // 补班日 (YYYY-MM-DD)
}

// NewHolidayAPI 创建节假日 API 客户端
func NewHolidayAPI() *HolidayAPI {
	return &HolidayAPI{
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		cache: make(map[int]*HolidayYear),
	}
}

// FetchHoliday 获取指定年份的节假日数据
func (h *HolidayAPI) FetchHoliday(year int) (*HolidayYear, error) {
	// 检查缓存
	if cached, ok := h.cache[year]; ok {
		return cached, nil
	}

	// 使用 timor.tech API
	url := fmt.Sprintf("https://timor.tech/api/holiday/year/%d", year)

	resp, err := h.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取节假日数据失败：%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态码：%d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败：%v", err)
	}

	// 解析 API 响应
	var apiResp struct {
		Code   int `json:"code"`
		Holiday struct {
			Name   string `json:"name"`
			Date   string `json:"date"`
			Work   string `json:"work"`
			Rest   string `json:"rest"`
			Target string `json:"target"`
		} `json:"holiday"`
		Weeks map[string]string `json:"weeks"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败：%v", err)
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("API 返回错误码：%d", apiResp.Code)
	}

	// 转换为内部格式
	hy := &HolidayYear{
		Year: year,
		Holidays: []Holiday{
			{
				Name:   apiResp.Holiday.Name,
				Date:   apiResp.Holiday.Date,
				WorkDays: []string{apiResp.Holiday.Work},
				RestDays: []string{apiResp.Holiday.Rest},
			},
		},
	}

	h.cache[year] = hy
	return hy, nil
}

// FetchHolidayRange 获取年份范围的节假日数据
func (h *HolidayAPI) FetchHolidayRange(startYear, endYear int) ([]*HolidayYear, error) {
	var results []*HolidayYear

	for year := startYear; year <= endYear; year++ {
		hy, err := h.FetchHoliday(year)
		if err != nil {
			// 继续尝试下一年
			continue
		}
		results = append(results, hy)
	}

	return results, nil
}

// IsHoliday 判断是否为节假日
func (h *HolidayAPI) IsHoliday(date time.Time) (bool, string) {
	hy, err := h.FetchHoliday(date.Year())
	if err != nil {
		return false, ""
	}

	dateStr := date.Format("2006-01-02")

	for _, holiday := range hy.Holidays {
		// 检查是否为休息日
		for _, restDay := range holiday.RestDays {
			if restDay == dateStr {
				return true, holiday.Name
			}
		}
	}

	return false, ""
}

// IsWorkday 判断是否为调休补班日
func (h *HolidayAPI) IsWorkday(date time.Time) bool {
	hy, err := h.FetchHoliday(date.Year())
	if err != nil {
		return false
	}

	dateStr := date.Format("2006-01-02")

	for _, holiday := range hy.Holidays {
		for _, workDay := range holiday.WorkDays {
			if workDay == dateStr {
				return true
			}
		}
	}

	return false
}

// GetHolidayName 获取节假日名称
func (h *HolidayAPI) GetHolidayName(date time.Time) string {
	hy, err := h.FetchHoliday(date.Year())
	if err != nil {
		return ""
	}

	dateStr := date.Format("2006-01-02")

	for _, holiday := range hy.Holidays {
		for _, restDay := range holiday.RestDays {
			if restDay == dateStr {
				return holiday.Name
			}
		}
	}

	return ""
}

// GetRestDays 获取某年的所有休息日
func (h *HolidayAPI) GetRestDays(year int) []string {
	hy, err := h.FetchHoliday(year)
	if err != nil {
		return nil
	}

	var restDays []string
	for _, holiday := range hy.Holidays {
		restDays = append(restDays, holiday.RestDays...)
	}

	return restDays
}

// GetWorkDays 获取某年的所有补班日
func (h *HolidayAPI) GetWorkDays(year int) []string {
	hy, err := h.FetchHoliday(year)
	if err != nil {
		return nil
	}

	var workDays []string
	for _, holiday := range hy.Holidays {
		workDays = append(workDays, holiday.WorkDays...)
	}

	return workDays
}

// 内置节假日数据（降级方案）
var builtinHolidays = map[string]string{
	// 元旦
	"01-01": "元旦",
	// 春节 (2026 年)
	"02-17": "春节",
	"02-18": "春节",
	"02-19": "春节",
	"02-20": "春节",
	"02-21": "春节",
	"02-22": "春节",
	"02-23": "春节",
	// 清明节
	"04-05": "清明节",
	"04-06": "清明节",
	// 劳动节
	"05-01": "劳动节",
	"05-02": "劳动节",
	"05-03": "劳动节",
	"05-04": "劳动节",
	"05-05": "劳动节",
	// 端午节
	"06-19": "端午节",
	"06-20": "端午节",
	"06-21": "端午节",
	// 中秋节
	"09-25": "中秋节",
	"09-26": "中秋节",
	"09-27": "中秋节",
	// 国庆节
	"10-01": "国庆节",
	"10-02": "国庆节",
	"10-03": "国庆节",
	"10-04": "国庆节",
	"10-05": "国庆节",
	"10-06": "国庆节",
	"10-07": "国庆节",
	"10-08": "国庆节",
}

// IsBuiltinHoliday 判断是否为内置节假日
func IsBuiltinHoliday(date time.Time) (bool, string) {
	key := date.Format("01-02")
	name, ok := builtinHolidays[key]
	return ok, name
}
