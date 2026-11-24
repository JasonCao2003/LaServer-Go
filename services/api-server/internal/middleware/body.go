// 请求体读取与大小限制中间件

package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
)

type BodyError struct {
	Status  int
	Message string
}

// 返回错误文本
func (e *BodyError) Error() string { return e.Message }

// 读取并解析请求体
func ReadObject(w http.ResponseWriter, r *http.Request) (map[string]any, *BodyError) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2<<20))
	if err != nil {
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			return nil, &BodyError{413, "Request body too large"}
		}
		return nil, &BodyError{400, "Invalid request body"}
	}
	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	var data map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&data) == nil && data != nil {
		var extra any
		if decoder.Decode(&extra) == io.EOF {
			return data, nil
		}
	}
	media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if r.URL.Path == "/account/auth/login" && media == "application/x-www-form-urlencoded" {
		values, err := url.ParseQuery(string(raw))
		if err != nil || len(values) == 0 {
			return nil, &BodyError{400, "Invalid request body"}
		}
		data := map[string]any{}
		for k, v := range values {
			if len(v) != 1 {
				return nil, &BodyError{400, "Ambiguous form fields"}
			}
			data[k] = v[0]
		}
		return data, nil
	}
	return nil, &BodyError{400, "Invalid JSON format"}
}
