package wry

import (
	"github.com/winezer0/ipinfo/pkg/iplocate"
	"strings"
)

// ParseWryResultToIPLocate 解析wry.Result为IPLocate
// wry的Country字段格式通常为: 国家 省份 城市 运营商/其他信息
func ParseWryResultToIPLocate(result *iplocate.IPLocate, wryResult *Result) *iplocate.IPLocate {
	if wryResult == nil {
		return result
	}

	country := FormatLocationResult(wryResult.Country)
	area := FormatLocationResult(wryResult.Area)

	result.Location = country
	if area != "" {
		result.Location = result.Location + " " + area
		result.Location = strings.TrimSpace(result.Location)
	}

	if country != "" {
		parts := strings.Fields(country)
		if len(parts) >= 1 {
			result.Country = parts[0]
		}
		if len(parts) >= 2 {
			result.Province = parts[1]
		}
		if len(parts) >= 3 {
			result.City = parts[2]
		}
	}

	if area != "" {
		result.ISP = area
	}

	return result
}

// FormatLocationResult 清理和格式化地理位置结果
func FormatLocationResult(country string) string {
	if country == "" {
		return ""
	}

	// 替换特殊字符
	result := strings.ReplaceAll(country, "–", " ")
	result = strings.ReplaceAll(result, "\t", " ")

	// 去除首尾空格
	result = strings.TrimSpace(result)

	return result
}
