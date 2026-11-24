// commerce 业务 HTTP 路由。
package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/commerce"
	"github.com/gin-gonic/gin"
)

// 注册 commerce 服务处理的路径与方法。
func registerCommerce(r *gin.Engine, s *commerce.Service) {
	// 获取精灵商店
	r.POST("/account/get_spirit_shops", post(s.SpiritShops))
	// 确认奖励（占位）
	r.POST("/account/redeem_reward", post(s.RedeemReward))
	// 确认锻造概率重置（占位）
	r.POST("/account/ack_forge_rate_reset", post(s.AckForgeRateReset))
	// 更新商城信息（占位）
	r.POST("/account/commerce/update", post(s.CommerceUpdate))
	// 获取商品信息
	r.POST("/account/commerce/product_info", post(s.CommerceProductInfo))
	// 获取商品信息
	r.GET("/account/commerce/product_info", func(c *gin.Context) { respond(c, s.CommerceProductInfo, object{}) })
	// 获取商店
	r.POST("/account/get_shop", post(s.GetShop))
	// 获取战利品箱
	r.POST("/account/lootboxes/get", post(s.GetLootboxes))
	// 校验邀请码
	r.POST("/account/code/verify", post(s.VerifyCode))
	// 购买活动增益
	r.POST("/account/purchase_event_buff", post(s.PurchaseEventBuff))
	// 购买染料
	r.POST("/account/purchase_dye", post(s.PurchaseDye))
	// 购买通用商店物品
	r.POST("/account/purchase_generic_shop_item", post(s.PurchaseGenericShopItem))
	// 购买精灵商店物品
	r.POST("/account/purchase_spirit_shop_item", post(s.PurchaseSpiritShopItem))
	// 兑换战利品箱
	r.POST("/account/lootboxes/redeem_box", post(s.RedeemBox))
	// 购买蜡烛蜡
	r.POST("/account/buy_candle_wax", post(s.BuyCandleWax))
	// 购买解锁
	r.POST("/account/purchase_unlock", post(s.PurchaseUnlock))
	// 处理交易收据
	r.POST("/account/commerce/receipt", post(s.CommerceReceipt))
	// 处理交易履约
	r.POST("/account/commerce/fulfill", post(s.CommerceFulfill))
}
