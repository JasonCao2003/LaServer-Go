// 房间预约、舞台与管理后台路由注册

package handler

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 注册房间舞台与管理路由
func registerRoomsAndAdmin(r *gin.Engine, s *service.Service) {
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
	// 列出管理员好友
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
