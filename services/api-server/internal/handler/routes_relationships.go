// 好友、关系与关注路由注册

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册好友关系路由
func registerRelationships(r *gin.Engine, s *service.Service) {
	r.POST("/account/get_blocked_friends", post(s.GetBlockedFriends))
	r.POST("/account/set_friend_name", post(s.SetFriendName))
	r.POST("/account/set_friend_block", post(s.SetFriendBlock))
	r.POST("/account/get_external_friends", post(s.GetExternalFriends))
	r.POST("/account/serendipity/get", post(s.Serendipity))
	r.POST("/account/star/get", post(s.StarTags))
	r.POST("/account/join_friend_game", post(s.JoinFriendGame))
	r.POST("/account/set_outfit", post(s.SetOutfit))
	r.POST("/account/set_joinable", post(s.SetJoinable))
	r.POST("/account/get_remote_outfit", post(s.GetRemoteOutfit))
	r.POST("/account/remove_relationship_hint", post(s.RemoveRelationshipHint))
	r.POST("/service/relationship/api/v1/get_all_friend_gist", post(s.GetAllFriendGist))
	r.POST("/service/relationship/api/v1/get_follower_count", post(s.GetFollowerCount))
	r.POST("/service/relationship/api/v1/update_friend_constellation_pages", post(s.UpdateFriendConstellationPages))
	r.OPTIONS("/service/relationship/api/v1/update_friend_constellation_pages", preflight("POST,OPTIONS", "Content-Type,Authorization,User-Agent,X-Requested-With", "86400"))
	r.POST("/service/trust/api/v1/get_capped", post(s.TrustGetCapped))
	r.POST("/service/trust/api/v1/get", post(s.TrustGet))
	r.POST("/service/relationship/api/v1/add_follow", post(s.AddFollow))
	r.POST("/service/relationship/api/v1/set_follow_name", post(s.SetFollowName))
	r.POST("/service/relationship/api/v1/delete_follow", post(s.DeleteFollow))
}
