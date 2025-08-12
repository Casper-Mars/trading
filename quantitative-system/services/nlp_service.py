"""NLP处理服务

提供自然语言处理功能，包括情感分析、实体识别和文本处理。
集成FinBERT模型进行金融文本情感分析。
"""

import asyncio
import hashlib
import re
from datetime import datetime
from typing import Any

from loguru import logger

from models.database import NewsData
from repositories.news_repo import NewsRepository
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

        # 新闻数据仓库
        self.news_repo: NewsRepository | None = None

        # 股票代码正则表达式
        self.stock_code_pattern = re.compile(r"\b[0-9]{6}\b|\b[A-Z]{2,5}\b")

        # 金融实体关键词
        self.financial_entities = {
            "公司": ["公司", "企业", "集团", "股份", "有限公司", "科技", "投资"],
            "行业": [
                "银行",
                "保险",
                "证券",
                "基金",
                "信托",
                "期货",
                "房地产",
                "制造业",
            ],
            "指标": ["营收", "利润", "净利润", "毛利率", "净资产", "负债", "现金流"],
            "事件": ["并购", "重组", "上市", "退市", "分红", "增发", "回购"],
        }

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

    def set_news_repository(self, news_repo: NewsRepository) -> None:
        """设置新闻数据仓库

        Args:
            news_repo: 新闻数据仓库实例
        """
        self.news_repo = news_repo
        logger.info("新闻数据仓库已设置")

    async def clean_and_deduplicate_text(self, text: str) -> str:
        """清洗和去重文本

        Args:
            text: 原始文本

        Returns:
            清洗后的文本
        """
        try:
            # 基础文本清洗
            cleaned_text = await self.preprocess_text(text)
            if not cleaned_text:
                return ""

            # 去除重复句子
            sentences = cleaned_text.split("。")
            unique_sentences = []
            seen_hashes = set()

            for sentence in sentences:
                sentence = sentence.strip()
                if not sentence:
                    continue

                # 计算句子哈希
                sentence_hash = hashlib.sha256(sentence.encode()).hexdigest()
                if sentence_hash not in seen_hashes:
                    seen_hashes.add(sentence_hash)
                    unique_sentences.append(sentence)

            return "。".join(unique_sentences)

        except Exception as e:
            logger.error(f"文本清洗和去重失败: {e}")
            return text

    async def extract_stock_codes(self, text: str) -> list[str]:
        """从文本中提取股票代码

        Args:
            text: 输入文本

        Returns:
            股票代码列表
        """
        try:
            # 使用正则表达式提取可能的股票代码
            matches = self.stock_code_pattern.findall(text)

            # 过滤和验证股票代码
            stock_codes = []
            for match in matches:
                # A股代码（6位数字）
                if match.isdigit() and len(match) == 6:
                    stock_codes.append(match)
                # 美股代码（2-5位字母）
                elif match.isalpha() and 2 <= len(match) <= 5:
                    stock_codes.append(match.upper())

            # 去重并返回
            return list(set(stock_codes))

        except Exception as e:
            logger.error(f"股票代码提取失败: {e}")
            return []

    async def extract_entities_and_keywords(self, text: str) -> dict[str, list[str]]:
        """提取实体和关键词

        Args:
            text: 输入文本

        Returns:
            包含实体和关键词的字典
        """
        try:
            result = {"entities": [], "keywords": [], "financial_entities": {}}

            # 提取金融术语作为关键词
            financial_terms = await self.extract_financial_terms(text)
            result["keywords"].extend(financial_terms)

            # 提取股票代码
            stock_codes = await self.extract_stock_codes(text)
            result["entities"].extend(stock_codes)

            # 提取金融实体
            for entity_type, keywords in self.financial_entities.items():
                found_entities = []
                for keyword in keywords:
                    if keyword in text:
                        found_entities.append(keyword)
                if found_entities:
                    result["financial_entities"][entity_type] = found_entities

            # 简单的关键词提取（基于词频）
            words = re.findall(r"\b[\u4e00-\u9fff]+\b", text)  # 中文词汇
            word_freq = {}
            for word in words:
                if len(word) >= 2:  # 过滤单字
                    word_freq[word] = word_freq.get(word, 0) + 1

            # 取频率最高的词作为关键词
            top_keywords = sorted(word_freq.items(), key=lambda x: x[1], reverse=True)[
                :10
            ]
            result["keywords"].extend([word for word, _ in top_keywords])

            # 去重
            result["keywords"] = list(set(result["keywords"]))
            result["entities"] = list(set(result["entities"]))

            return result

        except Exception as e:
            logger.error(f"实体和关键词提取失败: {e}")
            return {"entities": [], "keywords": [], "financial_entities": {}}

    async def quantify_sentiment_intensity(
        self, sentiment_result: dict[str, str | float]
    ) -> float:
        """量化情感强度

        Args:
            sentiment_result: 情感分析结果

        Returns:
            情感强度分数（-1到1）
        """
        try:
            if not sentiment_result:
                return 0.0

            label = sentiment_result.get("label", "").lower()
            score = float(sentiment_result.get("score", 0.0))

            # 根据标签调整分数
            if label == "positive":
                return score
            elif label == "negative":
                return -score
            else:  # neutral
                return 0.0

        except Exception as e:
            logger.error(f"情感强度量化失败: {e}")
            return 0.0

    async def process_news_data(self, news: NewsData) -> dict[str, Any] | None:
        """处理新闻数据

        按照时序图实现：获取待处理新闻→文本预处理→FinBERT情感分析→存储结果

        Args:
            news: 新闻数据对象

        Returns:
            处理结果
        """
        try:
            logger.info(f"开始处理新闻: {news.id} - {news.title[:50]}...")

            # 1. 文本预处理和清洗
            cleaned_title = await self.clean_and_deduplicate_text(news.title)
            cleaned_content = await self.clean_and_deduplicate_text(news.content or "")

            # 2. 情感分析
            title_sentiment = await self.analyze_sentiment(cleaned_title)
            content_sentiment = await self.analyze_sentiment(cleaned_content)

            # 3. 实体识别和关键词提取
            full_text = f"{cleaned_title} {cleaned_content}"
            entities_keywords = await self.extract_entities_and_keywords(full_text)

            # 4. 股票代码关联
            related_stocks = await self.extract_stock_codes(full_text)

            # 5. 情感强度量化
            title_intensity = await self.quantify_sentiment_intensity(title_sentiment)
            content_intensity = await self.quantify_sentiment_intensity(
                content_sentiment
            )

            # 综合情感分数（标题权重0.3，内容权重0.7）
            overall_sentiment_score = title_intensity * 0.3 + content_intensity * 0.7

            # 确定整体情感标签
            if overall_sentiment_score > 0.1:
                overall_sentiment_label = "positive"
            elif overall_sentiment_score < -0.1:
                overall_sentiment_label = "negative"
            else:
                overall_sentiment_label = "neutral"

            # 构建处理结果
            result = {
                "news_id": news.id,
                "title_sentiment": title_sentiment,
                "content_sentiment": content_sentiment,
                "overall_sentiment_score": overall_sentiment_score,
                "overall_sentiment_label": overall_sentiment_label,
                "entities": entities_keywords["entities"],
                "keywords": entities_keywords["keywords"],
                "financial_entities": entities_keywords["financial_entities"],
                "related_stocks": related_stocks,
                "processed_at": datetime.now().isoformat(),
            }

            # 6. 更新新闻记录
            if self.news_repo:
                await self.news_repo.update_news_sentiment(
                    news.id,
                    overall_sentiment_score,
                    overall_sentiment_label,
                    entities_keywords["keywords"],
                )

                # 更新相关股票
                if related_stocks:
                    news.related_stocks = related_stocks
                    news.updated_at = datetime.now()
                    await self.news_repo.session.commit()

            logger.info(
                f"新闻处理完成: {news.id} - 情感: {overall_sentiment_label} ({overall_sentiment_score:.3f})"
            )
            return result

        except Exception as e:
            logger.error(f"新闻数据处理失败: {e}")
            return None

    async def batch_process_news(
        self, news_list: list[NewsData]
    ) -> list[dict[str, Any]]:
        """批量处理新闻数据

        Args:
            news_list: 新闻数据列表

        Returns:
            处理结果列表
        """
        try:
            logger.info(f"开始批量处理新闻: {len(news_list)} 条")

            # 并发处理新闻（限制并发数避免资源耗尽）
            semaphore = asyncio.Semaphore(5)  # 最多5个并发任务

            async def process_with_semaphore(news: NewsData) -> dict[str, Any] | None:
                async with semaphore:
                    return await self.process_news_data(news)

            tasks = [process_with_semaphore(news) for news in news_list]
            results = await asyncio.gather(*tasks, return_exceptions=True)

            # 过滤异常结果
            valid_results = []
            for result in results:
                if isinstance(result, Exception):
                    logger.error(f"批量处理中的异常: {result}")
                elif result is not None:
                    valid_results.append(result)

            logger.info(f"批量处理完成: {len(valid_results)}/{len(news_list)} 条成功")
            return valid_results

        except Exception as e:
            logger.error(f"批量新闻处理失败: {e}")
            return []
