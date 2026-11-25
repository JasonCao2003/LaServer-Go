package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type preview struct {
	data []byte
	size int
}

func (p *preview) Write(b []byte) (int, error) {
	n := len(b)
	p.size += n
	if left := 4096 - len(p.data); left > 0 {
		p.data = append(p.data, b[:min(left, n)]...)
	}
	return n, nil
}
func (p *preview) String() string {
	if p.size == 0 {
		return ""
	}
	if p.size > 4096 {
		return "正文超过预览上限"
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(p.data, &obj) != nil || obj == nil {
		return "正文已隐藏"
	}
	safe := map[string]json.RawMessage{}
	for _, k := range []string{"ok", "success", "count", "quantity", "limit", "offset", "page", "status"} {
		v := bytes.TrimSpace(obj[k])
		if len(v) > 0 && (v[0] >= '0' && v[0] <= '9' || string(v) == "true" || string(v) == "false") {
			safe[k] = v
		}
	}
	body, _ := json.Marshal(safe)
	return fmt.Sprintf("%d 个字段，公开字段 %s", len(obj), body)
}

type previewReader struct {
	io.Reader
	io.Closer
}
