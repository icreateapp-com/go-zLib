package routes

import (
	"github.com/gin-gonic/gin"
)

// Guard 设置 guard 值到 gin.RouterGroup
func Guard(group *gin.RouterGroup, guards string) *gin.RouterGroup {
	if group == nil {
		return nil
	}
	// 仅负责写入 guard，鉴权由 AuthMiddleware 处理
	group.Use(func(c *gin.Context) {
		c.Set("guard", guards)
		c.Next()
	})
	return group
}
