"""股票数据API接口

提供股票相关数据的查询接口，包括：
- 股票基础信息查询
- 股票行情数据查询
- 财务数据查询
- 接口参数验证和错误处理
"""

from datetime import date

from fastapi import APIRouter, Depends, HTTPException, Query, status
from sqlalchemy.ext.asyncio import AsyncSession

from config.database import get_session
from models.schemas import (
    FinancialDataResponse,
    PaginatedFinancialDataResponse,
    PaginatedStockBasicInfoResponse,
    PaginatedStockDailyDataResponse,
    StockBasicInfoResponse,
    StockDailyDataResponse,
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

router = APIRouter(prefix="/stocks", tags=["stocks"])


@router.get(
    "/basic",
    response_model=PaginatedStockBasicInfoResponse,
    summary="查询股票基础信息",
    description="根据条件查询股票基础信息，支持分页、排序和过滤",
)
async def get_stocks_basic(
    # 查询参数
    ts_code: str | None = Query(None, description="TS代码，如：000001.SZ"),
    symbol: str | None = Query(None, description="股票代码，如：000001"),
    name: str | None = Query(None, description="股票名称，支持模糊查询"),
    industry: str | None = Query(None, description="所属行业"),
    market: str | None = Query(None, description="市场类型：主板、中小板、创业板等"),
    list_status: str | None = Query(
        None, description="上市状态：L-上市，D-退市，P-暂停"
    ),
    is_hs: str | None = Query(
        None, description="是否沪深港通标的：N-否，H-沪股通，S-深股通"
    ),
    # 分页参数
    page: int = Query(1, ge=1, description="页码，从1开始"),
    page_size: int = Query(20, ge=1, le=100, description="每页数量，最大100"),
    # 排序参数
    sort_field: str = Query(
        "ts_code", description="排序字段：ts_code, symbol, name, list_date"
    ),
    sort_order: str = Query("asc", description="排序方向：asc-升序，desc-降序"),
    db: AsyncSession = Depends(get_session),
) -> PaginatedStockBasicInfoResponse:
    """查询股票基础信息

    Args:
        ts_code: TS代码
        symbol: 股票代码
        name: 股票名称
        industry: 所属行业
        market: 市场类型
        list_status: 上市状态
        is_hs: 是否沪深港通标的
        page: 页码
        page_size: 每页数量
        sort_field: 排序字段
        sort_order: 排序方向
        db: 数据库会话

    Returns:
        分页的股票基础信息列表

    Raises:
        HTTPException: 参数验证失败或查询错误
    """
    try:
        # 构建查询参数
        pagination = PaginationParams(page=page, page_size=page_size)
        sort_params = SortParams(field=sort_field, order=sort_order)
        filter_params = FilterParams(
            stock_code=ts_code or symbol,
            start_date=None,
            end_date=None,
            sentiment_type=None,
            keywords=None,
        )

        # 构建额外过滤条件
        extra_filters = {}
        if name:
            extra_filters["name"] = name
        if industry:
            extra_filters["industry"] = industry
        if market:
            extra_filters["market"] = market
        if list_status:
            extra_filters["list_status"] = list_status
        if is_hs:
            extra_filters["is_hs"] = is_hs

        # 创建查询服务
        query_service = QueryService(db)

        # 执行查询
        result = await query_service.query_stocks(
            pagination=pagination,
            sort_params=sort_params,
            filter_params=filter_params,
            extra_filters=extra_filters,
        )

        # 转换为响应模型
        items = [StockBasicInfoResponse.model_validate(item) for item in result.items]

        return PaginatedStockBasicInfoResponse(
            items=items,
            total=result.total,
            page=result.page,
            page_size=result.page_size,
            total_pages=result.total_pages,
        )

    except ValidationError as e:
        logger.warning(f"股票基础信息查询参数验证失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"参数验证失败: {e!s}",
        ) from e

    except Exception as e:
        logger.error(f"股票基础信息查询失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="查询股票基础信息失败",
        ) from e


@router.get(
    "/daily",
    response_model=PaginatedStockDailyDataResponse,
    summary="查询股票日线数据",
    description="根据条件查询股票日线行情数据，支持分页、排序和时间范围过滤",
)
async def get_stocks_daily(
    # 查询参数
    ts_code: str | None = Query(None, description="TS代码，如：000001.SZ"),
    start_date: date | None = Query(None, description="开始日期，格式：YYYY-MM-DD"),
    end_date: date | None = Query(None, description="结束日期，格式：YYYY-MM-DD"),
    # 分页参数
    page: int = Query(1, ge=1, description="页码，从1开始"),
    page_size: int = Query(20, ge=1, le=100, description="每页数量，最大100"),
    # 排序参数
    sort_field: str = Query(
        "trade_date", description="排序字段：trade_date, ts_code, close, vol"
    ),
    sort_order: str = Query("desc", description="排序方向：asc-升序，desc-降序"),
    db: AsyncSession = Depends(get_session),
) -> PaginatedStockDailyDataResponse:
    """查询股票日线数据

    Args:
        ts_code: TS代码
        start_date: 开始日期
        end_date: 结束日期
        page: 页码
        page_size: 每页数量
        sort_field: 排序字段
        sort_order: 排序方向
        db: 数据库会话

    Returns:
        分页的股票日线数据列表

    Raises:
        HTTPException: 参数验证失败或查询错误
    """
    try:
        # 构建查询参数
        pagination = PaginationParams(page=page, page_size=page_size)
        sort_params = SortParams(field=sort_field, order=sort_order)
        filter_params = FilterParams(
            stock_code=ts_code,
            start_date=start_date,
            end_date=end_date,
            sentiment_type=None,
            keywords=None,
        )

        # 创建查询服务
        query_service = QueryService(db)

        # 执行查询
        result = await query_service.query_stock_daily_data(
            pagination=pagination,
            sort_params=sort_params,
            filter_params=filter_params,
        )

        # 转换为响应模型
        items = [StockDailyDataResponse.model_validate(item) for item in result.items]

        return PaginatedStockDailyDataResponse(
            items=items,
            total=result.total,
            page=result.page,
            page_size=result.page_size,
            total_pages=result.total_pages,
        )

    except ValidationError as e:
        logger.warning(f"股票日线数据查询参数验证失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"参数验证失败: {e!s}",
        ) from e
    except Exception as e:
        logger.error(f"股票日线数据查询失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="查询股票日线数据失败",
        ) from e


@router.get(
    "/financial",
    response_model=PaginatedFinancialDataResponse,
    summary="查询股票财务数据",
    description="根据条件查询股票财务数据，支持分页、排序和时间范围过滤",
)
async def get_stocks_financial(
    # 查询参数
    ts_code: str | None = Query(None, description="TS代码，如：000001.SZ"),
    start_date: date | None = Query(None, description="开始日期，格式：YYYY-MM-DD"),
    end_date: date | None = Query(None, description="结束日期，格式：YYYY-MM-DD"),
    report_type: str | None = Query(
        None,
        description="报告类型：1-合并报表，2-单季合并，3-调整单季合并表，4-调整合并报表，5-调整前合并报表",
    ),
    # 分页参数
    page: int = Query(1, ge=1, description="页码，从1开始"),
    page_size: int = Query(20, ge=1, le=100, description="每页数量，最大100"),
    # 排序参数
    sort_field: str = Query(
        "end_date", description="排序字段：end_date, ts_code, total_revenue, net_profit"
    ),
    sort_order: str = Query("desc", description="排序方向：asc-升序，desc-降序"),
    db: AsyncSession = Depends(get_session),
) -> PaginatedFinancialDataResponse:
    """查询股票财务数据

    Args:
        ts_code: TS代码
        start_date: 开始日期
        end_date: 结束日期
        report_type: 报告类型
        page: 页码
        page_size: 每页数量
        sort_field: 排序字段
        sort_order: 排序方向
        db: 数据库会话

    Returns:
        分页的股票财务数据列表

    Raises:
        HTTPException: 参数验证失败或查询错误
    """
    try:
        # 构建查询参数
        pagination = PaginationParams(page=page, page_size=page_size)
        sort_params = SortParams(field=sort_field, order=sort_order)
        filter_params = FilterParams(
            stock_code=ts_code,
            start_date=start_date,
            end_date=end_date,
            sentiment_type=None,
            keywords=None,
        )

        # 构建额外过滤条件
        extra_filters = {}
        if report_type:
            extra_filters["report_type"] = report_type

        # 创建查询服务
        query_service = QueryService(db)

        # 执行财务数据查询（需要在QueryService中实现）
        # 这里暂时使用通用查询方法，实际应该实现专门的财务数据查询方法
        result = await query_service.query_stocks(
            pagination=pagination,
            sort_params=sort_params,
            filter_params=filter_params,
            extra_filters=extra_filters,
        )

        # 转换为响应模型
        items = [FinancialDataResponse.model_validate(item) for item in result.items]

        return PaginatedFinancialDataResponse(
            items=items,
            total=result.total,
            page=result.page,
            page_size=result.page_size,
            total_pages=result.total_pages,
        )

    except ValidationError as e:
        logger.warning(f"股票财务数据查询参数验证失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"参数验证失败: {e!s}",
        ) from e
    except Exception as e:
        logger.error(f"股票财务数据查询失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="查询股票财务数据失败",
        ) from e


@router.get(
    "/{ts_code}/basic",
    response_model=StockBasicInfoResponse,
    summary="获取单个股票基础信息",
    description="根据TS代码获取单个股票的基础信息",
)
async def get_stock_basic(
    ts_code: str,
    db: AsyncSession = Depends(get_session),
) -> StockBasicInfoResponse:
    """获取单个股票基础信息

    Args:
        ts_code: TS代码
        db: 数据库会话

    Returns:
        股票基础信息

    Raises:
        HTTPException: 股票不存在或查询错误
    """
    try:
        # 构建查询参数
        pagination = PaginationParams(page=1, page_size=1)
        sort_params = SortParams(field="ts_code", order="asc")
        filter_params = FilterParams(
            stock_code=ts_code,
            start_date=None,
            end_date=None,
            sentiment_type=None,
            keywords=None,
        )

        # 创建查询服务
        query_service = QueryService(db)

        # 执行查询
        result = await query_service.query_stocks(
            pagination=pagination,
            sort_params=sort_params,
            filter_params=filter_params,
        )

        if not result.items:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"股票 {ts_code} 不存在",
            )

        # 转换为响应模型
        return StockBasicInfoResponse.model_validate(result.items[0])

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"获取股票基础信息失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="获取股票基础信息失败",
        ) from e


@router.get(
    "/{ts_code}/daily/latest",
    response_model=StockDailyDataResponse,
    summary="获取股票最新日线数据",
    description="根据TS代码获取股票最新的日线行情数据",
)
async def get_stock_latest_daily(
    ts_code: str,
    db: AsyncSession = Depends(get_session),
) -> StockDailyDataResponse:
    """获取股票最新日线数据

    Args:
        ts_code: TS代码
        db: 数据库会话

    Returns:
        最新的股票日线数据

    Raises:
        HTTPException: 股票不存在或查询错误
    """
    try:
        # 构建查询参数
        pagination = PaginationParams(page=1, page_size=1)
        sort_params = SortParams(field="trade_date", order="desc")
        filter_params = FilterParams(
            stock_code=ts_code,
            start_date=None,
            end_date=None,
            sentiment_type=None,
            keywords=None,
        )

        # 创建查询服务
        query_service = QueryService(db)

        # 执行查询
        result = await query_service.query_stock_daily_data(
            pagination=pagination,
            sort_params=sort_params,
            filter_params=filter_params,
        )

        if not result.items:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"股票 {ts_code} 的日线数据不存在",
            )

        # 转换为响应模型
        return StockDailyDataResponse.model_validate(result.items[0])

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"获取股票最新日线数据失败: {e!s}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="获取股票最新日线数据失败",
        ) from e
