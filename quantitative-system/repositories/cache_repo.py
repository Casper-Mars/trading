"""Redis缓存数据访问层"""

import json
import pickle
from datetime import datetime
from typing import Any

import redis
from loguru import logger

from config.settings import settings
from models.enums import CacheType
from utils.exceptions import CacheError


class CacheRepo:
    """Redis缓存仓库

    统一缓存操作接口，支持多种数据类型和过期策略
    """

    def __init__(self):
        """初始化Redis连接"""
        try:
            self.redis_client = redis.Redis(
                host=settings.REDIS_HOST,
                port=settings.REDIS_PORT,
                db=settings.REDIS_DB,
                password=settings.REDIS_PASSWORD,
                decode_responses=False,  # 保持二进制模式以支持pickle
                socket_connect_timeout=5,
                socket_timeout=5,
                retry_on_timeout=True,
            )
            # 测试连接
            self.redis_client.ping()
            logger.info("Redis连接成功")
        except Exception as e:
            logger.error(f"Redis连接失败: {e}")
            raise CacheError(f"Redis连接失败: {e}") from e

    def _build_key(self, cache_type: CacheType, key: str) -> str:
        """构建标准化缓存键

        Args:
            cache_type: 缓存类型
            key: 业务键

        Returns:
            标准化的缓存键
        """
        prefix_map = {
            CacheType.MARKET_DATA: "market",
            CacheType.USER_SESSION: "session",
            CacheType.API_RESPONSE: "api",
            CacheType.CALCULATION_RESULT: "calc",
            CacheType.TEMPORARY: "temp",
        }
        prefix = prefix_map.get(cache_type, "unknown")
        return f"quant:{prefix}:{key}"

    def _get_default_ttl(self, cache_type: CacheType) -> int:
        """获取默认过期时间(秒)

        Args:
            cache_type: 缓存类型

        Returns:
            过期时间(秒)
        """
        ttl_map = {
            CacheType.MARKET_DATA: 300,  # 5分钟
            CacheType.USER_SESSION: 3600,  # 1小时
            CacheType.API_RESPONSE: 600,  # 10分钟
            CacheType.CALCULATION_RESULT: 1800,  # 30分钟
            CacheType.TEMPORARY: 60,  # 1分钟
        }
        return ttl_map.get(cache_type, 300)

    def set(
        self,
        cache_type: CacheType,
        key: str,
        value: Any,
        ttl: int | None = None,
        serialize_method: str = "json",
    ) -> bool:
        """设置缓存

        Args:
            cache_type: 缓存类型
            key: 缓存键
            value: 缓存值
            ttl: 过期时间(秒)，None使用默认值
            serialize_method: 序列化方法，json或pickle

        Returns:
            是否设置成功
        """
        try:
            cache_key = self._build_key(cache_type, key)

            # 序列化数据
            if serialize_method == "json":
                serialized_value = json.dumps(value, ensure_ascii=False, default=str)
            elif serialize_method == "pickle":
                serialized_value = pickle.dumps(value)
            else:
                raise ValueError(f"不支持的序列化方法: {serialize_method}")

            # 设置过期时间
            if ttl is None:
                ttl = self._get_default_ttl(cache_type)

            # 存储到Redis
            result = self.redis_client.setex(cache_key, ttl, serialized_value)

            logger.debug(f"缓存设置成功: {cache_key}, TTL: {ttl}s")
            return bool(result)

        except Exception as e:
            logger.error(f"设置缓存失败: {cache_key}, 错误: {e}")
            return False

    def get(
        self, cache_type: CacheType, key: str, serialize_method: str = "json"
    ) -> Any | None:
        """获取缓存

        Args:
            cache_type: 缓存类型
            key: 缓存键
            serialize_method: 序列化方法，json或pickle

        Returns:
            缓存值，不存在返回None
        """
        try:
            cache_key = self._build_key(cache_type, key)
            serialized_value = self.redis_client.get(cache_key)

            if serialized_value is None:
                return None

            # 反序列化数据
            if serialize_method == "json":
                value = json.loads(serialized_value.decode("utf-8"))
            elif serialize_method == "pickle":
                value = pickle.loads(serialized_value)  # noqa: S301
            else:
                raise ValueError(f"不支持的序列化方法: {serialize_method}")

            logger.debug(f"缓存命中: {cache_key}")
            return value

        except Exception as e:
            logger.error(f"获取缓存失败: {cache_key}, 错误: {e}")
            return None

    def delete(self, cache_type: CacheType, key: str) -> bool:
        """删除缓存

        Args:
            cache_type: 缓存类型
            key: 缓存键

        Returns:
            是否删除成功
        """
        try:
            cache_key = self._build_key(cache_type, key)
            result = self.redis_client.delete(cache_key)

            logger.debug(f"缓存删除: {cache_key}, 结果: {bool(result)}")
            return bool(result)

        except Exception as e:
            logger.error(f"删除缓存失败: {cache_key}, 错误: {e}")
            return False

    def exists(self, cache_type: CacheType, key: str) -> bool:
        """检查缓存是否存在

        Args:
            cache_type: 缓存类型
            key: 缓存键

        Returns:
            是否存在
        """
        try:
            cache_key = self._build_key(cache_type, key)
            result = self.redis_client.exists(cache_key)
            return bool(result)

        except Exception as e:
            logger.error(f"检查缓存存在性失败: {cache_key}, 错误: {e}")
            return False

    def get_ttl(self, cache_type: CacheType, key: str) -> int:
        """获取缓存剩余过期时间

        Args:
            cache_type: 缓存类型
            key: 缓存键

        Returns:
            剩余过期时间(秒)，-1表示永不过期，-2表示不存在
        """
        try:
            cache_key = self._build_key(cache_type, key)
            return self.redis_client.ttl(cache_key)

        except Exception as e:
            logger.error(f"获取缓存TTL失败: {cache_key}, 错误: {e}")
            return -2

    def extend_ttl(self, cache_type: CacheType, key: str, ttl: int) -> bool:
        """延长缓存过期时间

        Args:
            cache_type: 缓存类型
            key: 缓存键
            ttl: 新的过期时间(秒)

        Returns:
            是否设置成功
        """
        try:
            cache_key = self._build_key(cache_type, key)
            result = self.redis_client.expire(cache_key, ttl)

            logger.debug(f"缓存TTL更新: {cache_key}, 新TTL: {ttl}s")
            return bool(result)

        except Exception as e:
            logger.error(f"延长缓存TTL失败: {cache_key}, 错误: {e}")
            return False

    def clear_by_pattern(self, cache_type: CacheType, pattern: str = "*") -> int:
        """按模式清理缓存

        Args:
            cache_type: 缓存类型
            pattern: 匹配模式

        Returns:
            删除的键数量
        """
        try:
            cache_pattern = self._build_key(cache_type, pattern)
            keys = self.redis_client.keys(cache_pattern)

            if not keys:
                return 0

            deleted_count = self.redis_client.delete(*keys)
            logger.info(f"批量删除缓存: 模式={cache_pattern}, 删除数量={deleted_count}")
            return deleted_count

        except Exception as e:
            logger.error(f"批量删除缓存失败: 模式={cache_pattern}, 错误: {e}")
            return 0

    def get_cache_info(self, cache_type: CacheType) -> dict[str, Any]:
        """获取缓存统计信息

        Args:
            cache_type: 缓存类型

        Returns:
            缓存统计信息
        """
        try:
            prefix_map = {
                CacheType.MARKET_DATA: "market",
                CacheType.USER_SESSION: "session",
                CacheType.API_RESPONSE: "api",
                CacheType.CALCULATION_RESULT: "calc",
                CacheType.TEMPORARY: "temp",
            }
            prefix = prefix_map.get(cache_type, "unknown")
            pattern = f"quant:{prefix}:*"

            keys = self.redis_client.keys(pattern)
            total_keys = len(keys)

            # 统计不同TTL范围的键数量
            ttl_stats = {"永不过期": 0, "1小时内": 0, "1天内": 0, "其他": 0}

            for key in keys[:100]:  # 限制检查数量避免性能问题
                ttl = self.redis_client.ttl(key)
                if ttl == -1:
                    ttl_stats["永不过期"] += 1
                elif 0 <= ttl <= 3600:
                    ttl_stats["1小时内"] += 1
                elif 3600 < ttl <= 86400:
                    ttl_stats["1天内"] += 1
                else:
                    ttl_stats["其他"] += 1

            return {
                "cache_type": cache_type.name,
                "total_keys": total_keys,
                "ttl_distribution": ttl_stats,
                "sample_size": min(100, total_keys),
            }

        except Exception as e:
            logger.error(f"获取缓存信息失败: {cache_type}, 错误: {e}")
            return {"error": str(e)}

    def health_check(self) -> dict[str, Any]:
        """Redis健康检查

        Returns:
            健康状态信息
        """
        try:
            # 测试连接
            start_time = datetime.now()
            self.redis_client.ping()
            ping_time = (datetime.now() - start_time).total_seconds() * 1000

            # 获取Redis信息
            info = self.redis_client.info()

            return {
                "status": "healthy",
                "ping_ms": round(ping_time, 2),
                "redis_version": info.get("redis_version"),
                "used_memory_human": info.get("used_memory_human"),
                "connected_clients": info.get("connected_clients"),
                "total_commands_processed": info.get("total_commands_processed"),
                "keyspace_hits": info.get("keyspace_hits", 0),
                "keyspace_misses": info.get("keyspace_misses", 0),
            }

        except Exception as e:
            logger.error(f"Redis健康检查失败: {e}")
            return {"status": "unhealthy", "error": str(e)}


# 全局缓存实例
cache_repo = CacheRepo()
