"""自定义异常定义"""


class QuantitativeSystemError(Exception):
    """量化系统基础异常"""
    def __init__(self, message: str, error_code: str = None, details: dict = None):
        self.message = message
        self.error_code = error_code
        self.details = details or {}
        super().__init__(self.message)


class ConfigurationError(QuantitativeSystemError):
    """配置错误"""
    pass


class DatabaseError(QuantitativeSystemError):
    """数据库错误"""
    pass


class DataCollectionError(QuantitativeSystemError):
    """数据采集错误"""
    pass


class DataValidationError(QuantitativeSystemError):
    """数据验证错误"""
    pass


class BacktestError(QuantitativeSystemError):
    """回测错误"""
    pass


class StrategyError(QuantitativeSystemError):
    """策略错误"""
    pass


class PositionError(QuantitativeSystemError):
    """持仓错误"""
    pass


class PlanError(QuantitativeSystemError):
    """方案错误"""
    pass


class AIServiceError(QuantitativeSystemError):
    """AI服务错误"""
    pass


class ExternalServiceError(QuantitativeSystemError):
    """外部服务错误"""
    pass


class CacheError(QuantitativeSystemError):
    """缓存错误"""
    pass


class AuthenticationError(QuantitativeSystemError):
    """认证错误"""
    pass


class AuthorizationError(QuantitativeSystemError):
    """授权错误"""
    pass


class RateLimitError(QuantitativeSystemError):
    """限流错误"""
    pass


class TimeoutError(QuantitativeSystemError):
    """超时错误"""
    pass


class ResourceNotFoundError(QuantitativeSystemError):
    """资源未找到错误"""
    pass


class NotFoundError(QuantitativeSystemError):
    """未找到错误（通用）"""
    pass


class ResourceConflictError(QuantitativeSystemError):
    """资源冲突错误"""
    pass


class BusinessLogicError(QuantitativeSystemError):
    """业务逻辑错误"""
    pass


class SchedulerError(QuantitativeSystemError):
    """调度器错误"""
    pass


class TaskExecutionError(QuantitativeSystemError):
    """任务执行错误"""
    pass