package processing

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"data-collection-system/model"
	dao "data-collection-system/repo/mysql"

	"github.com/go-redis/redis/v8"
)

// ImportanceEvaluator 新闻重要程度评估器
type ImportanceEvaluator struct {
	newsRepo    dao.NewsRepository
	redisClient *redis.Client
	// 关键词权重映射
	keywordWeights map[string]float64
	// 来源权重映射
	sourceWeights map[string]float64
	// 正则表达式
	numberRegex *regexp.Regexp
	percentRegex *regexp.Regexp
}

// NewImportanceEvaluator 创建重要程度评估器
func NewImportanceEvaluator(newsRepo dao.NewsRepository, redisClient *redis.Client) *ImportanceEvaluator {
	evaluator := &ImportanceEvaluator{
		newsRepo:       newsRepo,
		redisClient:    redisClient,
		keywordWeights: make(map[string]float64),
		sourceWeights:  make(map[string]float64),
	}

	// 初始化正则表达式
	evaluator.numberRegex = regexp.MustCompile(`\d+(?:\.\d+)?`)
	evaluator.percentRegex = regexp.MustCompile(`\d+(?:\.\d+)?%`)

	// 初始化权重映射
	evaluator.initKeywordWeights()
	evaluator.initSourceWeights()

	return evaluator
}

// initKeywordWeights 初始化关键词权重
func (ie *ImportanceEvaluator) initKeywordWeights() {
	// 高权重关键词（重大事件、政策、财务）
	highWeightKeywords := map[string]float64{
		// 政策相关
		"央行":     1.0,
		"证监会":    1.0,
		"银保监会":   1.0,
		"国务院":    1.0,
		"财政部":    0.9,
		"发改委":    0.9,
		"政策":     0.8,
		"监管":     0.8,
		"法规":     0.7,
		"规定":     0.7,
		
		// 财务相关
		"财报":     0.9,
		"业绩":     0.8,
		"营收":     0.8,
		"利润":     0.8,
		"亏损":     0.8,
		"分红":     0.7,
		"股息":     0.7,
		"重组":     0.9,
		"并购":     0.9,
		"收购":     0.8,
		"IPO":    0.9,
		"上市":     0.8,
		"退市":     0.9,
		"停牌":     0.8,
		"复牌":     0.7,
		
		// 市场相关
		"涨停":     0.7,
		"跌停":     0.7,
		"暴涨":     0.6,
		"暴跌":     0.6,
		"大涨":     0.5,
		"大跌":     0.5,
		"突破":     0.5,
		"创新高":    0.6,
		"创新低":    0.6,
		
		// 风险相关
		"风险":     0.7,
		"警告":     0.8,
		"违规":     0.8,
		"处罚":     0.8,
		"调查":     0.7,
		"诉讼":     0.6,
		"债务":     0.6,
		"违约":     0.8,
		"破产":     0.9,
		"清算":     0.8,
		
		// 行业相关
		"科技":     0.5,
		"创新":     0.5,
		"研发":     0.5,
		"专利":     0.4,
		"技术":     0.4,
		"数字化":    0.4,
		"智能":     0.4,
		"新能源":    0.6,
		"环保":     0.5,
		"碳中和":    0.6,
		"碳达峰":    0.6,
	}

	// 中等权重关键词
	midWeightKeywords := map[string]float64{
		"合作":   0.3,
		"协议":   0.3,
		"签约":   0.3,
		"投资":   0.4,
		"融资":   0.4,
		"贷款":   0.3,
		"项目":   0.3,
		"建设":   0.3,
		"扩产":   0.4,
		"产能":   0.3,
		"销售":   0.3,
		"市场":   0.3,
		"客户":   0.2,
		"订单":   0.3,
		"合同":   0.3,
		"产品":   0.2,
		"服务":   0.2,
		"业务":   0.2,
		"战略":   0.3,
		"规划":   0.3,
		"发展":   0.2,
		"增长":   0.3,
		"下降":   0.3,
		"变化":   0.2,
		"调整":   0.3,
		"优化":   0.2,
		"改革":   0.4,
		"转型":   0.4,
		"升级":   0.3,
	}

	// 合并权重映射
	for keyword, weight := range highWeightKeywords {
		ie.keywordWeights[keyword] = weight
	}
	for keyword, weight := range midWeightKeywords {
		ie.keywordWeights[keyword] = weight
	}
}

// initSourceWeights 初始化来源权重
func (ie *ImportanceEvaluator) initSourceWeights() {
	ie.sourceWeights = map[string]float64{
		// 官方媒体 - 最高权重
		"新华社":      1.0,
		"人民日报":     1.0,
		"央视":       1.0,
		"中国证券报":    0.9,
		"上海证券报":    0.9,
		"证券时报":     0.9,
		"证券日报":     0.9,
		"金融时报":     0.9,
		"经济日报":     0.8,
		"第一财经":     0.8,
		
		// 专业财经媒体 - 高权重
		"财新":       0.8,
		"财联社":      0.8,
		"东方财富":     0.7,
		"同花顺":      0.7,
		"和讯":       0.6,
		"金融界":      0.6,
		"中金在线":     0.6,
		"雪球":       0.5,
		"格隆汇":      0.6,
		"智通财经":     0.6,
		
		// 综合门户 - 中等权重
		"新浪":       0.5,
		"搜狐":       0.5,
		"网易":       0.5,
		"腾讯":       0.5,
		"凤凰":       0.5,
		"今日头条":     0.4,
		
		// 其他来源 - 默认权重
		"其他":       0.3,
	}
}

// EvaluateImportance 评估新闻重要程度
func (ie *ImportanceEvaluator) EvaluateImportance(ctx context.Context, news *model.NewsData) (int8, error) {
	// 计算各个维度的得分
	keywordScore := ie.calculateKeywordScore(news)
	sourceScore := ie.calculateSourceScore(news)
	timeScore := ie.calculateTimeScore(news)
	stockScore := ie.calculateStockScore(news)
	sentimentScore := ie.calculateSentimentScore(news)
	numberScore := ie.calculateNumberScore(news)
	lengthScore := ie.calculateLengthScore(news)

	// 加权计算总分
	totalScore := keywordScore*0.3 + sourceScore*0.2 + timeScore*0.1 + 
				 stockScore*0.15 + sentimentScore*0.1 + numberScore*0.1 + lengthScore*0.05

	// 转换为1-5级别
	importanceLevel := ie.scoreToLevel(totalScore)

	return importanceLevel, nil
}

// calculateKeywordScore 计算关键词得分
func (ie *ImportanceEvaluator) calculateKeywordScore(news *model.NewsData) float64 {
	text := strings.ToLower(news.Title + " " + news.Content)
	totalScore := 0.0
	matchCount := 0

	for keyword, weight := range ie.keywordWeights {
		if strings.Contains(text, keyword) {
			totalScore += weight
			matchCount++
		}
	}

	// 标题中的关键词给予额外权重
	titleText := strings.ToLower(news.Title)
	for keyword, weight := range ie.keywordWeights {
		if strings.Contains(titleText, keyword) {
			totalScore += weight * 0.5 // 标题额外加权50%
		}
	}

	// 归一化得分
	if matchCount > 0 {
		return math.Min(totalScore/float64(matchCount), 1.0)
	}
	return 0.0
}

// calculateSourceScore 计算来源得分
func (ie *ImportanceEvaluator) calculateSourceScore(news *model.NewsData) float64 {
	if news.Source == "" {
		return 0.3 // 默认得分
	}

	// 检查来源权重
	for source, weight := range ie.sourceWeights {
		if strings.Contains(news.Source, source) {
			return weight
		}
	}

	return 0.3 // 未知来源默认得分
}

// calculateTimeScore 计算时效性得分
func (ie *ImportanceEvaluator) calculateTimeScore(news *model.NewsData) float64 {
	now := time.Now()
	timeDiff := now.Sub(news.PublishTime)

	// 时效性衰减函数
	if timeDiff <= time.Hour {
		return 1.0 // 1小时内最高分
	} else if timeDiff <= 6*time.Hour {
		return 0.8 // 6小时内高分
	} else if timeDiff <= 24*time.Hour {
		return 0.6 // 24小时内中等分
	} else if timeDiff <= 3*24*time.Hour {
		return 0.4 // 3天内低分
	} else {
		return 0.2 // 超过3天很低分
	}
}

// calculateStockScore 计算股票关联得分
func (ie *ImportanceEvaluator) calculateStockScore(news *model.NewsData) float64 {
	if !news.HasRelatedStocks() {
		return 0.0
	}

	stockCount := len(news.RelatedStocks)
	if stockCount == 1 {
		return 0.8 // 单一股票高度相关
	} else if stockCount <= 3 {
		return 0.6 // 少数股票相关
	} else if stockCount <= 5 {
		return 0.4 // 多个股票相关
	} else {
		return 0.2 // 过多股票可能相关性不强
	}
}

// calculateSentimentScore 计算情感得分
func (ie *ImportanceEvaluator) calculateSentimentScore(news *model.NewsData) float64 {
	if news.Sentiment == nil {
		return 0.5 // 中性默认得分
	}

	switch *news.Sentiment {
	case model.SentimentPositive:
		return 0.7 // 正面情感
	case model.SentimentNegative:
		return 0.8 // 负面情感通常更重要
	case model.SentimentNeutral:
		return 0.5 // 中性情感
	default:
		return 0.5
	}
}

// calculateNumberScore 计算数字得分（包含具体数据的新闻通常更重要）
func (ie *ImportanceEvaluator) calculateNumberScore(news *model.NewsData) float64 {
	text := news.Title + " " + news.Content
	
	// 统计数字和百分比
	numbers := ie.numberRegex.FindAllString(text, -1)
	percentages := ie.percentRegex.FindAllString(text, -1)
	
	score := 0.0
	
	// 数字越多，重要性可能越高
	numberCount := len(numbers)
	if numberCount > 0 {
		score += math.Min(float64(numberCount)*0.1, 0.5)
	}
	
	// 百分比通常表示重要的变化
	percentageCount := len(percentages)
	if percentageCount > 0 {
		score += math.Min(float64(percentageCount)*0.2, 0.6)
	}
	
	return math.Min(score, 1.0)
}

// calculateLengthScore 计算长度得分
func (ie *ImportanceEvaluator) calculateLengthScore(news *model.NewsData) float64 {
	titleLength := len([]rune(news.Title))
	contentLength := len([]rune(news.Content))
	
	// 标题长度评分
	titleScore := 0.0
	if titleLength >= 10 && titleLength <= 50 {
		titleScore = 1.0 // 适中的标题长度
	} else if titleLength > 50 {
		titleScore = 0.8 // 较长标题
	} else {
		titleScore = 0.6 // 较短标题
	}
	
	// 内容长度评分
	contentScore := 0.0
	if contentLength >= 200 && contentLength <= 2000 {
		contentScore = 1.0 // 适中的内容长度
	} else if contentLength > 2000 {
		contentScore = 0.8 // 较长内容
	} else if contentLength >= 100 {
		contentScore = 0.6 // 较短内容
	} else {
		contentScore = 0.3 // 很短内容
	}
	
	return (titleScore + contentScore) / 2.0
}

// scoreToLevel 将得分转换为重要程度级别
func (ie *ImportanceEvaluator) scoreToLevel(score float64) int8 {
	if score >= 0.8 {
		return model.ImportanceLevelVeryHigh // 5级 - 很高
	} else if score >= 0.6 {
		return model.ImportanceLevelHigh // 4级 - 高
	} else if score >= 0.4 {
		return model.ImportanceLevelMedium // 3级 - 中等
	} else if score >= 0.2 {
		return model.ImportanceLevelLow // 2级 - 低
	} else {
		return model.ImportanceLevelVeryLow // 1级 - 很低
	}
}

// BatchEvaluateImportance 批量评估重要程度
func (ie *ImportanceEvaluator) BatchEvaluateImportance(ctx context.Context, newsList []*model.NewsData) error {
	for _, news := range newsList {
		importanceLevel, err := ie.EvaluateImportance(ctx, news)
		if err != nil {
			return fmt.Errorf("failed to evaluate importance for news %d: %w", news.ID, err)
		}
		news.ImportanceLevel = importanceLevel
	}
	return nil
}

// GetImportanceStats 获取重要程度统计信息
func (ie *ImportanceEvaluator) GetImportanceStats(ctx context.Context) (*dao.ImportanceStats, error) {
	// 直接使用NewsRepository的GetImportanceStats方法
	return ie.newsRepo.GetImportanceStats(ctx)
}

// UpdateKeywordWeight 更新关键词权重
func (ie *ImportanceEvaluator) UpdateKeywordWeight(keyword string, weight float64) {
	ie.keywordWeights[keyword] = weight
}

// UpdateSourceWeight 更新来源权重
func (ie *ImportanceEvaluator) UpdateSourceWeight(source string, weight float64) {
	ie.sourceWeights[source] = weight
}

// ImportanceStats 重要程度统计信息