"""新闻数据仓库模块

提供新闻数据的CRUD操作和查询功能。
"""

from datetime import datetime
from typing import Any

from sqlalchemy import and_, desc, func, or_
from sqlmodel import select

from models.database import NewsData
from repositories.base_repo import BaseRepository


class NewsRepository(BaseRepository):
    """新闻数据仓库类

    提供新闻数据的增删改查操作，支持复杂查询和批量处理。
    """

    async def create_news(self, news_data: dict[str, Any]) -> NewsData:
        """创建新闻记录

        Args:
            news_data: 新闻数据字典

        Returns:
            NewsData: 创建的新闻记录
        """
        news = NewsData(**news_data)
        self.session.add(news)
        await self.session.commit()
        await self.session.refresh(news)
        return news

    async def get_news_by_id(self, news_id: int) -> NewsData | None:
        """根据ID获取新闻

        Args:
            news_id: 新闻ID

        Returns:
            NewsData | None: 新闻记录或None
        """
        statement = select(NewsData).where(NewsData.id == news_id)
        result = await self.session.exec(statement)
        return result.first()

    async def get_unprocessed_news(
        self, limit: int = 100, offset: int = 0
    ) -> list[NewsData]:
        """获取未处理的新闻

        Args:
            limit: 限制数量
            offset: 偏移量

        Returns:
            list[NewsData]: 未处理的新闻列表
        """
        statement = (
            select(NewsData)
            .where(not NewsData.is_processed)
            .order_by(desc(NewsData.created_at))
            .offset(offset)
            .limit(limit)
        )
        result = await self.session.exec(statement)
        return list(result.all())

    async def get_news_by_date_range(
        self,
        start_date: datetime,
        end_date: datetime,
        limit: int = 100,
        offset: int = 0,
    ) -> list[NewsData]:
        """根据日期范围获取新闻

        Args:
            start_date: 开始日期
            end_date: 结束日期
            limit: 限制数量
            offset: 偏移量

        Returns:
            list[NewsData]: 新闻列表
        """
        statement = (
            select(NewsData)
            .where(
                and_(
                    NewsData.publish_time >= start_date,
                    NewsData.publish_time <= end_date,
                )
            )
            .order_by(desc(NewsData.publish_time))
            .offset(offset)
            .limit(limit)
        )
        result = await self.session.exec(statement)
        return list(result.all())

    async def get_news_by_source(
        self, source: str, limit: int = 100, offset: int = 0
    ) -> list[NewsData]:
        """根据来源获取新闻

        Args:
            source: 新闻来源
            limit: 限制数量
            offset: 偏移量

        Returns:
            list[NewsData]: 新闻列表
        """
        statement = (
            select(NewsData)
            .where(NewsData.source == source)
            .order_by(desc(NewsData.publish_time))
            .offset(offset)
            .limit(limit)
        )
        result = await self.session.exec(statement)
        return list(result.all())

    async def get_news_by_category(
        self, category: str, limit: int = 100, offset: int = 0
    ) -> list[NewsData]:
        """根据分类获取新闻

        Args:
            category: 新闻分类
            limit: 限制数量
            offset: 偏移量

        Returns:
            list[NewsData]: 新闻列表
        """
        statement = (
            select(NewsData)
            .where(NewsData.category == category)
            .order_by(desc(NewsData.publish_time))
            .offset(offset)
            .limit(limit)
        )
        result = await self.session.exec(statement)
        return list(result.all())

    async def search_news_by_keyword(
        self, keyword: str, limit: int = 100, offset: int = 0
    ) -> list[NewsData]:
        """根据关键词搜索新闻

        Args:
            keyword: 搜索关键词
            limit: 限制数量
            offset: 偏移量

        Returns:
            list[NewsData]: 新闻列表
        """
        statement = (
            select(NewsData)
            .where(
                or_(
                    NewsData.title.contains(keyword),
                    NewsData.content.contains(keyword),
                    NewsData.summary.contains(keyword),
                )
            )
            .order_by(desc(NewsData.publish_time))
            .offset(offset)
            .limit(limit)
        )
        result = await self.session.exec(statement)
        return list(result.all())

    async def get_news_by_stocks(
        self, stock_codes: list[str], limit: int = 100, offset: int = 0
    ) -> list[NewsData]:
        """根据股票代码获取相关新闻

        Args:
            stock_codes: 股票代码列表
            limit: 限制数量
            offset: 偏移量

        Returns:
            list[NewsData]: 新闻列表
        """
        # 使用JSON查询功能查找相关股票
        conditions = []
        for stock_code in stock_codes:
            conditions.append(NewsData.related_stocks.contains([stock_code]))

        statement = (
            select(NewsData)
            .where(or_(*conditions))
            .order_by(desc(NewsData.publish_time))
            .offset(offset)
            .limit(limit)
        )
        result = await self.session.exec(statement)
        return list(result.all())

    async def update_news_processing_status(
        self, news_id: int, is_processed: bool = True
    ) -> bool:
        """更新新闻处理状态

        Args:
            news_id: 新闻ID
            is_processed: 是否已处理

        Returns:
            bool: 更新是否成功
        """
        news = await self.get_news_by_id(news_id)
        if not news:
            return False

        news.is_processed = is_processed
        news.updated_at = datetime.now()
        await self.session.commit()
        return True

    async def update_news_sentiment(
        self,
        news_id: int,
        sentiment_score: float,
        sentiment_label: str,
        keywords: list[str] | None = None,
    ) -> bool:
        """更新新闻情感分析结果

        Args:
            news_id: 新闻ID
            sentiment_score: 情感分数
            sentiment_label: 情感标签
            keywords: 关键词列表

        Returns:
            bool: 更新是否成功
        """
        news = await self.get_news_by_id(news_id)
        if not news:
            return False

        news.sentiment_score = sentiment_score
        news.sentiment_label = sentiment_label
        if keywords:
            news.keywords = keywords
        news.updated_at = datetime.now()
        await self.session.commit()
        return True

    async def batch_update_processing_status(
        self, news_ids: list[int], is_processed: bool = True
    ) -> int:
        """批量更新新闻处理状态

        Args:
            news_ids: 新闻ID列表
            is_processed: 是否已处理

        Returns:
            int: 更新的记录数
        """
        statement = select(NewsData).where(NewsData.id.in_(news_ids))
        result = await self.session.exec(statement)
        news_list = list(result.all())

        updated_count = 0
        for news in news_list:
            news.is_processed = is_processed
            news.updated_at = datetime.now()
            updated_count += 1

        await self.session.commit()
        return updated_count

    async def get_news_count_by_status(self, is_processed: bool) -> int:
        """获取指定状态的新闻数量

        Args:
            is_processed: 是否已处理

        Returns:
            int: 新闻数量
        """
        statement = select(func.count(NewsData.id)).where(
            NewsData.is_processed == is_processed
        )
        result = await self.session.exec(statement)
        return result.first() or 0

    async def delete_news(self, news_id: int) -> bool:
        """删除新闻记录

        Args:
            news_id: 新闻ID

        Returns:
            bool: 删除是否成功
        """
        news = await self.get_news_by_id(news_id)
        if not news:
            return False

        await self.session.delete(news)
        await self.session.commit()
        return True

    async def get_latest_news(
        self, limit: int = 10, category: str | None = None
    ) -> list[NewsData]:
        """获取最新新闻

        Args:
            limit: 限制数量
            category: 可选的分类过滤

        Returns:
            list[NewsData]: 最新新闻列表
        """
        statement = select(NewsData).order_by(desc(NewsData.publish_time))

        if category:
            statement = statement.where(NewsData.category == category)

        statement = statement.limit(limit)
        result = await self.session.exec(statement)
        return list(result.all())
