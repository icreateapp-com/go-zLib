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

func response(responses []interface{}) (interface{}, int) {
	var message interface{}
	var code int

	if len(responses) == 0 {
		message = nil
		code = 200
	} else if len(responses) == 1 {
		// 检查第一个参数是否为 error 类型
		if err, ok := responses[0].(error); ok {
			message = err.Error() // 自动调用 Error() 方法
		} else {
			message = responses[0]
		}
		code = 200
	} else {
		// 检查第一个参数是否为 error 类型
		if err, ok := responses[0].(error); ok {
			message = err.Error() // 自动调用 Error() 方法
		} else {
			message = responses[0]
		}
		code = responses[1].(int)
	}

	return message, code
}

// Success 函数用于返回成功信息
func Success(c *gin.Context, responses ...interface{}) {
	message, code := response(responses)
	c.JSON(http.StatusOK, Response{Success: true, Message: message, Code: code})
}

// Failure 函数用于返回失败信息
func Failure(c *gin.Context, responses ...interface{}) {
	message, code := response(responses)
	c.JSON(http.StatusOK, Response{Success: false, Message: message, Code: code})
}
