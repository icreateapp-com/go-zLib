package http_middleware

import (
	"github.com/gin-gonic/gin"
	. "github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/provider/auth_provider"
	"github.com/icreateapp-com/go-zLib/z/provider/event_bus_provider"
	"net/http"
	"strings"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// skip anonymity url - 始终优先检查匿名路径
		if skips, err := Config.StringSlice("config.auth.anonymity"); err == nil {
			if c.Request.URL.Path == "/" {
				c.Next()
				return
			}
			for _, v := range skips {
				if strings.HasPrefix(c.Request.URL.Path, v) {
					c.Next()
					return
				}
			}
		}

		// 使用 auth_provider 从上下文验证 token
		if userID, isValid := auth_provider.AuthProvider.Verify(c); isValid {
			// token验证成功，放行
			event_bus_provider.EmitAsync[map[string]interface{}]("app.auth.verify", map[string]interface{}{"user_id": userID, "logined": true})
			// 将用户ID存储到上下文中，供后续使用
			c.Set("user_id", userID)
			c.Next()
			return
		} else {
			// token验证失败
			event_bus_provider.EmitAsync[map[string]interface{}]("app.auth.verify", map[string]interface{}{"user_id": userID, "logined": false})
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Authentication failed or session expired", "code": 20000})
			c.Abort()
			return
		}
	}
}
