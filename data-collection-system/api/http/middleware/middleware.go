package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, Cache-Control, X-File-Name, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")

		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestID 请求ID中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("RequestID", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// RateLimit 限流中间件
func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现限流逻辑
		c.Next()
	}
}

// Auth 认证中间件
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现认证逻辑
		c.Next()
	}
}

// Timeout 超时中间件
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现超时逻辑
		c.Next()
	}
}

// ContentFilter 新闻内容安全过滤中间件
func ContentFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 在响应前进行内容过滤
		c.Next()
		
		// 检查是否是新闻相关的API
		if isNewsAPI(c.Request.URL.Path) {
			// 获取响应数据
			if data, exists := c.Get("response_data"); exists {
				// 对新闻内容进行安全过滤
				filteredData := filterNewsContent(data)
				c.Set("response_data", filteredData)
			}
		}
	}
}

// isNewsAPI 判断是否是新闻相关的API
func isNewsAPI(path string) bool {
	return path == "/api/v1/news" || 
		   path == "/api/v1/news/hot" || 
		   path == "/api/v1/news/latest" ||
		   (len(path) > 15 && path[:15] == "/api/v1/news/")
}

// filterNewsContent 过滤新闻内容中的敏感信息
func filterNewsContent(data interface{}) interface{} {
	// TODO: 实现具体的内容过滤逻辑
	// 1. 过滤敏感词汇
	// 2. 检查内容合规性
	// 3. 移除可能的恶意链接
	// 4. 标准化内容格式
	return data
}