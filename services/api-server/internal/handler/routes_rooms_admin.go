// 房间预约、舞台与管理后台路由注册

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册房间舞台与管理路由
func registerRoomsAndAdmin(r *gin.Engine, s *service.Service) {
	r.POST("/service/reservation/api/v1/reserve", post(s.Reserve))
	r.POST("/service/stage/api/v1/get", post(s.StageGet))
	r.POST("/service/stage/api/v1/set", post(s.StageSet))
	r.POST("/service/stage/api/v1/friends_in_apartments", post(s.FriendsInApartments))
	r.POST("/service/stage/api/v1/leave_apartment", post(s.LeaveApartment))
	r.POST("/service/stage/api/v1/update_apartment_status", post(s.UpdateApartmentStatus))
	r.POST("/admin/api/friends_list", post(s.AdminFriendsList))
	r.GET("/admin/api/friends_list", adminQuery(s.AdminFriendsList))
	r.POST("/admin/api/friends_set", post(s.AdminFriendsSet))
	r.GET("/admin/api/friends_set", adminQuery(s.AdminFriendsSet))
	r.POST("/admin/api/friends_clear", post(s.AdminFriendsClear))
	r.GET("/admin/api/friends_clear", adminQuery(s.AdminFriendsClear))
}
