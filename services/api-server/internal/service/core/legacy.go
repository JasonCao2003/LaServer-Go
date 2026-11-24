// 旧版静态响应读取

package core

import "github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/legacydata"

// 从旧版静态数据读取响应
func LegacyResponse(name string, target any) error {
	return legacydata.Response(name, target)
}
