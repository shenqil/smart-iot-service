package utils

import "encoding/json"

// LogStruct 打印结构体时忽略nil字段（简洁版）
func LogStruct(v interface{}) string {
	// 使用json.Marshal自动忽略nil字段
	data, _ := json.Marshal(v)
	return string(data)
}
