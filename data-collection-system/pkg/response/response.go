package response

import (
	"net/http"
	"time"

	"data-collection-system/pkg/errors"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// PageResponse 分页响应结构
type PageResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// Pagination 分页信息
type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
	Pages    int   `json:"pages"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	response := Response{
		Code:      0,
		Message:   "success",
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}
	c.JSON(http.StatusOK, response)
}

// SuccessWithMessage 带自定义消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	response := Response{
		Code:      0,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}
	c.JSON(http.StatusOK, response)
}

// SuccessPage 分页成功响应
func SuccessPage(c *gin.Context, data interface{}, pagination *Pagination) {
	response := PageResponse{
		Code:       0,
		Message:    "success",
		Data:       data,
		Pagination: pagination,
		Timestamp:  time.Now().Unix(),
		RequestID:  getRequestID(c),
	}
	c.JSON(http.StatusOK, response)
}

// Error 错误响应
func Error(c *gin.Context, err error) {
	var code int
	var message string
	var httpStatus int

	if appErr := errors.GetAppError(err); appErr != nil {
		// 应用错误
		code = int(appErr.Code)
		message = appErr.Message
		httpStatus = appErr.HTTPStatus
		if appErr.Details != "" {
			message += ": " + appErr.Details
		}
	} else {
		// 普通错误
		code = int(errors.ErrCodeSystem)
		message = err.Error()
		httpStatus = http.StatusInternalServerError
	}

	response := Response{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}

	c.JSON(httpStatus, response)
}

// ErrorWithCode 带错误码的错误响应
func ErrorWithCode(c *gin.Context, code errors.ErrorCode, message string) {
	appErr := errors.New(code, message)
	Error(c, appErr)
}

// ErrorWithMessage 带自定义消息的错误响应
func ErrorWithMessage(c *gin.Context, httpStatus int, code int, message string) {
	response := Response{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}
	c.JSON(httpStatus, response)
}

// BadRequest 400错误响应
func BadRequest(c *gin.Context, message string) {
	ErrorWithCode(c, errors.ErrCodeInvalidParam, message)
}

// NotFound 404错误响应
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "资源不存在"
	}
	ErrorWithCode(c, errors.ErrCodeDataNotFound, message)
}

// Forbidden 403错误响应
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "权限不足"
	}
	ErrorWithCode(c, errors.ErrCodePermissionDenied, message)
}

// InternalServerError 500错误响应
func InternalServerError(c *gin.Context, message string) {
	if message == "" {
		message = "服务器内部错误"
	}
	ErrorWithCode(c, errors.ErrCodeSystem, message)
}

// TooManyRequests 429错误响应
func TooManyRequests(c *gin.Context, message string) {
	if message == "" {
		message = "请求过于频繁"
	}
	ErrorWithCode(c, errors.ErrCodeRateLimitExceeded, message)
}

// ServiceUnavailable 503错误响应
func ServiceUnavailable(c *gin.Context, message string) {
	if message == "" {
		message = "服务暂时不可用"
	}
	ErrorWithCode(c, errors.ErrCodeDataSourceUnavailable, message)
}

// getRequestID 获取请求ID
func getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("X-Request-ID"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// NewPagination 创建分页信息
func NewPagination(page, pageSize int, total int64) *Pagination {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	pages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		pages++
	}

	return &Pagination{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Pages:    pages,
	}
}

// ValidatePagination 验证分页参数
func ValidatePagination(page, pageSize int) (int, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		return 0, 0, errors.New(errors.ErrCodeInvalidParam, "页面大小不能超过100")
	}
	return page, pageSize, nil
}