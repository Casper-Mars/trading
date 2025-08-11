"""AI分析服务模块

基于阿里百炼(DashScope)提供AI分析功能，包括策略分析、风险评估和操作建议。
"""

import json
import time
from typing import Any

import dashscope
from dashscope import Generation
from loguru import logger
from pydantic import BaseModel, Field

from config.settings import settings
from utils.exceptions import ExternalServiceError


class AIAnalysisRequest(BaseModel):
    """AI分析请求模型"""

    strategy_name: str = Field(..., description="策略名称")
    backtest_results: dict[str, Any] = Field(..., description="回测结果")
    factor_scores: dict[str, float] = Field(..., description="因子评分")
    market_data: dict[str, Any] = Field(..., description="市场数据")
    position_data: dict[str, Any] | None = Field(None, description="持仓数据")
    analysis_type: str = Field("comprehensive", description="分析类型:strategy/risk/recommendation/comprehensive")


class AIAnalysisResponse(BaseModel):
    """AI分析响应模型"""

    strategy_analysis: str = Field(..., description="策略分析")
    risk_assessment: str = Field(..., description="风险评估")
    operation_suggestions: list[str] = Field(..., description="操作建议")
    confidence_score: float = Field(..., description="置信度评分(0-1)")
    market_outlook: str = Field(..., description="市场展望")
    key_factors: list[str] = Field(..., description="关键因子")
    warnings: list[str] = Field(default_factory=list, description="风险警告")
    timestamp: float = Field(default_factory=time.time, description="分析时间戳")


class AIService:
    """AI分析服务

    基于阿里百炼(DashScope)提供智能分析功能:
    - 策略分析：评估策略表现和优化建议
    - 风险评估：识别潜在风险和风险控制建议
    - 操作建议：基于分析结果提供具体操作建议
    - 失败降级：超时或失败时返回规则引擎结果
    """

    def __init__(self):
        """初始化AI服务"""
        self.api_key = settings.dashscope_api_key
        self.timeout = settings.dashscope_timeout
        self.max_retries = settings.dashscope_max_retries
        self.model = "qwen-plus"  # 使用通义千问Plus模型

        # 配置DashScope
        if self.api_key:
            dashscope.api_key = self.api_key
        else:
            logger.warning("DashScope API key not configured, AI service will use fallback mode")

        # 缓存最近的分析结果用于降级
        self._last_analysis: AIAnalysisResponse | None = None

        logger.info(f"AIService initialized with model: {self.model}")

    async def analyze(self, request: AIAnalysisRequest) -> AIAnalysisResponse:
        """执行AI分析

        Args:
            request: AI分析请求

        Returns:
            AI分析响应

        Raises:
            AIServiceError: AI服务错误
        """
        try:
            logger.info(f"Starting AI analysis for strategy: {request.strategy_name}")

            # 检查API密钥
            if not self.api_key:
                logger.warning("API key not available, using fallback analysis")
                return self._fallback_analysis(request)

            # 构建提示词
            prompt = self._build_prompt(request)

            # 调用DashScope API
            response = await self._call_dashscope_with_retry(prompt)

            # 解析响应
            analysis_result = self._parse_response(response, request)

            # 缓存结果
            self._last_analysis = analysis_result

            logger.info(f"AI analysis completed for strategy: {request.strategy_name}")
            return analysis_result

        except Exception as e:
            logger.error(f"AI analysis failed: {e!s}")
            # 降级处理
            return self._fallback_analysis(request)

    def _build_prompt(self, request: AIAnalysisRequest) -> str:
        """构建AI分析提示词

        Args:
            request: 分析请求

        Returns:
            构建的提示词
        """
        # 提取关键数据
        backtest_summary = self._extract_backtest_summary(request.backtest_results)
        factor_summary = self._extract_factor_summary(request.factor_scores)
        market_summary = self._extract_market_summary(request.market_data)

        prompt = f"""
你是一位资深的量化交易分析师,请基于以下数据进行专业分析:

## 策略信息
策略名称:{request.strategy_name}
分析类型:{request.analysis_type}

## 回测结果
{backtest_summary}

## 因子评分
{factor_summary}

## 市场数据
{market_summary}

## 分析要求
请提供以下分析内容,并以JSON格式返回:

1. **策略分析** (strategy_analysis):
   - 策略表现评估
   - 优势和劣势分析
   - 改进建议

2. **风险评估** (risk_assessment):
   - 主要风险识别
   - 风险等级评估
   - 风险控制建议

3. **操作建议** (operation_suggestions):
   - 具体操作建议列表
   - 建议的优先级

4. **置信度评分** (confidence_score):
   - 0-1之间的数值,表示分析的可信度

5. **市场展望** (market_outlook):
   - 短期市场预期
   - 影响因素分析

6. **关键因子** (key_factors):
   - 最重要的影响因子列表

7. **风险警告** (warnings):
   - 需要特别注意的风险点列表

请确保返回的JSON格式正确,所有字段都包含有意义的内容。
        """.strip()

        return prompt

    def _extract_backtest_summary(self, backtest_results: dict[str, Any]) -> str:
        """提取回测结果摘要"""
        try:
            summary_parts = []

            # 基础指标
            if "total_return" in backtest_results:
                summary_parts.append(f"总收益率: {backtest_results['total_return']:.2%}")

            if "sharpe_ratio" in backtest_results:
                summary_parts.append(f"夏普比率: {backtest_results['sharpe_ratio']:.3f}")

            if "max_drawdown" in backtest_results:
                summary_parts.append(f"最大回撤: {backtest_results['max_drawdown']:.2%}")

            if "win_rate" in backtest_results:
                summary_parts.append(f"胜率: {backtest_results['win_rate']:.2%}")

            if "total_trades" in backtest_results:
                summary_parts.append(f"交易次数: {backtest_results['total_trades']}")

            return "\n".join(summary_parts) if summary_parts else "回测数据不完整"

        except Exception as e:
            logger.warning(f"Failed to extract backtest summary: {e}")
            return "回测数据解析失败"

    def _extract_factor_summary(self, factor_scores: dict[str, float]) -> str:
        """提取因子评分摘要"""
        try:
            summary_parts = []

            for factor_name, score in factor_scores.items():
                summary_parts.append(f"{factor_name}: {score:.3f}")

            return "\n".join(summary_parts) if summary_parts else "因子评分数据为空"

        except Exception as e:
            logger.warning(f"Failed to extract factor summary: {e}")
            return "因子评分数据解析失败"

    def _extract_market_summary(self, market_data: dict[str, Any]) -> str:
        """提取市场数据摘要"""
        try:
            summary_parts = []

            if "market_trend" in market_data:
                summary_parts.append(f"市场趋势: {market_data['market_trend']}")

            if "volatility" in market_data:
                summary_parts.append(f"波动率: {market_data['volatility']:.3f}")

            if "volume" in market_data:
                summary_parts.append(f"成交量: {market_data['volume']}")

            return "\n".join(summary_parts) if summary_parts else "市场数据不完整"

        except Exception as e:
            logger.warning(f"Failed to extract market summary: {e}")
            return "市场数据解析失败"

    async def _call_dashscope_with_retry(self, prompt: str) -> str:
        """带重试的DashScope API调用

        Args:
            prompt: 提示词

        Returns:
            API响应内容

        Raises:
            ExternalServiceError: 外部服务错误
        """
        last_error = None

        for attempt in range(self.max_retries):
            try:
                logger.debug(f"Calling DashScope API, attempt {attempt + 1}/{self.max_retries}")

                response = Generation.call(
                    model=self.model,
                    prompt=prompt,
                    max_tokens=2000,
                    temperature=0.7,
                    top_p=0.9,
                    timeout=self.timeout
                )

                if response.status_code == 200:
                    content = response.output.text
                    logger.debug("DashScope API call successful")
                    return content
                else:
                    error_msg = f"DashScope API error: {response.status_code} - {response.message}"
                    logger.warning(error_msg)
                    last_error = ExternalServiceError(error_msg)

            except Exception as e:
                error_msg = f"DashScope API call failed: {e!s}"
                logger.warning(error_msg)
                last_error = ExternalServiceError(error_msg)

            # 重试前等待
            if attempt < self.max_retries - 1:
                wait_time = 2 ** attempt  # 指数退避
                logger.info(f"Retrying in {wait_time} seconds...")
                time.sleep(wait_time)

        # 所有重试都失败
        raise last_error or ExternalServiceError("DashScope API call failed after all retries")

    def _parse_response(self, response_text: str, request: AIAnalysisRequest) -> AIAnalysisResponse:
        """解析AI响应

        Args:
            response_text: AI响应文本
            request: 原始请求

        Returns:
            解析后的分析结果
        """
        try:
            # 尝试解析JSON响应
            # 提取JSON部分（可能包含在markdown代码块中）
            json_text = self._extract_json_from_text(response_text)

            if json_text:
                parsed_data = json.loads(json_text)

                # 验证必需字段并设置默认值
                return AIAnalysisResponse(
                    strategy_analysis=parsed_data.get("strategy_analysis", "策略分析数据解析失败"),
                    risk_assessment=parsed_data.get("risk_assessment", "风险评估数据解析失败"),
                    operation_suggestions=parsed_data.get("operation_suggestions", ["建议数据解析失败"]),
                    confidence_score=max(0.0, min(1.0, parsed_data.get("confidence_score", 0.5))),
                    market_outlook=parsed_data.get("market_outlook", "市场展望数据解析失败"),
                    key_factors=parsed_data.get("key_factors", ["关键因子解析失败"]),
                    warnings=parsed_data.get("warnings", [])
                )
            else:
                # JSON解析失败，使用文本解析
                return self._parse_text_response(response_text)

        except Exception as e:
            logger.warning(f"Failed to parse AI response: {e}")
            return self._parse_text_response(response_text)

    def _extract_json_from_text(self, text: str) -> str | None:
        """从文本中提取JSON部分"""
        try:
            # 查找JSON代码块
            import re

            # 匹配```json...```格式
            json_pattern = r'```json\s*([\s\S]*?)\s*```'
            match = re.search(json_pattern, text, re.IGNORECASE)
            if match:
                return match.group(1).strip()

            # 匹配```...```格式
            code_pattern = r'```\s*([\s\S]*?)\s*```'
            match = re.search(code_pattern, text)
            if match:
                content = match.group(1).strip()
                # 检查是否是JSON格式
                if content.startswith('{') and content.endswith('}'):
                    return content

            # 直接查找JSON对象
            json_pattern = r'\{[\s\S]*\}'
            match = re.search(json_pattern, text)
            if match:
                return match.group(0)

            return None

        except Exception:
            return None

    def _parse_text_response(self, response_text: str) -> AIAnalysisResponse:
        """解析纯文本响应"""
        # 简单的文本解析逻辑
        return AIAnalysisResponse(
            strategy_analysis=f"基于AI分析:{response_text[:200]}...",
            risk_assessment="基于文本分析的风险评估",
            operation_suggestions=["建议基于AI文本分析结果进行操作"],
            confidence_score=0.6,
            market_outlook="市场展望需要进一步分析",
            key_factors=["技术面因子", "基本面因子"],
            warnings=["AI响应解析不完整,建议人工复核"]
        )

    def _fallback_analysis(self, request: AIAnalysisRequest) -> AIAnalysisResponse:
        """降级分析（规则引擎）

        当AI服务不可用时，使用规则引擎提供基础分析

        Args:
            request: 分析请求

        Returns:
            降级分析结果
        """
        logger.info("Using fallback analysis due to AI service unavailability")

        # 如果有缓存的分析结果，优先使用
        if self._last_analysis:
            logger.info("Using cached analysis result")
            cached_result = self._last_analysis.model_copy()
            cached_result.timestamp = time.time()
            cached_result.warnings.append("使用缓存的AI分析结果")
            return cached_result

        # 基于规则的简单分析
        backtest_results = request.backtest_results
        factor_scores = request.factor_scores

        # 策略分析
        strategy_analysis = self._rule_based_strategy_analysis(backtest_results)

        # 风险评估
        risk_assessment = self._rule_based_risk_assessment(backtest_results)

        # 操作建议
        operation_suggestions = self._rule_based_operation_suggestions(factor_scores, backtest_results)

        # 置信度评分（规则引擎置信度较低）
        confidence_score = 0.4

        return AIAnalysisResponse(
            strategy_analysis=strategy_analysis,
            risk_assessment=risk_assessment,
            operation_suggestions=operation_suggestions,
            confidence_score=confidence_score,
            market_outlook="基于规则引擎的市场分析,建议结合人工判断",
            key_factors=list(factor_scores.keys())[:5],  # 取前5个因子
            warnings=["AI服务不可用,使用规则引擎降级分析", "建议人工复核分析结果"]
        )

    def _rule_based_strategy_analysis(self, backtest_results: dict[str, Any]) -> str:
        """基于规则的策略分析"""
        analysis_parts = []

        # 收益率分析
        total_return = backtest_results.get("total_return", 0)
        if total_return > 0.2:
            analysis_parts.append("策略表现优秀,总收益率超过20%")
        elif total_return > 0.1:
            analysis_parts.append("策略表现良好,总收益率超过10%")
        elif total_return > 0:
            analysis_parts.append("策略表现一般,收益率为正")
        else:
            analysis_parts.append("策略表现不佳,出现亏损")

        # 夏普比率分析
        sharpe_ratio = backtest_results.get("sharpe_ratio", 0)
        if sharpe_ratio > 2:
            analysis_parts.append("风险调整收益优秀,夏普比率大于2")
        elif sharpe_ratio > 1:
            analysis_parts.append("风险调整收益良好,夏普比率大于1")
        else:
            analysis_parts.append("风险调整收益需要改善")

        # 最大回撤分析
        max_drawdown = backtest_results.get("max_drawdown", 0)
        if abs(max_drawdown) < 0.1:
            analysis_parts.append("回撤控制良好,最大回撤小于10%")
        elif abs(max_drawdown) < 0.2:
            analysis_parts.append("回撤控制一般,最大回撤小于20%")
        else:
            analysis_parts.append("回撤较大,需要加强风险控制")

        return "\n".join(analysis_parts)

    def _rule_based_risk_assessment(self, backtest_results: dict[str, Any]) -> str:
        """基于规则的风险评估"""
        risk_parts = []

        # 回撤风险
        max_drawdown = backtest_results.get("max_drawdown", 0)
        if abs(max_drawdown) > 0.3:
            risk_parts.append("高风险:最大回撤超过30%,存在重大损失风险")
        elif abs(max_drawdown) > 0.2:
            risk_parts.append("中等风险:最大回撤超过20%,需要关注风险控制")
        else:
            risk_parts.append("低风险:回撤控制在可接受范围内")

        # 波动率风险
        volatility = backtest_results.get("volatility", 0)
        if volatility > 0.3:
            risk_parts.append("高波动风险:策略波动率较高")
        elif volatility > 0.2:
            risk_parts.append("中等波动风险:策略波动率适中")
        else:
            risk_parts.append("低波动风险:策略相对稳定")

        return "\n".join(risk_parts)

    def _rule_based_operation_suggestions(self, factor_scores: dict[str, float], backtest_results: dict[str, Any]) -> list[str]:
        """基于规则的操作建议"""
        suggestions = []

        # 基于因子评分的建议
        avg_score = sum(factor_scores.values()) / len(factor_scores) if factor_scores else 0

        if avg_score > 0.7:
            suggestions.append("因子评分较高,建议适当增加仓位")
        elif avg_score > 0.5:
            suggestions.append("因子评分中等,建议保持当前仓位")
        else:
            suggestions.append("因子评分较低,建议减少仓位或观望")

        # 基于回测结果的建议
        sharpe_ratio = backtest_results.get("sharpe_ratio", 0)
        if sharpe_ratio < 1:
            suggestions.append("夏普比率偏低,建议优化策略参数")

        max_drawdown = backtest_results.get("max_drawdown", 0)
        if abs(max_drawdown) > 0.2:
            suggestions.append("回撤较大,建议加强止损机制")

        # 通用建议
        suggestions.extend([
            "建议定期监控策略表现",
            "建议结合市场环境调整策略",
            "建议进行风险管理评估"
        ])

        return suggestions

    def get_service_status(self) -> dict[str, Any]:
        """获取服务状态

        Returns:
            服务状态信息
        """
        return {
            "service_name": "AIService",
            "api_key_configured": bool(self.api_key),
            "model": self.model,
            "timeout": self.timeout,
            "max_retries": self.max_retries,
            "has_cached_analysis": self._last_analysis is not None,
            "status": "healthy" if self.api_key else "degraded"
        }
