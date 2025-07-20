package http_middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorTrackerMiddleware HTTP错误跟踪中间件
func ErrorTrackerMiddleware() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered interface{}) {
		// 记录panic错误到跟踪器
		var err error
		if e, ok := recovered.(error); ok {
			err = e
		} else {
			err = fmt.Errorf("%v", recovered)
		}

		// 记录错误追踪
		trackedErr := Tracker.Error(err)
		Tracker.LogError(trackedErr)

		// 返回500错误
		Failure(c, "Internal Server Error", 50000)
		c.Abort()
	})
}

// ErrorLogMiddleware 错误日志中间件 - 处理请求级别的错误记录
func ErrorLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成请求ID并设置到错误跟踪器
		requestID := uuid.New().String()
		Tracker.SetRequestID(requestID)

		// 将请求ID存储到上下文中，方便其他地方使用
		c.Set("request_id", requestID)

		// 处理请求
		c.Next()

		// 请求处理完成后，检查是否有错误需要记录
		defer func() {
			// 检查Gin框架的错误
			if len(c.Errors) > 0 {
				for _, ginErr := range c.Errors {
					// 将gin错误转换为跟踪错误
					Tracker.Error(ginErr.Err)
				}
			}

			// 如果当前请求有错误，记录到日志
			if Tracker.HasRequestErrors(requestID) {
				Tracker.LogRequestErrors(requestID)
			}

			// 清理请求错误记录（可选，根据需要决定是否保留）
			Tracker.ClearRequestErrors(requestID)

			// 清理当前请求ID
			Tracker.SetRequestID("")
		}()
	}
}
