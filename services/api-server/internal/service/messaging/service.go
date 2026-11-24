// messaging 服务类型

package messaging

import "github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"

type Service struct {
	*core.Base
	bus *messageBus
}

// 创建服务实例
func New(base *core.Base) *Service {
	return &Service{Base: base, bus: newMessageBus()}
}
