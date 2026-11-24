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
type object = Object
type Request = core.Request
type Error = core.Error

var LegacyPaths = core.LegacyPaths

type (
	accountService       = account.Service
	commerceService      = commerce.Service
	contentService       = content.Service
	socialService        = social.Service
	messagingService     = messaging.Service
	relationshipsService = relationships.Service
	questsService        = quests.Service
	inventoryService     = inventory.Service
	stageService         = stage.Service
)

type Service struct {
	*accountService
	*commerceService
	*contentService
	*socialService
	*messagingService
	*relationshipsService
	*questsService
	*inventoryService
	*stageService
}

// 创建实例
func New(store *repository.Store, cfg *config.Config) *Service {
	base := core.NewBase(store, cfg)
	return &Service{
		accountService:       account.New(base),
		commerceService:      commerce.New(base),
		contentService:       content.New(base),
		socialService:        social.New(base),
		messagingService:     messaging.New(base),
		relationshipsService: relationships.New(base),
		questsService:        quests.New(base),
		inventoryService:     inventory.New(base),
		stageService:         stage.New(base),
	}
}
