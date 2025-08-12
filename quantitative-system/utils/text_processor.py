"""文本预处理器模块

提供金融文本的清洗、标准化和预处理功能，为FinBERT模型准备输入数据。
"""

import re

from loguru import logger


class TextProcessor:
    """文本预处理器

    负责金融文本的清洗、标准化和预处理，包括：
    - 文本清洗和去噪
    - 标准化处理
    - 分词和标记化
    - 输入验证
    """

    def __init__(self):
        """初始化文本处理器"""
        # 金融相关的停用词
        self.financial_stopwords = {
            "的",
            "了",
            "在",
            "是",
            "我",
            "有",
            "和",
            "就",
            "不",
            "人",
            "都",
            "一",
            "一个",
            "上",
            "也",
            "很",
            "到",
            "说",
            "要",
            "去",
            "你",
            "会",
            "着",
            "没有",
            "看",
            "好",
            "自己",
            "这",
            "那",
            "它",
            "他",
            "她",
            "们",
            "这个",
            "那个",
            "什么",
            "怎么",
            "为什么",
            "哪里",
            "什么时候",
            "如何",
            "多少",
            "哪个",
            "哪些",
        }

        # 需要保留的金融术语模式
        self.financial_terms_pattern = re.compile(
            r"(股票|股价|涨跌|涨幅|跌幅|成交量|市值|PE|PB|ROE|净利润|营收|毛利率|"
            r"利润率|现金流|资产|负债|股东|分红|配股|增发|重组|并购|IPO|"
            r"牛市|熊市|震荡|突破|支撑|阻力|技术分析|基本面|消息面|"
            r"央行|货币政策|利率|通胀|GDP|CPI|PMI|经济|金融|银行|"
            r"证券|基金|保险|期货|债券|外汇|黄金|原油|商品)"
        )

        # HTML标签清理模式
        self.html_pattern = re.compile(r"<[^>]+>")

        # URL清理模式
        self.url_pattern = re.compile(
            r"http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*\(\),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+"
        )

        # 特殊字符清理模式（保留中文、英文、数字、基本标点）
        self.special_chars_pattern = re.compile(
            r'[^\u4e00-\u9fff\w\s.,!?;:()\[\]{}"\'-]'
        )

        # 多空格清理模式
        self.whitespace_pattern = re.compile(r"\s+")

        logger.info("文本处理器初始化完成")

    def clean_text(self, text: str) -> str:
        """清洗文本

        Args:
            text: 原始文本

        Returns:
            清洗后的文本
        """
        if not text or not isinstance(text, str):
            return ""

        try:
            # 1. 移除HTML标签
            text = self.html_pattern.sub("", text)

            # 2. 移除URL链接
            text = self.url_pattern.sub("", text)

            # 3. 移除特殊字符（保留中文、英文、数字、基本标点）
            text = self.special_chars_pattern.sub(" ", text)

            # 4. 标准化空格
            text = self.whitespace_pattern.sub(" ", text)

            # 5. 去除首尾空格
            text = text.strip()

            return text

        except Exception as e:
            logger.error(f"文本清洗失败: {e}")
            return ""

    def normalize(self, text: str) -> str:
        """标准化文本

        Args:
            text: 输入文本

        Returns:
            标准化后的文本
        """
        if not text:
            return ""

        try:
            # 1. 转换为小写（仅英文）
            # 保持中文不变，只转换英文字母
            normalized = "".join(
                char.lower() if char.isascii() and char.isalpha() else char
                for char in text
            )

            # 2. 标准化标点符号
            punctuation_map = {
                "，": ",",
                "。": ".",
                "！": "!",
                "？": "?",
                "；": ";",
                "：": ":",
                "（": "(",
                "）": ")",
                "【": "[",
                "】": "]",
                "「": '"',
                "」": '"',
            }

            for old, new in punctuation_map.items():
                normalized = normalized.replace(old, new)

            # 3. 移除多余的标点符号
            normalized = re.sub(r"[.,!?;:]{2,}", ".", normalized)

            return normalized.strip()

        except Exception as e:
            logger.error(f"文本标准化失败: {e}")
            return text

    def tokenize(self, text: str) -> list[str]:
        """分词处理

        Args:
            text: 输入文本

        Returns:
            分词结果列表
        """
        if not text:
            return []

        try:
            # 简单的分词实现（基于空格和标点符号）
            # 对于更复杂的中文分词，可以集成jieba等库

            # 1. 按空格和标点符号分割
            tokens = re.findall(r"[\u4e00-\u9fff]+|[a-zA-Z]+|[0-9]+", text)

            # 2. 过滤空token和停用词
            filtered_tokens = [
                token
                for token in tokens
                if token and len(token) > 1 and token not in self.financial_stopwords
            ]

            return filtered_tokens

        except Exception as e:
            logger.error(f"分词处理失败: {e}")
            return []

    def validate_input(self, text: str) -> bool:
        """验证输入文本

        Args:
            text: 输入文本

        Returns:
            是否为有效输入
        """
        if not text or not isinstance(text, str):
            return False

        # 检查文本长度
        if len(text.strip()) < 5:
            return False

        # 检查是否包含有意义的内容（不全是标点符号或空格）
        meaningful_chars = re.sub(r"[\s\W]", "", text)
        return not len(meaningful_chars) < 3

    def extract_financial_terms(self, text: str) -> list[str]:
        """提取金融术语

        Args:
            text: 输入文本

        Returns:
            提取的金融术语列表
        """
        if not text:
            return []

        try:
            matches = self.financial_terms_pattern.findall(text)
            return list(set(matches))  # 去重

        except Exception as e:
            logger.error(f"金融术语提取失败: {e}")
            return []

    def preprocess_for_finbert(self, text: str, max_length: int = 512) -> str | None:
        """为FinBERT模型预处理文本

        Args:
            text: 原始文本
            max_length: 最大长度限制

        Returns:
            预处理后的文本，如果处理失败返回None
        """
        try:
            # 1. 验证输入
            if not self.validate_input(text):
                logger.warning(f"输入文本验证失败: {text[:50]}...")
                return None

            # 2. 清洗文本
            cleaned = self.clean_text(text)
            if not cleaned:
                return None

            # 3. 标准化文本
            normalized = self.normalize(cleaned)
            if not normalized:
                return None

            # 4. 长度截断（为BERT模型预留特殊token空间）
            if len(normalized) > max_length - 10:
                # 尝试在句号处截断，保持语义完整性
                sentences = normalized.split(".")
                truncated = ""
                for sentence in sentences:
                    if len(truncated + sentence + ".") <= max_length - 10:
                        truncated += sentence + "."
                    else:
                        break

                if truncated:
                    normalized = truncated.rstrip(".")
                else:
                    # 如果没有合适的截断点，直接截断
                    normalized = normalized[: max_length - 10]

            logger.debug(f"文本预处理完成: {len(normalized)} 字符")
            return normalized

        except Exception as e:
            logger.error(f"FinBERT文本预处理失败: {e}")
            return None

    def batch_preprocess(self, texts: list[str], max_length: int = 512) -> list[str]:
        """批量预处理文本

        Args:
            texts: 文本列表
            max_length: 最大长度限制

        Returns:
            预处理后的文本列表
        """
        if not texts:
            return []

        processed_texts = []
        for text in texts:
            processed = self.preprocess_for_finbert(text, max_length)
            if processed:
                processed_texts.append(processed)

        logger.info(f"批量预处理完成: {len(processed_texts)}/{len(texts)} 条文本")
        return processed_texts
