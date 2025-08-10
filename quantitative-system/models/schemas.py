"""Pydantic Schema模型定义"""

from datetime import date, datetime
from decimal import Decimal
from typing import Any

from pydantic import BaseModel, ConfigDict, Field

from .enums import (
    BacktestStatus,
    PlanStatus,
    PlanType,
    PositionStatus,
    PositionType,
    RiskLevel,
    StrategyType,
    TaskStatus,
    TaskType,
)

# ============= 基础Schema =============


class BaseSchema(BaseModel):
    """基础Schema类"""

    model_config = ConfigDict(
        from_attributes=True, use_enum_values=True, arbitrary_types_allowed=True
    )


# ============= 持仓相关Schema =============


class PositionBase(BaseSchema):
    """持仓基础Schema"""

    symbol: str = Field(..., max_length=20, description="股票代码")
    name: str = Field(..., max_length=100, description="股票名称")
    position_type: PositionType = Field(..., description="持仓类型")
    quantity: int = Field(..., gt=0, description="持仓数量")
    avg_cost: Decimal = Field(..., gt=0, description="平均成本")
    notes: str | None = Field(None, max_length=500, description="备注")


class PositionCreate(PositionBase):
    """创建持仓Schema"""

    open_date: date = Field(..., description="开仓日期")


class PositionUpdate(BaseSchema):
    """更新持仓Schema"""

    name: str | None = Field(None, max_length=100, description="股票名称")
    quantity: int | None = Field(None, gt=0, description="持仓数量")
    avg_cost: Decimal | None = Field(None, gt=0, description="平均成本")
    current_price: Decimal | None = Field(None, gt=0, description="当前价格")
    status: PositionStatus | None = Field(None, description="状态")
    close_date: date | None = Field(None, description="平仓日期")
    notes: str | None = Field(None, max_length=500, description="备注")


class PositionResponse(PositionBase):
    """持仓响应Schema"""

    id: int
    current_price: Decimal | None = Field(None, description="当前价格")
    market_value: Decimal | None = Field(None, description="市值")
    unrealized_pnl: Decimal | None = Field(None, description="浮动盈亏")
    realized_pnl: Decimal = Field(..., description="已实现盈亏")
    status: PositionStatus = Field(..., description="状态")
    open_date: date = Field(..., description="开仓日期")
    close_date: date | None = Field(None, description="平仓日期")
    created_at: datetime = Field(..., description="创建时间")
    updated_at: datetime = Field(..., description="更新时间")


class PositionSummary(BaseSchema):
    """持仓汇总Schema"""

    total_positions: int = Field(..., description="总持仓数")
    total_market_value: Decimal = Field(..., description="总市值")
    total_cost: Decimal = Field(..., description="总成本")
    total_unrealized_pnl: Decimal = Field(..., description="总浮动盈亏")
    total_realized_pnl: Decimal = Field(..., description="总已实现盈亏")
    total_return_rate: Decimal = Field(..., description="总收益率")
    active_positions: int = Field(..., description="活跃持仓数")
    closed_positions: int = Field(..., description="已平仓数")


# ============= 回测相关Schema =============


class BacktestBase(BaseSchema):
    """回测基础Schema"""

    name: str = Field(..., max_length=200, description="回测名称")
    strategy_type: StrategyType = Field(..., description="策略类型")
    strategy_params: dict[str, Any] = Field(..., description="策略参数")
    symbols: list[str] = Field(..., description="标的列表")
    start_date: date = Field(..., description="开始日期")
    end_date: date = Field(..., description="结束日期")
    initial_cash: Decimal = Field(..., gt=0, description="初始资金")


class BacktestCreate(BacktestBase):
    """创建回测Schema"""

    pass


class BacktestUpdate(BaseSchema):
    """更新回测Schema"""

    name: str | None = Field(None, max_length=200, description="回测名称")
    status: BacktestStatus | None = Field(None, description="状态")
    final_value: Decimal | None = Field(None, description="最终价值")
    total_return: Decimal | None = Field(None, description="总收益率")
    annual_return: Decimal | None = Field(None, description="年化收益率")
    sharpe_ratio: Decimal | None = Field(None, description="夏普比率")
    max_drawdown: Decimal | None = Field(None, description="最大回撤")
    win_rate: Decimal | None = Field(None, description="胜率")
    total_trades: int | None = Field(None, description="总交易次数")
    avg_trade_return: Decimal | None = Field(None, description="平均交易收益")
    volatility: Decimal | None = Field(None, description="波动率")
    calmar_ratio: Decimal | None = Field(None, description="卡玛比率")
    raw_data: dict[str, Any] | None = Field(None, description="原始数据")
    error_message: str | None = Field(None, description="错误信息")


class BacktestResponse(BacktestBase):
    """回测响应Schema"""

    id: int
    final_value: Decimal | None = Field(None, description="最终价值")
    total_return: Decimal | None = Field(None, description="总收益率")
    annual_return: Decimal | None = Field(None, description="年化收益率")
    sharpe_ratio: Decimal | None = Field(None, description="夏普比率")
    max_drawdown: Decimal | None = Field(None, description="最大回撤")
    win_rate: Decimal | None = Field(None, description="胜率")
    total_trades: int | None = Field(None, description="总交易次数")
    avg_trade_return: Decimal | None = Field(None, description="平均交易收益")
    volatility: Decimal | None = Field(None, description="波动率")
    calmar_ratio: Decimal | None = Field(None, description="卡玛比率")
    status: BacktestStatus = Field(..., description="状态")
    error_message: str | None = Field(None, description="错误信息")
    created_at: datetime = Field(..., description="创建时间")
    updated_at: datetime = Field(..., description="更新时间")


class BacktestMetrics(BaseSchema):
    """回测指标Schema"""

    total_return: Decimal = Field(..., description="总收益率")
    annual_return: Decimal = Field(..., description="年化收益率")
    sharpe_ratio: Decimal = Field(..., description="夏普比率")
    max_drawdown: Decimal = Field(..., description="最大回撤")
    win_rate: Decimal = Field(..., description="胜率")
    total_trades: int = Field(..., description="总交易次数")
    avg_trade_return: Decimal = Field(..., description="平均交易收益")
    volatility: Decimal = Field(..., description="波动率")
    calmar_ratio: Decimal = Field(..., description="卡玛比率")


# ============= 交易方案相关Schema =============


class TradingPlanBase(BaseSchema):
    """交易方案基础Schema"""

    plan_date: date = Field(..., description="方案日期")
    title: str = Field(..., max_length=200, description="方案标题")
    content: str = Field(..., description="方案内容")
    plan_type: PlanType = Field(..., description="方案类型")
    risk_level: RiskLevel = Field(..., description="风险等级")
    target_return: Decimal | None = Field(None, description="目标收益率")
    max_drawdown_limit: Decimal | None = Field(None, description="最大回撤限制")
    position_limit: Decimal | None = Field(None, description="仓位限制")
    notes: str | None = Field(None, max_length=1000, description="备注")


class TradingPlanCreate(TradingPlanBase):
    """创建交易方案Schema"""

    recommendations: dict[str, Any] | None = Field(None, description="推荐操作")
    backtest_results: dict[str, Any] | None = Field(None, description="回测结果")
    ai_analysis: dict[str, Any] | None = Field(None, description="AI分析结果")


class TradingPlanUpdate(BaseSchema):
    """更新交易方案Schema"""

    title: str | None = Field(None, max_length=200, description="方案标题")
    content: str | None = Field(None, description="方案内容")
    status: PlanStatus | None = Field(None, description="状态")
    execution_rate: Decimal | None = Field(None, description="执行率")
    actual_return: Decimal | None = Field(None, description="实际收益率")
    notes: str | None = Field(None, max_length=1000, description="备注")


class TradingPlanResponse(TradingPlanBase):
    """交易方案响应Schema"""

    id: int
    recommendations: dict[str, Any] | None = Field(None, description="推荐操作")
    backtest_results: dict[str, Any] | None = Field(None, description="回测结果")
    ai_analysis: dict[str, Any] | None = Field(None, description="AI分析结果")
    status: PlanStatus = Field(..., description="状态")
    execution_rate: Decimal | None = Field(None, description="执行率")
    actual_return: Decimal | None = Field(None, description="实际收益率")
    created_at: datetime = Field(..., description="创建时间")
    updated_at: datetime = Field(..., description="更新时间")


class PlanGenerationRequest(BaseSchema):
    """方案生成请求Schema"""

    plan_date: date | None = Field(None, description="方案日期，默认为今天")
    risk_level: RiskLevel | None = Field(None, description="风险等级")
    target_return: Decimal | None = Field(None, description="目标收益率")
    position_limit: Decimal | None = Field(None, description="仓位限制")
    include_backtest: bool = Field(True, description="是否包含回测")
    include_ai_analysis: bool = Field(True, description="是否包含AI分析")


# ============= 任务相关Schema =============


class TaskBase(BaseSchema):
    """任务基础Schema"""

    name: str = Field(..., max_length=200, description="任务名称")
    task_type: TaskType = Field(..., description="任务类型")
    priority: int = Field(5, ge=1, le=10, description="优先级")
    params: dict[str, Any] | None = Field(None, description="任务参数")
    max_retries: int = Field(3, ge=0, description="最大重试次数")
    scheduled_at: datetime | None = Field(None, description="计划执行时间")


class TaskCreate(TaskBase):
    """创建任务Schema"""

    created_by: str | None = Field(None, max_length=50, description="创建者")


class TaskUpdate(BaseSchema):
    """更新任务Schema"""

    status: TaskStatus | None = Field(None, description="任务状态")
    result: dict[str, Any] | None = Field(None, description="任务结果")
    error_message: str | None = Field(None, description="错误信息")
    retry_count: int | None = Field(None, description="重试次数")
    started_at: datetime | None = Field(None, description="开始执行时间")
    completed_at: datetime | None = Field(None, description="完成时间")
    execution_time: Decimal | None = Field(None, description="执行时间")


class TaskResponse(TaskBase):
    """任务响应Schema"""

    id: int
    status: TaskStatus = Field(..., description="任务状态")
    result: dict[str, Any] | None = Field(None, description="任务结果")
    error_message: str | None = Field(None, description="错误信息")
    retry_count: int = Field(..., description="重试次数")
    started_at: datetime | None = Field(None, description="开始执行时间")
    completed_at: datetime | None = Field(None, description="完成时间")
    execution_time: Decimal | None = Field(None, description="执行时间")
    created_by: str | None = Field(None, description="创建者")
    created_at: datetime = Field(..., description="创建时间")
    updated_at: datetime = Field(..., description="更新时间")


# ============= 系统相关Schema =============


class SystemHealthResponse(BaseSchema):
    """系统健康状态响应Schema"""

    status: str = Field(..., description="系统状态")
    timestamp: datetime = Field(..., description="检查时间")
    database: bool = Field(..., description="数据库状态")
    redis: bool = Field(..., description="Redis状态")
    data_collection_service: bool = Field(..., description="数据采集服务状态")
    ai_service: bool = Field(..., description="AI服务状态")
    uptime: str = Field(..., description="运行时间")


class SystemStatsResponse(BaseSchema):
    """系统统计响应Schema"""

    total_positions: int = Field(..., description="总持仓数")
    active_positions: int = Field(..., description="活跃持仓数")
    total_backtests: int = Field(..., description="总回测数")
    completed_backtests: int = Field(..., description="已完成回测数")
    total_plans: int = Field(..., description="总方案数")
    active_plans: int = Field(..., description="活跃方案数")
    pending_tasks: int = Field(..., description="待执行任务数")
    running_tasks: int = Field(..., description="运行中任务数")
    cache_hit_rate: Decimal = Field(..., description="缓存命中率")
    avg_response_time: Decimal = Field(..., description="平均响应时间")


# ============= 通用响应Schema =============


class MessageResponse(BaseSchema):
    """消息响应Schema"""

    message: str = Field(..., description="消息内容")
    code: int = Field(200, description="状态码")
    timestamp: datetime = Field(default_factory=datetime.now, description="时间戳")


class ErrorResponse(BaseSchema):
    """错误响应Schema"""

    error: str = Field(..., description="错误信息")
    code: int = Field(..., description="错误码")
    detail: str | None = Field(None, description="详细信息")
    timestamp: datetime = Field(default_factory=datetime.now, description="时间戳")


class PaginatedResponse(BaseSchema):
    """分页响应Schema"""

    total: int = Field(..., description="总数量")
    page: int = Field(..., description="当前页")
    size: int = Field(..., description="每页大小")
    pages: int = Field(..., description="总页数")
    has_next: bool = Field(..., description="是否有下一页")
    has_prev: bool = Field(..., description="是否有上一页")


class PaginatedPositionsResponse(PaginatedResponse):
    """分页持仓响应Schema"""

    items: list[PositionResponse] = Field(..., description="持仓列表")


class PaginatedBacktestsResponse(PaginatedResponse):
    """分页回测响应Schema"""

    items: list[BacktestResponse] = Field(..., description="回测列表")


class PaginatedPlansResponse(PaginatedResponse):
    """分页方案响应Schema"""

    items: list[TradingPlanResponse] = Field(..., description="方案列表")


class PaginatedTasksResponse(PaginatedResponse):
    """分页任务响应Schema"""

    items: list[TaskResponse] = Field(..., description="任务列表")
