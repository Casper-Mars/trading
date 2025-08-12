from collections.abc import Generator
from contextlib import contextmanager
from typing import Any

from sqlalchemy import event
from sqlalchemy.engine import Engine
from sqlalchemy.pool import QueuePool, StaticPool
from sqlmodel import Session, SQLModel, create_engine

from .settings import settings


def get_database_url() -> str:
    """根据配置动态构建数据库连接URL"""
    # 如果database_url明确指定了数据库类型，直接使用
    if settings.database_url.startswith(("sqlite", "postgresql", "mysql")):
        return settings.database_url

    # 如果没有明确指定，且MySQL配置完整，则构建MySQL URL
    if (
        settings.mysql_host and
        settings.mysql_user and
        settings.mysql_database
    ):
        # 构建MySQL连接URL
        password_part = f":{settings.mysql_password}" if settings.mysql_password else ""
        return (
            f"mysql+pymysql://{settings.mysql_user}{password_part}@"
            f"{settings.mysql_host}:{settings.mysql_port}/{settings.mysql_database}"
            f"?charset={settings.mysql_charset}"
        )

    # 默认返回配置的database_url
    return settings.database_url


# 创建数据库引擎
database_url = get_database_url()

if database_url.startswith("sqlite"):
    # SQLite配置
    engine = create_engine(
        database_url,
        echo=settings.database_echo,
        connect_args={"check_same_thread": False},
        poolclass=StaticPool,
    )
else:
    # MySQL和其他数据库配置
    engine = create_engine(
        database_url,
        echo=settings.database_echo,
        pool_size=settings.database_pool_size,
        max_overflow=settings.database_max_overflow,
        pool_timeout=settings.database_pool_timeout,
        pool_recycle=settings.database_pool_recycle,
        poolclass=QueuePool,
        pool_pre_ping=True,  # 连接前检查连接是否有效
    )


# 为MySQL设置严格模式
@event.listens_for(Engine, "connect")
def set_mysql_strict_mode(dbapi_connection: Any, connection_record: Any) -> None:
    """为MySQL连接设置严格模式"""
    if "mysql" in str(dbapi_connection):
        with dbapi_connection.cursor() as cursor:
            cursor.execute("SET sql_mode='STRICT_TRANS_TABLES'")


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


@contextmanager
def get_transaction() -> Generator[Session, None, None]:
    """获取事务会话，自动提交或回滚"""
    session = Session(engine)
    try:
        yield session
        session.commit()
    except Exception:
        session.rollback()
        raise
    finally:
        session.close()


def init_alembic() -> None:
    """初始化Alembic数据库迁移"""
    import contextlib

    from alembic import command
    from alembic.config import Config

    # 创建Alembic配置
    alembic_cfg = Config()
    alembic_cfg.set_main_option("script_location", "alembic")
    alembic_cfg.set_main_option("sqlalchemy.url", get_database_url())

    # 初始化迁移环境
    with contextlib.suppress(Exception):
        command.init(alembic_cfg, "alembic")
