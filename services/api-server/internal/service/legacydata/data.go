// 旧版静态响应数据（内嵌 JSON）

package legacydata

import (
	"embed"
	"encoding/json"
)

//go:embed *.json
var files embed.FS

// 读取原始数据文件
func Read(name string) ([]byte, error) {
	return files.ReadFile(name)
}

// 读取并解析旧版响应
func Response(name string, target any) error {
	b, err := Read(name + ".json")
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}
