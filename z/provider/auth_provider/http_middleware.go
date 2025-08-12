package auth_provider

import (
	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z"
)

// HttpAuthProviderMiddleware HTTP认证中间件
func HttpAuthProviderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用AuthProvider的统一认证方法
		success, _, err := AuthProvider.Authenticate(c)

		if !success {
			// 处理友好的错误消息
			var errorMsg string
			var errorCode string

			if authErr, ok := err.(*AuthError); ok {
				errorMsg = authErr.Message
				errorCode = authErr.Code
			} else {
				// 如果不是AuthError，转换为友好错误
				friendlyErr := convertToFriendlyError(err)
				errorMsg = friendlyErr.Message
				errorCode = friendlyErr.Code
			}

			// 返回结构化的错误响应
			z.Failure(c, map[string]interface{}{
				"error":   errorCode,
				"message": errorMsg,
			}, z.StatusUnauthorized, 401)
			c.Abort()
			return
		}

		// 认证成功或无需认证，继续处理
		c.Next()
	}
}
