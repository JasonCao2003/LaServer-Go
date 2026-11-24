// 服务门面：按业务组合子服务

package service

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/config"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/account"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/commerce"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/content"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/inventory"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/messaging"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/quests"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/relationships"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/social"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/stage"
)

type Object = core.Object
type Request = core.Request
type Error = core.Error

var LegacyPaths = core.LegacyPaths

type Service struct {
	Account       *account.Service
	Commerce      *commerce.Service
	Content       *content.Service
	Inventory     *inventory.Service
	Messaging     *messaging.Service
	Quests        *quests.Service
	Relationships *relationships.Service
	Social        *social.Service
	Stage         *stage.Service
}

// 创建实例
func New(store *repository.Store, cfg *config.Config) *Service {
	base := core.NewBase(store, cfg)
	return &Service{
		Account:       account.New(base),
		Commerce:      commerce.New(base),
		Content:       content.New(base),
		Social:        social.New(base),
		Messaging:     messaging.New(base),
		Relationships: relationships.New(base),
		Quests:        quests.New(base),
		Inventory:     inventory.New(base),
		Stage:         stage.New(base),
	}
}
