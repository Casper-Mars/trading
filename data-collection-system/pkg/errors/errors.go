package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode 错误码类型
type ErrorCode int

// 定义错误码常量
const (
	// 系统级错误 (1000-1999)
	ErrCodeSystem ErrorCode = 1000 + iota
	ErrCodeDatabase
	ErrCodeRedis
	ErrCodeConfig
	ErrCodeLogger
	ErrCodeNetwork
	ErrCodeTimeout
	ErrCodeUnknown

	// 业务级错误 (2000-2999)
	ErrCodeBusiness ErrorCode = 2000 + iota
	ErrCodeInvalidParam
	ErrCodeDataNotFound
	ErrCodeDataExists
	ErrCodeDataInvalid
	ErrCodePermissionDenied
	ErrCodeOperationFailed

	// 数据采集错误 (3000-3999)
	ErrCodeCollection ErrorCode = 3000 + iota
	ErrCodeDataSourceUnavailable
	ErrCodeDataParsingFailed
	ErrCodeDataValidationFailed
	ErrCodeRateLimitExceeded
	ErrCodeAPIKeyInvalid
	ErrCodeQuotaExceeded
)

// AppError 应用错误结构
type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	HTTPStatus int       `json:"-"`
	Cause      error     `json:"-"`
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 支持errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Cause
}

// New 创建新的应用错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatus(code),
	}
}

// Newf 创建格式化消息的应用错误
func Newf(code ErrorCode, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: getHTTPStatus(code),
	}
}

// Wrap 包装已有错误
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Cause:      err,
		HTTPStatus: getHTTPStatus(code),
	}
}

// Wrapf 包装已有错误并格式化消息
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		Cause:      err,
		HTTPStatus: getHTTPStatus(code),
	}
}

// WithDetails 添加详细信息
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithDetailsf 添加格式化详细信息
func (e *AppError) WithDetailsf(format string, args ...interface{}) *AppError {
	e.Details = fmt.Sprintf(format, args...)
	return e
}

// getHTTPStatus 根据错误码获取HTTP状态码
func getHTTPStatus(code ErrorCode) int {
	switch {
	case code >= 1000 && code < 2000:
		// 系统级错误
		switch code {
		case ErrCodeTimeout:
			return http.StatusRequestTimeout
		case ErrCodeNetwork:
			return http.StatusBadGateway
		default:
			return http.StatusInternalServerError
		}
	case code >= 2000 && code < 3000:
		// 业务级错误
		switch code {
		case ErrCodeInvalidParam:
			return http.StatusBadRequest
		case ErrCodeDataNotFound:
			return http.StatusNotFound
		case ErrCodeDataExists:
			return http.StatusConflict
		case ErrCodePermissionDenied:
			return http.StatusForbidden
		default:
			return http.StatusBadRequest
		}
	case code >= 3000 && code < 4000:
		// 数据采集错误
		switch code {
		case ErrCodeDataSourceUnavailable:
			return http.StatusServiceUnavailable
		case ErrCodeRateLimitExceeded:
			return http.StatusTooManyRequests
		case ErrCodeAPIKeyInvalid:
			return http.StatusUnauthorized
		case ErrCodeQuotaExceeded:
			return http.StatusPaymentRequired
		default:
			return http.StatusBadRequest
		}
	default:
		return http.StatusInternalServerError
	}
}

// IsAppError 检查是否为应用错误
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError 获取应用错误
func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return nil
}

// 预定义的常用错误
var (
	ErrSystemError        = New(ErrCodeSystem, "系统错误")
	ErrDatabaseError      = New(ErrCodeDatabase, "数据库错误")
	ErrRedisError         = New(ErrCodeRedis, "Redis错误")
	ErrConfigError        = New(ErrCodeConfig, "配置错误")
	ErrNetworkError       = New(ErrCodeNetwork, "网络错误")
	ErrTimeoutError       = New(ErrCodeTimeout, "请求超时")
	ErrInvalidParam       = New(ErrCodeInvalidParam, "参数无效")
	ErrDataNotFound       = New(ErrCodeDataNotFound, "数据不存在")
	ErrDataExists         = New(ErrCodeDataExists, "数据已存在")
	ErrPermissionDenied   = New(ErrCodePermissionDenied, "权限不足")
	ErrOperationFailed    = New(ErrCodeOperationFailed, "操作失败")
	ErrDataSourceError    = New(ErrCodeDataSourceUnavailable, "数据源不可用")
	ErrDataParsingError   = New(ErrCodeDataParsingFailed, "数据解析失败")
	ErrRateLimitError     = New(ErrCodeRateLimitExceeded, "请求频率超限")
	ErrAPIKeyError        = New(ErrCodeAPIKeyInvalid, "API密钥无效")
	ErrQuotaError         = New(ErrCodeQuotaExceeded, "配额已用完")
)