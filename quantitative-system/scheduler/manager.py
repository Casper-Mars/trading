"""任务管理器模块

提供任务的生命周期管理、状态监控、执行历史记录等功能。
"""

import asyncio
from collections.abc import Callable
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from enum import Enum
from typing import Any

from apscheduler.events import EVENT_JOB_ERROR, EVENT_JOB_EXECUTED, JobExecutionEvent
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from loguru import logger

from utils.exceptions import TaskManagerError


class JobStatus(Enum):
    """任务状态枚举"""

    PENDING = "pending"  # 等待执行
    RUNNING = "running"  # 正在执行
    COMPLETED = "completed"  # 执行完成
    FAILED = "failed"  # 执行失败
    CANCELLED = "cancelled"  # 已取消
    PAUSED = "paused"  # 已暂停


@dataclass
class JobExecutionRecord:
    """任务执行记录"""

    job_id: str
    job_name: str
    status: JobStatus
    start_time: datetime
    end_time: datetime | None = None
    execution_time: float | None = None
    result: dict[str, Any] | None = None
    error_message: str | None = None
    retry_count: int = 0
    metadata: dict[str, Any] = field(default_factory=dict)

    @property
    def duration(self) -> float | None:
        """计算执行时长（秒）"""
        if self.start_time and self.end_time:
            return (self.end_time - self.start_time).total_seconds()
        return None


@dataclass
class JobConfig:
    """任务配置"""

    job_id: str
    job_name: str
    job_func: Callable
    trigger_type: str  # 'cron', 'interval', 'date'
    trigger_args: dict[str, Any]
    max_retries: int = 3
    retry_delay: int = 60  # 重试延迟（秒）
    timeout: int | None = None  # 超时时间（秒）
    enabled: bool = True
    metadata: dict[str, Any] = field(default_factory=dict)


class TaskManager:
    """任务管理器

    负责管理定时任务的生命周期，包括任务的创建、执行、监控、重试等。
    """

    def __init__(self, scheduler: AsyncIOScheduler):
        """初始化任务管理器

        Args:
            scheduler: APScheduler调度器实例
        """
        self.scheduler = scheduler
        self.job_configs: dict[str, JobConfig] = {}
        self.execution_records: list[JobExecutionRecord] = []
        self.running_jobs: dict[str, JobExecutionRecord] = {}

        # 设置事件监听器
        self.scheduler.add_listener(self._on_job_executed, EVENT_JOB_EXECUTED)
        self.scheduler.add_listener(self._on_job_error, EVENT_JOB_ERROR)

    def register_job(self, config: JobConfig) -> bool:
        """注册任务

        Args:
            config: 任务配置

        Returns:
            是否注册成功
        """
        try:
            if config.job_id in self.job_configs:
                logger.warning(f"任务已存在, 将覆盖原配置: {config.job_id}")
                self.remove_job(config.job_id)

            # 包装任务函数以支持监控和重试
            wrapped_func = self._wrap_job_function(config)

            # 根据触发器类型添加任务
            if config.trigger_type == "cron":
                self.scheduler.add_job(
                    wrapped_func,
                    "cron",
                    id=config.job_id,
                    name=config.job_name,
                    **config.trigger_args,
                )
            elif config.trigger_type == "interval":
                self.scheduler.add_job(
                    wrapped_func,
                    "interval",
                    id=config.job_id,
                    name=config.job_name,
                    **config.trigger_args,
                )
            elif config.trigger_type == "date":
                self.scheduler.add_job(
                    wrapped_func,
                    "date",
                    id=config.job_id,
                    name=config.job_name,
                    **config.trigger_args,
                )
            else:
                raise TaskManagerError(f"不支持的触发器类型: {config.trigger_type}")

            # 保存配置
            self.job_configs[config.job_id] = config

            # 如果任务被禁用，则暂停
            if not config.enabled:
                self.pause_job(config.job_id)

            logger.info(f"任务注册成功: {config.job_id} ({config.job_name})")
            return True

        except Exception as e:
            logger.error(f"任务注册失败: {config.job_id}, 错误: {e}")
            return False

    def _wrap_job_function(self, config: JobConfig) -> Callable:
        """包装任务函数以支持监控和重试

        Args:
            config: 任务配置

        Returns:
            包装后的任务函数
        """

        async def wrapped_function():
            job_id = config.job_id
            start_time = datetime.now()

            # 创建执行记录
            record = JobExecutionRecord(
                job_id=job_id,
                job_name=config.job_name,
                status=JobStatus.RUNNING,
                start_time=start_time,
                metadata=config.metadata.copy(),
            )

            # 记录正在运行的任务
            self.running_jobs[job_id] = record

            try:
                logger.info(f"开始执行任务: {job_id} ({config.job_name})")

                # 执行原始任务函数
                if asyncio.iscoroutinefunction(config.job_func):
                    result = await config.job_func()
                else:
                    result = config.job_func()

                # 更新执行记录
                record.status = JobStatus.COMPLETED
                record.end_time = datetime.now()
                record.result = result
                record.execution_time = record.duration

                logger.info(f"任务执行完成: {job_id}, 耗时: {record.duration:.2f}秒")

            except Exception as e:
                # 更新执行记录
                record.status = JobStatus.FAILED
                record.end_time = datetime.now()
                record.error_message = str(e)
                record.execution_time = record.duration

                logger.error(f"任务执行失败: {job_id}, 错误: {e}")

                # 重新抛出异常以触发调度器的错误处理
                raise

            finally:
                # 移除正在运行的任务记录
                self.running_jobs.pop(job_id, None)

                # 添加到执行历史
                self.execution_records.append(record)

                # 保持执行历史记录数量在合理范围内
                if len(self.execution_records) > 1000:
                    self.execution_records = self.execution_records[-500:]

        return wrapped_function

    def _on_job_executed(self, event: JobExecutionEvent):
        """任务执行完成事件处理器"""
        job_id = event.job_id
        logger.debug(f"任务执行完成事件: {job_id}")

    def _on_job_error(self, event: JobExecutionEvent):
        """任务执行错误事件处理器"""
        job_id = event.job_id
        exception = event.exception
        logger.error(f"任务执行错误事件: {job_id}, 异常: {exception}")

        # TODO: 实现重试逻辑
        # config = self.job_configs.get(job_id)
        # if config and config.max_retries > 0:
        #     self._schedule_retry(job_id, config)

    def remove_job(self, job_id: str) -> bool:
        """移除任务

        Args:
            job_id: 任务ID

        Returns:
            是否移除成功
        """
        try:
            # 从调度器中移除
            self.scheduler.remove_job(job_id)

            # 从配置中移除
            self.job_configs.pop(job_id, None)

            # 从正在运行的任务中移除
            self.running_jobs.pop(job_id, None)

            logger.info(f"任务移除成功: {job_id}")
            return True

        except Exception as e:
            logger.error(f"任务移除失败: {job_id}, 错误: {e}")
            return False

    def pause_job(self, job_id: str) -> bool:
        """暂停任务

        Args:
            job_id: 任务ID

        Returns:
            是否暂停成功
        """
        try:
            self.scheduler.pause_job(job_id)

            # 更新配置状态
            if job_id in self.job_configs:
                self.job_configs[job_id].enabled = False

            logger.info(f"任务暂停成功: {job_id}")
            return True

        except Exception as e:
            logger.error(f"任务暂停失败: {job_id}, 错误: {e}")
            return False

    def resume_job(self, job_id: str) -> bool:
        """恢复任务

        Args:
            job_id: 任务ID

        Returns:
            是否恢复成功
        """
        try:
            self.scheduler.resume_job(job_id)

            # 更新配置状态
            if job_id in self.job_configs:
                self.job_configs[job_id].enabled = True

            logger.info(f"任务恢复成功: {job_id}")
            return True

        except Exception as e:
            logger.error(f"任务恢复失败: {job_id}, 错误: {e}")
            return False

    def get_job_status(self, job_id: str) -> JobStatus | None:
        """获取任务状态

        Args:
            job_id: 任务ID

        Returns:
            任务状态，如果任务不存在则返回None
        """
        # 检查是否正在运行
        if job_id in self.running_jobs:
            return JobStatus.RUNNING

        # 检查调度器中的任务状态
        job = self.scheduler.get_job(job_id)
        if job is None:
            return None

        # 检查是否被暂停
        if job_id in self.job_configs and not self.job_configs[job_id].enabled:
            return JobStatus.PAUSED

        return JobStatus.PENDING

    def get_job_info(self, job_id: str) -> dict[str, Any] | None:
        """获取任务信息

        Args:
            job_id: 任务ID

        Returns:
            任务信息字典
        """
        config = self.job_configs.get(job_id)
        if not config:
            return None

        job = self.scheduler.get_job(job_id)
        status = self.get_job_status(job_id)

        # 获取最近的执行记录
        recent_records = [r for r in self.execution_records if r.job_id == job_id]
        last_execution = recent_records[-1] if recent_records else None

        return {
            "job_id": job_id,
            "job_name": config.job_name,
            "status": status.value if status else "unknown",
            "enabled": config.enabled,
            "trigger_type": config.trigger_type,
            "next_run_time": job.next_run_time.isoformat()
            if job and job.next_run_time
            else None,
            "last_execution": {
                "start_time": last_execution.start_time.isoformat()
                if last_execution
                else None,
                "end_time": last_execution.end_time.isoformat()
                if last_execution and last_execution.end_time
                else None,
                "status": last_execution.status.value if last_execution else None,
                "duration": last_execution.duration if last_execution else None,
                "error_message": last_execution.error_message
                if last_execution
                else None,
            }
            if last_execution
            else None,
            "execution_count": len(recent_records),
            "success_count": len(
                [r for r in recent_records if r.status == JobStatus.COMPLETED]
            ),
            "failure_count": len(
                [r for r in recent_records if r.status == JobStatus.FAILED]
            ),
            "metadata": config.metadata,
        }

    def list_jobs(self) -> list[dict[str, Any]]:
        """列出所有任务

        Returns:
            任务信息列表
        """
        jobs = []
        for job_id in self.job_configs:
            job_info = self.get_job_info(job_id)
            if job_info:
                jobs.append(job_info)
        return jobs

    def get_execution_history(
        self, job_id: str | None = None, limit: int = 100
    ) -> list[JobExecutionRecord]:
        """获取执行历史

        Args:
            job_id: 任务ID，如果为None则返回所有任务的历史
            limit: 返回记录数量限制

        Returns:
            执行记录列表
        """
        records = self.execution_records

        if job_id:
            records = [r for r in records if r.job_id == job_id]

        # 按开始时间倒序排列
        records = sorted(records, key=lambda x: x.start_time, reverse=True)

        return records[:limit]

    def get_running_jobs(self) -> list[JobExecutionRecord]:
        """获取正在运行的任务

        Returns:
            正在运行的任务记录列表
        """
        return list(self.running_jobs.values())

    def get_statistics(self) -> dict[str, Any]:
        """获取统计信息

        Returns:
            统计信息字典
        """
        total_jobs = len(self.job_configs)
        enabled_jobs = len([c for c in self.job_configs.values() if c.enabled])
        running_jobs = len(self.running_jobs)

        # 最近24小时的执行统计
        now = datetime.now()
        recent_records = [
            r
            for r in self.execution_records
            if r.start_time > now - timedelta(hours=24)
        ]

        successful_executions = len(
            [r for r in recent_records if r.status == JobStatus.COMPLETED]
        )
        failed_executions = len(
            [r for r in recent_records if r.status == JobStatus.FAILED]
        )

        return {
            "total_jobs": total_jobs,
            "enabled_jobs": enabled_jobs,
            "disabled_jobs": total_jobs - enabled_jobs,
            "running_jobs": running_jobs,
            "recent_24h": {
                "total_executions": len(recent_records),
                "successful_executions": successful_executions,
                "failed_executions": failed_executions,
                "success_rate": successful_executions / len(recent_records) * 100
                if recent_records
                else 0,
            },
            "total_execution_records": len(self.execution_records),
        }

    async def trigger_job(self, job_id: str) -> bool:
        """手动触发任务执行

        Args:
            job_id: 任务ID

        Returns:
            是否触发成功
        """
        try:
            job = self.scheduler.get_job(job_id)
            if not job:
                logger.error(f"任务不存在: {job_id}")
                return False

            # 手动触发任务
            job.modify(next_run_time=datetime.now())

            logger.info(f"手动触发任务: {job_id}")
            return True

        except Exception as e:
            logger.error(f"手动触发任务失败: {job_id}, 错误: {e}")
            return False
