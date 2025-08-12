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

// convertToInt 将参数转换为 int 类型，支持 Status 和 int 类型
func convertToInt(value interface{}) int {
	if status, ok := value.(Status); ok {
		return int(status)
	}
	if intVal, ok := value.(int); ok {
		return intVal
	}
	return 200 // 默认值
}

func response(responses []interface{}) (interface{}, int, int) {
	var message interface{}
	var code int
	var httpStatus int

	if len(responses) == 0 {
		message = nil
		code = 200
		httpStatus = http.StatusOK
	} else if len(responses) == 1 {
		// 检查第一个参数是否为 error 类型
		if err, ok := responses[0].(error); ok {
			message = err.Error() // 自动调用 Error() 方法
		} else {
			message = responses[0]
		}
		code = 200
		httpStatus = http.StatusOK
	} else if len(responses) == 2 {
		// 检查第一个参数是否为 error 类型
		if err, ok := responses[0].(error); ok {
			message = err.Error() // 自动调用 Error() 方法
		} else {
			message = responses[0]
		}
		code = convertToInt(responses[1])
		httpStatus = http.StatusOK
	} else {
		// 检查第一个参数是否为 error 类型
		if err, ok := responses[0].(error); ok {
			message = err.Error() // 自动调用 Error() 方法
		} else {
			message = responses[0]
		}
		code = convertToInt(responses[1])
		httpStatus = convertToInt(responses[2])
	}

	return message, code, httpStatus
}

// Success 函数用于返回成功信息
func Success(c *gin.Context, responses ...interface{}) {
	message, code, httpStatus := response(responses)
	c.JSON(httpStatus, Response{Success: true, Message: message, Code: code})
}

// Failure 函数用于返回失败信息
func Failure(c *gin.Context, responses ...interface{}) {
	message, code, httpStatus := response(responses)
	c.JSON(httpStatus, Response{Success: false, Message: message, Code: code})
}
