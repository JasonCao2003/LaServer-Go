// content 业务 HTTP 路由。
package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/content"
	"github.com/gin-gonic/gin"
)

// 注册 content 服务处理的路径与方法。
func registerContent(r *gin.Engine, s *content.Service) {
	// 获取邀请列表
	r.POST("/account/get_invites", post(s.Invites))
	// 获取问卷分组
	r.POST("/account/get_questionnaire_groups", post(s.QuestionnaireGroups))
	// 获取通用商店
	r.POST("/account/get_generic_shops", post(s.GenericShops))
	// 获取货币定义
	r.POST("/account/get_currency_defs", post(s.CurrencyDefinitions))
	// 获取问卷列表
	r.POST("/account/get_surveys", post(s.Surveys))
	// 获取视频定义
	r.POST("/account/get_video_defs", post(s.VideoDefinitions))
	// 获取信任活动定义
	r.POST("/account/trust/get_trust_activity_defs", post(s.TrustActivityDefinitions))
	// 获取信任分组定义
	r.POST("/account/trust/get_trust_group_defs", post(s.TrustGroupDefinitions))
	// 获取违规状态
	r.POST("/service/infraction/api/v1/get_infractions_status_decorated", post(s.InfractionStatus))
	// 获取信任等级定义
	r.POST("/account/trust/get_trust_tier_defs", post(s.TrustTierDefinitions))
	// 获取辐光刷新定义
	r.POST("/account/get_radiance_spawn_defs", post(s.RadianceSpawnDefinitions))
	// 上传远程配置
	r.POST("/account/remote_config/upload", post(s.RemoteConfigUpload))
	// 记录通用商店购买跟踪
	r.POST("/account/get_generic_shop_item_tracking_info", post(s.GenericShopTracking))
	// 获取活动日程
	r.POST("/account/get_event_schedule", post(s.EventSchedule))
	// 获取关卡拾取
	r.POST("/account/get_level_pickups", post(s.LevelPickups))
	// 获取任务定义
	r.POST("/account/quest/get_quest_defs", post(s.QuestDefinitions))
	// 获取内购列表
	r.POST("/account/iaplist", post(s.IAPList))
	// 检查邀请
	r.POST("/account/check_invite", post(s.CheckInvite))
	// 删除邀请
	r.POST("/account/delete_invite", post(s.DeleteInvite))
	// 创建邀请
	r.POST("/account/create_invite", post(s.CreateInvite))
	// 接受邀请
	r.POST("/account/accept_invite", post(s.AcceptInvite))
}
