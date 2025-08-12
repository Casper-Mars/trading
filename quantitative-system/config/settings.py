from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """应用配置类"""

    # 应用基础配置
    app_name: str = "Quantitative Trading System"
    app_version: str = "1.0.0"
    debug: bool = False
    environment: str = "development"  # development, staging, production

    # 数据库配置 (使用MySQL数据库)
    database_url: str = (
        "mysql+pymysql://root:root123@localhost:3306/trading_data?charset=utf8mb4"
    )
    database_echo: bool = False
    database_pool_size: int = 10
    database_max_overflow: int = 20
    database_pool_timeout: int = 30
    database_pool_recycle: int = 3600

    # MySQL配置（Docker Compose环境）
    mysql_host: str = "localhost"
    mysql_port: int = 3306
    mysql_user: str = "root"
    mysql_password: str = "root123"  # noqa: S105
    mysql_database: str = "trading_data"
    mysql_charset: str = "utf8mb4"

    # Redis配置
    redis_host: str = "localhost"
    redis_port: int = 6379
    redis_db: int = 0
    redis_password: str | None = None
    redis_url: str | None = None

    # 阿里百炼配置
    dashscope_api_key: str | None = None
    dashscope_base_url: str = "https://dashscope.aliyuncs.com/api/v1"
    dashscope_timeout: int = 30
    dashscope_max_retries: int = 3

    # 数据采集系统配置
    data_collection_base_url: str = "http://localhost:8080"
    data_collection_timeout: int = 30
    data_collection_max_retries: int = 3

    # Tushare API配置
    tushare_token: str | None = None
    tushare_timeout: int = 30
    tushare_max_retries: int = 3
    tushare_retry_delay: float = 1.0

    # 回测配置
    backtest_initial_cash: float = 100000.0
    backtest_commission: float = 0.001  # 0.1%
    backtest_slippage: float = 0.0001  # 0.01%

    # 日志配置
    log_level: str = "INFO"
    log_file: str = "logs/quantitative_system.log"
    log_rotation: str = "1 day"
    log_retention: str = "30 days"

    # API配置
    api_host: str = "0.0.0.0"
    api_port: int = 8000
    api_reload: bool = False

    # 安全配置
    secret_key: str = "your-secret-key-change-in-production"  # noqa: S105
    access_token_expire_minutes: int = 30

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"
        case_sensitive = False

    @property
    def redis_dsn(self) -> str:
        """构建Redis连接字符串"""
        if self.redis_url:
            return self.redis_url

        if self.redis_password:
            return f"redis://:{self.redis_password}@{self.redis_host}:{self.redis_port}/{self.redis_db}"
        else:
            return f"redis://{self.redis_host}:{self.redis_port}/{self.redis_db}"

    def is_development(self) -> bool:
        """是否为开发环境"""
        return self.environment == "development"

    def is_production(self) -> bool:
        """是否为生产环境"""
        return self.environment == "production"


# 全局配置实例
settings = Settings()


def get_settings() -> Settings:
    """获取应用配置实例"""
    return settings
