"""数据采集服务模块

提供统一的数据采集接口，集成股票基础信息、行情数据、财务数据采集功能。
支持历史数据全量采集和增量更新，包含数据预处理和格式标准化。
支持手动触发的采集任务，包括任务队列管理、优先级控制、失败重试和错误处理机制。
"""

import asyncio
from datetime import date, datetime, timedelta
from typing import Any

from loguru import logger

from clients.tushare_client import get_tushare_client
from models.database import StockBasicInfo, Task
from models.enums import TaskStatus, TaskType
from repositories.stock_repo import StockRepository
from repositories.task_repo import TaskRepository
from services.quality_service import QualityService
from utils.exceptions import DataCollectionError, ValidationError


class CollectionService:
    """数据采集服务

    提供统一的数据采集接口，支持股票基础信息、行情数据、财务数据的采集。
    包含数据预处理、格式标准化、完整性验证和去重逻辑。
    """

    def __init__(
        self,
        stock_repo: StockRepository,
        quality_service: QualityService,
        task_repo: TaskRepository,
    ):
        """初始化数据采集服务

        Args:
            stock_repo: 股票数据仓库
            quality_service: 数据质量服务
            task_repo: 任务数据仓库
        """
        self.tushare_client = get_tushare_client()
        self.stock_repo = stock_repo
        self.quality_service = quality_service
        self.task_repo = task_repo
        logger.info("数据采集服务初始化完成")

    async def collect_stock_basic_info(
        self,
        list_status: str = "L",
        exchange: str | None = None,
        force_update: bool = False,
    ) -> int:
        """采集股票基础信息

        Args:
            list_status: 上市状态 L上市 D退市 P暂停上市
            exchange: 交易所 SSE上交所 SZSE深交所
            force_update: 是否强制更新已存在的数据

        Returns:
            采集到的股票数量

        Raises:
            DataCollectionError: 数据采集失败时
        """
        try:
            logger.info(
                f"开始采集股票基础信息, 状态: {list_status}, 交易所: {exchange}"
            )

            # 从Tushare获取数据
            raw_stocks = self.tushare_client.get_stock_basic(list_status, exchange)
            if not raw_stocks:
                logger.warning("未获取到股票基础信息")
                return 0

            # 数据质量验证
            validated_stocks = []
            for stock_data in raw_stocks:
                try:
                    # 验证数据完整性
                    self.quality_service.validate_stock_basic_info(stock_data)
                    validated_stocks.append(stock_data)
                except ValidationError as e:
                    logger.warning(
                        f"股票基础信息验证失败: {stock_data.get('ts_code')}, 错误: {e}"
                    )
                    continue

            # 去重和更新逻辑
            new_count = 0
            updated_count = 0

            for stock_data in validated_stocks:
                ts_code = stock_data.ts_code
                existing_stock = await self.stock_repo.get_stock_basic_info(ts_code)

                if existing_stock and not force_update:
                    # 检查是否需要更新
                    if self._should_update_stock_basic(existing_stock, stock_data):
                        await self.stock_repo.update_stock_basic_info(
                            ts_code, stock_data
                        )
                        updated_count += 1
                        logger.debug(f"更新股票基础信息: {ts_code}")
                elif not existing_stock or force_update:
                    await self.stock_repo.create_stock_basic_info(stock_data)
                    new_count += 1
                    logger.debug(f"新增股票基础信息: {ts_code}")

            total_count = new_count + updated_count
            logger.info(
                f"股票基础信息采集完成, 新增: {new_count}, 更新: {updated_count}, 总计: {total_count}"
            )
            return total_count

        except Exception as e:
            error_msg = f"采集股票基础信息失败: {e!s}"
            logger.error(error_msg)
            raise DataCollectionError(error_msg) from e

    async def collect_daily_data(
        self,
        ts_code: str | None = None,
        start_date: str | date | None = None,
        end_date: str | date | None = None,
        is_incremental: bool = True,
    ) -> int:
        """采集股票日线数据

        Args:
            ts_code: 股票代码，为None时采集所有股票
            start_date: 开始日期
            end_date: 结束日期
            is_incremental: 是否增量更新

        Returns:
            采集到的数据条数

        Raises:
            DataCollectionError: 数据采集失败时
        """
        try:
            # 确定采集范围
            if ts_code:
                stock_codes = [ts_code]
            else:
                # 获取所有上市股票代码
                stocks = await self.stock_repo.get_all_stock_codes()
                stock_codes = [stock.ts_code for stock in stocks]

            if not stock_codes:
                logger.warning("未找到需要采集的股票代码")
                return 0

            # 确定日期范围
            if is_incremental and not start_date:
                # 增量更新：从最后更新日期开始
                last_date = await self.stock_repo.get_last_daily_data_date(ts_code)
                start_date = (
                    last_date + timedelta(days=1)
                    if last_date
                    else date.today() - timedelta(days=30)
                )

            if not end_date:
                end_date = date.today()

            logger.info(
                f"开始采集日线数据, 股票数量: {len(stock_codes)}, 日期范围: {start_date} ~ {end_date}"
            )

            total_count = 0
            for i, code in enumerate(stock_codes, 1):
                try:
                    count = await self._collect_single_stock_daily_data(
                        code, start_date, end_date
                    )
                    total_count += count
                    logger.debug(
                        f"进度: {i}/{len(stock_codes)}, 股票: {code}, 采集: {count} 条"
                    )
                except Exception as e:
                    logger.error(f"采集股票 {code} 日线数据失败: {e}")
                    continue

            logger.info(f"日线数据采集完成, 总计: {total_count} 条")
            return total_count

        except Exception as e:
            error_msg = f"采集日线数据失败: {e!s}"
            logger.error(error_msg)
            raise DataCollectionError(error_msg) from e

    async def collect_financial_data(
        self,
        ts_code: str | None = None,
        start_date: str | date | None = None,
        end_date: str | date | None = None,
        period: str | None = None,
        is_incremental: bool = True,
    ) -> int:
        """采集财务数据

        Args:
            ts_code: 股票代码，为None时采集所有股票
            start_date: 开始日期
            end_date: 结束日期
            period: 报告期
            is_incremental: 是否增量更新

        Returns:
            采集到的数据条数

        Raises:
            DataCollectionError: 数据采集失败时
        """
        try:
            # 确定采集范围
            if ts_code:
                stock_codes = [ts_code]
            else:
                # 获取所有上市股票代码
                stocks = await self.stock_repo.get_all_stock_codes()
                stock_codes = [stock.ts_code for stock in stocks]

            if not stock_codes:
                logger.warning("未找到需要采集的股票代码")
                return 0

            # 确定日期范围
            if is_incremental and not start_date:
                # 增量更新：从最后更新日期开始
                last_date = await self.stock_repo.get_last_financial_data_date(ts_code)
                start_date = (
                    last_date if last_date else date.today() - timedelta(days=365)
                )

            if not end_date:
                end_date = date.today()

            logger.info(
                f"开始采集财务数据, 股票数量: {len(stock_codes)}, 日期范围: {start_date} ~ {end_date}"
            )

            total_count = 0
            for i, code in enumerate(stock_codes, 1):
                try:
                    count = await self._collect_single_stock_financial_data(
                        code, start_date, end_date, period
                    )
                    total_count += count
                    logger.debug(
                        f"进度: {i}/{len(stock_codes)}, 股票: {code}, 采集: {count} 条"
                    )
                except Exception as e:
                    logger.error(f"采集股票 {code} 财务数据失败: {e}")
                    continue

            logger.info(f"财务数据采集完成, 总计: {total_count} 条")
            return total_count

        except Exception as e:
            error_msg = f"采集财务数据失败: {e!s}"
            logger.error(error_msg)
            raise DataCollectionError(error_msg) from e

    async def _collect_single_stock_daily_data(
        self, ts_code: str, start_date: str | date, end_date: str | date
    ) -> int:
        """采集单只股票的日线数据

        Args:
            ts_code: 股票代码
            start_date: 开始日期
            end_date: 结束日期

        Returns:
            采集到的数据条数
        """
        # 从Tushare获取数据
        raw_data = self.tushare_client.get_daily_data(ts_code, start_date, end_date)
        if not raw_data:
            return 0

        # 数据质量验证和去重
        validated_data = []
        for data in raw_data:
            try:
                # 验证数据完整性
                self.quality_service.validate_daily_data(data)

                # 检查是否已存在
                existing = await self.stock_repo.get_daily_data(
                    ts_code, data.trade_date
                )
                if not existing:
                    validated_data.append(data)

            except ValidationError as e:
                logger.warning(
                    f"日线数据验证失败: {ts_code} {data.trade_date}, 错误: {e}"
                )
                continue

        # 批量保存
        if validated_data:
            await self.stock_repo.batch_create_daily_data(validated_data)

        return len(validated_data)

    async def _collect_single_stock_financial_data(
        self,
        ts_code: str,
        start_date: str | date,
        end_date: str | date,
        period: str | None,
    ) -> int:
        """采集单只股票的财务数据

        Args:
            ts_code: 股票代码
            start_date: 开始日期
            end_date: 结束日期
            period: 报告期

        Returns:
            采集到的数据条数
        """
        # 从Tushare获取数据
        raw_data = self.tushare_client.get_financial_data(
            ts_code, start_date, end_date, period
        )
        if not raw_data:
            return 0

        # 数据质量验证和去重
        validated_data = []
        for data in raw_data:
            try:
                # 验证数据完整性
                self.quality_service.validate_financial_data(data)

                # 检查是否已存在
                existing = await self.stock_repo.get_financial_data(
                    ts_code, data.end_date
                )
                if not existing:
                    validated_data.append(data)

            except ValidationError as e:
                logger.warning(
                    f"财务数据验证失败: {ts_code} {data.end_date}, 错误: {e}"
                )
                continue

        # 批量保存
        if validated_data:
            await self.stock_repo.batch_create_financial_data(validated_data)

        return len(validated_data)

    def _should_update_stock_basic(
        self, existing: StockBasicInfo, new_data: StockBasicInfo
    ) -> bool:
        """判断是否需要更新股票基础信息

        Args:
            existing: 现有数据
            new_data: 新数据

        Returns:
            是否需要更新
        """
        # 检查关键字段是否有变化
        key_fields = [
            "name",
            "industry",
            "market",
            "list_status",
            "list_date",
            "delist_date",
        ]
        for field in key_fields:
            if getattr(existing, field, None) != getattr(new_data, field, None):
                return True
        return False

    async def get_collection_stats(self) -> dict[str, Any]:
        """获取数据采集统计信息

        Returns:
            采集统计信息
        """
        try:
            stats = {
                "stock_count": await self.stock_repo.get_stock_count(),
                "daily_data_count": await self.stock_repo.get_daily_data_count(),
                "financial_data_count": await self.stock_repo.get_financial_data_count(),
                "last_daily_date": await self.stock_repo.get_last_daily_data_date(),
                "last_financial_date": await self.stock_repo.get_last_financial_data_date(),
            }
            return stats
        except Exception as e:
            logger.error(f"获取采集统计信息失败: {e}")
            return {}

    # 手动采集任务管理方法

    async def create_manual_collection_task(
        self,
        task_type: TaskType,
        task_name: str,
        params: dict[str, Any],
        priority: int = 5,
        max_retries: int = 3,
        created_by: str | None = None,
    ) -> int | None:
        """创建手动采集任务

        Args:
            task_type: 任务类型
            task_name: 任务名称
            params: 任务参数
            priority: 优先级(1-10,数字越小优先级越高)
            max_retries: 最大重试次数
            created_by: 创建者

        Returns:
            任务ID，创建失败返回None
        """
        try:
            # 验证任务参数
            self._validate_task_params(task_type, params)

            # 创建任务对象
            task = Task(
                name=task_name,
                task_type=task_type,
                status=TaskStatus.PENDING,
                priority=priority,
                params=params,
                max_retries=max_retries,
                created_by=created_by,
                created_at=datetime.now(),
                updated_at=datetime.now(),
            )

            # 保存到数据库
            task_id = await self.task_repo.create_task(task)
            if task_id:
                logger.info(
                    f"手动采集任务创建成功: ID={task_id}, 类型={task_type}, 名称={task_name}"
                )
            return task_id

        except Exception as e:
            logger.error(f"创建手动采集任务失败: {e}")
            return None

    async def execute_manual_collection_task(self, task_id: int) -> bool:
        """执行手动采集任务

        Args:
            task_id: 任务ID

        Returns:
            是否执行成功
        """
        task = self.task_repo.get_by_id(task_id)
        if not task:
            logger.error(f"任务不存在: ID={task_id}")
            return False

        if task.status != TaskStatus.PENDING:
            logger.warning(f"任务状态不允许执行: ID={task_id}, 状态={task.status}")
            return False

        try:
            # 更新任务状态为运行中
            self.task_repo.update_status(task_id, TaskStatus.RUNNING)

            # 根据任务类型执行相应的采集逻辑
            result = await self._execute_collection_by_type(task)

            # 更新任务结果和状态
            self.task_repo.update_result(task_id, result)
            self.task_repo.update_status(task_id, TaskStatus.COMPLETED)

            logger.info(f"手动采集任务执行成功: ID={task_id}")
            return True

        except Exception as e:
            error_msg = f"任务执行失败: {e}"
            logger.error(f"手动采集任务执行失败: ID={task_id}, 错误: {e}")

            # 检查是否需要重试
            if task.retry_count < task.max_retries:
                await self._retry_task(task_id, error_msg)
            else:
                self.task_repo.update_status(task_id, TaskStatus.FAILED, error_msg)

            return False

    async def get_pending_tasks_by_priority(self, limit: int = 10) -> list[Task]:
        """按优先级获取待执行任务

        Args:
            limit: 返回数量限制

        Returns:
            按优先级排序的待执行任务列表
        """
        try:
            tasks = self.task_repo.get_pending_tasks_by_priority(limit)
            return tasks
        except Exception as e:
            logger.error(f"获取待执行任务失败: {e}")
            return []

    async def process_task_queue(self, max_concurrent: int = 3) -> None:
        """处理任务队列

        Args:
            max_concurrent: 最大并发任务数
        """
        try:
            # 获取待执行任务
            pending_tasks = await self.get_pending_tasks_by_priority(max_concurrent)
            if not pending_tasks:
                logger.debug("没有待执行的任务")
                return

            # 并发执行任务
            tasks = [
                self.execute_manual_collection_task(task.id) for task in pending_tasks
            ]
            results = await asyncio.gather(*tasks, return_exceptions=True)

            # 统计执行结果
            success_count = sum(1 for result in results if result is True)
            logger.info(
                f"任务队列处理完成: 总数={len(pending_tasks)}, 成功={success_count}"
            )

        except Exception as e:
            logger.error(f"处理任务队列失败: {e}")

    async def _execute_collection_by_type(self, task: Task) -> dict[str, Any]:
        """根据任务类型执行采集逻辑

        Args:
            task: 任务对象

        Returns:
            执行结果
        """
        params = task.params or {}

        if task.task_type == TaskType.STOCK_BASIC_COLLECTION:
            # 股票基础信息采集
            count = await self.collect_stock_basic_info(
                list_status=params.get("list_status", "L"),
                exchange=params.get("exchange"),
                force_update=params.get("force_update", False),
            )
            return {"collected_count": count, "data_type": "stock_basic_info"}

        elif task.task_type == TaskType.DAILY_DATA_COLLECTION:
            # 日线数据采集
            count = await self.collect_daily_data(
                ts_code=params.get("ts_code"),
                start_date=params.get("start_date"),
                end_date=params.get("end_date"),
                is_incremental=params.get("is_incremental", True),
            )
            return {"collected_count": count, "data_type": "daily_data"}

        elif task.task_type == TaskType.FINANCIAL_DATA_COLLECTION:
            # 财务数据采集
            count = await self.collect_financial_data(
                ts_code=params.get("ts_code"),
                start_date=params.get("start_date"),
                end_date=params.get("end_date"),
                period=params.get("period"),
                is_incremental=params.get("is_incremental", True),
            )
            return {"collected_count": count, "data_type": "financial_data"}

        else:
            raise DataCollectionError(f"不支持的任务类型: {task.task_type}")

    async def _retry_task(self, task_id: int, error_msg: str) -> None:
        """重试任务

        Args:
            task_id: 任务ID
            error_msg: 错误信息
        """
        try:
            task = self.task_repo.get_by_id(task_id)
            if not task:
                return

            # 增加重试次数
            task.retry_count += 1
            task.status = TaskStatus.PENDING
            task.error_message = error_msg
            task.updated_at = datetime.now()

            # 计算重试延迟时间（指数退避）
            delay_seconds = min(60 * (2**task.retry_count), 3600)  # 最大1小时
            task.scheduled_at = datetime.now() + timedelta(seconds=delay_seconds)

            # 更新任务
            self.task_repo.update_task(task)

            logger.info(
                f"任务重试安排: ID={task_id}, 重试次数={task.retry_count}, 延迟={delay_seconds}秒"
            )

        except Exception as e:
            logger.error(f"安排任务重试失败: ID={task_id}, 错误: {e}")

    def _validate_task_params(self, task_type: TaskType, params: dict[str, Any]) -> None:
        """验证任务参数

        Args:
            task_type: 任务类型
            params: 任务参数

        Raises:
            ValidationError: 参数验证失败
        """
        if task_type == TaskType.STOCK_BASIC_COLLECTION:
            # 股票基础信息采集参数验证
            list_status = params.get("list_status", "L")
            if list_status not in ["L", "D", "P"]:
                raise ValidationError(f"无效的上市状态: {list_status}")

            exchange = params.get("exchange")
            if exchange and exchange not in ["SSE", "SZSE"]:
                raise ValidationError(f"无效的交易所: {exchange}")

        elif task_type == TaskType.DAILY_DATA_COLLECTION:
            # 日线数据采集参数验证
            start_date = params.get("start_date")
            end_date = params.get("end_date")

            if start_date and end_date:
                try:
                    if isinstance(start_date, str):
                        start_date = datetime.strptime(start_date, "%Y-%m-%d").date()
                    if isinstance(end_date, str):
                        end_date = datetime.strptime(end_date, "%Y-%m-%d").date()

                    if start_date > end_date:
                        raise ValidationError("开始日期不能大于结束日期")
                except ValueError as e:
                    raise ValidationError(f"日期格式错误: {e}") from e

        elif task_type == TaskType.FINANCIAL_DATA_COLLECTION:
            # 财务数据采集参数验证
            period = params.get("period")
            if period and period not in ["Q1", "Q2", "Q3", "Q4", "A"]:
                raise ValidationError(f"无效的报告期: {period}")

        else:
            raise ValidationError(f"不支持的任务类型: {task_type}")
