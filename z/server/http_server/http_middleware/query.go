package http_middleware

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	. "github.com/icreateapp-com/go-zLib/z/db"
	"net/http"
)

func QueryToJsonMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			queryStr := c.Query("query")
			if len(queryStr) > 0 {
				var query Query
				if err := json.Unmarshal([]byte(queryStr), &query); err != nil {
					c.JSON(http.StatusUnauthorized, gin.H{
						"success": false,
						"message": fmt.Sprintf("Query must be an json string: %s", err.Error()),
						"code":    20000,
					})
					c.Abort()
					return
				}
				c.Set("query", query)
			}
		}
		c.Next()
	}
}
