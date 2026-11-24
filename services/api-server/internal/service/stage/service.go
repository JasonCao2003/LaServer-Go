// stage 服务类型

package stage

import "github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"

type Service struct {
	*core.Base
}

// 创建服务实例
func New(base *core.Base) *Service {
	return &Service{Base: base}
}
