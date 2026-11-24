// 注册全部业务路由入口

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册全部业务路由
func registerRoutes(r *gin.Engine, s *service.Service) {
	// 返回健康检查响应
	r.GET("/healthz", health)
	// 提供 UGC 文件下载
	r.GET("/ugc/*subpath", serveUGC(s.Social))
	registerAccount(r, s.Account)
	registerCommerce(r, s.Commerce)
	registerContent(r, s.Content)
	registerInventory(r, s.Inventory)
	registerMessaging(r, s.Messaging)
	registerQuests(r, s.Quests)
	registerRelationships(r, s.Relationships)
	registerSocial(r, s.Social)
	registerStage(r, s.Stage)
}
