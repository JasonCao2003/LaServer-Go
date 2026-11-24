// 玩家状态、库存、增益与邀请路由注册

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册玩家状态路由
func registerPlayerState(r *gin.Engine, s *service.Service) {
	r.POST("/account/get_level_pickups", post(s.LevelPickups))
	r.POST("/account/quest/get_quest_defs", post(s.QuestDefinitions))
	r.POST("/account/iaplist", post(s.IAPList))
	r.POST("/account/check_invite", post(s.CheckInvite))
	r.POST("/account/delete_invite", post(s.DeleteInvite))
	r.POST("/account/create_invite", post(s.CreateInvite))
	r.POST("/account/accept_invite", post(s.AcceptInvite))
	r.POST("/account/consumable/get_consumable_defs", post(s.ConsumableDefinitions))
	r.POST("/account/consumable/get_consumables", post(s.GetConsumables))
	r.POST("/account/consumable/get_consumables1", post(s.GetConsumables1))
	r.POST("/account/consumable/consume_item1", post(s.ConsumeItem))
	r.POST("/account/buff/get_buff_defs", post(s.BuffDefinitions))
	r.POST("/account/buff/get_buffs", post(s.GetBuffs))
	r.POST("/account/buff/get_buffs1", post(s.GetBuffs1))
	r.POST("/account/buff/remove_buff", post(s.RemoveBuff))
	r.POST("/account/buff/remove_buff1", post(s.RemoveBuff))
	r.POST("/account/wing_buffs/get", post(s.GetWingBuffs))
	r.POST("/account/wing_buffs/collect", post(s.CollectWingBuffs))
	r.POST("/account/wing_buffs/drop", post(s.DropWingBuffs))
	r.POST("/account/wing_buffs/deposit", post(s.DepositWingBuffs))
	r.POST("/account/wing_buffs/convert", post(s.ConvertWingBuffs))
	r.POST("/account/collect_collectible", post(s.CollectCollectible))
	r.POST("/account/ack_collectible", post(s.AckCollectible))
	r.POST("/service/status/api/v1/get_unlocks", post(s.StatusUnlocks))
	r.POST("/service/status/api/v1/add_unlock", post(s.AddStatusUnlock))
	r.POST("/service/status/api/v1/delete_unlock", post(s.DeleteStatusUnlock))
	r.POST("/service/inventory/api/v1/unlocks/remove_dye", post(s.RemoveDye))
	r.POST("/account/redeem_reward", post(s.RedeemReward))
	r.POST("/account/ack_forge_rate_reset", post(s.AckForgeRateReset))
}
