"""NLP处理服务

提供自然语言处理功能，包括情感分析、实体识别和文本处理。
集成FinBERT模型进行金融文本情感分析。
"""

import asyncio

from loguru import logger

from utils.finbert_client import FinBERTClient
from utils.text_processor import TextProcessor


class NLPService:
    """NLP处理服务

    负责提供自然语言处理功能，包括：
    - 金融文本情感分析（基于FinBERT）
    - 文本预处理和清洗
    - 批量文本处理
    - 性能监控和优化
    """

    def __init__(
        self,
        model_name: str = "ProsusAI/finbert",
        cache_dir: str | None = None,
        device: str | None = None,
        max_length: int = 512,
    ):
        """初始化NLP服务

        Args:
            model_name: FinBERT模型名称
            cache_dir: 模型缓存目录
            device: 计算设备
            max_length: 最大文本长度
        """
        self.model_name = model_name
        self.max_length = max_length

        # 初始化组件
        self.text_processor = TextProcessor()
        self.finbert_client = FinBERTClient(
            model_name=model_name,
            cache_dir=cache_dir,
            device=device,
            max_length=max_length,
        )

        # 服务状态
        self._is_initialized = False
        self._initialization_lock = asyncio.Lock()

        logger.info("NLP服务初始化完成")

    async def initialize(self) -> bool:
        """异步初始化服务

        Returns:
            是否初始化成功
        """
        async with self._initialization_lock:
            if self._is_initialized:
                return True

            try:
                logger.info("开始初始化NLP服务")

                # 在线程池中加载模型（避免阻塞事件循环）
                loop = asyncio.get_event_loop()
                success = await loop.run_in_executor(
                    None, self.finbert_client.load_model
                )

                if success:
                    # 优化推理性能
                    await loop.run_in_executor(
                        None, self.finbert_client.optimize_inference
                    )

                    self._is_initialized = True
                    logger.info("NLP服务初始化成功")
                    return True
                else:
                    logger.error("NLP服务初始化失败：模型加载失败")
                    return False

            except Exception as e:
                logger.error(f"NLP服务初始化异常: {e}")
                return False

    def is_ready(self) -> bool:
        """检查服务是否就绪

        Returns:
            服务是否已初始化并就绪
        """
        return self._is_initialized and self.finbert_client.is_ready()

    async def analyze_sentiment(self, text: str) -> dict[str, str | float] | None:
        """分析文本情感

        Args:
            text: 输入文本

        Returns:
            情感分析结果
        """
        if not self.is_ready():
            logger.error("NLP服务未就绪，请先调用initialize()")
            return None

        try:
            # 在线程池中执行推理（避免阻塞事件循环）
            loop = asyncio.get_event_loop()
            result = await loop.run_in_executor(
                None, self.finbert_client.predict_sentiment, text
            )

            if result:
                logger.debug(f"情感分析完成: {result['label']} ({result['score']:.3f})")

            return result

        except Exception as e:
            logger.error(f"情感分析失败: {e}")
            return None

    async def batch_analyze_sentiment(
        self, texts: list[str]
    ) -> list[dict[str, str | float] | None]:
        """批量分析文本情感

        Args:
            texts: 文本列表

        Returns:
            情感分析结果列表
        """
        if not self.is_ready():
            logger.error("NLP服务未就绪，请先调用initialize()")
            return []

        if not texts:
            return []

        try:
            logger.info(f"开始批量情感分析: {len(texts)} 条文本")

            # 在线程池中执行批量推理
            loop = asyncio.get_event_loop()
            results = await loop.run_in_executor(
                None, self.finbert_client.batch_predict, texts
            )

            # 统计结果
            successful_count = sum(1 for r in results if r is not None)
            logger.info(f"批量情感分析完成: {successful_count}/{len(texts)} 条成功")

            return results

        except Exception as e:
            logger.error(f"批量情感分析失败: {e}")
            return [None] * len(texts)

    async def preprocess_text(self, text: str) -> str | None:
        """预处理文本

        Args:
            text: 原始文本

        Returns:
            预处理后的文本
        """
        try:
            # 文本预处理是CPU密集型任务，在线程池中执行
            loop = asyncio.get_event_loop()
            result = await loop.run_in_executor(
                None, self.text_processor.preprocess_for_finbert, text, self.max_length
            )

            return result

        except Exception as e:
            logger.error(f"文本预处理失败: {e}")
            return None

    async def batch_preprocess_text(self, texts: list[str]) -> list[str]:
        """批量预处理文本

        Args:
            texts: 文本列表

        Returns:
            预处理后的文本列表
        """
        try:
            # 批量预处理在线程池中执行
            loop = asyncio.get_event_loop()
            results = await loop.run_in_executor(
                None, self.text_processor.batch_preprocess, texts, self.max_length
            )

            return results

        except Exception as e:
            logger.error(f"批量文本预处理失败: {e}")
            return []

    async def extract_financial_terms(self, text: str) -> list[str]:
        """提取金融术语

        Args:
            text: 输入文本

        Returns:
            金融术语列表
        """
        try:
            loop = asyncio.get_event_loop()
            terms = await loop.run_in_executor(
                None, self.text_processor.extract_financial_terms, text
            )

            return terms

        except Exception as e:
            logger.error(f"金融术语提取失败: {e}")
            return []

    async def extract_features(self, text: str) -> list[float] | None:
        """提取文本特征向量

        Args:
            text: 输入文本

        Returns:
            特征向量列表
        """
        if not self.is_ready():
            logger.error("NLP服务未就绪，请先调用initialize()")
            return None

        try:
            loop = asyncio.get_event_loop()
            features_tensor = await loop.run_in_executor(
                None, self.finbert_client.extract_features, text
            )

            if features_tensor is not None:
                # 转换为列表格式
                return features_tensor.squeeze().tolist()

            return None

        except Exception as e:
            logger.error(f"特征提取失败: {e}")
            return None

    async def process_news_content(
        self, news_content: str
    ) -> dict[str, str | float | list[str]] | None:
        """处理新闻内容

        综合处理新闻文本，包括预处理、情感分析和术语提取

        Args:
            news_content: 新闻内容

        Returns:
            处理结果，包含情感分析和术语提取结果
        """
        try:
            # 并发执行多个处理任务
            sentiment_task = self.analyze_sentiment(news_content)
            terms_task = self.extract_financial_terms(news_content)

            sentiment_result, financial_terms = await asyncio.gather(
                sentiment_task, terms_task, return_exceptions=True
            )

            # 处理异常结果
            if isinstance(sentiment_result, Exception):
                logger.error(f"情感分析异常: {sentiment_result}")
                sentiment_result = None

            if isinstance(financial_terms, Exception):
                logger.error(f"术语提取异常: {financial_terms}")
                financial_terms = []

            # 构建结果
            result: dict[str, str | float | list[str]] = {
                "sentiment": sentiment_result,
                "financial_terms": financial_terms or [],
                "processed_at": asyncio.get_event_loop().time(),
            }

            return result

        except Exception as e:
            logger.error(f"新闻内容处理失败: {e}")
            return None

    async def get_performance_stats(self) -> dict[str, int | float | str]:
        """获取性能统计信息

        Returns:
            性能统计数据
        """
        try:
            loop = asyncio.get_event_loop()
            stats: dict[str, int | float | str] = await loop.run_in_executor(
                None, self.finbert_client.get_performance_stats
            )

            # 添加服务级别的统计信息
            stats.update(
                {
                    "service_initialized": self._is_initialized,
                    "service_ready": self.is_ready(),
                    "max_length": self.max_length,
                }
            )

            return stats

        except Exception as e:
            logger.error(f"获取性能统计失败: {e}")
            return {}

    async def cleanup(self) -> None:
        """清理服务资源"""
        try:
            logger.info("开始清理NLP服务资源")

            # 在线程池中执行清理
            loop = asyncio.get_event_loop()
            await loop.run_in_executor(None, self.finbert_client.cleanup)

            self._is_initialized = False
            logger.info("NLP服务资源清理完成")

        except Exception as e:
            logger.error(f"NLP服务清理失败: {e}")

    async def __aenter__(self) -> "NLPService":
        """异步上下文管理器入口"""
        await self.initialize()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """异步上下文管理器出口"""
        await self.cleanup()
