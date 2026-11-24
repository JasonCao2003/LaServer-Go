// 任务、赛季与奖励领取路由注册

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册任务赛季路由
func registerQuests(r *gin.Engine, s *service.Service) {
	r.POST("/account/set_achievement_stats", post(s.SetAchievementStats))
	r.POST("/account/get_daily_quests", post(s.EmptyDailyQuests))
	r.POST("/account/get_weekly_quests", post(s.EmptyWeeklyQuests))
	r.POST("/account/get_season_quests_v2", post(s.EmptySeasonQuestsV2))
	r.POST("/account/get_season_pass_defs", post(s.EmptySeasonPassDefs))
	r.POST("/account/get_event_currency", post(s.EmptyEventCurrency))
	r.POST("/account/season_finalize", post(s.SeasonFinalize))
	r.POST("/account/season_start", post(s.SeasonStart))
	r.POST("/account/activate_season_quest", post(s.ActivateSeasonQuest))
	r.POST("/account/claim_season_quest_reward", post(s.ClaimSeasonQuestReward))
	r.POST("/account/get_season_quests", post(s.SeasonQuests))
	r.POST("/account/get_account_world_quests", post(s.WorldQuests))
	r.POST("/account/claim_quest_reward", post(s.ClaimWorldQuestReward))
	r.POST("/account/collect_pickup_batch", post(s.CollectPickupBatch))
	r.POST("/service/quest/api/v1/check_for_previous_completion", post(s.CheckPreviousCompletion))
	r.POST("/service/quest/api/v1/get_player_quests", post(s.PlayerQuests))
	r.POST("/service/quest/api/v1/get_player_quests_by_name", post(s.PlayerQuestsByName))
	r.POST("/service/quest/api/v1/get_quest_schedule", post(s.QuestSchedule))
	r.POST("/service/quest/api/v1/create", post(s.CreateQuest))
	r.POST("/service/quest/api/v1/update_quest", post(s.UpdateQuest))
}
