package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"data-collection-system/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Validator 验证器实例
var validate *validator.Validate

// 预编译的正则表达式
var (
	stockCodeRegex = regexp.MustCompile(`^[0-9]{6}$`)
	dateRegex      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	emailRegex     = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRegex     = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

// ValidationError 验证错误
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// init 初始化验证器
func init() {
	validate = validator.New()

	// 注册自定义验证规则
	registerCustomValidators()

	// 注册自定义标签名称
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// registerCustomValidators 注册自定义验证规则
func registerCustomValidators() {
	// 股票代码验证
	validate.RegisterValidation("stock_code", validateStockCode)
	// 日期格式验证
	validate.RegisterValidation("date_format", validateDateFormat)
	// 市场类型验证
	validate.RegisterValidation("market_type", validateMarketType)
	// 报告类型验证
	validate.RegisterValidation("report_type", validateReportType)
	// 任务状态验证
	validate.RegisterValidation("task_status", validateTaskStatus)
	// 数据源验证
	validate.RegisterValidation("data_source", validateDataSource)
}

// validateStockCode 验证股票代码
func validateStockCode(fl validator.FieldLevel) bool {
	return stockCodeRegex.MatchString(fl.Field().String())
}

// validateDateFormat 验证日期格式
func validateDateFormat(fl validator.FieldLevel) bool {
	dateStr := fl.Field().String()
	if !dateRegex.MatchString(dateStr) {
		return false
	}
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}

// validateMarketType 验证市场类型
func validateMarketType(fl validator.FieldLevel) bool {
	market := fl.Field().String()
	return market == "SH" || market == "SZ"
}

// validateReportType 验证报告类型
func validateReportType(fl validator.FieldLevel) bool {
	reportType := fl.Field().String()
	validTypes := []string{"Q1", "Q2", "Q3", "Q4", "A"}
	for _, validType := range validTypes {
		if reportType == validType {
			return true
		}
	}
	return false
}

// validateTaskStatus 验证任务状态
func validateTaskStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := []string{"pending", "running", "success", "failed", "cancelled"}
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}
	return false
}

// validateDataSource 验证数据源
func validateDataSource(fl validator.FieldLevel) bool {
	source := fl.Field().String()
	validSources := []string{"tushare", "sina", "eastmoney", "163", "qq", "manual"}
	for _, validSource := range validSources {
		if source == validSource {
			return true
		}
	}
	return false
}

// ValidateStruct 验证结构体
func ValidateStruct(s interface{}) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var validationErrors []ValidationError
	for _, err := range err.(validator.ValidationErrors) {
		validationErrors = append(validationErrors, ValidationError{
			Field:   err.Field(),
			Tag:     err.Tag(),
			Value:   fmt.Sprintf("%v", err.Value()),
			Message: getErrorMessage(err),
		})
	}

	return errors.New(errors.ErrCodeInvalidParam, formatValidationErrors(validationErrors))
}

// ValidateVar 验证单个变量
func ValidateVar(field interface{}, tag string) error {
	err := validate.Var(field, tag)
	if err == nil {
		return nil
	}

	if validationErr, ok := err.(validator.ValidationErrors); ok {
		for _, err := range validationErr {
			return errors.New(errors.ErrCodeInvalidParam, getErrorMessage(err))
		}
	}

	return errors.New(errors.ErrCodeInvalidParam, err.Error())
}

// BindAndValidate 绑定并验证请求参数
func BindAndValidate(c *gin.Context, obj interface{}) error {
	// 绑定参数
	if err := c.ShouldBindJSON(obj); err != nil {
		return errors.New(errors.ErrCodeInvalidParam, "参数绑定失败: "+err.Error())
	}

	// 验证参数
	return ValidateStruct(obj)
}

// BindQueryAndValidate 绑定并验证查询参数
func BindQueryAndValidate(c *gin.Context, obj interface{}) error {
	// 绑定查询参数
	if err := c.ShouldBindQuery(obj); err != nil {
		return errors.New(errors.ErrCodeInvalidParam, "查询参数绑定失败: "+err.Error())
	}

	// 验证参数
	return ValidateStruct(obj)
}

// getErrorMessage 获取错误消息
func getErrorMessage(err validator.FieldError) string {
	field := err.Field()
	tag := err.Tag()
	value := fmt.Sprintf("%v", err.Value())

	switch tag {
	case "required":
		return fmt.Sprintf("%s是必填字段", field)
	case "min":
		return fmt.Sprintf("%s最小值为%s", field, err.Param())
	case "max":
		return fmt.Sprintf("%s最大值为%s", field, err.Param())
	case "len":
		return fmt.Sprintf("%s长度必须为%s", field, err.Param())
	case "email":
		return fmt.Sprintf("%s必须是有效的邮箱地址", field)
	case "url":
		return fmt.Sprintf("%s必须是有效的URL", field)
	case "stock_code":
		return fmt.Sprintf("%s必须是6位数字的股票代码", field)
	case "date_format":
		return fmt.Sprintf("%s必须是YYYY-MM-DD格式的日期", field)
	case "market_type":
		return fmt.Sprintf("%s必须是SH或SZ", field)
	case "report_type":
		return fmt.Sprintf("%s必须是Q1、Q2、Q3、Q4或A", field)
	case "task_status":
		return fmt.Sprintf("%s必须是有效的任务状态", field)
	case "data_source":
		return fmt.Sprintf("%s必须是有效的数据源", field)
	case "oneof":
		return fmt.Sprintf("%s必须是以下值之一: %s", field, err.Param())
	case "gte":
		return fmt.Sprintf("%s必须大于等于%s", field, err.Param())
	case "lte":
		return fmt.Sprintf("%s必须小于等于%s", field, err.Param())
	case "gt":
		return fmt.Sprintf("%s必须大于%s", field, err.Param())
	case "lt":
		return fmt.Sprintf("%s必须小于%s", field, err.Param())
	default:
		return fmt.Sprintf("%s验证失败，当前值: %s", field, value)
	}
}

// formatValidationErrors 格式化验证错误
func formatValidationErrors(errors []ValidationError) string {
	if len(errors) == 0 {
		return "参数验证失败"
	}

	if len(errors) == 1 {
		return errors[0].Message
	}

	var messages []string
	for _, err := range errors {
		messages = append(messages, err.Message)
	}
	return strings.Join(messages, "; ")
}

// ValidateStockCode 验证股票代码
func ValidateStockCode(code string) error {
	if !stockCodeRegex.MatchString(code) {
		return errors.New(errors.ErrCodeInvalidParam, "股票代码必须是6位数字")
	}
	return nil
}

// ValidateDate 验证日期格式
func ValidateDate(dateStr string) error {
	if !dateRegex.MatchString(dateStr) {
		return errors.New(errors.ErrCodeInvalidParam, "日期格式必须是YYYY-MM-DD")
	}
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return errors.New(errors.ErrCodeInvalidParam, "无效的日期")
	}
	return nil
}

// ValidateDateRange 验证日期范围
func ValidateDateRange(startDate, endDate string) error {
	if err := ValidateDate(startDate); err != nil {
		return err
	}
	if err := ValidateDate(endDate); err != nil {
		return err
	}

	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)

	if start.After(end) {
		return errors.New(errors.ErrCodeInvalidParam, "开始日期不能晚于结束日期")
	}

	return nil
}

// ValidatePagination 验证分页参数
func ValidatePagination(page, pageSize int) error {
	if page < 1 {
		return errors.New(errors.ErrCodeInvalidParam, "页码必须大于0")
	}
	if pageSize < 1 || pageSize > 100 {
		return errors.New(errors.ErrCodeInvalidParam, "页面大小必须在1-100之间")
	}
	return nil
}

// ParseInt 解析整数参数
func ParseInt(value string, defaultValue int) (int, error) {
	if value == "" {
		return defaultValue, nil
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New(errors.ErrCodeInvalidParam, "无效的整数参数")
	}
	return result, nil
}

// ParseFloat 解析浮点数参数
func ParseFloat(value string, defaultValue float64) (float64, error) {
	if value == "" {
		return defaultValue, nil
	}
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, errors.New(errors.ErrCodeInvalidParam, "无效的浮点数参数")
	}
	return result, nil
}