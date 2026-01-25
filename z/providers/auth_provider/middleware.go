package auth_provider

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// WrappedGroup 包装 gin.RouterGroup，提供 Guard 方法
type WrappedGroup struct {
	group *gin.RouterGroup
}

// WrapGroup 创建 WrappedGroup 实例
func WrapGroup(group *gin.RouterGroup) *WrappedGroup {
	return &WrappedGroup{group: group}
}

// Guard 设置 guard 值到 gin.Context
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
func AuthMiddleware(ap *Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ap == nil {
			c.Next()
			return
		}

		// 跳过 OPTIONS 请求（CORS 预检）
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// FIX: 修复 Guard 与 AuthMiddleware 执行顺序不确定导致 guard 为空的问题
		// 如果未读取到 guard，则回退到从 gin.Context 路径前缀推导 guard（利用 Auth.AuthenticateRequest 的 matchGuard 逻辑），并写回 c.Set("guard", guardName) 以兼容旧路由注册顺序
		if guardRaw, ok := c.Get("guard"); !ok || guardRaw == nil {
			path := ""
			if c.Request != nil && c.Request.URL != nil {
				path = c.Request.URL.Path
			}

			// 尝试从路径中推导 guard
			_, guardName, _, _ := ap.AuthenticateRequest(path, "", "")
			if guardName != "" {
				c.Set("guard", guardName)
			}
		}

		// 使用 Auth 的统一认证方法
		success, _, err := ap.Authenticate(c)

		if !success {
			// 处理友好的错误消息
			var errorMsg string

			if authErr, ok := err.(*AuthError); ok {
				errorMsg = authErr.Message
			} else {
				// 如果不是AuthError，转换为友好错误
				friendlyErr := ConvertToFriendlyError(err)
				errorMsg = friendlyErr.Message
			}

			// 返回结构化的错误响应
			c.JSON(401, gin.H{
				"success": false,
				"message": errorMsg,
				"code":    401,
			})
			c.Abort()
			return
		}

		// 认证成功或无需认证，继续处理
		c.Next()
	}
}

// AuthMiddlewareModule fx 模块
var AuthMiddlewareModule = fx.Provide(
	fx.Annotate(
		AuthMiddleware,
		fx.ParamTags(``),
		fx.ResultTags(`group:"http_middlewares"`),
	),
)
