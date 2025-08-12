"""数据采集服务模块

提供统一的数据采集接口，集成股票基础信息、行情数据、财务数据采集功能。
支持历史数据全量采集和增量更新，包含数据预处理和格式标准化。
"""

from datetime import date, timedelta
from typing import Any

from loguru import logger

from clients.tushare_client import get_tushare_client
from models.database import StockBasicInfo
from repositories.stock_repo import StockRepository
from services.quality_service import QualityService
from utils.exceptions import DataCollectionError, ValidationError


class CollectionService:
    """数据采集服务

    提供统一的数据采集接口，支持股票基础信息、行情数据、财务数据的采集。
    包含数据预处理、格式标准化、完整性验证和去重逻辑。
    """

    def __init__(self, stock_repo: StockRepository, quality_service: QualityService):
        """初始化数据采集服务

        Args:
            stock_repo: 股票数据仓库
            quality_service: 数据质量服务
        """
        self.tushare_client = get_tushare_client()
        self.stock_repo = stock_repo
        self.quality_service = quality_service
        logger.info("数据采集服务初始化完成")

    async def collect_stock_basic_info(self,
                                       list_status: str = "L",
                                       exchange: str | None = None,
                                       force_update: bool = False) -> int:
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
            logger.info(f"开始采集股票基础信息, 状态: {list_status}, 交易所: {exchange}")

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
                    logger.warning(f"股票基础信息验证失败: {stock_data.get('ts_code')}, 错误: {e}")
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
                        await self.stock_repo.update_stock_basic_info(ts_code, stock_data)
                        updated_count += 1
                        logger.debug(f"更新股票基础信息: {ts_code}")
                elif not existing_stock or force_update:
                    await self.stock_repo.create_stock_basic_info(stock_data)
                    new_count += 1
                    logger.debug(f"新增股票基础信息: {ts_code}")

            total_count = new_count + updated_count
            logger.info(f"股票基础信息采集完成, 新增: {new_count}, 更新: {updated_count}, 总计: {total_count}")
            return total_count

        except Exception as e:
            error_msg = f"采集股票基础信息失败: {e!s}"
            logger.error(error_msg)
            raise DataCollectionError(error_msg) from e

    async def collect_daily_data(self,
                                 ts_code: str | None = None,
                                 start_date: str | date | None = None,
                                 end_date: str | date | None = None,
                                 is_incremental: bool = True) -> int:
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
                start_date = last_date + timedelta(days=1) if last_date else date.today() - timedelta(days=30)

            if not end_date:
                end_date = date.today()

            logger.info(f"开始采集日线数据, 股票数量: {len(stock_codes)}, 日期范围: {start_date} ~ {end_date}")

            total_count = 0
            for i, code in enumerate(stock_codes, 1):
                try:
                    count = await self._collect_single_stock_daily_data(code, start_date, end_date)
                    total_count += count
                    logger.debug(f"进度: {i}/{len(stock_codes)}, 股票: {code}, 采集: {count} 条")
                except Exception as e:
                    logger.error(f"采集股票 {code} 日线数据失败: {e}")
                    continue

            logger.info(f"日线数据采集完成, 总计: {total_count} 条")
            return total_count

        except Exception as e:
            error_msg = f"采集日线数据失败: {e!s}"
            logger.error(error_msg)
            raise DataCollectionError(error_msg) from e

    async def collect_financial_data(self,
                                     ts_code: str | None = None,
                                     start_date: str | date | None = None,
                                     end_date: str | date | None = None,
                                     period: str | None = None,
                                     is_incremental: bool = True) -> int:
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
                start_date = last_date if last_date else date.today() - timedelta(days=365)

            if not end_date:
                end_date = date.today()

            logger.info(f"开始采集财务数据, 股票数量: {len(stock_codes)}, 日期范围: {start_date} ~ {end_date}")

            total_count = 0
            for i, code in enumerate(stock_codes, 1):
                try:
                    count = await self._collect_single_stock_financial_data(code, start_date, end_date, period)
                    total_count += count
                    logger.debug(f"进度: {i}/{len(stock_codes)}, 股票: {code}, 采集: {count} 条")
                except Exception as e:
                    logger.error(f"采集股票 {code} 财务数据失败: {e}")
                    continue

            logger.info(f"财务数据采集完成, 总计: {total_count} 条")
            return total_count

        except Exception as e:
            error_msg = f"采集财务数据失败: {e!s}"
            logger.error(error_msg)
            raise DataCollectionError(error_msg) from e

    async def _collect_single_stock_daily_data(self,
                                               ts_code: str,
                                               start_date: str | date,
                                               end_date: str | date) -> int:
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
                existing = await self.stock_repo.get_daily_data(ts_code, data.trade_date)
                if not existing:
                    validated_data.append(data)

            except ValidationError as e:
                logger.warning(f"日线数据验证失败: {ts_code} {data.trade_date}, 错误: {e}")
                continue

        # 批量保存
        if validated_data:
            await self.stock_repo.batch_create_daily_data(validated_data)

        return len(validated_data)

    async def _collect_single_stock_financial_data(self,
                                                   ts_code: str,
                                                   start_date: str | date,
                                                   end_date: str | date,
                                                   period: str | None) -> int:
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
        raw_data = self.tushare_client.get_financial_data(ts_code, start_date, end_date, period)
        if not raw_data:
            return 0

        # 数据质量验证和去重
        validated_data = []
        for data in raw_data:
            try:
                # 验证数据完整性
                self.quality_service.validate_financial_data(data)

                # 检查是否已存在
                existing = await self.stock_repo.get_financial_data(ts_code, data.end_date)
                if not existing:
                    validated_data.append(data)

            except ValidationError as e:
                logger.warning(f"财务数据验证失败: {ts_code} {data.end_date}, 错误: {e}")
                continue

        # 批量保存
        if validated_data:
            await self.stock_repo.batch_create_financial_data(validated_data)

        return len(validated_data)

    def _should_update_stock_basic(self, existing: StockBasicInfo, new_data: StockBasicInfo) -> bool:
        """判断是否需要更新股票基础信息

        Args:
            existing: 现有数据
            new_data: 新数据

        Returns:
            是否需要更新
        """
        # 检查关键字段是否有变化
        key_fields = ['name', 'industry', 'market', 'list_status', 'list_date', 'delist_date']
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
                'stock_count': await self.stock_repo.get_stock_count(),
                'daily_data_count': await self.stock_repo.get_daily_data_count(),
                'financial_data_count': await self.stock_repo.get_financial_data_count(),
                'last_daily_date': await self.stock_repo.get_last_daily_data_date(),
                'last_financial_date': await self.stock_repo.get_last_financial_data_date(),
            }
            return stats
        except Exception as e:
            logger.error(f"获取采集统计信息失败: {e}")
            return {}
