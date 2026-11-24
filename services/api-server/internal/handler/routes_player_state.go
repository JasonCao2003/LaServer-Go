// 玩家状态、库存、增益与邀请路由注册

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册玩家状态路由
func registerPlayerState(r *gin.Engine, s *service.Service) {
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
	// 获取消耗品定义
	r.POST("/account/consumable/get_consumable_defs", post(s.ConsumableDefinitions))
	// 获取消耗品（新版）
	r.POST("/account/consumable/get_consumables", post(s.GetConsumables))
	// 获取消耗品（旧版）
	r.POST("/account/consumable/get_consumables1", post(s.GetConsumables1))
	// 使用消耗品
	r.POST("/account/consumable/consume_item1", post(s.ConsumeItem))
	// 获取增益定义
	r.POST("/account/buff/get_buff_defs", post(s.BuffDefinitions))
	// 获取增益（新版）
	r.POST("/account/buff/get_buffs", post(s.GetBuffs))
	// 获取增益（旧版）
	r.POST("/account/buff/get_buffs1", post(s.GetBuffs1))
	// 移除增益（兼容别名，同 /account/buff/remove_buff1）
	r.POST("/account/buff/remove_buff", post(s.RemoveBuff))
	// 移除增益（兼容别名，同 /account/buff/remove_buff）
	r.POST("/account/buff/remove_buff1", post(s.RemoveBuff))
	// 获取光翼增益
	r.POST("/account/wing_buffs/get", post(s.GetWingBuffs))
	// 收集光翼增益
	r.POST("/account/wing_buffs/collect", post(s.CollectWingBuffs))
	// 丢弃光翼增益
	r.POST("/account/wing_buffs/drop", post(s.DropWingBuffs))
	// 存入光翼增益
	r.POST("/account/wing_buffs/deposit", post(s.DepositWingBuffs))
	// 转换光翼增益
	r.POST("/account/wing_buffs/convert", post(s.ConvertWingBuffs))
	// 收集收集品
	r.POST("/account/collect_collectible", post(s.CollectCollectible))
	// 确认收集品
	r.POST("/account/ack_collectible", post(s.AckCollectible))
	// 获取状态解锁
	r.POST("/service/status/api/v1/get_unlocks", post(s.StatusUnlocks))
	// 新增状态解锁
	r.POST("/service/status/api/v1/add_unlock", post(s.AddStatusUnlock))
	// 删除状态解锁
	r.POST("/service/status/api/v1/delete_unlock", post(s.DeleteStatusUnlock))
	// 移除染料
	r.POST("/service/inventory/api/v1/unlocks/remove_dye", post(s.RemoveDye))
	// 确认奖励（占位）
	r.POST("/account/redeem_reward", post(s.RedeemReward))
	// 确认锻造概率重置（占位）
	r.POST("/account/ack_forge_rate_reset", post(s.AckForgeRateReset))
}
