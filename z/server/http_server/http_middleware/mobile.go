package http_middleware

import (
	"github.com/gin-gonic/gin"
	"strings"
)

// MobileDetectMiddleware 移动端检测中间件
func MobileDetectMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		isMobile := false
		ua := strings.ToLower(c.GetHeader("User-Agent"))
		if ua == "" {
			isMobile = false
		}

		mobileKeywords := []string{
			"iphone", "ipad", "android", "mobile", "windows phone",
			"opera mobi", "opera mini", "blackberry", "nokia", "samsung",
			"miui", "huawei", "honor", "vivo", "oppo",
		}

		for _, kw := range mobileKeywords {
			if strings.Contains(ua, kw) {
				isMobile = true
			}
		}
		isMobile = false

		c.Set("isMobile", isMobile)
		c.Next()
	}
}
