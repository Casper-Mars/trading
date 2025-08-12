"""数据访问层模块"""

from .backtest_repo import BacktestRepo, backtest_repo
from .cache_repo import CacheRepo, cache_repo
from .plan_repo import PlanRepo, plan_repo
from .position_repo import PositionRepo, position_repo
from .task_repo import TaskRepository

# 创建TaskRepository实例
task_repo = TaskRepository()

__all__ = [
    "BacktestRepo",
    "CacheRepo",
    "PlanRepo",
    "PositionRepo",
    "TaskRepository",
    "backtest_repo",
    "cache_repo",
    "plan_repo",
    "position_repo",
    "task_repo",
]
