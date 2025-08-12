"""新闻数据API接口

提供新闻相关数据的查询接口，包括：
- 新闻数据查询
- 情感分析结果查询
- 情感统计数据查询
- 接口参数验证和错误处理
"""

from datetime import date

from fastapi import APIRouter, Depends, HTTPException, Query, status
from sqlalchemy.ext.asyncio import AsyncSession

from config.database import get_db
from models.schemas import (
    NewsDataResponse,
    PaginatedNewsDataResponse,
    PaginatedSentimentAnalysisResponse,
    SentimentAnalysisResponse,
    SentimentStatisticsResponse,
)
from services.query_service import (
    FilterParams,
    PaginationParams,
    QueryService,
    SortParams,
)
from utils.exceptions import ValidationError
from utils.logger import get_logger

logger = get_logger(__name__)

router = APIRouter(prefix="/news", tags=["news"])


@router.get(
    "/",
    response_model=PaginatedNewsDataResponse,
    summary="查询新闻数据",
    description="根据条件查询新闻数据，支持分页、排序和关键词过滤",
)
async def get_news(
    # 查询参数
    keywords: str | None = Query(None, description="关键词，支持多个关键词用逗号分隔"),
    source: str | None = Query(None, description="新闻来源"),
    category: str | None = Query(None, description="新闻分类"),
    start_date: date | None = Query(None, description="开始日期，格式：YYYY-MM-DD"),
    end_date: date | None = Query(None, description="结束日期，格式：YYYY-MM-DD"),
    # 分页参数
    page: int = Query(1, ge=1, description="页码，从1开始"),
    page_size: int = Query(20, ge=1, le=100, description="每页数量，最大100"),
    # 排序参数
    sort_field: str = Query(
        "publish_time", description="排序字段：publish_time, title, source"
    ),
    sort_order: str = Query("desc", description="排序方向：asc-升序，desc-降序"),
    db: AsyncSession = Depends(get_db),
) -> PaginatedNewsDataResponse:
    """查询新闻数据

    Args:
        keywords: 关键词
        source: 新闻来源
        category: 新闻分类
        start_date: 开始日期
        end_date: 结束日期
        page: 页码
        page_size: 每页数量
        sort_field: 排序字段
        sort_order: 排序方向
        db: 数据库会话

    Returns:
        分页的新闻数据列表

    Raises:
        HTTPException: 参数验证失败或查询错误
    """
    try:
        # 构建查询参数
        pagination = PaginationParams(page=page, page_size=page_size)
        sort_params = SortParams(field=sort_field, order=sort_order)
        filter_params = FilterParams(
            stock_code=None,
            start_date=start_date,
            end_date=end_date,
            sentiment_type=None,
            keywords=keywords,
        )

        # 构建额外过滤条件
        extra_filters = {}
        if source:
            extra_filters["source"] = source
        if category:
            extra_filters["category"] = category

        # 创建查询服务
        query_service = QueryService(db)

        # 执行查询
        result = await query_service.query_news(
            pagination=pagination,
            sort_params=sort_params,
            filter_params=filter_params,
            extra_filters=extra_filters,
        )

        # 转换为响应模型
        items = [NewsDataResponse.model_validate(item) for item in result.items]

        return PaginatedNewsDataResponse(
            items=items,
            total=result.total,
            page=result.page,
            page_size=result.page_size,
            total_pages=result.total_pages,
        )

    except ValidationError as e:
        logger.warning(f"新闻数据查询参数验证失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"参数验证失败: {e!s}",
        ) from e
    except Exception as e:
        logger.error(f"新闻数据查询失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="查询新闻数据失败",
        ) from e


@router.get(
    "/sentiment",
    response_model=PaginatedSentimentAnalysisResponse,
    summary="查询情感分析结果",
    description="根据条件查询新闻情感分析结果，支持分页、排序和情感类型过滤",
)
async def get_sentiment_analysis(
    # 查询参数
    sentiment_type: str | None = Query(
        None, description="情感类型：positive-正面，negative-负面，neutral-中性"
    ),
    keywords: str | None = Query(None, description="关键词，支持多个关键词用逗号分隔"),
    stock_code: str | None = Query(None, description="相关股票代码"),
    start_date: date | None = Query(None, description="开始日期，格式：YYYY-MM-DD"),
    end_date: date | None = Query(None, description="结束日期，格式：YYYY-MM-DD"),
    min_confidence: float | None = Query(
        None, ge=0.0, le=1.0, description="最小置信度，范围：0.0-1.0"
    ),
    # 分页参数
    page: int = Query(1, ge=1, description="页码，从1开始"),
    page_size: int = Query(20, ge=1, le=100, description="每页数量，最大100"),
    # 排序参数
    sort_field: str = Query(
        "created_at", description="排序字段：created_at, confidence, sentiment_score"
    ),
    sort_order: str = Query("desc", description="排序方向：asc-升序，desc-降序"),
    db: AsyncSession = Depends(get_db),
) -> PaginatedSentimentAnalysisResponse:
    """查询情感分析结果

    Args:
        sentiment_type: 情感类型
        keywords: 关键词
        stock_code: 相关股票代码
        start_date: 开始日期
        end_date: 结束日期
        min_confidence: 最小置信度
        page: 页码
        page_size: 每页数量
        sort_field: 排序字段
        sort_order: 排序方向
        db: 数据库会话

    Returns:
        分页的情感分析结果列表

    Raises:
        HTTPException: 参数验证失败或查询错误
    """
    try:
        # 构建查询参数
        pagination = PaginationParams(page=page, page_size=page_size)
        sort_params = SortParams(field=sort_field, order=sort_order)
        filter_params = FilterParams(
            stock_code=stock_code,
            start_date=start_date,
            end_date=end_date,
            sentiment_type=sentiment_type,
            keywords=keywords,
        )

        # 构建额外过滤条件
        extra_filters = {}
        if min_confidence is not None:
            extra_filters["min_confidence"] = min_confidence

        # 创建查询服务
        query_service = QueryService(db)

        # 执行查询（需要在QueryService中实现情感分析查询方法）
        # 这里暂时使用通用查询方法，实际应该实现专门的情感分析查询方法
        result = await query_service.query_news(
            pagination=pagination,
            sort_params=sort_params,
            filter_params=filter_params,
            extra_filters=extra_filters,
        )

        # 转换为响应模型
        items = [
            SentimentAnalysisResponse.model_validate(item) for item in result.items
        ]

        return PaginatedSentimentAnalysisResponse(
            items=items,
            total=result.total,
            page=result.page,
            page_size=result.page_size,
            total_pages=result.total_pages,
        )

    except ValidationError as e:
        logger.warning(f"情感分析查询参数验证失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"参数验证失败: {e!s}",
        ) from e
    except Exception as e:
        logger.error(f"情感分析查询失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="查询情感分析失败",
        ) from e


@router.get(
    "/sentiment/statistics",
    response_model=SentimentStatisticsResponse,
    summary="获取情感统计数据",
    description="根据条件获取情感分析的统计数据，包括各情感类型的数量和比例",
)
async def get_sentiment_statistics(
    # 查询参数
    keywords: str | None = Query(None, description="关键词，支持多个关键词用逗号分隔"),
    stock_code: str | None = Query(None, description="相关股票代码"),
    start_date: date | None = Query(None, description="开始日期，格式：YYYY-MM-DD"),
    end_date: date | None = Query(None, description="结束日期，格式：YYYY-MM-DD"),
    group_by: str = Query(
        "sentiment",
        description="分组方式：sentiment-按情感类型，date-按日期，stock-按股票",
    ),
    db: AsyncSession = Depends(get_db),
) -> SentimentStatisticsResponse:
    """获取情感统计数据

    Args:
        keywords: 关键词
        stock_code: 相关股票代码
        start_date: 开始日期
        end_date: 结束日期
        group_by: 分组方式
        db: 数据库会话

    Returns:
        情感统计数据

    Raises:
        HTTPException: 参数验证失败或查询错误
    """
    try:
        # 构建查询参数
        filter_params = FilterParams(
            stock_code=stock_code,
            start_date=start_date,
            end_date=end_date,
            sentiment_type=None,
            keywords=keywords,
        )

        # 创建查询服务
        query_service = QueryService(db)

        # 执行统计查询
        result = await query_service.aggregate_sentiment_statistics(
            filter_params=filter_params,
            group_by=group_by,
        )

        # 转换为响应模型
        return SentimentStatisticsResponse.model_validate(result)

    except ValidationError as e:
        logger.warning(f"情感统计查询参数验证失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"参数验证失败: {e!s}",
        ) from e
    except Exception as e:
        logger.error(f"情感统计查询失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="查询情感统计失败",
        ) from e


@router.get(
    "/{news_id}",
    response_model=NewsDataResponse,
    summary="获取单条新闻详情",
    description="根据新闻ID获取单条新闻的详细信息",
)
async def get_news_detail(
    news_id: int,
    db: AsyncSession = Depends(get_db),
) -> NewsDataResponse:
    """获取单条新闻详情

    Args:
        news_id: 新闻ID
        db: 数据库会话

    Returns:
        新闻详细信息

    Raises:
        HTTPException: 新闻不存在或查询错误
    """
    try:
        # 构建查询参数
        pagination = PaginationParams(page=1, page_size=1)
        sort_params = SortParams(field="id", order="asc")
        filter_params = FilterParams(
            stock_code=None,
            start_date=None,
            end_date=None,
            sentiment_type=None,
            keywords=None,
        )

        # 构建额外过滤条件
        extra_filters = {"id": news_id}

        # 创建查询服务
        query_service = QueryService(db)

        # 执行查询
        result = await query_service.query_news(
            pagination=pagination,
            sort_params=sort_params,
            filter_params=filter_params,
            extra_filters=extra_filters,
        )

        if not result.items:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"新闻 {news_id} 不存在",
            )

        # 转换为响应模型
        return NewsDataResponse.model_validate(result.items[0])

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"获取新闻详情失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="获取新闻详情失败",
        ) from e


@router.get(
    "/{news_id}/sentiment",
    response_model=SentimentAnalysisResponse,
    summary="获取新闻情感分析结果",
    description="根据新闻ID获取该新闻的情感分析结果",
)
async def get_news_sentiment(
    news_id: int,
    db: AsyncSession = Depends(get_db),
) -> SentimentAnalysisResponse:
    """获取新闻情感分析结果

    Args:
        news_id: 新闻ID
        db: 数据库会话

    Returns:
        新闻情感分析结果

    Raises:
        HTTPException: 新闻或情感分析结果不存在
    """
    try:
        # 构建查询参数
        pagination = PaginationParams(page=1, page_size=1)
        sort_params = SortParams(field="created_at", order="desc")
        filter_params = FilterParams(
            stock_code=None,
            start_date=None,
            end_date=None,
            sentiment_type=None,
            keywords=None,
        )

        # 构建额外过滤条件
        extra_filters = {"news_id": news_id}

        # 创建查询服务
        query_service = QueryService(db)

        # 执行查询（需要在QueryService中实现情感分析查询方法）
        # 这里暂时使用通用查询方法，实际应该实现专门的情感分析查询方法
        result = await query_service.query_news(
            pagination=pagination,
            sort_params=sort_params,
            filter_params=filter_params,
            extra_filters=extra_filters,
        )

        if not result.items:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"新闻 {news_id} 的情感分析结果不存在",
            )

        # 转换为响应模型
        return SentimentAnalysisResponse.model_validate(result.items[0])

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"获取新闻情感分析结果失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="获取新闻情感分析结果失败",
        ) from e
