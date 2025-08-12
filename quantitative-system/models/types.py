"""自定义类型定义模块"""

from decimal import Decimal
from typing import Annotated, Any

from pydantic import Field
from sqlalchemy import DECIMAL, String
from sqlalchemy.types import TypeDecorator

# ============= 基础类型别名 =============

# 股票代码类型
StockCode = Annotated[str, Field(max_length=20, description="股票代码")]

# 股票名称类型
StockName = Annotated[str, Field(max_length=100, description="股票名称")]

# 价格类型 (精确到4位小数)
Price = Annotated[Decimal, Field(max_digits=10, decimal_places=4, description="价格")]

# 金额类型 (精确到2位小数)
Amount = Annotated[Decimal, Field(max_digits=15, decimal_places=2, description="金额")]

# 比率类型 (精确到4位小数，如收益率、涨跌幅等)
Ratio = Annotated[Decimal, Field(max_digits=8, decimal_places=4, description="比率")]

# 成交量类型
Volume = Annotated[Decimal, Field(max_digits=20, decimal_places=2, description="成交量")]

# 情感分数类型 (-1到1之间)
SentimentScore = Annotated[Decimal, Field(max_digits=5, decimal_places=4, ge=-1, le=1, description="情感分数")]

# JSON数据类型
JSONData = dict[str, Any]

# 标签列表类型
TagList = list[str]

# 股票代码列表类型
StockCodeList = list[str]

# 关键词列表类型
KeywordList = list[str]

# ============= SQLAlchemy自定义类型 =============

class PreciseDecimal(TypeDecorator):
    """精确小数类型，用于金融数据"""

    impl = DECIMAL
    cache_ok = True

    def __init__(self, precision: int = 15, scale: int = 4):
        self.precision = precision
        self.scale = scale
        super().__init__(precision=precision, scale=scale)

    def load_dialect_impl(self, dialect):
        return dialect.type_descriptor(DECIMAL(precision=self.precision, scale=self.scale))


class LimitedString(TypeDecorator):
    """限制长度的字符串类型"""

    impl = String
    cache_ok = True

    def __init__(self, max_length: int):
        self.max_length = max_length
        super().__init__(length=max_length)

    def load_dialect_impl(self, dialect):
        return dialect.type_descriptor(String(length=self.max_length))


# ============= 业务类型定义 =============

class StockInfo:
    """股票基础信息类型"""

    def __init__(
        self,
        ts_code: str,
        symbol: str,
        name: str,
        area: str | None = None,
        industry: str | None = None,
        market: str | None = None,
    ):
        self.ts_code = ts_code
        self.symbol = symbol
        self.name = name
        self.area = area
        self.industry = industry
        self.market = market


class PriceData:
    """价格数据类型"""

    def __init__(
        self,
        open_price: Decimal | None = None,
        high_price: Decimal | None = None,
        low_price: Decimal | None = None,
        close_price: Decimal | None = None,
        volume: Decimal | None = None,
        amount: Decimal | None = None,
    ):
        self.open_price = open_price
        self.high_price = high_price
        self.low_price = low_price
        self.close_price = close_price
        self.volume = volume
        self.amount = amount


class SentimentData:
    """情感分析数据类型"""

    def __init__(
        self,
        score: Decimal | None = None,
        label: str | None = None,
        confidence: Decimal | None = None,
        keywords: list[str] | None = None,
    ):
        self.score = score
        self.label = label
        self.confidence = confidence
        self.keywords = keywords or []


# ============= 常量定义 =============

# 默认精度配置
DEFAULT_PRICE_PRECISION = 4
DEFAULT_AMOUNT_PRECISION = 2
DEFAULT_RATIO_PRECISION = 4

# 字符串长度限制
MAX_STOCK_CODE_LENGTH = 20
MAX_STOCK_NAME_LENGTH = 100
MAX_NEWS_TITLE_LENGTH = 500
MAX_NEWS_SOURCE_LENGTH = 100
MAX_CATEGORY_LENGTH = 50
MAX_SENTIMENT_LABEL_LENGTH = 20

# 数值范围限制
MIN_SENTIMENT_SCORE = -1.0
MAX_SENTIMENT_SCORE = 1.0
MIN_CONFIDENCE_SCORE = 0.0
MAX_CONFIDENCE_SCORE = 1.0

# 默认值
DEFAULT_SENTIMENT_SCORE = 0.0
DEFAULT_CONFIDENCE_SCORE = 0.0
