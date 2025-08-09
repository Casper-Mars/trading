package tushare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Client Tushare API客户端
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	limiter    *rate.Limiter
	mu         sync.RWMutex
}

// Config 客户端配置
type Config struct {
	Token       string        `json:"token"`
	BaseURL     string        `json:"base_url"`
	Timeout     time.Duration `json:"timeout"`
	RetryCount  int           `json:"retry_count"`
	RateLimit   int           `json:"rate_limit"` // 每分钟请求数限制
}

// NewClient 创建新的Tushare客户端
func NewClient(config *Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.tushare.pro"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RateLimit == 0 {
		config.RateLimit = 200 // 默认每分钟200次请求
	}

	// 创建限流器，每分钟允许的请求数
	limiter := rate.NewLimiter(rate.Limit(float64(config.RateLimit)/60), config.RateLimit)

	return &Client{
		baseURL: config.BaseURL,
		token:   config.Token,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		limiter: limiter,
	}
}

// Request Tushare API请求结构
type Request struct {
	APIName string                 `json:"api_name"`
	Token   string                 `json:"token"`
	Params  map[string]interface{} `json:"params,omitempty"`
	Fields  string                 `json:"fields,omitempty"`
}

// Response Tushare API响应结构
type Response struct {
	RequestID string          `json:"request_id"`
	Code      int             `json:"code"`
	Msg       string          `json:"msg"`
	Data      *ResponseData   `json:"data"`
}

// ResponseData 响应数据结构
type ResponseData struct {
	Fields []string        `json:"fields"`
	Items  [][]interface{} `json:"items"`
}

// Call 调用Tushare API
func (c *Client) Call(ctx context.Context, apiName string, params map[string]interface{}, fields string) (*ResponseData, error) {
	// 限流控制
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	req := &Request{
		APIName: apiName,
		Token:   c.token,
		Params:  params,
		Fields:  fields,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create http request failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "data-collection-system/1.0")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body failed: %w", err)
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	if response.Code != 0 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Msg)
	}

	return response.Data, nil
}

// CallWithRetry 带重试的API调用
func (c *Client) CallWithRetry(ctx context.Context, apiName string, params map[string]interface{}, fields string, maxRetries int) (*ResponseData, error) {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		data, err := c.Call(ctx, apiName, params, fields)
		if err == nil {
			return data, nil
		}
		lastErr = err

		// 如果不是最后一次重试，等待一段时间后重试
		if i < maxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(i+1) * time.Second):
				// 指数退避
			}
		}
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// GetToken 获取当前token
func (c *Client) GetToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// SetToken 设置新的token
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// IsHealthy 检查客户端健康状态
func (c *Client) IsHealthy(ctx context.Context) error {
	// 调用一个简单的API来检查连接状态
	_, err := c.Call(ctx, "stock_basic", map[string]interface{}{
		"list_status": "L",
		"limit":       1,
	}, "ts_code,symbol,name")
	return err
}