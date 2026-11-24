// 留言、动态、留影与社区路由注册

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册社区动态路由
func registerCommunity(r *gin.Engine, s *service.Service) {
	// 获取用户留言（兼容别名，同 /account/geonote/get_user_geonotes）
	r.POST("/account/get_user_geonotes", post(s.GetUserGeonotes))
	// 获取用户留言（兼容别名，同 /account/get_user_geonotes）
	r.POST("/account/geonote/get_user_geonotes", post(s.GetUserGeonotes))
	// 请求若干好友留言
	r.POST("/account/geonote/request_geonotes_for_n_friends", post(s.RequestGeonotesForNFriends))
	// 创建留言（兼容别名，同 /account/create_geonote2）
	r.POST("/account/geonote/create_geonote2", post(s.CreateGeonote))
	// 创建留言（兼容别名，同 /account/geonote/create_geonote2）
	r.POST("/account/create_geonote2", post(s.CreateGeonote))
	// 创建问题举报留言（兼容别名，同 /account/create_bug_report_geonote）
	r.POST("/account/geonote/create_bug_report_geonote", post(s.CreateBugReportGeonote))
	// 创建问题举报留言（兼容别名，同 /account/geonote/create_bug_report_geonote）
	r.POST("/account/create_bug_report_geonote", post(s.CreateBugReportGeonote))
	// 执行社交留言操作（兼容别名，同 /account/social_geonote2）
	r.POST("/account/geonote/social_geonote2", post(s.SocialGeonote))
	// 执行社交留言操作（兼容别名，同 /account/geonote/social_geonote2）
	r.POST("/account/social_geonote2", post(s.SocialGeonote))
	// 删除留言（兼容别名，同 /account/delete_geonote）
	r.POST("/account/geonote/delete_geonote", post(s.DeleteGeonote))
	// 删除留言（兼容别名，同 /account/geonote/delete_geonote）
	r.POST("/account/delete_geonote", post(s.DeleteGeonote))
	// 删除问题举报留言（兼容别名，同 /account/delete_bug_report_geonote）
	r.POST("/account/geonote/delete_bug_report_geonote", post(s.DeleteBugReportGeonote))
	// 删除问题举报留言（兼容别名，同 /account/geonote/delete_bug_report_geonote）
	r.POST("/account/delete_bug_report_geonote", post(s.DeleteBugReportGeonote))
	// 编辑留言（兼容别名，同 /account/edit_message_geonote）
	r.POST("/account/geonote/edit_message_geonote", post(s.EditMessageGeonote))
	// 编辑留言（兼容别名，同 /account/geonote/edit_message_geonote）
	r.POST("/account/edit_message_geonote", post(s.EditMessageGeonote))
	// 请求若干问题举报留言（兼容别名，同 /account/request_n_bug_report_geonotes）
	r.POST("/account/geonote/request_n_bug_report_geonotes", post(s.RequestNBugReportGeonotes))
	// 请求若干问题举报留言（兼容别名，同 /account/geonote/request_n_bug_report_geonotes）
	r.POST("/account/request_n_bug_report_geonotes", post(s.RequestNBugReportGeonotes))
	// 创建社交动态（账号接口）
	r.POST("/account/social_feed/create", post(s.SocialFeedCreate))
	// 获取用户动态（账号接口）
	r.POST("/account/social_feed/get_user_feeds", post(s.AccountGetUserFeeds))
	// 获取精选动态（账号接口）
	r.POST("/account/social_feed/get_curated_feeds", post(s.AccountGetCuratedFeeds))
	// 获取动态回应（账号接口）
	r.POST("/account/social_feed/get_reactions", post(s.AccountGetFeedReactions))
	// 点赞动态（账号接口）
	r.POST("/account/social_feed/like", post(s.AccountFeedLike))
	// 删除动态（账号接口）
	r.POST("/account/social_feed/delete", post(s.AccountDeleteFeed))
	// 创建留影（v1 旧版）
	r.POST("/service/recording/api/v1/create", post(s.RecordingCreateV1))
	// 获取留影（v1 旧版）
	r.POST("/service/recording/api/v1/get", post(s.RecordingGetV1))
	// 创建留影（v2 新版）
	r.POST("/service/recording/api/v2/create", post(s.RecordingCreateV2))
	// 更新留影（v2 新版）
	r.POST("/service/recording/api/v2/update", post(s.RecordingUpdateV2))
	// 获取留影（v2 新版）
	r.POST("/service/recording/api/v2/get", post(s.RecordingGetV2))
	// 获取精选动态（服务接口）
	r.POST("/service/social-feed/api/v2/get_curated_feeds", post(s.ServiceGetCuratedFeeds))
	// 获取用户动态（服务接口）
	r.POST("/service/social-feed/api/v1/get_user_feeds", post(s.ServiceGetUserFeeds))
	// 按 ID 批量获取动态
	r.POST("/service/social-feed/api/v1/get_by_ids", post(s.ServiceGetByIDs))
	// 创建社交动态（服务接口）
	r.POST("/service/social-feed/api/v1/create", post(s.ServiceSocialFeedCreate))
	// 隐藏动态
	r.POST("/service/social-feed/api/v1/hide", post(s.ServiceFeedHide))
	// 点赞动态（服务接口）
	r.POST("/service/social-feed/api/v1/like", post(s.ServiceFeedLike))
	// 超级点赞动态
	r.POST("/service/social-feed/api/v1/super_like", post(s.ServiceFeedSuperLike))
	// 删除动态（服务接口）
	r.POST("/service/social-feed/api/v1/delete", post(s.ServiceFeedDelete))
	// 编辑动态
	r.POST("/service/social-feed/api/v1/edit", post(s.ServiceFeedEdit))
	// 获取动态回应（服务接口）
	r.POST("/service/social-feed/api/v1/get_reactions", post(s.ServiceGetFeedReactions))
	// 社交动态评论占位
	r.POST("/service/social-feed/api/v2/comments", post(s.SocialFeedComments))
	// 社交动态举报占位
	r.POST("/service/social-feed/api/v1/report", post(s.SocialFeedReport))
	// 获取笔友列表（占位）
	r.POST("/service/social-feed/api/v2/get_penpals", post(s.ServiceGetPenpals))
	// 按 ID 获取笔友
	r.POST("/service/social-feed/api/v2/get_penpal_by_id", post(s.ServiceGetPenpalByID))
}
