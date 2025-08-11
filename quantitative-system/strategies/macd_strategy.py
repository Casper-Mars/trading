"""MACD金叉策略实现

MACD（Moving Average Convergence Divergence）策略基于MACD指标的金叉和死叉信号进行交易。
当MACD线上穿信号线时产生金叉买入信号，当MACD线下穿信号线时产生死叉卖出信号。
"""

from typing import Any

import backtrader as bt
from loguru import logger

from strategies.base_strategy import BaseStrategy, SignalType, TradingSignal


class MACDStrategy(BaseStrategy):
    """MACD金叉策略

    策略参数：
    - fast_period: 快速EMA周期，默认12
    - slow_period: 慢速EMA周期，默认26
    - signal_period: 信号线EMA周期，默认9
    - min_histogram: 最小柱状图值，用于过滤弱信号，默认0.0
    """

    params = (
        ("fast_period", 12),
        ("slow_period", 26),
        ("signal_period", 9),
        ("min_histogram", 0.0),
    )

    def __init__(self):
        """初始化策略"""
        super().__init__()

        # 计算MACD指标
        self.macd = bt.indicators.MACD(
            self.data.close,
            period_me1=self.params.fast_period,
            period_me2=self.params.slow_period,
            period_signal=self.params.signal_period,
        )

        # MACD线、信号线和柱状图
        self.macd_line = self.macd.macd
        self.signal_line = self.macd.signal
        self.histogram = self.macd.histo

        # 记录上一次的金叉/死叉状态
        self.last_cross_state = None  # 'golden', 'death', None

        logger.info(
            f"MACD策略初始化完成 - 快速EMA: {self.params.fast_period}, "
            f"慢速EMA: {self.params.slow_period}, 信号线: {self.params.signal_period}"
        )

    def get_strategy_name(self) -> str:
        """获取策略名称"""
        return "MACDStrategy"

    def get_strategy_description(self) -> str:
        """获取策略描述"""
        return (
            f"MACD金叉策略 - MACD({self.params.fast_period},{self.params.slow_period}) "
            f"vs Signal({self.params.signal_period})"
        )

    def get_required_params(self) -> list[str]:
        """获取策略必需参数列表"""
        return ["fast_period", "slow_period", "signal_period"]

    def validate_params(self, **params) -> bool:
        """验证策略参数"""
        fast_period = params.get("fast_period", self.params.fast_period)
        slow_period = params.get("slow_period", self.params.slow_period)
        signal_period = params.get("signal_period", self.params.signal_period)
        min_histogram = params.get("min_histogram", self.params.min_histogram)

        # 验证周期参数
        if not isinstance(fast_period, int) or fast_period <= 0:
            logger.error(f"快速EMA周期必须是正整数: {fast_period}")
            return False

        if not isinstance(slow_period, int) or slow_period <= 0:
            logger.error(f"慢速EMA周期必须是正整数: {slow_period}")
            return False

        if not isinstance(signal_period, int) or signal_period <= 0:
            logger.error(f"信号线周期必须是正整数: {signal_period}")
            return False

        # 验证周期大小关系
        if fast_period >= slow_period:
            logger.error(
                f"快速EMA周期({fast_period})必须小于慢速EMA周期({slow_period})"
            )
            return False

        # 验证最小柱状图值
        if not isinstance(min_histogram, int | float):
            logger.error(f"最小柱状图值必须是数字: {min_histogram}")
            return False

        return True

    def get_cross_state(self) -> str | None:
        """获取当前金叉/死叉状态

        Returns:
            str | None: 'golden' (金叉), 'death' (死叉), None (无明确状态)
        """
        if len(self.data) < max(self.params.slow_period, self.params.signal_period) + 1:
            return None

        # 当前和前一个时间点的MACD线和信号线值
        macd_current = self.macd_line[0]
        macd_previous = self.macd_line[-1]
        signal_current = self.signal_line[0]
        signal_previous = self.signal_line[-1]

        # 检测金叉：MACD线从下方穿越信号线
        if macd_previous <= signal_previous and macd_current > signal_current:
            return "golden"

        # 检测死叉：MACD线从上方穿越信号线
        elif macd_previous >= signal_previous and macd_current < signal_current:
            return "death"

        return None

    def is_strong_signal(self) -> bool:
        """判断是否为强信号

        基于柱状图的绝对值来判断信号强度
        """
        if len(self.data) < max(self.params.slow_period, self.params.signal_period):
            return False

        return abs(self.histogram[0]) >= self.params.min_histogram

    def generate_signal(self) -> TradingSignal | None:
        """生成交易信号"""
        # 确保有足够的数据
        if len(self.data) < max(self.params.slow_period, self.params.signal_period) + 1:
            return None

        cross_state = self.get_cross_state()
        current_price = self.data.close[0]

        signal = None

        # 金叉买入信号
        if cross_state == "golden" and self.is_strong_signal():
            signal = TradingSignal(
                signal_type=SignalType.BUY,
                price=current_price,
                timestamp=self.data.datetime.datetime(0),
                confidence=0.80,
                metadata={
                    "macd_line": self.macd_line[0],
                    "signal_line": self.signal_line[0],
                    "histogram": self.histogram[0],
                    "cross_type": "golden",
                    "signal_reason": "MACD金叉 - MACD线上穿信号线",
                },
            )

            logger.info(
                f"生成买入信号 - MACD金叉, 价格: {current_price:.2f}, "
                f"MACD: {self.macd_line[0]:.4f}, 信号线: {self.signal_line[0]:.4f}, "
                f"柱状图: {self.histogram[0]:.4f}"
            )

        # 死叉卖出信号
        elif cross_state == "death" and self.is_strong_signal():
            signal = TradingSignal(
                signal_type=SignalType.SELL,
                price=current_price,
                timestamp=self.data.datetime.datetime(0),
                confidence=0.80,
                metadata={
                    "macd_line": self.macd_line[0],
                    "signal_line": self.signal_line[0],
                    "histogram": self.histogram[0],
                    "cross_type": "death",
                    "signal_reason": "MACD死叉 - MACD线下穿信号线",
                },
            )

            logger.info(
                f"生成卖出信号 - MACD死叉, 价格: {current_price:.2f}, "
                f"MACD: {self.macd_line[0]:.4f}, 信号线: {self.signal_line[0]:.4f}, "
                f"柱状图: {self.histogram[0]:.4f}"
            )

        # 更新交叉状态
        if cross_state:
            self.last_cross_state = cross_state

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

        # 检查止损
        self.check_stop_loss()

    def get_strategy_state(self) -> dict[str, Any]:
        """获取策略当前状态"""
        base_state = super().get_strategy_state()

        # 添加策略特有状态
        strategy_state = {
            "macd_line": float(self.macd_line[0])
            if len(self.data) >= max(self.params.slow_period, self.params.signal_period)
            else None,
            "signal_line": float(self.signal_line[0])
            if len(self.data) >= max(self.params.slow_period, self.params.signal_period)
            else None,
            "histogram": float(self.histogram[0])
            if len(self.data) >= max(self.params.slow_period, self.params.signal_period)
            else None,
            "cross_state": self.get_cross_state(),
            "last_cross_state": self.last_cross_state,
            "fast_period": self.params.fast_period,
            "slow_period": self.params.slow_period,
            "signal_period": self.params.signal_period,
            "min_histogram": self.params.min_histogram,
        }

        base_state.update(strategy_state)
        return base_state
