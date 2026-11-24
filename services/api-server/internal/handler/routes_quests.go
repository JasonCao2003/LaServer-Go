// 任务、赛季与奖励领取路由注册

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册任务赛季路由
func registerQuests(r *gin.Engine, s *service.Service) {
	// 设置成就统计
	r.POST("/account/set_achievement_stats", post(s.SetAchievementStats))
	// 返回空的每日任务
	r.POST("/account/get_daily_quests", post(s.EmptyDailyQuests))
	// 返回空的每周任务
	r.POST("/account/get_weekly_quests", post(s.EmptyWeeklyQuests))
	// 返回空的赛季任务（v2 新版占位）
	r.POST("/account/get_season_quests_v2", post(s.EmptySeasonQuestsV2))
	// 返回空的赛季通行证定义
	r.POST("/account/get_season_pass_defs", post(s.EmptySeasonPassDefs))
	// 返回空的活动货币
	r.POST("/account/get_event_currency", post(s.EmptyEventCurrency))
	// 结算赛季
	r.POST("/account/season_finalize", post(s.SeasonFinalize))
	// 开始赛季
	r.POST("/account/season_start", post(s.SeasonStart))
	// 激活赛季任务
	r.POST("/account/activate_season_quest", post(s.ActivateSeasonQuest))
	// 领取赛季任务奖励
	r.POST("/account/claim_season_quest_reward", post(s.ClaimSeasonQuestReward))
	// 获取赛季任务
	r.POST("/account/get_season_quests", post(s.SeasonQuests))
	// 获取世界任务
	r.POST("/account/get_account_world_quests", post(s.WorldQuests))
	// 领取世界任务奖励
	r.POST("/account/claim_quest_reward", post(s.ClaimWorldQuestReward))
	// 批量收集拾取
	r.POST("/account/collect_pickup_batch", post(s.CollectPickupBatch))
	// 检查任务先前完成记录
	r.POST("/service/quest/api/v1/check_for_previous_completion", post(s.CheckPreviousCompletion))
	// 获取玩家任务
	r.POST("/service/quest/api/v1/get_player_quests", post(s.PlayerQuests))
	// 按名称获取玩家任务
	r.POST("/service/quest/api/v1/get_player_quests_by_name", post(s.PlayerQuestsByName))
	// 获取任务日程
	r.POST("/service/quest/api/v1/get_quest_schedule", post(s.QuestSchedule))
	// 创建任务
	r.POST("/service/quest/api/v1/create", post(s.CreateQuest))
	// 更新任务
	r.POST("/service/quest/api/v1/update_quest", post(s.UpdateQuest))
}
