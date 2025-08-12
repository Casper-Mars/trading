"""NLP处理编排器模块

实现NLP处理业务编排器，协调定时调度器→NLP服务→文本处理器→FinBERT模型→数据库存储的完整流程。
"""

import asyncio
from typing import Any

from pydantic import BaseModel

from biz.base_orchestrator import (
    BaseOrchestrator,
    OrchestrationContext,
    OrchestrationError,
)
from models.database import NewsData
from repositories.news_repo import NewsRepository
from services.nlp_service import NLPService
from utils.logger import get_logger

logger = get_logger(__name__)


class NLPProcessingRequest(BaseModel):
    """NLP处理请求模型"""

    batch_size: int = 50  # 批量处理大小
    max_concurrent: int = 5  # 最大并发数
    skip_processed: bool = True  # 是否跳过已处理的新闻
    force_reprocess: bool = False  # 是否强制重新处理
    target_news_ids: list[int] | None = None  # 指定处理的新闻ID列表


class NLPProcessingResult(BaseModel):
    """NLP处理结果模型"""

    total_processed: int
    successful_count: int
    failed_count: int
    processing_time: float
    error_details: list[dict[str, Any]] = []
    performance_stats: dict[str, Any] = {}


class NLPProcessingOrchestrator(BaseOrchestrator):
    """NLP处理编排器

    协调NLP处理的完整流程：
    1. 获取待处理新闻数据
    2. 批量NLP处理（情感分析、实体提取、关键词提取等）
    3. 更新处理结果到数据库
    4. 监控处理进度和性能统计
    """

    def __init__(self, nlp_service: NLPService, news_repo: NewsRepository):
        """初始化NLP处理编排器

        Args:
            nlp_service: NLP处理服务
            news_repo: 新闻数据仓库
        """
        super().__init__()
        self.nlp_service = nlp_service
        self.news_repo = news_repo

        # 设置NLP服务的新闻仓库依赖
        self.nlp_service.set_news_repository(news_repo)

    async def _pre_check(
        self, request: NLPProcessingRequest, context: OrchestrationContext
    ) -> None:
        """前置检查

        Args:
            request: NLP处理请求
            context: 编排上下文

        Raises:
            OrchestrationError: 前置检查失败
        """
        try:
            # 检查NLP服务是否就绪
            if not await self.nlp_service.is_ready():
                raise OrchestrationError("NLP service is not ready")

            # 检查数据库连接
            if not self.news_repo.session:
                raise OrchestrationError("Database connection is not available")

            # 验证请求参数
            if request.batch_size <= 0:
                raise OrchestrationError("Batch size must be positive")

            if request.max_concurrent <= 0:
                raise OrchestrationError("Max concurrent must be positive")

            logger.info(
                f"Pre-check completed for request_id: {context.request_id}, "
                f"batch_size: {request.batch_size}, max_concurrent: {request.max_concurrent}"
            )

        except Exception as e:
            logger.error(
                f"Pre-check failed for request_id: {context.request_id}, error: {e}"
            )
            raise OrchestrationError(f"Pre-check failed: {e}") from e

    async def _call_services(
        self, request: NLPProcessingRequest, context: OrchestrationContext
    ) -> dict[str, Any]:
        """调用服务

        Args:
            request: NLP处理请求
            context: 编排上下文

        Returns:
            服务调用结果字典

        Raises:
            OrchestrationError: 服务调用失败
        """
        try:
            # 1. 获取待处理新闻数据
            news_data = await self._fetch_news_data(request, context)
            context.intermediate_results["news_count"] = len(news_data)

            if not news_data:
                logger.info(
                    f"No news data to process for request_id: {context.request_id}"
                )
                return {
                    "news_data": [],
                    "processing_results": [],
                    "performance_stats": {},
                }

            # 2. 批量处理新闻数据
            processing_results = await self._process_news_batch(
                news_data, request, context
            )
            context.intermediate_results["processed_count"] = len(processing_results)

            # 3. 获取性能统计
            performance_stats = await self.nlp_service.get_performance_stats()

            logger.info(
                f"Service calls completed for request_id: {context.request_id}, "
                f"processed: {len(processing_results)}/{len(news_data)}"
            )

            return {
                "news_data": news_data,
                "processing_results": processing_results,
                "performance_stats": performance_stats,
            }

        except Exception as e:
            logger.error(
                f"Service calls failed for request_id: {context.request_id}, error: {e}"
            )
            raise OrchestrationError(f"Service calls failed: {e}") from e

    async def _aggregate_results(
        self, service_results: dict[str, Any], context: OrchestrationContext
    ) -> NLPProcessingResult:
        """聚合结果

        Args:
            service_results: 服务调用结果
            context: 编排上下文

        Returns:
            聚合后的NLP处理结果

        Raises:
            OrchestrationError: 结果聚合失败
        """
        try:
            news_data = service_results["news_data"]
            processing_results = service_results["processing_results"]
            performance_stats = service_results["performance_stats"]

            # 统计处理结果
            total_processed = len(news_data)
            successful_count = len([r for r in processing_results if r is not None])
            failed_count = total_processed - successful_count

            # 计算处理时间
            start_time = context.intermediate_results.get("start_time", 0)
            processing_time = asyncio.get_event_loop().time() - start_time

            # 收集错误详情
            error_details = []
            for i, result in enumerate(processing_results):
                if result is None and i < len(news_data):
                    error_details.append(
                        {
                            "news_id": news_data[i].id,
                            "title": news_data[i].title[:50] + "..."
                            if len(news_data[i].title) > 50
                            else news_data[i].title,
                            "error": "Processing failed",
                        }
                    )

            result = NLPProcessingResult(
                total_processed=total_processed,
                successful_count=successful_count,
                failed_count=failed_count,
                processing_time=processing_time,
                error_details=error_details,
                performance_stats=performance_stats,
            )

            logger.info(
                f"Results aggregated for request_id: {context.request_id}, "
                f"success: {successful_count}/{total_processed}, time: {processing_time:.3f}s"
            )

            return result

        except Exception as e:
            logger.error(
                f"Result aggregation failed for request_id: {context.request_id}, error: {e}"
            )
            raise OrchestrationError(f"Result aggregation failed: {e}") from e

    async def _fetch_news_data(
        self, request: NLPProcessingRequest, context: OrchestrationContext
    ) -> list[NewsData]:
        """获取待处理新闻数据

        Args:
            request: NLP处理请求
            context: 编排上下文

        Returns:
            待处理的新闻数据列表
        """
        try:
            if request.target_news_ids:
                # 处理指定的新闻ID
                news_data = []
                for news_id in request.target_news_ids:
                    news = await self.news_repo.get_by_id(news_id)
                    if news:
                        news_data.append(news)
                logger.info(
                    f"Fetched {len(news_data)} specific news items for processing"
                )
            else:
                # 获取未处理的新闻
                if request.skip_processed and not request.force_reprocess:
                    news_data = await self.news_repo.get_unprocessed_news(
                        limit=request.batch_size
                    )
                else:
                    # 获取所有新闻（用于重新处理）
                    news_data = await self.news_repo.get_all(limit=request.batch_size)

                logger.info(f"Fetched {len(news_data)} news items for processing")

            # 记录获取的新闻信息到上下文
            context.intermediate_results["fetched_news_ids"] = [
                news.id for news in news_data
            ]

            return news_data

        except Exception as e:
            logger.error(f"Failed to fetch news data: {e}")
            raise

    async def _process_news_batch(
        self,
        news_data: list[NewsData],
        request: NLPProcessingRequest,
        context: OrchestrationContext,
    ) -> list[dict[str, Any] | None]:
        """批量处理新闻数据

        Args:
            news_data: 新闻数据列表
            request: NLP处理请求
            context: 编排上下文

        Returns:
            处理结果列表
        """
        try:
            # 使用NLP服务的批量处理功能
            processing_results = await self.nlp_service.batch_process_news(
                news_data, max_concurrent=request.max_concurrent
            )

            # 记录处理状态到上下文
            successful_ids = []
            failed_ids = []

            for i, result in enumerate(processing_results):
                if result is not None:
                    successful_ids.append(news_data[i].id)
                else:
                    failed_ids.append(news_data[i].id)

            context.intermediate_results["successful_news_ids"] = successful_ids
            context.intermediate_results["failed_news_ids"] = failed_ids

            logger.info(
                f"Batch processing completed: {len(successful_ids)} successful, {len(failed_ids)} failed"
            )

            return processing_results

        except Exception as e:
            logger.error(f"Batch processing failed: {e}")
            raise

    async def _cleanup_context(self, context: OrchestrationContext) -> None:
        """清理编排上下文

        Args:
            context: 编排上下文
        """
        try:
            # 清理临时数据
            if "temp_data" in context.intermediate_results:
                del context.intermediate_results["temp_data"]

            logger.debug(
                f"Context cleanup completed for request_id: {context.request_id}"
            )

        except Exception as e:
            logger.warning(
                f"Context cleanup failed for request_id: {context.request_id}, error: {e}"
            )

    async def _execute_rollback_action(
        self, action: dict[str, Any], context: OrchestrationContext
    ) -> None:
        """执行回滚操作

        Args:
            action: 回滚操作
            context: 编排上下文
        """
        action_type = action.get("type")

        try:
            if action_type == "revert_news_processing":
                # 回滚新闻处理状态
                news_ids = action.get("news_ids", [])
                for news_id in news_ids:
                    await self.news_repo.update_processing_status(news_id, False)
                logger.info(
                    f"Reverted processing status for {len(news_ids)} news items"
                )

            elif action_type == "cleanup_temp_data":
                # 清理临时数据
                temp_keys = action.get("temp_keys", [])
                for key in temp_keys:
                    context.intermediate_results.pop(key, None)
                logger.info(f"Cleaned up {len(temp_keys)} temporary data keys")

            else:
                logger.warning(f"Unknown rollback action type: {action_type}")

        except Exception as e:
            logger.error(f"Rollback action failed: {action_type}, error: {e}")
            raise

    def _mark_step_completed(
        self, step_name: str, context: OrchestrationContext
    ) -> None:
        """标记步骤完成

        Args:
            step_name: 步骤名称
            context: 编排上下文
        """
        completed_steps = context.intermediate_results.get("completed_steps", [])
        completed_steps.append(step_name)
        context.intermediate_results["completed_steps"] = completed_steps

        logger.debug(
            f"Step completed: {step_name} for request_id: {context.request_id}"
        )

    async def process_news_continuously(
        self, request: NLPProcessingRequest, interval_seconds: int = 300
    ) -> None:
        """持续处理新闻数据

        Args:
            request: NLP处理请求
            interval_seconds: 处理间隔（秒）
        """
        logger.info(
            f"Starting continuous news processing with interval: {interval_seconds}s"
        )

        while True:
            try:
                # 创建新的上下文
                context = OrchestrationContext(
                    request_id=f"continuous_{asyncio.get_event_loop().time()}",
                    user_id="system",
                )

                # 执行处理
                result = await self.execute(request, context)

                if result.success:
                    logger.info(
                        f"Continuous processing completed: {result.result.successful_count} processed, "
                        f"time: {result.execution_time:.3f}s"
                    )
                else:
                    logger.error(f"Continuous processing failed: {result.error}")

                # 等待下一次处理
                await asyncio.sleep(interval_seconds)

            except asyncio.CancelledError:
                logger.info("Continuous processing cancelled")
                break
            except Exception as e:
                logger.error(f"Continuous processing error: {e}")
                # 发生错误时等待较短时间后重试
                await asyncio.sleep(min(interval_seconds, 60))
