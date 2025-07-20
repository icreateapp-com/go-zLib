package z

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool `json:"success"`
	Message any  `json:"message"`
	Code    int  `json:"code"`
}

// Json 函数用于返回JSON格式的数据
func Json(c *gin.Context, obj any) {
	c.JSON(http.StatusOK, obj)
}

// Success 函数用于返回成功信息
func Success(c *gin.Context, message any, code ...int) {
	if len(code) == 0 {
		code = append(code, 10000)
	}
	c.JSON(http.StatusOK, Response{Success: true, Message: message, Code: code[0]})
}

// Failure 函数用于返回失败信息
func Failure(c *gin.Context, message any, code ...int) {
	if len(code) == 0 {
		code = append(code, 20000)
	}
	if msg, ok := message.(string); ok && msg == "EOF" {
		message = "the body must be a json"
	}
	c.JSON(http.StatusOK, Response{Success: false, Message: message, Code: code[0]})
}
