"""任务调度器模块

基于APScheduler实现的任务调度器，支持定时任务的创建、管理和监控。
集成了任务管理器，提供完整的任务生命周期管理功能。
"""

from collections.abc import Callable
from datetime import datetime
from typing import Any

from apscheduler.executors.asyncio import AsyncIOExecutor
from apscheduler.jobstores.memory import MemoryJobStore
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from loguru import logger

from scheduler.jobs import DataCollectionJobs, HealthCheckJobs, SystemMaintenanceJobs
from scheduler.manager import JobConfig, TaskManager
from utils.exceptions import SchedulerError


class TaskScheduler:
    """任务调度器

    提供定时任务的调度、管理和监控功能。
    集成了任务管理器和预定义的任务集合。
    """

    def __init__(self, data_collection_orchestrator=None):
        """初始化调度器

        Args:
            data_collection_orchestrator: 数据采集编排器实例
        """
        # 配置调度器
        jobstores = {"default": MemoryJobStore()}
        executors = {"default": AsyncIOExecutor()}
        job_defaults = {"coalesce": False, "max_instances": 3}

        self.scheduler = AsyncIOScheduler(
            jobstores=jobstores,
            executors=executors,
            job_defaults=job_defaults,
            timezone="Asia/Shanghai",
        )

        # 初始化任务管理器
        self.task_manager = TaskManager(self.scheduler)

        # 初始化任务集合
        self.data_collection_jobs = (
            DataCollectionJobs(data_collection_orchestrator)
            if data_collection_orchestrator
            else None
        )
        self.system_maintenance_jobs = SystemMaintenanceJobs()
        self.health_check_jobs = HealthCheckJobs()

        logger.info("任务调度器初始化完成")

    async def start(self) -> None:
        """启动调度器"""
        try:
            self.scheduler.start()
            logger.info("任务调度器启动成功")

            # 注册预定义任务
            await self._register_predefined_jobs()

        except Exception as e:
            logger.error(f"任务调度器启动失败: {e}")
            raise SchedulerError(f"调度器启动失败: {e}") from e

    async def shutdown(self, wait: bool = True) -> None:
        """关闭调度器

        Args:
            wait: 是否等待正在执行的任务完成
        """
        try:
            self.scheduler.shutdown(wait=wait)
            logger.info("任务调度器关闭成功")
        except Exception as e:
            logger.error(f"任务调度器关闭失败: {e}")
            raise SchedulerError(f"调度器关闭失败: {e}") from e

    async def _register_predefined_jobs(self) -> None:
        """注册预定义任务"""
        logger.info("开始注册预定义任务")

        # 数据采集任务
        if self.data_collection_jobs:
            # 日度股票数据采集 - 每个交易日18:00执行
            self.task_manager.register_job(
                JobConfig(
                    job_id="daily_stock_data_collection",
                    job_name="日度股票数据采集",
                    job_func=self.data_collection_jobs.daily_stock_data_collection,
                    trigger_type="cron",
                    trigger_args={"hour": 18, "minute": 0, "day_of_week": "mon-fri"},
                    max_retries=3,
                    timeout=3600,  # 1小时超时
                    metadata={"category": "data_collection", "priority": "high"},
                )
            )

            # 周度股票基础信息更新 - 每周一09:00执行
            self.task_manager.register_job(
                JobConfig(
                    job_id="weekly_stock_basic_info_update",
                    job_name="周度股票基础信息更新",
                    job_func=self.data_collection_jobs.weekly_stock_basic_info_update,
                    trigger_type="cron",
                    trigger_args={"hour": 9, "minute": 0, "day_of_week": "mon"},
                    max_retries=2,
                    timeout=1800,  # 30分钟超时
                    metadata={"category": "data_collection", "priority": "medium"},
                )
            )

            # 月度财务数据采集 - 每月1日10:00执行
            self.task_manager.register_job(
                JobConfig(
                    job_id="monthly_financial_data_collection",
                    job_name="月度财务数据采集",
                    job_func=self.data_collection_jobs.monthly_financial_data_collection,
                    trigger_type="cron",
                    trigger_args={"hour": 10, "minute": 0, "day": 1},
                    max_retries=3,
                    timeout=7200,  # 2小时超时
                    metadata={"category": "data_collection", "priority": "medium"},
                )
            )

        # 系统维护任务
        # 日度日志清理 - 每日02:00执行
        self.task_manager.register_job(
            JobConfig(
                job_id="daily_log_cleanup",
                job_name="日度日志清理",
                job_func=self.system_maintenance_jobs.daily_log_cleanup,
                trigger_type="cron",
                trigger_args={"hour": 2, "minute": 0},
                max_retries=1,
                timeout=600,  # 10分钟超时
                metadata={"category": "maintenance", "priority": "low"},
            )
        )

        # 周度缓存清理 - 每周日03:00执行
        self.task_manager.register_job(
            JobConfig(
                job_id="weekly_cache_cleanup",
                job_name="周度缓存清理",
                job_func=self.system_maintenance_jobs.weekly_cache_cleanup,
                trigger_type="cron",
                trigger_args={"hour": 3, "minute": 0, "day_of_week": "sun"},
                max_retries=1,
                timeout=1800,  # 30分钟超时
                metadata={"category": "maintenance", "priority": "low"},
            )
        )

        # 健康检查任务
        # 小时级系统健康检查 - 每小时执行
        self.task_manager.register_job(
            JobConfig(
                job_id="hourly_system_health_check",
                job_name="小时级系统健康检查",
                job_func=self.health_check_jobs.hourly_system_health_check,
                trigger_type="cron",
                trigger_args={"minute": 0},  # 每小时的0分执行
                max_retries=1,
                timeout=300,  # 5分钟超时
                metadata={"category": "health_check", "priority": "high"},
            )
        )

        logger.info("预定义任务注册完成")

    def add_cron_job(
        self,
        func: Callable,
        job_id: str,
        name: str,
        cron_expression: str | None = None,
        **cron_kwargs,
    ) -> bool:
        """添加Cron定时任务

        Args:
            func: 任务函数
            job_id: 任务ID
            name: 任务名称
            cron_expression: Cron表达式（可选）
            **cron_kwargs: Cron参数（分钟、小时、日等）

        Returns:
            是否添加成功
        """
        try:
            if cron_expression:
                # 解析Cron表达式
                parts = cron_expression.split()
                if len(parts) == 5:
                    minute, hour, day, month, day_of_week = parts
                    trigger_args = {
                        "minute": minute,
                        "hour": hour,
                        "day": day,
                        "month": month,
                        "day_of_week": day_of_week,
                    }
                else:
                    raise ValueError(f"无效的Cron表达式: {cron_expression}")
            else:
                trigger_args = cron_kwargs

            # 使用任务管理器注册任务
            config = JobConfig(
                job_id=job_id,
                job_name=name,
                job_func=func,
                trigger_type="cron",
                trigger_args=trigger_args,
            )

            return self.task_manager.register_job(config)

        except Exception as e:
            logger.error(f"Cron任务添加失败: {job_id}, 错误: {e}")
            return False

    def add_interval_job(
        self,
        func: Callable,
        job_id: str,
        name: str,
        seconds: int = 0,
        minutes: int = 0,
        hours: int = 0,
        days: int = 0,
        **kwargs,
    ) -> bool:
        """添加间隔定时任务

        Args:
            func: 任务函数
            job_id: 任务ID
            name: 任务名称
            seconds: 间隔秒数
            minutes: 间隔分钟数
            hours: 间隔小时数
            days: 间隔天数
            **kwargs: 其他参数

        Returns:
            是否添加成功
        """
        try:
            trigger_args = {
                "seconds": seconds,
                "minutes": minutes,
                "hours": hours,
                "days": days,
            }

            # 使用任务管理器注册任务
            config = JobConfig(
                job_id=job_id,
                job_name=name,
                job_func=func,
                trigger_type="interval",
                trigger_args=trigger_args,
            )

            return self.task_manager.register_job(config)

        except Exception as e:
            logger.error(f"间隔任务添加失败: {job_id}, 错误: {e}")
            return False

    def add_date_job(
        self, func: Callable, job_id: str, name: str, run_date: datetime, **kwargs
    ) -> bool:
        """添加一次性定时任务

        Args:
            func: 任务函数
            job_id: 任务ID
            name: 任务名称
            run_date: 执行时间
            **kwargs: 其他参数

        Returns:
            是否添加成功
        """
        try:
            trigger_args = {"run_date": run_date}

            # 使用任务管理器注册任务
            config = JobConfig(
                job_id=job_id,
                job_name=name,
                job_func=func,
                trigger_type="date",
                trigger_args=trigger_args,
            )

            return self.task_manager.register_job(config)

        except Exception as e:
            logger.error(f"一次性任务添加失败: {job_id}, 错误: {e}")
            return False

    def remove_job(self, job_id: str) -> bool:
        """移除任务

        Args:
            job_id: 任务ID

        Returns:
            是否移除成功
        """
        return self.task_manager.remove_job(job_id)

    def pause_job(self, job_id: str) -> bool:
        """暂停任务

        Args:
            job_id: 任务ID

        Returns:
            是否暂停成功
        """
        return self.task_manager.pause_job(job_id)

    def resume_job(self, job_id: str) -> bool:
        """恢复任务

        Args:
            job_id: 任务ID

        Returns:
            是否恢复成功
        """
        return self.task_manager.resume_job(job_id)

    def get_job_info(self, job_id: str) -> dict[str, Any] | None:
        """获取任务信息

        Args:
            job_id: 任务ID

        Returns:
            任务信息字典
        """
        return self.task_manager.get_job_info(job_id)

    def list_jobs(self) -> list[dict[str, Any]]:
        """列出所有任务

        Returns:
            任务信息列表
        """
        return self.task_manager.list_jobs()

    def get_execution_history(
        self, job_id: str | None = None, limit: int = 100
    ) -> list[dict[str, Any]]:
        """获取任务执行历史

        Args:
            job_id: 任务ID，如果为None则返回所有任务的历史
            limit: 返回记录数量限制

        Returns:
            执行历史列表
        """
        records = self.task_manager.get_execution_history(job_id, limit)

        # 转换为字典格式以保持兼容性
        return [
            {
                "job_id": record.job_id,
                "job_name": record.job_name,
                "status": record.status.value,
                "start_time": record.start_time.isoformat(),
                "end_time": record.end_time.isoformat() if record.end_time else None,
                "duration": record.duration,
                "error_message": record.error_message,
                "retry_count": record.retry_count,
                "metadata": record.metadata,
            }
            for record in records
        ]

    def get_running_jobs(self) -> list[dict[str, Any]]:
        """获取正在运行的任务

        Returns:
            正在运行的任务列表
        """
        records = self.task_manager.get_running_jobs()

        return [
            {
                "job_id": record.job_id,
                "job_name": record.job_name,
                "status": record.status.value,
                "start_time": record.start_time.isoformat(),
                "metadata": record.metadata,
            }
            for record in records
        ]

    def get_statistics(self) -> dict[str, Any]:
        """获取统计信息

        Returns:
            统计信息字典
        """
        return self.task_manager.get_statistics()

    async def trigger_job(self, job_id: str) -> bool:
        """手动触发任务执行

        Args:
            job_id: 任务ID

        Returns:
            是否触发成功
        """
        return await self.task_manager.trigger_job(job_id)

    async def trigger_emergency_data_collection(
        self, task_type, force_update: bool = True
    ) -> bool:
        """触发紧急数据采集

        Args:
            task_type: 任务类型
            force_update: 是否强制更新

        Returns:
            是否触发成功
        """
        if not self.data_collection_jobs:
            logger.error("数据采集任务未初始化")
            return False

        try:
            result = await self.data_collection_jobs.emergency_data_collection(
                task_type, force_update
            )
            logger.info(f"紧急数据采集触发成功: {result}")
            return True
        except Exception as e:
            logger.error(f"紧急数据采集触发失败: {e}")
            return False
