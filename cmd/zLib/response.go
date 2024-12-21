package zLib

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Json 函数用于返回JSON格式的数据
func Json(c *gin.Context, obj any) {
	c.JSON(http.StatusOK, obj)
}

// Success 函数用于返回成功信息
func Success(c *gin.Context, message any, code ...int) {
	if len(code) == 0 {
		code = append(code, 10000)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": message, "code": code[0]})
}

// Failure 函数用于返回失败信息
func Failure(c *gin.Context, message any, code ...int) {
	if len(code) == 0 {
		code = append(code, 20000)
	}
	c.JSON(http.StatusOK, gin.H{"success": false, "message": message, "code": code[0]})
}
