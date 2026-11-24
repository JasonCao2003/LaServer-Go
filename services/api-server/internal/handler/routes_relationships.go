// 好友、关系与关注路由注册

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册好友关系路由
func registerRelationships(r *gin.Engine, s *service.Service) {
	// 获取拉黑好友
	r.POST("/account/get_blocked_friends", post(s.GetBlockedFriends))
	// 设置好友名称
	r.POST("/account/set_friend_name", post(s.SetFriendName))
	// 设置好友拉黑
	r.POST("/account/set_friend_block", post(s.SetFriendBlock))
	// 获取外部好友
	r.POST("/account/get_external_friends", post(s.GetExternalFriends))
	// 获取缘分匹配
	r.POST("/account/serendipity/get", post(s.Serendipity))
	// 获取星标标签
	r.POST("/account/star/get", post(s.StarTags))
	// 加入好友游戏
	r.POST("/account/join_friend_game", post(s.JoinFriendGame))
	// 设置装扮
	r.POST("/account/set_outfit", post(s.SetOutfit))
	// 设置可加入状态
	r.POST("/account/set_joinable", post(s.SetJoinable))
	// 获取远程装扮
	r.POST("/account/get_remote_outfit", post(s.GetRemoteOutfit))
	// 移除关系提示
	r.POST("/account/remove_relationship_hint", post(s.RemoveRelationshipHint))
	// 获取全部好友摘要
	r.POST("/service/relationship/api/v1/get_all_friend_gist", post(s.GetAllFriendGist))
	// 获取关注者数量
	r.POST("/service/relationship/api/v1/get_follower_count", post(s.GetFollowerCount))
	// 更新好友星座页面
	r.POST("/service/relationship/api/v1/update_friend_constellation_pages", post(s.UpdateFriendConstellationPages))
	r.OPTIONS("/service/relationship/api/v1/update_friend_constellation_pages", preflight("POST,OPTIONS", "Content-Type,Authorization,User-Agent,X-Requested-With", "86400"))
	// 获取信任上限信息
	r.POST("/service/trust/api/v1/get_capped", post(s.TrustGetCapped))
	// 获取信任信息
	r.POST("/service/trust/api/v1/get", post(s.TrustGet))
	// 新增关注
	r.POST("/service/relationship/api/v1/add_follow", post(s.AddFollow))
	// 设置关注名称
	r.POST("/service/relationship/api/v1/set_follow_name", post(s.SetFollowName))
	// 删除关注
	r.POST("/service/relationship/api/v1/delete_follow", post(s.DeleteFollow))
}
