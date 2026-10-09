package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorCode 错误码类型
type ErrorCode int

// 错误编码定义
const (
	// 通用错误 1xxx
	CodeUnknown        ErrorCode = 1000
	CodeInvalidRequest ErrorCode = 1001
	CodeUnauthorized   ErrorCode = 1002
	CodeForbidden      ErrorCode = 1003
	CodeNotFound       ErrorCode = 1004
	CodeInternal       ErrorCode = 1005

	// 订单相关错误 2xxx
	CodeOrderNotFound      ErrorCode = 2001
	CodeOrderInvalidStatus ErrorCode = 2002
	CodeOrderMissingFields ErrorCode = 2003
	CodeOrderNoItems       ErrorCode = 2004
	CodeItemEmptyPartName  ErrorCode = 2005

	// 客户相关错误 3xxx
	CodeCustomerNotFound   ErrorCode = 3001
	CodeCustomerNameExists ErrorCode = 3002

	// 认证相关错误 4xxx
	CodeLoginFailed   ErrorCode = 4001
	CodeAccountLocked ErrorCode = 4002
	CodeTokenInvalid  ErrorCode = 4003
	CodeTokenExpired  ErrorCode = 4004
)

// 错误消息映射
var errorMsgs = map[ErrorCode]string{
	CodeUnknown:        "未知错误",
	CodeInvalidRequest: "请求参数无效",
	CodeUnauthorized:   "未授权访问",
	CodeForbidden:      "禁止访问",
	CodeNotFound:       "资源不存在",
	CodeInternal:       "服务器内部错误",

	CodeOrderNotFound:      "订单不存在",
	CodeOrderInvalidStatus: "订单状态无效",
	CodeOrderMissingFields: "订单缺少必填字段",
	CodeOrderNoItems:       "订单必须包含至少一个零件",
	CodeItemEmptyPartName:  "零件名称不能为空",

	CodeCustomerNotFound:   "客户不存在",
	CodeCustomerNameExists: "客户名称已存在",

	CodeLoginFailed:   "登录失败",
	CodeAccountLocked: "账户已锁定",
	CodeTokenInvalid:  "登录已过期，请重新登录",
	CodeTokenExpired:  "登录已过期，请重新登录",
}

// GetMessage 获取错误消息
func (code ErrorCode) GetMessage() string {
	if msg, ok := errorMsgs[code]; ok {
		return msg
	}
	return "未知错误"
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": data,
	})
}

// SuccessWithMessage 成功响应（带消息）
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": message,
		"data":    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, httpStatus int, data interface{}) {
	c.JSON(httpStatus, data)
}

// ErrorWithCode 带错误码的错误响应
func ErrorWithCode(c *gin.Context, httpStatus int, code ErrorCode, message string) {
	c.JSON(httpStatus, gin.H{
		"code":    code,
		"message": message,
	})
}

// ErrorCode 带错误码的错误响应（使用默认消息）
func ErrorCodeResp(c *gin.Context, httpStatus int, code ErrorCode) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"code":    code,
		"message": code.GetMessage(),
	})
}

// BadRequest 请求参数错误
func BadRequest(c *gin.Context, code ErrorCode, message string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"code":    code,
		"message": message,
	})
}

// Unauthorized 未授权
func Unauthorized(c *gin.Context, code ErrorCode, message string) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"code":    code,
		"message": message,
	})
}

// Forbidden 禁止访问
func Forbidden(c *gin.Context, code ErrorCode, message string) {
	c.JSON(http.StatusForbidden, gin.H{
		"code":    code,
		"message": message,
	})
}

// InternalError 服务器内部错误
func InternalError(c *gin.Context, code ErrorCode, message string) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"code":    code,
		"message": message,
	})
}
