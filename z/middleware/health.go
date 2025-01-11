package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z"
	"time"
)

func HealthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/.well-known/alive" {
			z.Success(c, map[string]interface{}{
				"status":    "UP",
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
		}

		if c.Request.URL.Path == "/.well-known/health" {
			host, _ := z.Config.String("config.host")
			port, _ := z.Config.Int("config.port")
			name, _ := z.Config.String("config.name")
			z.Success(c, map[string]interface{}{
				"status":    "UP",
				"timestamp": time.Now().Unix(),
				"name":      name,
				"host":      fmt.Sprintf("%s:%d", host, port),
			})
			c.Abort()
		}

		c.Next()
	}
}
