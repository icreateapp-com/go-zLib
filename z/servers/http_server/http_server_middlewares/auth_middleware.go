package http_server_middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/providers/auth_provider"
	"go.uber.org/fx"
)

const SkipAuthContextKey = "skip_auth"

type WrappedGroup struct {
	group *gin.RouterGroup
}

func WrapGroup(group *gin.RouterGroup) *WrappedGroup {
	return &WrappedGroup{group: group}
}

func (wg *WrappedGroup) Guard(guards string) *gin.RouterGroup {
	if wg == nil || wg.group == nil {
		return nil
	}
	// 仅负责写入 guard，鉴权由 AuthMiddleware 处理
	wg.group.Use(func(c *gin.Context) {
		c.Set("guard", guards)
		c.Next()
	})
	return wg.group
}

// AuthMiddleware HTTP认证中间件
func AuthMiddleware(ap *auth_provider.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ap == nil {
			c.Next()
			return
		}

		// 使用 Auth 的统一认证方法
		success, _, err := ap.Authenticate(c)

		if !success {
			// 处理友好的错误消息
			var errorMsg string
			var errorCode string

			if authErr, ok := err.(*auth_provider.AuthError); ok {
				errorMsg = authErr.Message
				errorCode = authErr.Code
			} else {
				// 如果不是AuthError，转换为友好错误
				friendlyErr := auth_provider.ConvertToFriendlyError(err)
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

var AuthMiddlewareModule = fx.Provide(
	fx.Annotate(
		AuthMiddleware,
		fx.ParamTags(``),
		fx.ResultTags(`group:"http_middlewares"`),
	),
)
