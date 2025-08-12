"""多因子选股策略模块

实现基于四维度因子评分的多因子选股策略。
"""

from typing import Any

import numpy as np
from loguru import logger

from .base_strategy import BaseStrategy, SignalType, TradingSignal


class MultiFactorStrategy(BaseStrategy):
    """多因子选股策略

    基于技术面、基本面、消息面、市场面四个维度的因子评分进行选股和交易决策。
    支持动态权重配置、风险控制和仓位管理。
    """

    # 策略参数
    params = (
        # 四维度因子权重
        ("technical_weight", 0.35),  # 技术面权重
        ("fundamental_weight", 0.25),  # 基本面权重
        ("news_weight", 0.25),  # 消息面权重
        ("market_weight", 0.15),  # 市场面权重
        # 交易阈值
        ("buy_threshold", 0.7),  # 买入阈值
        ("sell_threshold", 0.3),  # 卖出阈值
        ("hold_threshold", 0.5),  # 持有阈值
        # 风险控制
        ("max_position_size", 0.2),  # 最大单个仓位比例
        ("stop_loss_pct", 0.08),  # 止损百分比
        ("max_drawdown_pct", 0.15),  # 最大回撤百分比
        # 置信度
        ("min_confidence_score", 0.6),  # 最小置信度
        # 回测参数
        ("rebalance_frequency", 5),  # 再平衡频率（交易日）
        ("lookback_period", 20),  # 回看期（交易日）
        # 基础参数
        ("position_size", 1000),  # 默认仓位大小
        ("max_positions", 5),  # 最大持仓数量
    )

    def __init__(self):
        """初始化策略"""
        super().__init__()

        # 策略状态
        self.day_count = 0
        self.last_rebalance_day = 0
        self.factor_scores_history = []
        self.portfolio_value_history = []

        # 验证权重配置
        self._validate_weights()

        logger.info(
            f"多因子策略初始化完成, 权重配置: 技术面={self.params.technical_weight}, "
            f"基本面={self.params.fundamental_weight}, 消息面={self.params.news_weight}, "
            f"市场面={self.params.market_weight}"
        )

    def _validate_weights(self) -> None:
        """验证权重配置"""
        total_weight = (
            self.params.technical_weight
            + self.params.fundamental_weight
            + self.params.news_weight
            + self.params.market_weight
        )

        if abs(total_weight - 1.0) > 0.01:  # 允许1%的误差
            logger.warning(f"权重总和不为1.0: {total_weight}, 将进行标准化")
            # 标准化权重
            self.params.technical_weight /= total_weight
            self.params.fundamental_weight /= total_weight
            self.params.news_weight /= total_weight
            self.params.market_weight /= total_weight

    def get_strategy_name(self) -> str:
        """获取策略名称"""
        return "MultiFactorStrategy"

    def get_strategy_description(self) -> str:
        """获取策略描述"""
        return "基于四维度因子评分的多因子选股策略, 支持技术面、基本面、消息面、市场面综合分析"

    def get_required_params(self) -> list[str]:
        """获取策略必需参数列表"""
        return [
            "technical_weight",
            "fundamental_weight",
            "news_weight",
            "market_weight",
            "buy_threshold",
            "sell_threshold",
        ]

    def validate_params(self) -> bool:
        """验证策略参数是否有效"""
        try:
            # 验证权重范围
            weights = [
                self.params.technical_weight,
                self.params.fundamental_weight,
                self.params.news_weight,
                self.params.market_weight,
            ]

            for weight in weights:
                if not (0 <= weight <= 1):
                    logger.error(f"权重必须在0-1范围内: {weight}")
                    return False

            # 验证阈值范围
            if not (0 <= self.params.buy_threshold <= 1):
                logger.error(f"买入阈值必须在0-1范围内: {self.params.buy_threshold}")
                return False

            if not (0 <= self.params.sell_threshold <= 1):
                logger.error(f"卖出阈值必须在0-1范围内: {self.params.sell_threshold}")
                return False

            if self.params.buy_threshold <= self.params.sell_threshold:
                logger.error(
                    f"买入阈值必须大于卖出阈值: {self.params.buy_threshold} <= {self.params.sell_threshold}"
                )
                return False

            # 验证风险控制参数
            if not (0 < self.params.max_position_size <= 1):
                logger.error(
                    f"最大仓位比例必须在0-1范围内: {self.params.max_position_size}"
                )
                return False

            if not (0 < self.params.stop_loss_pct <= 1):
                logger.error(f"止损百分比必须在0-1范围内: {self.params.stop_loss_pct}")
                return False

            return True

        except Exception as e:
            logger.error(f"参数验证失败: {e}")
            return False

    def generate_signal(self) -> TradingSignal | None:
        """生成交易信号"""
        try:
            self.day_count += 1

            # 检查是否需要再平衡
            if not self._should_rebalance():
                return None

            # 计算当前股票的因子评分
            factor_score = self._calculate_current_factor_score()

            if factor_score is None:
                return None

            # 记录评分历史
            self.factor_scores_history.append(
                {
                    "date": self.datas[0].datetime.date(0),
                    "score": factor_score,
                    "price": self.data.close[0],
                }
            )

            # 基于评分和阈值生成交易信号
            signal = self._generate_signal_from_score(factor_score)

            # 应用风险管理
            if signal:
                signal = self._apply_risk_management(signal)

            return signal

        except Exception as e:
            logger.error(f"生成交易信号失败: {e}")
            return None

    def _should_rebalance(self) -> bool:
        """检查是否需要再平衡"""
        return (
            self.day_count - self.last_rebalance_day
        ) >= self.params.rebalance_frequency

    def _calculate_current_factor_score(self) -> float | None:
        """计算当前股票的因子评分"""
        try:
            # 检查数据可用性
            if len(self.data) < self.params.lookback_period:
                return None

            # 计算四维度因子评分
            technical_score = self._calculate_technical_factor()
            fundamental_score = self._calculate_fundamental_factor()
            news_score = self._calculate_news_factor()
            market_score = self._calculate_market_factor()

            # 计算综合评分
            composite_score = (
                technical_score * self.params.technical_weight
                + fundamental_score * self.params.fundamental_weight
                + news_score * self.params.news_weight
                + market_score * self.params.market_weight
            )

            logger.debug(
                f"因子评分 - 技术面: {technical_score:.3f}, 基本面: {fundamental_score:.3f}, "
                f"消息面: {news_score:.3f}, 市场面: {market_score:.3f}, 综合: {composite_score:.3f}"
            )

            return composite_score

        except Exception as e:
            logger.error(f"计算因子评分失败: {e}")
            return None

    def _calculate_technical_factor(self) -> float:
        """计算技术面因子评分

        技术面因子包括：
        - 动量因子 (25%): 价格动量和成交量动量
        - 反转因子 (20%): 短期反转信号
        - 波动率因子 (20%): 价格波动率和成交量波动率
        - 技术指标因子 (35%): MA、MACD、RSI、布林带等
        """
        try:
            # 获取价格数据
            closes = np.array(
                [self.data.close[-i] for i in range(self.params.lookback_period, 0, -1)]
            )
            volumes = np.array(
                [
                    self.data.volume[-i]
                    for i in range(self.params.lookback_period, 0, -1)
                ]
            )
            highs = np.array(
                [self.data.high[-i] for i in range(self.params.lookback_period, 0, -1)]
            )
            lows = np.array(
                [self.data.low[-i] for i in range(self.params.lookback_period, 0, -1)]
            )

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
            return 0.5  # 默认中性评分

    def _calculate_fundamental_factor(self) -> float:
        """计算基本面因子评分

        基本面因子包括：
        - 盈利能力 (30%): 使用价格稳定性作为代理
        - 估值水平 (25%): 使用价格相对位置作为代理
        - 财务质量 (25%): 使用长期波动率作为代理
        - 成长性 (20%): 使用长期收益率作为代理
        """
        try:
            closes = np.array(
                [self.data.close[-i] for i in range(self.params.lookback_period, 0, -1)]
            )

            score = 0.0

            # 盈利能力 (30%) - 使用价格稳定性作为代理
            recent_volatility = np.std(closes[-10:]) / np.mean(closes[-10:])
            profitability_score = max(1 - recent_volatility * 10, 0)
            score += profitability_score * 0.3

            # 估值水平 (25%) - 使用价格相对位置作为代理
            current_price = closes[-1]
            price_range = np.max(closes) - np.min(closes)
            if price_range > 0:
                position = (current_price - np.min(closes)) / price_range
                # 低位估值得分高
                valuation_score = 1 - position
            else:
                valuation_score = 0.5
            score += valuation_score * 0.25

            # 财务质量 (25%) - 使用长期波动率作为代理
            if len(closes) >= 30:
                long_volatility = np.std(closes) / np.mean(closes)
                quality_score = max(1 - long_volatility * 8, 0)
            else:
                quality_score = 0.5
            score += quality_score * 0.25

            # 成长性 (20%) - 使用长期收益率作为代理
            if len(closes) >= 20:
                long_return = (closes[-1] - closes[0]) / closes[0]
                growth_score = min(max(long_return * 2 + 0.5, 0), 1)
            else:
                growth_score = 0.5
            score += growth_score * 0.2

            return min(max(score, 0), 1)

        except Exception as e:
            logger.error(f"计算基本面因子失败: {e}")
            return 0.5  # 默认中性评分

    def _calculate_news_factor(self) -> float:
        """计算消息面因子评分

        消息面因子包括：
        - 市场情绪 (40%): 使用近期价格变化作为代理
        - 新闻热度 (30%): 使用成交量变化作为代理
        - 事件影响 (30%): 使用价格跳空作为代理
        """
        try:
            closes = np.array(
                [self.data.close[-i] for i in range(min(10, len(self.data)), 0, -1)]
            )
            volumes = np.array(
                [self.data.volume[-i] for i in range(min(10, len(self.data)), 0, -1)]
            )

            if len(closes) < 2:
                return 0.5

            score = 0.0

            # 市场情绪 (40%) - 使用近期价格变化作为代理
            recent_returns = np.diff(closes) / closes[:-1]
            avg_return = np.mean(recent_returns)
            sentiment_score = min(max(avg_return * 10 + 0.5, 0), 1)
            score += sentiment_score * 0.4

            # 新闻热度 (30%) - 使用成交量变化作为代理
            if len(volumes) >= 5:
                recent_volume = np.mean(volumes[-3:])
                avg_volume = (
                    np.mean(volumes[:-3]) if len(volumes) > 3 else recent_volume
                )
                volume_ratio = recent_volume / avg_volume if avg_volume > 0 else 1
                heat_score = min(volume_ratio / 2, 1)  # 成交量放大2倍得满分
                score += heat_score * 0.3
            else:
                score += 0.5 * 0.3

            # 事件影响 (30%) - 使用价格跳空作为代理
            if len(closes) >= 3:
                # 检测价格跳空
                gap_ratio = abs(closes[-1] - closes[-2]) / closes[-2]
                if gap_ratio > 0.03:  # 3%以上的跳空
                    event_score = 0.8 if closes[-1] > closes[-2] else 0.2
                else:
                    event_score = 0.5
                score += event_score * 0.3
            else:
                score += 0.5 * 0.3

            return min(max(score, 0), 1)

        except Exception as e:
            logger.error(f"计算消息面因子失败: {e}")
            return 0.5  # 默认中性评分

    def _calculate_market_factor(self) -> float:
        """计算市场面因子评分

        市场面因子包括：
        - 市场表现 (40%): 相对于市场平均的表现
        - 资金流向 (30%): 基于成交量比率
        - 市场情绪 (20%): 短期价格动量
        - 板块轮动 (10%): 相对强度指标
        """
        try:
            closes = np.array(
                [self.data.close[-i] for i in range(self.params.lookback_period, 0, -1)]
            )
            volumes = np.array(
                [
                    self.data.volume[-i]
                    for i in range(self.params.lookback_period, 0, -1)
                ]
            )

            if len(closes) < 10:
                return 0.5

            score = 0.0

            # 市场表现 (40%) - 相对于假设的市场平均表现
            stock_return = (closes[-1] - closes[-10]) / closes[-10]
            # 假设市场平均收益率为0（实际应该从市场指数获取）
            market_return = 0.0
            relative_performance = stock_return - market_return
            performance_score = min(max(relative_performance * 3 + 0.5, 0), 1)
            score += performance_score * 0.4

            # 资金流向 (30%) - 基于成交量比率
            if len(volumes) >= 10:
                recent_volume = np.mean(volumes[-5:])
                avg_volume = np.mean(volumes[-10:-5])
                volume_ratio = recent_volume / avg_volume if avg_volume > 0 else 1
                flow_score = min(volume_ratio / 2.5, 1)  # 成交量放大2.5倍得满分
                score += flow_score * 0.3
            else:
                score += 0.5 * 0.3

            # 市场情绪 (20%) - 短期价格动量
            if len(closes) >= 5:
                short_momentum = (closes[-1] - closes[-3]) / closes[-3]
                sentiment_score = min(max(short_momentum * 8 + 0.5, 0), 1)
                score += sentiment_score * 0.2
            else:
                score += 0.5 * 0.2

            # 板块轮动 (10%) - 相对强度
            if len(closes) >= 20:
                long_return = (closes[-1] - closes[-20]) / closes[-20]
                short_return = (closes[-1] - closes[-5]) / closes[-5]
                relative_strength = (
                    short_return - long_return * 0.25
                )  # 短期相对长期的强度
                rotation_score = min(max(relative_strength * 4 + 0.5, 0), 1)
                score += rotation_score * 0.1
            else:
                score += 0.5 * 0.1

            return min(max(score, 0), 1)

        except Exception as e:
            logger.error(f"计算市场面因子失败: {e}")
            return 0.5  # 默认中性评分

    def _calculate_rsi(self, prices: np.ndarray, period: int = 14) -> float:
        """计算RSI指标"""
        try:
            if len(prices) < period + 1:
                return 50.0  # 默认中性值

            deltas = np.diff(prices)
            gains = np.where(deltas > 0, deltas, 0)
            losses = np.where(deltas < 0, -deltas, 0)

            avg_gain = np.mean(gains[-period:])
            avg_loss = np.mean(losses[-period:])

            if avg_loss == 0:
                return 100.0

            rs = avg_gain / avg_loss
            rsi = 100 - (100 / (1 + rs))

            return rsi

        except Exception:
            return 50.0

    def _generate_signal_from_score(self, factor_score: float) -> TradingSignal | None:
        """基于因子评分生成交易信号"""
        try:
            current_price = self.data.close[0]

            # 买入信号
            if factor_score >= self.params.buy_threshold:
                if not self.position:  # 没有持仓时才买入
                    confidence = min(factor_score, 1.0)
                    if confidence >= self.params.min_confidence_score:
                        size = self._calculate_position_size_by_score(factor_score)
                        return TradingSignal(
                            signal_type=SignalType.BUY,
                            price=current_price,
                            size=size,
                            reason=f"因子评分达到买入阈值: {factor_score:.3f} >= {self.params.buy_threshold}",
                            confidence=confidence,
                        )

            # 卖出信号
            elif factor_score <= self.params.sell_threshold and self.position:
                return TradingSignal(
                    signal_type=SignalType.SELL,
                    price=current_price,
                    reason=f"因子评分低于卖出阈值: {factor_score:.3f} <= {self.params.sell_threshold}",
                    confidence=1.0 - factor_score,
                )

            # 持有信号（不执行交易）
            return None

        except Exception as e:
            logger.error(f"生成交易信号失败: {e}")
            return None

    def _calculate_position_size_by_score(self, factor_score: float) -> int:
        """基于因子评分计算仓位大小"""
        try:
            # 基础仓位
            base_size = self.params.position_size

            # 根据评分调整仓位（评分越高，仓位越大）
            score_multiplier = factor_score  # 0-1之间
            adjusted_size = int(base_size * score_multiplier)

            # 应用最大仓位限制
            max_size = int(
                self.broker.get_value()
                * self.params.max_position_size
                / self.data.close[0]
            )
            final_size = min(adjusted_size, max_size)

            return max(final_size, 100)  # 最小100股

        except Exception as e:
            logger.error(f"计算仓位大小失败: {e}")
            return self.params.position_size

    def _apply_risk_management(self, signal: TradingSignal) -> TradingSignal | None:
        """应用风险管理规则"""
        try:
            # 检查止损
            if self.check_stop_loss() and self.position:
                return TradingSignal(
                    signal_type=SignalType.SELL,
                    price=self.data.close[0],
                    reason="触发止损",
                    confidence=1.0,
                )

            # 检查最大回撤
            if self._check_max_drawdown() and signal.signal_type == SignalType.BUY:
                logger.warning("达到最大回撤限制, 取消买入信号")
                return None

            # 检查持仓数量限制
            if signal.signal_type == SignalType.BUY:
                current_positions = len(
                    [d for d in self.datas if self.getposition(d).size > 0]
                )
                if current_positions >= self.params.max_positions:
                    logger.warning(f"达到最大持仓数量限制: {current_positions}")
                    return None

            return signal

        except Exception as e:
            logger.error(f"应用风险管理失败: {e}")
            return signal

    def _check_max_drawdown(self) -> bool:
        """检查最大回撤"""
        try:
            current_value = self.broker.get_value()
            self.portfolio_value_history.append(current_value)

            if len(self.portfolio_value_history) < 2:
                return False

            peak_value = max(self.portfolio_value_history)
            current_drawdown = (peak_value - current_value) / peak_value

            if current_drawdown >= self.params.max_drawdown_pct:
                logger.warning(f"达到最大回撤限制: {current_drawdown:.2%}")
                return True

            return False

        except Exception:
            return False

    def next(self):
        """策略主逻辑"""
        try:
            # 调用父类的next方法
            super().next()

            # 更新再平衡计数
            if self.order is None:  # 没有未完成订单时才更新
                signal = self.generate_signal()
                if signal:
                    self.last_rebalance_day = self.day_count

        except Exception as e:
            logger.error(f"策略执行失败: {e}")

    def get_strategy_stats(self) -> dict[str, Any]:
        """获取策略统计信息"""
        base_stats = super().get_strategy_stats()

        # 添加多因子策略特有的统计信息
        factor_stats = {
            "factor_scores_count": len(self.factor_scores_history),
            "avg_factor_score": np.mean(
                [s["score"] for s in self.factor_scores_history]
            )
            if self.factor_scores_history
            else 0,
            "rebalance_count": len(
                [s for s in self.signals if s.reason and "因子评分" in s.reason]
            ),
            "day_count": self.day_count,
            "last_rebalance_day": self.last_rebalance_day,
            "weights": {
                "technical": self.params.technical_weight,
                "fundamental": self.params.fundamental_weight,
                "news": self.params.news_weight,
                "market": self.params.market_weight,
            },
        }

        base_stats.update(factor_stats)
        return base_stats

    # ===== 从FactorService迁移的详细因子计算方法 =====

    def _calculate_momentum_factor(
        self, closes: np.ndarray, volumes: np.ndarray
    ) -> float:
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
                avg_volume = (
                    np.mean(volumes[:-3]) if len(volumes) > 3 else recent_volume
                )
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

    def _calculate_volatility_factor(
        self, closes: np.ndarray, volumes: np.ndarray
    ) -> float:
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
        self,
        closes: np.ndarray,
        highs: np.ndarray,
        lows: np.ndarray,
        volumes: np.ndarray,
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
