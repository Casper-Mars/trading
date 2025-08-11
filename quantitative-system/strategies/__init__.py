"""交易策略模块

本模块包含各种交易策略的实现，包括技术分析策略、基本面策略等。
所有策略都继承自BaseStrategy基类，提供统一的接口和功能。"""

from .base_strategy import BaseStrategy, SignalType, TradingSignal
from .bollinger_strategy import BollingerBandsStrategy
from .ma_strategy import MovingAverageStrategy
from .macd_strategy import MACDStrategy
from .rsi_strategy import RSIStrategy
from .strategy_registry import (
    StrategyInfo,
    StrategyRegistry,
    create_strategy,
    get_strategy,
    register_strategy,
    strategy_registry,
)
from .triple_ma_strategy import TripleMovingAverageStrategy

__all__ = [
    "BaseStrategy",
    "BollingerBandsStrategy",
    "MACDStrategy",
    "MovingAverageStrategy",
    "RSIStrategy",
    "SignalType",
    "StrategyInfo",
    "StrategyRegistry",
    "TradingSignal",
    "TripleMovingAverageStrategy",
    "create_strategy",
    "get_strategy",
    "register_strategy",
    "strategy_registry",
]
