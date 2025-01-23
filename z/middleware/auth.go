package middleware

import (
	"github.com/gin-gonic/gin"
	. "github.com/icreateapp-com/go-zLib/z"
	"net/http"
	"strings"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// skip anonymity url
		if skips, err := Config.StringSlice("config.anonymity"); err == nil {
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

		// get token
		inputToken := c.Request.Header.Get("Authorization")
		if StringIsEmpty(inputToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Access token cannot be empty", "code": 20000})
			c.Abort()
			return
		}

		// 获取 auth 配置
		authConfig, err := Config.StringMap("config.auth")
		if err != nil {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// 标准化请求路径
		requestPath := strings.TrimPrefix(c.Request.URL.Path, "/")

		// 遍历 auth 配置，找到匹配的路径前缀
		for pathPrefix, configToken := range authConfig {
			normalizedPrefix := strings.TrimPrefix(pathPrefix, "/")
			if strings.HasPrefix(requestPath, normalizedPrefix) {
				if inputToken == configToken {
					c.Next()
					return
				} else {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					c.Abort()
					return
				}
			}
		}

		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Access token error", "code": 20000})
		c.Abort()
	}
}
