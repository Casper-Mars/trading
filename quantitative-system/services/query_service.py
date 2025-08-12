"""统一查询服务

实现统一的数据查询服务，包括：
- 分页查询功能
- 结果聚合功能
- 查询参数验证
- 缓存集成
- 查询优化
"""

import hashlib
import json
from datetime import datetime
from typing import Any

from pydantic import BaseModel, Field, validator
from sqlalchemy import and_, desc, func
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload
from sqlalchemy.sql import Select

from models.database import NewsData, SentimentAnalysis, StockBasicInfo, StockDailyData
from models.enums import CacheType, SentimentType
from repositories.cache_repo import CacheRepo
from utils.exceptions import ValidationError
from utils.logger import get_logger

logger = get_logger(__name__)


class PaginationParams(BaseModel):
    """分页参数模型"""

    page: int = Field(default=1, ge=1, description="页码，从1开始")
    page_size: int = Field(default=20, ge=1, le=100, description="每页数量，最大100")

    @property
    def offset(self) -> int:
        """计算偏移量"""
        return (self.page - 1) * self.page_size

    @property
    def limit(self) -> int:
        """获取限制数量"""
        return self.page_size


class SortParams(BaseModel):
    """排序参数模型"""

    field: str = Field(description="排序字段")
    order: str = Field(default="desc", pattern="^(asc|desc)$", description="排序方向")

    @validator("field")
    def validate_field(cls, v):  # noqa: N805
        """验证排序字段"""
        allowed_fields = {
            "created_at",
            "updated_at",
            "trade_date",
            "close_price",
            "volume",
            "market_cap",
            "sentiment_score",
            "published_at",
            # 股票相关字段
            "ts_code",
            "symbol",
            "name",
            "industry",
            "market",
            "list_date",
            "list_status",
            # 股票价格相关字段
            "open",
            "high",
            "low",
            "close",
            "pre_close",
            "change",
            "pct_chg",
            "vol",
            "amount",
        }
        if v not in allowed_fields:
            raise ValueError(f"不支持的排序字段: {v}")
        return v


class FilterParams(BaseModel):
    """过滤参数模型"""

    stock_code: str | None = Field(None, description="股票代码")
    start_date: datetime | None = Field(None, description="开始日期")
    end_date: datetime | None = Field(None, description="结束日期")
    sentiment_type: SentimentType | None = Field(None, description="情感类型")
    keywords: str | None = Field(None, description="关键词搜索")

    @validator("end_date")
    def validate_date_range(cls, v, values):  # noqa: N805
        """验证日期范围"""
        if (
            v
            and "start_date" in values
            and values["start_date"]
            and v < values["start_date"]
        ):
            raise ValueError("结束日期不能早于开始日期")
        return v


class QueryResult(BaseModel):
    """查询结果模型"""

    data: list[dict[str, Any]] = Field(description="查询数据")
    total: int = Field(description="总记录数")
    page: int = Field(description="当前页码")
    page_size: int = Field(description="每页数量")
    total_pages: int = Field(description="总页数")
    has_next: bool = Field(description="是否有下一页")
    has_prev: bool = Field(description="是否有上一页")

    @classmethod
    def create(
        cls, data: list[dict[str, Any]], total: int, pagination: PaginationParams
    ) -> "QueryResult":
        """创建查询结果"""
        total_pages = (total + pagination.page_size - 1) // pagination.page_size

        return cls(
            data=data,
            total=total,
            page=pagination.page,
            page_size=pagination.page_size,
            total_pages=total_pages,
            has_next=pagination.page < total_pages,
            has_prev=pagination.page > 1,
        )


class QueryService:
    """统一查询服务

    提供统一的数据查询接口，支持分页、排序、过滤、缓存等功能
    """

    def __init__(self, cache_repo: CacheRepo | None = None):
        """初始化查询服务

        Args:
            cache_repo: 缓存仓库实例
        """
        self.cache_repo = cache_repo or CacheRepo()
        self.cache_ttl = 300  # 默认缓存5分钟

    def _generate_cache_key(self, query_type: str, params: dict[str, Any]) -> str:
        """生成缓存键

        Args:
            query_type: 查询类型
            params: 查询参数

        Returns:
            缓存键
        """
        # 将参数序列化并生成哈希
        params_str = json.dumps(params, sort_keys=True, default=str)
        params_hash = hashlib.sha256(params_str.encode()).hexdigest()[:8]
        return f"query:{query_type}:{params_hash}"

    def _apply_filters(
        self, query: Select, filters: FilterParams, model_class
    ) -> Select:
        """应用过滤条件

        Args:
            query: SQLAlchemy查询对象
            filters: 过滤参数
            model_class: 模型类

        Returns:
            应用过滤条件后的查询对象
        """
        conditions = []

        # 股票代码过滤
        if filters.stock_code:
            if hasattr(model_class, "stock_code"):
                conditions.append(model_class.stock_code == filters.stock_code)
            elif hasattr(model_class, "code"):
                conditions.append(model_class.code == filters.stock_code)

        # 日期范围过滤
        date_field = None
        if hasattr(model_class, "trade_date"):
            date_field = model_class.trade_date
        elif hasattr(model_class, "published_at"):
            date_field = model_class.published_at
        elif hasattr(model_class, "created_at"):
            date_field = model_class.created_at

        if date_field is not None:
            if filters.start_date:
                conditions.append(date_field >= filters.start_date)
            if filters.end_date:
                conditions.append(date_field <= filters.end_date)

        # 情感类型过滤
        if filters.sentiment_type and hasattr(model_class, "sentiment_type"):
            conditions.append(model_class.sentiment_type == filters.sentiment_type)

        # 关键词搜索
        if filters.keywords:
            if hasattr(model_class, "title"):
                conditions.append(model_class.title.contains(filters.keywords))
            elif hasattr(model_class, "content"):
                conditions.append(model_class.content.contains(filters.keywords))
            elif hasattr(model_class, "name"):
                conditions.append(model_class.name.contains(filters.keywords))

        # 应用所有条件
        if conditions:
            query = query.where(and_(*conditions))

        return query

    def _apply_sorting(
        self, query: Select, sort_params: SortParams | None, model_class
    ) -> Select:
        """应用排序

        Args:
            query: SQLAlchemy查询对象
            sort_params: 排序参数
            model_class: 模型类

        Returns:
            应用排序后的查询对象
        """
        if not sort_params:
            # 默认排序
            if hasattr(model_class, "created_at"):
                return query.order_by(desc(model_class.created_at))
            elif hasattr(model_class, "trade_date"):
                return query.order_by(desc(model_class.trade_date))
            return query

        # 获取排序字段
        sort_field = getattr(model_class, sort_params.field, None)
        if sort_field is None:
            logger.warning(
                f"模型 {model_class.__name__} 不存在字段 {sort_params.field}"
            )
            return query

        # 应用排序
        if sort_params.order == "asc":
            return query.order_by(sort_field)
        else:
            return query.order_by(desc(sort_field))

    async def query_stocks(
        self,
        session: AsyncSession,
        pagination: PaginationParams,
        filters: FilterParams | None = None,
        sort_params: SortParams | None = None,
        use_cache: bool = True,
    ) -> QueryResult:
        """查询股票数据

        Args:
            session: 数据库会话
            pagination: 分页参数
            filters: 过滤参数
            sort_params: 排序参数
            use_cache: 是否使用缓存

        Returns:
            查询结果
        """
        # 生成缓存键
        cache_key = None
        if use_cache:
            params = {
                "pagination": pagination.dict(),
                "filters": filters.dict() if filters else {},
                "sort": sort_params.dict() if sort_params else {},
            }
            cache_key = self._generate_cache_key("stocks", params)

            # 尝试从缓存获取
            cached_result = await self.cache_repo.get(CacheType.QUERY_RESULT, cache_key)
            if cached_result:
                logger.debug(f"从缓存获取股票查询结果: {cache_key}")
                return QueryResult(**cached_result)

        try:
            # 构建基础查询
            query = session.query(StockBasicInfo)

            # 应用过滤条件
            if filters:
                query = self._apply_filters(query, filters, StockBasicInfo)

            # 获取总数
            total_query = query.statement.with_only_columns(func.count())
            total_result = await session.execute(total_query)
            total = total_result.scalar()

            # 应用排序
            query = self._apply_sorting(query, sort_params, StockBasicInfo)

            # 应用分页
            query = query.offset(pagination.offset).limit(pagination.limit)

            # 执行查询
            result = await session.execute(query)
            stocks = result.scalars().all()

            # 转换为字典格式
            data = [
                {
                    "code": stock.code,
                    "name": stock.name,
                    "industry": stock.industry,
                    "market": stock.market,
                    "list_date": stock.list_date.isoformat()
                    if stock.list_date
                    else None,
                    "created_at": stock.created_at.isoformat(),
                    "updated_at": stock.updated_at.isoformat(),
                }
                for stock in stocks
            ]

            # 创建查询结果
            query_result = QueryResult.create(data, total, pagination)

            # 缓存结果
            if use_cache and cache_key:
                await self.cache_repo.set(
                    CacheType.QUERY_RESULT,
                    cache_key,
                    query_result.dict(),
                    ttl=self.cache_ttl,
                )
                logger.debug(f"缓存股票查询结果: {cache_key}")

            return query_result

        except Exception as e:
            logger.error(f"查询股票数据失败: {e}")
            raise ValidationError(f"查询股票数据失败: {e}") from e

    async def query_stock_daily(
        self,
        session: AsyncSession,
        pagination: PaginationParams,
        filters: FilterParams | None = None,
        sort_params: SortParams | None = None,
        use_cache: bool = True,
    ) -> QueryResult:
        """查询股票日线数据

        Args:
            session: 数据库会话
            pagination: 分页参数
            filters: 过滤参数
            sort_params: 排序参数
            use_cache: 是否使用缓存

        Returns:
            查询结果
        """
        # 生成缓存键
        cache_key = None
        if use_cache:
            params = {
                "pagination": pagination.dict(),
                "filters": filters.dict() if filters else {},
                "sort": sort_params.dict() if sort_params else {},
            }
            cache_key = self._generate_cache_key("stock_daily", params)

            # 尝试从缓存获取
            cached_result = await self.cache_repo.get(CacheType.QUERY_RESULT, cache_key)
            if cached_result:
                logger.debug(f"从缓存获取股票日线查询结果: {cache_key}")
                return QueryResult(**cached_result)

        try:
            # 构建基础查询
            query = session.query(StockDailyData).options(selectinload(StockDailyData.stock))

            # 应用过滤条件
            if filters:
                query = self._apply_filters(query, filters, StockDailyData)

            # 获取总数
            total_query = query.statement.with_only_columns(func.count())
            total_result = await session.execute(total_query)
            total = total_result.scalar()

            # 应用排序
            query = self._apply_sorting(query, sort_params, StockDailyData)

            # 应用分页
            query = query.offset(pagination.offset).limit(pagination.limit)

            # 执行查询
            result = await session.execute(query)
            stock_dailies = result.scalars().all()

            # 转换为字典格式
            data = [
                {
                    "stock_code": daily.stock_code,
                    "stock_name": daily.stock.name if daily.stock else None,
                    "trade_date": daily.trade_date.isoformat(),
                    "open_price": float(daily.open_price),
                    "high_price": float(daily.high_price),
                    "low_price": float(daily.low_price),
                    "close_price": float(daily.close_price),
                    "volume": daily.volume,
                    "amount": float(daily.amount),
                    "turnover_rate": float(daily.turnover_rate)
                    if daily.turnover_rate
                    else None,
                    "pe_ratio": float(daily.pe_ratio) if daily.pe_ratio else None,
                    "pb_ratio": float(daily.pb_ratio) if daily.pb_ratio else None,
                    "created_at": daily.created_at.isoformat(),
                    "updated_at": daily.updated_at.isoformat(),
                }
                for daily in stock_dailies
            ]

            # 创建查询结果
            query_result = QueryResult.create(data, total, pagination)

            # 缓存结果
            if use_cache and cache_key:
                await self.cache_repo.set(
                    CacheType.QUERY_RESULT,
                    cache_key,
                    query_result.dict(),
                    ttl=self.cache_ttl,
                )
                logger.debug(f"缓存股票日线查询结果: {cache_key}")

            return query_result

        except Exception as e:
            logger.error(f"查询股票日线数据失败: {e}")
            raise ValidationError(f"查询股票日线数据失败: {e}") from e

    async def query_news(
        self,
        session: AsyncSession,
        pagination: PaginationParams,
        filters: FilterParams | None = None,
        sort_params: SortParams | None = None,
        use_cache: bool = True,
    ) -> QueryResult:
        """查询新闻数据

        Args:
            session: 数据库会话
            pagination: 分页参数
            filters: 过滤参数
            sort_params: 排序参数
            use_cache: 是否使用缓存

        Returns:
            查询结果
        """
        # 生成缓存键
        cache_key = None
        if use_cache:
            params = {
                "pagination": pagination.dict(),
                "filters": filters.dict() if filters else {},
                "sort": sort_params.dict() if sort_params else {},
            }
            cache_key = self._generate_cache_key("news", params)

            # 尝试从缓存获取
            cached_result = await self.cache_repo.get(CacheType.QUERY_RESULT, cache_key)
            if cached_result:
                logger.debug(f"从缓存获取新闻查询结果: {cache_key}")
                return QueryResult(**cached_result)

        try:
            # 构建基础查询
            query = session.query(NewsData).options(selectinload(NewsData.sentiment_analysis))

            # 应用过滤条件
            if filters:
                query = self._apply_filters(query, filters, NewsData)

            # 获取总数
            total_query = query.statement.with_only_columns(func.count())
            total_result = await session.execute(total_query)
            total = total_result.scalar()

            # 应用排序
            query = self._apply_sorting(query, sort_params, NewsData)

            # 应用分页
            query = query.offset(pagination.offset).limit(pagination.limit)

            # 执行查询
            result = await session.execute(query)
            news_list = result.scalars().all()

            # 转换为字典格式
            data = [
                {
                    "id": news.id,
                    "title": news.title,
                    "content": news.content[:500] + "..."
                    if len(news.content) > 500
                    else news.content,
                    "source": news.source,
                    "url": news.url,
                    "published_at": news.published_at.isoformat()
                    if news.published_at
                    else None,
                    "stock_codes": news.stock_codes,
                    "sentiment_score": float(news.sentiment_analysis[0].sentiment_score)
                    if news.sentiment_analysis
                    else None,
                    "sentiment_type": news.sentiment_analysis[0].sentiment_type.value
                    if news.sentiment_analysis
                    else None,
                    "created_at": news.created_at.isoformat(),
                    "updated_at": news.updated_at.isoformat(),
                }
                for news in news_list
            ]

            # 创建查询结果
            query_result = QueryResult.create(data, total, pagination)

            # 缓存结果
            if use_cache and cache_key:
                await self.cache_repo.set(
                    CacheType.QUERY_RESULT,
                    cache_key,
                    query_result.dict(),
                    ttl=self.cache_ttl,
                )
                logger.debug(f"缓存新闻查询结果: {cache_key}")

            return query_result

        except Exception as e:
            logger.error(f"查询新闻数据失败: {e}")
            raise ValidationError(f"查询新闻数据失败: {e}") from e

    async def aggregate_sentiment_stats(
        self,
        session: AsyncSession,
        filters: FilterParams | None = None,
        use_cache: bool = True,
    ) -> dict[str, Any]:
        """聚合情感分析统计数据

        Args:
            session: 数据库会话
            filters: 过滤参数
            use_cache: 是否使用缓存

        Returns:
            情感分析统计结果
        """
        # 生成缓存键
        cache_key = None
        if use_cache:
            params = {"filters": filters.dict() if filters else {}}
            cache_key = self._generate_cache_key("sentiment_stats", params)

            # 尝试从缓存获取
            cached_result = await self.cache_repo.get(CacheType.QUERY_RESULT, cache_key)
            if cached_result:
                logger.debug(f"从缓存获取情感统计结果: {cache_key}")
                return cached_result

        try:
            # 构建基础查询
            query = session.query(SentimentAnalysis)

            # 应用过滤条件
            if filters:
                # 通过关联的新闻表进行过滤
                query = query.join(NewsData)
                if filters.stock_code:
                    query = query.filter(NewsData.stock_codes.contains(filters.stock_code))
                if filters.start_date:
                    query = query.filter(NewsData.published_at >= filters.start_date)
                if filters.end_date:
                    query = query.filter(NewsData.published_at <= filters.end_date)

            # 统计各种情感类型的数量
            sentiment_counts = await session.execute(
                query.with_entities(
                    SentimentAnalysis.sentiment_type,
                    func.count(SentimentAnalysis.id).label("count"),
                ).group_by(SentimentAnalysis.sentiment_type)
            )

            # 统计平均情感分数
            avg_score_result = await session.execute(
                query.with_entities(
                    func.avg(SentimentAnalysis.sentiment_score).label("avg_score")
                )
            )

            # 统计总数
            total_result = await session.execute(
                query.with_entities(func.count(SentimentAnalysis.id))
            )

            # 组装结果
            sentiment_distribution = {}
            for sentiment_type, count in sentiment_counts:
                sentiment_distribution[sentiment_type.value] = count

            avg_score = avg_score_result.scalar() or 0.0
            total_count = total_result.scalar() or 0

            result = {
                "total_count": total_count,
                "average_score": round(float(avg_score), 4),
                "sentiment_distribution": sentiment_distribution,
                "positive_ratio": round(
                    sentiment_distribution.get("positive", 0)
                    / max(total_count, 1)
                    * 100,
                    2,
                ),
                "negative_ratio": round(
                    sentiment_distribution.get("negative", 0)
                    / max(total_count, 1)
                    * 100,
                    2,
                ),
                "neutral_ratio": round(
                    sentiment_distribution.get("neutral", 0)
                    / max(total_count, 1)
                    * 100,
                    2,
                ),
            }

            # 缓存结果
            if use_cache and cache_key:
                await self.cache_repo.set(
                    CacheType.QUERY_RESULT, cache_key, result, ttl=self.cache_ttl
                )
                logger.debug(f"缓存情感统计结果: {cache_key}")

            return result

        except Exception as e:
            logger.error(f"聚合情感分析统计失败: {e}")
            raise ValidationError(f"聚合情感分析统计失败: {e}") from e

    async def clear_cache(self, pattern: str | None = None) -> int:
        """清理查询缓存

        Args:
            pattern: 缓存键模式，如果为None则清理所有查询缓存

        Returns:
            清理的缓存数量
        """
        try:
            if pattern:
                return await self.cache_repo.delete_pattern(f"query:*{pattern}*")
            else:
                return await self.cache_repo.delete_pattern("query:*")
        except Exception as e:
            logger.error(f"清理查询缓存失败: {e}")
            return 0
