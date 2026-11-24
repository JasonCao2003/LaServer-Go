// 留言、动态、留影与社区路由注册

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册社区动态路由
func registerCommunity(r *gin.Engine, s *service.Service) {
	r.POST("/account/get_user_geonotes", post(s.GetUserGeonotes))
	r.POST("/account/geonote/get_user_geonotes", post(s.GetUserGeonotes))
	r.POST("/account/geonote/request_geonotes_for_n_friends", post(s.RequestGeonotesForNFriends))
	r.POST("/account/geonote/create_geonote2", post(s.CreateGeonote))
	r.POST("/account/create_geonote2", post(s.CreateGeonote))
	r.POST("/account/geonote/create_bug_report_geonote", post(s.CreateBugReportGeonote))
	r.POST("/account/create_bug_report_geonote", post(s.CreateBugReportGeonote))
	r.POST("/account/geonote/social_geonote2", post(s.SocialGeonote))
	r.POST("/account/social_geonote2", post(s.SocialGeonote))
	r.POST("/account/geonote/delete_geonote", post(s.DeleteGeonote))
	r.POST("/account/delete_geonote", post(s.DeleteGeonote))
	r.POST("/account/geonote/delete_bug_report_geonote", post(s.DeleteBugReportGeonote))
	r.POST("/account/delete_bug_report_geonote", post(s.DeleteBugReportGeonote))
	r.POST("/account/geonote/edit_message_geonote", post(s.EditMessageGeonote))
	r.POST("/account/edit_message_geonote", post(s.EditMessageGeonote))
	r.POST("/account/geonote/request_n_bug_report_geonotes", post(s.RequestNBugReportGeonotes))
	r.POST("/account/request_n_bug_report_geonotes", post(s.RequestNBugReportGeonotes))
	r.POST("/account/social_feed/create", post(s.SocialFeedCreate))
	r.POST("/account/social_feed/get_user_feeds", post(s.AccountGetUserFeeds))
	r.POST("/account/social_feed/get_curated_feeds", post(s.AccountGetCuratedFeeds))
	r.POST("/account/social_feed/get_reactions", post(s.AccountGetFeedReactions))
	r.POST("/account/social_feed/like", post(s.AccountFeedLike))
	r.POST("/account/social_feed/delete", post(s.AccountDeleteFeed))
	r.POST("/service/recording/api/v1/create", post(s.RecordingCreateV1))
	r.POST("/service/recording/api/v1/get", post(s.RecordingGetV1))
	r.POST("/service/recording/api/v2/create", post(s.RecordingCreateV2))
	r.POST("/service/recording/api/v2/update", post(s.RecordingUpdateV2))
	r.POST("/service/recording/api/v2/get", post(s.RecordingGetV2))
	r.POST("/service/social-feed/api/v2/get_curated_feeds", post(s.ServiceGetCuratedFeeds))
	r.POST("/service/social-feed/api/v1/get_user_feeds", post(s.ServiceGetUserFeeds))
	r.POST("/service/social-feed/api/v1/get_by_ids", post(s.ServiceGetByIDs))
	r.POST("/service/social-feed/api/v1/create", post(s.ServiceSocialFeedCreate))
	r.POST("/service/social-feed/api/v1/hide", post(s.ServiceFeedHide))
	r.POST("/service/social-feed/api/v1/like", post(s.ServiceFeedLike))
	r.POST("/service/social-feed/api/v1/super_like", post(s.ServiceFeedSuperLike))
	r.POST("/service/social-feed/api/v1/delete", post(s.ServiceFeedDelete))
	r.POST("/service/social-feed/api/v1/edit", post(s.ServiceFeedEdit))
	r.POST("/service/social-feed/api/v1/get_reactions", post(s.ServiceGetFeedReactions))
	r.POST("/service/social-feed/api/v2/comments", post(s.SocialFeedComments))
	r.POST("/service/social-feed/api/v1/report", post(s.SocialFeedReport))
	r.POST("/service/social-feed/api/v2/get_penpals", post(s.ServiceGetPenpals))
	r.POST("/service/social-feed/api/v2/get_penpal_by_id", post(s.ServiceGetPenpalByID))
}
