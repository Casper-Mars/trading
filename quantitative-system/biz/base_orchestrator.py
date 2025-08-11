"""基础编排器模块

提供编排器基类和通用的编排流程管理功能。
"""

import asyncio
from abc import ABC, abstractmethod
from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager
from typing import Any, TypeVar

from pydantic import BaseModel

from utils.exceptions import OrchestrationError
from utils.logger import get_logger

logger = get_logger(__name__)

T = TypeVar('T', bound=BaseModel)
R = TypeVar('R', bound=BaseModel)


class OrchestrationContext(BaseModel):
    """编排上下文模型

    用于在编排流程中传递数据和状态信息。
    """

    request_id: str
    user_id: str | None = None
    session_data: dict[str, Any] = {}
    intermediate_results: dict[str, Any] = {}
    error_context: dict[str, Any] = {}
    rollback_actions: list[dict[str, Any]] = []


class OrchestrationResult(BaseModel):
    """编排结果模型"""

    success: bool
    result: Any | None = None
    error: str | None = None
    context: OrchestrationContext
    execution_time: float
    steps_completed: list[str] = []
    steps_failed: list[str] = []


class BaseOrchestrator(ABC):
    """基础编排器

    提供编排流程的通用模式和基础功能:
    - 前置检查 → 服务调用 → 结果聚合 → 异常回滚
    - 统一的错误处理、日志记录、事务管理
    - 编排上下文管理和跨服务数据传递
    """

    def __init__(self):
        """初始化基础编排器"""
        self.name = self.__class__.__name__
        logger.info(f"Initializing orchestrator: {self.name}")

    async def execute(self, request: T, context: OrchestrationContext) -> OrchestrationResult:
        """执行编排流程

        Args:
            request: 编排请求
            context: 编排上下文

        Returns:
            编排结果
        """
        start_time = asyncio.get_event_loop().time()

        logger.info(f"Starting orchestration: {self.name}, request_id: {context.request_id}")

        try:
            async with self._manage_context(context) as managed_context:
                # 执行编排流程
                result = await self._execute_workflow(request, managed_context)

                execution_time = asyncio.get_event_loop().time() - start_time

                logger.info(
                    f"Orchestration completed successfully: {self.name}, "
                    f"request_id: {context.request_id}, time: {execution_time:.3f}s"
                )

                return OrchestrationResult(
                    success=True,
                    result=result,
                    context=managed_context,
                    execution_time=execution_time,
                    steps_completed=managed_context.intermediate_results.get('completed_steps', [])
                )

        except Exception as e:
            execution_time = asyncio.get_event_loop().time() - start_time

            logger.error(
                f"Orchestration failed: {self.name}, "
                f"request_id: {context.request_id}, error: {e!s}, time: {execution_time:.3f}s"
            )

            # 执行回滚
            await self._rollback(context)

            return OrchestrationResult(
                success=False,
                error=str(e),
                context=context,
                execution_time=execution_time,
                steps_failed=context.intermediate_results.get('failed_steps', [])
            )

    @asynccontextmanager
    async def _manage_context(self, context: OrchestrationContext) -> AsyncGenerator[OrchestrationContext, None]:
        """管理编排上下文

        Args:
            context: 编排上下文

        Yields:
            管理的上下文
        """
        try:
            # 初始化上下文
            context.intermediate_results['completed_steps'] = []
            context.intermediate_results['failed_steps'] = []
            context.intermediate_results['start_time'] = asyncio.get_event_loop().time()

            yield context

        except Exception as e:
            # 记录错误上下文
            context.error_context['exception'] = str(e)
            context.error_context['exception_type'] = type(e).__name__
            raise
        finally:
            # 清理资源
            await self._cleanup_context(context)

    async def _execute_workflow(self, request: T, context: OrchestrationContext) -> R:
        """执行编排工作流

        Args:
            request: 编排请求
            context: 编排上下文

        Returns:
            编排结果
        """
        # 1. 前置检查
        await self._pre_check(request, context)
        self._mark_step_completed('pre_check', context)

        # 2. 服务调用
        service_results = await self._call_services(request, context)
        self._mark_step_completed('service_calls', context)

        # 3. 结果聚合
        result = await self._aggregate_results(service_results, context)
        self._mark_step_completed('result_aggregation', context)

        return result

    @abstractmethod
    async def _pre_check(self, request: T, context: OrchestrationContext) -> None:
        """前置检查

        Args:
            request: 编排请求
            context: 编排上下文

        Raises:
            OrchestrationError: 前置检查失败
        """
        pass

    @abstractmethod
    async def _call_services(self, request: T, context: OrchestrationContext) -> dict[str, Any]:
        """调用服务

        Args:
            request: 编排请求
            context: 编排上下文

        Returns:
            服务调用结果字典

        Raises:
            OrchestrationError: 服务调用失败
        """
        pass

    @abstractmethod
    async def _aggregate_results(self, service_results: dict[str, Any], context: OrchestrationContext) -> R:
        """聚合结果

        Args:
            service_results: 服务调用结果
            context: 编排上下文

        Returns:
            聚合后的结果

        Raises:
            OrchestrationError: 结果聚合失败
        """
        pass

    async def _rollback(self, context: OrchestrationContext) -> None:
        """执行回滚操作

        Args:
            context: 编排上下文
        """
        if not context.rollback_actions:
            logger.info(f"No rollback actions for request_id: {context.request_id}")
            return

        logger.info(f"Starting rollback for request_id: {context.request_id}, actions: {len(context.rollback_actions)}")

        # 按逆序执行回滚操作
        for action in reversed(context.rollback_actions):
            try:
                await self._execute_rollback_action(action, context)
                logger.info(f"Rollback action completed: {action.get('type', 'unknown')}")
            except Exception as e:
                logger.error(f"Rollback action failed: {action.get('type', 'unknown')}, error: {e!s}")
                # 继续执行其他回滚操作

        logger.info(f"Rollback completed for request_id: {context.request_id}")

    async def _execute_rollback_action(self, action: dict[str, Any], context: OrchestrationContext) -> None:
        """执行单个回滚操作

        Args:
            action: 回滚操作定义
            context: 编排上下文
        """
        action_type = action.get('type')

        if action_type == 'delete_data':
            await self._rollback_delete_data(action, context)
        elif action_type == 'restore_state':
            await self._rollback_restore_state(action, context)
        elif action_type == 'cleanup_resources':
            await self._rollback_cleanup_resources(action, context)
        else:
            logger.warning(f"Unknown rollback action type: {action_type}")

    async def _rollback_delete_data(self, action: dict[str, Any], context: OrchestrationContext) -> None:
        """回滚数据删除操作

        Args:
            action: 回滚操作定义
            context: 编排上下文
        """
        # 子类可以重写此方法实现具体的数据删除回滚逻辑
        logger.info(f"Executing delete_data rollback: {action}")

    async def _rollback_restore_state(self, action: dict[str, Any], context: OrchestrationContext) -> None:
        """回滚状态恢复操作

        Args:
            action: 回滚操作定义
            context: 编排上下文
        """
        # 子类可以重写此方法实现具体的状态恢复回滚逻辑
        logger.info(f"Executing restore_state rollback: {action}")

    async def _rollback_cleanup_resources(self, action: dict[str, Any], context: OrchestrationContext) -> None:
        """回滚资源清理操作

        Args:
            action: 回滚操作定义
            context: 编排上下文
        """
        # 子类可以重写此方法实现具体的资源清理回滚逻辑
        logger.info(f"Executing cleanup_resources rollback: {action}")

    async def _cleanup_context(self, context: OrchestrationContext) -> None:
        """清理编排上下文

        Args:
            context: 编排上下文
        """
        # 清理临时数据
        if 'temp_data' in context.session_data:
            del context.session_data['temp_data']

        logger.debug(f"Context cleaned up for request_id: {context.request_id}")

    def _mark_step_completed(self, step_name: str, context: OrchestrationContext) -> None:
        """标记步骤完成

        Args:
            step_name: 步骤名称
            context: 编排上下文
        """
        completed_steps = context.intermediate_results.get('completed_steps', [])
        completed_steps.append(step_name)
        context.intermediate_results['completed_steps'] = completed_steps

        logger.debug(f"Step completed: {step_name}, request_id: {context.request_id}")

    def _mark_step_failed(self, step_name: str, context: OrchestrationContext, error: str) -> None:
        """标记步骤失败

        Args:
            step_name: 步骤名称
            context: 编排上下文
            error: 错误信息
        """
        failed_steps = context.intermediate_results.get('failed_steps', [])
        failed_steps.append(step_name)
        context.intermediate_results['failed_steps'] = failed_steps

        context.error_context[f'{step_name}_error'] = error

        logger.error(f"Step failed: {step_name}, error: {error}, request_id: {context.request_id}")

    def _add_rollback_action(self, action_type: str, action_data: dict[str, Any], context: OrchestrationContext) -> None:
        """添加回滚操作

        Args:
            action_type: 回滚操作类型
            action_data: 回滚操作数据
            context: 编排上下文
        """
        rollback_action = {
            'type': action_type,
            'data': action_data,
            'timestamp': asyncio.get_event_loop().time()
        }

        context.rollback_actions.append(rollback_action)

        logger.debug(f"Rollback action added: {action_type}, request_id: {context.request_id}")

    async def _safe_service_call(self, service_name: str, service_call, context: OrchestrationContext) -> Any:
        """安全的服务调用

        Args:
            service_name: 服务名称
            service_call: 服务调用函数
            context: 编排上下文

        Returns:
            服务调用结果

        Raises:
            OrchestrationError: 服务调用失败
        """
        try:
            logger.debug(f"Calling service: {service_name}, request_id: {context.request_id}")

            result = await service_call()

            logger.debug(f"Service call completed: {service_name}, request_id: {context.request_id}")

            return result

        except Exception as e:
            error_msg = f"Service call failed: {service_name}, error: {e!s}"
            logger.error(f"{error_msg}, request_id: {context.request_id}")

            self._mark_step_failed(f'service_{service_name}', context, str(e))

            raise OrchestrationError(error_msg) from e

    def _validate_request(self, request: T, context: OrchestrationContext) -> None:
        """验证请求参数

        Args:
            request: 编排请求
            context: 编排上下文

        Raises:
            OrchestrationError: 请求验证失败
        """
        if not isinstance(request, BaseModel):
            raise OrchestrationError("Request must be a Pydantic model")

        if not context.request_id:
            raise OrchestrationError("Request ID is required")

        logger.debug(f"Request validation passed, request_id: {context.request_id}")

    def _get_context_data(self, key: str, context: OrchestrationContext, default: Any = None) -> Any:
        """获取上下文数据

        Args:
            key: 数据键
            context: 编排上下文
            default: 默认值

        Returns:
            上下文数据
        """
        return context.intermediate_results.get(key, default)

    def _set_context_data(self, key: str, value: Any, context: OrchestrationContext) -> None:
        """设置上下文数据

        Args:
            key: 数据键
            value: 数据值
            context: 编排上下文
        """
        context.intermediate_results[key] = value

        logger.debug(f"Context data set: {key}, request_id: {context.request_id}")
