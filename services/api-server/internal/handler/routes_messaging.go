// 消息、聊天与礼物路由注册

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册消息礼物路由
func registerMessages(r *gin.Engine, s *service.Service) {
	r.POST("/account/get_pending_messages", post(s.GetPendingMessages))
	r.POST("/account/chat/send", post(s.ChatSend))
	r.POST("/account/send", post(s.AccountSend))
	r.POST("/account/send_message", post(s.AccountSend))
	r.POST("/account/translate", post(s.Translate))
	r.POST("/account/set_device_token", post(s.SetDeviceToken))
	r.POST("/account/give_candle", post(s.GiveCandle))
	r.POST("/account/claim_message_gift", post(s.ClaimMessageGift))
	r.POST("/account/claim_message_gifts", post(s.ClaimMessageGift))
	r.POST("/service/message/api/v2/get_all", post(s.MessageGetAllV2))
	r.POST("/service/message/api/v1/get_all", post(s.MessageGetAllV1))
	r.POST("/service/message/api/v1/get", post(s.MessageGet))
	r.POST("/service/message/api/v1/mark_received", post(s.MarkMessagesReceived))
	r.POST("/service/message/api/v1/mark_seen", post(s.MarkMessagesSeen))
	r.POST("/service/relationship/api/v1/free_gifts/get_pending", post(s.FreeGiftsGetPending))
	r.POST("/service/relationship/api/v1/free_gifts/send", post(s.FreeGiftsSend))
	r.OPTIONS("/service/relationship/api/v1/free_gifts/send", preflight("POST,OPTIONS", "Content-Type,Authorization,X-Requested-With", "3600"))
	r.POST("/service/relationship/api/v1/free_gifts/claim", post(s.FreeGiftsClaim))
}
