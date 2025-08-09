package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"data-collection-system/pkg/errors"
	"data-collection-system/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ValidateParams 参数验证中间件
func ValidateParams() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 验证分页参数
		if err := validatePaginationParams(c); err != nil {
			response.Error(c, err)
			c.Abort()
			return
		}

		// 验证日期格式参数
		if err := validateDateParams(c); err != nil {
			response.Error(c, err)
			c.Abort()
			return
		}

		c.Next()
	}
}

// validatePaginationParams 验证分页参数
func validatePaginationParams(c *gin.Context) error {
	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")

	if pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return errors.New(errors.ErrCodeInvalidParam, "页码必须是大于0的整数")
		}
	}

	if pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 || pageSize > 1000 {
			return errors.New(errors.ErrCodeInvalidParam, "每页大小必须是1-1000之间的整数")
		}
	}

	return nil
}

// validateDateParams 验证日期参数
func validateDateParams(c *gin.Context) error {
	dateParams := []string{"start_date", "end_date", "start_time", "end_time"}

	for _, param := range dateParams {
		value := c.Query(param)
		if value == "" {
			continue
		}

		// 验证日期格式
		if strings.Contains(param, "date") {
			if !isValidDateFormat(value) {
				return errors.Newf(errors.ErrCodeInvalidParam, "%s格式错误，应为YYYY-MM-DD", param)
			}
		}

		// 验证时间格式
		if strings.Contains(param, "time") {
			if !isValidTimeFormat(value) {
				return errors.Newf(errors.ErrCodeInvalidParam, "%s格式错误，应为YYYY-MM-DD HH:MM:SS", param)
			}
		}
	}

	return nil
}

// isValidDateFormat 验证日期格式 YYYY-MM-DD
func isValidDateFormat(date string) bool {
	if len(date) != 10 {
		return false
	}
	parts := strings.Split(date, "-")
	if len(parts) != 3 {
		return false
	}
	// 验证年份
	if len(parts[0]) != 4 {
		return false
	}
	// 验证月份
	month, err := strconv.Atoi(parts[1])
	if err != nil || month < 1 || month > 12 {
		return false
	}
	// 验证日期
	day, err := strconv.Atoi(parts[2])
	if err != nil || day < 1 || day > 31 {
		return false
	}
	return true
}

// isValidTimeFormat 验证时间格式 YYYY-MM-DD HH:MM:SS
func isValidTimeFormat(datetime string) bool {
	if len(datetime) != 19 {
		return false
	}
	parts := strings.Split(datetime, " ")
	if len(parts) != 2 {
		return false
	}
	// 验证日期部分
	if !isValidDateFormat(parts[0]) {
		return false
	}
	// 验证时间部分
	timeParts := strings.Split(parts[1], ":")
	if len(timeParts) != 3 {
		return false
	}
	// 验证小时
	hour, err := strconv.Atoi(timeParts[0])
	if err != nil || hour < 0 || hour > 23 {
		return false
	}
	// 验证分钟
	minute, err := strconv.Atoi(timeParts[1])
	if err != nil || minute < 0 || minute > 59 {
		return false
	}
	// 验证秒
	second, err := strconv.Atoi(timeParts[2])
	if err != nil || second < 0 || second > 59 {
		return false
	}
	return true
}

// ValidateStockSymbol 验证股票代码格式
func ValidateStockSymbol() gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := c.Param("symbol")
		if symbol == "" {
			symbol = c.Query("symbol")
		}

		if symbol != "" && !isValidStockSymbol(symbol) {
			response.Error(c, errors.New(errors.ErrCodeInvalidParam, "股票代码格式错误"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// isValidStockSymbol 验证股票代码格式
func isValidStockSymbol(symbol string) bool {
	// 股票代码格式：6位数字.交易所代码
	// 例如：000001.SZ, 600000.SH
	parts := strings.Split(symbol, ".")
	if len(parts) != 2 {
		return false
	}

	// 验证股票代码部分（6位数字）
	code := parts[0]
	if len(code) != 6 {
		return false
	}
	for _, char := range code {
		if char < '0' || char > '9' {
			return false
		}
	}

	// 验证交易所代码
	exchange := parts[1]
	validExchanges := []string{"SH", "SZ", "BJ"} // 上海、深圳、北京
	for _, validExchange := range validExchanges {
		if exchange == validExchange {
			return true
		}
	}

	return false
}

// ValidateJSON 验证JSON请求体
func ValidateJSON(obj interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
			if err := c.ShouldBindJSON(obj); err != nil {
				if validationErrors, ok := err.(validator.ValidationErrors); ok {
					// 处理验证错误
					errorMsg := formatValidationErrors(validationErrors)
					response.Error(c, errors.New(errors.ErrCodeInvalidParam, errorMsg))
				} else {
					// JSON格式错误
					response.Error(c, errors.New(errors.ErrCodeInvalidParam, "请求体格式错误"))
				}
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// formatValidationErrors 格式化验证错误信息
func formatValidationErrors(errors validator.ValidationErrors) string {
	var messages []string
	for _, err := range errors {
		switch err.Tag() {
		case "required":
			messages = append(messages, err.Field()+"是必填字段")
		case "min":
			messages = append(messages, err.Field()+"最小值为"+err.Param())
		case "max":
			messages = append(messages, err.Field()+"最大值为"+err.Param())
		case "email":
			messages = append(messages, err.Field()+"必须是有效的邮箱地址")
		default:
			messages = append(messages, err.Field()+"格式错误")
		}
	}
	return strings.Join(messages, ", ")
}

// SecurityCheck 安全检查中间件
func SecurityCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查SQL注入
		if err := checkSQLInjection(c); err != nil {
			response.Error(c, err)
			c.Abort()
			return
		}

		// 检查XSS攻击
		if err := checkXSS(c); err != nil {
			response.Error(c, err)
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkSQLInjection 检查SQL注入
func checkSQLInjection(c *gin.Context) error {
	sqlKeywords := []string{
		"select", "insert", "update", "delete", "drop", "create", "alter",
		"union", "or", "and", "where", "order", "group", "having",
		"exec", "execute", "sp_", "xp_", "--", "/*", "*/",
	}

	// 检查查询参数
	for key, values := range c.Request.URL.Query() {
		for _, value := range values {
			lowerValue := strings.ToLower(value)
			for _, keyword := range sqlKeywords {
				if strings.Contains(lowerValue, keyword) {
					return errors.Newf(errors.ErrCodeInvalidParam, "参数%s包含非法字符", key)
				}
			}
		}
	}

	return nil
}

// checkXSS 检查XSS攻击
func checkXSS(c *gin.Context) error {
	xssPatterns := []string{
		"<script", "</script>", "javascript:", "onload=", "onerror=",
		"onclick=", "onmouseover=", "onfocus=", "onblur=",
	}

	// 检查查询参数
	for key, values := range c.Request.URL.Query() {
		for _, value := range values {
			lowerValue := strings.ToLower(value)
			for _, pattern := range xssPatterns {
				if strings.Contains(lowerValue, pattern) {
					return errors.Newf(errors.ErrCodeInvalidParam, "参数%s包含非法字符", key)
				}
			}
		}
	}

	return nil
}
