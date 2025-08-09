package baidu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"data-collection-system/pkg/config"
	"data-collection-system/pkg/logger"
)

// Client 百度AI客户端
type Client struct {
	config      *config.BaiduAIConfig
	httpClient  *http.Client
	accessToken string
	tokenExpiry time.Time
	tokenMutex  sync.RWMutex
}

// NewClient 创建百度AI客户端
func NewClient(cfg *config.BaiduAIConfig) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
	}
}

// AccessTokenResponse 访问令牌响应
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

// getAccessToken 获取访问令牌
func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMutex.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		token := c.accessToken
		c.tokenMutex.RUnlock()
		return token, nil
	}
	c.tokenMutex.RUnlock()

	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	// 双重检查
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	// 构建请求URL
	tokenURL := fmt.Sprintf("%s/oauth/2.0/token", c.config.BaseURL)
	params := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.config.APIKey},
		"client_secret": {c.config.SecretKey},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	var tokenResp AccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("token request failed: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token received")
	}

	// 缓存令牌，提前5分钟过期
	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-300) * time.Second)

	logger.Info("Successfully obtained Baidu AI access token")
	return c.accessToken, nil
}

// makeRequest 发起API请求
func (c *Client) makeRequest(ctx context.Context, endpoint string, data interface{}) ([]byte, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// 序列化请求数据
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	// 构建请求URL
	requestURL := fmt.Sprintf("%s%s?access_token=%s", c.config.BaseURL, endpoint, token)

	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// BaseResponse 基础响应结构
type BaseResponse struct {
	LogID    uint64 `json:"log_id"`
	Error    int    `json:"error_code,omitempty"`
	ErrorMsg string `json:"error_msg,omitempty"`
}

// checkError 检查响应错误
func (c *Client) checkError(resp *BaseResponse) error {
	if resp.Error != 0 {
		return fmt.Errorf("API error %d: %s", resp.Error, resp.ErrorMsg)
	}
	return nil
}
