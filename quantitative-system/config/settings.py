from pydantic_settings import BaseSettings
from typing import Optional
import os


class Settings(BaseSettings):
    """应用配置类"""
    
    # 应用基础配置
    app_name: str = "Quantitative Trading System"
    app_version: str = "1.0.0"
    debug: bool = False
    environment: str = "development"  # development, staging, production
    
    # 数据库配置
    database_url: str = "sqlite:///./quantitative_system.db"
    database_echo: bool = False
    
    # Redis配置
    redis_host: str = "localhost"
    redis_port: int = 6379
    redis_db: int = 0
    redis_password: Optional[str] = None
    redis_url: Optional[str] = None
    
    # 阿里百炼配置
    dashscope_api_key: Optional[str] = None
    dashscope_base_url: str = "https://dashscope.aliyuncs.com/api/v1"
    dashscope_timeout: int = 30
    dashscope_max_retries: int = 3
    
    # 数据采集系统配置
    data_collection_base_url: str = "http://localhost:8080"
    data_collection_timeout: int = 30
    data_collection_max_retries: int = 3
    
    # 回测配置
    backtest_initial_cash: float = 100000.0
    backtest_commission: float = 0.001  # 0.1%
    backtest_slippage: float = 0.0001   # 0.01%
    
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
    secret_key: str = "your-secret-key-change-in-production"
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