// stage 业务 HTTP 路由。
package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/stage"
	"github.com/gin-gonic/gin"
)

// 注册 stage 服务处理的路径与方法。
func registerStage(r *gin.Engine, s *stage.Service) {
	// 预约房间（占位）
	r.POST("/service/reservation/api/v1/reserve", post(s.Reserve))
	// 获取舞台
	r.POST("/service/stage/api/v1/get", post(s.StageGet))
	// 保存舞台
	r.POST("/service/stage/api/v1/set", post(s.StageSet))
	// 获取公寓内好友（占位）
	r.POST("/service/stage/api/v1/friends_in_apartments", post(s.FriendsInApartments))
	// 离开公寓（占位）
	r.POST("/service/stage/api/v1/leave_apartment", post(s.LeaveApartment))
	// 更新公寓状态（占位）
	r.POST("/service/stage/api/v1/update_apartment_status", post(s.UpdateApartmentStatus))
}
