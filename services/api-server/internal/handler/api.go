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
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		write(w, 405, object{"error": "Method not allowed"})
		return
	}
	data, bodyErr := middleware.ReadObject(w, r)
	if bodyErr != nil {
		write(w, bodyErr.Status, object{"error": bodyErr.Message})
		return
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
