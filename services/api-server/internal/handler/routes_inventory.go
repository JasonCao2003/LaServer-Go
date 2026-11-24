// inventory 业务 HTTP 路由。
package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/inventory"
	"github.com/gin-gonic/gin"
)

// 注册 inventory 服务处理的路径与方法。
func registerInventory(r *gin.Engine, s *inventory.Service) {
	// 读取解锁列表
	r.POST("/account/get_unlocks", post(s.Unlocks))
	// 确认解锁
	r.POST("/account/ack_unlock", post(s.AckUnlock))
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
}
