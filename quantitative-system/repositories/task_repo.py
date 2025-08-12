"""任务数据访问层"""

from datetime import datetime
from typing import Any

from loguru import logger
from sqlmodel import Session, and_, desc, select

from config.database import get_session
from models.database import Task
from models.enums import TaskStatus, TaskType


class TaskRepository:
    """任务数据仓库

    提供任务数据的增删改查和状态管理功能
    """

    def __init__(self, session: Session | None = None):
        """初始化任务仓库

        Args:
            session: 数据库会话，如果为None则使用默认会话
        """
        self.session = session

    def _get_session(self) -> Session:
        """获取数据库会话"""
        if self.session:
            return self.session
        return next(get_session())

    def get_by_id(self, task_id: int) -> Task | None:
        """根据ID获取任务

        Args:
            task_id: 任务ID

        Returns:
            任务对象，不存在返回None
        """
        try:
            with self._get_session() as session:
                statement = select(Task).where(Task.id == task_id)
                task = session.exec(statement).first()
                return task

        except Exception as e:
            logger.error(f"获取任务失败: ID={task_id}, 错误: {e}")
            return None

    def get_by_status(self, status: TaskStatus, limit: int = 100) -> list[Task]:
        """根据状态获取任务列表

        Args:
            status: 任务状态
            limit: 返回数量限制

        Returns:
            任务列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(Task)
                    .where(Task.status == status)
                    .order_by(desc(Task.created_at))
                    .limit(limit)
                )
                tasks = session.exec(statement).all()
                return list(tasks)

        except Exception as e:
            logger.error(f"根据状态获取任务失败: status={status}, 错误: {e}")
            return []

    def get_by_type(self, task_type: TaskType, limit: int = 100) -> list[Task]:
        """根据类型获取任务列表

        Args:
            task_type: 任务类型
            limit: 返回数量限制

        Returns:
            任务列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(Task)
                    .where(Task.task_type == task_type)
                    .order_by(desc(Task.created_at))
                    .limit(limit)
                )
                tasks = session.exec(statement).all()
                return list(tasks)

        except Exception as e:
            logger.error(f"根据类型获取任务失败: task_type={task_type}, 错误: {e}")
            return []

    def get_recent_tasks(
        self,
        limit: int = 50,
        task_type: TaskType | None = None,
        status: TaskStatus | None = None,
    ) -> list[Task]:
        """获取最近的任务列表

        Args:
            limit: 返回数量限制
            task_type: 可选的任务类型过滤
            status: 可选的状态过滤

        Returns:
            任务列表
        """
        try:
            with self._get_session() as session:
                statement = select(Task)

                # 添加过滤条件
                conditions = []
                if task_type is not None:
                    conditions.append(Task.task_type == task_type)
                if status is not None:
                    conditions.append(Task.status == status)

                if conditions:
                    statement = statement.where(and_(*conditions))

                statement = statement.order_by(desc(Task.created_at)).limit(limit)
                tasks = session.exec(statement).all()
                return list(tasks)

        except Exception as e:
            logger.error(f"获取最近任务失败: 错误: {e}")
            return []

    def update_status(
        self, task_id: int, status: TaskStatus, error_message: str | None = None
    ) -> bool:
        """更新任务状态

        Args:
            task_id: 任务ID
            status: 新状态
            error_message: 可选的错误信息

        Returns:
            是否更新成功
        """
        try:
            with self._get_session() as session:
                task = session.get(Task, task_id)
                if not task:
                    logger.warning(f"任务不存在: ID={task_id}")
                    return False

                task.status = status
                task.updated_at = datetime.now()

                if error_message:
                    task.error_message = error_message

                # 根据状态设置时间戳
                if status == TaskStatus.RUNNING:
                    task.started_at = datetime.now()
                elif status in [TaskStatus.COMPLETED, TaskStatus.FAILED]:
                    task.completed_at = datetime.now()
                    if task.started_at:
                        execution_time = (
                            task.completed_at - task.started_at
                        ).total_seconds()
                        task.execution_time = execution_time

                session.add(task)
                session.commit()
                logger.info(f"任务状态更新成功: ID={task_id}, 状态={status}")
                return True

        except Exception as e:
            logger.error(f"更新任务状态失败: ID={task_id}, 错误: {e}")
            return False

    def update_result(self, task_id: int, result: dict[str, Any]) -> bool:
        """更新任务结果

        Args:
            task_id: 任务ID
            result: 任务结果数据

        Returns:
            是否更新成功
        """
        try:
            with self._get_session() as session:
                task = session.get(Task, task_id)
                if not task:
                    logger.warning(f"任务不存在: ID={task_id}")
                    return False

                task.result = result
                task.updated_at = datetime.now()

                session.add(task)
                session.commit()
                logger.info(f"任务结果更新成功: ID={task_id}")
                return True

        except Exception as e:
            logger.error(f"更新任务结果失败: ID={task_id}, 错误: {e}")
            return False

    def cancel_task(self, task_id: int) -> bool:
        """取消任务

        Args:
            task_id: 任务ID

        Returns:
            是否取消成功
        """
        try:
            with self._get_session() as session:
                task = session.get(Task, task_id)
                if not task:
                    logger.warning(f"任务不存在: ID={task_id}")
                    return False

                # 只有待执行和运行中的任务可以取消
                if task.status not in [TaskStatus.PENDING, TaskStatus.RUNNING]:
                    logger.warning(
                        f"任务状态不允许取消: ID={task_id}, 状态={task.status}"
                    )
                    return False

                task.status = TaskStatus.CANCELLED
                task.completed_at = datetime.now()
                task.updated_at = datetime.now()

                session.add(task)
                session.commit()
                logger.info(f"任务取消成功: ID={task_id}")
                return True

        except Exception as e:
            logger.error(f"取消任务失败: ID={task_id}, 错误: {e}")
            return False

    def get_task_statistics(self) -> dict[str, Any]:
        """获取任务统计信息

        Returns:
            任务统计数据
        """
        try:
            with self._get_session() as session:
                # 统计各状态的任务数量
                status_counts = {}
                for status in TaskStatus:
                    count_stmt = select(Task).where(Task.status == status)
                    count = len(session.exec(count_stmt).all())
                    status_counts[status.value] = count

                # 统计各类型的任务数量
                type_counts = {}
                for task_type in TaskType:
                    count_stmt = select(Task).where(Task.task_type == task_type)
                    count = len(session.exec(count_stmt).all())
                    type_counts[task_type.value] = count

                return {
                    "status_counts": status_counts,
                    "type_counts": type_counts,
                    "total_tasks": sum(status_counts.values()),
                }

        except Exception as e:
            logger.error(f"获取任务统计失败: 错误: {e}")
            return {
                "status_counts": {},
                "type_counts": {},
                "total_tasks": 0,
            }

    def create_task(self, task: Task) -> int | None:
        """创建任务

        Args:
            task: 任务对象

        Returns:
            创建的任务ID，失败返回None
        """
        try:
            with self._get_session() as session:
                session.add(task)
                session.commit()
                session.refresh(task)
                logger.info(f"任务创建成功: ID={task.id}, 名称={task.name}")
                return task.id

        except Exception as e:
            logger.error(f"创建任务失败: {e}")
            return None

    def get_pending_tasks_by_priority(self, limit: int = 10) -> list[Task]:
        """按优先级获取待执行任务

        Args:
            limit: 返回数量限制

        Returns:
            按优先级排序的待执行任务列表
        """
        try:
            with self._get_session() as session:
                from sqlalchemy import or_

                statement = (
                    select(Task)
                    .where(
                        and_(
                            Task.status == TaskStatus.PENDING,
                            or_(
                                Task.scheduled_at.is_(None),
                                Task.scheduled_at <= datetime.now(),
                            ),
                        )
                    )
                    .order_by(Task.priority, Task.created_at)
                    .limit(limit)
                )
                tasks = session.exec(statement).all()
                return list(tasks)

        except Exception as e:
            logger.error(f"获取待执行任务失败: {e}")
            return []

    def get_running_tasks_by_type(self, task_type: TaskType) -> list[Task]:
        """获取指定类型的运行中任务

        Args:
            task_type: 任务类型

        Returns:
            运行中的任务列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(Task)
                    .where(
                        and_(
                            Task.task_type == task_type,
                            Task.status.in_([TaskStatus.PENDING, TaskStatus.RUNNING])
                        )
                    )
                    .order_by(desc(Task.created_at))
                )
                tasks = session.exec(statement).all()
                return list(tasks)

        except Exception as e:
            logger.error(f"获取运行中任务失败: 任务类型={task_type}, 错误: {e}")
            return []

    async def update_task_progress(self, task_id: int, processed_records: int, total_records: int) -> bool:
        """更新任务进度

        Args:
            task_id: 任务ID
            processed_records: 已处理记录数
            total_records: 总记录数

        Returns:
            是否更新成功
        """
        try:
            with self._get_session() as session:
                statement = select(Task).where(Task.id == task_id)
                task = session.exec(statement).first()

                if not task:
                    logger.error(f"任务不存在: ID={task_id}")
                    return False

                # 更新进度信息
                task.progress = processed_records
                task.total_count = total_records
                task.updated_at = datetime.now()

                session.add(task)
                session.commit()

                logger.info(f"任务进度更新成功: ID={task_id}, 进度={processed_records}/{total_records}")
                return True

        except Exception as e:
            logger.error(f"更新任务进度失败: ID={task_id}, 错误: {e}")
            return False

    def update_task(self, task: Task) -> bool:
        """更新任务

        Args:
            task: 任务对象

        Returns:
            是否更新成功
        """
        try:
            with self._get_session() as session:
                # 合并任务对象到会话
                session.merge(task)
                session.commit()
                logger.debug(f"任务更新成功: ID={task.id}")
                return True

        except Exception as e:
            logger.error(f"更新任务失败: ID={task.id}, 错误: {e}")
            return False
