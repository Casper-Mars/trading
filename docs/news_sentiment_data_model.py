#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
消息面指标原始数据模型
用于定义新闻、政策、舆论等消息面数据的结构和处理方法

作者: 量化分析师
创建时间: 2024
"""

from dataclasses import dataclass, field
from datetime import datetime
from typing import List, Dict, Optional, Union
from enum import Enum
import json

class SentimentType(Enum):
    """情绪类型枚举"""
    POSITIVE = "positive"  # 积极
    NEGATIVE = "negative"  # 消极
    NEUTRAL = "neutral"   # 中性
    MIXED = "mixed"       # 混合

class NewsCategory(Enum):
    """新闻分类枚举"""
    FINANCIAL = "financial"     # 财经新闻
    POLICY = "policy"           # 政策新闻
    COMPANY = "company"         # 公司新闻
    INDUSTRY = "industry"       # 行业新闻
    MACRO = "macro"             # 宏观新闻
    MARKET = "market"           # 市场新闻

class PolicyLevel(Enum):
    """政策层级枚举"""
    NATIONAL = "national"       # 国家级
    MINISTRY = "ministry"       # 部委级
    PROVINCIAL = "provincial"   # 省级
    MUNICIPAL = "municipal"     # 市级

class EventSeverity(Enum):
    """事件严重程度枚举"""
    CRITICAL = "critical"       # 严重
    HIGH = "high"               # 高
    MEDIUM = "medium"           # 中等
    LOW = "low"                 # 低

@dataclass
class NewsData:
    """新闻数据模型"""
    # 基本信息
    news_id: str                    # 新闻ID
    title: str                      # 新闻标题
    content: str                    # 新闻内容
    summary: Optional[str] = None   # 新闻摘要
    
    # 时间信息
    publish_time: datetime          # 发布时间
    crawl_time: datetime           # 抓取时间
    
    # 来源信息
    source: str                     # 新闻来源
    author: Optional[str] = None    # 作者
    url: str                        # 原文链接
    
    # 分类信息
    category: NewsCategory          # 新闻分类
    tags: List[str] = field(default_factory=list)  # 标签
    
    # 关联信息
    stock_codes: List[str] = field(default_factory=list)  # 相关股票代码
    industry_codes: List[str] = field(default_factory=list)  # 相关行业代码
    
    # 影响力指标
    view_count: int = 0             # 阅读量
    comment_count: int = 0          # 评论数
    share_count: int = 0            # 分享数
    like_count: int = 0             # 点赞数
    
    # 情绪分析结果
    sentiment_score: Optional[float] = None     # 情绪得分 (-1到1)
    sentiment_type: Optional[SentimentType] = None  # 情绪类型
    confidence: Optional[float] = None          # 置信度
    
    # 重要性评估
    importance_score: Optional[float] = None    # 重要性得分
    urgency_level: Optional[str] = None         # 紧急程度
    
    def calculate_influence_score(self) -> float:
        """计算影响力得分"""
        base_score = (
            self.view_count * 0.4 +
            self.comment_count * 0.3 +
            self.share_count * 0.2 +
            self.like_count * 0.1
        )
        return min(base_score / 10000, 10.0)  # 归一化到0-10
    
    def extract_keywords(self) -> List[str]:
        """提取关键词"""
        # 这里应该实现关键词提取算法
        # 可以使用jieba、TF-IDF、TextRank等方法
        pass
    
    def is_breaking_news(self) -> bool:
        """判断是否为突发新闻"""
        time_diff = (datetime.now() - self.publish_time).total_seconds() / 3600
        return (
            time_diff <= 2 and  # 2小时内
            self.importance_score and self.importance_score > 0.8 and
            self.urgency_level == "high"
        )

@dataclass
class PolicyData:
    """政策数据模型"""
    # 基本信息
    policy_id: str                  # 政策ID
    title: str                      # 政策标题
    content: str                    # 政策内容
    
    # 发布信息
    publish_date: datetime          # 发布日期
    effective_date: Optional[datetime] = None  # 生效日期
    expire_date: Optional[datetime] = None     # 失效日期
    
    # 发布机构
    issuer: str                     # 发布机构
    level: PolicyLevel              # 政策层级
    department: Optional[str] = None # 具体部门
    
    # 政策分类
    policy_type: str                # 政策类型（产业、财税、货币、监管等）
    industry_impact: List[str] = field(default_factory=list)  # 影响行业
    
    # 影响评估
    impact_scope: str               # 影响范围（全国、地区、行业等）
    impact_degree: str              # 影响程度（重大、一般、轻微）
    sentiment_impact: SentimentType # 情绪影响（利好、利空、中性）
    
    # 执行相关
    implementation_probability: Optional[float] = None  # 执行概率
    implementation_timeline: Optional[str] = None       # 执行时间表
    
    # 历史对比
    similar_policies: List[str] = field(default_factory=list)  # 类似政策
    historical_impact: Optional[Dict] = None                   # 历史影响数据
    
    def calculate_policy_score(self) -> float:
        """计算政策影响得分"""
        level_weight = {
            PolicyLevel.NATIONAL: 1.0,
            PolicyLevel.MINISTRY: 0.8,
            PolicyLevel.PROVINCIAL: 0.6,
            PolicyLevel.MUNICIPAL: 0.4
        }
        
        impact_weight = {
            "重大": 1.0,
            "一般": 0.6,
            "轻微": 0.3
        }
        
        sentiment_weight = {
            SentimentType.POSITIVE: 1.0,
            SentimentType.NEGATIVE: -1.0,
            SentimentType.NEUTRAL: 0.0,
            SentimentType.MIXED: 0.5
        }
        
        score = (
            level_weight.get(self.level, 0.5) *
            impact_weight.get(self.impact_degree, 0.5) *
            sentiment_weight.get(self.sentiment_impact, 0.0)
        )
        
        if self.implementation_probability:
            score *= self.implementation_probability
            
        return score

@dataclass
class SocialMediaData:
    """社交媒体数据模型"""
    # 基本信息
    post_id: str                    # 帖子ID
    platform: str                   # 平台（微博、股吧、雪球等）
    content: str                    # 内容
    
    # 用户信息
    user_id: str                    # 用户ID
    user_name: str                  # 用户名
    user_followers: int = 0         # 粉丝数
    user_level: Optional[str] = None # 用户等级
    is_verified: bool = False       # 是否认证
    
    # 时间信息
    post_time: datetime             # 发布时间
    
    # 互动数据
    like_count: int = 0             # 点赞数
    comment_count: int = 0          # 评论数
    share_count: int = 0            # 转发数
    
    # 关联信息
    mentioned_stocks: List[str] = field(default_factory=list)  # 提及股票
    hashtags: List[str] = field(default_factory=list)          # 话题标签
    
    # 情绪分析
    sentiment_score: Optional[float] = None     # 情绪得分
    sentiment_type: Optional[SentimentType] = None
    emotion_tags: List[str] = field(default_factory=list)  # 情绪标签
    
    # 影响力评估
    influence_score: Optional[float] = None     # 影响力得分
    credibility_score: Optional[float] = None   # 可信度得分
    
    def calculate_user_influence(self) -> float:
        """计算用户影响力"""
        base_influence = min(self.user_followers / 10000, 10.0)  # 粉丝数影响
        
        # 认证用户加权
        if self.is_verified:
            base_influence *= 1.5
            
        # 互动数据加权
        interaction_score = (
            self.like_count * 0.1 +
            self.comment_count * 0.3 +
            self.share_count * 0.6
        )
        
        return min(base_influence + interaction_score / 1000, 10.0)
    
    def is_spam_content(self) -> bool:
        """判断是否为垃圾内容"""
        spam_keywords = ["广告", "推广", "加群", "QQ", "微信"]
        return any(keyword in self.content for keyword in spam_keywords)

@dataclass
class EventData:
    """事件数据模型"""
    # 基本信息
    event_id: str                   # 事件ID
    title: str                      # 事件标题
    description: str                # 事件描述
    
    # 时间信息
    event_time: datetime            # 事件发生时间
    discovery_time: datetime        # 发现时间
    
    # 事件分类
    event_type: str                 # 事件类型
    category: str                   # 事件分类
    severity: EventSeverity         # 严重程度
    
    # 影响范围
    affected_stocks: List[str] = field(default_factory=list)     # 影响股票
    affected_industries: List[str] = field(default_factory=list) # 影响行业
    affected_regions: List[str] = field(default_factory=list)    # 影响地区
    
    # 市场反应
    market_reaction: Optional[Dict] = None      # 市场反应数据
    price_impact: Optional[float] = None        # 价格影响
    volume_impact: Optional[float] = None       # 成交量影响
    
    # 新闻覆盖
    news_coverage: List[str] = field(default_factory=list)  # 相关新闻
    media_attention: Optional[float] = None                 # 媒体关注度
    
    # 历史对比
    similar_events: List[str] = field(default_factory=list)  # 类似事件
    historical_impact: Optional[Dict] = None                 # 历史影响
    
    def calculate_event_impact(self) -> float:
        """计算事件影响度"""
        severity_weight = {
            EventSeverity.CRITICAL: 1.0,
            EventSeverity.HIGH: 0.8,
            EventSeverity.MEDIUM: 0.5,
            EventSeverity.LOW: 0.2
        }
        
        # 基础影响度
        base_impact = severity_weight.get(self.severity, 0.5)
        
        # 影响范围加权
        scope_multiplier = (
            len(self.affected_stocks) * 0.1 +
            len(self.affected_industries) * 0.3 +
            len(self.affected_regions) * 0.2
        )
        
        # 媒体关注度加权
        if self.media_attention:
            scope_multiplier *= (1 + self.media_attention)
            
        return min(base_impact * (1 + scope_multiplier), 10.0)

@dataclass
class AnnouncementData:
    """公告数据模型"""
    # 基本信息
    announcement_id: str            # 公告ID
    stock_code: str                 # 股票代码
    title: str                      # 公告标题
    content: str                    # 公告内容
    
    # 时间信息
    publish_time: datetime          # 发布时间
    
    # 公告分类
    announcement_type: str          # 公告类型
    category: str                   # 分类（业绩、重组、风险等）
    importance_level: str           # 重要程度
    
    # 财务数据（如果是业绩公告）
    financial_data: Optional[Dict] = None       # 财务数据
    
    # 预期对比
    market_expectation: Optional[Dict] = None   # 市场预期
    surprise_factor: Optional[float] = None     # 超预期因子
    
    # 影响评估
    estimated_impact: Optional[float] = None    # 预估影响
    analyst_rating: Optional[str] = None        # 分析师评级
    
    def extract_key_metrics(self) -> Dict:
        """提取关键指标"""
        # 这里应该实现从公告中提取关键财务指标的逻辑
        # 如营收、净利润、EPS等
        pass
    
    def calculate_surprise_score(self) -> float:
        """计算超预期得分"""
        if not self.market_expectation or not self.financial_data:
            return 0.0
            
        # 这里应该实现超预期计算逻辑
        # 比较实际值与预期值的差异
        pass

class NewsDataProcessor:
    """消息面数据处理器"""
    
    def __init__(self):
        self.sentiment_model = None  # 情绪分析模型
        self.keyword_extractor = None  # 关键词提取器
        self.importance_classifier = None  # 重要性分类器
    
    def process_news_batch(self, news_list: List[NewsData]) -> List[NewsData]:
        """批量处理新闻数据"""
        processed_news = []
        
        for news in news_list:
            # 情绪分析
            news.sentiment_score, news.sentiment_type = self.analyze_sentiment(news.content)
            
            # 重要性评估
            news.importance_score = self.assess_importance(news)
            
            # 关键词提取
            keywords = self.extract_keywords(news.content)
            news.tags.extend(keywords)
            
            # 股票关联
            news.stock_codes = self.extract_stock_codes(news.content)
            
            processed_news.append(news)
            
        return processed_news
    
    def analyze_sentiment(self, text: str) -> tuple[float, SentimentType]:
        """分析文本情绪"""
        # 这里应该实现情绪分析逻辑
        # 可以使用预训练模型或规则方法
        pass
    
    def assess_importance(self, news: NewsData) -> float:
        """评估新闻重要性"""
        # 这里应该实现重要性评估逻辑
        # 考虑来源权威性、内容关键词、影响范围等
        pass
    
    def extract_keywords(self, text: str) -> List[str]:
        """提取关键词"""
        # 这里应该实现关键词提取逻辑
        pass
    
    def extract_stock_codes(self, text: str) -> List[str]:
        """提取股票代码"""
        # 这里应该实现股票代码识别逻辑
        pass
    
    def calculate_market_sentiment(self, 
                                 news_list: List[NewsData],
                                 time_window: int = 24) -> Dict:
        """计算市场整体情绪"""
        # 计算指定时间窗口内的市场情绪指标
        current_time = datetime.now()
        recent_news = [
            news for news in news_list
            if (current_time - news.publish_time).total_seconds() / 3600 <= time_window
        ]
        
        if not recent_news:
            return {"sentiment_score": 0.0, "confidence": 0.0}
        
        # 加权平均情绪得分
        total_weight = 0
        weighted_sentiment = 0
        
        for news in recent_news:
            if news.sentiment_score is not None:
                weight = news.importance_score or 1.0
                weighted_sentiment += news.sentiment_score * weight
                total_weight += weight
        
        if total_weight == 0:
            return {"sentiment_score": 0.0, "confidence": 0.0}
        
        avg_sentiment = weighted_sentiment / total_weight
        confidence = min(len(recent_news) / 10, 1.0)  # 新闻数量越多置信度越高
        
        return {
            "sentiment_score": avg_sentiment,
            "confidence": confidence,
            "news_count": len(recent_news),
            "time_window": time_window
        }

class PolicyMonitor:
    """政策监控器"""
    
    def __init__(self):
        self.policy_database = []  # 政策数据库
        self.industry_mapping = {}  # 行业映射
    
    def monitor_policy_changes(self, industry: str = None) -> List[PolicyData]:
        """监控政策变化"""
        # 这里应该实现政策监控逻辑
        pass
    
    def assess_policy_impact(self, policy: PolicyData, stock_codes: List[str]) -> Dict:
        """评估政策对特定股票的影响"""
        # 这里应该实现政策影响评估逻辑
        pass
    
    def generate_policy_alert(self, policy: PolicyData) -> Dict:
        """生成政策预警"""
        alert = {
            "policy_id": policy.policy_id,
            "title": policy.title,
            "level": policy.level.value,
            "impact_score": policy.calculate_policy_score(),
            "affected_industries": policy.industry_impact,
            "alert_time": datetime.now(),
            "urgency": "high" if abs(policy.calculate_policy_score()) > 0.8 else "normal"
        }
        return alert

# 使用示例
if __name__ == "__main__":
    # 创建新闻数据示例
    news = NewsData(
        news_id="news_001",
        title="央行降准释放流动性",
        content="中国人民银行决定下调存款准备金率0.5个百分点...",
        publish_time=datetime.now(),
        crawl_time=datetime.now(),
        source="财联社",
        url="https://example.com/news/001",
        category=NewsCategory.POLICY,
        tags=["央行", "降准", "流动性"],
        stock_codes=["000001", "000002"],
        view_count=10000,
        sentiment_score=0.8,
        sentiment_type=SentimentType.POSITIVE
    )
    
    print(f"新闻影响力得分: {news.calculate_influence_score()}")
    print(f"是否为突发新闻: {news.is_breaking_news()}")
    
    # 创建政策数据示例
    policy = PolicyData(
        policy_id="policy_001",
        title="关于支持新能源汽车发展的若干意见",
        content="为促进新能源汽车产业发展...",
        publish_date=datetime.now(),
        issuer="国务院",
        level=PolicyLevel.NATIONAL,
        policy_type="产业政策",
        industry_impact=["新能源汽车", "电池", "充电桩"],
        impact_scope="全国",
        impact_degree="重大",
        sentiment_impact=SentimentType.POSITIVE,
        implementation_probability=0.9
    )
    
    print(f"政策影响得分: {policy.calculate_policy_score()}")