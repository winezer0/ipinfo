package utils

import (
	"encoding/json"
)

// ToJSON 将任意 map 转换为格式化的 JSON 字符串（用于输出）
func ToJSON(v interface{}) string {
	return string(ToJSONBytes(v))
}

// ToJSONBytes  将任意 map 转换为格式化的 JSON 字符串（用于输出）
func ToJSONBytes(v interface{}) []byte {
	data, _ := json.MarshalIndent(v, "", "  ")
	return data
}
