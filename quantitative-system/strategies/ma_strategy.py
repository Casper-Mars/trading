"""双均线策略实现

双均线策略是一种经典的技术分析策略，通过短期均线和长期均线的交叉来产生买卖信号。
当短期均线上穿长期均线时产生买入信号（金叉），当短期均线下穿长期均线时产生卖出信号（死叉）。
"""

from typing import Any

import backtrader as bt
from loguru import logger

from strategies.base_strategy import BaseStrategy, SignalType, TradingSignal


class MovingAverageStrategy(BaseStrategy):
    """双均线策略

    策略参数：
    - short_window: 短期均线窗口期，默认5
    - long_window: 长期均线窗口期，默认20
    - stop_loss_pct: 止损百分比，可选，默认None
    """

    params = (
        ("short_window", 5),
        ("long_window", 20),
        ("stop_loss_pct", None),
    )

    def __init__(self):
        """初始化策略"""
        super().__init__()

        # 计算移动平均线
        self.short_ma = bt.indicators.SimpleMovingAverage(
            self.data.close, period=self.params.short_window
        )
        self.long_ma = bt.indicators.SimpleMovingAverage(
            self.data.close, period=self.params.long_window
        )

        # 计算均线交叉信号
        self.crossover = bt.indicators.CrossOver(self.short_ma, self.long_ma)

        logger.info(
            f"双均线策略初始化完成 - 短期窗口: {self.params.short_window}, "
            f"长期窗口: {self.params.long_window}, 止损: {self.params.stop_loss_pct}"
        )

    def get_strategy_name(self) -> str:
        """获取策略名称"""
        return "MovingAverageStrategy"

    def get_strategy_description(self) -> str:
        """获取策略描述"""
        return f"双均线策略 - 短期MA({self.params.short_window}) vs 长期MA({self.params.long_window})"

    def get_required_params(self) -> list[str]:
        """获取策略必需参数列表"""
        return ["short_window", "long_window"]

    def validate_params(self, **params) -> bool:
        """验证策略参数"""
        short_window = params.get("short_window", self.params.short_window)
        long_window = params.get("long_window", self.params.long_window)
        stop_loss_pct = params.get("stop_loss_pct", self.params.stop_loss_pct)

        # 验证窗口期参数
        if not isinstance(short_window, int) or short_window <= 0:
            logger.error(f"短期窗口期必须是正整数: {short_window}")
            return False

        if not isinstance(long_window, int) or long_window <= 0:
            logger.error(f"长期窗口期必须是正整数: {long_window}")
            return False

        if short_window >= long_window:
            logger.error(f"短期窗口期({short_window})必须小于长期窗口期({long_window})")
            return False

        # 验证止损参数
        if stop_loss_pct is not None and (
            not isinstance(stop_loss_pct, int | float)
            or stop_loss_pct <= 0
            or stop_loss_pct >= 1
        ):
            logger.error(f"止损百分比必须在0-1之间: {stop_loss_pct}")
            return False

        return True

    def generate_signal(self) -> TradingSignal | None:
        """生成交易信号"""
        # 确保有足够的数据
        if len(self.data) < self.params.long_window:
            return None

        current_price = self.data.close[0]
        short_ma_value = self.short_ma[0]
        long_ma_value = self.long_ma[0]

        # 检查金叉信号（买入）
        if self.crossover[0] > 0:
            signal = TradingSignal(
                signal_type=SignalType.BUY,
                price=current_price,
                timestamp=self.data.datetime.datetime(0),
                confidence=0.8,
                metadata={
                    "short_ma": short_ma_value,
                    "long_ma": long_ma_value,
                    "signal_reason": "金叉 - 短期均线上穿长期均线",
                    "crossover_value": self.crossover[0],
                },
            )

            logger.info(
                f"生成买入信号 - 价格: {current_price:.2f}, "
                f"短期MA: {short_ma_value:.2f}, 长期MA: {long_ma_value:.2f}"
            )

            return signal

        # 检查死叉信号（卖出）
        elif self.crossover[0] < 0:
            signal = TradingSignal(
                signal_type=SignalType.SELL,
                price=current_price,
                timestamp=self.data.datetime.datetime(0),
                confidence=0.8,
                metadata={
                    "short_ma": short_ma_value,
                    "long_ma": long_ma_value,
                    "signal_reason": "死叉 - 短期均线下穿长期均线",
                    "crossover_value": self.crossover[0],
                },
            )

            logger.info(
                f"生成卖出信号 - 价格: {current_price:.2f}, "
                f"短期MA: {short_ma_value:.2f}, 长期MA: {long_ma_value:.2f}"
            )

            return signal

        return None

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

        # 检查止损
        if self.params.stop_loss_pct and self.position and self.entry_price:
            self.check_stop_loss(self.params.stop_loss_pct)

    def get_strategy_state(self) -> dict[str, Any]:
        """获取策略当前状态"""
        base_state = super().get_strategy_state()

        # 添加策略特有状态
        strategy_state = {
            "short_ma_current": float(self.short_ma[0])
            if len(self.data) >= self.params.short_window
            else None,
            "long_ma_current": float(self.long_ma[0])
            if len(self.data) >= self.params.long_window
            else None,
            "crossover_current": float(self.crossover[0])
            if len(self.data) >= self.params.long_window
            else None,
            "short_window": self.params.short_window,
            "long_window": self.params.long_window,
            "stop_loss_pct": self.params.stop_loss_pct,
        }

        base_state.update(strategy_state)
        return base_state
