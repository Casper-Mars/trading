"""风险管理策略模块

实现多种风险管理策略，包括仓位管理、止损策略等。
"""

from abc import ABC, abstractmethod

import backtrader as bt
from loguru import logger

from .base_strategy import BaseStrategy, SignalType, TradingSignal


class RiskManagementMixin(ABC):
    """风险管理混入类

    提供通用的风险管理功能，可以被其他策略继承使用。
    """

    @abstractmethod
    def calculate_position_size(self, signal: TradingSignal) -> int:
        """计算仓位大小

        Args:
            signal: 交易信号

        Returns:
            int: 仓位大小
        """
        pass

    @abstractmethod
    def check_stop_loss_condition(self) -> bool:
        """检查止损条件

        Returns:
            bool: 是否触发止损
        """
        pass


class EqualWeightStrategy(BaseStrategy, RiskManagementMixin):
    """等权重仓位策略

    将可用资金平均分配给所有持仓，确保每个仓位的权重相等。
    """

    # 策略参数
    params = (
        ("weight_per_position", 0.2),  # 每个仓位的权重
        ("min_trade_unit", 100),  # 最小交易单位
        ("max_positions", 5),  # 最大持仓数量
        ("rebalance_threshold", 0.05),  # 再平衡阈值
    )

    def __init__(self):
        """初始化策略"""
        super().__init__()
        self.target_positions = {}  # 目标仓位
        self.last_rebalance_value = 0  # 上次再平衡时的组合价值

        logger.info(
            f"等权重策略初始化完成, 每仓位权重: {self.params.weight_per_position}, "
            f"最大持仓数: {self.params.max_positions}"
        )

    def get_strategy_name(self) -> str:
        """获取策略名称"""
        return "EqualWeightStrategy"

    def get_strategy_description(self) -> str:
        """获取策略描述"""
        return "等权重仓位管理策略, 将资金平均分配给所有持仓"

    def get_required_params(self) -> list[str]:
        """获取策略必需参数列表"""
        return ["weight_per_position", "max_positions"]

    def validate_params(self) -> bool:
        """验证策略参数是否有效"""
        try:
            # 验证权重范围
            if not (0 < self.params.weight_per_position <= 1):
                logger.error(
                    f"每仓位权重必须在0-1范围内: {self.params.weight_per_position}"
                )
                return False

            # 验证最大持仓数量
            if self.params.max_positions <= 0:
                logger.error(f"最大持仓数量必须大于0: {self.params.max_positions}")
                return False

            # 验证权重与持仓数量的一致性
            total_weight = self.params.weight_per_position * self.params.max_positions
            if total_weight > 1.0:
                logger.warning(f"总权重超过100%: {total_weight:.2%}, 将调整单仓位权重")
                self.params.weight_per_position = 1.0 / self.params.max_positions

            # 验证最小交易单位
            if self.params.min_trade_unit <= 0:
                logger.error(f"最小交易单位必须大于0: {self.params.min_trade_unit}")
                return False

            return True

        except Exception as e:
            logger.error(f"参数验证失败: {e}")
            return False

    def calculate_position_size(self, signal: TradingSignal) -> int:
        """计算等权重仓位大小

        Args:
            signal: 交易信号

        Returns:
            int: 仓位大小
        """
        try:
            # 获取当前组合价值
            portfolio_value = self.broker.get_value()

            # 计算目标仓位价值
            target_position_value = portfolio_value * self.params.weight_per_position

            # 计算股数
            shares = int(target_position_value / signal.price)

            # 应用最小交易单位约束
            shares = (shares // self.params.min_trade_unit) * self.params.min_trade_unit

            # 确保不超过可用资金
            max_affordable_shares = int(self.broker.get_cash() / signal.price)
            shares = min(shares, max_affordable_shares)

            logger.debug(
                f"等权重仓位计算: 组合价值={portfolio_value:.2f}, "
                f"目标仓位价值={target_position_value:.2f}, 股数={shares}"
            )

            return max(shares, self.params.min_trade_unit)

        except Exception as e:
            logger.error(f"计算等权重仓位失败: {e}")
            return self.params.min_trade_unit

    def check_stop_loss_condition(self) -> bool:
        """等权重策略不实现止损逻辑"""
        return False

    def should_rebalance(self) -> bool:
        """检查是否需要再平衡

        Returns:
            bool: 是否需要再平衡
        """
        try:
            current_value = self.broker.get_value()

            # 首次运行或价值变化超过阈值时再平衡
            if self.last_rebalance_value == 0:
                return True

            value_change = (
                abs(current_value - self.last_rebalance_value)
                / self.last_rebalance_value
            )
            return value_change >= self.params.rebalance_threshold

        except Exception:
            return False

    def rebalance_positions(self):
        """再平衡所有仓位"""
        try:
            current_positions = {}

            # 获取当前所有持仓
            for data in self.datas:
                position = self.getposition(data)
                if position.size > 0:
                    current_positions[data] = position.size

            # 计算目标仓位
            portfolio_value = self.broker.get_value()
            target_position_value = portfolio_value * self.params.weight_per_position

            # 调整每个仓位
            for data, current_size in current_positions.items():
                current_price = data.close[0]
                target_size = int(target_position_value / current_price)
                target_size = (
                    target_size // self.params.min_trade_unit
                ) * self.params.min_trade_unit

                size_diff = target_size - current_size

                if abs(size_diff) >= self.params.min_trade_unit:
                    if size_diff > 0:
                        # 增加仓位
                        self.buy(data=data, size=size_diff)
                        logger.info(f"再平衡增加仓位: {data._name}, 数量: {size_diff}")
                    else:
                        # 减少仓位
                        self.sell(data=data, size=abs(size_diff))
                        logger.info(
                            f"再平衡减少仓位: {data._name}, 数量: {abs(size_diff)}"
                        )

            self.last_rebalance_value = portfolio_value

        except Exception as e:
            logger.error(f"再平衡失败: {e}")

    def generate_signal(self) -> TradingSignal | None:
        """等权重策略主要通过再平衡管理仓位"""
        # 检查是否需要再平衡
        if self.should_rebalance():
            self.rebalance_positions()

        return None

    def next(self):
        """策略主逻辑"""
        super().next()


class FixedStopLossStrategy(BaseStrategy, RiskManagementMixin):
    """固定止损策略

    当价格回撤达到设定百分比时触发止损。
    """

    # 策略参数
    params = (
        ("stop_loss_pct", 0.08),  # 止损百分比
        ("position_size", 1000),  # 默认仓位大小
        ("trailing_stop", False),  # 是否启用移动止损
    )

    def __init__(self):
        """初始化策略"""
        super().__init__()
        self.entry_prices = {}  # 记录入场价格
        self.highest_prices = {}  # 记录最高价格（用于移动止损）

        logger.info(
            f"固定止损策略初始化完成, 止损百分比: {self.params.stop_loss_pct:.1%}, "
            f"移动止损: {self.params.trailing_stop}"
        )

    def get_strategy_name(self) -> str:
        """获取策略名称"""
        return "FixedStopLossStrategy"

    def get_strategy_description(self) -> str:
        """获取策略描述"""
        return "固定止损策略, 当价格回撤达到设定百分比时触发止损"

    def get_required_params(self) -> list[str]:
        """获取策略必需参数列表"""
        return ["stop_loss_pct"]

    def validate_params(self) -> bool:
        """验证策略参数是否有效"""
        try:
            # 验证止损百分比
            if not (0 < self.params.stop_loss_pct <= 1):
                logger.error(f"止损百分比必须在0-1范围内: {self.params.stop_loss_pct}")
                return False

            # 验证仓位大小
            if self.params.position_size <= 0:
                logger.error(f"仓位大小必须大于0: {self.params.position_size}")
                return False

            return True

        except Exception as e:
            logger.error(f"参数验证失败: {e}")
            return False

    def calculate_position_size(self, signal: TradingSignal) -> int:
        """计算仓位大小

        Args:
            signal: 交易信号

        Returns:
            int: 仓位大小
        """
        return signal.size if signal.size else self.params.position_size

    def check_stop_loss_condition(self) -> bool:
        """检查止损条件

        Returns:
            bool: 是否触发止损
        """
        try:
            if not self.position:
                return False

            current_price = self.data.close[0]
            data_name = self.data._name

            # 获取入场价格
            if data_name not in self.entry_prices:
                return False

            entry_price = self.entry_prices[data_name]

            if self.params.trailing_stop:
                # 移动止损逻辑
                if data_name not in self.highest_prices:
                    self.highest_prices[data_name] = current_price
                else:
                    self.highest_prices[data_name] = max(
                        self.highest_prices[data_name], current_price
                    )

                # 计算从最高点的回撤
                drawdown = (
                    self.highest_prices[data_name] - current_price
                ) / self.highest_prices[data_name]

                if drawdown >= self.params.stop_loss_pct:
                    logger.info(
                        f"触发移动止损: {data_name}, 最高价: {self.highest_prices[data_name]:.2f}, "
                        f"当前价: {current_price:.2f}, 回撤: {drawdown:.2%}"
                    )
                    return True
            else:
                # 固定止损逻辑
                loss = (entry_price - current_price) / entry_price

                if loss >= self.params.stop_loss_pct:
                    logger.info(
                        f"触发固定止损: {data_name}, 入场价: {entry_price:.2f}, "
                        f"当前价: {current_price:.2f}, 亏损: {loss:.2%}"
                    )
                    return True

            return False

        except Exception as e:
            logger.error(f"检查止损条件失败: {e}")
            return False

    def record_entry_price(self, price: float):
        """记录入场价格

        Args:
            price: 入场价格
        """
        data_name = self.data._name
        self.entry_prices[data_name] = price
        if self.params.trailing_stop:
            self.highest_prices[data_name] = price

        logger.debug(f"记录入场价格: {data_name}, 价格: {price:.2f}")

    def clear_position_records(self):
        """清除仓位记录"""
        data_name = self.data._name
        if data_name in self.entry_prices:
            del self.entry_prices[data_name]
        if data_name in self.highest_prices:
            del self.highest_prices[data_name]

        logger.debug(f"清除仓位记录: {data_name}")

    def generate_signal(self) -> TradingSignal | None:
        """生成交易信号（主要是止损信号）"""
        # 检查止损条件
        if self.check_stop_loss_condition():
            return TradingSignal(
                signal_type=SignalType.SELL,
                price=self.data.close[0],
                reason="触发止损",
                confidence=1.0,
            )

        return None

    def notify_order(self, order):
        """订单状态通知"""
        super().notify_order(order)

        # 记录买入时的入场价格
        if order.status in [order.Completed]:
            if order.isbuy():
                self.record_entry_price(order.executed.price)
            elif order.issell():
                self.clear_position_records()


class ATRStopLossStrategy(BaseStrategy, RiskManagementMixin):
    """动态止损（ATR）策略

    基于平均真实波幅（ATR）计算动态止损价格。
    """

    # 策略参数
    params = (
        ("atr_period", 14),  # ATR计算周期
        ("atr_multiplier", 2.0),  # ATR倍数
        ("position_size", 1000),  # 默认仓位大小
        ("min_atr_value", 0.01),  # 最小ATR值
    )

    def __init__(self):
        """初始化策略"""
        super().__init__()
        self.atr = bt.indicators.ATR(self.data, period=self.params.atr_period)
        self.entry_prices = {}  # 记录入场价格
        self.stop_prices = {}  # 记录止损价格

        logger.info(
            f"ATR止损策略初始化完成, ATR周期: {self.params.atr_period}, "
            f"ATR倍数: {self.params.atr_multiplier}"
        )

    def get_strategy_name(self) -> str:
        """获取策略名称"""
        return "ATRStopLossStrategy"

    def get_strategy_description(self) -> str:
        """获取策略描述"""
        return "基于ATR的动态止损策略, 根据市场波动性调整止损距离"

    def get_required_params(self) -> list[str]:
        """获取策略必需参数列表"""
        return ["atr_period", "atr_multiplier"]

    def validate_params(self) -> bool:
        """验证策略参数是否有效"""
        try:
            # 验证ATR周期
            if self.params.atr_period <= 0:
                logger.error(f"ATR周期必须大于0: {self.params.atr_period}")
                return False

            # 验证ATR倍数
            if self.params.atr_multiplier <= 0:
                logger.error(f"ATR倍数必须大于0: {self.params.atr_multiplier}")
                return False

            # 验证仓位大小
            if self.params.position_size <= 0:
                logger.error(f"仓位大小必须大于0: {self.params.position_size}")
                return False

            return True

        except Exception as e:
            logger.error(f"参数验证失败: {e}")
            return False

    def calculate_position_size(self, signal: TradingSignal) -> int:
        """计算仓位大小

        Args:
            signal: 交易信号

        Returns:
            int: 仓位大小
        """
        return signal.size if signal.size else self.params.position_size

    def calculate_atr_stop_price(self, entry_price: float) -> float:
        """计算ATR止损价格

        Args:
            entry_price: 入场价格

        Returns:
            float: 止损价格
        """
        try:
            # 获取当前ATR值
            current_atr = self.atr[0]

            # 确保ATR值不为零
            if current_atr < self.params.min_atr_value:
                current_atr = self.params.min_atr_value

            # 计算止损距离
            stop_distance = current_atr * self.params.atr_multiplier

            # 计算止损价格（做多时在入场价下方）
            stop_price = entry_price - stop_distance

            logger.debug(
                f"ATR止损计算: 入场价={entry_price:.2f}, ATR={current_atr:.4f}, "
                f"止损距离={stop_distance:.4f}, 止损价={stop_price:.2f}"
            )

            return stop_price

        except Exception as e:
            logger.error(f"计算ATR止损价格失败: {e}")
            return entry_price * (1 - 0.05)  # 默认5%止损

    def update_atr_stop_price(self):
        """更新ATR止损价格"""
        try:
            data_name = self.data._name

            if data_name not in self.entry_prices:
                return

            entry_price = self.entry_prices[data_name]
            new_stop_price = self.calculate_atr_stop_price(entry_price)

            # 只有当新止损价更高时才更新（移动止损）
            if (
                data_name not in self.stop_prices
                or new_stop_price > self.stop_prices[data_name]
            ):
                self.stop_prices[data_name] = new_stop_price
                logger.debug(
                    f"更新ATR止损价: {data_name}, 新止损价: {new_stop_price:.2f}"
                )

        except Exception as e:
            logger.error(f"更新ATR止损价格失败: {e}")

    def check_stop_loss_condition(self) -> bool:
        """检查ATR止损条件

        Returns:
            bool: 是否触发止损
        """
        try:
            if not self.position:
                return False

            current_price = self.data.close[0]
            data_name = self.data._name

            # 更新止损价格
            self.update_atr_stop_price()

            # 检查是否触发止损
            if data_name in self.stop_prices:
                stop_price = self.stop_prices[data_name]

                if current_price <= stop_price:
                    logger.info(
                        f"触发ATR止损: {data_name}, 当前价: {current_price:.2f}, "
                        f"止损价: {stop_price:.2f}"
                    )
                    return True

            return False

        except Exception as e:
            logger.error(f"检查ATR止损条件失败: {e}")
            return False

    def record_entry_price(self, price: float):
        """记录入场价格并计算初始止损价

        Args:
            price: 入场价格
        """
        data_name = self.data._name
        self.entry_prices[data_name] = price

        # 计算初始止损价格
        initial_stop_price = self.calculate_atr_stop_price(price)
        self.stop_prices[data_name] = initial_stop_price

        logger.info(
            f"记录ATR入场价格: {data_name}, 入场价: {price:.2f}, "
            f"初始止损价: {initial_stop_price:.2f}"
        )

    def clear_position_records(self):
        """清除仓位记录"""
        data_name = self.data._name
        if data_name in self.entry_prices:
            del self.entry_prices[data_name]
        if data_name in self.stop_prices:
            del self.stop_prices[data_name]

        logger.debug(f"清除ATR仓位记录: {data_name}")

    def generate_signal(self) -> TradingSignal | None:
        """生成交易信号（主要是止损信号）"""
        # 检查ATR止损条件
        if self.check_stop_loss_condition():
            return TradingSignal(
                signal_type=SignalType.SELL,
                price=self.data.close[0],
                reason="触发ATR止损",
                confidence=1.0,
            )

        return None

    def notify_order(self, order):
        """订单状态通知"""
        super().notify_order(order)

        # 记录买入时的入场价格
        if order.status in [order.Completed]:
            if order.isbuy():
                self.record_entry_price(order.executed.price)
            elif order.issell():
                self.clear_position_records()

    def next(self):
        """策略主逻辑"""
        super().next()

        # 确保ATR指标有足够的数据
        if len(self.data) < self.params.atr_period:
            return
