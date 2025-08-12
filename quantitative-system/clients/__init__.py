"""外部客户端模块"""

from .data_collection_client import DataCollectionClient
from .news_crawler import CrawlerConfig, NewsArticle, NewsCrawler, crawl_financial_news
from .tushare_client import TushareClient

__all__ = [
    "CrawlerConfig",
    "DataCollectionClient",
    "NewsArticle",
    "NewsCrawler",
    "TushareClient",
    "crawl_financial_news",
]
