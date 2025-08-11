"""布林带突破策略实现

布林带策略基于布林带指标进行交易，当价格突破上轨时产生买入信号，
当价格跌破下轨时产生卖出信号。同时考虑价格回归中轨的特性。
"""

from typing import Any

import backtrader as bt
from loguru import logger

from strategies.base_strategy import BaseStrategy, SignalType, TradingSignal


class BollingerBandsStrategy(BaseStrategy):
    """布林带突破策略

    策略参数：
    - period: 移动平均线周期，默认20
    - devfactor: 标准差倍数，默认2.0
    - min_volume_ratio: 最小成交量比率，用于过滤信号，默认1.2
    - breakout_threshold: 突破阈值百分比，默认0.001 (0.1%)
    """

    params = (
        ("period", 20),
        ("devfactor", 2.0),
        ("min_volume_ratio", 1.2),
        ("breakout_threshold", 0.001),
    )

    def __init__(self):
        """初始化策略"""
        super().__init__()

        # 计算布林带指标
        self.bollinger = bt.indicators.BollingerBands(
            self.data.close, period=self.params.period, devfactor=self.params.devfactor
        )

        # 布林带的上轨、中轨、下轨
        self.bb_top = self.bollinger.top
        self.bb_mid = self.bollinger.mid
        self.bb_bot = self.bollinger.bot

        # 计算布林带宽度（用于判断市场波动性）
        self.bb_width = (self.bb_top - self.bb_bot) / self.bb_mid

        # 计算成交量移动平均（用于成交量过滤）
        self.volume_ma = bt.indicators.SimpleMovingAverage(
            self.data.volume, period=self.params.period
        )

        # 记录上一次的突破状态
        self.last_breakout = None  # 'upper', 'lower', None

        logger.info(
            f"布林带策略初始化完成 - 周期: {self.params.period}, "
            f"标准差倍数: {self.params.devfactor}"
        )

    def get_strategy_name(self) -> str:
        """获取策略名称"""
        return "BollingerBandsStrategy"

    def get_strategy_description(self) -> str:
        """获取策略描述"""
        return f"布林带突破策略 - BB({self.params.period}, {self.params.devfactor})"

    def get_required_params(self) -> list[str]:
        """获取策略必需参数列表"""
        return ["period", "devfactor"]

    def validate_params(self, **params) -> bool:
        """验证策略参数"""
        period = params.get("period", self.params.period)
        devfactor = params.get("devfactor", self.params.devfactor)
        min_volume_ratio = params.get("min_volume_ratio", self.params.min_volume_ratio)
        breakout_threshold = params.get(
            "breakout_threshold", self.params.breakout_threshold
        )

        # 验证周期参数
        if not isinstance(period, int) or period <= 0:
            logger.error(f"周期必须是正整数: {period}")
            return False

        # 验证标准差倍数
        if not isinstance(devfactor, int | float) or devfactor <= 0:
            logger.error(f"标准差倍数必须是正数: {devfactor}")
            return False

        # 验证成交量比率
        if not isinstance(min_volume_ratio, int | float) or min_volume_ratio <= 0:
            logger.error(f"最小成交量比率必须是正数: {min_volume_ratio}")
            return False

        # 验证突破阈值
        if not isinstance(breakout_threshold, int | float) or breakout_threshold < 0:
            logger.error(f"突破阈值必须是非负数: {breakout_threshold}")
            return False

        return True

    def get_price_position(self) -> str:
        """获取价格在布林带中的位置

        Returns:
            str: 'above_upper' (上轨上方), 'below_lower' (下轨下方),
                 'between' (上下轨之间), 'on_upper' (接近上轨), 'on_lower' (接近下轨)
        """
        if len(self.data) < self.params.period:
            return "between"

        current_price = self.data.close[0]
        upper_band = self.bb_top[0]
        lower_band = self.bb_bot[0]

        # 计算突破阈值
        threshold = self.params.breakout_threshold

        if current_price > upper_band * (1 + threshold):
            return "above_upper"
        elif current_price < lower_band * (1 - threshold):
            return "below_lower"
        elif current_price > upper_band * (1 - threshold):
            return "on_upper"
        elif current_price < lower_band * (1 + threshold):
            return "on_lower"
        else:
            return "between"

    def is_high_volume(self) -> bool:
        """判断是否为高成交量

        基于成交量移动平均来判断
        """
        if len(self.data) < self.params.period:
            return False

        current_volume = self.data.volume[0]
        avg_volume = self.volume_ma[0]

        return current_volume >= avg_volume * self.params.min_volume_ratio

    def is_high_volatility(self) -> bool:
        """判断是否为高波动性环境

        基于布林带宽度来判断
        """
        if len(self.data) < self.params.period:
            return False

        # 布林带宽度大于5%认为是高波动性
        return self.bb_width[0] > 0.05

    def generate_signal(self) -> TradingSignal | None:
        """生成交易信号"""
        # 确保有足够的数据
        if len(self.data) < self.params.period:
            return None

        price_position = self.get_price_position()
        current_price = self.data.close[0]

        signal = None

        # 上轨突破买入信号
        if (
            price_position == "above_upper"
            and self.last_breakout != "upper"
            and self.is_high_volume()
            and self.is_high_volatility()
        ):
            signal = TradingSignal(
                signal_type=SignalType.BUY,
                price=current_price,
                timestamp=self.data.datetime.datetime(0),
                confidence=0.75,
                metadata={
                    "bb_upper": self.bb_top[0],
                    "bb_mid": self.bb_mid[0],
                    "bb_lower": self.bb_bot[0],
                    "bb_width": self.bb_width[0],
                    "price_position": price_position,
                    "volume_ratio": self.data.volume[0] / self.volume_ma[0],
                    "signal_reason": "价格突破布林带上轨",
                },
            )

            logger.info(
                f"生成买入信号 - 突破上轨, 价格: {current_price:.2f}, "
                f"上轨: {self.bb_top[0]:.2f}, 中轨: {self.bb_mid[0]:.2f}, "
                f"下轨: {self.bb_bot[0]:.2f}"
            )

            self.last_breakout = "upper"

        # 下轨突破卖出信号
        elif (
            price_position == "below_lower"
            and self.last_breakout != "lower"
            and self.is_high_volume()
            and self.is_high_volatility()
        ):
            signal = TradingSignal(
                signal_type=SignalType.SELL,
                price=current_price,
                timestamp=self.data.datetime.datetime(0),
                confidence=0.75,
                metadata={
                    "bb_upper": self.bb_top[0],
                    "bb_mid": self.bb_mid[0],
                    "bb_lower": self.bb_bot[0],
                    "bb_width": self.bb_width[0],
                    "price_position": price_position,
                    "volume_ratio": self.data.volume[0] / self.volume_ma[0],
                    "signal_reason": "价格跌破布林带下轨",
                },
            )

            logger.info(
                f"生成卖出信号 - 跌破下轨, 价格: {current_price:.2f}, "
                f"上轨: {self.bb_top[0]:.2f}, 中轨: {self.bb_mid[0]:.2f}, "
                f"下轨: {self.bb_bot[0]:.2f}"
            )

            self.last_breakout = "lower"

        # 重置突破状态（当价格回到布林带内部时）
        elif price_position == "between":
            self.last_breakout = None

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

        # 检查止损和中轨回归
        self.check_stop_loss()
        self.check_mean_reversion()

    def check_mean_reversion(self):
        """检查均值回归（价格回到中轨附近时平仓）"""
        if not self.position or len(self.data) < self.params.period:
            return

        current_price = self.data.close[0]
        mid_band = self.bb_mid[0]

        # 如果价格接近中轨（±1%范围内），考虑平仓
        if abs(current_price - mid_band) / mid_band < 0.01:
            if self.position.size > 0:  # 多头持仓
                profit_pct = (current_price - self.entry_price) / self.entry_price
                if profit_pct > 0.02:  # 盈利超过2%时平仓
                    self.order = self.sell(size=self.position.size)
                    logger.info(
                        f"均值回归平仓 - 价格回到中轨附近, 盈利: {profit_pct:.2%}"
                    )

            elif self.position.size < 0:  # 空头持仓
                profit_pct = (self.entry_price - current_price) / self.entry_price
                if profit_pct > 0.02:  # 盈利超过2%时平仓
                    self.order = self.buy(size=abs(self.position.size))
                    logger.info(
                        f"均值回归平仓 - 价格回到中轨附近, 盈利: {profit_pct:.2%}"
                    )

    def get_strategy_state(self) -> dict[str, Any]:
        """获取策略当前状态"""
        base_state = super().get_strategy_state()

        # 添加策略特有状态
        strategy_state = {
            "bb_upper": float(self.bb_top[0])
            if len(self.data) >= self.params.period
            else None,
            "bb_mid": float(self.bb_mid[0])
            if len(self.data) >= self.params.period
            else None,
            "bb_lower": float(self.bb_bot[0])
            if len(self.data) >= self.params.period
            else None,
            "bb_width": float(self.bb_width[0])
            if len(self.data) >= self.params.period
            else None,
            "price_position": self.get_price_position(),
            "last_breakout": self.last_breakout,
            "is_high_volume": self.is_high_volume(),
            "is_high_volatility": self.is_high_volatility(),
            "period": self.params.period,
            "devfactor": self.params.devfactor,
            "min_volume_ratio": self.params.min_volume_ratio,
            "breakout_threshold": self.params.breakout_threshold,
        }

        base_state.update(strategy_state)
        return base_state
