from collections.abc import Generator

from sqlalchemy.pool import StaticPool
from sqlmodel import Session, SQLModel, create_engine

from .settings import settings

# 创建数据库引擎
if settings.database_url.startswith("sqlite"):
    # SQLite配置
    engine = create_engine(
        settings.database_url,
        echo=settings.database_echo,
        connect_args={"check_same_thread": False},
        poolclass=StaticPool,
    )
else:
    # 其他数据库配置
    engine = create_engine(
        settings.database_url,
        echo=settings.database_echo,
    )


def create_db_and_tables() -> None:
    """创建数据库表"""
    SQLModel.metadata.create_all(engine)


def get_session() -> Generator[Session, None, None]:
    """获取数据库会话"""
    with Session(engine) as session:
        yield session


def get_db_session() -> Session:
    """获取数据库会话(同步版本)"""
    return Session(engine)
