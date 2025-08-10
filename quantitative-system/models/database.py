"""数据库模型定义"""

from sqlmodel import SQLModel, Field, Column, Text, JSON
from sqlalchemy import Index, UniqueConstraint
from datetime import datetime, date
from decimal import Decimal
from typing import Optional, Dict, Any, List
from .enums import (
    PositionStatus, PositionType, OrderType, OrderStatus,
    PlanStatus, PlanType, BacktestStatus, StrategyType,
    RiskLevel, MarketType, TimeFrame, DataSource,
    TaskStatus, TaskType, LogLevel
)


class Position(SQLModel, table=True):
    """持仓表"""
    __tablename__ = "positions"
    
    id: Optional[int] = Field(default=None, primary_key=True)
    symbol: str = Field(max_length=20, description="股票代码")
    name: str = Field(max_length=100, description="股票名称")
    position_type: PositionType = Field(description="持仓类型：1-多头，2-空头")
    quantity: int = Field(description="持仓数量")
    avg_cost: Decimal = Field(max_digits=10, decimal_places=4, description="平均成本")
    current_price: Optional[Decimal] = Field(default=None, max_digits=10, decimal_places=4, description="当前价格")
    market_value: Optional[Decimal] = Field(default=None, max_digits=15, decimal_places=2, description="市值")
    unrealized_pnl: Optional[Decimal] = Field(default=None, max_digits=15, decimal_places=2, description="浮动盈亏")
    realized_pnl: Decimal = Field(default=0, max_digits=15, decimal_places=2, description="已实现盈亏")
    status: PositionStatus = Field(default=PositionStatus.ACTIVE, description="状态：1-活跃，2-已平仓，3-暂停")
    open_date: date = Field(description="开仓日期")
    close_date: Optional[date] = Field(default=None, description="平仓日期")
    notes: Optional[str] = Field(default=None, max_length=500, description="备注")
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    updated_at: datetime = Field(default_factory=datetime.now, description="更新时间")
    
    __table_args__ = (
        Index("idx_positions_symbol", "symbol"),
        Index("idx_positions_status", "status"),
        Index("idx_positions_open_date", "open_date"),
    )


class BacktestResult(SQLModel, table=True):
    """回测结果表"""
    __tablename__ = "backtest_results"
    
    id: Optional[int] = Field(default=None, primary_key=True)
    name: str = Field(max_length=200, description="回测名称")
    strategy_type: StrategyType = Field(description="策略类型")
    strategy_params: Dict[str, Any] = Field(sa_column=Column(JSON), description="策略参数")
    symbols: List[str] = Field(sa_column=Column(JSON), description="标的列表")
    start_date: date = Field(description="开始日期")
    end_date: date = Field(description="结束日期")
    initial_cash: Decimal = Field(max_digits=15, decimal_places=2, description="初始资金")
    final_value: Optional[Decimal] = Field(default=None, max_digits=15, decimal_places=2, description="最终价值")
    total_return: Optional[Decimal] = Field(default=None, max_digits=8, decimal_places=4, description="总收益率")
    annual_return: Optional[Decimal] = Field(default=None, max_digits=8, decimal_places=4, description="年化收益率")
    sharpe_ratio: Optional[Decimal] = Field(default=None, max_digits=8, decimal_places=4, description="夏普比率")
    max_drawdown: Optional[Decimal] = Field(default=None, max_digits=8, decimal_places=4, description="最大回撤")
    win_rate: Optional[Decimal] = Field(default=None, max_digits=5, decimal_places=4, description="胜率")
    total_trades: Optional[int] = Field(default=None, description="总交易次数")
    avg_trade_return: Optional[Decimal] = Field(default=None, max_digits=8, decimal_places=4, description="平均交易收益")
    volatility: Optional[Decimal] = Field(default=None, max_digits=8, decimal_places=4, description="波动率")
    calmar_ratio: Optional[Decimal] = Field(default=None, max_digits=8, decimal_places=4, description="卡玛比率")
    status: BacktestStatus = Field(default=BacktestStatus.PENDING, description="状态")
    raw_data: Optional[Dict[str, Any]] = Field(default=None, sa_column=Column(JSON), description="原始数据")
    error_message: Optional[str] = Field(default=None, sa_column=Column(Text), description="错误信息")
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    updated_at: datetime = Field(default_factory=datetime.now, description="更新时间")
    
    __table_args__ = (
        Index("idx_backtest_strategy_type", "strategy_type"),
        Index("idx_backtest_status", "status"),
        Index("idx_backtest_created_at", "created_at"),
    )


class TradingPlan(SQLModel, table=True):
    """交易方案表"""
    __tablename__ = "trading_plans"
    
    id: Optional[int] = Field(default=None, primary_key=True)
    plan_date: date = Field(description="方案日期")
    title: str = Field(max_length=200, description="方案标题")
    content: str = Field(sa_column=Column(Text), description="方案内容（Markdown格式）")
    plan_type: PlanType = Field(description="方案类型：1-手动，2-自动，3-混合")
    risk_level: RiskLevel = Field(description="风险等级：1-低，2-中，3-高，4-极高")
    target_return: Optional[Decimal] = Field(default=None, max_digits=8, decimal_places=4, description="目标收益率")
    max_drawdown_limit: Optional[Decimal] = Field(default=None, max_digits=8, decimal_places=4, description="最大回撤限制")
    position_limit: Optional[Decimal] = Field(default=None, max_digits=8, decimal_places=4, description="仓位限制")
    recommendations: Optional[Dict[str, Any]] = Field(default=None, sa_column=Column(JSON), description="推荐操作")
    backtest_results: Optional[Dict[str, Any]] = Field(default=None, sa_column=Column(JSON), description="回测结果")
    ai_analysis: Optional[Dict[str, Any]] = Field(default=None, sa_column=Column(JSON), description="AI分析结果")
    status: PlanStatus = Field(default=PlanStatus.DRAFT, description="状态")
    execution_rate: Optional[Decimal] = Field(default=None, max_digits=5, decimal_places=4, description="执行率")
    actual_return: Optional[Decimal] = Field(default=None, max_digits=8, decimal_places=4, description="实际收益率")
    notes: Optional[str] = Field(default=None, max_length=1000, description="备注")
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    updated_at: datetime = Field(default_factory=datetime.now, description="更新时间")
    
    __table_args__ = (
        Index("idx_trading_plans_date", "plan_date"),
        Index("idx_trading_plans_status", "status"),
        Index("idx_trading_plans_type", "plan_type"),
        UniqueConstraint("plan_date", "title", name="uq_plan_date_title"),
    )


class MarketDataCache(SQLModel, table=True):
    """市场数据缓存表"""
    __tablename__ = "market_data_cache"
    
    id: Optional[int] = Field(default=None, primary_key=True)
    cache_key: str = Field(max_length=200, description="缓存键")
    symbol: Optional[str] = Field(default=None, max_length=20, description="股票代码")
    data_type: str = Field(max_length=50, description="数据类型")
    time_frame: Optional[TimeFrame] = Field(default=None, description="时间周期")
    start_date: Optional[date] = Field(default=None, description="开始日期")
    end_date: Optional[date] = Field(default=None, description="结束日期")
    data_source: DataSource = Field(description="数据源")
    data_content: Dict[str, Any] = Field(sa_column=Column(JSON), description="数据内容")
    data_size: int = Field(description="数据大小（字节）")
    hit_count: int = Field(default=0, description="命中次数")
    last_hit_at: Optional[datetime] = Field(default=None, description="最后命中时间")
    expires_at: Optional[datetime] = Field(default=None, description="过期时间")
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    updated_at: datetime = Field(default_factory=datetime.now, description="更新时间")
    
    __table_args__ = (
        Index("idx_cache_key", "cache_key"),
        Index("idx_cache_symbol", "symbol"),
        Index("idx_cache_data_type", "data_type"),
        Index("idx_cache_expires_at", "expires_at"),
        UniqueConstraint("cache_key", name="uq_cache_key"),
    )


class SystemLog(SQLModel, table=True):
    """系统日志表"""
    __tablename__ = "system_logs"
    
    id: Optional[int] = Field(default=None, primary_key=True)
    level: LogLevel = Field(description="日志级别")
    module: str = Field(max_length=100, description="模块名称")
    function: Optional[str] = Field(default=None, max_length=100, description="函数名称")
    message: str = Field(sa_column=Column(Text), description="日志消息")
    details: Optional[Dict[str, Any]] = Field(default=None, sa_column=Column(JSON), description="详细信息")
    user_id: Optional[str] = Field(default=None, max_length=50, description="用户ID")
    session_id: Optional[str] = Field(default=None, max_length=100, description="会话ID")
    request_id: Optional[str] = Field(default=None, max_length=100, description="请求ID")
    ip_address: Optional[str] = Field(default=None, max_length=45, description="IP地址")
    user_agent: Optional[str] = Field(default=None, max_length=500, description="用户代理")
    execution_time: Optional[Decimal] = Field(default=None, max_digits=10, decimal_places=6, description="执行时间（秒）")
    stack_trace: Optional[str] = Field(default=None, sa_column=Column(Text), description="堆栈跟踪")
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    
    __table_args__ = (
        Index("idx_system_logs_level", "level"),
        Index("idx_system_logs_module", "module"),
        Index("idx_system_logs_created_at", "created_at"),
        Index("idx_system_logs_user_id", "user_id"),
    )


class Task(SQLModel, table=True):
    """任务表"""
    __tablename__ = "tasks"
    
    id: Optional[int] = Field(default=None, primary_key=True)
    name: str = Field(max_length=200, description="任务名称")
    task_type: TaskType = Field(description="任务类型")
    status: TaskStatus = Field(default=TaskStatus.PENDING, description="任务状态")
    priority: int = Field(default=5, description="优先级（1-10，数字越小优先级越高）")
    params: Optional[Dict[str, Any]] = Field(default=None, sa_column=Column(JSON), description="任务参数")
    result: Optional[Dict[str, Any]] = Field(default=None, sa_column=Column(JSON), description="任务结果")
    error_message: Optional[str] = Field(default=None, sa_column=Column(Text), description="错误信息")
    retry_count: int = Field(default=0, description="重试次数")
    max_retries: int = Field(default=3, description="最大重试次数")
    scheduled_at: Optional[datetime] = Field(default=None, description="计划执行时间")
    started_at: Optional[datetime] = Field(default=None, description="开始执行时间")
    completed_at: Optional[datetime] = Field(default=None, description="完成时间")
    execution_time: Optional[Decimal] = Field(default=None, max_digits=10, decimal_places=6, description="执行时间（秒）")
    created_by: Optional[str] = Field(default=None, max_length=50, description="创建者")
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    updated_at: datetime = Field(default_factory=datetime.now, description="更新时间")
    
    __table_args__ = (
        Index("idx_tasks_type", "task_type"),
        Index("idx_tasks_status", "status"),
        Index("idx_tasks_priority", "priority"),
        Index("idx_tasks_scheduled_at", "scheduled_at"),
        Index("idx_tasks_created_at", "created_at"),
    )