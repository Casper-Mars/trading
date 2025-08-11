"""策略注册器模块

提供策略的动态注册、管理和实例化功能。
"""

from dataclasses import dataclass
from typing import Any

from loguru import logger

from .base_strategy import BaseStrategy


@dataclass
class StrategyInfo:
    """策略信息"""

    name: str
    description: str
    strategy_class: type[BaseStrategy]
    required_params: list[str]
    default_params: dict[str, Any]
    category: str = "general"
    version: str = "1.0.0"
    author: str = "unknown"


class StrategyRegistry:
    """策略注册器

    负责策略的注册、管理和实例化。
    支持策略的动态注册和查询。
    """

    def __init__(self):
        self._strategies: dict[str, StrategyInfo] = {}
        self._categories: dict[str, list[str]] = {}
        logger.info("策略注册器初始化完成")

    def register(
        self,
        strategy_class: type[BaseStrategy],
        name: str | None = None,
        description: str | None = None,
        category: str = "general",
        version: str = "1.0.0",
        author: str = "unknown",
    ) -> bool:
        """注册策略

        Args:
            strategy_class: 策略类
            name: 策略名称，如果为None则使用类名
            description: 策略描述
            category: 策略分类
            version: 策略版本
            author: 策略作者

        Returns:
            bool: 注册是否成功
        """
        try:
            # 验证策略类
            if not issubclass(strategy_class, BaseStrategy):
                logger.error(
                    f"策略类 {strategy_class.__name__} 必须继承自 BaseStrategy"
                )
                return False

            # 获取策略信息
            strategy_name = name or strategy_class.__name__
            strategy_description = (
                description or strategy_class.get_strategy_description(strategy_class())
            )

            # 创建临时实例以获取参数信息
            temp_instance = strategy_class()
            required_params = temp_instance.get_required_params()

            # 获取默认参数
            default_params = {}
            if hasattr(strategy_class, "params"):
                default_params = {
                    param[0]: param[1] if len(param) > 1 else None
                    for param in strategy_class.params._getpairs()
                }

            # 创建策略信息
            strategy_info = StrategyInfo(
                name=strategy_name,
                description=strategy_description,
                strategy_class=strategy_class,
                required_params=required_params,
                default_params=default_params,
                category=category,
                version=version,
                author=author,
            )

            # 注册策略
            self._strategies[strategy_name] = strategy_info

            # 更新分类索引
            if category not in self._categories:
                self._categories[category] = []
            if strategy_name not in self._categories[category]:
                self._categories[category].append(strategy_name)

            logger.info(f"策略 {strategy_name} 注册成功, 分类: {category}")
            return True

        except Exception as e:
            logger.error(f"注册策略 {strategy_class.__name__} 失败: {e}")
            return False

    def unregister(self, strategy_name: str) -> bool:
        """注销策略

        Args:
            strategy_name: 策略名称

        Returns:
            bool: 注销是否成功
        """
        if strategy_name not in self._strategies:
            logger.warning(f"策略 {strategy_name} 未注册")
            return False

        strategy_info = self._strategies[strategy_name]

        # 从分类索引中移除
        if (
            strategy_info.category in self._categories
            and strategy_name in self._categories[strategy_info.category]
        ):
            self._categories[strategy_info.category].remove(strategy_name)

            # 如果分类为空，删除分类
            if not self._categories[strategy_info.category]:
                del self._categories[strategy_info.category]

        # 删除策略
        del self._strategies[strategy_name]

        logger.info(f"策略 {strategy_name} 注销成功")
        return True

    def get_strategy(self, strategy_name: str) -> StrategyInfo | None:
        """获取策略信息

        Args:
            strategy_name: 策略名称

        Returns:
            StrategyInfo: 策略信息，如果不存在则返回None
        """
        return self._strategies.get(strategy_name)

    def list_strategies(self, category: str | None = None) -> list[StrategyInfo]:
        """列出策略

        Args:
            category: 策略分类，如果为None则返回所有策略

        Returns:
            List[StrategyInfo]: 策略信息列表
        """
        if category is None:
            return list(self._strategies.values())

        if category not in self._categories:
            return []

        return [self._strategies[name] for name in self._categories[category]]

    def list_categories(self) -> list[str]:
        """列出所有策略分类

        Returns:
            List[str]: 分类列表
        """
        return list(self._categories.keys())

    def create_strategy(self, strategy_name: str, **params) -> BaseStrategy | None:
        """创建策略实例

        Args:
            strategy_name: 策略名称
            **params: 策略参数

        Returns:
            BaseStrategy: 策略实例，如果创建失败则返回None
        """
        strategy_info = self.get_strategy(strategy_name)
        if not strategy_info:
            logger.error(f"策略 {strategy_name} 未注册")
            return None

        try:
            # 合并默认参数和用户参数
            final_params = strategy_info.default_params.copy()
            final_params.update(params)

            # 验证必需参数
            missing_params = []
            for required_param in strategy_info.required_params:
                if required_param not in final_params:
                    missing_params.append(required_param)

            if missing_params:
                logger.error(f"策略 {strategy_name} 缺少必需参数: {missing_params}")
                return None

            # 创建策略实例
            strategy_class = strategy_info.strategy_class

            # 动态设置参数
            if hasattr(strategy_class, "params"):
                # 创建新的参数元组
                param_tuples = []
                for param_name, param_value in final_params.items():
                    param_tuples.append((param_name, param_value))

                # 创建带参数的策略类
                class ParameterizedStrategy(strategy_class):
                    params = tuple(param_tuples)

                strategy_instance = ParameterizedStrategy()
            else:
                strategy_instance = strategy_class()

            # 验证参数
            if not strategy_instance.validate_params():
                logger.error(f"策略 {strategy_name} 参数验证失败")
                return None

            logger.info(f"策略 {strategy_name} 实例创建成功")
            return strategy_instance

        except Exception as e:
            logger.error(f"创建策略 {strategy_name} 实例失败: {e}")
            return None

    def validate_strategy_params(
        self, strategy_name: str, params: dict[str, Any]
    ) -> tuple[bool, list[str]]:
        """验证策略参数

        Args:
            strategy_name: 策略名称
            params: 参数字典

        Returns:
            tuple[bool, List[str]]: (是否有效, 错误信息列表)
        """
        strategy_info = self.get_strategy(strategy_name)
        if not strategy_info:
            return False, [f"策略 {strategy_name} 未注册"]

        errors = []

        # 检查必需参数
        for required_param in strategy_info.required_params:
            if required_param not in params:
                errors.append(f"缺少必需参数: {required_param}")

        # 检查参数类型（基础检查）
        for param_name, param_value in params.items():
            if param_name in strategy_info.default_params:
                default_value = strategy_info.default_params[param_name]
                if default_value is not None and not isinstance(
                    param_value, type(default_value)
                ):
                    errors.append(
                        f"参数 {param_name} 类型错误, 期望 {type(default_value).__name__}, 实际 {type(param_value).__name__}"
                    )

        return len(errors) == 0, errors

    def get_strategy_count(self) -> int:
        """获取已注册策略数量

        Returns:
            int: 策略数量
        """
        return len(self._strategies)

    def clear(self):
        """清空所有注册的策略"""
        self._strategies.clear()
        self._categories.clear()
        logger.info("策略注册器已清空")

    def get_registry_info(self) -> dict[str, Any]:
        """获取注册器信息

        Returns:
            Dict[str, Any]: 注册器统计信息
        """
        return {
            "total_strategies": len(self._strategies),
            "categories": list(self._categories.keys()),
            "strategies_by_category": {
                cat: len(strategies) for cat, strategies in self._categories.items()
            },
            "strategy_names": list(self._strategies.keys()),
        }


# 全局策略注册器实例
strategy_registry = StrategyRegistry()


def register_strategy(strategy_class: type[BaseStrategy], **kwargs) -> bool:
    """注册策略的便捷函数

    Args:
        strategy_class: 策略类
        **kwargs: 其他注册参数

    Returns:
        bool: 注册是否成功
    """
    return strategy_registry.register(strategy_class, **kwargs)


def get_strategy(strategy_name: str) -> StrategyInfo | None:
    """获取策略信息的便捷函数

    Args:
        strategy_name: 策略名称

    Returns:
        StrategyInfo: 策略信息
    """
    return strategy_registry.get_strategy(strategy_name)


def create_strategy(strategy_name: str, **params) -> BaseStrategy | None:
    """创建策略实例的便捷函数

    Args:
        strategy_name: 策略名称
        **params: 策略参数

    Returns:
        BaseStrategy: 策略实例
    """
    return strategy_registry.create_strategy(strategy_name, **params)
