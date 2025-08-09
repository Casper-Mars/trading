package processing

import (
	"context"
	"crypto/md5"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"data-collection-system/model"
	dao "data-collection-system/repo/mysql"

	"github.com/go-redis/redis/v8"
)

// NewsDeduplicator 新闻去重器
type NewsDeduplicator struct {
	newsRepo    dao.NewsRepository
	redisClient *redis.Client
	// 文本预处理正则表达式
	cleanRegex *regexp.Regexp
	// 停用词列表
	stopWords map[string]bool
}

// NewNewsDeduplicator 创建新闻去重器
func NewNewsDeduplicator(newsRepo dao.NewsRepository, redisClient *redis.Client) *NewsDeduplicator {
	deduplicator := &NewsDeduplicator{
		newsRepo:    newsRepo,
		redisClient: redisClient,
		stopWords:   make(map[string]bool),
	}

	// 初始化文本清理正则表达式
	deduplicator.cleanRegex = regexp.MustCompile(`[^\p{Han}\p{L}\p{N}\s]`)

	// 初始化停用词
	deduplicator.initStopWords()

	return deduplicator
}

// initStopWords 初始化停用词列表
func (nd *NewsDeduplicator) initStopWords() {
	stopWords := []string{
		"的", "了", "在", "是", "我", "有", "和", "就", "不", "人", "都", "一", "一个", "上", "也", "很", "到", "说", "要", "去", "你", "会", "着", "没有", "看", "好", "自己", "这",
		"那", "里", "后", "以", "时", "来", "用", "们", "生", "大", "为", "能", "作", "分", "成", "者", "多", "部", "性", "通", "被", "从", "起", "同", "由", "其", "当", "两", "些", "如", "应", "等",
		"而", "可", "下", "又", "前", "头", "关", "还", "因", "只", "可以", "这个", "我们", "他们", "什么", "知道", "现在", "这样", "因为", "所以", "但是", "如果", "已经", "还是", "比较", "非常", "怎么", "为了", "虽然", "或者", "不过", "然后", "而且", "不是", "就是", "这种", "那种", "这些", "那些", "这里", "那里", "出来", "起来", "进行", "通过", "得到", "成为", "作为", "由于", "根据", "按照", "关于", "对于", "除了", "另外", "此外", "然而", "因此", "所以", "总之", "首先", "其次", "最后", "同时", "另一方面",
	}

	for _, word := range stopWords {
		nd.stopWords[word] = true
	}
}

// DetectDuplicates 检测重复新闻
func (nd *NewsDeduplicator) DetectDuplicates(ctx context.Context, news *model.NewsData) ([]*model.NewsData, error) {
	duplicates := make([]*model.NewsData, 0)

	// 1. 基于URL的精确去重
	urlDuplicates, err := nd.findDuplicatesByURL(ctx, news)
	if err != nil {
		return nil, fmt.Errorf("failed to find URL duplicates: %w", err)
	}
	duplicates = append(duplicates, urlDuplicates...)

	// 2. 基于标题的相似度去重
	titleDuplicates, err := nd.findDuplicatesByTitle(ctx, news)
	if err != nil {
		return nil, fmt.Errorf("failed to find title duplicates: %w", err)
	}
	duplicates = append(duplicates, titleDuplicates...)

	// 3. 基于内容的相似度去重
	contentDuplicates, err := nd.findDuplicatesByContent(ctx, news)
	if err != nil {
		return nil, fmt.Errorf("failed to find content duplicates: %w", err)
	}
	duplicates = append(duplicates, contentDuplicates...)

	// 去重结果
	deduplicatedResults := nd.deduplicateResults(duplicates)

	return deduplicatedResults, nil
}

// findDuplicatesByURL 基于URL查找重复新闻
func (nd *NewsDeduplicator) findDuplicatesByURL(ctx context.Context, news *model.NewsData) ([]*model.NewsData, error) {
	// 通过URL查找重复新闻
	existing, err := nd.newsRepo.GetByURL(ctx, news.URL)
	if err != nil {
		return nil, err
	}
	
	// 如果找到相同URL的新闻且不是同一条新闻，则认为是重复
	if existing != nil && existing.ID != news.ID {
		return []*model.NewsData{existing}, nil
	}
	
	return []*model.NewsData{}, nil
}

// findDuplicatesByTitle 基于标题相似度查找重复新闻
func (nd *NewsDeduplicator) findDuplicatesByTitle(ctx context.Context, news *model.NewsData) ([]*model.NewsData, error) {
	// 查找相似标题的新闻
	titleWords := strings.Fields(news.Title)
	if len(titleWords) < 3 {
		return []*model.NewsData{}, nil // 标题太短，跳过
	}

	// 使用主要关键词进行搜索
	keyword := titleWords[0]
	if len(titleWords) > 1 {
		keyword = titleWords[1] // 通常第二个词更有意义
	}

	// 使用搜索接口查找相关新闻
	searchParams := &dao.NewsSearchParams{
		Keyword: keyword,
		Limit:   100, // 限制搜索结果数量
		Offset:  0,
	}
	candidates, _, err := nd.newsRepo.Search(ctx, searchParams)
	if err != nil {
		return nil, err
	}

	duplicates := make([]*model.NewsData, 0)
	for _, candidate := range candidates {
		similarity := nd.calculateTextSimilarity(news.Title, candidate.Title)
		if similarity > 0.8 { // 标题相似度阈值
			duplicates = append(duplicates, candidate)
		}
	}

	return duplicates, nil
}

// findDuplicatesByContent 基于内容相似度查找重复新闻
func (nd *NewsDeduplicator) findDuplicatesByContent(ctx context.Context, news *model.NewsData) ([]*model.NewsData, error) {
	// 查找同一时间段内的新闻（前后12小时）
	startTime := news.PublishTime.Add(-12 * time.Hour)
	endTime := news.PublishTime.Add(12 * time.Hour)

	// 使用时间范围查询
	candidates, err := nd.newsRepo.GetByTimeRange(ctx, startTime, endTime, 1000, 0)
	if err != nil {
		return nil, err
	}

	// 过滤掉当前新闻本身
	filteredCandidates := make([]*model.NewsData, 0)
	for _, candidate := range candidates {
		if candidate.ID != news.ID {
			filteredCandidates = append(filteredCandidates, candidate)
		}
	}
	candidates = filteredCandidates

	duplicates := make([]*model.NewsData, 0)
	for _, candidate := range candidates {
		// 计算内容相似度
		similarity := nd.calculateContentSimilarity(news.Content, candidate.Content)
		if similarity > 0.7 { // 内容相似度阈值
			duplicates = append(duplicates, candidate)
		}
	}

	return duplicates, nil
}

// calculateTextSimilarity 计算文本相似度（基于Jaccard相似度）
func (nd *NewsDeduplicator) calculateTextSimilarity(text1, text2 string) float64 {
	// 文本预处理
	words1 := nd.preprocessText(text1)
	words2 := nd.preprocessText(text2)

	// 转换为集合
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, word := range words1 {
		set1[word] = true
	}
	for _, word := range words2 {
		set2[word] = true
	}

	// 计算交集和并集
	intersection := 0
	union := make(map[string]bool)

	for word := range set1 {
		union[word] = true
		if set2[word] {
			intersection++
		}
	}
	for word := range set2 {
		union[word] = true
	}

	// Jaccard相似度 = |交集| / |并集|
	if len(union) == 0 {
		return 0.0
	}
	return float64(intersection) / float64(len(union))
}

// calculateContentSimilarity 计算内容相似度（基于余弦相似度）
func (nd *NewsDeduplicator) calculateContentSimilarity(content1, content2 string) float64 {
	// 文本预处理
	words1 := nd.preprocessText(content1)
	words2 := nd.preprocessText(content2)

	// 构建词频向量
	vocab := make(map[string]int)
	index := 0
	for _, word := range words1 {
		if _, exists := vocab[word]; !exists {
			vocab[word] = index
			index++
		}
	}
	for _, word := range words2 {
		if _, exists := vocab[word]; !exists {
			vocab[word] = index
			index++
		}
	}

	// 创建词频向量
	vector1 := make([]float64, len(vocab))
	vector2 := make([]float64, len(vocab))

	for _, word := range words1 {
		if idx, exists := vocab[word]; exists {
			vector1[idx]++
		}
	}
	for _, word := range words2 {
		if idx, exists := vocab[word]; exists {
			vector2[idx]++
		}
	}

	// 计算余弦相似度
	return nd.cosineSimilarity(vector1, vector2)
}

// cosineSimilarity 计算余弦相似度
func (nd *NewsDeduplicator) cosineSimilarity(vector1, vector2 []float64) float64 {
	if len(vector1) != len(vector2) {
		return 0.0
	}

	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	for i := 0; i < len(vector1); i++ {
		dotProduct += vector1[i] * vector2[i]
		norm1 += vector1[i] * vector1[i]
		norm2 += vector2[i] * vector2[i]
	}

	if norm1 == 0.0 || norm2 == 0.0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// preprocessText 文本预处理
func (nd *NewsDeduplicator) preprocessText(text string) []string {
	// 转换为小写
	text = strings.ToLower(text)

	// 移除标点符号和特殊字符
	text = nd.cleanRegex.ReplaceAllString(text, " ")

	// 分词
	words := strings.Fields(text)

	// 移除停用词和短词
	filtered := make([]string, 0)
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(word) > 1 && !nd.stopWords[word] {
			filtered = append(filtered, word)
		}
	}

	return filtered
}

// deduplicateResults 去重结果列表
func (nd *NewsDeduplicator) deduplicateResults(duplicates []*model.NewsData) []*model.NewsData {
	seen := make(map[uint64]bool)
	result := make([]*model.NewsData, 0)

	for _, news := range duplicates {
		if !seen[news.ID] {
			seen[news.ID] = true
			result = append(result, news)
		}
	}

	return result
}

// GenerateContentHash 生成内容哈希
func (nd *NewsDeduplicator) GenerateContentHash(news *model.NewsData) string {
	// 使用标题和内容的关键部分生成哈希
	content := nd.normalizeForHash(news.Title + " " + news.Content)
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// normalizeForHash 为哈希生成标准化文本
func (nd *NewsDeduplicator) normalizeForHash(text string) string {
	// 预处理文本
	words := nd.preprocessText(text)

	// 排序以确保一致性
	sort.Strings(words)

	// 只保留前100个最重要的词（避免哈希过长）
	if len(words) > 100 {
		words = words[:100]
	}

	return strings.Join(words, " ")
}

// MarkAsDuplicate 标记为重复新闻
func (nd *NewsDeduplicator) MarkAsDuplicate(ctx context.Context, newsID uint64, originalID uint64) error {
	// 在Redis中记录重复关系
	if nd.redisClient != nil {
		key := fmt.Sprintf("news:duplicate:%d", newsID)
		err := nd.redisClient.Set(ctx, key, originalID, 7*24*time.Hour).Err()
		if err != nil {
			return fmt.Errorf("failed to mark duplicate in Redis: %w", err)
		}
	}

	// 可以选择在数据库中添加重复标记字段
	// 这里暂时只使用Redis记录

	return nil
}

// IsDuplicate 检查是否为重复新闻
func (nd *NewsDeduplicator) IsDuplicate(ctx context.Context, newsID uint64) (bool, uint64, error) {
	if nd.redisClient == nil {
		return false, 0, nil
	}

	key := fmt.Sprintf("news:duplicate:%d", newsID)
	originalID, err := nd.redisClient.Get(ctx, key).Uint64()
	if err == redis.Nil {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, fmt.Errorf("failed to check duplicate in Redis: %w", err)
	}

	return true, originalID, nil
}

// GetDuplicationStats 获取去重统计信息
func (nd *NewsDeduplicator) GetDuplicationStats(ctx context.Context) (*dao.DuplicationStats, error) {
	// 直接使用NewsRepository的GetDuplicationStats方法
	return nd.newsRepo.GetDuplicationStats(ctx)
}