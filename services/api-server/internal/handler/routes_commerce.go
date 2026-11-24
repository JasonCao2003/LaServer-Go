// 商城、购买与交易收据路由注册

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册商城交易路由
func registerCommerce(r *gin.Engine, s *service.Service) {
	r.POST("/account/commerce/update", post(s.CommerceUpdate))
	r.POST("/account/commerce/product_info", post(s.CommerceProductInfo))
	r.GET("/account/commerce/product_info", func(c *gin.Context) { respond(c, s.CommerceProductInfo, object{}) })
	r.POST("/account/get_shop", post(s.GetShop))
	r.POST("/account/lootboxes/get", post(s.GetLootboxes))
	r.POST("/account/code/verify", post(s.VerifyCode))
	r.POST("/account/purchase_event_buff", post(s.PurchaseEventBuff))
	r.POST("/account/purchase_dye", post(s.PurchaseDye))
	r.POST("/account/purchase_generic_shop_item", post(s.PurchaseGenericShopItem))
	r.POST("/account/purchase_spirit_shop_item", post(s.PurchaseSpiritShopItem))
	r.POST("/account/lootboxes/redeem_box", post(s.RedeemBox))
	r.POST("/account/buy_candle_wax", post(s.BuyCandleWax))
	r.POST("/account/purchase_unlock", post(s.PurchaseUnlock))
	r.POST("/account/commerce/receipt", post(s.CommerceReceipt))
	r.POST("/account/commerce/fulfill", post(s.CommerceFulfill))
}
