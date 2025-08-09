package processing

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"data-collection-system/model"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// StockMatcher 股票匹配器
type StockMatcher struct {
	db          *gorm.DB
	redisClient *redis.Client
	// 缓存股票信息
	stockCache map[string]*model.Stock // symbol -> stock
	nameCache  map[string]*model.Stock // name -> stock
	// 正则表达式
	stockCodeRegex *regexp.Regexp
	// 公司名称变体映射
	nameVariants map[string][]string
}

// NewStockMatcher 创建股票匹配器
func NewStockMatcher(db *gorm.DB, redisClient *redis.Client) *StockMatcher {
	matcher := &StockMatcher{
		db:          db,
		redisClient: redisClient,
		stockCache:  make(map[string]*model.Stock),
		nameCache:   make(map[string]*model.Stock),
		nameVariants: make(map[string][]string),
	}

	// 初始化股票代码正则表达式
	// A股：6位数字（000001-999999）
	// 港股：4-5位数字
	// 美股：字母组合
	matcher.stockCodeRegex = regexp.MustCompile(`\b([0-9]{6}|[0-9]{4,5}|[A-Z]{1,5})\b`)

	// 初始化缓存
	matcher.initializeCache()

	return matcher
}

// initializeCache 初始化股票缓存
func (sm *StockMatcher) initializeCache() {
	ctx := context.Background()

	// 从数据库加载所有活跃股票
	var stocks []*model.Stock
	err := sm.db.Where("status = ?", model.StockStatusActive).Find(&stocks).Error
	if err != nil {
		fmt.Printf("Failed to load stocks for cache: %v\n", err)
		return
	}

	// 构建缓存
	for _, stock := range stocks {
		sm.stockCache[stock.Symbol] = stock
		sm.nameCache[stock.Name] = stock
		
		// 生成公司名称变体
		variants := sm.generateNameVariants(stock.Name)
		for _, variant := range variants {
			sm.nameCache[variant] = stock
		}
		sm.nameVariants[stock.Symbol] = variants
	}

	// 缓存到Redis（可选，用于分布式环境）
	sm.cacheToRedis(ctx)
}

// generateNameVariants 生成公司名称变体
func (sm *StockMatcher) generateNameVariants(companyName string) []string {
	variants := make([]string, 0)
	
	// 移除常见后缀
	suffixes := []string{"股份有限公司", "有限公司", "集团", "控股", "科技", "实业", "投资", "发展", "建设"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(companyName, suffix) {
			variant := strings.TrimSuffix(companyName, suffix)
			if len(variant) > 0 {
				variants = append(variants, variant)
			}
		}
	}
	
	// 提取括号内的简称
	bracketRegex := regexp.MustCompile(`（([^）]+)）|\(([^)]+)\)`)
	matches := bracketRegex.FindAllStringSubmatch(companyName, -1)
	for _, match := range matches {
		for i := 1; i < len(match); i++ {
			if match[i] != "" {
				variants = append(variants, match[i])
			}
		}
	}
	
	// 生成首字母缩写（对于长公司名）
	if len(companyName) > 6 {
		words := strings.Fields(companyName)
		if len(words) > 2 {
			var acronym strings.Builder
			for _, word := range words {
				if len(word) > 0 {
					acronym.WriteString(string([]rune(word)[0]))
				}
			}
			if acronym.Len() > 0 {
				variants = append(variants, acronym.String())
			}
		}
	}
	
	return variants
}

// MatchStocks 智能匹配新闻中的股票
func (sm *StockMatcher) MatchStocks(ctx context.Context, news *model.NewsData) ([]string, error) {
	matchedStocks := make(map[string]bool) // 使用map去重
	text := news.Title + " " + news.Content
	
	// 1. 股票代码匹配
	codes := sm.extractStockCodes(text)
	for _, code := range codes {
		if stock, exists := sm.stockCache[code]; exists && stock.IsActive() {
			matchedStocks[code] = true
		}
	}
	
	// 2. 公司名称精确匹配
	companyNames := sm.extractCompanyNames(text)
	for _, name := range companyNames {
		if stock, exists := sm.nameCache[name]; exists && stock.IsActive() {
			matchedStocks[stock.Symbol] = true
		}
	}
	
	// 3. 公司名称模糊匹配
	fuzzyMatches := sm.fuzzyMatchCompanyNames(text)
	for _, symbol := range fuzzyMatches {
		if stock, exists := sm.stockCache[symbol]; exists && stock.IsActive() {
			matchedStocks[symbol] = true
		}
	}
	
	// 4. 行业关键词匹配（可选）
	industryMatches := sm.matchByIndustryKeywords(text)
	for _, symbol := range industryMatches {
		if stock, exists := sm.stockCache[symbol]; exists && stock.IsActive() {
			matchedStocks[symbol] = true
		}
	}
	
	// 转换为切片
	result := make([]string, 0, len(matchedStocks))
	for symbol := range matchedStocks {
		result = append(result, symbol)
	}
	
	return result, nil
}

// extractStockCodes 提取股票代码
func (sm *StockMatcher) extractStockCodes(text string) []string {
	codes := make([]string, 0)
	
	// 使用正则表达式匹配股票代码
	matches := sm.stockCodeRegex.FindAllString(text, -1)
	for _, match := range matches {
		// 验证是否为有效的股票代码格式
		if sm.isValidStockCodeFormat(match) {
			codes = append(codes, match)
		}
	}
	
	return codes
}

// isValidStockCodeFormat 验证股票代码格式
func (sm *StockMatcher) isValidStockCodeFormat(code string) bool {
	// A股代码规则
	if len(code) == 6 {
		// 沪市：600xxx, 601xxx, 603xxx, 605xxx
		// 深市：000xxx, 001xxx, 002xxx, 003xxx
		// 创业板：300xxx
		// 科创板：688xxx
		if strings.HasPrefix(code, "60") || strings.HasPrefix(code, "00") || 
		   strings.HasPrefix(code, "30") || strings.HasPrefix(code, "68") {
			return true
		}
	}
	
	// 港股代码规则（4-5位数字）
	if len(code) >= 4 && len(code) <= 5 {
		for _, char := range code {
			if char < '0' || char > '9' {
				return false
			}
		}
		return true
	}
	
	// 美股代码规则（1-5位字母）
	if len(code) >= 1 && len(code) <= 5 {
		for _, char := range code {
			if (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') {
				return false
			}
		}
		return true
	}
	
	return false
}

// extractCompanyNames 提取公司名称
func (sm *StockMatcher) extractCompanyNames(text string) []string {
	names := make([]string, 0)
	
	// 使用正则表达式匹配公司名称模式
	// 匹配包含"公司"、"集团"、"控股"等关键词的词组
	companyRegex := regexp.MustCompile(`([\p{Han}A-Za-z0-9]+(?:股份有限公司|有限公司|集团|控股|科技|实业|投资|发展|建设))`)
	matches := companyRegex.FindAllString(text, -1)
	
	for _, match := range matches {
		// 清理匹配结果
		cleanName := strings.TrimSpace(match)
		if len(cleanName) > 2 { // 过滤太短的匹配
			names = append(names, cleanName)
		}
	}
	
	return names
}

// fuzzyMatchCompanyNames 模糊匹配公司名称
func (sm *StockMatcher) fuzzyMatchCompanyNames(text string) []string {
	matches := make([]string, 0)
	
	// 遍历所有股票，检查其名称变体是否在文本中出现
	for symbol, variants := range sm.nameVariants {
		for _, variant := range variants {
			if len(variant) >= 2 && strings.Contains(text, variant) {
				matches = append(matches, symbol)
				break // 找到一个匹配就跳出
			}
		}
	}
	
	return matches
}

// matchByIndustryKeywords 基于行业关键词匹配
func (sm *StockMatcher) matchByIndustryKeywords(text string) []string {
	matches := make([]string, 0)
	
	// 定义行业关键词映射
	industryKeywords := map[string][]string{
		"银行":     {"银行", "金融", "贷款", "存款", "利率"},
		"房地产":   {"房地产", "地产", "楼市", "房价", "土地"},
		"医药生物": {"医药", "生物", "疫苗", "药品", "医疗"},
		"电子":     {"电子", "芯片", "半导体", "集成电路"},
		"汽车":     {"汽车", "新能源车", "电动车", "车企"},
		"食品饮料": {"食品", "饮料", "白酒", "啤酒", "乳制品"},
	}
	
	// 检查文本中是否包含行业关键词
	for industry, keywords := range industryKeywords {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				// 查找该行业的股票
				industryStocks := sm.getStocksByIndustry(industry)
				matches = append(matches, industryStocks...)
				break
			}
		}
	}
	
	return matches
}

// getStocksByIndustry 根据行业获取股票列表
func (sm *StockMatcher) getStocksByIndustry(industry string) []string {
	stocks := make([]string, 0)
	
	// 从缓存中查找指定行业的股票
	for symbol, stock := range sm.stockCache {
		if stock.Industry == industry && stock.IsActive() {
			stocks = append(stocks, symbol)
		}
	}
	
	return stocks
}

// cacheToRedis 将股票信息缓存到Redis
func (sm *StockMatcher) cacheToRedis(ctx context.Context) {
	if sm.redisClient == nil {
		return
	}
	
	// 缓存股票基本信息
	for symbol, stock := range sm.stockCache {
		key := fmt.Sprintf("stock:info:%s", symbol)
		data := fmt.Sprintf("%s|%s|%s", stock.Name, stock.Industry, stock.Exchange)
		sm.redisClient.Set(ctx, key, data, 24*time.Hour)
	}
	
	// 缓存名称变体映射
	for symbol, variants := range sm.nameVariants {
		key := fmt.Sprintf("stock:variants:%s", symbol)
		data := strings.Join(variants, "|")
		sm.redisClient.Set(ctx, key, data, 24*time.Hour)
	}
}

// RefreshCache 刷新缓存
func (sm *StockMatcher) RefreshCache() {
	// 清空现有缓存
	sm.stockCache = make(map[string]*model.Stock)
	sm.nameCache = make(map[string]*model.Stock)
	sm.nameVariants = make(map[string][]string)
	
	// 重新初始化缓存
	sm.initializeCache()
}

// GetMatchingConfidence 获取匹配置信度
func (sm *StockMatcher) GetMatchingConfidence(text string, symbol string) float64 {
	stock, exists := sm.stockCache[symbol]
	if !exists {
		return 0.0
	}
	
	confidence := 0.0
	
	// 股票代码直接匹配 - 最高置信度
	if strings.Contains(text, symbol) {
		confidence += 0.9
	}
	
	// 公司全名匹配 - 高置信度
	if strings.Contains(text, stock.Name) {
		confidence += 0.8
	}
	
	// 名称变体匹配 - 中等置信度
	if variants, exists := sm.nameVariants[symbol]; exists {
		for _, variant := range variants {
			if strings.Contains(text, variant) {
				confidence += 0.6
				break
			}
		}
	}
	
	// 行业关键词匹配 - 低置信度
	if stock.Industry != "" && strings.Contains(text, stock.Industry) {
		confidence += 0.3
	}
	
	// 确保置信度不超过1.0
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	return confidence
}