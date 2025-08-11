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
        """计算技术面因子评分"""
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

            score = 0.0

            # 动量因子 (20%)
            momentum = (closes[-1] - closes[0]) / closes[0]
            momentum_score = min(max(momentum * 2 + 0.5, 0), 1)  # 标准化到0-1
            score += momentum_score * 0.2

            # 移动平均因子 (25%)
            ma_5 = np.mean(closes[-5:])
            ma_20 = np.mean(closes)
            ma_signal = 1.0 if closes[-1] > ma_5 > ma_20 else 0.0
            score += ma_signal * 0.25

            # RSI因子 (20%)
            rsi = self._calculate_rsi(closes)
            rsi_score = 0.0
            if 30 <= rsi <= 70:  # 正常范围
                rsi_score = 0.7
            elif rsi < 30:  # 超卖
                rsi_score = 0.9
            elif rsi > 70:  # 超买
                rsi_score = 0.3
            score += rsi_score * 0.2

            # 波动率因子 (15%)
            volatility = np.std(closes) / np.mean(closes)
            vol_score = max(1 - volatility * 10, 0)  # 低波动率得分高
            score += vol_score * 0.15

            # 成交量因子 (20%)
            avg_volume = np.mean(volumes)
            current_volume = volumes[-1]
            volume_ratio = current_volume / avg_volume if avg_volume > 0 else 1
            volume_score = min(volume_ratio / 2, 1)  # 成交量放大得分高
            score += volume_score * 0.2

            return min(max(score, 0), 1)

        except Exception as e:
            logger.error(f"计算技术面因子失败: {e}")
            return 0.5  # 默认中性评分

    def _calculate_fundamental_factor(self) -> float:
        """计算基本面因子评分"""
        try:
            # 简化的基本面评分（实际应该从数据采集系统获取财务数据）
            # 这里使用价格相关的基本面指标作为代理

            closes = np.array(
                [self.data.close[-i] for i in range(self.params.lookback_period, 0, -1)]
            )

            score = 0.0

            # 价格趋势稳定性 (40%)
            price_stability = 1 - (np.std(closes) / np.mean(closes))
            score += max(price_stability, 0) * 0.4

            # 价格相对位置 (30%)
            current_price = closes[-1]
            price_range = np.max(closes) - np.min(closes)
            if price_range > 0:
                price_position = (current_price - np.min(closes)) / price_range
                score += price_position * 0.3

            # 长期趋势 (30%)
            long_term_return = (closes[-1] - closes[0]) / closes[0]
            trend_score = min(max(long_term_return + 0.5, 0), 1)
            score += trend_score * 0.3

            return min(max(score, 0), 1)

        except Exception as e:
            logger.error(f"计算基本面因子失败: {e}")
            return 0.5  # 默认中性评分

    def _calculate_news_factor(self) -> float:
        """计算消息面因子评分"""
        try:
            # 简化的消息面评分（实际应该从数据采集系统获取新闻情感数据）
            # 这里使用价格变化作为消息面的代理指标

            closes = np.array(
                [self.data.close[-i] for i in range(min(5, len(self.data)), 0, -1)]
            )

            if len(closes) < 2:
                return 0.5

            # 近期价格变化反映市场情绪
            recent_returns = np.diff(closes) / closes[:-1]
            avg_return = np.mean(recent_returns)

            # 将收益率转换为0-1评分
            news_score = min(max(avg_return * 10 + 0.5, 0), 1)

            return news_score

        except Exception as e:
            logger.error(f"计算消息面因子失败: {e}")
            return 0.5  # 默认中性评分

    def _calculate_market_factor(self) -> float:
        """计算市场面因子评分"""
        try:
            # 简化的市场面评分（实际应该考虑大盘走势、行业轮动等）
            # 这里使用相对强度作为市场面指标

            closes = np.array(
                [self.data.close[-i] for i in range(self.params.lookback_period, 0, -1)]
            )

            score = 0.0

            # 相对强度 (50%)
            stock_return = (closes[-1] - closes[0]) / closes[0]
            # 假设市场平均收益为0（实际应该获取大盘数据）
            market_return = 0.0
            relative_strength = stock_return - market_return
            rs_score = min(max(relative_strength * 2 + 0.5, 0), 1)
            score += rs_score * 0.5

            # 市场情绪 (30%)
            # 使用成交量变化作为市场情绪代理
            volumes = np.array(
                [
                    self.data.volume[-i]
                    for i in range(self.params.lookback_period, 0, -1)
                ]
            )
            volume_trend = np.polyfit(range(len(volumes)), volumes, 1)[0]
            volume_score = min(max(volume_trend / np.mean(volumes) + 0.5, 0), 1)
            score += volume_score * 0.3

            # 价格动量 (20%)
            momentum = (closes[-5:].mean() - closes[:5].mean()) / closes[:5].mean()
            momentum_score = min(max(momentum * 2 + 0.5, 0), 1)
            score += momentum_score * 0.2

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
