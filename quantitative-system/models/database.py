"""数据库模型定义"""

from datetime import date, datetime
from decimal import Decimal
from typing import Any

from sqlalchemy import Index, UniqueConstraint
from sqlmodel import JSON, Column, Field, SQLModel, Text

from .enums import (
    BacktestStatus,
    DataSource,
    LogLevel,
    PlanStatus,
    PlanType,
    PositionStatus,
    PositionType,
    RiskLevel,
    StrategyType,
    TaskStatus,
    TaskType,
    TimeFrame,
)


class Position(SQLModel, table=True):
    """持仓表"""

    __tablename__ = "positions"

    id: int | None = Field(default=None, primary_key=True)
    symbol: str = Field(max_length=20, description="股票代码")
    name: str = Field(max_length=100, description="股票名称")
    position_type: PositionType = Field(description="持仓类型:1-多头,2-空头")
    quantity: int = Field(description="持仓数量")
    avg_cost: Decimal = Field(max_digits=10, decimal_places=4, description="平均成本")
    current_price: Decimal | None = Field(
        default=None, max_digits=10, decimal_places=4, description="当前价格"
    )
    market_value: Decimal | None = Field(
        default=None, max_digits=15, decimal_places=2, description="市值"
    )
    unrealized_pnl: Decimal | None = Field(
        default=None, max_digits=15, decimal_places=2, description="浮动盈亏"
    )
    realized_pnl: Decimal = Field(
        default=0, max_digits=15, decimal_places=2, description="已实现盈亏"
    )
    status: PositionStatus = Field(
        default=PositionStatus.ACTIVE, description="状态:1-活跃,2-已平仓,3-暂停"
    )
    open_date: date = Field(description="开仓日期")
    close_date: date | None = Field(default=None, description="平仓日期")
    notes: str | None = Field(default=None, max_length=500, description="备注")
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

    id: int | None = Field(default=None, primary_key=True)
    name: str = Field(max_length=200, description="回测名称")
    strategy_type: StrategyType = Field(description="策略类型")
    strategy_params: dict[str, Any] = Field(
        sa_column=Column(JSON), description="策略参数"
    )
    symbols: list[str] = Field(sa_column=Column(JSON), description="标的列表")
    start_date: date = Field(description="开始日期")
    end_date: date = Field(description="结束日期")
    initial_cash: Decimal = Field(
        max_digits=15, decimal_places=2, description="初始资金"
    )
    final_value: Decimal | None = Field(
        default=None, max_digits=15, decimal_places=2, description="最终价值"
    )
    total_return: Decimal | None = Field(
        default=None, max_digits=8, decimal_places=4, description="总收益率"
    )
    annual_return: Decimal | None = Field(
        default=None, max_digits=8, decimal_places=4, description="年化收益率"
    )
    sharpe_ratio: Decimal | None = Field(
        default=None, max_digits=8, decimal_places=4, description="夏普比率"
    )
    max_drawdown: Decimal | None = Field(
        default=None, max_digits=8, decimal_places=4, description="最大回撤"
    )
    win_rate: Decimal | None = Field(
        default=None, max_digits=5, decimal_places=4, description="胜率"
    )
    total_trades: int | None = Field(default=None, description="总交易次数")
    avg_trade_return: Decimal | None = Field(
        default=None, max_digits=8, decimal_places=4, description="平均交易收益"
    )
    volatility: Decimal | None = Field(
        default=None, max_digits=8, decimal_places=4, description="波动率"
    )
    calmar_ratio: Decimal | None = Field(
        default=None, max_digits=8, decimal_places=4, description="卡玛比率"
    )
    status: BacktestStatus = Field(default=BacktestStatus.PENDING, description="状态")
    raw_data: dict[str, Any] | None = Field(
        default=None, sa_column=Column(JSON), description="原始数据"
    )
    error_message: str | None = Field(
        default=None, sa_column=Column(Text), description="错误信息"
    )
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

    id: int | None = Field(default=None, primary_key=True)
    plan_date: date = Field(description="方案日期")
    title: str = Field(max_length=200, description="方案标题")
    content: str = Field(sa_column=Column(Text), description="方案内容(Markdown格式)")
    plan_type: PlanType = Field(description="方案类型:1-手动,2-自动,3-混合")
    risk_level: RiskLevel = Field(description="风险等级:1-低,2-中,3-高,4-极高")
    target_return: Decimal | None = Field(
        default=None, max_digits=8, decimal_places=4, description="目标收益率"
    )
    max_drawdown_limit: Decimal | None = Field(
        default=None, max_digits=8, decimal_places=4, description="最大回撤限制"
    )
    position_limit: Decimal | None = Field(
        default=None, max_digits=8, decimal_places=4, description="仓位限制"
    )
    recommendations: dict[str, Any] | None = Field(
        default=None, sa_column=Column(JSON), description="推荐操作"
    )
    backtest_results: dict[str, Any] | None = Field(
        default=None, sa_column=Column(JSON), description="回测结果"
    )
    ai_analysis: dict[str, Any] | None = Field(
        default=None, sa_column=Column(JSON), description="AI分析结果"
    )
    status: PlanStatus = Field(default=PlanStatus.DRAFT, description="状态")
    execution_rate: Decimal | None = Field(
        default=None, max_digits=5, decimal_places=4, description="执行率"
    )
    actual_return: Decimal | None = Field(
        default=None, max_digits=8, decimal_places=4, description="实际收益率"
    )
    notes: str | None = Field(default=None, max_length=1000, description="备注")
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

    id: int | None = Field(default=None, primary_key=True)
    cache_key: str = Field(max_length=200, description="缓存键")
    symbol: str | None = Field(default=None, max_length=20, description="股票代码")
    data_type: str = Field(max_length=50, description="数据类型")
    time_frame: TimeFrame | None = Field(default=None, description="时间周期")
    start_date: date | None = Field(default=None, description="开始日期")
    end_date: date | None = Field(default=None, description="结束日期")
    data_source: DataSource = Field(description="数据源")
    data_content: dict[str, Any] = Field(sa_column=Column(JSON), description="数据内容")
    data_size: int = Field(description="数据大小(字节)")
    hit_count: int = Field(default=0, description="命中次数")
    last_hit_at: datetime | None = Field(default=None, description="最后命中时间")
    expires_at: datetime | None = Field(default=None, description="过期时间")
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

    id: int | None = Field(default=None, primary_key=True)
    level: LogLevel = Field(description="日志级别")
    module: str = Field(max_length=100, description="模块名称")
    function: str | None = Field(default=None, max_length=100, description="函数名称")
    message: str = Field(sa_column=Column(Text), description="日志消息")
    details: dict[str, Any] | None = Field(
        default=None, sa_column=Column(JSON), description="详细信息"
    )
    user_id: str | None = Field(default=None, max_length=50, description="用户ID")
    session_id: str | None = Field(default=None, max_length=100, description="会话ID")
    request_id: str | None = Field(default=None, max_length=100, description="请求ID")
    ip_address: str | None = Field(default=None, max_length=45, description="IP地址")
    user_agent: str | None = Field(default=None, max_length=500, description="用户代理")
    execution_time: Decimal | None = Field(
        default=None, max_digits=10, decimal_places=6, description="执行时间(秒)"
    )
    stack_trace: str | None = Field(
        default=None, sa_column=Column(Text), description="堆栈跟踪"
    )
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

    id: int | None = Field(default=None, primary_key=True)
    name: str = Field(max_length=200, description="任务名称")
    task_type: TaskType = Field(description="任务类型")
    status: TaskStatus = Field(default=TaskStatus.PENDING, description="任务状态")
    priority: int = Field(default=5, description="优先级(1-10,数字越小优先级越高)")
    params: dict[str, Any] | None = Field(
        default=None, sa_column=Column(JSON), description="任务参数"
    )
    result: dict[str, Any] | None = Field(
        default=None, sa_column=Column(JSON), description="任务结果"
    )
    error_message: str | None = Field(
        default=None, sa_column=Column(Text), description="错误信息"
    )
    retry_count: int = Field(default=0, description="重试次数")
    max_retries: int = Field(default=3, description="最大重试次数")
    scheduled_at: datetime | None = Field(default=None, description="计划执行时间")
    started_at: datetime | None = Field(default=None, description="开始执行时间")
    completed_at: datetime | None = Field(default=None, description="完成时间")
    execution_time: Decimal | None = Field(
        default=None, max_digits=10, decimal_places=6, description="执行时间(秒)"
    )
    created_by: str | None = Field(default=None, max_length=50, description="创建者")
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    updated_at: datetime = Field(default_factory=datetime.now, description="更新时间")

    __table_args__ = (
        Index("idx_tasks_type", "task_type"),
        Index("idx_tasks_status", "status"),
        Index("idx_tasks_priority", "priority"),
        Index("idx_tasks_scheduled_at", "scheduled_at"),
        Index("idx_tasks_created_at", "created_at"),
    )


class StockBasicInfo(SQLModel, table=True):
    """股票基本信息表"""

    __tablename__ = "stock_basic_info"

    ts_code: str = Field(primary_key=True, max_length=20, description="TS代码")
    symbol: str = Field(max_length=10, description="股票代码")
    name: str = Field(max_length=20, description="股票名称")
    area: str | None = Field(default=None, max_length=20, description="地域")
    industry: str | None = Field(default=None, max_length=50, description="所属行业")
    market: str | None = Field(default=None, max_length=10, description="市场类型")
    exchange: str | None = Field(default=None, max_length=10, description="交易所代码")
    curr_type: str | None = Field(default=None, max_length=10, description="交易货币")
    list_status: str | None = Field(default=None, max_length=1, description="上市状态")
    list_date: str | None = Field(default=None, max_length=8, description="上市日期")
    delist_date: str | None = Field(default=None, max_length=8, description="退市日期")
    is_hs: str | None = Field(
        default=None, max_length=1, description="是否沪深港通标的"
    )
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    updated_at: datetime = Field(default_factory=datetime.now, description="更新时间")

    __table_args__ = (
        Index("idx_stock_basic_symbol", "symbol"),
        Index("idx_stock_basic_name", "name"),
        Index("idx_stock_basic_industry", "industry"),
        Index("idx_stock_basic_market", "market"),
        Index("idx_stock_basic_list_status", "list_status"),
    )


class StockDailyData(SQLModel, table=True):
    """股票日线数据表"""

    __tablename__ = "stock_daily_data"

    id: int | None = Field(default=None, primary_key=True)
    ts_code: str = Field(max_length=20, description="TS代码")
    trade_date: str = Field(max_length=8, description="交易日期")
    open: Decimal | None = Field(
        default=None, max_digits=10, decimal_places=4, description="开盘价"
    )
    high: Decimal | None = Field(
        default=None, max_digits=10, decimal_places=4, description="最高价"
    )
    low: Decimal | None = Field(
        default=None, max_digits=10, decimal_places=4, description="最低价"
    )
    close: Decimal | None = Field(
        default=None, max_digits=10, decimal_places=4, description="收盘价"
    )
    pre_close: Decimal | None = Field(
        default=None, max_digits=10, decimal_places=4, description="昨收价"
    )
    change: Decimal | None = Field(
        default=None, max_digits=10, decimal_places=4, description="涨跌额"
    )
    pct_chg: Decimal | None = Field(
        default=None, max_digits=8, decimal_places=4, description="涨跌幅"
    )
    vol: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=2, description="成交量(手)"
    )
    amount: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="成交额(千元)"
    )
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    updated_at: datetime = Field(default_factory=datetime.now, description="更新时间")

    __table_args__ = (
        Index("idx_stock_daily_ts_code", "ts_code"),
        Index("idx_stock_daily_trade_date", "trade_date"),
        Index("idx_stock_daily_ts_code_date", "ts_code", "trade_date"),
        UniqueConstraint("ts_code", "trade_date", name="uq_stock_daily_ts_code_date"),
    )


class NewsData(SQLModel, table=True):
    """新闻数据表"""

    __tablename__ = "news_data"

    id: int | None = Field(default=None, primary_key=True)
    title: str = Field(max_length=500, description="新闻标题")
    content: str = Field(sa_column=Column(Text), description="新闻内容")
    source: str | None = Field(default=None, max_length=100, description="新闻来源")
    author: str | None = Field(default=None, max_length=100, description="作者")
    publish_time: datetime | None = Field(default=None, description="发布时间")
    url: str | None = Field(default=None, max_length=1000, description="原文链接")
    category: str | None = Field(default=None, max_length=50, description="新闻分类")
    tags: list[str] | None = Field(
        default=None, sa_column=Column(JSON), description="标签"
    )
    related_stocks: list[str] | None = Field(
        default=None, sa_column=Column(JSON), description="相关股票"
    )
    sentiment_score: Decimal | None = Field(
        default=None, max_digits=5, decimal_places=4, description="情感分数"
    )
    sentiment_label: str | None = Field(
        default=None, max_length=20, description="情感标签"
    )
    keywords: list[str] | None = Field(
        default=None, sa_column=Column(JSON), description="关键词"
    )
    summary: str | None = Field(
        default=None, sa_column=Column(Text), description="新闻摘要"
    )
    is_processed: bool = Field(default=False, description="是否已处理")
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    updated_at: datetime = Field(default_factory=datetime.now, description="更新时间")

    __table_args__ = (
        Index("idx_news_title", "title"),
        Index("idx_news_source", "source"),
        Index("idx_news_publish_time", "publish_time"),
        Index("idx_news_category", "category"),
        Index("idx_news_is_processed", "is_processed"),
        Index("idx_news_created_at", "created_at"),
    )


class DataCollectionTask(SQLModel, table=True):
    """数据采集任务表"""

    __tablename__ = "data_collection_tasks"

    id: int | None = Field(default=None, primary_key=True)
    task_name: str = Field(max_length=200, description="任务名称")
    task_type: str = Field(max_length=50, description="任务类型")
    data_source: str = Field(max_length=50, description="数据源")
    target_symbols: list[str] | None = Field(
        default=None, sa_column=Column(JSON), description="目标股票代码"
    )
    start_date: date | None = Field(default=None, description="开始日期")
    end_date: date | None = Field(default=None, description="结束日期")
    schedule_config: dict[str, Any] | None = Field(
        default=None, sa_column=Column(JSON), description="调度配置"
    )
    status: TaskStatus = Field(default=TaskStatus.PENDING, description="任务状态")
    progress: int = Field(default=0, description="进度百分比")
    total_records: int | None = Field(default=None, description="总记录数")
    processed_records: int = Field(default=0, description="已处理记录数")
    failed_records: int = Field(default=0, description="失败记录数")
    last_run_at: datetime | None = Field(default=None, description="最后运行时间")
    next_run_at: datetime | None = Field(default=None, description="下次运行时间")
    error_message: str | None = Field(
        default=None, sa_column=Column(Text), description="错误信息"
    )
    config: dict[str, Any] | None = Field(
        default=None, sa_column=Column(JSON), description="任务配置"
    )
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    updated_at: datetime = Field(default_factory=datetime.now, description="更新时间")

    __table_args__ = (
        Index("idx_data_collection_task_type", "task_type"),
        Index("idx_data_collection_status", "status"),
        Index("idx_data_collection_data_source", "data_source"),
        Index("idx_data_collection_last_run_at", "last_run_at"),
        Index("idx_data_collection_next_run_at", "next_run_at"),
    )


class FinancialData(SQLModel, table=True):
    """财务数据表"""

    __tablename__ = "financial_data"

    id: int | None = Field(default=None, primary_key=True)
    ts_code: str = Field(max_length=20, description="TS代码")
    ann_date: str | None = Field(default=None, max_length=8, description="公告日期")
    f_ann_date: str | None = Field(
        default=None, max_length=8, description="实际公告日期"
    )
    end_date: str = Field(max_length=8, description="报告期")
    report_type: str | None = Field(default=None, max_length=10, description="报告类型")
    comp_type: str | None = Field(default=None, max_length=10, description="公司类型")
    basic_eps: Decimal | None = Field(
        default=None, max_digits=10, decimal_places=4, description="基本每股收益"
    )
    diluted_eps: Decimal | None = Field(
        default=None, max_digits=10, decimal_places=4, description="稀释每股收益"
    )
    total_revenue: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="营业总收入"
    )
    revenue: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="营业收入"
    )
    int_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="利息收入"
    )
    prem_earned: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="已赚保费"
    )
    comm_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="手续费及佣金收入"
    )
    n_commis_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="手续费及佣金净收入"
    )
    n_oth_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="其他经营净收益"
    )
    n_oth_b_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="加:其他业务净收益"
    )
    prem_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="保险业务收入"
    )
    out_prem: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="减:分出保费"
    )
    une_prem_reser: Decimal | None = Field(
        default=None,
        max_digits=20,
        decimal_places=4,
        description="提取未到期责任准备金",
    )
    reins_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="其中:分保费收入"
    )
    n_sec_tb_income: Decimal | None = Field(
        default=None,
        max_digits=20,
        decimal_places=4,
        description="代理买卖证券业务净收入",
    )
    n_sec_uw_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="证券承销业务净收入"
    )
    n_asset_mg_income: Decimal | None = Field(
        default=None,
        max_digits=20,
        decimal_places=4,
        description="受托客户资产管理业务净收入",
    )
    oth_b_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="其他业务收入"
    )
    fv_value_chg_gain: Decimal | None = Field(
        default=None,
        max_digits=20,
        decimal_places=4,
        description="加:公允价值变动净收益",
    )
    invest_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="加:投资净收益"
    )
    ass_invest_income: Decimal | None = Field(
        default=None,
        max_digits=20,
        decimal_places=4,
        description="其中:对联营企业和合营企业的投资收益",
    )
    forex_gain: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="加:汇兑净收益"
    )
    total_cogs: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="营业总成本"
    )
    oper_cost: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="减:营业成本"
    )
    int_exp: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="减:利息支出"
    )
    comm_exp: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="减:手续费及佣金支出"
    )
    biz_tax_surchg: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="减:营业税金及附加"
    )
    sell_exp: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="减:销售费用"
    )
    admin_exp: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="减:管理费用"
    )
    fin_exp: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="减:财务费用"
    )
    assets_impair_loss: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="减:资产减值损失"
    )
    prem_refund: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="退保金"
    )
    compens_payout: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="赔付总支出"
    )
    reser_insur_liab: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="提取保险责任准备金"
    )
    div_payt: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="保户红利支出"
    )
    reins_exp: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="分保费用"
    )
    oper_exp: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="营业支出"
    )
    compens_payout_refu: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="减:摊回赔付支出"
    )
    insur_reser_refu: Decimal | None = Field(
        default=None,
        max_digits=20,
        decimal_places=4,
        description="减:摊回保险责任准备金",
    )
    reins_cost_refund: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="减:摊回分保费用"
    )
    other_bus_cost: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="其他业务成本"
    )
    operate_profit: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="营业利润"
    )
    non_oper_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="加:营业外收入"
    )
    non_oper_exp: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="减:营业外支出"
    )
    nca_disploss: Decimal | None = Field(
        default=None,
        max_digits=20,
        decimal_places=4,
        description="其中:减:非流动资产处置净损失",
    )
    total_profit: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="利润总额"
    )
    income_tax: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="所得税费用"
    )
    n_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="净利润"
    )
    n_income_attr_p: Decimal | None = Field(
        default=None,
        max_digits=20,
        decimal_places=4,
        description="净利润(不含少数股东损益)",
    )
    minority_gain: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="少数股东损益"
    )
    oth_compr_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="其他综合收益"
    )
    t_compr_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="综合收益总额"
    )
    compr_inc_attr_p: Decimal | None = Field(
        default=None,
        max_digits=20,
        decimal_places=4,
        description="归属于母公司(或股东)的综合收益总额",
    )
    compr_inc_attr_m_s: Decimal | None = Field(
        default=None,
        max_digits=20,
        decimal_places=4,
        description="归属于少数股东的综合收益总额",
    )
    ebit: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="息税前利润"
    )
    ebitda: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="息税折旧摊销前利润"
    )
    insurance_exp: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="保险业务支出"
    )
    undist_profit: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="年初未分配利润"
    )
    distable_profit: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="可分配利润"
    )
    rd_exp: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="研发费用"
    )
    fin_exp_int_exp: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="财务费用:利息费用"
    )
    fin_exp_int_inc: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="财务费用:利息收入"
    )
    transfer_surplus_rese: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="盈余公积转入"
    )
    transfer_housing_imprest: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="住房周转金转入"
    )
    transfer_oth: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="其他转入"
    )
    adj_lossgain: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="调整以前年度损益"
    )
    withdra_legal_surplus: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="提取法定盈余公积"
    )
    withdra_legal_pubfund: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="提取法定公益金"
    )
    withdra_biz_devfund: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="提取企业发展基金"
    )
    withdra_rese_fund: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="提取储备基金"
    )
    withdra_oth_ersu: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="提取任意盈余公积金"
    )
    workers_welfare: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="职工奖金福利"
    )
    distr_profit_shrhder: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="可供股东分配的利润"
    )
    prfshare_payable_dvd: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="应付优先股股利"
    )
    comshare_payable_dvd: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="应付普通股股利"
    )
    capit_comstock_div: Decimal | None = Field(
        default=None,
        max_digits=20,
        decimal_places=4,
        description="转作股本的普通股股利",
    )
    net_after_nr_lp_correct: Decimal | None = Field(
        default=None,
        max_digits=20,
        decimal_places=4,
        description="扣除非经常性损益后的净利润",
    )
    credit_impa_loss: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="信用减值损失"
    )
    net_expo_hedging_benefits: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="净敞口套期收益"
    )
    oth_impair_loss_assets: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="其他资产减值损失"
    )
    total_opcost: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="营业总成本"
    )
    amodcost_fin_assets: Decimal | None = Field(
        default=None,
        max_digits=20,
        decimal_places=4,
        description="以摊余成本计量的金融资产终止确认收益",
    )
    oth_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="其他收益"
    )
    asset_disp_income: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="资产处置收益"
    )
    continued_net_profit: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="持续经营净利润"
    )
    end_net_profit: Decimal | None = Field(
        default=None, max_digits=20, decimal_places=4, description="终止经营净利润"
    )
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    updated_at: datetime = Field(default_factory=datetime.now, description="更新时间")

    __table_args__ = (
        Index("idx_financial_ts_code", "ts_code"),
        Index("idx_financial_end_date", "end_date"),
        Index("idx_financial_ann_date", "ann_date"),
        Index("idx_financial_ts_code_end_date", "ts_code", "end_date"),
        UniqueConstraint(
            "ts_code",
            "end_date",
            "report_type",
            name="uq_financial_ts_code_end_date_type",
        ),
    )


# 删除重复的表定义，使用已存在的NewsData、StockBasicInfo和StockDailyData


class SentimentAnalysis(SQLModel, table=True):
    """情感分析结果表"""

    __tablename__ = "sentiment_analysis"

    id: int | None = Field(default=None, primary_key=True)
    news_id: int = Field(description="新闻ID, 关联news_data表")
    content_type: str = Field(
        max_length=20, description="内容类型: title, content, summary"
    )
    sentiment_score: Decimal = Field(
        max_digits=5, decimal_places=4, description="情感分数(-1到1)"
    )
    sentiment_label: str = Field(
        max_length=20, description="情感标签: positive, negative, neutral"
    )
    confidence: Decimal = Field(
        max_digits=5, decimal_places=4, description="置信度(0到1)"
    )
    model_name: str = Field(max_length=100, description="使用的模型名称")
    model_version: str | None = Field(
        default=None, max_length=50, description="模型版本"
    )
    keywords: list[str] | None = Field(
        default=None, sa_column=Column(JSON), description="关键词"
    )
    emotions: dict[str, Any] | None = Field(
        default=None, sa_column=Column(JSON), description="详细情感分析"
    )
    topics: list[str] | None = Field(
        default=None, sa_column=Column(JSON), description="主题标签"
    )
    entities: list[dict[str, Any]] | None = Field(
        default=None, sa_column=Column(JSON), description="实体识别结果"
    )
    processing_time: Decimal | None = Field(
        default=None, max_digits=10, decimal_places=6, description="处理时间(秒)"
    )
    error_message: str | None = Field(
        default=None, sa_column=Column(Text), description="错误信息"
    )
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    updated_at: datetime = Field(default_factory=datetime.now, description="更新时间")

    __table_args__ = (
        Index("idx_sentiment_news_id", "news_id"),
        Index("idx_sentiment_content_type", "content_type"),
        Index("idx_sentiment_label", "sentiment_label"),
        Index("idx_sentiment_score", "sentiment_score"),
        Index("idx_sentiment_created_at", "created_at"),
        UniqueConstraint(
            "news_id",
            "content_type",
            "model_name",
            name="uq_sentiment_news_content_model",
        ),
    )
