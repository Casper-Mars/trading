package collection

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"data-collection-system/model"
	"data-collection-system/pkg/errors"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/sirupsen/logrus"
)

// NewsCrawlerService 新闻爬虫服务
type NewsCrawlerService struct {
	config     *NewsCrawlerConfig
	collector  *colly.Collector
	newsRepo   NewsRepository
	logger     *logrus.Logger
	userAgents []string
}

// NewsCrawlerConfig 新闻爬虫配置
type NewsCrawlerConfig struct {
	MaxDepth        int           `yaml:"max_depth"`         // 最大爬取深度
	Delay           time.Duration `yaml:"delay"`             // 请求延迟
	RandomDelay     time.Duration `yaml:"random_delay"`      // 随机延迟
	Concurrency     int           `yaml:"concurrency"`       // 并发数
	Timeout         time.Duration `yaml:"timeout"`           // 超时时间
	RetryCount      int           `yaml:"retry_count"`       // 重试次数
	Debug           bool          `yaml:"debug"`             // 是否启用调试
	RespectRobots   bool          `yaml:"respect_robots"`    // 是否遵守robots.txt
	AllowedDomains  []string      `yaml:"allowed_domains"`   // 允许的域名
	BlockedDomains  []string      `yaml:"blocked_domains"`   // 禁止的域名
}

// NewsSource 新闻源配置
type NewsSource struct {
	Name         string            `yaml:"name"`          // 新闻源名称
	BaseURL      string            `yaml:"base_url"`      // 基础URL
	ListURL      string            `yaml:"list_url"`      // 列表页URL
	Selectors    NewsSelectors     `yaml:"selectors"`     // CSS选择器
	Headers      map[string]string `yaml:"headers"`       // 请求头
	Enabled      bool              `yaml:"enabled"`       // 是否启用
	Category     string            `yaml:"category"`      // 新闻分类
	UpdateFreq   time.Duration     `yaml:"update_freq"`   // 更新频率
}

// NewsSelectors CSS选择器配置
type NewsSelectors struct {
	ListItem    string `yaml:"list_item"`    // 列表项选择器
	Title       string `yaml:"title"`        // 标题选择器
	Content     string `yaml:"content"`      // 内容选择器
	Author      string `yaml:"author"`       // 作者选择器
	PublishTime string `yaml:"publish_time"` // 发布时间选择器
	URL         string `yaml:"url"`          // 链接选择器
	Summary     string `yaml:"summary"`      // 摘要选择器
	Tags        string `yaml:"tags"`         // 标签选择器
}

// NewsRepository 新闻数据仓库接口
type NewsRepository interface {
	Create(ctx context.Context, news *model.NewsData) error
	ExistsByURL(ctx context.Context, url string) (bool, error)
	BatchCreate(ctx context.Context, newsList []*model.NewsData) error
}

// NewNewsCrawlerService 创建新闻爬虫服务
func NewNewsCrawlerService(config *NewsCrawlerConfig, newsRepo NewsRepository) *NewsCrawlerService {
	// 创建Colly收集器
	c := colly.NewCollector(
		colly.Async(true),
	)

	// 配置收集器
	if config.MaxDepth > 0 {
		c.Limit(&colly.LimitRule{
			DomainGlob:  "*",
			Parallelism: config.Concurrency,
			Delay:       config.Delay,
			RandomDelay: config.RandomDelay,
		})
	}

	// 设置允许和禁止的域名
	if len(config.AllowedDomains) > 0 {
		c.AllowedDomains = config.AllowedDomains
	}
	if len(config.BlockedDomains) > 0 {
		c.DisallowedDomains = config.BlockedDomains
	}

	// 设置超时
	c.SetRequestTimeout(config.Timeout)

	// 启用调试模式
	if config.Debug {
		c.OnRequest(func(r *colly.Request) {
			fmt.Printf("Visiting %s\n", r.URL.String())
		})
	}

	// 设置User-Agent轮换
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:89.0) Gecko/20100101 Firefox/89.0",
		"Mozilla/5.0 (X11; Linux x86_64; rv:89.0) Gecko/20100101 Firefox/89.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edge/91.0.864.59",
	}

	// 启用扩展
	extensions.RandomUserAgent(c)
	extensions.Referer(c)

	service := &NewsCrawlerService{
		collector:  c,
		newsRepo:   newsRepo,
		config:     config,
		userAgents: userAgents,
		logger:     logrus.New(),
	}

	// 设置回调函数
	service.setupCallbacks()

	return service
}

// setupCallbacks 设置爬虫回调函数
func (s *NewsCrawlerService) setupCallbacks() {
	// 请求前回调
	s.collector.OnRequest(func(r *colly.Request) {
		// 随机选择User-Agent
		userAgent := s.userAgents[rand.Intn(len(s.userAgents))]
		r.Headers.Set("User-Agent", userAgent)

		// 设置通用请求头
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("DNT", "1")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")

		s.logger.Infof("开始请求: %s", r.URL.String())
	})

	// 响应回调
	s.collector.OnResponse(func(r *colly.Response) {
		s.logger.Infof("收到响应: %s, 状态码: %d, 大小: %d bytes", r.Request.URL.String(), r.StatusCode, len(r.Body))
	})

	// 错误回调
	s.collector.OnError(func(r *colly.Response, err error) {
		s.logger.Errorf("请求失败: %s, 错误: %v", r.Request.URL.String(), err)
		
		// 实现重试机制
		if r.StatusCode == http.StatusTooManyRequests || r.StatusCode >= 500 {
			// 延迟后重试
			time.Sleep(time.Duration(rand.Intn(5)+1) * time.Second)
			r.Request.Retry()
		}
	})

	// HTML解析回调
	s.collector.OnHTML("html", func(e *colly.HTMLElement) {
		s.logger.Debugf("解析HTML: %s", e.Request.URL.String())
	})

	// 完成回调
	s.collector.OnScraped(func(r *colly.Response) {
		s.logger.Infof("完成爬取: %s", r.Request.URL.String())
	})
}

// CrawlNewsSource 爬取指定新闻源
func (s *NewsCrawlerService) CrawlNewsSource(ctx context.Context, source *NewsSource) error {
	if !source.Enabled {
		s.logger.Infof("新闻源 %s 已禁用，跳过爬取", source.Name)
		return nil
	}

	s.logger.Infof("开始爬取新闻源: %s", source.Name)

	// 创建新的收集器实例用于此次爬取
	c := s.collector.Clone()

	// 设置自定义请求头
	for key, value := range source.Headers {
		c.OnRequest(func(r *colly.Request) {
			r.Headers.Set(key, value)
		})
	}

	// 存储爬取的新闻
	var newsList []*model.NewsData

	// 设置列表页解析
	c.OnHTML(source.Selectors.ListItem, func(e *colly.HTMLElement) {
		news := s.extractNewsFromElement(e, source)
		if news != nil {
			// 检查是否已存在
			exists, err := s.newsRepo.ExistsByURL(ctx, news.URL)
			if err != nil {
				s.logger.Errorf("检查新闻是否存在失败: %v", err)
				return
			}
			if !exists {
				newsList = append(newsList, news)
				s.logger.Debugf("提取新闻: %s", news.Title)
			}
		}
	})

	// 访问列表页
	err := c.Visit(source.ListURL)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeCrawlerFailed, "访问新闻列表页失败")
	}

	// 等待所有请求完成
	c.Wait()

	// 批量保存新闻
	if len(newsList) > 0 {
		err = s.newsRepo.BatchCreate(ctx, newsList)
		if err != nil {
			return errors.Wrap(err, errors.ErrCodeDatabase, "批量保存新闻失败")
		}
		s.logger.Infof("成功爬取并保存 %d 条新闻，来源: %s", len(newsList), source.Name)
	} else {
		s.logger.Infof("未发现新的新闻，来源: %s", source.Name)
	}

	return nil
}

// extractNewsFromElement 从HTML元素中提取新闻信息
func (s *NewsCrawlerService) extractNewsFromElement(e *colly.HTMLElement, source *NewsSource) *model.NewsData {
	// 提取标题
	title := strings.TrimSpace(e.ChildText(source.Selectors.Title))
	if title == "" {
		s.logger.Debug("标题为空，跳过此新闻")
		return nil
	}

	// 提取URL
	url := e.ChildAttr(source.Selectors.URL, "href")
	if url == "" {
		s.logger.Debug("URL为空，跳过此新闻")
		return nil
	}

	// 处理相对URL
	if strings.HasPrefix(url, "/") {
		url = source.BaseURL + url
	} else if !strings.HasPrefix(url, "http") {
		url = source.BaseURL + "/" + url
	}

	// 提取内容/摘要
	content := strings.TrimSpace(e.ChildText(source.Selectors.Content))
	if content == "" {
		content = strings.TrimSpace(e.ChildText(source.Selectors.Summary))
	}

	// 提取发布时间
	publishTimeStr := strings.TrimSpace(e.ChildText(source.Selectors.PublishTime))
	publishTime := s.parsePublishTime(publishTimeStr)

	// 创建新闻数据
	news := &model.NewsData{
		Title:       title,
		Content:     content,
		URL:         url,
		Source:      source.Name,
		Category:    source.Category,
		PublishTime: publishTime,
	}

	return news
}

// parsePublishTime 解析发布时间
func (s *NewsCrawlerService) parsePublishTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Now()
	}

	// 常见的时间格式
	timeFormats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/01/02",
		"01-02 15:04",
		"01/02 15:04",
	}

	// 清理时间字符串
	timeStr = strings.TrimSpace(timeStr)
	timeStr = regexp.MustCompile(`[^\d\-/:\s]`).ReplaceAllString(timeStr, "")

	// 尝试解析
	for _, format := range timeFormats {
		if t, err := time.Parse(format, timeStr); err == nil {
			// 如果没有年份，使用当前年份
			if t.Year() == 0 {
				now := time.Now()
				t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)
			}
			return t
		}
	}

	s.logger.Debugf("无法解析时间格式: %s，使用当前时间", timeStr)
	return time.Now()
}

// extractTags 提取标签
func (s *NewsCrawlerService) extractTags(e *colly.HTMLElement, selector string) []string {
	if selector == "" {
		return []string{}
	}

	var tags []string
	e.ForEach(selector, func(i int, el *colly.HTMLElement) {
		tag := strings.TrimSpace(el.Text)
		if tag != "" {
			tags = append(tags, tag)
		}
	})

	return tags
}

// CrawlMultipleSources 爬取多个新闻源
func (s *NewsCrawlerService) CrawlMultipleSources(ctx context.Context, sources []*NewsSource) error {
	s.logger.Infof("开始爬取 %d 个新闻源", len(sources))

	var errors []error
	for _, source := range sources {
		if err := s.CrawlNewsSource(ctx, source); err != nil {
			s.logger.Errorf("爬取新闻源 %s 失败: %v", source.Name, err)
			errors = append(errors, fmt.Errorf("爬取 %s 失败: %w", source.Name, err))
			continue
		}

		// 在不同新闻源之间添加延迟
		time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)
	}

	if len(errors) > 0 {
		return fmt.Errorf("部分新闻源爬取失败: %v", errors)
	}

	s.logger.Info("所有新闻源爬取完成")
	return nil
}

// GetDefaultNewsSources 获取默认新闻源配置
func GetDefaultNewsSources() []*NewsSource {
	return []*NewsSource{
		{
			Name:     "东方财富",
			BaseURL:  "https://finance.eastmoney.com",
			ListURL:  "https://finance.eastmoney.com/news/cjxw.html",
			Category: "财经新闻",
			Enabled:  true,
			Selectors: NewsSelectors{
				ListItem:    ".news-item",
				Title:       ".news-title a",
				URL:         ".news-title a",
				Summary:     ".news-desc",
				PublishTime: ".news-time",
			},
			Headers: map[string]string{
				"Referer": "https://finance.eastmoney.com",
			},
			UpdateFreq: 30 * time.Minute,
		},
		{
			Name:     "新浪财经",
			BaseURL:  "https://finance.sina.com.cn",
			ListURL:  "https://finance.sina.com.cn/roll/",
			Category: "财经新闻",
			Enabled:  true,
			Selectors: NewsSelectors{
				ListItem:    ".feed-card-item",
				Title:       ".feed-card-title",
				URL:         ".feed-card-title",
				Summary:     ".feed-card-summary",
				PublishTime: ".feed-card-time",
			},
			Headers: map[string]string{
				"Referer": "https://finance.sina.com.cn",
			},
			UpdateFreq: 30 * time.Minute,
		},
		{
			Name:     "央行官网",
			BaseURL:  "http://www.pbc.gov.cn",
			ListURL:  "http://www.pbc.gov.cn/goutongjiaoliu/113456/113469/index.html",
			Category: "政策新闻",
			Enabled:  true,
			Selectors: NewsSelectors{
				ListItem:    ".news_list li",
				Title:       "a",
				URL:         "a",
				PublishTime: ".time",
			},
			Headers: map[string]string{
				"Referer": "http://www.pbc.gov.cn",
			},
			UpdateFreq: 60 * time.Minute,
		},
		{
			Name:     "证监会官网",
			BaseURL:  "http://www.csrc.gov.cn",
			ListURL:  "http://www.csrc.gov.cn/csrc/c100028/common_list.shtml",
			Category: "监管新闻",
			Enabled:  true,
			Selectors: NewsSelectors{
				ListItem:    ".zx_ml_list li",
				Title:       "a",
				URL:         "a",
				PublishTime: ".time",
			},
			Headers: map[string]string{
				"Referer": "http://www.csrc.gov.cn",
			},
			UpdateFreq: 60 * time.Minute,
		},
	}
}

// Stop 停止爬虫服务
func (s *NewsCrawlerService) Stop() {
	if s.collector != nil {
		// 清理回调函数
		s.collector = nil
	}
	s.logger.Info("新闻爬虫服务已停止")
}