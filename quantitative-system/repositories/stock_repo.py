"""股票数据仓库模块

提供股票相关数据的持久化操作，包括股票基础信息、日线数据、财务数据的CRUD操作。
支持批量操作、分页查询、数据统计等功能。
"""

from datetime import date, datetime
from typing import Any

from loguru import logger
from sqlalchemy import and_, desc, func, select
from sqlalchemy.ext.asyncio import AsyncSession

from models.database import FinancialData, StockBasicInfo, StockDailyData
from repositories.base_repo import BaseRepository
from utils.exceptions import DatabaseError, NotFoundError


class StockRepository(BaseRepository):
    """股票数据仓库

    提供股票基础信息、日线数据、财务数据的数据访问操作。
    包含数据查询、创建、更新、删除和统计功能。
    """

    def __init__(self, session: AsyncSession):
        """初始化股票数据仓库

        Args:
            session: 数据库会话
        """
        super().__init__(session)
        logger.debug("股票数据仓库初始化完成")

    # ==================== 股票基础信息操作 ====================

    async def get_stock_basic_info(self, ts_code: str) -> StockBasicInfo | None:
        """获取股票基础信息

        Args:
            ts_code: 股票代码

        Returns:
            股票基础信息，不存在时返回None

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            stmt = select(StockBasicInfo).where(StockBasicInfo.ts_code == ts_code)
            result = await self.session.execute(stmt)
            return result.scalar_one_or_none()
        except Exception as e:
            error_msg = f"获取股票基础信息失败: {ts_code}, 错误: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    async def create_stock_basic_info(
        self, stock_data: StockBasicInfo
    ) -> StockBasicInfo:
        """创建股票基础信息

        Args:
            stock_data: 股票基础信息

        Returns:
            创建的股票基础信息

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            self.session.add(stock_data)
            await self.session.commit()
            await self.session.refresh(stock_data)
            logger.debug(f"创建股票基础信息成功: {stock_data.ts_code}")
            return stock_data
        except Exception as e:
            await self.session.rollback()
            error_msg = f"创建股票基础信息失败: {stock_data.ts_code}, 错误: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    async def update_stock_basic_info(
        self, ts_code: str, update_data: dict[str, Any]
    ) -> StockBasicInfo:
        """更新股票基础信息

        Args:
            ts_code: 股票代码
            update_data: 更新数据

        Returns:
            更新后的股票基础信息

        Raises:
            NotFoundError: 股票不存在时
            DatabaseError: 数据库操作失败时
        """
        try:
            stock = await self.get_stock_basic_info(ts_code)
            if not stock:
                raise NotFoundError(f"股票不存在: {ts_code}")

            for key, value in update_data.items():
                if hasattr(stock, key):
                    setattr(stock, key, value)

            stock.updated_at = datetime.now()
            await self.session.commit()
            await self.session.refresh(stock)
            logger.debug(f"更新股票基础信息成功: {ts_code}")
            return stock
        except NotFoundError:
            raise
        except Exception as e:
            await self.session.rollback()
            error_msg = f"更新股票基础信息失败: {ts_code}, 错误: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    async def get_all_stock_codes(self, list_status: str = "L") -> list[StockBasicInfo]:
        """获取所有股票代码

        Args:
            list_status: 上市状态 L上市 D退市 P暂停上市

        Returns:
            股票基础信息列表

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            stmt = select(StockBasicInfo).where(
                StockBasicInfo.list_status == list_status
            )
            result = await self.session.execute(stmt)
            return result.scalars().all()
        except Exception as e:
            error_msg = f"获取股票代码列表失败: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    async def get_stock_count(self, list_status: str | None = None) -> int:
        """获取股票数量

        Args:
            list_status: 上市状态，为None时统计所有状态

        Returns:
            股票数量

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            stmt = select(func.count(StockBasicInfo.ts_code))
            if list_status:
                stmt = stmt.where(StockBasicInfo.list_status == list_status)
            result = await self.session.execute(stmt)
            return result.scalar() or 0
        except Exception as e:
            error_msg = f"获取股票数量失败: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    # ==================== 股票日线数据操作 ====================

    async def get_daily_data(
        self, ts_code: str, trade_date: str | date
    ) -> StockDailyData | None:
        """获取股票日线数据

        Args:
            ts_code: 股票代码
            trade_date: 交易日期

        Returns:
            股票日线数据，不存在时返回None

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            if isinstance(trade_date, str):
                trade_date = datetime.strptime(trade_date, "%Y%m%d").date()

            stmt = select(StockDailyData).where(
                and_(
                    StockDailyData.ts_code == ts_code,
                    StockDailyData.trade_date == trade_date,
                )
            )
            result = await self.session.execute(stmt)
            return result.scalar_one_or_none()
        except Exception as e:
            error_msg = f"获取股票日线数据失败: {ts_code} {trade_date}, 错误: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    async def get_daily_data_range(
        self,
        ts_code: str,
        start_date: str | date | None = None,
        end_date: str | date | None = None,
        limit: int | None = None,
    ) -> list[StockDailyData]:
        """获取股票日线数据范围

        Args:
            ts_code: 股票代码
            start_date: 开始日期
            end_date: 结束日期
            limit: 限制条数

        Returns:
            股票日线数据列表

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            stmt = select(StockDailyData).where(StockDailyData.ts_code == ts_code)

            if start_date:
                if isinstance(start_date, str):
                    start_date = datetime.strptime(start_date, "%Y%m%d").date()
                stmt = stmt.where(StockDailyData.trade_date >= start_date)

            if end_date:
                if isinstance(end_date, str):
                    end_date = datetime.strptime(end_date, "%Y%m%d").date()
                stmt = stmt.where(StockDailyData.trade_date <= end_date)

            stmt = stmt.order_by(desc(StockDailyData.trade_date))

            if limit:
                stmt = stmt.limit(limit)

            result = await self.session.execute(stmt)
            return result.scalars().all()
        except Exception as e:
            error_msg = f"获取股票日线数据范围失败: {ts_code}, 错误: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    async def batch_create_daily_data(
        self, daily_data_list: list[StockDailyData]
    ) -> int:
        """批量创建股票日线数据

        Args:
            daily_data_list: 股票日线数据列表

        Returns:
            创建的数据条数

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            if not daily_data_list:
                return 0

            self.session.add_all(daily_data_list)
            await self.session.commit()
            count = len(daily_data_list)
            logger.debug(f"批量创建股票日线数据成功: {count} 条")
            return count
        except Exception as e:
            await self.session.rollback()
            error_msg = f"批量创建股票日线数据失败: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    async def get_last_daily_data_date(self, ts_code: str | None = None) -> date | None:
        """获取最后的日线数据日期

        Args:
            ts_code: 股票代码，为None时获取所有股票的最后日期

        Returns:
            最后的交易日期，无数据时返回None

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            stmt = select(func.max(StockDailyData.trade_date))
            if ts_code:
                stmt = stmt.where(StockDailyData.ts_code == ts_code)

            result = await self.session.execute(stmt)
            return result.scalar()
        except Exception as e:
            error_msg = f"获取最后日线数据日期失败: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    async def get_daily_data_count(self, ts_code: str | None = None) -> int:
        """获取日线数据数量

        Args:
            ts_code: 股票代码，为None时统计所有股票

        Returns:
            日线数据数量

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            stmt = select(func.count(StockDailyData.id))
            if ts_code:
                stmt = stmt.where(StockDailyData.ts_code == ts_code)

            result = await self.session.execute(stmt)
            return result.scalar() or 0
        except Exception as e:
            error_msg = f"获取日线数据数量失败: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    # ==================== 财务数据操作 ====================

    async def get_financial_data(
        self, ts_code: str, end_date: str | date
    ) -> FinancialData | None:
        """获取财务数据

        Args:
            ts_code: 股票代码
            end_date: 报告期结束日期

        Returns:
            财务数据，不存在时返回None

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            if isinstance(end_date, str):
                end_date = datetime.strptime(end_date, "%Y%m%d").date()

            stmt = select(FinancialData).where(
                and_(
                    FinancialData.ts_code == ts_code, FinancialData.end_date == end_date
                )
            )
            result = await self.session.execute(stmt)
            return result.scalar_one_or_none()
        except Exception as e:
            error_msg = f"获取财务数据失败: {ts_code} {end_date}, 错误: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    async def get_financial_data_range(
        self,
        ts_code: str,
        start_date: str | date | None = None,
        end_date: str | date | None = None,
        limit: int | None = None,
    ) -> list[FinancialData]:
        """获取财务数据范围

        Args:
            ts_code: 股票代码
            start_date: 开始日期
            end_date: 结束日期
            limit: 限制条数

        Returns:
            财务数据列表

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            stmt = select(FinancialData).where(FinancialData.ts_code == ts_code)

            if start_date:
                if isinstance(start_date, str):
                    start_date = datetime.strptime(start_date, "%Y%m%d").date()
                stmt = stmt.where(FinancialData.end_date >= start_date)

            if end_date:
                if isinstance(end_date, str):
                    end_date = datetime.strptime(end_date, "%Y%m%d").date()
                stmt = stmt.where(FinancialData.end_date <= end_date)

            stmt = stmt.order_by(desc(FinancialData.end_date))

            if limit:
                stmt = stmt.limit(limit)

            result = await self.session.execute(stmt)
            return result.scalars().all()
        except Exception as e:
            error_msg = f"获取财务数据范围失败: {ts_code}, 错误: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    async def batch_create_financial_data(
        self, financial_data_list: list[FinancialData]
    ) -> int:
        """批量创建财务数据

        Args:
            financial_data_list: 财务数据列表

        Returns:
            创建的数据条数

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            if not financial_data_list:
                return 0

            self.session.add_all(financial_data_list)
            await self.session.commit()
            count = len(financial_data_list)
            logger.debug(f"批量创建财务数据成功: {count} 条")
            return count
        except Exception as e:
            await self.session.rollback()
            error_msg = f"批量创建财务数据失败: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    async def get_last_financial_data_date(
        self, ts_code: str | None = None
    ) -> date | None:
        """获取最后的财务数据日期

        Args:
            ts_code: 股票代码，为None时获取所有股票的最后日期

        Returns:
            最后的报告期日期，无数据时返回None

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            stmt = select(func.max(FinancialData.end_date))
            if ts_code:
                stmt = stmt.where(FinancialData.ts_code == ts_code)

            result = await self.session.execute(stmt)
            return result.scalar()
        except Exception as e:
            error_msg = f"获取最后财务数据日期失败: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    async def get_financial_data_count(self, ts_code: str | None = None) -> int:
        """获取财务数据数量

        Args:
            ts_code: 股票代码，为None时统计所有股票

        Returns:
            财务数据数量

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            stmt = select(func.count(FinancialData.id))
            if ts_code:
                stmt = stmt.where(FinancialData.ts_code == ts_code)

            result = await self.session.execute(stmt)
            return result.scalar() or 0
        except Exception as e:
            error_msg = f"获取财务数据数量失败: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    # ==================== 综合查询操作 ====================

    async def get_stock_with_latest_data(self, ts_code: str) -> dict[str, Any] | None:
        """获取股票及其最新数据

        Args:
            ts_code: 股票代码

        Returns:
            包含股票基础信息、最新日线数据、最新财务数据的字典

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            # 获取股票基础信息
            stock_info = await self.get_stock_basic_info(ts_code)
            if not stock_info:
                return None

            # 获取最新日线数据
            latest_daily = await self.get_daily_data_range(ts_code, limit=1)
            latest_daily_data = latest_daily[0] if latest_daily else None

            # 获取最新财务数据
            latest_financial = await self.get_financial_data_range(ts_code, limit=1)
            latest_financial_data = latest_financial[0] if latest_financial else None

            return {
                "basic_info": stock_info,
                "latest_daily": latest_daily_data,
                "latest_financial": latest_financial_data,
            }
        except Exception as e:
            error_msg = f"获取股票综合数据失败: {ts_code}, 错误: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e

    async def search_stocks(
        self,
        keyword: str | None = None,
        industry: str | None = None,
        market: str | None = None,
        list_status: str = "L",
        limit: int = 100,
        offset: int = 0,
    ) -> list[StockBasicInfo]:
        """搜索股票

        Args:
            keyword: 关键词（股票代码或名称）
            industry: 行业
            market: 市场
            list_status: 上市状态
            limit: 限制条数
            offset: 偏移量

        Returns:
            股票基础信息列表

        Raises:
            DatabaseError: 数据库操作失败时
        """
        try:
            stmt = select(StockBasicInfo).where(
                StockBasicInfo.list_status == list_status
            )

            if keyword:
                stmt = stmt.where(
                    StockBasicInfo.ts_code.contains(keyword)
                    | StockBasicInfo.name.contains(keyword)
                )

            if industry:
                stmt = stmt.where(StockBasicInfo.industry == industry)

            if market:
                stmt = stmt.where(StockBasicInfo.market == market)

            stmt = stmt.order_by(StockBasicInfo.ts_code).limit(limit).offset(offset)

            result = await self.session.execute(stmt)
            return result.scalars().all()
        except Exception as e:
            error_msg = f"搜索股票失败: {e}"
            logger.error(error_msg)
            raise DatabaseError(error_msg) from e
