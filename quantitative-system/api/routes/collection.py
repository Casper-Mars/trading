"""手动数据采集API接口

提供手动触发数据采集的接口，包括：
- 全量数据采集
- 增量数据采集
- 指定股票数据采集
- 日期范围数据采集
"""

from datetime import date, datetime

from fastapi import APIRouter, Depends, HTTPException, Query, status
from pydantic import BaseModel, Field
from sqlalchemy.ext.asyncio import AsyncSession

from biz.data_collection_orchestrator import (
    DataCollectionOrchestrator,
    DataCollectionRequest,
)
from config.database import get_session
from models.enums import TaskStatus, TaskType
from repositories import task_repo
from utils.exceptions import DataCollectionError, OrchestrationError
from utils.logger import get_logger

logger = get_logger(__name__)

router = APIRouter(prefix="/collection", tags=["data-collection"])


# ============= 请求模型 =============


class FullCollectionRequest(BaseModel):
    """全量数据采集请求"""

    task_type: TaskType = Field(..., description="采集任务类型")
    force_update: bool = Field(default=False, description="是否强制更新")
    batch_size: int = Field(default=100, ge=1, le=1000, description="批处理大小")
    quality_check: bool = Field(default=True, description="是否进行质量检查")


class IncrementalCollectionRequest(BaseModel):
    """增量数据采集请求"""

    task_type: TaskType = Field(..., description="采集任务类型")
    days_back: int = Field(default=1, ge=1, le=30, description="回溯天数")
    force_update: bool = Field(default=False, description="是否强制更新")
    batch_size: int = Field(default=100, ge=1, le=1000, description="批处理大小")
    quality_check: bool = Field(default=True, description="是否进行质量检查")


class StockCollectionRequest(BaseModel):
    """指定股票数据采集请求"""

    task_type: TaskType = Field(..., description="采集任务类型")
    symbols: list[str] = Field(..., min_length=1, max_length=100, description="股票代码列表")
    target_date: date | None = Field(default=None, description="目标日期，默认为最新交易日")
    force_update: bool = Field(default=False, description="是否强制更新")
    batch_size: int = Field(default=100, ge=1, le=1000, description="批处理大小")
    quality_check: bool = Field(default=True, description="是否进行质量检查")


class RangeCollectionRequest(BaseModel):
    """日期范围数据采集请求"""

    task_type: TaskType = Field(..., description="采集任务类型")
    start_date: date = Field(..., description="开始日期")
    end_date: date = Field(..., description="结束日期")
    symbols: list[str] | None = Field(default=None, description="股票代码列表，为空则采集所有股票")
    force_update: bool = Field(default=False, description="是否强制更新")
    batch_size: int = Field(default=100, ge=1, le=1000, description="批处理大小")
    quality_check: bool = Field(default=True, description="是否进行质量检查")


# ============= 响应模型 =============


class CollectionTaskResponse(BaseModel):
    """数据采集任务响应"""

    task_id: int = Field(..., description="任务ID")
    task_type: TaskType = Field(..., description="任务类型")
    status: str = Field(..., description="任务状态")
    message: str = Field(..., description="响应消息")
    total_records: int = Field(default=0, description="总记录数")
    processed_records: int = Field(default=0, description="已处理记录数")
    valid_records: int = Field(default=0, description="有效记录数")
    invalid_records: int = Field(default=0, description="无效记录数")
    quality_score: float = Field(default=0.0, description="数据质量分数")
    execution_time: float = Field(default=0.0, description="执行时间(秒)")
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")


# ============= 依赖注入 =============


async def get_data_collection_orchestrator(
    session: AsyncSession = Depends(get_session),
) -> DataCollectionOrchestrator:
    """获取数据采集编排器实例"""
    from repositories.stock_repo import StockRepository
    from repositories.task_repo import TaskRepository
    from services.collection_service import CollectionService
    from services.quality_service import QualityService

    # 创建依赖实例
    stock_repo = StockRepository(session)
    task_repo = TaskRepository(session)
    collection_service = CollectionService(stock_repo)
    quality_service = QualityService()

    # 创建编排器
    orchestrator = DataCollectionOrchestrator(
        collection_service=collection_service,
        quality_service=quality_service,
        stock_repo=stock_repo,
        task_repo=task_repo,
    )

    return orchestrator


# ============= API接口 =============


@router.post(
    "/full",
    response_model=CollectionTaskResponse,
    summary="全量数据采集",
    description="触发全量数据采集任务，采集指定类型的所有数据",
)
async def trigger_full_collection(
    request: FullCollectionRequest,
    orchestrator: DataCollectionOrchestrator = Depends(get_data_collection_orchestrator),
) -> CollectionTaskResponse:
    """全量数据采集接口

    Args:
        request: 全量采集请求参数
        orchestrator: 数据采集编排器

    Returns:
        采集任务响应

    Raises:
        HTTPException: 采集失败时抛出异常
    """
    try:
        logger.info(f"开始全量数据采集, 任务类型: {request.task_type}")

        # 构建数据采集请求
        collection_request = DataCollectionRequest(
            task_type=request.task_type,
            force_update=request.force_update,
            batch_size=request.batch_size,
            quality_check=request.quality_check,
        )

        # 执行数据采集
        result = await orchestrator.execute(collection_request)

        # 构建响应
        response = CollectionTaskResponse(
            task_id=result.task_id,
            task_type=result.task_type,
            status=result.status.value,
            message=f"全量{request.task_type.value}采集任务已启动",
            total_records=result.total_records,
            processed_records=result.processed_records,
            valid_records=result.valid_records,
            invalid_records=result.invalid_records,
            quality_score=result.quality_score,
            execution_time=result.execution_time,
        )

        logger.info(f"全量数据采集完成, 任务ID: {result.task_id}")
        return response

    except (DataCollectionError, OrchestrationError) as e:
        logger.error(f"全量数据采集失败: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"全量数据采集失败: {e}",
        ) from e

    except Exception as e:
        logger.error(f"全量数据采集异常: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="全量数据采集服务异常",
        ) from e


@router.post(
    "/incremental",
    response_model=CollectionTaskResponse,
    summary="增量数据采集",
    description="触发增量数据采集任务，采集最近几天的数据",
)
async def trigger_incremental_collection(
    request: IncrementalCollectionRequest,
    orchestrator: DataCollectionOrchestrator = Depends(get_data_collection_orchestrator),
) -> CollectionTaskResponse:
    """增量数据采集接口

    Args:
        request: 增量采集请求参数
        orchestrator: 数据采集编排器

    Returns:
        采集任务响应

    Raises:
        HTTPException: 采集失败时抛出异常
    """
    try:
        logger.info(f"开始增量数据采集, 任务类型: {request.task_type}, 回溯天数: {request.days_back}")

        # 计算目标日期（回溯指定天数）
        from datetime import timedelta

        target_date = date.today() - timedelta(days=request.days_back)

        # 构建数据采集请求
        collection_request = DataCollectionRequest(
            task_type=request.task_type,
            target_date=target_date,
            force_update=request.force_update,
            batch_size=request.batch_size,
            quality_check=request.quality_check,
        )

        # 执行数据采集
        result = await orchestrator.execute(collection_request)

        # 构建响应
        response = CollectionTaskResponse(
            task_id=result.task_id,
            task_type=result.task_type,
            status=result.status.value,
            message=f"增量{request.task_type.value}采集任务已启动，回溯{request.days_back}天",
            total_records=result.total_records,
            processed_records=result.processed_records,
            valid_records=result.valid_records,
            invalid_records=result.invalid_records,
            quality_score=result.quality_score,
            execution_time=result.execution_time,
        )

        logger.info(f"增量数据采集完成, 任务ID: {result.task_id}")
        return response

    except (DataCollectionError, OrchestrationError) as e:
        logger.error(f"增量数据采集失败: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"增量数据采集失败: {e}",
        ) from e
    except Exception as e:
        logger.error(f"增量数据采集异常: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="增量数据采集服务异常",
        ) from e


@router.post(
    "/stocks",
    response_model=CollectionTaskResponse,
    summary="指定股票数据采集",
    description="触发指定股票的数据采集任务",
)
async def trigger_stock_collection(
    request: StockCollectionRequest,
    orchestrator: DataCollectionOrchestrator = Depends(get_data_collection_orchestrator),
) -> CollectionTaskResponse:
    """指定股票数据采集接口

    Args:
        request: 股票采集请求参数
        orchestrator: 数据采集编排器

    Returns:
        采集任务响应

    Raises:
        HTTPException: 采集失败时抛出异常
    """
    try:
        logger.info(
            f"开始指定股票数据采集, 任务类型: {request.task_type}, 股票数量: {len(request.symbols)}"
        )

        # 验证股票代码格式
        for symbol in request.symbols:
            if not symbol or len(symbol.strip()) == 0:
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail="股票代码不能为空",
                )

        # 构建数据采集请求
        collection_request = DataCollectionRequest(
            task_type=request.task_type,
            target_date=request.target_date,
            symbols=request.symbols,
            force_update=request.force_update,
            batch_size=request.batch_size,
            quality_check=request.quality_check,
        )

        # 执行数据采集
        result = await orchestrator.execute(collection_request)

        # 构建响应
        response = CollectionTaskResponse(
            task_id=result.task_id,
            task_type=result.task_type,
            status=result.status.value,
            message=f"指定股票{request.task_type.value}采集任务已启动，股票数量: {len(request.symbols)}",
            total_records=result.total_records,
            processed_records=result.processed_records,
            valid_records=result.valid_records,
            invalid_records=result.invalid_records,
            quality_score=result.quality_score,
            execution_time=result.execution_time,
        )

        logger.info(f"指定股票数据采集完成, 任务ID: {result.task_id}")
        return response

    except HTTPException:
        raise
    except (DataCollectionError, OrchestrationError) as e:
        logger.error(f"指定股票数据采集失败: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"指定股票数据采集失败: {e}",
        ) from e
    except Exception as e:
        logger.error(f"指定股票数据采集异常: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="指定股票数据采集服务异常",
        ) from e


@router.post(
    "/range",
    response_model=CollectionTaskResponse,
    summary="日期范围数据采集",
    description="触发指定日期范围的数据采集任务",
)
async def trigger_range_collection(
    request: RangeCollectionRequest,
    orchestrator: DataCollectionOrchestrator = Depends(get_data_collection_orchestrator),
) -> CollectionTaskResponse:
    """日期范围数据采集接口

    Args:
        request: 范围采集请求参数
        orchestrator: 数据采集编排器

    Returns:
        采集任务响应

    Raises:
        HTTPException: 采集失败时抛出异常
    """
    try:
        logger.info(
            f"开始日期范围数据采集, 任务类型: {request.task_type}, "
            f"日期范围: {request.start_date} - {request.end_date}"
        )

        # 验证日期范围
        if request.start_date > request.end_date:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="开始日期不能晚于结束日期",
            )

        # 验证日期范围不能超过1年
        if (request.end_date - request.start_date).days > 365:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="日期范围不能超过1年",
            )

        # 验证股票代码（如果提供）
        if request.symbols:
            for symbol in request.symbols:
                if not symbol or len(symbol.strip()) == 0:
                    raise HTTPException(
                        status_code=status.HTTP_400_BAD_REQUEST,
                        detail="股票代码不能为空",
                    )

        # 构建数据采集请求
        collection_request = DataCollectionRequest(
            task_type=request.task_type,
            target_date=request.start_date,  # 使用开始日期作为目标日期
            symbols=request.symbols,
            force_update=request.force_update,
            batch_size=request.batch_size,
            quality_check=request.quality_check,
        )

        # 执行数据采集
        result = await orchestrator.execute(collection_request)

        # 构建响应
        symbols_info = f"，股票数量: {len(request.symbols)}" if request.symbols else "，全部股票"
        response = CollectionTaskResponse(
            task_id=result.task_id,
            task_type=result.task_type,
            status=result.status.value,
            message=(
                f"日期范围{request.task_type.value}采集任务已启动，"
                f"日期范围: {request.start_date} - {request.end_date}{symbols_info}"
            ),
            total_records=result.total_records,
            processed_records=result.processed_records,
            valid_records=result.valid_records,
            invalid_records=result.invalid_records,
            quality_score=result.quality_score,
            execution_time=result.execution_time,
        )

        logger.info(f"日期范围数据采集完成, 任务ID: {result.task_id}")
        return response

    except HTTPException:
        raise
    except (DataCollectionError, OrchestrationError) as e:
        logger.error(f"日期范围数据采集失败: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"日期范围数据采集失败: {e}",
        ) from e
    except Exception as e:
        logger.error(f"日期范围数据采集异常: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="日期范围数据采集服务异常",
        ) from e


# ==================== 任务管理接口 ====================

class TaskStatusResponse(BaseModel):
    """任务状态响应模型"""

    task_id: int = Field(description="任务ID")
    name: str = Field(description="任务名称")
    task_type: str = Field(description="任务类型")
    status: str = Field(description="任务状态")
    priority: int = Field(description="优先级")
    params: dict | None = Field(default=None, description="任务参数")
    result: dict | None = Field(default=None, description="任务结果")
    error_message: str | None = Field(default=None, description="错误信息")
    retry_count: int = Field(description="重试次数")
    max_retries: int = Field(description="最大重试次数")
    scheduled_at: datetime | None = Field(default=None, description="计划执行时间")
    started_at: datetime | None = Field(default=None, description="开始执行时间")
    completed_at: datetime | None = Field(default=None, description="完成时间")
    execution_time: float | None = Field(default=None, description="执行时间(秒)")
    created_by: str | None = Field(default=None, description="创建者")
    created_at: datetime = Field(description="创建时间")
    updated_at: datetime = Field(description="更新时间")


class TaskListResponse(BaseModel):
    """任务列表响应模型"""

    tasks: list[TaskStatusResponse] = Field(description="任务列表")
    total: int = Field(description="总数量")
    page: int = Field(description="当前页码")
    page_size: int = Field(description="每页大小")


class TaskCancelResponse(BaseModel):
    """任务取消响应模型"""

    task_id: int = Field(description="任务ID")
    success: bool = Field(description="是否取消成功")
    message: str = Field(description="响应消息")


class TaskStatisticsResponse(BaseModel):
    """任务统计响应模型"""

    status_counts: dict[str, int] = Field(description="各状态任务数量")
    type_counts: dict[str, int] = Field(description="各类型任务数量")
    total_tasks: int = Field(description="总任务数量")


@router.get(
    "/tasks/{task_id}",
    response_model=TaskStatusResponse,
    summary="查询任务状态",
    description="根据任务ID查询任务的详细状态信息",
)
async def get_task_status(task_id: int) -> TaskStatusResponse:
    """查询任务状态

    Args:
        task_id: 任务ID

    Returns:
        任务状态信息

    Raises:
        HTTPException: 任务不存在或查询失败
    """
    try:
        # 查询任务
        task = task_repo.get_by_id(task_id)
        if not task:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"任务不存在: ID={task_id}",
            )

        # 构建响应
        response = TaskStatusResponse(
            task_id=task.id,
            name=task.name,
            task_type=task.task_type.value,
            status=task.status.value,
            priority=task.priority,
            params=task.params,
            result=task.result,
            error_message=task.error_message,
            retry_count=task.retry_count,
            max_retries=task.max_retries,
            scheduled_at=task.scheduled_at,
            started_at=task.started_at,
            completed_at=task.completed_at,
            execution_time=float(task.execution_time) if task.execution_time else None,
            created_by=task.created_by,
            created_at=task.created_at,
            updated_at=task.updated_at,
        )

        logger.info(f"查询任务状态成功: ID={task_id}")
        return response

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"查询任务状态失败: ID={task_id}, 错误: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="查询任务状态失败",
        ) from e


@router.get(
    "/tasks",
    response_model=TaskListResponse,
    summary="获取任务列表",
    description="获取任务列表，支持按状态、类型过滤和分页",
)
async def get_task_list(
    page: int = Query(1, ge=1, description="页码"),
    page_size: int = Query(20, ge=1, le=100, description="每页大小"),
    task_type: TaskType | None = Query(None, description="任务类型过滤"),
    status: TaskStatus | None = Query(None, description="状态过滤"),
) -> TaskListResponse:
    """获取任务列表

    Args:
        page: 页码
        page_size: 每页大小
        task_type: 可选的任务类型过滤
        status: 可选的状态过滤

    Returns:
        任务列表
    """
    try:
        # 计算偏移量
        limit = page_size

        # 查询任务列表
        tasks = task_repo.get_recent_tasks(
            limit=limit,
            task_type=task_type,
            status=status,
        )

        # 转换为响应模型
        task_responses = []
        for task in tasks:
            task_response = TaskStatusResponse(
                task_id=task.id,
                name=task.name,
                task_type=task.task_type.value,
                status=task.status.value,
                priority=task.priority,
                params=task.params,
                result=task.result,
                error_message=task.error_message,
                retry_count=task.retry_count,
                max_retries=task.max_retries,
                scheduled_at=task.scheduled_at,
                started_at=task.started_at,
                completed_at=task.completed_at,
                execution_time=float(task.execution_time) if task.execution_time else None,
                created_by=task.created_by,
                created_at=task.created_at,
                updated_at=task.updated_at,
            )
            task_responses.append(task_response)

        # 构建响应
        response = TaskListResponse(
            tasks=task_responses,
            total=len(task_responses),
            page=page,
            page_size=page_size,
        )

        logger.info(f"获取任务列表成功: 数量={len(task_responses)}")
        return response

    except Exception as e:
        logger.error(f"获取任务列表失败: 错误: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="获取任务列表失败",
        ) from e


@router.delete(
    "/tasks/{task_id}",
    response_model=TaskCancelResponse,
    summary="取消任务",
    description="取消指定的任务（仅限待执行和运行中的任务）",
)
async def cancel_task(task_id: int) -> TaskCancelResponse:
    """取消任务

    Args:
        task_id: 任务ID

    Returns:
        取消结果

    Raises:
        HTTPException: 任务不存在或取消失败
    """
    try:
        # 取消任务
        success = task_repo.cancel_task(task_id)

        if success:
            response = TaskCancelResponse(
                task_id=task_id,
                success=True,
                message="任务取消成功",
            )
            logger.info(f"任务取消成功: ID={task_id}")
        else:
            response = TaskCancelResponse(
                task_id=task_id,
                success=False,
                message="任务取消失败，可能任务不存在或状态不允许取消",
            )
            logger.warning(f"任务取消失败: ID={task_id}")

        return response

    except Exception as e:
        logger.error(f"取消任务异常: ID={task_id}, 错误: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="取消任务失败",
        ) from e


@router.get(
    "/tasks/statistics",
    response_model=TaskStatisticsResponse,
    summary="获取任务统计",
    description="获取任务的统计信息，包括各状态和类型的任务数量",
)
async def get_task_statistics() -> TaskStatisticsResponse:
    """获取任务统计信息

    Returns:
        任务统计数据
    """
    try:
        # 获取统计数据
        statistics = task_repo.get_task_statistics()

        response = TaskStatisticsResponse(
            status_counts=statistics["status_counts"],
            type_counts=statistics["type_counts"],
            total_tasks=statistics["total_tasks"],
        )

        logger.info("获取任务统计成功")
        return response

    except Exception as e:
        logger.error(f"获取任务统计失败: 错误: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="获取任务统计失败",
        ) from e
