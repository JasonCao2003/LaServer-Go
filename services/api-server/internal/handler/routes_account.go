// account 业务 HTTP 路由。
package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/account"
	"github.com/gin-gonic/gin"
)

// 注册 account 服务处理的路径与方法。
func registerAccount(r *gin.Engine, s *account.Service) {
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
	// 签发资源访问令牌
	r.POST("/service/auth/api/v1/token", post(s.ResourceAuthToken))
	// 执行星巡鉴权
	r.POST("/starwatch/auth/api/v1/auth", post(s.StarwatchAuth))
}
