package baidu

import (
	"context"
	"encoding/json"
	"fmt"
)

// SentimentRequest 情感分析请求
type SentimentRequest struct {
	Text string `json:"text"`
}

// SentimentResponse 情感分析响应
type SentimentResponse struct {
	BaseResponse
	Text  string          `json:"text"`
	Items []SentimentItem `json:"items"`
}

// SentimentItem 情感分析结果项
type SentimentItem struct {
	Sentiment    int     `json:"sentiment"`     // 0:负向，1:中性，2:正向
	Confidence   float64 `json:"confidence"`    // 分类置信度
	PositiveProb float64 `json:"positive_prob"` // 积极概率
	NegativeProb float64 `json:"negative_prob"` // 消极概率
}

// AnalyzeSentiment 情感分析
func (c *Client) AnalyzeSentiment(ctx context.Context, text string) (*SentimentResponse, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	req := SentimentRequest{
		Text: text,
	}

	body, err := c.makeRequest(ctx, "/rpc/2.0/nlp/v1/sentiment_classify", req)
	if err != nil {
		return nil, fmt.Errorf("sentiment analysis request failed: %w", err)
	}

	var resp SentimentResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse sentiment response: %w", err)
	}

	if err := c.checkError(&resp.BaseResponse); err != nil {
		return nil, err
	}

	return &resp, nil
}

// LexerRequest 词法分析请求
type LexerRequest struct {
	Text string `json:"text"`
}

// LexerResponse 词法分析响应
type LexerResponse struct {
	BaseResponse
	Text  string      `json:"text"`
	Items []LexerItem `json:"items"`
}

// LexerItem 词法分析结果项
type LexerItem struct {
	Item string `json:"item"` // 词汇
	NE   string `json:"ne"`   // 命名实体类型
	Pos  string `json:"pos"`  // 词性
}

// AnalyzeLexer 词法分析
func (c *Client) AnalyzeLexer(ctx context.Context, text string) (*LexerResponse, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	req := LexerRequest{
		Text: text,
	}

	body, err := c.makeRequest(ctx, "/rpc/2.0/nlp/v1/lexer", req)
	if err != nil {
		return nil, fmt.Errorf("lexer analysis request failed: %w", err)
	}

	var resp LexerResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse lexer response: %w", err)
	}

	if err := c.checkError(&resp.BaseResponse); err != nil {
		return nil, err
	}

	return &resp, nil
}

// EntityRequest 实体分析请求
type EntityRequest struct {
	Text    string `json:"text"`
	Mention string `json:"mention,omitempty"` // 可选，指定分析的实体
}

// EntityResponse 实体分析响应
type EntityResponse struct {
	BaseResponse
	Text           string       `json:"text"`
	EntityAnalysis []EntityItem `json:"entity_analysis"`
}

// EntityItem 实体分析结果项
type EntityItem struct {
	Mention    string         `json:"mention"`    // 识别出的实体
	Category   EntityCategory `json:"category"`   // 实体概念分析结果
	Confidence float64        `json:"confidence"` // 置信度
	Desc       string         `json:"desc"`       // 实体简介
	Status     string         `json:"status"`     // 关联状态：LINKED/NIL
}

// EntityCategory 实体分类
type EntityCategory struct {
	Level1 string `json:"level_1"` // 一级概念
	Level2 string `json:"level_2"` // 二级概念
	Level3 string `json:"level_3"` // 三级概念
}

// AnalyzeEntity 实体分析
func (c *Client) AnalyzeEntity(ctx context.Context, text string, mention ...string) (*EntityResponse, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	req := EntityRequest{
		Text: text,
	}

	if len(mention) > 0 && mention[0] != "" {
		req.Mention = mention[0]
	}

	body, err := c.makeRequest(ctx, "/rpc/2.0/nlp/v1/entity_analysis", req)
	if err != nil {
		return nil, fmt.Errorf("entity analysis request failed: %w", err)
	}

	var resp EntityResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse entity response: %w", err)
	}

	if err := c.checkError(&resp.BaseResponse); err != nil {
		return nil, err
	}

	return &resp, nil
}

// KeywordRequest 关键词提取请求
type KeywordRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// KeywordResponse 关键词提取响应
type KeywordResponse struct {
	BaseResponse
	Items []KeywordItem `json:"items"`
}

// KeywordItem 关键词项
type KeywordItem struct {
	Tag   string  `json:"tag"`   // 关键词
	Score float64 `json:"score"` // 权重分数
}

// ExtractKeywords 关键词提取
func (c *Client) ExtractKeywords(ctx context.Context, title, content string) (*KeywordResponse, error) {
	if title == "" && content == "" {
		return nil, fmt.Errorf("title and content cannot both be empty")
	}

	req := KeywordRequest{
		Title:   title,
		Content: content,
	}

	body, err := c.makeRequest(ctx, "/rpc/2.0/nlp/v1/keyword", req)
	if err != nil {
		return nil, fmt.Errorf("keyword extraction request failed: %w", err)
	}

	var resp KeywordResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse keyword response: %w", err)
	}

	if err := c.checkError(&resp.BaseResponse); err != nil {
		return nil, err
	}

	return &resp, nil
}

// TextCorrectionRequest 文本纠错请求
type TextCorrectionRequest struct {
	Text string `json:"text"`
}

// TextCorrectionResponse 文本纠错响应
type TextCorrectionResponse struct {
	BaseResponse
	Item CorrectionItem `json:"item"`
}

// CorrectionItem 纠错结果项
type CorrectionItem struct {
	CorrectQuery string             `json:"correct_query"` // 纠错后的文本
	Details      []CorrectionDetail `json:"details"`       // 纠错详情
}

// CorrectionDetail 纠错详情
type CorrectionDetail struct {
	ErrFragment string `json:"err_fragment"` // 错误片段
	CorrectFrag string `json:"correct_frag"` // 正确片段
	OriBegin    int    `json:"ori_begin"`    // 错误开始位置
	OriEnd      int    `json:"ori_end"`      // 错误结束位置
	CorrBegin   int    `json:"corr_begin"`   // 纠正开始位置
	CorrEnd     int    `json:"corr_end"`     // 纠正结束位置
}

// CorrectText 文本纠错
func (c *Client) CorrectText(ctx context.Context, text string) (*TextCorrectionResponse, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	req := TextCorrectionRequest{
		Text: text,
	}

	body, err := c.makeRequest(ctx, "/rpc/2.0/nlp/v1/ecnet", req)
	if err != nil {
		return nil, fmt.Errorf("text correction request failed: %w", err)
	}

	var resp TextCorrectionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse correction response: %w", err)
	}

	if err := c.checkError(&resp.BaseResponse); err != nil {
		return nil, err
	}

	return &resp, nil
}
