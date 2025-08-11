"""数据服务层

提供市场数据获取、清洗、预处理、缓存和技术指标计算功能
"""

import asyncio
from typing import Any

import numpy as np
import pandas as pd
from loguru import logger

from clients.data_collection_client import DataCollectionClient
from models.enums import CacheType
from repositories.cache_repo import CacheRepo
from utils.exceptions import DataProcessingError, ExternalServiceError


class DataService:
    """数据服务

    负责市场数据的获取、清洗、预处理、缓存和技术指标计算
    """

    def __init__(self, data_client: DataCollectionClient, cache_repo: CacheRepo) -> None:
        """初始化数据服务

        Args:
            data_client: 数据采集客户端
            cache_repo: 缓存仓库
        """
        self.data_client = data_client
        self.cache_repo = cache_repo
        self._retry_count = 3
        self._cache_ttl = 300  # 5分钟缓存

    async def get_market_data(
        self,
        symbols: list[str],
        start_date: str | None = None,
        end_date: str | None = None,
        use_cache: bool = True,
    ) -> pd.DataFrame:
        """获取市场数据

        Args:
            symbols: 股票代码列表
            start_date: 开始日期 (YYYY-MM-DD)
            end_date: 结束日期 (YYYY-MM-DD)
            use_cache: 是否使用缓存

        Returns:
            标准化的市场数据DataFrame

        Raises:
            DataProcessingError: 数据处理失败
            ExternalServiceError: 外部服务调用失败
        """
        try:
            # 构建缓存键
            cache_key = self._build_cache_key("market_data", symbols, start_date, end_date)

            # 尝试从缓存获取
            if use_cache:
                cached_data = self.cache_repo.get(
                    CacheType.MARKET_DATA, cache_key, serialize_method="pickle"
                )
                if cached_data is not None:
                    logger.info(f"从缓存获取市场数据: {symbols}")
                    return cached_data

            # 从数据采集系统获取数据
            raw_data = await self._fetch_market_data_with_retry(
                symbols, start_date, end_date
            )

            # 数据清洗和预处理
            cleaned_data = self._clean_market_data(raw_data)
            processed_data = self._preprocess_market_data(cleaned_data)

            # 缓存处理后的数据
            if use_cache:
                self.cache_repo.set(
                    CacheType.MARKET_DATA,
                    cache_key,
                    processed_data,
                    ttl=self._cache_ttl,
                    serialize_method="pickle",
                )

            logger.info(f"成功获取并处理市场数据: {symbols}, 数据量: {len(processed_data)}")
            return processed_data

        except Exception as e:
            logger.error(f"获取市场数据失败: {symbols}, 错误: {e}")
            raise DataProcessingError(f"获取市场数据失败: {e}") from e

    async def get_latest_prices(self, symbols: list[str]) -> dict[str, float]:
        """获取最新价格

        Args:
            symbols: 股票代码列表

        Returns:
            股票代码到最新价格的映射
        """
        try:
            cache_key = f"latest_prices_{'-'.join(sorted(symbols))}"

            # 尝试从缓存获取
            cached_prices = self.cache_repo.get(
                CacheType.MARKET_DATA, cache_key, serialize_method="json"
            )
            if cached_prices is not None:
                logger.debug(f"从缓存获取最新价格: {symbols}")
                return cached_prices

            # 从数据采集系统获取最新数据
            prices = {}
            for symbol in symbols:
                try:
                    response = await asyncio.to_thread(
                        self.data_client.get_latest_market_data, symbol=symbol
                    )
                    if response.get("success") and response.get("data"):
                        data = response["data"]
                        if isinstance(data, list) and data:
                            prices[symbol] = float(data[0].get("close", 0))
                        elif isinstance(data, dict):
                            prices[symbol] = float(data.get("close", 0))
                except Exception as e:
                    logger.warning(f"获取{symbol}最新价格失败: {e}")
                    prices[symbol] = 0.0

            # 缓存价格数据（短期缓存）
            self.cache_repo.set(
                CacheType.MARKET_DATA,
                cache_key,
                prices,
                ttl=60,  # 1分钟缓存
                serialize_method="json",
            )

            logger.info(f"成功获取最新价格: {len(prices)}个股票")
            return prices

        except Exception as e:
            logger.error(f"获取最新价格失败: {symbols}, 错误: {e}")
            return dict.fromkeys(symbols, 0.0)

    def calculate_technical_indicators(
        self, data: pd.DataFrame, indicators: list[str] | None = None
    ) -> pd.DataFrame:
        """计算技术指标

        Args:
            data: 市场数据DataFrame，必须包含OHLCV列
            indicators: 要计算的指标列表，None表示计算所有支持的指标

        Returns:
            包含技术指标的DataFrame

        Raises:
            DataProcessingError: 数据处理失败
        """
        try:
            if data.empty:
                raise DataProcessingError("输入数据为空")

            # 验证必需的列
            required_columns = ["open", "high", "low", "close", "volume"]
            missing_columns = [col for col in required_columns if col not in data.columns]
            if missing_columns:
                raise DataProcessingError(f"缺少必需的列: {missing_columns}")

            result_data = data.copy()

            # 默认计算所有支持的指标
            if indicators is None:
                indicators = ["ma", "macd", "rsi", "bollinger", "ema", "sma"]

            # 计算移动平均线
            if "ma" in indicators or "sma" in indicators:
                result_data = self._calculate_moving_averages(result_data)

            # 计算指数移动平均线
            if "ema" in indicators:
                result_data = self._calculate_ema(result_data)

            # 计算MACD
            if "macd" in indicators:
                result_data = self._calculate_macd(result_data)

            # 计算RSI
            if "rsi" in indicators:
                result_data = self._calculate_rsi(result_data)

            # 计算布林带
            if "bollinger" in indicators:
                result_data = self._calculate_bollinger_bands(result_data)

            logger.info(f"成功计算技术指标: {indicators}, 数据量: {len(result_data)}")
            return result_data

        except Exception as e:
            logger.error(f"计算技术指标失败: {e}")
            raise DataProcessingError(f"计算技术指标失败: {e}") from e

    def generate_trading_signals(
        self, data: pd.DataFrame, strategy_params: dict[str, Any] | None = None
    ) -> pd.DataFrame:
        """生成交易信号

        Args:
            data: 包含技术指标的DataFrame
            strategy_params: 策略参数

        Returns:
            包含交易信号的DataFrame，信号强度范围0-1
        """
        try:
            if data.empty:
                raise DataProcessingError("输入数据为空")

            result_data = data.copy()

            # 默认策略参数
            params = strategy_params or {
                "ma_short": 5,
                "ma_long": 20,
                "rsi_oversold": 30,
                "rsi_overbought": 70,
                "macd_signal_threshold": 0,
            }

            # 初始化信号列
            result_data["buy_signal"] = 0.0
            result_data["sell_signal"] = 0.0
            result_data["signal_strength"] = 0.0

            # MA交叉信号
            if "ma_5" in result_data.columns and "ma_20" in result_data.columns:
                ma_cross_up = (
                    (result_data["ma_5"] > result_data["ma_20"]) &
                    (result_data["ma_5"].shift(1) <= result_data["ma_20"].shift(1))
                )
                ma_cross_down = (
                    (result_data["ma_5"] < result_data["ma_20"]) &
                    (result_data["ma_5"].shift(1) >= result_data["ma_20"].shift(1))
                )
                result_data.loc[ma_cross_up, "buy_signal"] += 0.3
                result_data.loc[ma_cross_down, "sell_signal"] += 0.3

            # RSI信号
            if "rsi" in result_data.columns:
                rsi_oversold = result_data["rsi"] < params["rsi_oversold"]
                rsi_overbought = result_data["rsi"] > params["rsi_overbought"]
                result_data.loc[rsi_oversold, "buy_signal"] += 0.4
                result_data.loc[rsi_overbought, "sell_signal"] += 0.4

            # MACD信号
            if "macd" in result_data.columns and "macd_signal" in result_data.columns:
                macd_bullish = (
                    (result_data["macd"] > result_data["macd_signal"]) &
                    (result_data["macd"].shift(1) <= result_data["macd_signal"].shift(1))
                )
                macd_bearish = (
                    (result_data["macd"] < result_data["macd_signal"]) &
                    (result_data["macd"].shift(1) >= result_data["macd_signal"].shift(1))
                )
                result_data.loc[macd_bullish, "buy_signal"] += 0.3
                result_data.loc[macd_bearish, "sell_signal"] += 0.3

            # 计算综合信号强度
            result_data["signal_strength"] = np.maximum(
                result_data["buy_signal"], result_data["sell_signal"]
            )

            # 限制信号强度在0-1范围内
            result_data["buy_signal"] = np.clip(result_data["buy_signal"], 0, 1)
            result_data["sell_signal"] = np.clip(result_data["sell_signal"], 0, 1)
            result_data["signal_strength"] = np.clip(result_data["signal_strength"], 0, 1)

            logger.info(f"成功生成交易信号, 数据量: {len(result_data)}")
            return result_data

        except Exception as e:
            logger.error(f"生成交易信号失败: {e}")
            raise DataProcessingError(f"生成交易信号失败: {e}") from e

    async def _fetch_market_data_with_retry(
        self, symbols: list[str], start_date: str | None, end_date: str | None
    ) -> list[dict[str, Any]]:
        """带重试的数据获取"""
        last_error = None

        for attempt in range(self._retry_count):
            try:
                all_data = []

                for symbol in symbols:
                    response = await asyncio.to_thread(
                        self.data_client.get_market_data,
                        symbol=symbol,
                        start_date=start_date,
                        end_date=end_date,
                    )

                    if response.get("success") and response.get("data"):
                        data = response["data"]
                        if isinstance(data, list):
                            all_data.extend(data)
                        elif isinstance(data, dict):
                            all_data.append(data)

                if all_data:
                    return all_data
                else:
                    raise ExternalServiceError("未获取到有效数据")

            except Exception as e:
                last_error = e
                if attempt < self._retry_count - 1:
                    wait_time = 2 ** attempt  # 指数退避
                    logger.warning(f"数据获取失败, {wait_time}秒后重试 (第{attempt + 1}次): {e}")
                    await asyncio.sleep(wait_time)
                else:
                    logger.error(f"数据获取最终失败: {e}")

        raise ExternalServiceError(f"数据获取失败, 已重试{self._retry_count}次: {last_error}")

    def _clean_market_data(self, raw_data: list[dict[str, Any]]) -> pd.DataFrame:
        """清洗市场数据"""
        try:
            if not raw_data:
                return pd.DataFrame()

            # 转换为DataFrame
            df = pd.DataFrame(raw_data)

            # 标准化列名
            column_mapping = {
                "ts_code": "symbol",
                "trade_date": "date",
                "open": "open",
                "high": "high",
                "low": "low",
                "close": "close",
                "vol": "volume",
                "amount": "amount",
            }

            # 重命名列
            df = df.rename(columns=column_mapping)

            # 确保必需的列存在
            required_columns = ["symbol", "date", "open", "high", "low", "close", "volume"]
            for col in required_columns:
                if col not in df.columns:
                    if col == "volume":
                        df[col] = 0
                    else:
                        df[col] = np.nan

            # 数据类型转换
            numeric_columns = ["open", "high", "low", "close", "volume"]
            for col in numeric_columns:
                df[col] = pd.to_numeric(df[col], errors="coerce")

            # 日期转换
            if "date" in df.columns:
                df["date"] = pd.to_datetime(df["date"], errors="coerce")

            # 删除无效行
            df = df.dropna(subset=["symbol", "date", "close"])

            # 处理异常值
            df = self._handle_outliers(df)

            logger.debug(f"数据清洗完成, 原始数据: {len(raw_data)}行, 清洗后: {len(df)}行")
            return df

        except Exception as e:
            logger.error(f"数据清洗失败: {e}")
            raise DataProcessingError(f"数据清洗失败: {e}") from e

    def _preprocess_market_data(self, data: pd.DataFrame) -> pd.DataFrame:
        """预处理市场数据"""
        try:
            if data.empty:
                return data

            result_data = data.copy()

            # 按股票代码和日期排序
            result_data = result_data.sort_values(["symbol", "date"])

            # 重置索引
            result_data = result_data.reset_index(drop=True)

            # 计算基础指标
            result_data["price_change"] = result_data.groupby("symbol")["close"].pct_change()
            result_data["price_change_abs"] = result_data.groupby("symbol")["close"].diff()

            # 计算成交额（如果没有的话）
            if "amount" not in result_data.columns:
                result_data["amount"] = result_data["close"] * result_data["volume"]

            # 填充缺失值
            result_data = result_data.fillna(method="ffill").fillna(0)

            logger.debug(f"数据预处理完成, 数据量: {len(result_data)}")
            return result_data

        except Exception as e:
            logger.error(f"数据预处理失败: {e}")
            raise DataProcessingError(f"数据预处理失败: {e}") from e

    def _handle_outliers(self, data: pd.DataFrame) -> pd.DataFrame:
        """处理异常值"""
        try:
            result_data = data.copy()

            # 价格异常值处理
            price_columns = ["open", "high", "low", "close"]
            for col in price_columns:
                if col in result_data.columns:
                    # 使用3倍标准差方法检测异常值
                    mean_val = result_data[col].mean()
                    std_val = result_data[col].std()
                    threshold = 3 * std_val

                    # 标记异常值
                    outliers = (
                        (result_data[col] > mean_val + threshold) |
                        (result_data[col] < mean_val - threshold)
                    )

                    # 用中位数替换异常值
                    if outliers.any():
                        median_val = result_data[col].median()
                        result_data.loc[outliers, col] = median_val
                        logger.debug(f"处理{col}列异常值: {outliers.sum()}个")

            # 成交量异常值处理
            if "volume" in result_data.columns:
                # 成交量不能为负数
                result_data.loc[result_data["volume"] < 0, "volume"] = 0

                # 处理极大的成交量异常值
                volume_q99 = result_data["volume"].quantile(0.99)
                outliers = result_data["volume"] > volume_q99 * 10
                if outliers.any():
                    result_data.loc[outliers, "volume"] = volume_q99
                    logger.debug(f"处理成交量异常值: {outliers.sum()}个")

            return result_data

        except Exception as e:
            logger.error(f"处理异常值失败: {e}")
            return data

    def _calculate_moving_averages(self, data: pd.DataFrame) -> pd.DataFrame:
        """计算移动平均线"""
        result_data = data.copy()

        # 计算不同周期的移动平均线
        periods = [5, 10, 20, 30, 60]

        for period in periods:
            col_name = f"ma_{period}"
            result_data[col_name] = (
                result_data.groupby("symbol")["close"]
                .rolling(window=period, min_periods=1)
                .mean()
                .reset_index(0, drop=True)
            )

        return result_data

    def _calculate_ema(self, data: pd.DataFrame) -> pd.DataFrame:
        """计算指数移动平均线"""
        result_data = data.copy()

        # 计算不同周期的指数移动平均线
        periods = [12, 26]

        for period in periods:
            col_name = f"ema_{period}"
            result_data[col_name] = (
                result_data.groupby("symbol")["close"]
                .ewm(span=period, adjust=False)
                .mean()
                .reset_index(0, drop=True)
            )

        return result_data

    def _calculate_macd(self, data: pd.DataFrame) -> pd.DataFrame:
        """计算MACD指标"""
        result_data = data.copy()

        # 确保有EMA数据
        if "ema_12" not in result_data.columns or "ema_26" not in result_data.columns:
            result_data = self._calculate_ema(result_data)

        # 计算MACD线
        result_data["macd"] = result_data["ema_12"] - result_data["ema_26"]

        # 计算信号线（MACD的9日EMA）
        result_data["macd_signal"] = (
            result_data.groupby("symbol")["macd"]
            .ewm(span=9, adjust=False)
            .mean()
            .reset_index(0, drop=True)
        )

        # 计算MACD柱状图
        result_data["macd_histogram"] = result_data["macd"] - result_data["macd_signal"]

        return result_data

    def _calculate_rsi(self, data: pd.DataFrame, period: int = 14) -> pd.DataFrame:
        """计算RSI指标"""
        result_data = data.copy()

        def calculate_rsi_for_group(group: pd.DataFrame) -> pd.Series:
            close_prices = group["close"]
            delta = close_prices.diff()

            gain = delta.where(delta > 0, 0)
            loss = -delta.where(delta < 0, 0)

            avg_gain = gain.rolling(window=period, min_periods=1).mean()
            avg_loss = loss.rolling(window=period, min_periods=1).mean()

            rs = avg_gain / avg_loss
            rsi = 100 - (100 / (1 + rs))

            return rsi

        result_data["rsi"] = (
            result_data.groupby("symbol")
            .apply(calculate_rsi_for_group)
            .reset_index(0, drop=True)
        )

        return result_data

    def _calculate_bollinger_bands(
        self, data: pd.DataFrame, period: int = 20, std_dev: float = 2.0
    ) -> pd.DataFrame:
        """计算布林带指标"""
        result_data = data.copy()

        # 计算中轨（移动平均线）
        result_data["bb_middle"] = (
            result_data.groupby("symbol")["close"]
            .rolling(window=period, min_periods=1)
            .mean()
            .reset_index(0, drop=True)
        )

        # 计算标准差
        result_data["bb_std"] = (
            result_data.groupby("symbol")["close"]
            .rolling(window=period, min_periods=1)
            .std()
            .reset_index(0, drop=True)
        )

        # 计算上轨和下轨
        result_data["bb_upper"] = result_data["bb_middle"] + (std_dev * result_data["bb_std"])
        result_data["bb_lower"] = result_data["bb_middle"] - (std_dev * result_data["bb_std"])

        # 计算布林带宽度
        result_data["bb_width"] = (
            (result_data["bb_upper"] - result_data["bb_lower"]) / result_data["bb_middle"]
        )

        # 计算价格在布林带中的位置
        result_data["bb_position"] = (
            (result_data["close"] - result_data["bb_lower"]) /
            (result_data["bb_upper"] - result_data["bb_lower"])
        )

        return result_data

    async def get_news_sentiment_data(
        self,
        symbols: list[str] | None = None,
        start_date: str | None = None,
        end_date: str | None = None,
        sentiment_threshold: float = 0.5,
        use_cache: bool = True,
    ) -> dict[str, Any]:
        """获取新闻情感数据

        Args:
            symbols: 股票代码列表，None表示获取全市场新闻
            start_date: 开始日期 (YYYY-MM-DD)
            end_date: 结束日期 (YYYY-MM-DD)
            sentiment_threshold: 情感阈值，用于过滤新闻
            use_cache: 是否使用缓存

        Returns:
            包含新闻情感分析结果的字典
        """
        try:
            # 构建缓存键
            cache_key = self._build_cache_key(
                "news_sentiment", symbols or [], start_date, end_date
            )

            # 尝试从缓存获取
            if use_cache:
                cached_data = self.cache_repo.get(
                    CacheType.MARKET_DATA, cache_key, serialize_method="json"
                )
                if cached_data is not None:
                    logger.info(f"从缓存获取新闻情感数据: {symbols}")
                    return cached_data

            # 从数据采集系统获取新闻数据
            params = {}
            if symbols:
                params["symbols"] = ",".join(symbols)
            if start_date:
                params["start_date"] = start_date
            if end_date:
                params["end_date"] = end_date

            response = await asyncio.to_thread(
                self.data_client.get_news, **params
            )

            if not response.get("success") or not response.get("data"):
                logger.warning("未获取到新闻数据")
                return {"sentiment_score": 0.0, "confidence": 0.0, "news_count": 0}

            news_data = response["data"]
            if not isinstance(news_data, list):
                news_data = [news_data]

            # 分析新闻情感
            sentiment_result = self._analyze_news_sentiment(
                news_data, sentiment_threshold
            )

            # 缓存结果
            if use_cache:
                self.cache_repo.set(
                    CacheType.MARKET_DATA,
                    cache_key,
                    sentiment_result,
                    ttl=self._cache_ttl,
                    serialize_method="json",
                )

            logger.info(
                f"成功获取新闻情感数据: {symbols}, 新闻数量: {sentiment_result.get('news_count', 0)}"
            )
            return sentiment_result

        except Exception as e:
            logger.error(f"获取新闻情感数据失败: {symbols}, 错误: {e}")
            return {"sentiment_score": 0.0, "confidence": 0.0, "news_count": 0}

    async def get_policy_impact_data(
        self,
        symbols: list[str] | None = None,
        start_date: str | None = None,
        end_date: str | None = None,
        impact_threshold: float = 0.5,
        use_cache: bool = True,
    ) -> dict[str, Any]:
        """获取政策影响数据

        Args:
            symbols: 股票代码列表
            start_date: 开始日期 (YYYY-MM-DD)
            end_date: 结束日期 (YYYY-MM-DD)
            impact_threshold: 影响阈值
            use_cache: 是否使用缓存

        Returns:
            包含政策影响分析结果的字典
        """
        try:
            # 构建缓存键
            cache_key = self._build_cache_key(
                "policy_impact", symbols or [], start_date, end_date
            )

            # 尝试从缓存获取
            if use_cache:
                cached_data = self.cache_repo.get(
                    CacheType.MARKET_DATA, cache_key, serialize_method="json"
                )
                if cached_data is not None:
                    logger.info(f"从缓存获取政策影响数据: {symbols}")
                    return cached_data

            # 获取政策相关新闻（通过关键词过滤）
            params = {
                "keywords": "政策,监管,法规,政府,央行,证监会,银保监会",
                "limit": 100,
            }
            if symbols:
                params["symbols"] = ",".join(symbols)
            if start_date:
                params["start_date"] = start_date
            if end_date:
                params["end_date"] = end_date

            response = await asyncio.to_thread(
                self.data_client.get_news, **params
            )

            if not response.get("success") or not response.get("data"):
                logger.warning("未获取到政策相关新闻数据")
                return {"impact_score": 0.0, "confidence": 0.0, "policy_count": 0}

            news_data = response["data"]
            if not isinstance(news_data, list):
                news_data = [news_data]

            # 分析政策影响
            policy_result = self._analyze_policy_impact(
                news_data, impact_threshold
            )

            # 缓存结果
            if use_cache:
                self.cache_repo.set(
                    CacheType.MARKET_DATA,
                    cache_key,
                    policy_result,
                    ttl=self._cache_ttl,
                    serialize_method="json",
                )

            logger.info(
                f"成功获取政策影响数据: {symbols}, 政策新闻数量: {policy_result.get('policy_count', 0)}"
            )
            return policy_result

        except Exception as e:
            logger.error(f"获取政策影响数据失败: {symbols}, 错误: {e}")
            return {"impact_score": 0.0, "confidence": 0.0, "policy_count": 0}

    async def get_event_severity_data(
        self,
        symbols: list[str] | None = None,
        start_date: str | None = None,
        end_date: str | None = None,
        severity_threshold: float = 0.5,
        use_cache: bool = True,
    ) -> dict[str, Any]:
        """获取事件严重性数据

        Args:
            symbols: 股票代码列表
            start_date: 开始日期 (YYYY-MM-DD)
            end_date: 结束日期 (YYYY-MM-DD)
            severity_threshold: 严重性阈值
            use_cache: 是否使用缓存

        Returns:
            包含事件严重性分析结果的字典
        """
        try:
            # 构建缓存键
            cache_key = self._build_cache_key(
                "event_severity", symbols or [], start_date, end_date
            )

            # 尝试从缓存获取
            if use_cache:
                cached_data = self.cache_repo.get(
                    CacheType.MARKET_DATA, cache_key, serialize_method="json"
                )
                if cached_data is not None:
                    logger.info(f"从缓存获取事件严重性数据: {symbols}")
                    return cached_data

            # 获取重大事件相关新闻
            params = {
                "keywords": "重大,突发,紧急,危机,风险,事故,违规,处罚,停牌,退市",
                "limit": 100,
            }
            if symbols:
                params["symbols"] = ",".join(symbols)
            if start_date:
                params["start_date"] = start_date
            if end_date:
                params["end_date"] = end_date

            response = await asyncio.to_thread(
                self.data_client.get_news, **params
            )

            if not response.get("success") or not response.get("data"):
                logger.warning("未获取到事件相关新闻数据")
                return {"severity_score": 0.0, "confidence": 0.0, "event_count": 0}

            news_data = response["data"]
            if not isinstance(news_data, list):
                news_data = [news_data]

            # 分析事件严重性
            event_result = self._analyze_event_severity(
                news_data, severity_threshold
            )

            # 缓存结果
            if use_cache:
                self.cache_repo.set(
                    CacheType.MARKET_DATA,
                    cache_key,
                    event_result,
                    ttl=self._cache_ttl,
                    serialize_method="json",
                )

            logger.info(
                f"成功获取事件严重性数据: {symbols}, 事件新闻数量: {event_result.get('event_count', 0)}"
            )
            return event_result

        except Exception as e:
            logger.error(f"获取事件严重性数据失败: {symbols}, 错误: {e}")
            return {"severity_score": 0.0, "confidence": 0.0, "event_count": 0}

    def _analyze_news_sentiment(
        self, news_data: list[dict[str, Any]], threshold: float
    ) -> dict[str, Any]:
        """分析新闻情感"""
        try:
            if not news_data:
                return {"sentiment_score": 0.0, "confidence": 0.0, "news_count": 0}

            total_sentiment = 0.0
            total_confidence = 0.0
            valid_count = 0

            for news in news_data:
                # 从新闻数据中提取情感分数
                sentiment = news.get("sentiment_score", 0.0)
                confidence = news.get("confidence", 0.0)

                # 如果没有预分析的情感数据，使用简单的关键词分析
                if sentiment == 0.0 and "title" in news:
                    sentiment = self._simple_sentiment_analysis(news["title"])
                    confidence = 0.5  # 简单分析的置信度较低

                if confidence >= threshold:
                    total_sentiment += sentiment * confidence
                    total_confidence += confidence
                    valid_count += 1

            if valid_count == 0:
                return {"sentiment_score": 0.0, "confidence": 0.0, "news_count": len(news_data)}

            avg_sentiment = total_sentiment / total_confidence if total_confidence > 0 else 0.0
            avg_confidence = total_confidence / valid_count

            return {
                "sentiment_score": round(avg_sentiment, 3),
                "confidence": round(avg_confidence, 3),
                "news_count": len(news_data),
                "valid_count": valid_count,
            }

        except Exception as e:
            logger.error(f"分析新闻情感失败: {e}")
            return {"sentiment_score": 0.0, "confidence": 0.0, "news_count": 0}

    def _analyze_policy_impact(
        self, news_data: list[dict[str, Any]], threshold: float
    ) -> dict[str, Any]:
        """分析政策影响"""
        try:
            if not news_data:
                return {"impact_score": 0.0, "confidence": 0.0, "policy_count": 0}

            total_impact = 0.0
            total_confidence = 0.0
            valid_count = 0

            # 政策关键词权重
            policy_keywords = {
                "央行": 0.9, "证监会": 0.9, "银保监会": 0.8, "政府": 0.7,
                "政策": 0.6, "监管": 0.7, "法规": 0.6, "规定": 0.5,
                "利率": 0.8, "准备金": 0.8, "IPO": 0.7, "退市": 0.9
            }

            for news in news_data:
                # 从新闻数据中提取政策影响分数
                impact = news.get("policy_impact_score", 0.0)
                confidence = news.get("confidence", 0.0)

                # 如果没有预分析的政策影响数据，使用关键词分析
                if impact == 0.0 and "title" in news:
                    impact = self._calculate_policy_impact(news["title"], policy_keywords)
                    confidence = 0.6  # 关键词分析的置信度

                if confidence >= threshold:
                    total_impact += impact * confidence
                    total_confidence += confidence
                    valid_count += 1

            if valid_count == 0:
                return {"impact_score": 0.0, "confidence": 0.0, "policy_count": len(news_data)}

            avg_impact = total_impact / total_confidence if total_confidence > 0 else 0.0
            avg_confidence = total_confidence / valid_count

            return {
                "impact_score": round(avg_impact, 3),
                "confidence": round(avg_confidence, 3),
                "policy_count": len(news_data),
                "valid_count": valid_count,
            }

        except Exception as e:
            logger.error(f"分析政策影响失败: {e}")
            return {"impact_score": 0.0, "confidence": 0.0, "policy_count": 0}

    def _analyze_event_severity(
        self, news_data: list[dict[str, Any]], threshold: float
    ) -> dict[str, Any]:
        """分析事件严重性"""
        try:
            if not news_data:
                return {"severity_score": 0.0, "confidence": 0.0, "event_count": 0}

            total_severity = 0.0
            total_confidence = 0.0
            valid_count = 0

            # 事件严重性关键词权重
            severity_keywords = {
                "重大": 0.9, "突发": 0.8, "紧急": 0.8, "危机": 0.9,
                "风险": 0.7, "事故": 0.8, "违规": 0.7, "处罚": 0.6,
                "停牌": 0.8, "退市": 0.9, "破产": 1.0, "倒闭": 1.0
            }

            for news in news_data:
                # 从新闻数据中提取事件严重性分数
                severity = news.get("event_severity_score", 0.0)
                confidence = news.get("confidence", 0.0)

                # 如果没有预分析的事件严重性数据，使用关键词分析
                if severity == 0.0 and "title" in news:
                    severity = self._calculate_event_severity(news["title"], severity_keywords)
                    confidence = 0.6  # 关键词分析的置信度

                if confidence >= threshold:
                    total_severity += severity * confidence
                    total_confidence += confidence
                    valid_count += 1

            if valid_count == 0:
                return {"severity_score": 0.0, "confidence": 0.0, "event_count": len(news_data)}

            avg_severity = total_severity / total_confidence if total_confidence > 0 else 0.0
            avg_confidence = total_confidence / valid_count

            return {
                "severity_score": round(avg_severity, 3),
                "confidence": round(avg_confidence, 3),
                "event_count": len(news_data),
                "valid_count": valid_count,
            }

        except Exception as e:
            logger.error(f"分析事件严重性失败: {e}")
            return {"severity_score": 0.0, "confidence": 0.0, "event_count": 0}

    def _simple_sentiment_analysis(self, text: str) -> float:
        """简单的情感分析"""
        positive_words = ["上涨", "利好", "增长", "盈利", "收益", "成功", "突破", "创新"]
        negative_words = ["下跌", "利空", "下降", "亏损", "风险", "失败", "危机", "问题"]

        positive_count = sum(1 for word in positive_words if word in text)
        negative_count = sum(1 for word in negative_words if word in text)

        total_count = positive_count + negative_count
        if total_count == 0:
            return 0.0

        return (positive_count - negative_count) / total_count

    def _calculate_policy_impact(self, text: str, keywords: dict[str, float]) -> float:
        """计算政策影响分数"""
        total_weight = 0.0
        for keyword, weight in keywords.items():
            if keyword in text:
                total_weight += weight

        # 归一化到0-1范围
        return min(total_weight / 2.0, 1.0)

    def _calculate_event_severity(self, text: str, keywords: dict[str, float]) -> float:
        """计算事件严重性分数"""
        max_weight = 0.0
        for keyword, weight in keywords.items():
            if keyword in text:
                max_weight = max(max_weight, weight)

        return max_weight

    def _build_cache_key(self, prefix: str, symbols: list[str], start_date: str | None, end_date: str | None) -> str:
        """构建缓存键"""
        symbols_str = "-".join(sorted(symbols))
        date_str = f"{start_date or 'none'}_{end_date or 'none'}"
        return f"{prefix}_{symbols_str}_{date_str}"
