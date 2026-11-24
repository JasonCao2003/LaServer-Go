// messaging 业务 HTTP 路由。
package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/messaging"
	"github.com/gin-gonic/gin"
)

// 注册 messaging 服务处理的路径与方法。
func registerMessaging(r *gin.Engine, s *messaging.Service) {
	// 获取待处理消息
	r.POST("/account/get_pending_messages", post(s.GetPendingMessages))
	// 发送聊天
	r.POST("/account/chat/send", post(s.ChatSend))
	// 发送账号消息或礼物（兼容别名，同 /account/send_message）
	r.POST("/account/send", post(s.AccountSend))
	// 发送账号消息或礼物（兼容别名，同 /account/send）
	r.POST("/account/send_message", post(s.AccountSend))
	// 翻译占位接口
	r.POST("/account/translate", post(s.Translate))
	// 设置设备令牌（占位）
	r.POST("/account/set_device_token", post(s.SetDeviceToken))
	// 赠送蜡烛
	r.POST("/account/give_candle", post(s.GiveCandle))
	// 领取消息礼物（兼容别名，同 /account/claim_message_gifts）
	r.POST("/account/claim_message_gift", post(s.ClaimMessageGift))
	// 领取消息礼物（兼容别名，同 /account/claim_message_gift）
	r.POST("/account/claim_message_gifts", post(s.ClaimMessageGift))
	// 获取全部消息（v2 新版）
	r.POST("/service/message/api/v2/get_all", post(s.MessageGetAllV2))
	// 获取全部消息（v1 旧版）
	r.POST("/service/message/api/v1/get_all", post(s.MessageGetAllV1))
	// 获取单条消息
	r.POST("/service/message/api/v1/get", post(s.MessageGet))
	// 标记消息已接收
	r.POST("/service/message/api/v1/mark_received", post(s.MarkMessagesReceived))
	// 标记消息已读
	r.POST("/service/message/api/v1/mark_seen", post(s.MarkMessagesSeen))
	// 获取待领取免费礼物
	r.POST("/service/relationship/api/v1/free_gifts/get_pending", post(s.FreeGiftsGetPending))
	// 发送免费礼物
	r.POST("/service/relationship/api/v1/free_gifts/send", post(s.FreeGiftsSend))

	r.OPTIONS("/service/relationship/api/v1/free_gifts/send", preflight("POST,OPTIONS", "Content-Type,Authorization,X-Requested-With", "3600"))
	// 领取免费礼物
	r.POST("/service/relationship/api/v1/free_gifts/claim", post(s.FreeGiftsClaim))
}
