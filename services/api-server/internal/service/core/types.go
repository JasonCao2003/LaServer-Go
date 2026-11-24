// 共享类型与字段工具

package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Object = map[string]any
type object = Object

type Request struct {
	Context                         context.Context
	Path, UserAgent, RemoteIP, Host string
	Secure                          bool
	Header                          http.Header
}

type Error struct {
	Status int
	Body   Object
}

// 返回错误文本
func (e *Error) Error() string { return http.StatusText(e.Status) }

// 构造服务错误响应
func Fail(status int, message string) error { return &Error{status, object{"error": message}} }

// 读取字符串字段
func Text(m object, k string) string { s, _ := m[k].(string); return s }

// 读取整数字段
func Number(m object, k string, def int64) int64 {
	v := m[k]
	var s string
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case json.Number:
		s = x.String()
	case string:
		s = x
	default:
		return def
	}
	n, e := strconv.ParseInt(s, 10, 64)
	if e != nil {
		return def
	}
	return n
}

// 读取整数指针
func Integer(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// 读取字符串指针
func StringValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// 返回请求来源 IP
func RemoteIP(r Request) string { return r.RemoteIP }

// 返回当前时间戳
func Now() int64 { return time.Now().Unix() }

// 校验必需字段
func Require(m object, keys ...string) error {
	var missing []string
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return Fail(400, "Missing required keys: "+strings.Join(missing, ", "))
	}
	return nil
}

// 读取通用整数
func DefInt(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case json.Number:
		v, _ = n.Int64()
	case string:
		parsed, _ := strconv.ParseInt(n, 10, 64)
		return parsed
	}
	if n, ok := v.(int64); ok {
		return n
	}
	return 0
}

// 生成随机令牌
func Token(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// 生成 UUID
func UUID() (string, error) {
	s, e := Token(16)
	if e != nil {
		return "", e
	}
	return s[:8] + "-" + s[8:12] + "-4" + s[13:16] + "-8" + s[17:20] + "-" + s[20:], nil
}
