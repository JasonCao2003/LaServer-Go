// 账号、鉴权与公共配置路由注册

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册账号与鉴权路由
func registerAccounts(r *gin.Engine, s *service.Service) {
	// 创建账号或资源（兼容别名，同 /account/auth/create）
	r.POST("/account/create", post(s.Create))
	// 创建账号或资源（兼容别名，同 /account/create）
	r.POST("/account/auth/create", post(s.Create))
	// 执行账号登录（兼容别名，同 /account/auth/login）
	r.POST("/account/login", post(s.Login))
	// 执行账号登录（兼容别名，同 /account/login）
	r.POST("/account/auth/login", post(s.Login))
	// 读取或保存用户数据（兼容别名，同 /account/sync_user_data）
	r.POST("/account/userdata", post(s.Userdata))
	// 读取或保存用户数据（兼容别名，同 /account/userdata）
	r.POST("/account/sync_user_data", post(s.Userdata))
	// 获取账号货币
	r.POST("/account/get_currency", post(s.Currency))
	// 读取解锁列表
	r.POST("/account/get_unlocks", post(s.Unlocks))
	// 确认解锁
	r.POST("/account/ack_unlock", post(s.AckUnlock))
	// 获取排行信息
	r.POST("/account/get_rank", post(s.Rank))
	// 查找可加入玩家
	r.POST("/account/find_previous_or_empty", post(s.Discover))
	// 校验房间准入
	r.POST("/internal/v1/room/admissions/verify", post(s.VerifyRoomAdmission))
	// 读取或设置检查点（读写共用）
	r.POST("/account/get_checkpoints", post(s.Checkpoints))
	// 读取或设置检查点（读写共用）
	r.POST("/account/set_checkpoint", post(s.Checkpoints))
	// 获取最新构建版本
	r.POST("/account/get_latest_build_version", post(s.LatestBuild))
	// 获取设备能力配置
	r.POST("/account/device_capabilities/lookup", post(s.Capabilities))
	// 获取客户端变量
	r.POST("/account/get_vars", post(s.Variables))
	// 获取公告信息
	r.POST("/account/get_motd", post(s.Motd))
	// 查询客服回复
	r.POST("/account/support/has_reply", post(s.Support))
	// 获取收集品列表
	r.POST("/account/get_collectibles", post(s.Collectibles))
	// 获取成就列表
	r.POST("/account/get_achievements", post(s.Achievements))
	// 获取徽章数量
	r.POST("/account/get_app_badge_number", post(s.Badge))
	// 获取成就定义
	r.POST("/account/get_achievement_defs", post(s.AchievementDefinitions))
	// 获取地图定义
	r.POST("/account/get_map_defs", post(s.MapDefinitions))
	// 读取好友列表
	r.POST("/account/get_friends", post(s.Friends))
	// 获取好友状态
	r.POST("/account/get_friend_statues", post(s.FriendStatues))
	r.OPTIONS("/account/get_friend_statues", preflight("GET,PUT,POST,DELETE,OPTIONS", "Content-Type,Authorization", ""))
	// 获取在线好友
	r.POST("/account/get_online_friends", post(s.OnlineFriends))
	// 读取锻造概率
	r.POST("/account/get_forge_rates", post(s.ForgeRates))
	// 获取装扮
	r.POST("/account/get_outfit", post(s.Outfit))
	// 获取问卷
	r.POST("/account/get_questionnaires", post(s.Questionnaires))
	// 读取成就统计
	r.POST("/account/get_achievement_stats", post(s.AchievementStats))
	// 获取成就统计跟踪
	r.POST("/account/get_achievement_stats_tracking", post(s.AchievementStatsTracking))
	// 获取活动货币定义
	r.POST("/account/get_event_currency_defs", post(s.EventCurrencyDefinitions))
	// 获取关系能力定义
	r.POST("/account/get_relationship_abilities", post(s.RelationshipDefinitions))
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
	// 获取精灵商店
	r.POST("/account/get_spirit_shops", post(s.SpiritShops))
	// 获取活动日程
	r.POST("/account/get_event_schedule", post(s.EventSchedule))
	// 签发资源访问令牌
	r.POST("/service/auth/api/v1/token", post(s.ResourceAuthToken))
	// 执行星巡鉴权
	r.POST("/starwatch/auth/api/v1/auth", post(s.StarwatchAuth))
}
