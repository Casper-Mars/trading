"""FinBERT模型客户端

提供FinBERT模型的本地部署、加载和推理功能，专门用于金融文本情感分析。
"""

import os
import time

import torch
from loguru import logger
from transformers import (
    AutoModelForSequenceClassification,
    AutoTokenizer,
    Pipeline,
    pipeline,
)

from utils.text_processor import TextProcessor


class FinBERTClient:
    """FinBERT模型客户端

    负责FinBERT模型的本地部署和推理，提供：
    - 模型初始化和加载
    - 单条和批量情感分析
    - 特征提取
    - 性能优化
    - 错误处理和重试机制
    """

    def __init__(
        self,
        model_name: str = "ProsusAI/finbert",
        cache_dir: str | None = None,
        device: str | None = None,
        max_length: int = 512,
    ):
        """初始化FinBERT客户端

        Args:
            model_name: 模型名称，默认使用ProsusAI/finbert
            cache_dir: 模型缓存目录
            device: 计算设备，None表示自动选择
            max_length: 最大输入长度
        """
        self.model_name = model_name
        self.cache_dir = cache_dir or os.path.expanduser(
            "~/.cache/huggingface/transformers"
        )
        self.max_length = max_length

        # 设备选择
        if device is None:
            self.device = "cuda" if torch.cuda.is_available() else "cpu"
        else:
            self.device = device

        # 模型组件
        self.tokenizer: AutoTokenizer | None = None
        self.model: AutoModelForSequenceClassification | None = None
        self.pipeline: Pipeline | None = None

        # 文本处理器
        self.text_processor = TextProcessor()

        # 性能统计
        self.inference_times = []
        self.total_predictions = 0

        # 标签映射
        self.label_mapping = {
            "LABEL_0": "negative",
            "LABEL_1": "neutral",
            "LABEL_2": "positive",
        }

        logger.info(f"FinBERT客户端初始化完成，设备: {self.device}")

    def load_model(self) -> bool:
        """加载FinBERT模型

        Returns:
            是否加载成功
        """
        try:
            logger.info(f"开始加载FinBERT模型: {self.model_name}")
            start_time = time.time()

            # 加载tokenizer
            self.tokenizer = AutoTokenizer.from_pretrained(
                self.model_name, cache_dir=self.cache_dir, trust_remote_code=True
            )

            # 加载模型
            self.model = AutoModelForSequenceClassification.from_pretrained(
                self.model_name, cache_dir=self.cache_dir, trust_remote_code=True
            )

            # 移动到指定设备
            self.model.to(self.device)

            # 设置为评估模式
            self.model.eval()

            # 创建pipeline
            self.pipeline = pipeline(
                "sentiment-analysis",
                model=self.model,
                tokenizer=self.tokenizer,
                device=0 if self.device == "cuda" else -1,
                return_all_scores=True,
            )

            load_time = time.time() - start_time
            logger.info(f"FinBERT模型加载完成，耗时: {load_time:.2f}秒")

            return True

        except Exception as e:
            logger.error(f"FinBERT模型加载失败: {e}")
            return False

    def is_ready(self) -> bool:
        """检查模型是否就绪

        Returns:
            模型是否已加载并就绪
        """
        return (
            self.tokenizer is not None
            and self.model is not None
            and self.pipeline is not None
        )

    def predict_sentiment(self, text: str) -> dict[str, str | float] | None:
        """预测单条文本情感

        Args:
            text: 输入文本

        Returns:
            情感分析结果，包含label和score
        """
        if not self.is_ready():
            logger.error("模型未就绪，请先调用load_model()")
            return None

        try:
            # 预处理文本
            processed_text = self.text_processor.preprocess_for_finbert(
                text, self.max_length
            )

            if not processed_text:
                logger.warning(f"文本预处理失败: {text[:50]}...")
                return None

            start_time = time.time()

            # 执行推理
            with torch.no_grad():
                results = self.pipeline(processed_text)

            inference_time = time.time() - start_time
            self.inference_times.append(inference_time)
            self.total_predictions += 1

            # 处理结果
            if results and len(results) > 0:
                # 获取最高分数的结果
                best_result = max(results[0], key=lambda x: x["score"])

                # 标准化标签
                label = self.label_mapping.get(
                    best_result["label"], best_result["label"]
                )

                result = {
                    "label": label,
                    "score": float(best_result["score"]),
                    "confidence": float(best_result["score"]),
                    "inference_time": inference_time,
                    "all_scores": [
                        {
                            "label": self.label_mapping.get(r["label"], r["label"]),
                            "score": float(r["score"]),
                        }
                        for r in results[0]
                    ],
                }

                logger.debug(f"情感分析完成: {label} ({best_result['score']:.3f})")
                return result

            return None

        except Exception as e:
            logger.error(f"情感预测失败: {e}")
            return None

    def batch_predict(
        self, texts: list[str]
    ) -> list[dict[str, str | float] | None]:
        """批量预测文本情感

        Args:
            texts: 文本列表

        Returns:
            情感分析结果列表
        """
        if not self.is_ready():
            logger.error("模型未就绪，请先调用load_model()")
            return []

        if not texts:
            return []

        try:
            logger.info(f"开始批量情感分析: {len(texts)} 条文本")
            start_time = time.time()

            # 预处理所有文本
            processed_texts = self.text_processor.batch_preprocess(
                texts, self.max_length
            )

            if not processed_texts:
                logger.warning("所有文本预处理失败")
                return [None] * len(texts)

            # 批量推理
            with torch.no_grad():
                batch_results = self.pipeline(processed_texts)

            # 处理结果
            results = []
            for _i, text_results in enumerate(batch_results):
                if text_results and len(text_results) > 0:
                    best_result = max(text_results, key=lambda x: x["score"])
                    label = self.label_mapping.get(
                        best_result["label"], best_result["label"]
                    )

                    result = {
                        "label": label,
                        "score": float(best_result["score"]),
                        "confidence": float(best_result["score"]),
                        "all_scores": [
                            {
                                "label": self.label_mapping.get(r["label"], r["label"]),
                                "score": float(r["score"]),
                            }
                            for r in text_results
                        ],
                    }
                    results.append(result)
                else:
                    results.append(None)

            # 补齐结果列表（处理预处理失败的文本）
            while len(results) < len(texts):
                results.append(None)

            batch_time = time.time() - start_time
            self.total_predictions += len(processed_texts)

            logger.info(
                f"批量情感分析完成: {len(processed_texts)}/{len(texts)} 条文本，"
                f"耗时: {batch_time:.2f}秒，平均: {batch_time / len(processed_texts):.3f}秒/条"
            )

            return results

        except Exception as e:
            logger.error(f"批量情感预测失败: {e}")
            return [None] * len(texts)

    def extract_features(self, text: str) -> torch.Tensor | None:
        """提取文本特征向量

        Args:
            text: 输入文本

        Returns:
            特征向量，如果提取失败返回None
        """
        if not self.is_ready():
            logger.error("模型未就绪，请先调用load_model()")
            return None

        try:
            # 预处理文本
            processed_text = self.text_processor.preprocess_for_finbert(
                text, self.max_length
            )

            if not processed_text:
                return None

            # 编码文本
            inputs = self.tokenizer(
                processed_text,
                return_tensors="pt",
                max_length=self.max_length,
                truncation=True,
                padding=True,
            )

            # 移动到设备
            inputs = {k: v.to(self.device) for k, v in inputs.items()}

            # 提取特征
            with torch.no_grad():
                outputs = self.model(**inputs, output_hidden_states=True)
                # 使用最后一层的[CLS]token作为句子表示
                features = outputs.hidden_states[-1][:, 0, :]

            return features.cpu()

        except Exception as e:
            logger.error(f"特征提取失败: {e}")
            return None

    def optimize_inference(self) -> bool:
        """优化推理性能

        Returns:
            是否优化成功
        """
        if not self.is_ready():
            logger.error("模型未就绪，无法优化")
            return False

        try:
            logger.info("开始优化推理性能")

            # 1. 启用半精度推理（如果支持）
            if self.device == "cuda" and torch.cuda.is_available():
                try:
                    self.model = self.model.half()
                    logger.info("启用半精度推理")
                except Exception as e:
                    logger.warning(f"半精度推理启用失败: {e}")

            # 2. 编译模型（PyTorch 2.0+）
            try:
                if hasattr(torch, "compile"):
                    self.model = torch.compile(self.model)
                    logger.info("模型编译完成")
            except Exception as e:
                logger.warning(f"模型编译失败: {e}")

            # 3. 设置推理模式
            self.model.eval()

            # 4. 预热模型
            self._warmup_model()

            logger.info("推理性能优化完成")
            return True

        except Exception as e:
            logger.error(f"推理优化失败: {e}")
            return False

    def _warmup_model(self, warmup_texts: list[str] | None = None) -> None:
        """预热模型

        Args:
            warmup_texts: 预热文本，如果为None则使用默认文本
        """
        if warmup_texts is None:
            warmup_texts = [
                "股票市场今天表现良好，投资者信心增强。",
                "公司业绩下滑，股价大幅下跌。",
                "市场波动较大，建议谨慎投资。",
            ]

        try:
            logger.info("开始模型预热")
            start_time = time.time()

            for text in warmup_texts:
                self.predict_sentiment(text)

            warmup_time = time.time() - start_time
            logger.info(f"模型预热完成，耗时: {warmup_time:.2f}秒")

        except Exception as e:
            logger.warning(f"模型预热失败: {e}")

    def get_performance_stats(self) -> dict[str, int | float]:
        """获取性能统计信息

        Returns:
            性能统计数据
        """
        if not self.inference_times:
            return {
                "total_predictions": self.total_predictions,
                "avg_inference_time": 0.0,
                "min_inference_time": 0.0,
                "max_inference_time": 0.0,
            }

        return {
            "total_predictions": self.total_predictions,
            "avg_inference_time": sum(self.inference_times) / len(self.inference_times),
            "min_inference_time": min(self.inference_times),
            "max_inference_time": max(self.inference_times),
            "device": self.device,
            "model_name": self.model_name,
        }

    def cleanup(self) -> None:
        """清理资源"""
        try:
            if self.model is not None:
                del self.model
            if self.tokenizer is not None:
                del self.tokenizer
            if self.pipeline is not None:
                del self.pipeline

            # 清理GPU缓存
            if torch.cuda.is_available():
                torch.cuda.empty_cache()

            logger.info("FinBERT客户端资源清理完成")

        except Exception as e:
            logger.error(f"资源清理失败: {e}")

    def __del__(self):
        """析构函数"""
        self.cleanup()
