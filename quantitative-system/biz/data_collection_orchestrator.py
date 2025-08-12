"""数据采集编排器模块

协调数据采集的完整流程：定时调度器→数据采集服务→Tushare客户端→数据质量服务→数据库存储
实现采集任务的状态管理、错误处理、进度监控和日志记录。
"""

from datetime import date, datetime, timedelta
from typing import Any

from pydantic import BaseModel, Field

from biz.base_orchestrator import BaseOrchestrator, OrchestrationContext
from models.enums import TaskStatus, TaskType
from repositories.stock_repo import StockRepository
from repositories.task_repo import TaskRepository
from services.collection_service import CollectionService
from services.quality_service import QualityService
from utils.exceptions import DataCollectionError, OrchestrationError
from utils.logger import get_logger

logger = get_logger(__name__)


class DataCollectionRequest(BaseModel):
    """数据采集请求模型"""

    task_type: TaskType = Field(description="采集任务类型")
    target_date: date | None = Field(default=None, description="目标日期")
    symbols: list[str] | None = Field(default=None, description="指定股票代码列表")
    force_update: bool = Field(default=False, description="是否强制更新")
    batch_size: int = Field(default=100, description="批处理大小")
    max_retries: int = Field(default=3, description="最大重试次数")
    quality_check: bool = Field(default=True, description="是否进行质量检查")


class DataCollectionResponse(BaseModel):
    """数据采集响应模型"""

    task_id: int
    task_type: TaskType
    status: TaskStatus
    total_records: int = 0
    processed_records: int = 0
    valid_records: int = 0
    invalid_records: int = 0
    quality_score: float = 0.0
    start_time: datetime
    end_time: datetime | None = None
    execution_time: float = 0.0
    error_message: str | None = None
    progress_details: dict[str, Any] = Field(default_factory=dict)


class DataCollectionOrchestrator(BaseOrchestrator):
    """数据采集编排器

    协调数据采集的完整流程，包括：
    1. 任务创建和状态管理
    2. 数据采集服务调用
    3. 数据质量验证
    4. 数据库存储
    5. 进度监控和错误处理
    """

    def __init__(
        self,
        collection_service: CollectionService,
        quality_service: QualityService,
        stock_repo: StockRepository,
        task_repo: TaskRepository,
    ):
        """初始化数据采集编排器

        Args:
            collection_service: 数据采集服务
            quality_service: 数据质量服务
            stock_repo: 股票数据仓库
            task_repo: 任务数据仓库
        """
        super().__init__()
        self.collection_service = collection_service
        self.quality_service = quality_service
        self.stock_repo = stock_repo
        self.task_repo = task_repo
        logger.info("数据采集编排器初始化完成")

    async def _pre_check(
        self, request: DataCollectionRequest, context: OrchestrationContext
    ) -> None:
        """前置检查

        Args:
            request: 数据采集请求
            context: 编排上下文

        Raises:
            OrchestrationError: 前置检查失败
        """
        logger.info(
            f"开始前置检查, 任务类型: {request.task_type}, request_id: {context.request_id}"
        )

        # 验证请求参数
        self._validate_request(request, context)

        # 检查任务类型支持
        supported_types = [
            TaskType.STOCK_BASIC_COLLECTION,
            TaskType.DAILY_DATA_COLLECTION,
            TaskType.FINANCIAL_DATA_COLLECTION,
        ]
        if request.task_type not in supported_types:
            raise OrchestrationError(f"不支持的任务类型: {request.task_type}")

        # 检查目标日期合理性
        if request.target_date and request.target_date > date.today():
            raise OrchestrationError(f"目标日期不能是未来日期: {request.target_date}")

        # 检查是否有正在执行的同类型任务
        running_tasks = await self.task_repo.get_running_tasks_by_type(
            request.task_type
        )
        if running_tasks and not request.force_update:
            raise OrchestrationError(
                f"已有正在执行的{request.task_type}任务, 请等待完成或使用force_update参数"
            )

        # 设置上下文数据
        self._set_context_data("validated_request", request, context)
        self._set_context_data("pre_check_time", datetime.now(), context)

        logger.info(f"前置检查完成, request_id: {context.request_id}")

    async def _call_services(
        self, request: DataCollectionRequest, context: OrchestrationContext
    ) -> dict[str, Any]:
        """调用服务

        Args:
            request: 数据采集请求
            context: 编排上下文

        Returns:
            服务调用结果字典

        Raises:
            OrchestrationError: 服务调用失败
        """
        logger.info(
            f"开始服务调用, 任务类型: {request.task_type}, request_id: {context.request_id}"
        )

        results = {}

        try:
            # 1. 创建采集任务
            task_result = await self._safe_service_call(
                "task_creation",
                lambda: self._create_collection_task(request, context),
                context,
            )
            results["task"] = task_result

            # 2. 执行数据采集
            collection_result = await self._safe_service_call(
                "data_collection",
                lambda: self._execute_data_collection(
                    request, task_result["task_id"], context
                ),
                context,
            )
            results["collection"] = collection_result

            # 3. 数据质量检查（如果启用）
            if request.quality_check and collection_result.get("collected_data"):
                quality_result = await self._safe_service_call(
                    "quality_check",
                    lambda: self._perform_quality_check(
                        collection_result["collected_data"], request.task_type, context
                    ),
                    context,
                )
                results["quality"] = quality_result

            # 4. 更新任务状态
            await self._safe_service_call(
                "task_update",
                lambda: self._update_task_status(
                    task_result["task_id"], results, context
                ),
                context,
            )

            logger.info(f"服务调用完成, request_id: {context.request_id}")
            return results

        except Exception as e:
            # 如果有任务ID，标记任务失败
            if "task" in results and "task_id" in results["task"]:
                await self._mark_task_failed(
                    results["task"]["task_id"], str(e), context
                )
            raise

    async def _aggregate_results(
        self, service_results: dict[str, Any], context: OrchestrationContext
    ) -> DataCollectionResponse:
        """聚合结果

        Args:
            service_results: 服务调用结果
            context: 编排上下文

        Returns:
            聚合后的数据采集响应

        Raises:
            OrchestrationError: 结果聚合失败
        """
        logger.info(f"开始结果聚合, request_id: {context.request_id}")

        try:
            task_result = service_results.get("task", {})
            collection_result = service_results.get("collection", {})
            quality_result = service_results.get("quality", {})

            # 计算执行时间
            start_time = self._get_context_data(
                "pre_check_time", context, datetime.now()
            )
            end_time = datetime.now()
            execution_time = (end_time - start_time).total_seconds()

            # 构建响应
            response = DataCollectionResponse(
                task_id=task_result.get("task_id", 0),
                task_type=task_result.get("task_type", TaskType.STOCK_BASIC_COLLECTION),
                status=task_result.get("status", TaskStatus.COMPLETED),
                total_records=collection_result.get("total_records", 0),
                processed_records=collection_result.get("processed_records", 0),
                valid_records=quality_result.get(
                    "valid_records", collection_result.get("processed_records", 0)
                ),
                invalid_records=quality_result.get("invalid_records", 0),
                quality_score=quality_result.get("quality_score", 1.0),
                start_time=start_time,
                end_time=end_time,
                execution_time=execution_time,
                progress_details={
                    "collection_details": collection_result,
                    "quality_details": quality_result,
                    "context_summary": {
                        "completed_steps": context.intermediate_results.get(
                            "completed_steps", []
                        ),
                        "failed_steps": context.intermediate_results.get(
                            "failed_steps", []
                        ),
                    },
                },
            )

            logger.info(
                f"结果聚合完成, 任务ID: {response.task_id}, 质量分数: {response.quality_score}, request_id: {context.request_id}"
            )
            return response

        except Exception as e:
            error_msg = f"结果聚合失败: {e!s}"
            logger.error(f"{error_msg}, request_id: {context.request_id}")
            raise OrchestrationError(error_msg) from e

    async def _create_collection_task(
        self, request: DataCollectionRequest, context: OrchestrationContext
    ) -> dict[str, Any]:
        """创建采集任务

        Args:
            request: 数据采集请求
            context: 编排上下文

        Returns:
            任务创建结果
        """
        logger.info(
            f"创建采集任务, 类型: {request.task_type}, request_id: {context.request_id}"
        )

        task_name = (
            f"{request.task_type.value}_采集_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
        )

        task_params = {
            "target_date": request.target_date.isoformat()
            if request.target_date
            else None,
            "symbols": request.symbols,
            "force_update": request.force_update,
            "batch_size": request.batch_size,
            "max_retries": request.max_retries,
            "quality_check": request.quality_check,
        }

        task = await self.task_repo.create_task(
            name=task_name,
            task_type=request.task_type,
            params=task_params,
            status=TaskStatus.RUNNING,
        )

        # 添加回滚操作
        self._add_rollback_action(
            "delete_data", {"task_id": task.id, "table": "tasks"}, context
        )

        logger.info(
            f"任务创建成功, 任务ID: {task.id}, request_id: {context.request_id}"
        )

        return {
            "task_id": task.id,
            "task_type": request.task_type,
            "status": TaskStatus.RUNNING,
            "created_at": task.created_at,
        }

    async def _execute_data_collection(
        self,
        request: DataCollectionRequest,
        task_id: int,
        context: OrchestrationContext,
    ) -> dict[str, Any]:
        """执行数据采集

        Args:
            request: 数据采集请求
            task_id: 任务ID
            context: 编排上下文

        Returns:
            数据采集结果
        """
        logger.info(
            f"开始数据采集, 任务类型: {request.task_type}, 任务ID: {task_id}, request_id: {context.request_id}"
        )

        collected_data = []
        total_records = 0
        processed_records = 0

        try:
            if request.task_type == TaskType.STOCK_BASIC_COLLECTION:
                # 采集股票基础信息
                processed_records = (
                    await self.collection_service.collect_stock_basic_info(
                        force_update=request.force_update
                    )
                )
                # 获取采集的数据用于质量检查
                if request.quality_check:
                    collected_data = await self.stock_repo.get_recent_stock_basic_info(
                        limit=processed_records
                    )

            elif request.task_type == TaskType.DAILY_DATA_COLLECTION:
                # 采集日线数据
                target_date = request.target_date or date.today() - timedelta(days=1)
                processed_records = await self.collection_service.collect_daily_data(
                    trade_date=target_date,
                    symbols=request.symbols,
                    force_update=request.force_update,
                )
                # 获取采集的数据用于质量检查
                if request.quality_check:
                    collected_data = await self.stock_repo.get_daily_data_by_date(
                        target_date, limit=processed_records
                    )

            elif request.task_type == TaskType.FINANCIAL_DATA_COLLECTION:
                # 采集财务数据
                processed_records = (
                    await self.collection_service.collect_financial_data(
                        symbols=request.symbols, force_update=request.force_update
                    )
                )
                # 获取采集的数据用于质量检查
                if request.quality_check:
                    collected_data = await self.stock_repo.get_recent_financial_data(
                        limit=processed_records
                    )

            total_records = processed_records

            # 更新任务进度
            await self.task_repo.update_task_progress(
                task_id, processed_records, total_records
            )

            logger.info(
                f"数据采集完成, 处理记录数: {processed_records}, 任务ID: {task_id}, request_id: {context.request_id}"
            )

            return {
                "collected_data": collected_data,
                "total_records": total_records,
                "processed_records": processed_records,
                "collection_time": datetime.now(),
            }

        except Exception as e:
            error_msg = f"数据采集失败: {e!s}"
            logger.error(
                f"{error_msg}, 任务ID: {task_id}, request_id: {context.request_id}"
            )
            raise DataCollectionError(error_msg) from e

    async def _perform_quality_check(
        self,
        collected_data: list[dict],
        data_type: TaskType,
        context: OrchestrationContext,
    ) -> dict[str, Any]:
        """执行数据质量检查

        Args:
            collected_data: 采集的数据
            data_type: 数据类型
            context: 编排上下文

        Returns:
            质量检查结果
        """
        logger.info(
            f"开始数据质量检查, 数据类型: {data_type}, 数据量: {len(collected_data)}, request_id: {context.request_id}"
        )

        if not collected_data:
            logger.warning(f"没有数据需要质量检查, request_id: {context.request_id}")
            return {
                "valid_records": 0,
                "invalid_records": 0,
                "quality_score": 1.0,
                "quality_details": {},
            }

        try:
            # 根据任务类型确定数据类型
            quality_data_type = {
                TaskType.STOCK_BASIC_COLLECTION: "stock_basic",
                TaskType.DAILY_DATA_COLLECTION: "daily_data",
                TaskType.FINANCIAL_DATA_COLLECTION: "financial_data",
            }.get(data_type, "unknown")

            # 执行质量检查
            quality_report = self.quality_service.get_data_quality_report(
                collected_data, quality_data_type
            )

            logger.info(
                f"数据质量检查完成, 质量分数: {quality_report.get('quality_score', 0)}, request_id: {context.request_id}"
            )

            return {
                "valid_records": quality_report.get("valid_records", 0),
                "invalid_records": quality_report.get("invalid_records", 0),
                "quality_score": quality_report.get("quality_score", 0),
                "quality_details": quality_report,
            }

        except Exception as e:
            error_msg = f"数据质量检查失败: {e!s}"
            logger.error(f"{error_msg}, request_id: {context.request_id}")
            # 质量检查失败不应该阻止整个流程，返回默认值
            return {
                "valid_records": len(collected_data),
                "invalid_records": 0,
                "quality_score": 0.5,  # 未知质量
                "quality_details": {"error": error_msg},
            }

    async def _update_task_status(
        self, task_id: int, results: dict[str, Any], context: OrchestrationContext
    ) -> None:
        """更新任务状态

        Args:
            task_id: 任务ID
            results: 执行结果
            context: 编排上下文
        """
        logger.info(
            f"更新任务状态, 任务ID: {task_id}, request_id: {context.request_id}"
        )

        try:
            collection_result = results.get("collection", {})
            quality_result = results.get("quality", {})

            # 准备结果数据
            result_data = {
                "total_records": collection_result.get("total_records", 0),
                "processed_records": collection_result.get("processed_records", 0),
                "valid_records": quality_result.get("valid_records", 0),
                "invalid_records": quality_result.get("invalid_records", 0),
                "quality_score": quality_result.get("quality_score", 0),
                "collection_time": collection_result.get(
                    "collection_time", datetime.now()
                ).isoformat(),
                "quality_details": quality_result.get("quality_details", {}),
            }

            # 更新任务为完成状态
            await self.task_repo.update_task_status(
                task_id=task_id, status=TaskStatus.COMPLETED, result=result_data
            )

            logger.info(
                f"任务状态更新完成, 任务ID: {task_id}, request_id: {context.request_id}"
            )

        except Exception as e:
            error_msg = f"更新任务状态失败: {e!s}"
            logger.error(
                f"{error_msg}, 任务ID: {task_id}, request_id: {context.request_id}"
            )
            # 尝试标记任务失败
            await self._mark_task_failed(task_id, error_msg, context)
            raise

    async def _mark_task_failed(
        self, task_id: int, error_message: str, context: OrchestrationContext
    ) -> None:
        """标记任务失败

        Args:
            task_id: 任务ID
            error_message: 错误信息
            context: 编排上下文
        """
        try:
            await self.task_repo.update_task_status(
                task_id=task_id, status=TaskStatus.FAILED, error_message=error_message
            )
            logger.info(
                f"任务已标记为失败, 任务ID: {task_id}, 错误: {error_message}, request_id: {context.request_id}"
            )
        except Exception as e:
            logger.error(
                f"标记任务失败时出错, 任务ID: {task_id}, 错误: {e!s}, request_id: {context.request_id}"
            )

    async def get_collection_progress(self, task_id: int) -> dict[str, Any]:
        """获取采集进度

        Args:
            task_id: 任务ID

        Returns:
            进度信息
        """
        try:
            task = await self.task_repo.get_task_by_id(task_id)
            if not task:
                raise OrchestrationError(f"任务不存在: {task_id}")

            progress = {
                "task_id": task.id,
                "task_name": task.name,
                "task_type": task.task_type,
                "status": task.status,
                "progress": task.progress,
                "total_count": task.total_count,
                "created_at": task.created_at,
                "updated_at": task.updated_at,
                "result": task.result,
                "error_message": task.error_message,
            }

            return progress

        except Exception as e:
            error_msg = f"获取采集进度失败: {e!s}"
            logger.error(error_msg)
            raise OrchestrationError(error_msg) from e

    async def cancel_collection_task(self, task_id: int) -> bool:
        """取消采集任务

        Args:
            task_id: 任务ID

        Returns:
            是否取消成功
        """
        try:
            task = await self.task_repo.get_task_by_id(task_id)
            if not task:
                raise OrchestrationError(f"任务不存在: {task_id}")

            if task.status not in [TaskStatus.PENDING, TaskStatus.RUNNING]:
                raise OrchestrationError(f"任务状态不允许取消: {task.status}")

            await self.task_repo.update_task_status(
                task_id=task_id,
                status=TaskStatus.CANCELLED,
                error_message="用户取消任务",
            )

            logger.info(f"任务已取消, 任务ID: {task_id}")
            return True

        except Exception as e:
            error_msg = f"取消采集任务失败: {e!s}"
            logger.error(error_msg)
            raise OrchestrationError(error_msg) from e
