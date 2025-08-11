"""三重均线策略实现

三重均线策略使用三条不同周期的移动平均线来判断趋势方向和交易时机。
当三条均线呈多头排列（短期>中期>长期）时看多，呈空头排列（短期<中期<长期）时看空。
"""

from typing import Any

import backtrader as bt
from loguru import logger

from strategies.base_strategy import BaseStrategy, SignalType, TradingSignal


class TripleMovingAverageStrategy(BaseStrategy):
    """三重均线策略

    策略参数：
    - short_window: 短期均线窗口期，默认5
    - mid_window: 中期均线窗口期，默认10
    - long_window: 长期均线窗口期，默认20
    """

    params = (
        ("short_window", 5),
        ("mid_window", 10),
        ("long_window", 20),
    )

    def __init__(self):
        """初始化策略"""
        super().__init__()

        # 计算三条移动平均线
        self.short_ma = bt.indicators.SimpleMovingAverage(
            self.data.close, period=self.params.short_window
        )
        self.mid_ma = bt.indicators.SimpleMovingAverage(
            self.data.close, period=self.params.mid_window
        )
        self.long_ma = bt.indicators.SimpleMovingAverage(
            self.data.close, period=self.params.long_window
        )

        # 记录上一次的排列状态
        self.last_arrangement = None  # 'bullish', 'bearish', 'neutral'

        logger.info(
            f"三重均线策略初始化完成 - 短期: {self.params.short_window}, "
            f"中期: {self.params.mid_window}, 长期: {self.params.long_window}"
        )

    def get_strategy_name(self) -> str:
        """获取策略名称"""
        return "TripleMovingAverageStrategy"

    def get_strategy_description(self) -> str:
        """获取策略描述"""
        return (
            f"三重均线策略 - MA({self.params.short_window}) vs "
            f"MA({self.params.mid_window}) vs MA({self.params.long_window})"
        )

    def get_required_params(self) -> list[str]:
        """获取策略必需参数列表"""
        return ["short_window", "mid_window", "long_window"]

    def validate_params(self, **params) -> bool:
        """验证策略参数"""
        short_window = params.get("short_window", self.params.short_window)
        mid_window = params.get("mid_window", self.params.mid_window)
        long_window = params.get("long_window", self.params.long_window)

        # 验证窗口期参数
        if not isinstance(short_window, int) or short_window <= 0:
            logger.error(f"短期窗口期必须是正整数: {short_window}")
            return False

        if not isinstance(mid_window, int) or mid_window <= 0:
            logger.error(f"中期窗口期必须是正整数: {mid_window}")
            return False

        if not isinstance(long_window, int) or long_window <= 0:
            logger.error(f"长期窗口期必须是正整数: {long_window}")
            return False

        # 验证窗口期大小关系
        if not (short_window < mid_window < long_window):
            logger.error(
                f"窗口期必须满足: 短期({short_window}) < 中期({mid_window}) < 长期({long_window})"
            )
            return False

        return True

    def get_ma_arrangement(self) -> str:
        """获取当前均线排列状态

        Returns:
            str: 'bullish' (多头排列), 'bearish' (空头排列), 'neutral' (中性)
        """
        if len(self.data) < self.params.long_window:
            return "neutral"

        short_value = self.short_ma[0]
        mid_value = self.mid_ma[0]
        long_value = self.long_ma[0]

        # 多头排列：短期 > 中期 > 长期
        if short_value > mid_value > long_value:
            return "bullish"
        # 空头排列：短期 < 中期 < 长期
        elif short_value < mid_value < long_value:
            return "bearish"
        else:
            return "neutral"

    def generate_signal(self) -> TradingSignal | None:
        """生成交易信号"""
        # 确保有足够的数据
        if len(self.data) < self.params.long_window:
            return None

        current_arrangement = self.get_ma_arrangement()
        current_price = self.data.close[0]

        signal = None

        # 从非多头排列转为多头排列 -> 买入信号
        if self.last_arrangement != "bullish" and current_arrangement == "bullish":
            signal = TradingSignal(
                signal_type=SignalType.BUY,
                price=current_price,
                timestamp=self.data.datetime.datetime(0),
                confidence=0.85,
                metadata={
                    "short_ma": self.short_ma[0],
                    "mid_ma": self.mid_ma[0],
                    "long_ma": self.long_ma[0],
                    "arrangement": current_arrangement,
                    "last_arrangement": self.last_arrangement,
                    "signal_reason": "转为多头排列 - 短期MA > 中期MA > 长期MA",
                },
            )

            logger.info(
                f"生成买入信号 - 多头排列形成, 价格: {current_price:.2f}, "
                f"MA({self.params.short_window}): {self.short_ma[0]:.2f}, "
                f"MA({self.params.mid_window}): {self.mid_ma[0]:.2f}, "
                f"MA({self.params.long_window}): {self.long_ma[0]:.2f}"
            )

        # 从非空头排列转为空头排列 -> 卖出信号
        elif self.last_arrangement != "bearish" and current_arrangement == "bearish":
            signal = TradingSignal(
                signal_type=SignalType.SELL,
                price=current_price,
                timestamp=self.data.datetime.datetime(0),
                confidence=0.85,
                metadata={
                    "short_ma": self.short_ma[0],
                    "mid_ma": self.mid_ma[0],
                    "long_ma": self.long_ma[0],
                    "arrangement": current_arrangement,
                    "last_arrangement": self.last_arrangement,
                    "signal_reason": "转为空头排列 - 短期MA < 中期MA < 长期MA",
                },
            )

            logger.info(
                f"生成卖出信号 - 空头排列形成, 价格: {current_price:.2f}, "
                f"MA({self.params.short_window}): {self.short_ma[0]:.2f}, "
                f"MA({self.params.mid_window}): {self.mid_ma[0]:.2f}, "
                f"MA({self.params.long_window}): {self.long_ma[0]:.2f}"
            )

        # 更新排列状态
        self.last_arrangement = current_arrangement

        return signal

    def next(self):
        """策略主逻辑"""
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

                logger.info(f"执行买入订单 - 数量: {size}, 价格: {signal.price:.2f}")

            elif signal.signal_type == SignalType.SELL and self.position:
                # 卖出信号且当前有持仓
                self.order = self.sell(size=self.position.size)

                logger.info(
                    f"执行卖出订单 - 数量: {self.position.size}, 价格: {signal.price:.2f}"
                )

    def get_strategy_state(self) -> dict[str, Any]:
        """获取策略当前状态"""
        base_state = super().get_strategy_state()

        # 添加策略特有状态
        strategy_state = {
            "short_ma_current": float(self.short_ma[0])
            if len(self.data) >= self.params.short_window
            else None,
            "mid_ma_current": float(self.mid_ma[0])
            if len(self.data) >= self.params.mid_window
            else None,
            "long_ma_current": float(self.long_ma[0])
            if len(self.data) >= self.params.long_window
            else None,
            "ma_arrangement": self.get_ma_arrangement(),
            "last_arrangement": self.last_arrangement,
            "short_window": self.params.short_window,
            "mid_window": self.params.mid_window,
            "long_window": self.params.long_window,
        }

        base_state.update(strategy_state)
        return base_state
