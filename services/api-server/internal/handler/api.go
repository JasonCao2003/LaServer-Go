package handler

import (
	"encoding/json"
	"errors"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/config"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/middleware"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"log/slog"
	"net"
	"net/http"
)

type Config = config.Config
type object = service.Object
type endpoint func(service.Request, object) (object, error)
type API struct {
	cfg    *Config
	routes map[string]endpoint
	legacy map[string]bool
}

func NewAPI(s *service.Service, cfg *Config) *API {
	a := &API{cfg: cfg, routes: map[string]endpoint{}, legacy: map[string]bool{}}
	for _, path := range service.LegacyPaths {
		a.legacy[path] = true
	}
	a.routes["/account/create"] = s.Create
	a.routes["/account/auth/create"] = s.Create
	a.routes["/account/login"] = s.Login
	a.routes["/account/auth/login"] = s.Login
	a.routes["/account/userdata"] = s.Userdata
	a.routes["/account/sync_user_data"] = s.Userdata
	a.routes["/account/get_currency"] = s.Currency
	a.routes["/account/get_unlocks"] = s.Unlocks
	a.routes["/account/ack_unlock"] = s.AckUnlock
	a.routes["/account/get_rank"] = s.Rank
	a.routes["/account/find_previous_or_empty"] = s.Discover
	a.routes["/account/get_checkpoints"] = s.Checkpoints
	a.routes["/account/set_checkpoint"] = s.Checkpoints
	a.routes["/account/get_latest_build_version"] = s.LatestBuild
	a.routes["/account/device_capabilities/lookup"] = s.Capabilities
	a.routes["/account/get_vars"] = s.Variables
	a.routes["/account/get_motd"] = s.Motd
	a.routes["/account/support/has_reply"] = s.Support
	a.routes["/account/get_collectibles"] = s.Collectibles
	a.routes["/account/get_achievements"] = s.Achievements
	a.routes["/account/get_app_badge_number"] = s.Badge
	a.routes["/account/get_achievement_defs"] = s.AchievementDefinitions
	a.routes["/account/get_map_defs"] = s.MapDefinitions
	a.routes["/account/get_friends"] = s.Friends
	a.routes["/account/get_friend_statues"] = s.FriendStatues
	a.routes["/account/get_online_friends"] = s.OnlineFriends
	a.routes["/account/get_forge_rates"] = s.ForgeRates
	a.routes["/account/get_outfit"] = s.Outfit
	a.routes["/account/get_questionnaires"] = s.Questionnaires
	a.routes["/account/get_achievement_stats"] = s.AchievementStats
	a.routes["/account/get_achievement_stats_tracking"] = s.AchievementStatsTracking
	a.routes["/account/get_event_currency_defs"] = s.EventCurrencyDefinitions
	a.routes["/account/get_relationship_abilities"] = s.RelationshipDefinitions
	a.routes["/account/get_invites"] = s.Invites
	a.routes["/account/get_questionnaire_groups"] = s.QuestionnaireGroups
	a.routes["/account/get_generic_shops"] = s.GenericShops
	a.routes["/account/get_currency_defs"] = s.CurrencyDefinitions
	a.routes["/account/get_surveys"] = s.Surveys
	a.routes["/account/get_video_defs"] = s.VideoDefinitions
	a.routes["/account/trust/get_trust_activity_defs"] = s.TrustActivityDefinitions
	a.routes["/account/trust/get_trust_group_defs"] = s.TrustGroupDefinitions
	a.routes["/service/infraction/api/v1/get_infractions_status_decorated"] = s.InfractionStatus
	a.routes["/account/trust/get_trust_tier_defs"] = s.TrustTierDefinitions
	a.routes["/account/get_radiance_spawn_defs"] = s.RadianceSpawnDefinitions
	a.routes["/account/remote_config/upload"] = s.RemoteConfigUpload
	a.routes["/account/get_generic_shop_item_tracking_info"] = s.GenericShopTracking
	a.routes["/account/get_spirit_shops"] = s.SpiritShops
	a.routes["/account/get_event_schedule"] = s.EventSchedule
	a.routes["/service/auth/api/v1/token"] = s.ResourceAuthToken
	a.routes["/starwatch/auth/api/v1/auth"] = s.StarwatchAuth
	a.routes["/account/get_level_pickups"] = s.LevelPickups
	a.routes["/account/quest/get_quest_defs"] = s.QuestDefinitions
	a.routes["/account/iaplist"] = s.IAPList
	a.routes["/account/check_invite"] = s.CheckInvite
	a.routes["/account/delete_invite"] = s.DeleteInvite
	a.routes["/account/create_invite"] = s.CreateInvite
	a.routes["/account/accept_invite"] = s.AcceptInvite
	a.routes["/account/consumable/get_consumable_defs"] = s.ConsumableDefinitions
	a.routes["/account/consumable/get_consumables"] = s.GetConsumables
	a.routes["/account/consumable/get_consumables1"] = s.GetConsumables1
	a.routes["/account/consumable/consume_item1"] = s.ConsumeItem
	a.routes["/account/buff/get_buff_defs"] = s.BuffDefinitions
	a.routes["/account/buff/get_buffs"] = s.GetBuffs
	a.routes["/account/buff/get_buffs1"] = s.GetBuffs1
	a.routes["/account/buff/remove_buff"] = s.RemoveBuff
	a.routes["/account/buff/remove_buff1"] = s.RemoveBuff
	a.routes["/account/wing_buffs/get"] = s.GetWingBuffs
	a.routes["/account/wing_buffs/collect"] = s.CollectWingBuffs
	a.routes["/account/wing_buffs/drop"] = s.DropWingBuffs
	a.routes["/account/wing_buffs/deposit"] = s.DepositWingBuffs
	a.routes["/account/wing_buffs/convert"] = s.ConvertWingBuffs
	a.routes["/account/collect_collectible"] = s.CollectCollectible
	a.routes["/account/ack_collectible"] = s.AckCollectible
	a.routes["/service/status/api/v1/get_unlocks"] = s.StatusUnlocks
	a.routes["/service/status/api/v1/add_unlock"] = s.AddStatusUnlock
	a.routes["/service/status/api/v1/delete_unlock"] = s.DeleteStatusUnlock
	a.routes["/service/inventory/api/v1/unlocks/remove_dye"] = s.RemoveDye
	a.routes["/account/redeem_reward"] = s.RedeemReward
	a.routes["/account/ack_forge_rate_reset"] = s.AckForgeRateReset
	a.routes["/account/commerce/update"] = s.CommerceUpdate
	a.routes["/account/commerce/product_info"] = s.CommerceProductInfo
	a.routes["/account/get_shop"] = s.GetShop
	a.routes["/account/lootboxes/get"] = s.GetLootboxes
	a.routes["/account/code/verify"] = s.VerifyCode
	a.routes["/account/purchase_event_buff"] = s.PurchaseEventBuff
	a.routes["/account/purchase_dye"] = s.PurchaseDye
	a.routes["/account/purchase_generic_shop_item"] = s.PurchaseGenericShopItem
	a.routes["/account/purchase_spirit_shop_item"] = s.PurchaseSpiritShopItem
	a.routes["/account/lootboxes/redeem_box"] = s.RedeemBox
	a.routes["/account/buy_candle_wax"] = s.BuyCandleWax
	a.routes["/account/purchase_unlock"] = s.PurchaseUnlock
	a.routes["/account/commerce/receipt"] = s.CommerceReceipt
	a.routes["/account/commerce/fulfill"] = s.CommerceFulfill
	a.routes["/account/set_achievement_stats"] = s.SetAchievementStats
	a.routes["/account/get_daily_quests"] = s.EmptyDailyQuests
	a.routes["/account/get_weekly_quests"] = s.EmptyWeeklyQuests
	a.routes["/account/get_season_quests_v2"] = s.EmptySeasonQuestsV2
	a.routes["/account/get_season_pass_defs"] = s.EmptySeasonPassDefs
	a.routes["/account/get_event_currency"] = s.EmptyEventCurrency
	a.routes["/account/season_finalize"] = s.SeasonFinalize
	a.routes["/account/season_start"] = s.SeasonStart
	a.routes["/account/activate_season_quest"] = s.ActivateSeasonQuest
	a.routes["/account/claim_season_quest_reward"] = s.ClaimSeasonQuestReward
	a.routes["/account/get_season_quests"] = s.SeasonQuests
	a.routes["/account/get_account_world_quests"] = s.WorldQuests
	a.routes["/account/claim_quest_reward"] = s.ClaimWorldQuestReward
	a.routes["/account/collect_pickup_batch"] = s.CollectPickupBatch
	a.routes["/service/quest/api/v1/check_for_previous_completion"] = s.CheckPreviousCompletion
	a.routes["/service/quest/api/v1/get_player_quests"] = s.PlayerQuests
	a.routes["/service/quest/api/v1/get_player_quests_by_name"] = s.PlayerQuestsByName
	a.routes["/service/quest/api/v1/get_quest_schedule"] = s.QuestSchedule
	a.routes["/service/quest/api/v1/create"] = s.CreateQuest
	a.routes["/service/quest/api/v1/update_quest"] = s.UpdateQuest
	a.routes["/account/get_blocked_friends"] = s.GetBlockedFriends
	a.routes["/account/set_friend_name"] = s.SetFriendName
	a.routes["/account/set_friend_block"] = s.SetFriendBlock
	a.routes["/account/get_external_friends"] = s.GetExternalFriends
	a.routes["/account/serendipity/get"] = s.Serendipity
	a.routes["/account/star/get"] = s.StarTags
	a.routes["/account/join_friend_game"] = s.JoinFriendGame
	a.routes["/account/set_outfit"] = s.SetOutfit
	a.routes["/account/set_joinable"] = s.SetJoinable
	a.routes["/account/get_remote_outfit"] = s.GetRemoteOutfit
	a.routes["/account/remove_relationship_hint"] = s.RemoveRelationshipHint
	a.routes["/service/relationship/api/v1/get_all_friend_gist"] = s.GetAllFriendGist
	a.routes["/service/relationship/api/v1/get_follower_count"] = s.GetFollowerCount
	a.routes["/service/relationship/api/v1/update_friend_constellation_pages"] = s.UpdateFriendConstellationPages
	a.routes["/service/trust/api/v1/get_capped"] = s.TrustGetCapped
	a.routes["/service/trust/api/v1/get"] = s.TrustGet
	a.routes["/service/relationship/api/v1/add_follow"] = s.AddFollow
	a.routes["/service/relationship/api/v1/set_follow_name"] = s.SetFollowName
	a.routes["/service/relationship/api/v1/delete_follow"] = s.DeleteFollow
	return a
}
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Path == "/healthz" {
		if r.Method != "GET" {
			write(w, 405, object{"error": "Method not allowed"})
			return
		}
		write(w, 200, object{"status": "ok"})
		return
	}
	handler, ok := a.routes[r.URL.Path]
	if !ok {
		status := 404
		if a.legacy[r.URL.Path] {
			status = 501
		}
		write(w, status, object{"error": http.StatusText(status)})
		return
	}
	if r.URL.Path == "/service/relationship/api/v1/update_friend_constellation_pages" && r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,User-Agent,X-Requested-With")
		w.Header().Set("Access-Control-Allow-Methods", "POST,OPTIONS")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(200)
		return
	}
	productInfoGet := r.URL.Path == "/account/commerce/product_info" && r.Method == "GET"
	if r.Method != "POST" && !productInfoGet {
		if r.URL.Path == "/account/commerce/product_info" {
			w.Header().Set("Allow", "GET, POST")
		} else if r.URL.Path == "/service/relationship/api/v1/update_friend_constellation_pages" {
			w.Header().Set("Allow", "POST, OPTIONS")
		} else {
			w.Header().Set("Allow", "POST")
		}
		write(w, 405, object{"error": "Method not allowed"})
		return
	}
	data := object{}
	if !productInfoGet {
		var bodyErr *middleware.BodyError
		data, bodyErr = middleware.ReadObject(w, r)
		if bodyErr != nil {
			write(w, bodyErr.Status, object{"error": bodyErr.Message})
			return
		}
	}
	result, err := handler(service.Request{Context: r.Context(), Path: r.URL.Path, UserAgent: r.UserAgent(), RemoteIP: remoteIP(r), Host: r.Host, Secure: r.TLS != nil}, data)
	if err != nil {
		var e *service.Error
		if errors.As(err, &e) {
			write(w, e.Status, e.Body)
		} else {
			slog.Error("request failed", "path", r.URL.Path, "error_type", errorType(err))
			write(w, 500, object{"error": "Internal server error"})
		}
		return
	}
	write(w, 200, result)
}
func errorType(err error) string { return "internal" }
func write(w http.ResponseWriter, status int, body object) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
func remoteIP(r *http.Request) string {
	h, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		return r.RemoteAddr
	}
	return h
}
