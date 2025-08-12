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

    plan_date: date | None = Field(None, description="方案日期, 默认为今天")
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


# ============= 财务数据相关Schema =============


class FinancialDataBase(BaseSchema):
    """财务数据基础Schema"""

    ts_code: str = Field(..., max_length=20, description="TS代码")
    end_date: str = Field(..., max_length=8, description="报告期")
    report_type: str | None = Field(None, max_length=10, description="报告类型")
    basic_eps: Decimal | None = Field(None, description="基本每股收益")
    diluted_eps: Decimal | None = Field(None, description="稀释每股收益")
    total_revenue: Decimal | None = Field(None, description="营业总收入")
    revenue: Decimal | None = Field(None, description="营业收入")
    n_income: Decimal | None = Field(None, description="净利润")
    n_income_attr_p: Decimal | None = Field(None, description="净利润(不含少数股东损益)")
    total_profit: Decimal | None = Field(None, description="利润总额")
    operate_profit: Decimal | None = Field(None, description="营业利润")
    ebit: Decimal | None = Field(None, description="息税前利润")
    ebitda: Decimal | None = Field(None, description="息税折旧摊销前利润")
    rd_exp: Decimal | None = Field(None, description="研发费用")


class FinancialDataCreate(FinancialDataBase):
    """创建财务数据Schema"""

    ann_date: str | None = Field(None, max_length=8, description="公告日期")
    f_ann_date: str | None = Field(None, max_length=8, description="实际公告日期")
    comp_type: str | None = Field(None, max_length=10, description="公司类型")
    # 包含所有其他财务指标字段
    int_income: Decimal | None = Field(None, description="利息收入")
    prem_earned: Decimal | None = Field(None, description="已赚保费")
    comm_income: Decimal | None = Field(None, description="手续费及佣金收入")
    n_commis_income: Decimal | None = Field(None, description="手续费及佣金净收入")
    n_oth_income: Decimal | None = Field(None, description="其他经营净收益")
    n_oth_b_income: Decimal | None = Field(None, description="加:其他业务净收益")
    prem_income: Decimal | None = Field(None, description="保险业务收入")
    out_prem: Decimal | None = Field(None, description="减:分出保费")
    une_prem_reser: Decimal | None = Field(None, description="提取未到期责任准备金")
    reins_income: Decimal | None = Field(None, description="其中:分保费收入")
    n_sec_tb_income: Decimal | None = Field(None, description="代理买卖证券业务净收入")
    n_sec_uw_income: Decimal | None = Field(None, description="证券承销业务净收入")
    n_asset_mg_income: Decimal | None = Field(None, description="受托客户资产管理业务净收入")
    oth_b_income: Decimal | None = Field(None, description="其他业务收入")
    fv_value_chg_gain: Decimal | None = Field(None, description="加:公允价值变动净收益")
    invest_income: Decimal | None = Field(None, description="加:投资净收益")
    ass_invest_income: Decimal | None = Field(None, description="其中:对联营企业和合营企业的投资收益")
    forex_gain: Decimal | None = Field(None, description="加:汇兑净收益")
    total_cogs: Decimal | None = Field(None, description="营业总成本")
    oper_cost: Decimal | None = Field(None, description="减:营业成本")
    int_exp: Decimal | None = Field(None, description="减:利息支出")
    comm_exp: Decimal | None = Field(None, description="减:手续费及佣金支出")
    biz_tax_surchg: Decimal | None = Field(None, description="减:营业税金及附加")
    sell_exp: Decimal | None = Field(None, description="减:销售费用")
    admin_exp: Decimal | None = Field(None, description="减:管理费用")
    fin_exp: Decimal | None = Field(None, description="减:财务费用")
    assets_impair_loss: Decimal | None = Field(None, description="减:资产减值损失")
    income_tax: Decimal | None = Field(None, description="所得税费用")
    minority_gain: Decimal | None = Field(None, description="少数股东损益")
    oth_compr_income: Decimal | None = Field(None, description="其他综合收益")
    t_compr_income: Decimal | None = Field(None, description="综合收益总额")
    compr_inc_attr_p: Decimal | None = Field(None, description="归属于母公司(或股东)的综合收益总额")
    compr_inc_attr_m_s: Decimal | None = Field(None, description="归属于少数股东的综合收益总额")


class FinancialDataUpdate(BaseSchema):
    """更新财务数据Schema"""

    ann_date: str | None = Field(None, max_length=8, description="公告日期")
    f_ann_date: str | None = Field(None, max_length=8, description="实际公告日期")
    report_type: str | None = Field(None, max_length=10, description="报告类型")
    basic_eps: Decimal | None = Field(None, description="基本每股收益")
    diluted_eps: Decimal | None = Field(None, description="稀释每股收益")
    total_revenue: Decimal | None = Field(None, description="营业总收入")
    revenue: Decimal | None = Field(None, description="营业收入")
    n_income: Decimal | None = Field(None, description="净利润")
    n_income_attr_p: Decimal | None = Field(None, description="净利润(不含少数股东损益)")
    total_profit: Decimal | None = Field(None, description="利润总额")
    operate_profit: Decimal | None = Field(None, description="营业利润")
    ebit: Decimal | None = Field(None, description="息税前利润")
    ebitda: Decimal | None = Field(None, description="息税折旧摊销前利润")
    rd_exp: Decimal | None = Field(None, description="研发费用")


class FinancialDataResponse(FinancialDataBase):
    """财务数据响应Schema"""

    id: int
    ann_date: str | None = Field(None, description="公告日期")
    f_ann_date: str | None = Field(None, description="实际公告日期")
    comp_type: str | None = Field(None, description="公司类型")
    created_at: datetime = Field(..., description="创建时间")
    updated_at: datetime = Field(..., description="更新时间")


class FinancialDataQuery(BaseSchema):
    """财务数据查询Schema"""

    ts_code: str | None = Field(None, description="TS代码")
    start_date: str | None = Field(None, description="开始日期")
    end_date: str | None = Field(None, description="结束日期")
    report_type: str | None = Field(None, description="报告类型")
    limit: int = Field(100, ge=1, le=1000, description="返回数量限制")


# ============= 情感分析相关Schema =============


class SentimentAnalysisBase(BaseSchema):
    """情感分析基础Schema"""

    news_id: int = Field(..., description="新闻ID")
    content_type: str = Field(..., max_length=20, description="内容类型")
    sentiment_score: Decimal = Field(..., description="情感分数(-1到1)")
    sentiment_label: str = Field(..., max_length=20, description="情感标签")
    confidence: Decimal = Field(..., description="置信度(0到1)")
    model_name: str = Field(..., max_length=100, description="使用的模型名称")
    model_version: str | None = Field(None, max_length=50, description="模型版本")


class SentimentAnalysisCreate(SentimentAnalysisBase):
    """创建情感分析Schema"""

    keywords: list[str] | None = Field(None, description="关键词")
    emotions: dict[str, Any] | None = Field(None, description="详细情感分析")
    topics: list[str] | None = Field(None, description="主题标签")
    entities: list[dict[str, Any]] | None = Field(None, description="实体识别结果")
    processing_time: Decimal | None = Field(None, description="处理时间(秒)")
    error_message: str | None = Field(None, description="错误信息")


class SentimentAnalysisUpdate(BaseSchema):
    """更新情感分析Schema"""

    sentiment_score: Decimal | None = Field(None, description="情感分数(-1到1)")
    sentiment_label: str | None = Field(None, max_length=20, description="情感标签")
    confidence: Decimal | None = Field(None, description="置信度(0到1)")
    keywords: list[str] | None = Field(None, description="关键词")
    emotions: dict[str, Any] | None = Field(None, description="详细情感分析")
    topics: list[str] | None = Field(None, description="主题标签")
    entities: list[dict[str, Any]] | None = Field(None, description="实体识别结果")
    error_message: str | None = Field(None, description="错误信息")


class SentimentAnalysisResponse(SentimentAnalysisBase):
    """情感分析响应Schema"""

    id: int
    keywords: list[str] | None = Field(None, description="关键词")
    emotions: dict[str, Any] | None = Field(None, description="详细情感分析")
    topics: list[str] | None = Field(None, description="主题标签")
    entities: list[dict[str, Any]] | None = Field(None, description="实体识别结果")
    processing_time: Decimal | None = Field(None, description="处理时间(秒)")
    error_message: str | None = Field(None, description="错误信息")
    created_at: datetime = Field(..., description="创建时间")
    updated_at: datetime = Field(..., description="更新时间")


class SentimentAnalysisQuery(BaseSchema):
    """情感分析查询Schema"""

    news_id: int | None = Field(None, description="新闻ID")
    content_type: str | None = Field(None, description="内容类型")
    sentiment_label: str | None = Field(None, description="情感标签")
    model_name: str | None = Field(None, description="模型名称")
    min_confidence: Decimal | None = Field(None, description="最小置信度")
    start_date: datetime | None = Field(None, description="开始时间")
    end_date: datetime | None = Field(None, description="结束时间")
    limit: int = Field(100, ge=1, le=1000, description="返回数量限制")


class SentimentBatchRequest(BaseSchema):
    """批量情感分析请求Schema"""

    news_ids: list[int] = Field(..., description="新闻ID列表")
    content_types: list[str] = Field(["title", "content"], description="内容类型列表")
    model_name: str = Field("default", description="使用的模型名称")
    force_reprocess: bool = Field(False, description="是否强制重新处理")


class SentimentBatchResponse(BaseSchema):
    """批量情感分析响应Schema"""

    total_requested: int = Field(..., description="请求总数")
    total_processed: int = Field(..., description="处理成功数")
    total_failed: int = Field(..., description="处理失败数")
    results: list[SentimentAnalysisResponse] = Field(..., description="分析结果列表")
    errors: list[dict[str, Any]] = Field(..., description="错误信息列表")
    processing_time: Decimal = Field(..., description="总处理时间(秒)")


# ============= 股票基础信息相关Schema =============


class StockBasicInfoBase(BaseSchema):
    """股票基础信息基础Schema"""

    ts_code: str = Field(..., max_length=20, description="TS代码")
    symbol: str = Field(..., max_length=10, description="股票代码")
    name: str = Field(..., max_length=20, description="股票名称")
    area: str | None = Field(None, max_length=20, description="地域")
    industry: str | None = Field(None, max_length=50, description="所属行业")
    market: str | None = Field(None, max_length=10, description="市场类型")
    exchange: str | None = Field(None, max_length=10, description="交易所代码")
    list_status: str | None = Field(None, max_length=1, description="上市状态")
    list_date: str | None = Field(None, max_length=8, description="上市日期")
    is_hs: str | None = Field(None, max_length=1, description="是否沪深港通标的")


class StockBasicInfoCreate(StockBasicInfoBase):
    """创建股票基础信息Schema"""

    curr_type: str | None = Field(None, max_length=10, description="交易货币")
    delist_date: str | None = Field(None, max_length=8, description="退市日期")


class StockBasicInfoUpdate(BaseSchema):
    """更新股票基础信息Schema"""

    name: str | None = Field(None, max_length=20, description="股票名称")
    area: str | None = Field(None, max_length=20, description="地域")
    industry: str | None = Field(None, max_length=50, description="所属行业")
    list_status: str | None = Field(None, max_length=1, description="上市状态")
    delist_date: str | None = Field(None, max_length=8, description="退市日期")
    is_hs: str | None = Field(None, max_length=1, description="是否沪深港通标的")


class StockBasicInfoResponse(StockBasicInfoBase):
    """股票基础信息响应Schema"""

    curr_type: str | None = Field(None, description="交易货币")
    delist_date: str | None = Field(None, description="退市日期")
    created_at: datetime = Field(..., description="创建时间")
    updated_at: datetime = Field(..., description="更新时间")


class StockBasicInfoQuery(BaseSchema):
    """股票基础信息查询Schema"""

    ts_code: str | None = Field(None, description="TS代码")
    symbol: str | None = Field(None, description="股票代码")
    name: str | None = Field(None, description="股票名称")
    industry: str | None = Field(None, description="所属行业")
    market: str | None = Field(None, description="市场类型")
    list_status: str | None = Field(None, description="上市状态")
    is_hs: str | None = Field(None, description="是否沪深港通标的")
    limit: int = Field(100, ge=1, le=1000, description="返回数量限制")


# ============= 股票日线数据相关Schema =============


class StockDailyDataBase(BaseSchema):
    """股票日线数据基础Schema"""

    ts_code: str = Field(..., max_length=20, description="TS代码")
    trade_date: str = Field(..., max_length=8, description="交易日期")
    open: Decimal | None = Field(None, description="开盘价")
    high: Decimal | None = Field(None, description="最高价")
    low: Decimal | None = Field(None, description="最低价")
    close: Decimal | None = Field(None, description="收盘价")
    pre_close: Decimal | None = Field(None, description="昨收价")
    change: Decimal | None = Field(None, description="涨跌额")
    pct_chg: Decimal | None = Field(None, description="涨跌幅")
    vol: Decimal | None = Field(None, description="成交量(手)")
    amount: Decimal | None = Field(None, description="成交额(千元)")


class StockDailyDataCreate(StockDailyDataBase):
    """创建股票日线数据Schema"""

    pass


class StockDailyDataUpdate(BaseSchema):
    """更新股票日线数据Schema"""

    open: Decimal | None = Field(None, description="开盘价")
    high: Decimal | None = Field(None, description="最高价")
    low: Decimal | None = Field(None, description="最低价")
    close: Decimal | None = Field(None, description="收盘价")
    pre_close: Decimal | None = Field(None, description="昨收价")
    change: Decimal | None = Field(None, description="涨跌额")
    pct_chg: Decimal | None = Field(None, description="涨跌幅")
    vol: Decimal | None = Field(None, description="成交量(手)")
    amount: Decimal | None = Field(None, description="成交额(千元)")


class StockDailyDataResponse(StockDailyDataBase):
    """股票日线数据响应Schema"""

    id: int
    created_at: datetime = Field(..., description="创建时间")
    updated_at: datetime = Field(..., description="更新时间")


class StockDailyDataQuery(BaseSchema):
    """股票日线数据查询Schema"""

    ts_code: str | None = Field(None, description="TS代码")
    start_date: str | None = Field(None, description="开始日期")
    end_date: str | None = Field(None, description="结束日期")
    limit: int = Field(100, ge=1, le=1000, description="返回数量限制")


# ============= 新闻数据相关Schema =============


class NewsDataBase(BaseSchema):
    """新闻数据基础Schema"""

    title: str = Field(..., max_length=500, description="新闻标题")
    content: str = Field(..., description="新闻内容")
    source: str | None = Field(None, max_length=100, description="新闻来源")
    author: str | None = Field(None, max_length=100, description="作者")
    publish_time: datetime | None = Field(None, description="发布时间")
    url: str | None = Field(None, max_length=1000, description="原文链接")
    category: str | None = Field(None, max_length=50, description="新闻分类")
    tags: list[str] | None = Field(None, description="标签")
    related_stocks: list[str] | None = Field(None, description="相关股票")


class NewsDataCreate(NewsDataBase):
    """创建新闻数据Schema"""

    sentiment_score: Decimal | None = Field(None, description="情感分数")
    sentiment_label: str | None = Field(None, max_length=20, description="情感标签")
    keywords: list[str] | None = Field(None, description="关键词")
    summary: str | None = Field(None, description="新闻摘要")
    is_processed: bool = Field(False, description="是否已处理")


class NewsDataUpdate(BaseSchema):
    """更新新闻数据Schema"""

    title: str | None = Field(None, max_length=500, description="新闻标题")
    content: str | None = Field(None, description="新闻内容")
    source: str | None = Field(None, max_length=100, description="新闻来源")
    author: str | None = Field(None, max_length=100, description="作者")
    publish_time: datetime | None = Field(None, description="发布时间")
    category: str | None = Field(None, max_length=50, description="新闻分类")
    tags: list[str] | None = Field(None, description="标签")
    related_stocks: list[str] | None = Field(None, description="相关股票")
    sentiment_score: Decimal | None = Field(None, description="情感分数")
    sentiment_label: str | None = Field(None, max_length=20, description="情感标签")
    keywords: list[str] | None = Field(None, description="关键词")
    summary: str | None = Field(None, description="新闻摘要")
    is_processed: bool | None = Field(None, description="是否已处理")


class NewsDataResponse(NewsDataBase):
    """新闻数据响应Schema"""

    id: int
    sentiment_score: Decimal | None = Field(None, description="情感分数")
    sentiment_label: str | None = Field(None, description="情感标签")
    keywords: list[str] | None = Field(None, description="关键词")
    summary: str | None = Field(None, description="新闻摘要")
    is_processed: bool = Field(..., description="是否已处理")
    created_at: datetime = Field(..., description="创建时间")
    updated_at: datetime = Field(..., description="更新时间")


class NewsDataQuery(BaseSchema):
    """新闻数据查询Schema"""

    title: str | None = Field(None, description="新闻标题关键词")
    source: str | None = Field(None, description="新闻来源")
    category: str | None = Field(None, description="新闻分类")
    related_stocks: list[str] | None = Field(None, description="相关股票")
    sentiment_label: str | None = Field(None, description="情感标签")
    is_processed: bool | None = Field(None, description="是否已处理")
    start_date: datetime | None = Field(None, description="开始时间")
    end_date: datetime | None = Field(None, description="结束时间")
    limit: int = Field(100, ge=1, le=1000, description="返回数量限制")


# ============= 分页响应Schema =============


class PaginatedFinancialDataResponse(PaginatedResponse):
    """分页财务数据响应Schema"""

    items: list[FinancialDataResponse] = Field(..., description="财务数据列表")


class PaginatedSentimentAnalysisResponse(PaginatedResponse):
    """分页情感分析响应Schema"""

    items: list[SentimentAnalysisResponse] = Field(..., description="情感分析列表")


class PaginatedStockBasicInfoResponse(PaginatedResponse):
    """分页股票基础信息响应Schema"""

    items: list[StockBasicInfoResponse] = Field(..., description="股票基础信息列表")


class PaginatedStockDailyDataResponse(PaginatedResponse):
    """分页股票日线数据响应Schema"""

    items: list[StockDailyDataResponse] = Field(..., description="股票日线数据列表")


class PaginatedNewsDataResponse(PaginatedResponse):
    """分页新闻数据响应Schema"""

    items: list[NewsDataResponse] = Field(..., description="新闻数据列表")
