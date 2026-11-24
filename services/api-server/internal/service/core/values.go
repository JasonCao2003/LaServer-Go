// 通用取值与格式化工具

package core

import (
	"encoding/json"
	"math"
	"strconv"
	"time"
)

// 格式化关系时间
func RelationTime(epoch int64) string {
	return time.Unix(epoch, 0).UTC().Format("2006-01-02T15:04:05.000Z")
}

// 读取布尔值
func BoolValue(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x != "" && x != "false"
	case json.Number:
		n, err := x.Float64()
		return err == nil && n != 0
	case float64:
		return x != 0
	default:
		return false
	}
}

// 读取浮点值
func FloatValue(v any) (float64, bool) {
	switch x := v.(type) {
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case float64:
		return x, true
	case int64:
		return float64(x), true
	case string:
		var f float64
		if err := json.Unmarshal([]byte(x), &f); err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

// 解析整数列表
func IntList(raw string) []int64 {
	var values []json.RawMessage
	if json.Unmarshal([]byte(raw), &values) != nil {
		return []int64{}
	}
	out := []int64{}
	for _, v := range values {
		s := string(v)
		var t string
		if json.Unmarshal(v, &t) == nil {
			s = t
		}
		if n, e := strconv.ParseInt(s, 10, 64); e == nil {
			out = append(out, n)
		}
	}
	return out
}

// 将定义秒值转换为秒
func DefSeconds(v any) int64 {
	var n float64
	switch x := v.(type) {
	case float64:
		n = x
	case json.Number:
		n, _ = x.Float64()
	case int64:
		n = float64(x)
	}
	if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 {
		return 0
	}
	return int64(n * 60)
}

// 转换为对象
func AsObject(value any) Object { out, _ := value.(map[string]any); return out }

// 读取布尔设置
func BoolSetting(value any, fallback bool) bool {
	if v, ok := value.(bool); ok {
		return v
	}
	return fallback
}

// 校验配置数组
func ValidConfiguredArray(value any, requiredField string) bool {
	entries, ok := value.([]any)
	if !ok {
		return false
	}
	for _, entry := range entries {
		obj, ok := entry.(map[string]any)
		if !ok || len(obj) == 0 {
			return false
		}
		if requiredField != "" && Text(obj, requiredField) == "" {
			return false
		}
	}
	return true
}

// 返回当前毫秒时间戳
func NowMillis() int64 { return time.Now().UnixMilli() }

// 格式化浮点数
func strconvFormatFloat(v float64) string {
	encoded, _ := json.Marshal(v)
	return string(encoded)
}

// 转换标量为字符串
func ScalarString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		return strconvFormatFloat(x)
	default:
		return ""
	}
}

// 读取第一个非空字符串
func FirstText(m Object, keys ...string) string {
	for _, key := range keys {
		if value := Text(m, key); value != "" {
			return value
		}
	}
	return ""
}
