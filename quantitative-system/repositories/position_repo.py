"""持仓数据访问层"""

from datetime import datetime
from decimal import Decimal
from typing import Any

from loguru import logger
from sqlmodel import Session, and_, col, desc, func, select

from config.database import get_session
from models.database import Position
from models.enums import PositionStatus, PositionType
from models.schemas import PositionCreate, PositionUpdate
from utils.exceptions import DatabaseError, NotFoundError


class PositionRepo:
    """持仓数据仓库

    提供持仓数据的增删改查和聚合统计功能
    """

    def __init__(self, session: Session | None = None):
        """初始化持仓仓库

        Args:
            session: 数据库会话，如果为None则使用默认会话
        """
        self.session = session

    def _get_session(self) -> Session:
        """获取数据库会话"""
        if self.session:
            return self.session
        return next(get_session())

    def create(self, position_data: PositionCreate) -> Position:
        """创建新持仓

        Args:
            position_data: 持仓创建数据

        Returns:
            创建的持仓对象

        Raises:
            DatabaseError: 数据库操作失败
        """
        try:
            with self._get_session() as session:
                # 创建持仓对象
                position = Position(
                    symbol=position_data.symbol,
                    name=position_data.name,
                    position_type=position_data.position_type,
                    quantity=position_data.quantity,
                    avg_price=position_data.avg_cost,
                    current_price=position_data.avg_cost,  # 初始设为成本价
                    market_value=position_data.quantity * position_data.avg_cost,
                    unrealized_pnl=Decimal("0"),  # 初始无浮动盈亏
                    realized_pnl=Decimal("0"),
                    status=PositionStatus.ACTIVE,
                    open_date=position_data.open_date,
                    notes=position_data.notes,
                )

                session.add(position)
                session.commit()
                session.refresh(position)

                logger.info(
                    f"创建持仓成功: {position.symbol}, 数量: {position.quantity}"
                )
                return position

        except Exception as e:
            logger.error(f"创建持仓失败: {e}")
            raise DatabaseError(f"创建持仓失败: {e}") from e

    def get_by_id(self, position_id: int) -> Position | None:
        """根据ID获取持仓

        Args:
            position_id: 持仓ID

        Returns:
            持仓对象，不存在返回None
        """
        try:
            with self._get_session() as session:
                statement = select(Position).where(Position.id == position_id)
                position = session.exec(statement).first()
                return position

        except Exception as e:
            logger.error(f"获取持仓失败: ID={position_id}, 错误: {e}")
            return None

    def get_by_symbol(self, symbol: str) -> list[Position]:
        """根据股票代码获取持仓列表

        Args:
            symbol: 股票代码

        Returns:
            持仓列表
        """
        try:
            with self._get_session() as session:
                statement = select(Position).where(Position.symbol == symbol)
                positions = session.exec(statement).all()
                return list(positions)

        except Exception as e:
            logger.error(f"获取持仓失败: symbol={symbol}, 错误: {e}")
            return []

    def get_active_positions(self) -> list[Position]:
        """获取所有活跃持仓

        Returns:
            活跃持仓列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(Position)
                    .where(Position.status == PositionStatus.ACTIVE)
                    .order_by(desc(Position.updated_at))
                )
                positions = session.exec(statement).all()
                return list(positions)

        except Exception as e:
            logger.error(f"获取活跃持仓失败: {e}")
            return []

    def get_positions_by_type(self, position_type: PositionType) -> list[Position]:
        """根据持仓类型获取持仓列表

        Args:
            position_type: 持仓类型

        Returns:
            持仓列表
        """
        try:
            with self._get_session() as session:
                statement = (
                    select(Position)
                    .where(Position.position_type == position_type)
                    .order_by(desc(Position.updated_at))
                )
                positions = session.exec(statement).all()
                return list(positions)

        except Exception as e:
            logger.error(f"获取持仓失败: type={position_type}, 错误: {e}")
            return []

    def get_paginated(
        self,
        page: int = 1,
        size: int = 20,
        status: PositionStatus | None = None,
        position_type: PositionType | None = None,
        symbol: str | None = None,
    ) -> tuple[list[Position], int]:
        """分页获取持仓列表

        Args:
            page: 页码(从1开始)
            size: 每页大小
            status: 持仓状态过滤
            position_type: 持仓类型过滤
            symbol: 股票代码过滤

        Returns:
            (持仓列表, 总数量)
        """
        try:
            with self._get_session() as session:
                # 构建查询条件
                conditions = []
                if status is not None:
                    conditions.append(col(Position.status) == status)
                if position_type is not None:
                    conditions.append(col(Position.position_type) == position_type)
                if symbol:
                    conditions.append(col(Position.symbol).contains(symbol))

                # 查询总数
                count_statement = select(func.count())
                if conditions:
                    count_statement = count_statement.where(and_(*conditions))
                total = session.exec(count_statement).one()

                # 分页查询
                statement = select(Position)
                if conditions:
                    statement = statement.where(and_(*conditions))

                statement = statement.order_by(desc(Position.updated_at))
                statement = statement.offset((page - 1) * size).limit(size)

                positions = session.exec(statement).all()
                return list(positions), total

        except Exception as e:
            logger.error(f"分页获取持仓失败: {e}")
            return [], 0

    def update(self, position_id: int, update_data: PositionUpdate) -> Position | None:
        """更新持仓

        Args:
            position_id: 持仓ID
            update_data: 更新数据

        Returns:
            更新后的持仓对象

        Raises:
            NotFoundError: 持仓不存在
            DatabaseError: 数据库操作失败
        """
        try:
            with self._get_session() as session:
                # 获取现有持仓
                statement = select(Position).where(Position.id == position_id)
                position = session.exec(statement).first()

                if not position:
                    raise NotFoundError(f"持仓不存在: ID={position_id}")

                # 更新字段
                update_dict = update_data.model_dump(exclude_unset=True)
                for field, value in update_dict.items():
                    if hasattr(position, field) and value is not None:
                        setattr(position, field, value)

                # 重新计算市值和未实现盈亏
                if (
                    update_data.quantity is not None
                    or update_data.current_price is not None
                ) and position.current_price is not None:
                    position.market_value = position.quantity * position.current_price
                    position.unrealized_pnl = position.quantity * (
                        position.current_price - position.avg_cost
                    )

                position.updated_at = datetime.utcnow()

                session.add(position)
                session.commit()
                session.refresh(position)

                logger.info(f"更新持仓成功: ID={position_id}")
                return position

        except NotFoundError:
            raise
        except Exception as e:
            logger.error(f"更新持仓失败: ID={position_id}, 错误: {e}")
            raise DatabaseError(f"更新持仓失败: {e}") from e

    def delete(self, position_id: int) -> bool:
        """删除持仓

        Args:
            position_id: 持仓ID

        Returns:
            是否删除成功

        Raises:
            NotFoundError: 持仓不存在
            DatabaseError: 数据库操作失败
        """
        try:
            with self._get_session() as session:
                # 获取现有持仓
                statement = select(Position).where(Position.id == position_id)
                position = session.exec(statement).first()

                if not position:
                    raise NotFoundError(f"持仓不存在: ID={position_id}")

                session.delete(position)
                session.commit()

                logger.info(f"删除持仓成功: ID={position_id}")
                return True

        except NotFoundError:
            raise
        except Exception as e:
            logger.error(f"删除持仓失败: ID={position_id}, 错误: {e}")
            raise DatabaseError(f"删除持仓失败: {e}") from e

    def get_portfolio_summary(self) -> dict[str, Any]:
        """获取投资组合汇总信息

        Returns:
            投资组合汇总数据
        """
        try:
            with self._get_session() as session:
                # 获取活跃持仓统计
                active_statement = select(
                    func.count().label("total_positions"),
                    func.sum(Position.market_value).label("total_market_value"),
                    func.sum(Position.unrealized_pnl).label("total_unrealized_pnl"),
                    func.sum(Position.realized_pnl).label("total_realized_pnl"),
                ).where(Position.status == PositionStatus.ACTIVE)

                result = session.exec(active_statement).first()
                if not result:
                    return {
                        "total_positions": 0,
                        "total_market_value": 0.0,
                        "total_unrealized_pnl": 0.0,
                        "total_realized_pnl": 0.0,
                        "total_pnl": 0.0,
                        "long_positions": {"count": 0, "market_value": 0.0},
                        "short_positions": {"count": 0, "market_value": 0.0},
                    }

                # 获取多头和空头统计
                long_statement = select(
                    func.count().label("long_count"),
                    func.sum(Position.market_value).label("long_market_value"),
                ).where(
                    and_(
                        Position.status == PositionStatus.ACTIVE,
                        Position.position_type == PositionType.LONG,
                    )
                )
                long_result = session.exec(long_statement).first()

                short_statement = select(
                    func.count().label("short_count"),
                    func.sum(Position.market_value).label("short_market_value"),
                ).where(
                    and_(
                        Position.status == PositionStatus.ACTIVE,
                        Position.position_type == PositionType.SHORT,
                    )
                )
                short_result = session.exec(short_statement).first()

                # 计算总盈亏
                total_pnl = (result[2] or Decimal("0")) + (result[3] or Decimal("0"))

                return {
                    "total_positions": result[0] or 0,
                    "total_market_value": float(result[1] or Decimal("0")),
                    "total_unrealized_pnl": float(result[2] or Decimal("0")),
                    "total_realized_pnl": float(result[3] or Decimal("0")),
                    "total_pnl": float(total_pnl),
                    "long_positions": {
                        "count": long_result[0] if long_result else 0,
                        "market_value": float(long_result[1] or Decimal("0"))
                        if long_result
                        else 0.0,
                    },
                    "short_positions": {
                        "count": short_result[0] if short_result else 0,
                        "market_value": float(short_result[1] or Decimal("0"))
                        if short_result
                        else 0.0,
                    },
                }

        except Exception as e:
            logger.error(f"获取投资组合汇总失败: {e}")
            return {
                "total_positions": 0,
                "total_market_value": 0.0,
                "total_unrealized_pnl": 0.0,
                "total_realized_pnl": 0.0,
                "total_pnl": 0.0,
                "long_positions": {"count": 0, "market_value": 0.0},
                "short_positions": {"count": 0, "market_value": 0.0},
            }

    def get_positions_by_symbols(self, symbols: list[str]) -> list[Position]:
        """根据股票代码列表获取持仓

        Args:
            symbols: 股票代码列表

        Returns:
            持仓列表
        """
        try:
            with self._get_session() as session:
                statement = select(Position).where(
                    and_(
                        col(Position.symbol).in_(symbols),
                        Position.status == PositionStatus.ACTIVE,
                    )
                )
                positions = session.exec(statement).all()
                return list(positions)

        except Exception as e:
            logger.error(f"批量获取持仓失败: symbols={symbols}, 错误: {e}")
            return []

    def update_current_prices(self, price_updates: dict[str, Decimal]) -> int:
        """批量更新当前价格

        Args:
            price_updates: {symbol: current_price} 的字典

        Returns:
            更新的持仓数量
        """
        try:
            updated_count = 0
            with self._get_session() as session:
                for symbol, current_price in price_updates.items():
                    statement = select(Position).where(
                        and_(
                            Position.symbol == symbol,
                            Position.status == PositionStatus.ACTIVE,
                        )
                    )
                    positions = session.exec(statement).all()

                    for position in positions:
                        position.current_price = current_price
                        position.market_value = position.quantity * current_price
                        position.unrealized_pnl = position.quantity * (
                            current_price - position.avg_cost
                        )
                        position.updated_at = datetime.utcnow()
                        session.add(position)
                        updated_count += 1

                session.commit()
                logger.info(f"批量更新价格成功: 更新了{updated_count}个持仓")
                return updated_count

        except Exception as e:
            logger.error(f"批量更新价格失败: {e}")
            return 0


# 全局持仓仓库实例
position_repo = PositionRepo()
