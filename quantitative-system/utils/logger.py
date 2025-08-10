import sys
from pathlib import Path
from typing import Any

from loguru import logger

from config.settings import settings


def setup_logger() -> Any:
    """配置日志系统"""
    # 移除默认的控制台处理器
    logger.remove()

    # 控制台日志配置
    console_format = (
        "<green>{time:YYYY-MM-DD HH:mm:ss}</green> | "
        "<level>{level: <8}</level> | "
        "<cyan>{name}</cyan>:<cyan>{function}</cyan>:<cyan>{line}</cyan> - "
        "<level>{message}</level>"
    )

    logger.add(
        sys.stdout,
        format=console_format,
        level=settings.log_level,
        colorize=True,
        backtrace=True,
        diagnose=True,
    )

    # 文件日志配置
    if settings.log_file:
        # 确保日志目录存在
        log_file_path = Path(settings.log_file)
        log_file_path.parent.mkdir(parents=True, exist_ok=True)

        file_format = (
            "{time:YYYY-MM-DD HH:mm:ss} | "
            "{level: <8} | "
            "{name}:{function}:{line} - "
            "{message}"
        )

        logger.add(
            settings.log_file,
            format=file_format,
            level=settings.log_level,
            rotation=settings.log_rotation,
            retention=settings.log_retention,
            compression="zip",
            backtrace=True,
            diagnose=True,
        )

    return logger


# 创建全局日志实例
app_logger = setup_logger()


def get_logger(name: str | None = None) -> Any:
    """获取日志实例"""
    if name:
        return logger.bind(name=name)
    return logger


# 导出常用的日志方法
def debug(message: str, **kwargs: Any) -> None:
    """调试日志"""
    logger.debug(message, **kwargs)


def info(message: str, **kwargs: Any) -> None:
    """信息日志"""
    logger.info(message, **kwargs)


def warning(message: str, **kwargs: Any) -> None:
    """警告日志"""
    logger.warning(message, **kwargs)


def error(message: str, **kwargs: Any) -> None:
    """错误日志"""
    logger.error(message, **kwargs)


def critical(message: str, **kwargs: Any) -> None:
    """严重错误日志"""
    logger.critical(message, **kwargs)


def exception(message: str, **kwargs: Any) -> None:
    """异常日志(包含堆栈信息)"""
    logger.exception(message, **kwargs)
