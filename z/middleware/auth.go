package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	. "github.com/icreateapp-com/go-zLib/z"
	"net/http"
	"regexp"
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

		// 根据访问域判断采样那个 token 进行验证
		re := regexp.MustCompile(`/api/([^/]+)`)
		domain := re.FindStringSubmatch(c.Request.URL.Path)
		if len(domain) >= 2 {
			configToken, _ := Config.String(fmt.Sprintf("config.auth.%s", domain[1]))
			if inputToken == configToken {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Access token error", "code": 20000})
		c.Abort()
	}
}
