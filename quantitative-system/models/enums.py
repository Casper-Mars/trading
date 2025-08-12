"""枚举定义模块"""

from enum import IntEnum


class PositionStatus(IntEnum):
    """持仓状态"""

    ACTIVE = 1  # 活跃
    CLOSED = 2  # 已平仓
    SUSPENDED = 3  # 暂停


class PositionType(IntEnum):
    """持仓类型"""

    LONG = 1  # 多头
    SHORT = 2  # 空头


class OrderType(IntEnum):
    """订单类型"""

    BUY = 1  # 买入
    SELL = 2  # 卖出


class OrderStatus(IntEnum):
    """订单状态"""

    PENDING = 1  # 待执行
    EXECUTED = 2  # 已执行
    CANCELLED = 3  # 已取消
    FAILED = 4  # 执行失败


class PlanStatus(IntEnum):
    """方案状态"""

    DRAFT = 1  # 草稿
    ACTIVE = 2  # 活跃
    PAUSED = 3  # 暂停
    COMPLETED = 4  # 已完成
    CANCELLED = 5  # 已取消


class PlanType(IntEnum):
    """方案类型"""

    MANUAL = 1  # 手动方案
    AUTO = 2  # 自动方案
    HYBRID = 3  # 混合方案


class BacktestStatus(IntEnum):
    """回测状态"""

    PENDING = 1  # 待执行
    RUNNING = 2  # 运行中
    COMPLETED = 3  # 已完成
    FAILED = 4  # 执行失败
    CANCELLED = 5  # 已取消


class StrategyType(IntEnum):
    """策略类型"""

    TREND_FOLLOWING = 1  # 趋势跟踪
    MEAN_REVERSION = 2  # 均值回归
    MOMENTUM = 3  # 动量策略
    ARBITRAGE = 4  # 套利策略
    CUSTOM = 5  # 自定义策略


class RiskLevel(IntEnum):
    """风险等级"""

    LOW = 1  # 低风险
    MEDIUM = 2  # 中风险
    HIGH = 3  # 高风险
    EXTREME = 4  # 极高风险


class MarketType(IntEnum):
    """市场类型"""

    STOCK = 1  # 股票
    FUTURES = 2  # 期货
    OPTIONS = 3  # 期权
    FOREX = 4  # 外汇
    CRYPTO = 5  # 加密货币


class TimeFrame(IntEnum):
    """时间周期"""

    MINUTE_1 = 1  # 1分钟
    MINUTE_5 = 5  # 5分钟
    MINUTE_15 = 15  # 15分钟
    MINUTE_30 = 30  # 30分钟
    HOUR_1 = 60  # 1小时
    HOUR_4 = 240  # 4小时
    DAY_1 = 1440  # 1天
    WEEK_1 = 10080  # 1周


class DataSource(IntEnum):
    """数据源"""

    TUSHARE = 1  # Tushare
    AKSHARE = 2  # AKShare
    WIND = 3  # Wind
    BLOOMBERG = 4  # Bloomberg
    CUSTOM_API = 5  # 自定义API
    DATA_COLLECTION = 6  # 数据采集系统


class TaskStatus(IntEnum):
    """任务状态"""

    PENDING = 1  # 待执行
    RUNNING = 2  # 运行中
    COMPLETED = 3  # 已完成
    FAILED = 4  # 执行失败
    CANCELLED = 5  # 已取消
    RETRYING = 6  # 重试中


class TaskType(IntEnum):
    """任务类型"""

    DATA_SYNC = 1  # 数据同步
    BACKTEST = 2  # 回测任务
    PLAN_GENERATION = 3  # 方案生成
    POSITION_UPDATE = 4  # 持仓更新
    RISK_ANALYSIS = 5  # 风险分析
    REPORT_GENERATION = 6  # 报告生成


class NotificationType(IntEnum):
    """通知类型"""

    INFO = 1  # 信息
    WARNING = 2  # 警告
    ERROR = 3  # 错误
    SUCCESS = 4  # 成功


class LogLevel(IntEnum):
    """日志级别"""

    DEBUG = 1  # 调试
    INFO = 2  # 信息
    WARNING = 3  # 警告
    ERROR = 4  # 错误
    CRITICAL = 5  # 严重错误


class SentimentType(IntEnum):
    """情感类型"""

    POSITIVE = 1  # 积极
    NEGATIVE = 2  # 消极
    NEUTRAL = 3  # 中性


class CacheType(IntEnum):
    """缓存类型"""

    MEMORY = 1  # 内存缓存
    REDIS = 2  # Redis缓存
    FILE = 3  # 文件缓存
    MARKET_DATA = 4  # 市场数据缓存
    USER_SESSION = 5  # 用户会话缓存
    API_RESPONSE = 6  # API响应缓存
    CALCULATION_RESULT = 7  # 计算结果缓存
    TEMPORARY = 8  # 临时缓存
    QUERY_RESULT = 9  # 查询结果缓存


class APIStatus(IntEnum):
    """API状态"""

    ACTIVE = 1  # 活跃
    INACTIVE = 2  # 非活跃
    MAINTENANCE = 3  # 维护中
    DEPRECATED = 4  # 已废弃


class UserRole(IntEnum):
    """用户角色"""

    ADMIN = 1  # 管理员
    TRADER = 2  # 交易员
    ANALYST = 3  # 分析师
    VIEWER = 4  # 查看者


class SystemStatus(IntEnum):
    """系统状态"""

    RUNNING = 1  # 运行中
    STOPPED = 2  # 已停止
    MAINTENANCE = 3  # 维护中
    ERROR = 4  # 错误状态
