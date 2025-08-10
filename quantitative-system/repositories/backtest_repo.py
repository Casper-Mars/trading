"""回测数据访问层"""

import json
from datetime import datetime
from typing import Any

from loguru import logger
from sqlmodel import Session, and_, func, select

from config.database import get_session
from models.database import BacktestResult
from models.enums import BacktestStatus, StrategyType
from models.schemas import BacktestCreate, BacktestUpdate
from utils.exceptions import DatabaseError, NotFoundError


class BacktestRepo:
    """回测数据仓库

    提供回测任务和结果的数据访问功能
    """

    def __init__(self, session: Session | None = None):
        """初始化回测仓库

        Args:
            session: 数据库会话，如果为None则使用默认会话
        """
        self.session = session

    def _get_session(self) -> Session:
        """获取数据库会话"""
        if self.session:
            return self.session
        return next(get_session())

    def create(self, backtest_data: BacktestCreate) -> BacktestResult:
        """创建新回测任务

        Args:
            backtest_data: 回测创建数据

        Returns:
            创建的回测对象

        Raises:
            DatabaseError: 数据库操作失败
        """
        try:
            with self._get_session() as session:
                # 序列化策略参数和回测配置
                strategy_params_json = None
                if backtest_data.strategy_params:
                    strategy_params_json = json.dumps(
                        backtest_data.strategy_params, ensure_ascii=False, default=str
                    )

                backtest_config_json = None
                if backtest_data.backtest_config:
                    backtest_config_json = json.dumps(
                        backtest_data.backtest_config, ensure_ascii=False, default=str
                    )

                # 创建回测对象
                backtest = BacktestResult(
                    strategy_name=backtest_data.strategy_name,
                    strategy_type=backtest_data.strategy_type,
                    symbol=backtest_data.symbol,
                    start_date=backtest_data.start_date,
                    end_date=backtest_data.end_date,
                    initial_capital=backtest_data.initial_capital,
                    strategy_params=strategy_params_json,
                    backtest_config=backtest_config_json,
                    status=BacktestStatus.PENDING,
                )

                session.add(backtest)
                session.commit()
                session.refresh(backtest)

                logger.info(
                    f"创建回测任务成功: {backtest.strategy_name}, ID: {backtest.id}"
                )
                return backtest

        except Exception as e:
            logger.error(f"创建回测任务失败: {e}")
            raise DatabaseError(f"创建回测任务失败: {e}") from e

    def get_by_id(self, backtest_id: int) -> BacktestResult | None:
        """根据ID获取回测结果

        Args:
            backtest_id: 回测ID

        Returns:
            回测对象，不存在返回None
        """
        try:
            with self._get_session() as session:
                statement = select(BacktestResult).where(
                    BacktestResult.id == backtest_id
                )
                backtest = session.exec(statement).first()
                return backtest

        except Exception as e:
            logger.error(f"获取回测结果失败: ID={backtest_id}, 错误: {e}")
            return None

    def get_by_strategy_name(self, strategy_name: str) -> list[BacktestResult]:
        """根据策略名称获取回测结果列表

        Args:
            strategy_name: 策略名称

        Returns:
            回测结果列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(BacktestResult)
                    .where(BacktestResult.strategy_name == strategy_name)
                    .order_by(BacktestResult.created_at.desc())
                )
                backtests = session.exec(statement).all()
                return list(backtests)

        except Exception as e:
            logger.error(f"获取回测结果失败: strategy_name={strategy_name}, 错误: {e}")
            return []

    def get_by_symbol(self, symbol: str) -> list[BacktestResult]:
        """根据股票代码获取回测结果列表

        Args:
            symbol: 股票代码

        Returns:
            回测结果列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(BacktestResult)
                    .where(BacktestResult.symbol == symbol)
                    .order_by(BacktestResult.created_at.desc())
                )
                backtests = session.exec(statement).all()
                return list(backtests)

        except Exception as e:
            logger.error(f"获取回测结果失败: symbol={symbol}, 错误: {e}")
            return []

    def get_by_status(self, status: BacktestStatus) -> list[BacktestResult]:
        """根据状态获取回测结果列表

        Args:
            status: 回测状态

        Returns:
            回测结果列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(BacktestResult)
                    .where(BacktestResult.status == status)
                    .order_by(BacktestResult.created_at.desc())
                )
                backtests = session.exec(statement).all()
                return list(backtests)

        except Exception as e:
            logger.error(f"获取回测结果失败: status={status}, 错误: {e}")
            return []

    def get_paginated(
        self,
        page: int = 1,
        size: int = 20,
        status: BacktestStatus | None = None,
        strategy_type: StrategyType | None = None,
        symbol: str | None = None,
        strategy_name: str | None = None,
    ) -> tuple[list[BacktestResult], int]:
        """分页获取回测结果列表

        Args:
            page: 页码(从1开始)
            size: 每页大小
            status: 回测状态过滤
            strategy_type: 策略类型过滤
            symbol: 股票代码过滤
            strategy_name: 策略名称过滤

        Returns:
            (回测结果列表, 总数量)
        """
        try:
            with self._get_session() as session:
                # 构建查询条件
                conditions = []
                if status is not None:
                    conditions.append(BacktestResult.status == status)
                if strategy_type is not None:
                    conditions.append(BacktestResult.strategy_type == strategy_type)
                if symbol:
                    conditions.append(BacktestResult.symbol.contains(symbol))
                if strategy_name:
                    conditions.append(
                        BacktestResult.strategy_name.contains(strategy_name)
                    )

                # 查询总数
                count_statement = select(func.count(BacktestResult.id))
                if conditions:
                    count_statement = count_statement.where(and_(*conditions))
                total = session.exec(count_statement).one()

                # 分页查询
                statement = select(BacktestResult)
                if conditions:
                    statement = statement.where(and_(*conditions))

                statement = statement.order_by(BacktestResult.created_at.desc())
                statement = statement.offset((page - 1) * size).limit(size)

                backtests = session.exec(statement).all()
                return list(backtests), total

        except Exception as e:
            logger.error(f"分页获取回测结果失败: {e}")
            return [], 0

    def update(
        self, backtest_id: int, update_data: BacktestUpdate
    ) -> BacktestResult | None:
        """更新回测结果

        Args:
            backtest_id: 回测ID
            update_data: 更新数据

        Returns:
            更新后的回测对象

        Raises:
            NotFoundError: 回测不存在
            DatabaseError: 数据库操作失败
        """
        try:
            with self._get_session() as session:
                # 获取现有回测
                statement = select(BacktestResult).where(
                    BacktestResult.id == backtest_id
                )
                backtest = session.exec(statement).first()

                if not backtest:
                    raise NotFoundError(f"回测不存在: ID={backtest_id}")

                # 更新字段
                update_dict = update_data.model_dump(exclude_unset=True)
                for field, value in update_dict.items():
                    if hasattr(backtest, field) and value is not None:
                        # 特殊处理JSON字段
                        if field in [
                            "strategy_params",
                            "backtest_config",
                            "metrics",
                            "raw_data",
                        ]:
                            if isinstance(value, dict):
                                setattr(
                                    backtest,
                                    field,
                                    json.dumps(value, ensure_ascii=False, default=str),
                                )
                            else:
                                setattr(backtest, field, value)
                        else:
                            setattr(backtest, field, value)

                backtest.updated_at = datetime.utcnow()

                session.add(backtest)
                session.commit()
                session.refresh(backtest)

                logger.info(f"更新回测结果成功: ID={backtest_id}")
                return backtest

        except NotFoundError:
            raise
        except Exception as e:
            logger.error(f"更新回测结果失败: ID={backtest_id}, 错误: {e}")
            raise DatabaseError(f"更新回测结果失败: {e}") from e

    def update_status(
        self, backtest_id: int, status: BacktestStatus, error_message: str | None = None
    ) -> bool:
        """更新回测状态

        Args:
            backtest_id: 回测ID
            status: 新状态
            error_message: 错误信息(状态为FAILED时使用)

        Returns:
            是否更新成功
        """
        try:
            with self._get_session() as session:
                statement = select(BacktestResult).where(
                    BacktestResult.id == backtest_id
                )
                backtest = session.exec(statement).first()

                if not backtest:
                    logger.warning(f"回测不存在: ID={backtest_id}")
                    return False

                backtest.status = status
                if error_message:
                    backtest.error_message = error_message

                # 设置完成时间
                if status in [BacktestStatus.COMPLETED, BacktestStatus.FAILED]:
                    backtest.completed_at = datetime.utcnow()

                backtest.updated_at = datetime.utcnow()

                session.add(backtest)
                session.commit()

                logger.info(f"更新回测状态成功: ID={backtest_id}, 状态: {status}")
                return True

        except Exception as e:
            logger.error(f"更新回测状态失败: ID={backtest_id}, 错误: {e}")
            return False

    def save_results(
        self,
        backtest_id: int,
        metrics: dict[str, Any],
        raw_data: dict[str, Any] | None = None,
    ) -> bool:
        """保存回测结果数据

        Args:
            backtest_id: 回测ID
            metrics: 回测指标数据
            raw_data: 原始回测数据(可选)

        Returns:
            是否保存成功
        """
        try:
            with self._get_session() as session:
                statement = select(BacktestResult).where(
                    BacktestResult.id == backtest_id
                )
                backtest = session.exec(statement).first()

                if not backtest:
                    logger.warning(f"回测不存在: ID={backtest_id}")
                    return False

                # 保存指标数据
                backtest.metrics = json.dumps(metrics, ensure_ascii=False, default=str)

                # 保存原始数据(如果提供)
                if raw_data:
                    backtest.raw_data = json.dumps(
                        raw_data, ensure_ascii=False, default=str
                    )

                # 从指标中提取关键数据
                if "total_return" in metrics:
                    backtest.total_return = metrics["total_return"]
                if "sharpe_ratio" in metrics:
                    backtest.sharpe_ratio = metrics["sharpe_ratio"]
                if "max_drawdown" in metrics:
                    backtest.max_drawdown = metrics["max_drawdown"]
                if "win_rate" in metrics:
                    backtest.win_rate = metrics["win_rate"]

                backtest.updated_at = datetime.utcnow()

                session.add(backtest)
                session.commit()

                logger.info(f"保存回测结果成功: ID={backtest_id}")
                return True

        except Exception as e:
            logger.error(f"保存回测结果失败: ID={backtest_id}, 错误: {e}")
            return False

    def delete(self, backtest_id: int) -> bool:
        """删除回测结果

        Args:
            backtest_id: 回测ID

        Returns:
            是否删除成功

        Raises:
            NotFoundError: 回测不存在
            DatabaseError: 数据库操作失败
        """
        try:
            with self._get_session() as session:
                # 获取现有回测
                statement = select(BacktestResult).where(
                    BacktestResult.id == backtest_id
                )
                backtest = session.exec(statement).first()

                if not backtest:
                    raise NotFoundError(f"回测不存在: ID={backtest_id}")

                session.delete(backtest)
                session.commit()

                logger.info(f"删除回测结果成功: ID={backtest_id}")
                return True

        except NotFoundError:
            raise
        except Exception as e:
            logger.error(f"删除回测结果失败: ID={backtest_id}, 错误: {e}")
            raise DatabaseError(f"删除回测结果失败: {e}") from e

    def get_strategy_performance_summary(self, strategy_name: str) -> dict[str, Any]:
        """获取策略性能汇总

        Args:
            strategy_name: 策略名称

        Returns:
            策略性能汇总数据
        """
        try:
            with self._get_session() as session:
                # 获取已完成的回测统计
                statement = select(
                    func.count(BacktestResult.id).label("total_backtests"),
                    func.avg(BacktestResult.total_return).label("avg_return"),
                    func.avg(BacktestResult.sharpe_ratio).label("avg_sharpe"),
                    func.avg(BacktestResult.max_drawdown).label("avg_drawdown"),
                    func.avg(BacktestResult.win_rate).label("avg_win_rate"),
                    func.max(BacktestResult.total_return).label("best_return"),
                    func.min(BacktestResult.total_return).label("worst_return"),
                ).where(
                    and_(
                        BacktestResult.strategy_name == strategy_name,
                        BacktestResult.status == BacktestStatus.COMPLETED,
                    )
                )

                result = session.exec(statement).first()

                return {
                    "strategy_name": strategy_name,
                    "total_backtests": result.total_backtests or 0,
                    "avg_return": float(result.avg_return or 0),
                    "avg_sharpe_ratio": float(result.avg_sharpe or 0),
                    "avg_max_drawdown": float(result.avg_drawdown or 0),
                    "avg_win_rate": float(result.avg_win_rate or 0),
                    "best_return": float(result.best_return or 0),
                    "worst_return": float(result.worst_return or 0),
                }

        except Exception as e:
            logger.error(
                f"获取策略性能汇总失败: strategy_name={strategy_name}, 错误: {e}"
            )
            return {
                "strategy_name": strategy_name,
                "total_backtests": 0,
                "avg_return": 0.0,
                "avg_sharpe_ratio": 0.0,
                "avg_max_drawdown": 0.0,
                "avg_win_rate": 0.0,
                "best_return": 0.0,
                "worst_return": 0.0,
            }

    def get_pending_backtests(self) -> list[BacktestResult]:
        """获取待执行的回测任务

        Returns:
            待执行回测任务列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(BacktestResult)
                    .where(BacktestResult.status == BacktestStatus.PENDING)
                    .order_by(BacktestResult.created_at.asc())
                )
                backtests = session.exec(statement).all()
                return list(backtests)

        except Exception as e:
            logger.error(f"获取待执行回测任务失败: {e}")
            return []

    def get_running_backtests(self) -> list[BacktestResult]:
        """获取正在运行的回测任务

        Returns:
            正在运行的回测任务列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(BacktestResult)
                    .where(BacktestResult.status == BacktestStatus.RUNNING)
                    .order_by(BacktestResult.created_at.asc())
                )
                backtests = session.exec(statement).all()
                return list(backtests)

        except Exception as e:
            logger.error(f"获取正在运行回测任务失败: {e}")
            return []


# 全局回测仓库实例
backtest_repo = BacktestRepo()
