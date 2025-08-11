"""策略基类模块

提供所有交易策略的基础框架和通用接口。
"""

from abc import ABC, abstractmethod
from dataclasses import dataclass
from enum import Enum
from typing import Any

import backtrader as bt
from loguru import logger


class SignalType(Enum):
    """交易信号类型"""

    BUY = "buy"
    SELL = "sell"
    HOLD = "hold"


@dataclass
class TradingSignal:
    """交易信号数据类"""

    signal_type: SignalType
    price: float
    size: int | None = None
    reason: str | None = None
    confidence: float | None = None


class BaseStrategy(bt.Strategy, ABC):
    """策略基类

    所有交易策略都应该继承此基类，并实现必要的抽象方法。
    提供统一的参数管理、信号生成、风险控制等功能。
    """

    # 默认参数，子类可以覆盖
    params = (
        ("stop_loss_pct", 0.05),  # 止损百分比
        ("take_profit_pct", 0.10),  # 止盈百分比
        ("position_size", 1000),  # 默认仓位大小
        ("max_positions", 5),  # 最大持仓数量
    )

    def __init__(self):
        """初始化策略"""
        super().__init__()
        self.order = None
        self.signals: list[TradingSignal] = []
        self.entry_price = None
        self.strategy_name = self.__class__.__name__

        # 记录策略参数
        self.log_params()

    def log_params(self):
        """记录策略参数"""
        params_dict = {
            param[0]: getattr(self.params, param[0])
            for param in self.params._getpairs()
        }

        logger.info(f"策略 {self.strategy_name} 初始化参数: {params_dict}")

    def log(self, txt: str, dt=None):
        """统一日志记录"""
        dt = dt or self.datas[0].datetime.date(0)
        logger.info(f"{dt.isoformat()} [{self.strategy_name}] {txt}")

    @abstractmethod
    def get_strategy_name(self) -> str:
        """获取策略名称"""
        pass

    @abstractmethod
    def get_strategy_description(self) -> str:
        """获取策略描述"""
        pass

    @abstractmethod
    def get_required_params(self) -> list[str]:
        """获取策略必需参数列表"""
        pass

    @abstractmethod
    def validate_params(self) -> bool:
        """验证策略参数是否有效"""
        pass

    @abstractmethod
    def generate_signal(self) -> TradingSignal | None:
        """生成交易信号

        Returns:
            TradingSignal: 交易信号，如果没有信号则返回None
        """
        pass

    def next(self):
        """策略主逻辑

        每个数据点都会调用此方法，子类可以覆盖以实现自定义逻辑。
        """
        # 如果有未完成的订单，跳过
        if self.order:
            return

        # 生成交易信号
        signal = self.generate_signal()
        if signal:
            self.execute_signal(signal)

    def execute_signal(self, signal: TradingSignal):
        """执行交易信号"""
        if signal.signal_type == SignalType.BUY:
            self.execute_buy(signal)
        elif signal.signal_type == SignalType.SELL:
            self.execute_sell(signal)

        # 记录信号
        self.signals.append(signal)

    def execute_buy(self, signal: TradingSignal):
        """执行买入信号"""
        if not self.position:
            size = signal.size or self.calculate_position_size()
            self.order = self.buy(size=size)
            self.entry_price = self.data.close[0]
            self.log(
                f"买入信号: 价格={signal.price:.2f}, 数量={size}, 原因={signal.reason}"
            )

    def execute_sell(self, signal: TradingSignal):
        """执行卖出信号"""
        if self.position:
            self.order = self.sell()
            self.log(f"卖出信号: 价格={signal.price:.2f}, 原因={signal.reason}")

    def calculate_position_size(self) -> int:
        """计算仓位大小

        子类可以覆盖此方法实现自定义仓位管理。
        """
        return self.params.position_size

    def notify_order(self, order):
        """订单状态通知"""
        if order.status in [order.Submitted, order.Accepted]:
            return

        if order.status in [order.Completed]:
            if order.isbuy():
                self.log(
                    f"买入执行: 价格={order.executed.price:.2f}, 数量={order.executed.size}"
                )
            else:
                self.log(
                    f"卖出执行: 价格={order.executed.price:.2f}, 数量={order.executed.size}"
                )
        elif order.status in [order.Canceled, order.Margin, order.Rejected]:
            self.log(f"订单失败: {order.status}")

        self.order = None

    def notify_trade(self, trade):
        """交易完成通知"""
        if not trade.isclosed:
            return

        self.log(f"交易完成: 盈亏={trade.pnl:.2f}, 净盈亏={trade.pnlcomm:.2f}")

    def check_stop_loss(self) -> bool:
        """检查止损条件"""
        if not self.position or not self.entry_price:
            return False

        current_price = self.data.close[0]
        if self.position.size > 0:  # 多头持仓
            loss_pct = (self.entry_price - current_price) / self.entry_price
            if loss_pct >= self.params.stop_loss_pct:
                self.log(
                    f"触发止损: 当前价格={current_price:.2f}, 入场价格={self.entry_price:.2f}, 亏损={loss_pct:.2%}"
                )
                return True

        return False

    def check_take_profit(self) -> bool:
        """检查止盈条件"""
        if not self.position or not self.entry_price:
            return False

        current_price = self.data.close[0]
        if self.position.size > 0:  # 多头持仓
            profit_pct = (current_price - self.entry_price) / self.entry_price
            if profit_pct >= self.params.take_profit_pct:
                self.log(
                    f"触发止盈: 当前价格={current_price:.2f}, 入场价格={self.entry_price:.2f}, 盈利={profit_pct:.2%}"
                )
                return True

        return False

    def get_strategy_stats(self) -> dict[str, Any]:
        """获取策略统计信息"""
        return {
            "strategy_name": self.strategy_name,
            "total_signals": len(self.signals),
            "buy_signals": len(
                [s for s in self.signals if s.signal_type == SignalType.BUY]
            ),
            "sell_signals": len(
                [s for s in self.signals if s.signal_type == SignalType.SELL]
            ),
            "current_position": self.position.size if self.position else 0,
            "entry_price": self.entry_price,
        }
