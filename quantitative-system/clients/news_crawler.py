"""新闻爬虫客户端模块

实现金融新闻网站的爬虫功能，包括：
- 多个金融新闻网站的爬取
- robots.txt规则遵守
- 访问频率控制
- 新闻内容解析和提取
- 反爬虫机制和错误处理
"""

import asyncio
import re
import time
from datetime import datetime, timedelta
from typing import Any
from urllib.parse import urljoin, urlparse
from urllib.robotparser import RobotFileParser

import aiohttp
from bs4 import BeautifulSoup
from pydantic import BaseModel, Field

from utils.exceptions import CrawlerError, RateLimitError
from utils.logger import get_logger

logger = get_logger(__name__)


class NewsArticle(BaseModel):
    """新闻文章数据模型"""

    title: str = Field(..., description="新闻标题")
    content: str = Field(..., description="新闻内容")
    url: str = Field(..., description="新闻链接")
    source: str = Field(..., description="新闻来源")
    publish_time: datetime | None = Field(None, description="发布时间")
    author: str | None = Field(None, description="作者")
    tags: list[str] = Field(default_factory=list, description="标签")
    summary: str | None = Field(None, description="摘要")
    crawl_time: datetime = Field(default_factory=datetime.now, description="爬取时间")


class CrawlerConfig(BaseModel):
    """爬虫配置模型"""

    user_agent: str = Field(
        default="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
        description="用户代理"
    )
    request_delay: float = Field(default=1.0, description="请求间隔(秒)")
    timeout: int = Field(default=30, description="请求超时时间(秒)")
    max_retries: int = Field(default=3, description="最大重试次数")
    respect_robots_txt: bool = Field(default=True, description="是否遵守robots.txt")
    max_concurrent_requests: int = Field(default=5, description="最大并发请求数")


class RateLimiter:
    """访问频率控制器"""

    def __init__(self, delay: float = 1.0):
        self.delay = delay
        self.last_request_time = 0.0
        self._lock = asyncio.Lock()

    async def wait(self) -> None:
        """等待到允许的下次请求时间"""
        async with self._lock:
            current_time = time.time()
            time_since_last = current_time - self.last_request_time

            if time_since_last < self.delay:
                wait_time = self.delay - time_since_last
                await asyncio.sleep(wait_time)

            self.last_request_time = time.time()


class RobotsChecker:
    """robots.txt检查器"""

    def __init__(self):
        self._robots_cache: dict[str, RobotFileParser] = {}
        self._cache_expiry: dict[str, datetime] = {}

    async def can_fetch(self, url: str, user_agent: str = "*") -> bool:
        """检查是否允许爬取指定URL"""
        try:
            parsed_url = urlparse(url)
            base_url = f"{parsed_url.scheme}://{parsed_url.netloc}"

            # 检查缓存
            if base_url in self._robots_cache and datetime.now() < self._cache_expiry[base_url]:
                rp = self._robots_cache[base_url]
                return rp.can_fetch(user_agent, url)

            # 获取robots.txt
            robots_url = urljoin(base_url, "/robots.txt")
            rp = RobotFileParser()
            rp.set_url(robots_url)

            try:
                rp.read()
                self._robots_cache[base_url] = rp
                self._cache_expiry[base_url] = datetime.now() + timedelta(hours=24)
                return rp.can_fetch(user_agent, url)
            except Exception:
                # 如果无法获取robots.txt，默认允许
                return True

        except Exception as e:
            logger.warning(f"检查robots.txt失败: {e}")
            return True


class NewsContentExtractor:
    """新闻内容提取器"""

    def __init__(self):
        # 常见的新闻内容选择器
        self.content_selectors = [
            'article',
            '.article-content',
            '.news-content',
            '.content',
            '.post-content',
            '#content',
            '.entry-content',
            '.article-body'
        ]

        # 常见的标题选择器
        self.title_selectors = [
            'h1',
            '.article-title',
            '.news-title',
            '.title',
            '.post-title',
            '.entry-title'
        ]

        # 需要移除的元素
        self.remove_selectors = [
            'script',
            'style',
            '.advertisement',
            '.ad',
            '.sidebar',
            '.related',
            '.comments',
            'nav',
            'footer',
            'header'
        ]

    def extract_article(self, html: str, url: str) -> dict[str, Any]:
        """从HTML中提取新闻文章信息"""
        try:
            soup = BeautifulSoup(html, 'html.parser')

            # 移除不需要的元素
            for selector in self.remove_selectors:
                for element in soup.select(selector):
                    element.decompose()

            # 提取标题
            title = self._extract_title(soup)

            # 提取内容
            content = self._extract_content(soup)

            # 提取发布时间
            publish_time = self._extract_publish_time(soup)

            # 提取作者
            author = self._extract_author(soup)

            # 提取摘要
            summary = self._extract_summary(soup, content)

            return {
                'title': title,
                'content': content,
                'url': url,
                'publish_time': publish_time,
                'author': author,
                'summary': summary
            }

        except Exception as e:
            logger.error(f"内容提取失败 {url}: {e}")
            raise CrawlerError(f"内容提取失败: {e}") from e

    def _extract_title(self, soup: BeautifulSoup) -> str:
        """提取标题"""
        for selector in self.title_selectors:
            element = soup.select_one(selector)
            if element and element.get_text(strip=True):
                return element.get_text(strip=True)

        # 回退到页面标题
        title_tag = soup.find('title')
        if title_tag:
            return title_tag.get_text(strip=True)

        return "未知标题"

    def _extract_content(self, soup: BeautifulSoup) -> str:
        """提取正文内容"""
        for selector in self.content_selectors:
            element = soup.select_one(selector)
            if element:
                # 移除嵌套的不需要元素
                for remove_sel in self.remove_selectors:
                    for nested in element.select(remove_sel):
                        nested.decompose()

                text = element.get_text(separator='\n', strip=True)
                if len(text) > 100:  # 确保内容足够长
                    return text

        # 回退到body内容
        body = soup.find('body')
        if body:
            return body.get_text(separator='\n', strip=True)

        return ""

    def _extract_publish_time(self, soup: BeautifulSoup) -> datetime | None:
        """提取发布时间"""
        time_selectors = [
            'time[datetime]',
            '.publish-time',
            '.date',
            '.time',
            '.article-date'
        ]

        for selector in time_selectors:
            element = soup.select_one(selector)
            if element:
                # 尝试从datetime属性获取
                datetime_attr = element.get('datetime')
                if datetime_attr:
                    try:
                        return datetime.fromisoformat(datetime_attr.replace('Z', '+00:00'))
                    except ValueError:
                        pass

                # 尝试从文本内容解析
                text = element.get_text(strip=True)
                if text:
                    return self._parse_time_text(text)

        return None

    def _extract_author(self, soup: BeautifulSoup) -> str | None:
        """提取作者"""
        author_selectors = [
            '.author',
            '.byline',
            '.writer',
            '[rel="author"]'
        ]

        for selector in author_selectors:
            element = soup.select_one(selector)
            if element and element.get_text(strip=True):
                return element.get_text(strip=True)

        return None

    def _extract_summary(self, soup: BeautifulSoup, content: str) -> str | None:
        """提取摘要"""
        # 尝试从meta标签获取
        meta_desc = soup.find('meta', attrs={'name': 'description'})
        if meta_desc and meta_desc.get('content'):
            return meta_desc['content']

        # 从内容中生成摘要（前200字符）
        if content:
            summary = content[:200]
            if len(content) > 200:
                summary += "..."
            return summary

        return None

    def _parse_time_text(self, text: str) -> datetime | None:
        """解析时间文本"""
        # 常见的时间格式模式
        patterns = [
            r'(\d{4})-(\d{2})-(\d{2})\s+(\d{2}):(\d{2})',
            r'(\d{4})/(\d{2})/(\d{2})\s+(\d{2}):(\d{2})',
            r'(\d{2})-(\d{2})-(\d{4})\s+(\d{2}):(\d{2})'
        ]

        for pattern in patterns:
            match = re.search(pattern, text)
            if match:
                try:
                    groups = match.groups()
                    if len(groups) >= 5:
                        year, month, day, hour, minute = groups[:5]
                        return datetime(int(year), int(month), int(day), int(hour), int(minute))
                except ValueError:
                    continue

        return None


class NewsCrawler:
    """新闻爬虫主类"""

    def __init__(self, config: CrawlerConfig | None = None):
        self.config = config or CrawlerConfig()
        self.rate_limiter = RateLimiter(self.config.request_delay)
        self.robots_checker = RobotsChecker()
        self.content_extractor = NewsContentExtractor()
        self.session: aiohttp.ClientSession | None = None

        # 金融新闻网站配置
        self.news_sources = {
            'sina_finance': {
                'base_url': 'https://finance.sina.com.cn',
                'list_urls': [
                    'https://finance.sina.com.cn/stock/',
                    'https://finance.sina.com.cn/money/'
                ],
                'source_name': '新浪财经'
            },
            'eastmoney': {
                'base_url': 'https://finance.eastmoney.com',
                'list_urls': [
                    'https://finance.eastmoney.com/news/',
                    'https://stock.eastmoney.com/news/'
                ],
                'source_name': '东方财富'
            },
            'hexun': {
                'base_url': 'https://www.hexun.com',
                'list_urls': [
                    'https://stock.hexun.com/',
                    'https://finance.hexun.com/'
                ],
                'source_name': '和讯网'
            }
        }

    async def __aenter__(self):
        """异步上下文管理器入口"""
        await self.start()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """异步上下文管理器出口"""
        await self.close()

    async def start(self) -> None:
        """启动爬虫会话"""
        if self.session is None:
            timeout = aiohttp.ClientTimeout(total=self.config.timeout)
            connector = aiohttp.TCPConnector(
                limit=self.config.max_concurrent_requests,
                limit_per_host=self.config.max_concurrent_requests
            )

            self.session = aiohttp.ClientSession(
                timeout=timeout,
                connector=connector,
                headers={'User-Agent': self.config.user_agent}
            )

            logger.info("新闻爬虫会话已启动")

    async def close(self) -> None:
        """关闭爬虫会话"""
        if self.session:
            await self.session.close()
            self.session = None
            logger.info("新闻爬虫会话已关闭")

    async def crawl_news_list(self, source: str, limit: int = 50) -> list[str]:
        """爬取新闻列表页，获取新闻链接"""
        if source not in self.news_sources:
            raise CrawlerError(f"不支持的新闻源: {source}")

        source_config = self.news_sources[source]
        news_urls = []

        try:
            for list_url in source_config['list_urls']:
                if len(news_urls) >= limit:
                    break

                urls = await self._extract_news_urls_from_page(list_url, source_config)
                news_urls.extend(urls)

                if len(news_urls) >= limit:
                    news_urls = news_urls[:limit]
                    break

            logger.info(f"从 {source} 获取到 {len(news_urls)} 个新闻链接")
            return news_urls

        except Exception as e:
            logger.error(f"爬取新闻列表失败 {source}: {e}")
            raise CrawlerError(f"爬取新闻列表失败: {e}") from e

    async def crawl_article(self, url: str, source: str) -> NewsArticle:
        """爬取单篇新闻文章"""
        try:
            # 检查robots.txt
            if self.config.respect_robots_txt:
                can_fetch = await self.robots_checker.can_fetch(url, self.config.user_agent)
                if not can_fetch:
                    raise CrawlerError(f"robots.txt禁止访问: {url}")

            # 频率控制
            await self.rate_limiter.wait()

            # 获取页面内容
            html = await self._fetch_page(url)

            # 提取文章信息
            article_data = self.content_extractor.extract_article(html, url)
            article_data['source'] = self.news_sources.get(source, {}).get('source_name', source)

            return NewsArticle(**article_data)

        except Exception as e:
            logger.error(f"爬取文章失败 {url}: {e}")
            raise CrawlerError(f"爬取文章失败: {e}") from e

    async def crawl_news_batch(self, source: str, limit: int = 50) -> list[NewsArticle]:
        """批量爬取新闻"""
        try:
            # 获取新闻链接列表
            news_urls = await self.crawl_news_list(source, limit)

            # 并发爬取文章
            semaphore = asyncio.Semaphore(self.config.max_concurrent_requests)
            tasks = []

            for url in news_urls:
                task = self._crawl_article_with_semaphore(semaphore, url, source)
                tasks.append(task)

            # 等待所有任务完成
            results = await asyncio.gather(*tasks, return_exceptions=True)

            # 过滤成功的结果
            articles = []
            for result in results:
                if isinstance(result, NewsArticle):
                    articles.append(result)
                elif isinstance(result, Exception):
                    logger.warning(f"爬取文章失败: {result}")

            logger.info(f"成功爬取 {len(articles)} 篇文章")
            return articles

        except Exception as e:
            logger.error(f"批量爬取新闻失败: {e}")
            raise CrawlerError(f"批量爬取新闻失败: {e}") from e

    async def _crawl_article_with_semaphore(self, semaphore: asyncio.Semaphore, url: str, source: str) -> NewsArticle:
        """使用信号量控制并发的文章爬取"""
        async with semaphore:
            return await self.crawl_article(url, source)

    async def _extract_news_urls_from_page(self, list_url: str, source_config: dict) -> list[str]:
        """从列表页提取新闻链接"""
        try:
            html = await self._fetch_page(list_url)
            soup = BeautifulSoup(html, 'html.parser')

            # 查找新闻链接
            news_urls = []

            # 通用的新闻链接选择器
            link_selectors = [
                'a[href*="/news/"]',
                'a[href*="/stock/"]',
                'a[href*="/finance/"]',
                'a[href*="/money/"]',
                '.news-item a',
                '.article-item a',
                '.title a'
            ]

            for selector in link_selectors:
                links = soup.select(selector)
                for link in links:
                    href = link.get('href')
                    if href:
                        # 转换为绝对URL
                        if href.startswith('/'):
                            href = urljoin(source_config['base_url'], href)
                        elif not href.startswith('http'):
                            continue

                        # 过滤有效的新闻链接
                        if self._is_valid_news_url(href):
                            news_urls.append(href)

            # 去重并限制数量
            news_urls = list(set(news_urls))[:50]
            return news_urls

        except Exception as e:
            logger.error(f"提取新闻链接失败 {list_url}: {e}")
            return []

    def _is_valid_news_url(self, url: str) -> bool:
        """检查是否为有效的新闻URL"""
        # 排除不相关的链接
        exclude_patterns = [
            r'/video/',
            r'/live/',
            r'/comment/',
            r'/user/',
            r'/login',
            r'/register',
            r'\.(jpg|png|gif|pdf|doc)$',
            r'javascript:',
            r'mailto:'
        ]

        for pattern in exclude_patterns:
            if re.search(pattern, url, re.IGNORECASE):
                return False

        # 包含关键词的链接
        include_patterns = [
            r'/news/',
            r'/stock/',
            r'/finance/',
            r'/money/',
            r'/article/'
        ]

        for pattern in include_patterns:
            if re.search(pattern, url, re.IGNORECASE):
                return True

        return False

    async def _fetch_page(self, url: str) -> str:
        """获取页面内容"""
        if not self.session:
            await self.start()

        for attempt in range(self.config.max_retries):
            try:
                async with self.session.get(url) as response:
                    if response.status == 429:
                        # 遇到限流，增加等待时间
                        wait_time = (attempt + 1) * 2
                        logger.warning(f"遇到限流, 等待 {wait_time} 秒后重试")
                        await asyncio.sleep(wait_time)
                        raise RateLimitError("请求频率过高")

                    response.raise_for_status()

                    # 尝试获取正确的编码
                    content = await response.read()
                    encoding = response.charset or 'utf-8'

                    try:
                        return content.decode(encoding)
                    except UnicodeDecodeError:
                        # 回退到常见编码
                        for fallback_encoding in ['gbk', 'gb2312', 'utf-8']:
                            try:
                                return content.decode(fallback_encoding)
                            except UnicodeDecodeError:
                                continue

                        # 最后使用错误忽略模式
                        return content.decode('utf-8', errors='ignore')

            except (aiohttp.ClientError, RateLimitError) as e:
                if attempt == self.config.max_retries - 1:
                    raise CrawlerError(f"获取页面失败: {e}") from e

                wait_time = (attempt + 1) * 2
                logger.warning(f"请求失败, {wait_time} 秒后重试: {e}")
                await asyncio.sleep(wait_time)

        raise CrawlerError("达到最大重试次数")


# 便捷函数
async def crawl_financial_news(sources: list[str] | None = None, limit_per_source: int = 20) -> list[NewsArticle]:
    """爬取金融新闻的便捷函数"""
    if sources is None:
        sources = ['sina_finance', 'eastmoney', 'hexun']

    all_articles = []

    async with NewsCrawler() as crawler:
        for source in sources:
            try:
                articles = await crawler.crawl_news_batch(source, limit_per_source)
                all_articles.extend(articles)
                logger.info(f"从 {source} 爬取了 {len(articles)} 篇文章")
            except Exception as e:
                logger.error(f"爬取 {source} 失败: {e}")

    logger.info(f"总共爬取了 {len(all_articles)} 篇文章")
    return all_articles


if __name__ == "__main__":
    # 测试代码
    async def test_crawler():
        """测试爬虫功能"""
        articles = await crawl_financial_news(['sina_finance'], 5)
        for article in articles:
            print(f"标题: {article.title}")
            print(f"来源: {article.source}")
            print(f"链接: {article.url}")
            print(f"内容长度: {len(article.content)}")
            print("-" * 50)

    asyncio.run(test_crawler())
