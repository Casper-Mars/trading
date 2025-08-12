"""API路由模块

统一管理所有API路由的导入和注册
"""

from fastapi import APIRouter

from .news import router as news_router
from .stocks import router as stocks_router

# 创建主路由器
api_router = APIRouter(prefix="/api/v1")

# 注册子路由
api_router.include_router(stocks_router)
api_router.include_router(news_router)

__all__ = ["api_router"]
