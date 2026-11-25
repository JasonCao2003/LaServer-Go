// relationships 业务 HTTP 路由。
package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/relationships"
	"github.com/gin-gonic/gin"
)

// 注册 relationships 服务处理的路径与方法。
func registerRelationships(r *gin.Engine, s *relationships.Service) {
	// 读取好友列表
	r.POST("/account/get_friends", post(s.Friends))
	// 获取好友状态
	r.POST("/account/get_friend_statues", post(s.FriendStatues))

	r.OPTIONS("/account/get_friend_statues", preflight("GET,PUT,POST,DELETE,OPTIONS", "Content-Type,Authorization", ""))
	// 获取在线好友
	r.POST("/account/get_online_friends", post(s.OnlineFriends))
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
	// 列出管理员好友
	r.POST("/admin/api/accounts_list", post(s.AdminAccountsList))
	r.POST("/admin/api/catalog", post(s.AdminCatalog))
	r.POST("/admin/api/accounts_change", post(s.AdminAccountsChange))
	r.POST("/admin/api/friends_list", post(s.AdminFriendsList))
	// 列出管理员好友
	r.GET("/admin/api/friends_list", adminQuery(s.AdminFriendsList))
	// 设置管理员好友关系
	r.POST("/admin/api/friends_set", post(s.AdminFriendsSet))
	// 设置管理员好友关系
	r.GET("/admin/api/friends_set", adminQuery(s.AdminFriendsSet))
	// 清空管理员好友关系
	r.POST("/admin/api/friends_clear", post(s.AdminFriendsClear))
	// 清空管理员好友关系
	r.GET("/admin/api/friends_clear", adminQuery(s.AdminFriendsClear))
}
