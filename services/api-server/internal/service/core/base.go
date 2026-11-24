// 服务共享依赖

package core

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/config"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

type Base struct {
	Store *repository.Store
	Cfg   *config.Config
}

// 创建共享依赖
func NewBase(store *repository.Store, cfg *config.Config) *Base {
	return &Base{Store: store, Cfg: cfg}
}
