"""多因子评分服务

实现技术面、基本面、消息面、市场面四个维度的因子计算和综合评分
"""

import asyncio
from typing import ClassVar

import numpy as np
import pandas as pd
from loguru import logger

from clients.data_collection_client import DataCollectionClient
from models.enums import CacheType
from repositories.cache_repo import CacheRepo
from services.data_service import DataService
from utils.exceptions import DataProcessingError, ValidationError


class FactorService:
    """多因子评分服务

    提供技术面、基本面、消息面、市场面四个维度的因子计算和综合评分功能
    支持权重配置管理和批量股票评分
    """

    # 默认四维度权重配置
    DEFAULT_WEIGHTS: ClassVar[dict[str, float]] = {
        "technical": 0.35,    # 技术面权重
        "fundamental": 0.25,  # 基本面权重
        "news": 0.25,         # 消息面权重
        "market": 0.15,       # 市场面权重
    }

    def __init__(
        self,
        data_service: DataService,
        data_client: DataCollectionClient,
        cache_repo: CacheRepo,
        factor_weights: dict[str, float] | None = None,
    ) -> None:
        """初始化多因子评分服务

        Args:
            data_service: 数据服务
            data_client: 数据采集客户端
            cache_repo: 缓存仓库
            factor_weights: 可选的自定义因子权重配置
        """
        self.data_service = data_service
        self.data_client = data_client
        self.cache_repo = cache_repo

        # 初始化权重配置
        if factor_weights is not None:
            self.factor_weights = self._validate_and_normalize_weights(factor_weights)
        else:
            self.factor_weights = self.DEFAULT_WEIGHTS.copy()

        self._cache_ttl = 300  # 5分钟缓存

        logger.info(
            f"FactorService初始化完成, 权重配置: {self.factor_weights}"
        )

    def _validate_and_normalize_weights(self, weights: dict[str, float]) -> dict[str, float]:
        """验证和标准化权重配置

        Args:
            weights: 权重配置字典

        Returns:
            标准化后的权重配置

        Raises:
            ValidationError: 权重配置无效时抛出
        """
        try:
            # 检查必需的因子维度
            required_factors = ["technical", "fundamental", "news", "market"]
            for factor in required_factors:
                if factor not in weights:
                    raise ValidationError(f"缺少必需的因子权重: {factor}")

            # 检查权重值范围
            for factor, weight in weights.items():
                if not isinstance(weight, int | float):
                    raise ValidationError(f"因子权重必须是数值类型: {factor}={weight}")
                if weight < 0 or weight > 1:
                    raise ValidationError(f"因子权重必须在0-1之间: {factor}={weight}")

            # 计算权重总和并标准化
            total_weight = sum(weights.values())
            if total_weight <= 0:
                raise ValidationError("权重总和必须大于0")

            # 标准化权重使总和为1
            normalized_weights = {k: v / total_weight for k, v in weights.items()}

            logger.info(f"权重配置验证通过并标准化: {normalized_weights}")
            return normalized_weights

        except Exception as e:
            logger.error(f"权重配置验证失败: {e}")
            raise ValidationError(f"权重配置验证失败: {e}") from e

    def update_weights(self, new_weights: dict[str, float]) -> None:
        """更新因子权重配置

        Args:
            new_weights: 新的权重配置
        """
        self.factor_weights = self._validate_and_normalize_weights(new_weights)
        logger.info(f"因子权重已更新: {self.factor_weights}")

    def get_weights(self) -> dict[str, float]:
        """获取当前因子权重配置

        Returns:
            当前权重配置
        """
        return self.factor_weights.copy()

    def reset_weights(self) -> None:
        """重置为默认权重配置"""
        self.factor_weights = self.DEFAULT_WEIGHTS.copy()
        logger.info(f"因子权重已重置为默认配置: {self.factor_weights}")

    async def calculate_factor_scores(
        self,
        symbols: list[str],
        lookback_period: int = 20,
        use_cache: bool = True,
    ) -> dict[str, dict[str, float]]:
        """批量计算股票的四维度因子评分

        Args:
            symbols: 股票代码列表
            lookback_period: 回看期（交易日）
            use_cache: 是否使用缓存

        Returns:
            股票因子评分字典，格式为:
            {
                "symbol1": {
                    "technical": 0.75,
                    "fundamental": 0.65,
                    "news": 0.55,
                    "market": 0.70,
                    "composite": 0.68
                },
                ...
            }
        """
        try:
            results = {}

            # 并行计算所有股票的因子评分
            tasks = [
                self._calculate_single_stock_factors(symbol, lookback_period, use_cache)
                for symbol in symbols
            ]

            factor_results = await asyncio.gather(*tasks, return_exceptions=True)

            # 处理结果
            for symbol, result in zip(symbols, factor_results, strict=False):
                if isinstance(result, Exception):
                    logger.error(f"计算股票 {symbol} 因子评分失败: {result}")
                    # 使用默认中性评分
                    results[symbol] = {
                        "technical": 0.5,
                        "fundamental": 0.5,
                        "news": 0.5,
                        "market": 0.5,
                        "composite": 0.5,
                    }
                else:
                    results[symbol] = result

            logger.info(f"批量计算因子评分完成, 股票数量: {len(symbols)}")
            return results

        except Exception as e:
            logger.error(f"批量计算因子评分失败: {e}")
            raise DataProcessingError(f"批量计算因子评分失败: {e}") from e

    async def _calculate_single_stock_factors(
        self,
        symbol: str,
        lookback_period: int,
        use_cache: bool,
    ) -> dict[str, float]:
        """计算单个股票的四维度因子评分

        Args:
            symbol: 股票代码
            lookback_period: 回看期
            use_cache: 是否使用缓存

        Returns:
            因子评分字典
        """
        try:
            # 构建缓存键
            cache_key = f"factor_scores_{symbol}_{lookback_period}_{hash(str(self.factor_weights))}"

            # 尝试从缓存获取
            if use_cache:
                cached_scores = self.cache_repo.get(
                    CacheType.CALCULATION_RESULT, cache_key, serialize_method="json"
                )
                if cached_scores is not None:
                    return cached_scores

            # 获取市场数据
            end_date = pd.Timestamp.now().strftime("%Y-%m-%d")
            start_date = (pd.Timestamp.now() - pd.Timedelta(days=lookback_period * 2)).strftime("%Y-%m-%d")

            market_data = await self.data_service.get_market_data(
                [symbol], start_date, end_date, use_cache=use_cache
            )

            if market_data.empty:
                raise DataProcessingError(f"无法获取股票 {symbol} 的市场数据")

            # 过滤指定股票的数据
            stock_data = market_data[market_data["symbol"] == symbol].copy()
            if len(stock_data) < lookback_period:
                logger.warning(f"股票 {symbol} 数据不足, 实际: {len(stock_data)}, 需要: {lookback_period}")

            # 计算四维度因子评分
            technical_score = await self._calculate_technical_factors(stock_data, lookback_period)
            fundamental_score = await self._calculate_fundamental_factors(stock_data, symbol)
            news_score = await self._calculate_news_factors(symbol)
            market_score = await self._calculate_market_factors(stock_data, lookback_period)

            # 计算综合评分
            composite_score = (
                technical_score * self.factor_weights["technical"]
                + fundamental_score * self.factor_weights["fundamental"]
                + news_score * self.factor_weights["news"]
                + market_score * self.factor_weights["market"]
            )

            scores = {
                "technical": technical_score,
                "fundamental": fundamental_score,
                "news": news_score,
                "market": market_score,
                "composite": composite_score,
            }

            # 缓存结果
            if use_cache:
                self.cache_repo.set(
                    CacheType.CALCULATION_RESULT,
                    cache_key,
                    scores,
                    ttl=self._cache_ttl,
                    serialize_method="json",
                )

            logger.debug(
                f"股票 {symbol} 因子评分: 技术面={technical_score:.3f}, "
                f"基本面={fundamental_score:.3f}, 消息面={news_score:.3f}, "
                f"市场面={market_score:.3f}, 综合={composite_score:.3f}"
            )

            return scores

        except Exception as e:
            logger.error(f"计算股票 {symbol} 因子评分失败: {e}")
            raise DataProcessingError(f"计算股票 {symbol} 因子评分失败: {e}") from e

    async def _calculate_technical_factors(
        self, stock_data: pd.DataFrame, lookback_period: int
    ) -> float:
        """计算技术面因子评分

        技术面因子包括：
        - 动量因子 (25%): 价格动量和成交量动量
        - 反转因子 (20%): 短期反转信号
        - 波动率因子 (20%): 价格波动率和成交量波动率
        - 技术指标因子 (35%): MA、MACD、RSI、布林带等

        Args:
            stock_data: 股票历史数据
            lookback_period: 回看期

        Returns:
            技术面因子评分 (0-1)
        """
        try:
            if len(stock_data) < lookback_period:
                return 0.5  # 数据不足时返回中性评分

            # 获取最近的数据
            recent_data = stock_data.tail(lookback_period).copy()
            closes = recent_data["close"].values
            volumes = recent_data["volume"].values
            highs = recent_data["high"].values
            lows = recent_data["low"].values

            score = 0.0

            # 1. 动量因子 (25%)
            momentum_score = self._calculate_momentum_factor(closes, volumes)
            score += momentum_score * 0.25

            # 2. 反转因子 (20%)
            reversal_score = self._calculate_reversal_factor(closes)
            score += reversal_score * 0.20

            # 3. 波动率因子 (20%)
            volatility_score = self._calculate_volatility_factor(closes, volumes)
            score += volatility_score * 0.20

            # 4. 技术指标因子 (35%)
            technical_indicator_score = self._calculate_technical_indicators(
                closes, highs, lows, volumes
            )
            score += technical_indicator_score * 0.35

            return min(max(score, 0), 1)

        except Exception as e:
            logger.error(f"计算技术面因子失败: {e}")
            return 0.5

    def _calculate_momentum_factor(self, closes: np.ndarray, volumes: np.ndarray) -> float:
        """计算动量因子"""
        try:
            score = 0.0

            # 价格动量 (60%)
            if len(closes) >= 10:
                short_momentum = (closes[-1] - closes[-5]) / closes[-5]  # 5日动量
                long_momentum = (closes[-1] - closes[-10]) / closes[-10]  # 10日动量
                price_momentum = (short_momentum + long_momentum) / 2
                price_score = min(max(price_momentum * 5 + 0.5, 0), 1)
                score += price_score * 0.6

            # 成交量动量 (40%)
            if len(volumes) >= 5:
                recent_volume = np.mean(volumes[-3:])
                avg_volume = np.mean(volumes[:-3]) if len(volumes) > 3 else recent_volume
                volume_ratio = recent_volume / avg_volume if avg_volume > 0 else 1
                volume_score = min(volume_ratio / 3, 1)  # 成交量放大3倍得满分
                score += volume_score * 0.4

            return score

        except Exception:
            return 0.5

    def _calculate_reversal_factor(self, closes: np.ndarray) -> float:
        """计算反转因子"""
        try:
            if len(closes) < 5:
                return 0.5

            # 短期反转信号
            recent_return = (closes[-1] - closes[-2]) / closes[-2]
            prev_return = (closes[-2] - closes[-3]) / closes[-3]

            # 反转信号：前期下跌后反弹
            if prev_return < -0.02 and recent_return > 0.01:
                return 0.8
            elif prev_return > 0.02 and recent_return < -0.01:
                return 0.2
            else:
                return 0.5

        except Exception:
            return 0.5

    def _calculate_volatility_factor(self, closes: np.ndarray, volumes: np.ndarray) -> float:
        """计算波动率因子"""
        try:
            score = 0.0

            # 价格波动率 (70%)
            if len(closes) >= 10:
                returns = np.diff(closes) / closes[:-1]
                volatility = np.std(returns)
                # 低波动率得分高
                vol_score = max(1 - volatility * 20, 0)
                score += vol_score * 0.7

            # 成交量波动率 (30%)
            if len(volumes) >= 10:
                volume_changes = np.diff(volumes) / volumes[:-1]
                volume_vol = np.std(volume_changes)
                # 适中的成交量波动率得分高
                vol_vol_score = max(1 - abs(volume_vol - 0.3) * 2, 0)
                score += vol_vol_score * 0.3

            return score

        except Exception:
            return 0.5

    def _calculate_technical_indicators(
        self, closes: np.ndarray, highs: np.ndarray, lows: np.ndarray, volumes: np.ndarray
    ) -> float:
        """计算技术指标因子"""
        try:
            score = 0.0

            # MA指标 (25%)
            ma_score = self._calculate_ma_score(closes)
            score += ma_score * 0.25

            # MACD指标 (25%)
            macd_score = self._calculate_macd_score(closes)
            score += macd_score * 0.25

            # RSI指标 (25%)
            rsi_score = self._calculate_rsi_score(closes)
            score += rsi_score * 0.25

            # 布林带指标 (25%)
            bb_score = self._calculate_bollinger_score(closes)
            score += bb_score * 0.25

            return score

        except Exception:
            return 0.5

    def _calculate_ma_score(self, closes: np.ndarray) -> float:
        """计算移动平均线评分"""
        try:
            if len(closes) < 20:
                return 0.5

            ma5 = np.mean(closes[-5:])
            ma10 = np.mean(closes[-10:])
            ma20 = np.mean(closes[-20:])
            current_price = closes[-1]

            # 多头排列得分高
            if current_price > ma5 > ma10 > ma20:
                return 0.9
            elif current_price > ma5 > ma10:
                return 0.7
            elif current_price > ma5:
                return 0.6
            elif current_price < ma5 < ma10 < ma20:
                return 0.1
            else:
                return 0.4

        except Exception:
            return 0.5

    def _calculate_macd_score(self, closes: np.ndarray) -> float:
        """计算MACD评分"""
        try:
            if len(closes) < 26:
                return 0.5

            # 计算MACD
            ema12 = self._calculate_ema(closes, 12)
            ema26 = self._calculate_ema(closes, 26)
            macd_line = ema12 - ema26
            signal_line = self._calculate_ema(macd_line, 9)
            histogram = macd_line - signal_line

            # MACD金叉和柱状图分析
            if len(histogram) >= 2:
                if macd_line[-1] > signal_line[-1] and histogram[-1] > histogram[-2]:
                    return 0.8  # 金叉且柱状图增长
                elif macd_line[-1] < signal_line[-1] and histogram[-1] < histogram[-2]:
                    return 0.2  # 死叉且柱状图下降
                else:
                    return 0.5

            return 0.5

        except Exception:
            return 0.5

    def _calculate_rsi_score(self, closes: np.ndarray) -> float:
        """计算RSI评分"""
        try:
            if len(closes) < 15:
                return 0.5

            # 计算RSI
            deltas = np.diff(closes)
            gains = np.where(deltas > 0, deltas, 0)
            losses = np.where(deltas < 0, -deltas, 0)

            avg_gain = np.mean(gains[-14:])
            avg_loss = np.mean(losses[-14:])

            if avg_loss == 0:
                rsi = 100
            else:
                rs = avg_gain / avg_loss
                rsi = 100 - (100 / (1 + rs))

            # RSI评分
            if 30 <= rsi <= 70:
                return 0.7  # 正常范围
            elif 20 <= rsi < 30:
                return 0.9  # 超卖
            elif 70 < rsi <= 80:
                return 0.3  # 超买
            elif rsi < 20:
                return 0.95  # 严重超卖
            else:  # rsi > 80
                return 0.1  # 严重超买

        except Exception:
            return 0.5

    def _calculate_bollinger_score(self, closes: np.ndarray) -> float:
        """计算布林带评分"""
        try:
            if len(closes) < 20:
                return 0.5

            # 计算布林带
            ma20 = np.mean(closes[-20:])
            std20 = np.std(closes[-20:])
            upper_band = ma20 + 2 * std20
            lower_band = ma20 - 2 * std20
            current_price = closes[-1]

            # 布林带位置评分
            if current_price <= lower_band:
                return 0.9  # 触及下轨，超卖
            elif current_price >= upper_band:
                return 0.1  # 触及上轨，超买
            else:
                # 在布林带内的相对位置
                position = (current_price - lower_band) / (upper_band - lower_band)
                return 0.3 + 0.4 * (1 - abs(position - 0.5) * 2)  # 中间位置得分高

        except Exception:
            return 0.5

    def _calculate_ema(self, data: np.ndarray, period: int) -> np.ndarray:
        """计算指数移动平均线"""
        alpha = 2 / (period + 1)
        ema = np.zeros_like(data)
        ema[0] = data[0]

        for i in range(1, len(data)):
            ema[i] = alpha * data[i] + (1 - alpha) * ema[i - 1]

        return ema

    async def _calculate_fundamental_factors(
        self, stock_data: pd.DataFrame, symbol: str
    ) -> float:
        """计算基本面因子评分

        基本面因子包括：
        - 盈利能力 (30%): ROE、ROA、净利润增长率
        - 估值水平 (25%): PE、PB、PS等估值指标
        - 财务质量 (25%): 资产负债率、流动比率、现金流
        - 成长性 (20%): 营收增长率、利润增长率

        Args:
            stock_data: 股票历史数据
            symbol: 股票代码

        Returns:
            基本面因子评分 (0-1)
        """
        try:
            # 这里使用简化的基本面评分
            # 实际应该从数据采集系统获取财务数据

            if len(stock_data) < 20:
                return 0.5

            closes = stock_data["close"].values
            score = 0.0

            # 1. 价格稳定性代理盈利能力 (30%)
            price_stability = 1 - (np.std(closes) / np.mean(closes))
            stability_score = max(price_stability, 0)
            score += stability_score * 0.3

            # 2. 价格相对位置代理估值水平 (25%)
            current_price = closes[-1]
            price_range = np.max(closes) - np.min(closes)
            if price_range > 0:
                price_position = (current_price - np.min(closes)) / price_range
                # 中等价位得分较高
                valuation_score = 1 - abs(price_position - 0.6) * 2
                score += max(valuation_score, 0) * 0.25

            # 3. 价格波动率代理财务质量 (25%)
            returns = np.diff(closes) / closes[:-1]
            volatility = np.std(returns)
            quality_score = max(1 - volatility * 15, 0)  # 低波动率代表高质量
            score += quality_score * 0.25

            # 4. 长期趋势代理成长性 (20%)
            if len(closes) >= 10:
                long_term_return = (closes[-1] - closes[-10]) / closes[-10]
                growth_score = min(max(long_term_return * 2 + 0.5, 0), 1)
                score += growth_score * 0.2

            return min(max(score, 0), 1)

        except Exception as e:
            logger.error(f"计算基本面因子失败: {e}")
            return 0.5

    async def _calculate_news_factors(self, symbol: str) -> float:
        """计算消息面因子评分

        消息面因子包括：
        - 新闻情感 (40%): 正面/负面新闻情感分析
        - 政策影响 (25%): 政策相关新闻影响
        - 事件驱动 (20%): 重大事件对股价的影响
        - 市场关注度 (15%): 媒体关注度和讨论热度

        Args:
            symbol: 股票代码

        Returns:
            消息面因子评分 (0-1)
        """
        try:
            # 尝试从数据采集系统获取新闻情感数据
            try:
                news_data = await self.data_client.get_news_sentiment(symbol)
                if news_data and "sentiment_score" in news_data:
                    # 使用实际的新闻情感评分
                    sentiment_score = news_data["sentiment_score"]
                    policy_score = news_data.get("policy_impact", 0.5)
                    event_score = news_data.get("event_impact", 0.5)
                    attention_score = news_data.get("attention_score", 0.5)

                    composite_score = (
                        sentiment_score * 0.4
                        + policy_score * 0.25
                        + event_score * 0.2
                        + attention_score * 0.15
                    )

                    return min(max(composite_score, 0), 1)
            except Exception as e:
                logger.debug(f"无法获取股票 {symbol} 的新闻数据: {e}")

            # 如果无法获取新闻数据，返回中性评分
            return 0.5

        except Exception as e:
            logger.error(f"计算消息面因子失败: {e}")
            return 0.5

    async def _calculate_market_factors(
        self, stock_data: pd.DataFrame, lookback_period: int
    ) -> float:
        """计算市场面因子评分

        市场面因子包括：
        - 市场表现 (35%): 相对大盘的表现
        - 资金流向 (25%): 成交量和资金流入流出
        - 市场情绪 (25%): 整体市场情绪指标
        - 行业轮动 (15%): 行业相对表现

        Args:
            stock_data: 股票历史数据
            lookback_period: 回看期

        Returns:
            市场面因子评分 (0-1)
        """
        try:
            if len(stock_data) < lookback_period:
                return 0.5

            recent_data = stock_data.tail(lookback_period)
            closes = recent_data["close"].values
            volumes = recent_data["volume"].values

            score = 0.0

            # 1. 市场表现 (35%)
            stock_return = (closes[-1] - closes[0]) / closes[0]
            # 假设市场平均收益为0（实际应该获取大盘数据）
            market_return = 0.0
            relative_performance = stock_return - market_return
            performance_score = min(max(relative_performance * 3 + 0.5, 0), 1)
            score += performance_score * 0.35

            # 2. 资金流向 (25%)
            if len(volumes) >= 5:
                recent_volume = np.mean(volumes[-3:])
                avg_volume = np.mean(volumes[:-3]) if len(volumes) > 3 else recent_volume
                volume_ratio = recent_volume / avg_volume if avg_volume > 0 else 1
                flow_score = min(volume_ratio / 2, 1)  # 成交量放大代表资金流入
                score += flow_score * 0.25

            # 3. 市场情绪 (25%)
            # 使用价格动量作为市场情绪代理
            if len(closes) >= 5:
                short_momentum = (closes[-1] - closes[-5]) / closes[-5]
                sentiment_score = min(max(short_momentum * 5 + 0.5, 0), 1)
                score += sentiment_score * 0.25

            # 4. 行业轮动 (15%)
            # 简化处理，使用相对强度
            if len(closes) >= 10:
                medium_return = (closes[-1] - closes[-10]) / closes[-10]
                sector_score = min(max(medium_return * 2 + 0.5, 0), 1)
                score += sector_score * 0.15

            return min(max(score, 0), 1)

        except Exception as e:
            logger.error(f"计算市场面因子失败: {e}")
            return 0.5

    async def calculate_composite_score(
        self,
        symbol: str,
        lookback_period: int = 20,
        use_cache: bool = True,
    ) -> float:
        """计算单个股票的综合因子评分

        Args:
            symbol: 股票代码
            lookback_period: 回看期
            use_cache: 是否使用缓存

        Returns:
            综合因子评分 (0-1)
        """
        try:
            scores = await self._calculate_single_stock_factors(
                symbol, lookback_period, use_cache
            )
            return scores["composite"]

        except Exception as e:
            logger.error(f"计算股票 {symbol} 综合评分失败: {e}")
            return 0.5
