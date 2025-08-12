"""数据仓库基类模块

提供数据访问层的基础功能和通用操作。
所有具体的数据仓库类都应该继承此基类。
"""

from abc import ABC
from typing import Generic, TypeVar

from loguru import logger
from sqlalchemy.ext.asyncio import AsyncSession

T = TypeVar("T")


class BaseRepository(ABC, Generic[T]):
    """数据仓库基类

    提供数据访问层的基础功能，包括数据库会话管理、
    通用的CRUD操作模板和事务处理。
    """

    def __init__(self, session: AsyncSession):
        """初始化数据仓库基类

        Args:
            session: 数据库会话
        """
        self.session = session
        logger.debug(f"{self.__class__.__name__} 初始化完成")

    async def begin_transaction(self) -> None:
        """开始事务"""
        if not self.session.in_transaction():
            await self.session.begin()
            logger.debug("事务开始")

    async def commit_transaction(self) -> None:
        """提交事务"""
        if self.session.in_transaction():
            await self.session.commit()
            logger.debug("事务提交")

    async def rollback_transaction(self) -> None:
        """回滚事务"""
        if self.session.in_transaction():
            await self.session.rollback()
            logger.debug("事务回滚")

    async def close_session(self) -> None:
        """关闭会话"""
        await self.session.close()
        logger.debug("数据库会话关闭")

    def get_session(self) -> AsyncSession:
        """获取数据库会话

        Returns:
            数据库会话
        """
        return self.session
