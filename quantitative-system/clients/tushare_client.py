#!/usr/bin/env python3
"""Tushare API客户端

提供对Tushare API的封装，包括：
- API密钥管理和认证
- 调用频率限制和重试机制
- 错误处理和日志记录
- 数据格式标准化
"""

import time
from datetime import date, datetime
from typing import Any

import pandas as pd
import tushare as ts
from tenacity import (
    retry,
    retry_if_exception_type,
    stop_after_attempt,
    wait_exponential,
)

from config.settings import get_settings
from models.types import FinancialData, StockBasicInfo, StockDailyData
from utils.exceptions import ConfigurationError, DataCollectionError
from utils.logger import get_logger

logger = get_logger(__name__)
settings = get_settings()


class TushareClient:
    """Tushare API客户端

    提供对Tushare API的统一访问接口，包括频率限制、重试机制和错误处理。
    """

    def __init__(self, token: str | None = None):
        """初始化Tushare客户端

        Args:
            token: Tushare API token，如果不提供则从配置中获取

        Raises:
            ConfigurationError: 当token未配置时

        """
        self.token = token or settings.tushare_token
        if not self.token:
            raise ConfigurationError("Tushare token未配置")

        # 初始化Tushare API
        ts.set_token(self.token)
        self.pro = ts.pro_api()

        # 配置频率限制参数
        self.timeout = settings.tushare_timeout
        self.max_retries = settings.tushare_retries
        self.retry_delay = settings.tushare_retry_delay

        # 记录最后调用时间，用于频率控制
        self._last_call_time = 0.0
        self._min_interval = 0.2  # 最小调用间隔（秒）

        logger.info(f"Tushare客户端初始化完成, token: {self.token[:8]}***")

    def _rate_limit(self) -> None:
        """实现API调用频率限制"""
        current_time = time.time()
        time_since_last_call = current_time - self._last_call_time

        if time_since_last_call < self._min_interval:
            sleep_time = self._min_interval - time_since_last_call
            logger.debug(f"频率限制: 等待 {sleep_time:.2f} 秒")
            time.sleep(sleep_time)

        self._last_call_time = time.time()

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=10),
        retry=retry_if_exception_type((Exception,))
    )
    def _api_call(self, api_name: str, **kwargs) -> pd.DataFrame:
        """执行API调用，包含重试机制

        Args:
            api_name: API方法名
            **kwargs: API参数

        Returns:
            API返回的DataFrame

        Raises:
            DataCollectionError: 当API调用失败时
        """
        self._rate_limit()

        try:
            logger.debug(f"调用Tushare API: {api_name}, 参数: {kwargs}")

            # 获取API方法
            api_method = getattr(self.pro, api_name)
            if not api_method:
                raise DataCollectionError(f"未找到API方法: {api_name}")

            # 执行API调用
            result = api_method(**kwargs)

            if result is None or result.empty:
                logger.warning(f"API {api_name} 返回空数据, 参数: {kwargs}")
                return pd.DataFrame()

            logger.debug(f"API {api_name} 成功返回 {len(result)} 条数据")
            return result

        except Exception as e:
            error_msg = f"Tushare API调用失败: {api_name}, 错误: {e!s}"
            logger.error(error_msg)
            raise DataCollectionError(error_msg) from e

    def get_stock_basic(self,
                       list_status: str = 'L',
                       exchange: str | None = None) -> list[StockBasicInfo]:
        """获取股票基础信息

        Args:
            list_status: 上市状态 L上市 D退市 P暂停上市
            exchange: 交易所 SSE上交所 SZSE深交所

        Returns:
            股票基础信息列表
        """
        try:
            params = {'list_status': list_status}
            if exchange:
                params['exchange'] = exchange

            df = self._api_call('stock_basic', **params)

            if df.empty:
                return []

            # 转换为标准格式
            stocks = []
            for _, row in df.iterrows():
                stock = StockBasicInfo(
                    ts_code=row['ts_code'],
                    symbol=row['symbol'],
                    name=row['name'],
                    area=row.get('area', ''),
                    industry=row.get('industry', ''),
                    market=row.get('market', ''),
                    list_date=self._parse_date(row.get('list_date')),
                    list_status=row.get('list_status', 'L'),
                    delist_date=self._parse_date(row.get('delist_date')),
                    is_hs=row.get('is_hs', 'N')
                )
                stocks.append(stock)

            logger.info(f"获取股票基础信息成功, 共 {len(stocks)} 只股票")
            return stocks

        except Exception as e:
            error_msg = f"获取股票基础信息失败: {e!s}"
            logger.error(error_msg)
            raise DataCollectionError(error_msg) from e

    def get_daily_data(self,
                      ts_code: str | None = None,
                      trade_date: str | date | None = None,
                      start_date: str | date | None = None,
                      end_date: str | date | None = None) -> list[StockDailyData]:
        """获取股票日线数据

        Args:
            ts_code: 股票代码
            trade_date: 交易日期
            start_date: 开始日期
            end_date: 结束日期

        Returns:
            财务数据列表
        """
        try:
            params = {}

            if ts_code:
                params['ts_code'] = ts_code
            if trade_date:
                params['trade_date'] = self._format_date(trade_date)
            if start_date:
                params['start_date'] = self._format_date(start_date)
            if end_date:
                params['end_date'] = self._format_date(end_date)

            df = self._api_call('daily', **params)

            if df.empty:
                return []

            # 转换为标准格式
            daily_data = []
            for _, row in df.iterrows():
                data = StockDailyData(
                    ts_code=row['ts_code'],
                    trade_date=self._parse_date(row['trade_date']),
                    open_price=float(row.get('open', 0)),
                    high_price=float(row.get('high', 0)),
                    low_price=float(row.get('low', 0)),
                    close_price=float(row.get('close', 0)),
                    pre_close=float(row.get('pre_close', 0)),
                    change=float(row.get('change', 0)),
                    pct_chg=float(row.get('pct_chg', 0)),
                    vol=float(row.get('vol', 0)),
                    amount=float(row.get('amount', 0))
                )
                daily_data.append(data)

            logger.info(f"获取日线数据成功, 共 {len(daily_data)} 条记录")
            return daily_data

        except Exception as e:
            error_msg = f"获取日线数据失败: {e!s}"
            logger.error(error_msg)
            raise DataCollectionError(error_msg) from e

    def get_financial_data(self,
                          ts_code: str,
                          ann_date: str | date | None = None,
                          start_date: str | date | None = None,
                          end_date: str | date | None = None,
                          period: str | None = None) -> list[FinancialData]:
        """获取财务数据

        Args:
            ts_code: 股票代码
            ann_date: 公告日期
            start_date: 开始日期
            end_date: 结束日期
            period: 报告期

        Returns:
            财务数据列表
        """
        try:
            params = {'ts_code': ts_code}

            if ann_date:
                params['ann_date'] = self._format_date(ann_date)
            if start_date:
                params['start_date'] = self._format_date(start_date)
            if end_date:
                params['end_date'] = self._format_date(end_date)
            if period:
                params['period'] = period

            # 获取利润表数据
            income_df = self._api_call('income', **params)
            # 获取资产负债表数据
            balancesheet_df = self._api_call('balancesheet', **params)
            # 获取现金流量表数据
            cashflow_df = self._api_call('cashflow', **params)

            # 合并财务数据
            financial_data = self._merge_financial_data(
                income_df, balancesheet_df, cashflow_df
            )

            logger.info(f"获取财务数据成功, 共 {len(financial_data)} 条记录")
            return financial_data

        except Exception as e:
            error_msg = f"获取财务数据失败: {e!s}"
            logger.error(error_msg)
            raise DataCollectionError(error_msg) from e

    def _merge_financial_data(self,
                             income_df: pd.DataFrame,
                             balance_df: pd.DataFrame,
                             cashflow_df: pd.DataFrame) -> list[FinancialData]:
        """合并财务数据

        Args:
            income_df: 利润表数据
            balance_df: 资产负债表数据
            cashflow_df: 现金流量表数据

        Returns:
            合并后的财务数据列表
        """
        financial_data = []

        # 以利润表为基础进行合并
        for _, income_row in income_df.iterrows():
            ts_code = income_row['ts_code']
            end_date = income_row['end_date']

            # 查找对应的资产负债表数据
            balance_row = balance_df[
                (balance_df['ts_code'] == ts_code) &
                (balance_df['end_date'] == end_date)
            ]

            # 查找对应的现金流量表数据
            cashflow_row = cashflow_df[
                (cashflow_df['ts_code'] == ts_code) &
                (cashflow_df['end_date'] == end_date)
            ]

            # 创建财务数据对象
            data = FinancialData(
                ts_code=ts_code,
                ann_date=self._parse_date(income_row.get('ann_date')),
                f_ann_date=self._parse_date(income_row.get('f_ann_date')),
                end_date=self._parse_date(end_date),
                report_type=income_row.get('report_type', ''),
                comp_type=income_row.get('comp_type', ''),
                # 利润表数据
                total_revenue=self._safe_float(income_row.get('total_revenue')),
                revenue=self._safe_float(income_row.get('revenue')),
                operate_profit=self._safe_float(income_row.get('operate_profit')),
                total_profit=self._safe_float(income_row.get('total_profit')),
                n_income=self._safe_float(income_row.get('n_income')),
                n_income_attr_p=self._safe_float(income_row.get('n_income_attr_p')),
                # 资产负债表数据
                total_assets=self._safe_float(balance_row.iloc[0].get('total_assets') if not balance_row.empty else None),
                total_liab=self._safe_float(balance_row.iloc[0].get('total_liab') if not balance_row.empty else None),
                total_hldr_eqy_exc_min_int=self._safe_float(balance_row.iloc[0].get('total_hldr_eqy_exc_min_int') if not balance_row.empty else None),
                # 现金流量表数据
                n_cashflow_act=self._safe_float(cashflow_row.iloc[0].get('n_cashflow_act') if not cashflow_row.empty else None),
                n_cashflow_inv_act=self._safe_float(cashflow_row.iloc[0].get('n_cashflow_inv_act') if not cashflow_row.empty else None),
                n_cashflow_fin_act=self._safe_float(cashflow_row.iloc[0].get('n_cashflow_fin_act') if not cashflow_row.empty else None)
            )

            financial_data.append(data)

        return financial_data

    def _format_date(self, date_value: str | date | datetime) -> str:
        """格式化日期为Tushare API要求的格式

        Args:
            date_value: 日期值

        Returns:
            格式化后的日期字符串 (YYYYMMDD)
        """
        if isinstance(date_value, str):
            # 假设输入格式为 YYYY-MM-DD 或 YYYYMMDD
            if '-' in date_value:
                return date_value.replace('-', '')
            return date_value
        elif isinstance(date_value, date | datetime):
            return date_value.strftime('%Y%m%d')
        else:
            raise ValueError(f"不支持的日期格式: {type(date_value)}")

    def _parse_date(self, date_str: str | None) -> date | None:
        """解析日期字符串

        Args:
            date_str: 日期字符串 (YYYYMMDD)

        Returns:
            解析后的日期对象
        """
        if not date_str or pd.isna(date_str):
            return None

        try:
            date_str = str(date_str).strip()
            if len(date_str) == 8:
                return datetime.strptime(date_str, '%Y%m%d').date()
            elif len(date_str) == 10 and '-' in date_str:
                return datetime.strptime(date_str, '%Y-%m-%d').date()
            else:
                logger.warning(f"无法解析日期格式: {date_str}")
                return None
        except ValueError as e:
            logger.warning(f"日期解析失败: {date_str}, 错误: {e}")
            return None

    def _safe_float(self, value: Any) -> float | None:
        """安全转换为浮点数

        Args:
            value: 待转换的值

        Returns:
            转换后的浮点数，失败时返回None
        """
        if value is None or pd.isna(value):
            return None

        try:
            return float(value)
        except (ValueError, TypeError):
            return None

    def test_connection(self) -> bool:
        """测试API连接

        Returns:
            连接是否成功
        """
        try:
            # 尝试获取少量数据来测试连接
            df = self._api_call('stock_basic', list_status='L', limit=1)
            return not df.empty
        except Exception as e:
            logger.error(f"Tushare连接测试失败: {e}")
            return False

    def get_trade_cal(self,
                     exchange: str = 'SSE',
                     start_date: str | date | None = None,
                     end_date: str | date | None = None) -> list[dict[str, Any]]:
        """获取交易日历

        Args:
            exchange: 交易所代码
            start_date: 开始日期
            end_date: 结束日期

        Returns:
            交易日历数据
        """
        try:
            params = {'exchange': exchange}

            if start_date:
                params['start_date'] = self._format_date(start_date)
            if end_date:
                params['end_date'] = self._format_date(end_date)

            df = self._api_call('trade_cal', **params)

            if df.empty:
                return []

            # 转换为字典列表
            calendar_data = []
            for _, row in df.iterrows():
                data = {
                    'exchange': row['exchange'],
                    'cal_date': self._parse_date(row['cal_date']),
                    'is_open': bool(row.get('is_open', 0))
                }
                calendar_data.append(data)

            logger.info(f"获取交易日历成功, 共 {len(calendar_data)} 条记录")
            return calendar_data

        except Exception as e:
            error_msg = f"获取交易日历失败: {e!s}"
            logger.error(error_msg)
            raise DataCollectionError(error_msg) from e


# 创建全局客户端实例
_tushare_client: TushareClient | None = None


def get_tushare_client() -> TushareClient:
    """获取Tushare客户端单例

    Returns:
        Tushare客户端实例
    """
    global _tushare_client

    if _tushare_client is None:
        _tushare_client = TushareClient()

    return _tushare_client


def reset_tushare_client() -> None:
    """重置Tushare客户端（主要用于测试）"""
    global _tushare_client
    _tushare_client = None
