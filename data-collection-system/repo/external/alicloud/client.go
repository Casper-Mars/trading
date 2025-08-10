package alicloud

import (
	"encoding/json"
	"fmt"
	"time"

	"data-collection-system/pkg/config"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/nlp-automl"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
)

// Client 阿里云NLP客户端
type Client struct {
	client *nlp_automl.Client
	config *config.AliCloudNLPConfig
}

// NewClient 创建阿里云NLP客户端
func NewClient(cfg *config.AliCloudNLPConfig) *Client {
	// 创建认证配置
	credential := credentials.NewAccessKeyCredential(
		cfg.AccessKeyID,
		cfg.AccessKeySecret,
	)

	// 创建SDK配置
	sdkConfig := sdk.NewConfig()
	sdkConfig.WithTimeout(time.Duration(cfg.Timeout) * time.Second)

	// 创建NLP客户端
	client, err := nlp_automl.NewClientWithOptions(
		cfg.RegionID,
		sdkConfig,
		credential,
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to create AliCloud NLP client: %v", err))
	}

	return &Client{
		client: client,
		config: cfg,
	}
}

// SentimentAnalysisRequest 情感分析请求
type SentimentAnalysisRequest struct {
	Content string `json:"content"`
	Lang    string `json:"lang,omitempty"`
}

// SentimentAnalysisResponse 情感分析响应
type SentimentAnalysisResponse struct {
	RequestID string `json:"RequestId"`
	Data      struct {
		Sentiment  string  `json:"sentiment"`
		Confidence float64 `json:"confidence"`
	} `json:"Data"`
}

// AnalyzeSentiment 情感分析
func (c *Client) AnalyzeSentiment(content string) (*SentimentAnalysisResponse, error) {
	request := nlp_automl.CreateGetPredictResultRequest()
	request.Scheme = "https"
	
	// 构建请求内容
	requestContent := SentimentAnalysisRequest{
		Content: content,
		Lang:    "zh",
	}
	
	requestData, err := json.Marshal(requestContent)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}
	
	request.Content = string(requestData)
	request.ModelId = "sentiment_analysis" // 情感分析模型ID
	
	response, err := c.client.GetPredictResult(request)
	if err != nil {
		return nil, fmt.Errorf("sentiment analysis failed: %w", err)
	}
	
	var result SentimentAnalysisResponse
	if err := json.Unmarshal([]byte(response.GetHttpContentString()), &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}
	
	return &result, nil
}

// EntityExtractionRequest 实体抽取请求
type EntityExtractionRequest struct {
	Content string `json:"content"`
	Lang    string `json:"lang,omitempty"`
}

// EntityExtractionResponse 实体抽取响应
type EntityExtractionResponse struct {
	RequestID string `json:"RequestId"`
	Data      struct {
		Entities []struct {
			Type  string `json:"type"`
			Value string `json:"value"`
			Start int    `json:"start"`
			End   int    `json:"end"`
		} `json:"entities"`
	} `json:"Data"`
}

// ExtractEntities 实体抽取
func (c *Client) ExtractEntities(content string) (*EntityExtractionResponse, error) {
	request := nlp_automl.CreateGetPredictResultRequest()
	request.Scheme = "https"
	
	// 构建请求内容
	requestContent := EntityExtractionRequest{
		Content: content,
		Lang:    "zh",
	}
	
	requestData, err := json.Marshal(requestContent)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}
	
	request.Content = string(requestData)
	request.ModelId = "entity_extraction" // 实体抽取模型ID
	
	response, err := c.client.GetPredictResult(request)
	if err != nil {
		return nil, fmt.Errorf("entity extraction failed: %w", err)
	}
	
	var result EntityExtractionResponse
	if err := json.Unmarshal([]byte(response.GetHttpContentString()), &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}
	
	return &result, nil
}

// KeywordExtractionRequest 关键词抽取请求
type KeywordExtractionRequest struct {
	Content string `json:"content"`
	Lang    string `json:"lang,omitempty"`
	TopK    int    `json:"topK,omitempty"`
}

// KeywordExtractionResponse 关键词抽取响应
type KeywordExtractionResponse struct {
	RequestID string `json:"RequestId"`
	Data      struct {
		Keywords []struct {
			Word  string  `json:"word"`
			Score float64 `json:"score"`
		} `json:"keywords"`
	} `json:"Data"`
}

// ExtractKeywords 关键词抽取
func (c *Client) ExtractKeywords(content string, topK int) (*KeywordExtractionResponse, error) {
	request := nlp_automl.CreateGetPredictResultRequest()
	request.Scheme = "https"
	
	// 构建请求内容
	requestContent := KeywordExtractionRequest{
		Content: content,
		Lang:    "zh",
		TopK:    topK,
	}
	
	requestData, err := json.Marshal(requestContent)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}
	
	request.Content = string(requestData)
	request.ModelId = "keyword_extraction" // 关键词抽取模型ID
	
	response, err := c.client.GetPredictResult(request)
	if err != nil {
		return nil, fmt.Errorf("keyword extraction failed: %w", err)
	}
	
	var result KeywordExtractionResponse
	if err := json.Unmarshal([]byte(response.GetHttpContentString()), &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}
	
	return &result, nil
}

// TextClassificationRequest 文本分类请求
type TextClassificationRequest struct {
	Content string `json:"content"`
	Lang    string `json:"lang,omitempty"`
}

// TextClassificationResponse 文本分类响应
type TextClassificationResponse struct {
	RequestID string `json:"RequestId"`
	Data      struct {
		Categories []struct {
			Category string  `json:"category"`
			Score    float64 `json:"score"`
		} `json:"categories"`
	} `json:"Data"`
}

// ClassifyText 文本分类
func (c *Client) ClassifyText(content string) (*TextClassificationResponse, error) {
	request := nlp_automl.CreateGetPredictResultRequest()
	request.Scheme = "https"
	
	// 构建请求内容
	requestContent := TextClassificationRequest{
		Content: content,
		Lang:    "zh",
	}
	
	requestData, err := json.Marshal(requestContent)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}
	
	request.Content = string(requestData)
	request.ModelId = "text_classification" // 文本分类模型ID
	
	response, err := c.client.GetPredictResult(request)
	if err != nil {
		return nil, fmt.Errorf("text classification failed: %w", err)
	}
	
	var result TextClassificationResponse
	if err := json.Unmarshal([]byte(response.GetHttpContentString()), &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}
	
	return &result, nil
}