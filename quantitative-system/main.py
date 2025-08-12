"""FastAPI应用入口文件

实现FastAPI应用框架搭建，包括：
- FastAPI应用初始化
- CORS配置
- 中间件配置
- 异常处理
- 依赖注入
- API文档配置
"""

from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager

from fastapi import FastAPI, Request, status
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from sqlalchemy import text
from sqlalchemy.exc import SQLAlchemyError

from api.routes import api_router
from config.database import get_db_session
from config.settings import get_settings
from utils.exceptions import (
    BusinessError,
    DataNotFoundError,
    ValidationError,
)
from utils.logger import get_logger

# 获取配置和日志器
settings = get_settings()
logger = get_logger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncGenerator[None, None]:
    """应用生命周期管理"""
    logger.info("启动FastAPI应用")

    # 启动时的初始化逻辑
    try:
        # 测试数据库连接
        with get_db_session() as session:
            session.execute(text("SELECT 1"))
        logger.info("数据库连接测试成功")
    except Exception as e:
        logger.error(f"数据库连接测试失败: {e}")
        raise

    yield

    # 关闭时的清理逻辑
    logger.info("关闭FastAPI应用")


# 创建FastAPI应用实例
app = FastAPI(
    title="量化交易平台数据采集API",
    description="提供股票数据、新闻数据和情感分析结果的RESTful API服务",
    version="1.0.0",
    docs_url="/docs" if settings.debug else None,
    redoc_url="/redoc" if settings.debug else None,
    lifespan=lifespan,
)

# 配置CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.allowed_origins,
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "DELETE"],
    allow_headers=["*"],
)


# 请求日志中间件
@app.middleware("http")
async def log_requests(request: Request, call_next):
    """记录请求日志"""
    start_time = time.time()

    # 记录请求信息
    logger.info(
        f"请求开始: {request.method} {request.url.path}",
        extra={
            "method": request.method,
            "path": request.url.path,
            "query_params": str(request.query_params),
            "client_ip": request.client.host if request.client else None,
        },
    )

    # 处理请求
    response = await call_next(request)

    # 计算处理时间
    process_time = time.time() - start_time

    # 记录响应信息
    logger.info(
        f"请求完成: {request.method} {request.url.path} - {response.status_code}",
        extra={
            "method": request.method,
            "path": request.url.path,
            "status_code": response.status_code,
            "process_time": round(process_time, 4),
        },
    )

    # 添加处理时间到响应头
    response.headers["X-Process-Time"] = str(process_time)

    return response


# 全局异常处理器
@app.exception_handler(BusinessError)
async def business_error_handler(request: Request, exc: BusinessError):
    """业务异常处理"""
    logger.warning(
        f"业务异常: {exc.message}",
        extra={
            "error_code": exc.error_code,
            "path": request.url.path,
            "method": request.method,
        },
    )

    return JSONResponse(
        status_code=status.HTTP_400_BAD_REQUEST,
        content={
            "error": "业务错误",
            "message": exc.message,
            "error_code": exc.error_code,
            "timestamp": datetime.utcnow().isoformat(),
        },
    )


@app.exception_handler(DataNotFoundError)
async def data_not_found_handler(request: Request, exc: DataNotFoundError):
    """数据未找到异常处理"""
    logger.info(
        f"数据未找到: {exc.message}",
        extra={
            "path": request.url.path,
            "method": request.method,
        },
    )

    return JSONResponse(
        status_code=status.HTTP_404_NOT_FOUND,
        content={
            "error": "数据未找到",
            "message": exc.message,
            "timestamp": datetime.utcnow().isoformat(),
        },
    )


@app.exception_handler(ValidationError)
async def validation_error_handler(request: Request, exc: ValidationError):
    """数据验证异常处理"""
    logger.warning(
        f"数据验证错误: {exc.message}",
        extra={
            "path": request.url.path,
            "method": request.method,
            "validation_errors": exc.details,
        },
    )

    return JSONResponse(
        status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
        content={
            "error": "数据验证错误",
            "message": exc.message,
            "details": exc.details,
            "timestamp": datetime.utcnow().isoformat(),
        },
    )


@app.exception_handler(RequestValidationError)
async def request_validation_error_handler(
    request: Request, exc: RequestValidationError
):
    """请求参数验证异常处理"""
    logger.warning(
        f"请求参数验证错误: {exc.errors()}",
        extra={
            "path": request.url.path,
            "method": request.method,
            "validation_errors": exc.errors(),
        },
    )

    return JSONResponse(
        status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
        content={
            "error": "请求参数错误",
            "message": "请求参数格式不正确",
            "details": exc.errors(),
            "timestamp": datetime.utcnow().isoformat(),
        },
    )


@app.exception_handler(SQLAlchemyError)
async def database_error_handler(request: Request, exc: SQLAlchemyError):
    """数据库异常处理"""
    logger.error(
        f"数据库错误: {exc!s}",
        extra={
            "path": request.url.path,
            "method": request.method,
            "error_type": type(exc).__name__,
        },
    )

    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content={
            "error": "数据库错误",
            "message": "数据库操作失败，请稍后重试",
            "timestamp": datetime.utcnow().isoformat(),
        },
    )


@app.exception_handler(Exception)
async def general_exception_handler(request: Request, exc: Exception):
    """通用异常处理"""
    logger.error(
        f"未处理的异常: {exc!s}",
        extra={
            "path": request.url.path,
            "method": request.method,
            "error_type": type(exc).__name__,
        },
        exc_info=True,
    )

    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content={
            "error": "服务器内部错误",
            "message": "服务器遇到了一个错误，请稍后重试",
            "timestamp": datetime.utcnow().isoformat(),
        },
    )


# 健康检查端点
@app.get("/health", tags=["健康检查"])
async def health_check():
    """健康检查端点"""
    return {
        "status": "healthy",
        "timestamp": datetime.utcnow().isoformat(),
        "version": "1.0.0",
    }


# 依赖注入函数
async def get_database():
    """获取数据库会话依赖"""
    async with get_db_session() as session:
        try:
            yield session
        except Exception:
            await session.rollback()
            raise
        finally:
            await session.close()


# 注册路由
app.include_router(api_router)


if __name__ == "__main__":
    import time
    from datetime import datetime

    import uvicorn

    # 开发环境启动配置
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=8000,
        reload=True,
        log_level="info",
    )
else:
    import time
    from datetime import datetime
