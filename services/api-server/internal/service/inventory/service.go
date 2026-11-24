// inventory 服务类型

package inventory

import "github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"

type Service struct {
	*core.Base
}

// 创建服务实例
func New(base *core.Base) *Service {
	return &Service{Base: base}
}

// 读取锻造概率
func (a *Service) ForgeRates(r Request, m Object) (Object, error) {
	return a.Base.ForgeRates(r, m)
}
