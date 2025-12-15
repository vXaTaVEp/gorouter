package gorouter

import (
	"encoding/json"
	"fmt"
)

// PrettyJSON 将任何对象转换为格式化的JSON字符串
func PrettyJSON(v interface{}) string {
	// bytes, err := json.MarshalIndent(v, "", "  ")
	bytes, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("failed to marshal json: %v", err)
	}
	return string(bytes)
}

// ParseJSON 将JSON字符串解析为map
func ParseJSON(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	return result, err
}
