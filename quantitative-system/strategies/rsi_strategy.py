"""RSI反转策略实现

RSI（Relative Strength Index）策略基于RSI指标的超买超卖信号进行反转交易。
当RSI低于超卖线时产生买入信号，当RSI高于超买线时产生卖出信号。
"""

from typing import Any

import backtrader as bt
from loguru import logger

from strategies.base_strategy import BaseStrategy, SignalType, TradingSignal


class RSIStrategy(BaseStrategy):
    """RSI反转策略

    策略参数：
    - period: RSI计算周期，默认14
    - oversold_level: 超卖线，默认30
    - overbought_level: 超买线，默认70
    - extreme_oversold: 极度超卖线，默认20
    - extreme_overbought: 极度超买线，默认80
    - min_holding_period: 最小持仓周期，默认3
    """

    params = (
        ("period", 14),
        ("oversold_level", 30),
        ("overbought_level", 70),
        ("extreme_oversold", 20),
        ("extreme_overbought", 80),
        ("min_holding_period", 3),
    )

    def __init__(self):
        """初始化策略"""
        super().__init__()

        # 计算RSI指标
        self.rsi = bt.indicators.RSI(self.data.close, period=self.params.period)

        # 记录持仓时间
        self.holding_days = 0

        # 记录上一次的RSI状态
        self.last_rsi_state = None  # 'oversold', 'overbought', 'normal'

        # 记录信号确认
        self.signal_confirmed = False

        logger.info(
            f"RSI策略初始化完成 - 周期: {self.params.period}, "
            f"超卖线: {self.params.oversold_level}, 超买线: {self.params.overbought_level}"
        )

    def get_strategy_name(self) -> str:
        """获取策略名称"""
        return "RSIStrategy"

    def get_strategy_description(self) -> str:
        """获取策略描述"""
        return (
            f"RSI反转策略 - RSI({self.params.period}) "
            f"超卖: {self.params.oversold_level}, 超买: {self.params.overbought_level}"
        )

    def get_required_params(self) -> list[str]:
        """获取策略必需参数列表"""
        return ["period", "oversold_level", "overbought_level"]

    def validate_params(self, **params) -> bool:
        """验证策略参数"""
        period = params.get("period", self.params.period)
        oversold_level = params.get("oversold_level", self.params.oversold_level)
        overbought_level = params.get("overbought_level", self.params.overbought_level)
        extreme_oversold = params.get("extreme_oversold", self.params.extreme_oversold)
        extreme_overbought = params.get(
            "extreme_overbought", self.params.extreme_overbought
        )
        min_holding_period = params.get(
            "min_holding_period", self.params.min_holding_period
        )

        # 验证周期参数
        if not isinstance(period, int) or period <= 0:
            logger.error(f"RSI周期必须是正整数: {period}")
            return False

        # 验证RSI水平参数
        if not (0 <= oversold_level <= 100):
            logger.error(f"超卖线必须在0-100之间: {oversold_level}")
            return False

        if not (0 <= overbought_level <= 100):
            logger.error(f"超买线必须在0-100之间: {overbought_level}")
            return False

        if oversold_level >= overbought_level:
            logger.error(f"超卖线({oversold_level})必须小于超买线({overbought_level})")
            return False

        # 验证极值参数
        if not (0 <= extreme_oversold <= oversold_level):
            logger.error(
                f"极度超卖线({extreme_oversold})必须小于等于超卖线({oversold_level})"
            )
            return False

        if not (overbought_level <= extreme_overbought <= 100):
            logger.error(
                f"极度超买线({extreme_overbought})必须大于等于超买线({overbought_level})"
            )
            return False

        # 验证最小持仓周期
        if not isinstance(min_holding_period, int) or min_holding_period < 0:
            logger.error(f"最小持仓周期必须是非负整数: {min_holding_period}")
            return False

        return True

    def get_rsi_state(self) -> str:
        """获取当前RSI状态

        Returns:
            str: 'extreme_oversold', 'oversold', 'normal', 'overbought', 'extreme_overbought'
        """
        if len(self.data) < self.params.period:
            return "normal"

        rsi_value = self.rsi[0]

        if rsi_value <= self.params.extreme_oversold:
            return "extreme_oversold"
        elif rsi_value <= self.params.oversold_level:
            return "oversold"
        elif rsi_value >= self.params.extreme_overbought:
            return "extreme_overbought"
        elif rsi_value >= self.params.overbought_level:
            return "overbought"
        else:
            return "normal"

    def is_rsi_divergence(self) -> tuple[bool, str | None]:
        """检测RSI背离

        Returns:
            tuple: (是否存在背离, 背离类型)
        """
        if len(self.data) < self.params.period + 5:
            return False, None

        # 简单的背离检测：比较最近5个周期的价格和RSI趋势
        price_trend = self.data.close[0] - self.data.close[-5]
        rsi_trend = self.rsi[0] - self.rsi[-5]

        # 顶背离：价格创新高但RSI未创新高
        if (
            price_trend > 0
            and rsi_trend < 0
            and self.rsi[0] > self.params.overbought_level
        ):
            return True, "bearish"

        # 底背离：价格创新低但RSI未创新低
        if (
            price_trend < 0
            and rsi_trend > 0
            and self.rsi[0] < self.params.oversold_level
        ):
            return True, "bullish"

        return False, None

    def generate_signal(self) -> TradingSignal | None:
        """生成交易信号"""
        # 确保有足够的数据
        if len(self.data) < self.params.period:
            return None

        rsi_state = self.get_rsi_state()
        current_price = self.data.close[0]
        rsi_value = self.rsi[0]

        # 检测背离
        has_divergence, divergence_type = self.is_rsi_divergence()

        signal = None

        # 超卖买入信号
        if rsi_state in [
            "oversold",
            "extreme_oversold",
        ] and self.last_rsi_state not in ["oversold", "extreme_oversold"]:
            # 计算信号强度
            confidence = 0.70
            if rsi_state == "extreme_oversold":
                confidence = 0.85
            if has_divergence and divergence_type == "bullish":
                confidence = min(confidence + 0.10, 0.95)

            signal = TradingSignal(
                signal_type=SignalType.BUY,
                price=current_price,
                timestamp=self.data.datetime.datetime(0),
                confidence=confidence,
                metadata={
                    "rsi_value": rsi_value,
                    "rsi_state": rsi_state,
                    "has_divergence": has_divergence,
                    "divergence_type": divergence_type,
                    "signal_reason": f"RSI超卖反转 - RSI: {rsi_value:.2f}",
                },
            )

            logger.info(
                f"生成买入信号 - RSI超卖, 价格: {current_price:.2f}, "
                f"RSI: {rsi_value:.2f}, 状态: {rsi_state}"
            )

        # 超买卖出信号
        elif rsi_state in [
            "overbought",
            "extreme_overbought",
        ] and self.last_rsi_state not in ["overbought", "extreme_overbought"]:
            # 计算信号强度
            confidence = 0.70
            if rsi_state == "extreme_overbought":
                confidence = 0.85
            if has_divergence and divergence_type == "bearish":
                confidence = min(confidence + 0.10, 0.95)

            signal = TradingSignal(
                signal_type=SignalType.SELL,
                price=current_price,
                timestamp=self.data.datetime.datetime(0),
                confidence=confidence,
                metadata={
                    "rsi_value": rsi_value,
                    "rsi_state": rsi_state,
                    "has_divergence": has_divergence,
                    "divergence_type": divergence_type,
                    "signal_reason": f"RSI超买反转 - RSI: {rsi_value:.2f}",
                },
            )

            logger.info(
                f"生成卖出信号 - RSI超买, 价格: {current_price:.2f}, "
                f"RSI: {rsi_value:.2f}, 状态: {rsi_state}"
            )

        # 更新RSI状态
        self.last_rsi_state = rsi_state

        return signal

    def next(self):
        """策略主逻辑"""
        # 更新持仓天数
        if self.position:
            self.holding_days += 1
        else:
            self.holding_days = 0

        # 生成交易信号
        signal = self.generate_signal()

        if signal:
            self.signals.append(signal)

            # 执行交易逻辑
            if signal.signal_type == SignalType.BUY and not self.position:
                # 买入信号且当前无持仓
                size = self.calculate_position_size(signal.price)
                self.order = self.buy(size=size)
                self.entry_price = signal.price
                self.holding_days = 0

                logger.info(f"执行买入订单 - 数量: {size}, 价格: {signal.price:.2f}")

            elif signal.signal_type == SignalType.SELL and self.position:
                # 卖出信号且当前有持仓
                self.order = self.sell(size=self.position.size)

                logger.info(
                    f"执行卖出订单 - 数量: {self.position.size}, 价格: {signal.price:.2f}"
                )

        # 检查止损和RSI回归
        self.check_stop_loss()
        self.check_rsi_reversion()

    def check_rsi_reversion(self):
        """检查RSI回归（RSI回到正常区间时考虑平仓）"""
        if (
            not self.position
            or len(self.data) < self.params.period
            or self.holding_days < self.params.min_holding_period
        ):
            return

        rsi_value = self.rsi[0]
        current_price = self.data.close[0]

        # 多头持仓：RSI回到正常区间且有盈利时平仓
        if self.position.size > 0:
            profit_pct = (current_price - self.entry_price) / self.entry_price

            # RSI从超卖区间回到正常区间，且盈利超过3%
            if rsi_value > self.params.oversold_level + 10 and profit_pct > 0.03:
                self.order = self.sell(size=self.position.size)
                logger.info(
                    f"RSI回归平仓 - RSI回到正常区间, 盈利: {profit_pct:.2%}, "
                    f"RSI: {rsi_value:.2f}"
                )

        # 空头持仓：RSI回到正常区间且有盈利时平仓
        elif self.position.size < 0:
            profit_pct = (self.entry_price - current_price) / self.entry_price

            # RSI从超买区间回到正常区间，且盈利超过3%
            if rsi_value < self.params.overbought_level - 10 and profit_pct > 0.03:
                self.order = self.buy(size=abs(self.position.size))
                logger.info(
                    f"RSI回归平仓 - RSI回到正常区间, 盈利: {profit_pct:.2%}, "
                    f"RSI: {rsi_value:.2f}"
                )

    def get_strategy_state(self) -> dict[str, Any]:
        """获取策略当前状态"""
        base_state = super().get_strategy_state()

        # 检测背离
        has_divergence, divergence_type = self.is_rsi_divergence()

        # 添加策略特有状态
        strategy_state = {
            "rsi_value": float(self.rsi[0])
            if len(self.data) >= self.params.period
            else None,
            "rsi_state": self.get_rsi_state(),
            "last_rsi_state": self.last_rsi_state,
            "holding_days": self.holding_days,
            "has_divergence": has_divergence,
            "divergence_type": divergence_type,
            "period": self.params.period,
            "oversold_level": self.params.oversold_level,
            "overbought_level": self.params.overbought_level,
            "extreme_oversold": self.params.extreme_oversold,
            "extreme_overbought": self.params.extreme_overbought,
            "min_holding_period": self.params.min_holding_period,
        }

        base_state.update(strategy_state)
        return base_state
