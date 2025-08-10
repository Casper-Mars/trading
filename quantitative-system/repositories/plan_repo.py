"""交易方案数据访问层"""

import json
from datetime import date, datetime
from typing import Any

from loguru import logger
from sqlmodel import Session, and_, col, desc, func, or_, select

from config.database import get_session
from models.database import TradingPlan
from models.enums import PlanStatus, PlanType
from models.schemas import TradingPlanCreate, TradingPlanUpdate
from utils.exceptions import DatabaseError, NotFoundError


class PlanRepo:
    """交易方案数据仓库

    提供交易方案的数据访问功能，支持按日期查询、历史分页等
    """

    def __init__(self, session: Session | None = None):
        """初始化方案仓库

        Args:
            session: 数据库会话，如果为None则使用默认会话
        """
        self.session = session

    def _get_session(self) -> Session:
        """获取数据库会话"""
        if self.session:
            return self.session
        return next(get_session())

    def create(self, plan_data: TradingPlanCreate) -> TradingPlan:
        """创建新交易方案

        Args:
            plan_data: 方案创建数据

        Returns:
            创建的方案对象

        Raises:
            DatabaseError: 数据库操作失败
        """
        try:
            with self._get_session() as session:
                # 创建方案对象
                plan = TradingPlan(
                    plan_date=plan_data.plan_date,
                    title=plan_data.title,
                    content=plan_data.content,
                    plan_type=plan_data.plan_type,
                    risk_level=plan_data.risk_level,
                    target_return=plan_data.target_return,
                    max_drawdown_limit=plan_data.max_drawdown_limit,
                    position_limit=plan_data.position_limit,
                    recommendations=plan_data.recommendations,
                    backtest_results=plan_data.backtest_results,
                    ai_analysis=plan_data.ai_analysis,
                    notes=plan_data.notes,
                )

                session.add(plan)
                session.commit()
                session.refresh(plan)

                logger.info(f"创建交易方案成功: {plan.title}, ID: {plan.id}")
                return plan

        except Exception as e:
            logger.error(f"创建交易方案失败: {e}")
            raise DatabaseError(f"创建交易方案失败: {e}") from e

    def get_by_id(self, plan_id: int) -> TradingPlan | None:
        """根据ID获取交易方案

        Args:
            plan_id: 方案ID

        Returns:
            方案对象，不存在返回None
        """
        try:
            with self._get_session() as session:
                statement = select(TradingPlan).where(TradingPlan.id == plan_id)
                plan = session.exec(statement).first()
                return plan

        except Exception as e:
            logger.error(f"获取交易方案失败: ID={plan_id}, 错误: {e}")
            return None

    def get_by_date(self, plan_date: date) -> TradingPlan | None:
        """根据日期获取交易方案

        Args:
            plan_date: 方案日期

        Returns:
            方案对象，不存在返回None
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(TradingPlan)
                    .where(TradingPlan.plan_date == plan_date)
                    .order_by(desc(TradingPlan.created_at))
                )
                plan = session.exec(statement).first()
                return plan

        except Exception as e:
            logger.error(f"获取交易方案失败: date={plan_date}, 错误: {e}")
            return None

    def get_today_plan(self) -> TradingPlan | None:
        """获取今日交易方案

        Returns:
            今日方案对象，不存在返回None
        """
        today = date.today()
        return self.get_by_date(today)

    def get_latest_plan(self) -> TradingPlan | None:
        """获取最新的交易方案

        Returns:
            最新方案对象，不存在返回None
        """
        try:
            with self._get_session() as session:
                statement = select(TradingPlan).order_by(
                    desc(TradingPlan.plan_date), desc(TradingPlan.created_at)
                )
                plan = session.exec(statement).first()
                return plan

        except Exception as e:
            logger.error(f"获取最新交易方案失败: {e}")
            return None

    def get_by_date_range(self, start_date: date, end_date: date) -> list[TradingPlan]:
        """根据日期范围获取交易方案列表

        Args:
            start_date: 开始日期
            end_date: 结束日期

        Returns:
            方案列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(TradingPlan)
                    .where(
                        and_(
                            TradingPlan.plan_date >= start_date,
                            TradingPlan.plan_date <= end_date,
                        )
                    )
                    .order_by(desc(TradingPlan.plan_date))
                )
                plans = session.exec(statement).all()
                return list(plans)

        except Exception as e:
            logger.error(f"获取日期范围方案失败: {start_date} - {end_date}, 错误: {e}")
            return []

    def get_recent_plans(self, days: int = 7) -> list[TradingPlan]:
        """获取最近N天的交易方案

        Args:
            days: 天数

        Returns:
            方案列表
        """
        try:
            from datetime import timedelta

            end_date = date.today()
            start_date = end_date - timedelta(days=days)
            return self.get_by_date_range(start_date, end_date)

        except Exception as e:
            logger.error(f"获取最近{days}天方案失败: {e}")
            return []

    def get_by_status(self, status: PlanStatus) -> list[TradingPlan]:
        """根据状态获取交易方案列表

        Args:
            status: 方案状态

        Returns:
            方案列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(TradingPlan)
                    .where(TradingPlan.status == status)
                    .order_by(desc(TradingPlan.plan_date))
                )
                plans = session.exec(statement).all()
                return list(plans)

        except Exception as e:
            logger.error(f"获取方案失败: status={status}, 错误: {e}")
            return []

    def get_by_type(self, plan_type: PlanType) -> list[TradingPlan]:
        """根据类型获取交易方案列表

        Args:
            plan_type: 方案类型

        Returns:
            方案列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(TradingPlan)
                    .where(TradingPlan.plan_type == plan_type)
                    .order_by(desc(TradingPlan.plan_date))
                )
                plans = session.exec(statement).all()
                return list(plans)

        except Exception as e:
            logger.error(f"获取方案失败: type={plan_type}, 错误: {e}")
            return []

    def get_paginated(
        self,
        page: int = 1,
        size: int = 20,
        status: PlanStatus | None = None,
        plan_type: PlanType | None = None,
        start_date: date | None = None,
        end_date: date | None = None,
        title_keyword: str | None = None,
    ) -> tuple[list[TradingPlan], int]:
        """分页获取交易方案列表

        Args:
            page: 页码(从1开始)
            size: 每页大小
            status: 方案状态过滤
            plan_type: 方案类型过滤
            start_date: 开始日期过滤
            end_date: 结束日期过滤
            title_keyword: 标题关键词过滤

        Returns:
            (方案列表, 总数量)
        """
        try:
            with self._get_session() as session:
                # 构建查询条件
                conditions = []
                if status is not None:
                    conditions.append(col(TradingPlan.status) == status)
                if plan_type is not None:
                    conditions.append(col(TradingPlan.plan_type) == plan_type)
                if start_date is not None:
                    conditions.append(col(TradingPlan.plan_date) >= start_date)
                if end_date is not None:
                    conditions.append(col(TradingPlan.plan_date) <= end_date)
                if title_keyword:
                    conditions.append(col(TradingPlan.title).contains(title_keyword))

                # 查询总数
                count_statement = select(func.count())
                if conditions:
                    count_statement = count_statement.where(and_(*conditions))
                total = session.exec(count_statement).one()

                # 分页查询
                statement = select(TradingPlan)
                if conditions:
                    statement = statement.where(and_(*conditions))

                statement = statement.order_by(desc(TradingPlan.plan_date))
                statement = statement.offset((page - 1) * size).limit(size)

                plans = session.exec(statement).all()
                return list(plans), total

        except Exception as e:
            logger.error(f"分页获取交易方案失败: {e}")
            return [], 0

    def update(
        self, plan_id: int, update_data: TradingPlanUpdate
    ) -> TradingPlan | None:
        """更新交易方案

        Args:
            plan_id: 方案ID
            update_data: 更新数据

        Returns:
            更新后的方案对象

        Raises:
            NotFoundError: 方案不存在
            DatabaseError: 数据库操作失败
        """
        try:
            with self._get_session() as session:
                # 获取现有方案
                statement = select(TradingPlan).where(TradingPlan.id == plan_id)
                plan = session.exec(statement).first()

                if not plan:
                    raise NotFoundError(f"交易方案不存在: ID={plan_id}")

                # 更新字段
                update_dict = update_data.model_dump(exclude_unset=True)
                for field, value in update_dict.items():
                    if hasattr(plan, field) and value is not None:
                        # 特殊处理JSON字段
                        if field in ["analysis_data", "config"]:
                            if isinstance(value, dict):
                                setattr(
                                    plan,
                                    field,
                                    json.dumps(value, ensure_ascii=False, default=str),
                                )
                            else:
                                setattr(plan, field, value)
                        else:
                            setattr(plan, field, value)

                plan.updated_at = datetime.utcnow()

                session.add(plan)
                session.commit()
                session.refresh(plan)

                logger.info(f"更新交易方案成功: ID={plan_id}")
                return plan

        except NotFoundError:
            raise
        except Exception as e:
            logger.error(f"更新交易方案失败: ID={plan_id}, 错误: {e}")
            raise DatabaseError(f"更新交易方案失败: {e}") from e

    def update_status(self, plan_id: int, status: PlanStatus) -> bool:
        """更新方案状态

        Args:
            plan_id: 方案ID
            status: 新状态

        Returns:
            是否更新成功
        """
        try:
            with self._get_session() as session:
                statement = select(TradingPlan).where(TradingPlan.id == plan_id)
                plan = session.exec(statement).first()

                if not plan:
                    logger.warning(f"交易方案不存在: ID={plan_id}")
                    return False

                plan.status = status
                plan.updated_at = datetime.utcnow()

                session.add(plan)
                session.commit()

                logger.info(f"更新方案状态成功: ID={plan_id}, 状态: {status}")
                return True

        except Exception as e:
            logger.error(f"更新方案状态失败: ID={plan_id}, 错误: {e}")
            return False

    def delete(self, plan_id: int) -> bool:
        """删除交易方案

        Args:
            plan_id: 方案ID

        Returns:
            是否删除成功

        Raises:
            NotFoundError: 方案不存在
            DatabaseError: 数据库操作失败
        """
        try:
            with self._get_session() as session:
                # 获取现有方案
                statement = select(TradingPlan).where(TradingPlan.id == plan_id)
                plan = session.exec(statement).first()

                if not plan:
                    raise NotFoundError(f"交易方案不存在: ID={plan_id}")

                session.delete(plan)
                session.commit()

                logger.info(f"删除交易方案成功: ID={plan_id}")
                return True

        except NotFoundError:
            raise
        except Exception as e:
            logger.error(f"删除交易方案失败: ID={plan_id}, 错误: {e}")
            raise DatabaseError(f"删除交易方案失败: {e}") from e

    def get_plan_statistics(self) -> dict[str, Any]:
        """获取方案统计信息

        Returns:
            方案统计数据
        """
        try:
            with self._get_session() as session:
                # 总体统计
                total_statement = select(
                    func.count().label("total_plans"),
                )
                total_result = session.exec(total_statement).first()

                # 按状态统计
                status_statement = select(
                    TradingPlan.status, func.count().label("count")
                ).group_by(col(TradingPlan.status))
                status_results = session.exec(status_statement).all()

                # 按类型统计
                type_statement = select(
                    TradingPlan.plan_type, func.count().label("count")
                ).group_by(col(TradingPlan.plan_type))
                type_results = session.exec(type_statement).all()

                # 最近30天统计
                from datetime import timedelta

                thirty_days_ago = date.today() - timedelta(days=30)
                recent_statement = select(func.count().label("recent_count")).where(
                    TradingPlan.plan_date >= thirty_days_ago
                )
                recent_result = session.exec(recent_statement).first()

                # 添加空值检查
                if total_result is None:
                    total_plans = 0
                    avg_confidence = 0.0
                else:
                    total_plans = total_result or 0
                    avg_confidence = 0.0  # 移除了confidence_score字段

                recent_count = 0 if recent_result is None else recent_result or 0

                return {
                    "total_plans": total_plans,
                    "avg_confidence_score": avg_confidence,
                    "recent_30_days": recent_count,
                    "status_distribution": {
                        result[0]: result[1] for result in status_results
                    },
                    "type_distribution": {
                        result[0]: result[1] for result in type_results
                    },
                }

        except Exception as e:
            logger.error(f"获取方案统计失败: {e}")
            return {
                "total_plans": 0,
                "avg_confidence_score": 0.0,
                "recent_30_days": 0,
                "status_distribution": {},
                "type_distribution": {},
            }

    def search_by_content(self, keyword: str, limit: int = 10) -> list[TradingPlan]:
        """根据内容关键词搜索方案

        Args:
            keyword: 搜索关键词
            limit: 结果数量限制

        Returns:
            匹配的方案列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(TradingPlan)
                    .where(
                        or_(
                            col(TradingPlan.title).like(f"%{keyword}%"),
                            col(TradingPlan.content).like(f"%{keyword}%"),
                            func.json_extract(
                                col(TradingPlan.recommendations), "$"
                            ).like(f"%{keyword}%"),
                        )
                    )
                    .order_by(desc(TradingPlan.plan_date))
                    .limit(limit)
                )

                plans = session.exec(statement).all()
                return list(plans)

        except Exception as e:
            logger.error(f"搜索方案失败: keyword={keyword}, 错误: {e}")
            return []

    def get_plans_by_tags(self, tags: list[str]) -> list[TradingPlan]:
        """根据标签获取方案列表

        Args:
            tags: 标签列表

        Returns:
            匹配的方案列表
        """
        try:
            with self._get_session() as session:
                conditions = []
                for tag in tags:
                    conditions.append(
                        func.json_extract(TradingPlan.recommendations, "$").like(
                            f"%{tag}%"
                        )
                    )

                statement = (
                    select(TradingPlan)
                    .where(or_(*conditions))
                    .order_by(desc(TradingPlan.plan_date))
                )

                plans = session.exec(statement).all()
                return list(plans)

        except Exception as e:
            logger.error(f"根据标签获取方案失败: tags={tags}, 错误: {e}")
            return []


# 全局方案仓库实例
plan_repo = PlanRepo()
