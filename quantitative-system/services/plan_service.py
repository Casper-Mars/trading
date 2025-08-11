"""方案生成服务模块

提供投资方案的生成、格式化、保存和查询功能。
"""

from datetime import datetime
from typing import Any

from pydantic import BaseModel, Field

from utils.exceptions import PlanServiceError
from utils.logger import get_logger

logger = get_logger(__name__)


class PlanRequest(BaseModel):
    """方案生成请求模型"""

    strategy_name: str = Field(..., description="策略名称")
    analysis_result: dict[str, Any] = Field(..., description="AI分析结果")
    backtest_result: dict[str, Any] = Field(..., description="回测结果")
    market_data: dict[str, Any] = Field(..., description="市场数据")
    user_preferences: dict[str, Any] | None = Field(None, description="用户偏好设置")


class PlanResponse(BaseModel):
    """方案生成响应模型"""

    plan_id: str = Field(..., description="方案ID")
    markdown_content: str = Field(..., description="Markdown格式的方案内容")
    created_at: datetime = Field(..., description="创建时间")
    strategy_name: str = Field(..., description="策略名称")
    summary: str = Field(..., description="方案摘要")


class PlanQueryRequest(BaseModel):
    """方案查询请求模型"""

    plan_id: str | None = Field(None, description="方案ID")
    strategy_name: str | None = Field(None, description="策略名称")
    start_date: datetime | None = Field(None, description="开始日期")
    end_date: datetime | None = Field(None, description="结束日期")
    limit: int = Field(10, description="查询限制数量")
    offset: int = Field(0, description="查询偏移量")


class PlanService:
    """方案生成服务

    负责将AI分析结果和回测数据格式化为结构化的投资方案,
    并提供方案的保存、查询和管理功能。
    """

    def __init__(self):
        """初始化方案服务"""
        self.plans_storage: dict[str, dict[str, Any]] = {}  # 简单内存存储,实际应使用数据库
        logger.info("PlanService initialized")

    async def generate_plan(self, request: PlanRequest) -> PlanResponse:
        """生成投资方案

        Args:
            request: 方案生成请求

        Returns:
            生成的方案响应

        Raises:
            PlanServiceError: 方案生成错误
        """
        try:
            logger.info(f"Generating plan for strategy: {request.strategy_name}")

            # 生成方案ID
            plan_id = self._generate_plan_id(request.strategy_name)

            # 格式化为Markdown
            markdown_content = self._format_to_markdown(request)

            # 生成摘要
            summary = self._generate_summary(request.analysis_result)

            # 创建响应
            response = PlanResponse(
                plan_id=plan_id,
                markdown_content=markdown_content,
                created_at=datetime.now(),
                strategy_name=request.strategy_name,
                summary=summary
            )

            # 保存方案
            await self._save_plan(response, request)

            logger.info(f"Plan generated successfully: {plan_id}")
            return response

        except Exception as e:
            logger.error(f"Failed to generate plan: {e!s}")
            raise PlanServiceError(f"方案生成失败: {e!s}") from e

    async def query_plans(self, request: PlanQueryRequest) -> list[PlanResponse]:
        """查询投资方案

        Args:
            request: 查询请求

        Returns:
            匹配的方案列表

        Raises:
            PlanServiceError: 查询错误
        """
        try:
            logger.info(f"Querying plans with filters: {request.model_dump()}")

            # 过滤方案
            filtered_plans = self._filter_plans(request)

            # 分页
            start_idx = request.offset
            end_idx = start_idx + request.limit
            paginated_plans = filtered_plans[start_idx:end_idx]

            logger.info(f"Found {len(paginated_plans)} plans")
            return paginated_plans

        except Exception as e:
            logger.error(f"Failed to query plans: {e!s}")
            raise PlanServiceError(f"方案查询失败: {e!s}") from e

    async def get_plan_by_id(self, plan_id: str) -> PlanResponse | None:
        """根据ID获取方案

        Args:
            plan_id: 方案ID

        Returns:
            方案响应或None

        Raises:
            PlanServiceError: 获取错误
        """
        try:
            logger.info(f"Getting plan by ID: {plan_id}")

            if plan_id not in self.plans_storage:
                logger.warning(f"Plan not found: {plan_id}")
                return None

            plan_data = self.plans_storage[plan_id]
            return PlanResponse(**plan_data["response"])

        except Exception as e:
            logger.error(f"Failed to get plan: {e!s}")
            raise PlanServiceError(f"获取方案失败: {e!s}") from e

    def _generate_plan_id(self, strategy_name: str) -> str:
        """生成方案ID

        Args:
            strategy_name: 策略名称

        Returns:
            生成的方案ID
        """
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        return f"plan_{strategy_name}_{timestamp}"

    def _format_to_markdown(self, request: PlanRequest) -> str:
        """将分析结果格式化为Markdown

        Args:
            request: 方案请求

        Returns:
            Markdown格式的内容
        """
        analysis = request.analysis_result
        backtest = request.backtest_result

        markdown_content = f"""# 投资方案报告

## 基本信息

| 项目 | 内容 |
|------|------|
| 策略名称 | {request.strategy_name} |
| 生成时间 | {datetime.now().strftime('%Y-%m-%d %H:%M:%S')} |
| 分析类型 | {analysis.get('analysis_type', '综合分析')} |

## 策略分析

### 策略表现评估

{self._format_strategy_analysis(analysis.get('strategy_analysis', {}))}

### 回测结果概览

| 指标 | 数值 |
|------|------|
| 总收益率 | {backtest.get('total_return', 'N/A')} |
| 年化收益率 | {backtest.get('annual_return', 'N/A')} |
| 最大回撤 | {backtest.get('max_drawdown', 'N/A')} |
| 夏普比率 | {backtest.get('sharpe_ratio', 'N/A')} |
| 胜率 | {backtest.get('win_rate', 'N/A')} |

## 风险评估

### 风险等级

{self._format_risk_assessment(analysis.get('risk_assessment', {}))}

### 风险控制建议

{self._format_risk_suggestions(analysis.get('risk_assessment', {}))}

## 操作建议

### 具体建议

{self._format_operation_suggestions(analysis.get('operation_suggestions', []))}

### 建议优先级

{self._format_suggestion_priorities(analysis.get('operation_suggestions', []))}

## 市场展望

### 短期预期

{self._format_market_outlook(analysis.get('market_outlook', {}))}

### 关键因子

{self._format_key_factors(analysis.get('key_factors', []))}

## 风险警告

{self._format_warnings(analysis.get('warnings', []))}

## 置信度评分

**分析可信度**: {analysis.get('confidence_score', 'N/A')}

---

*本报告由AI量化分析系统自动生成,仅供参考,投资有风险,决策需谨慎。*
"""

        return markdown_content

    def _format_strategy_analysis(self, strategy_analysis: dict[str, Any]) -> str:
        """格式化策略分析"""
        if not strategy_analysis:
            return "暂无策略分析数据"

        content = []
        if "performance" in strategy_analysis:
            content.append(f"**表现评估**: {strategy_analysis['performance']}")
        if "advantages" in strategy_analysis:
            content.append(f"**优势**: {strategy_analysis['advantages']}")
        if "disadvantages" in strategy_analysis:
            content.append(f"**劣势**: {strategy_analysis['disadvantages']}")
        if "improvements" in strategy_analysis:
            content.append(f"**改进建议**: {strategy_analysis['improvements']}")

        return "\n\n".join(content) if content else "暂无详细分析"

    def _format_risk_assessment(self, risk_assessment: dict[str, Any]) -> str:
        """格式化风险评估"""
        if not risk_assessment:
            return "暂无风险评估数据"

        content = []
        if "level" in risk_assessment:
            content.append(f"**风险等级**: {risk_assessment['level']}")
        if "main_risks" in risk_assessment:
            risks = risk_assessment['main_risks']
            if isinstance(risks, list):
                risk_list = "\n".join([f"- {risk}" for risk in risks])
                content.append(f"**主要风险**:\n{risk_list}")

        return "\n\n".join(content) if content else "暂无风险评估"

    def _format_risk_suggestions(self, risk_assessment: dict[str, Any]) -> str:
        """格式化风险控制建议"""
        if not risk_assessment or "control_suggestions" not in risk_assessment:
            return "暂无风险控制建议"

        suggestions = risk_assessment["control_suggestions"]
        if isinstance(suggestions, list):
            return "\n".join([f"- {suggestion}" for suggestion in suggestions])
        return str(suggestions)

    def _format_operation_suggestions(self, suggestions: list[dict[str, Any]]) -> str:
        """格式化操作建议"""
        if not suggestions:
            return "暂无操作建议"

        content = []
        for i, suggestion in enumerate(suggestions, 1):
            if isinstance(suggestion, dict):
                action = suggestion.get("action", "未知操作")
                reason = suggestion.get("reason", "")
                content.append(f"{i}. **{action}**{f': {reason}' if reason else ''}")
            else:
                content.append(f"{i}. {suggestion}")

        return "\n".join(content)

    def _format_suggestion_priorities(self, suggestions: list[dict[str, Any]]) -> str:
        """格式化建议优先级"""
        if not suggestions:
            return "暂无优先级信息"

        high_priority = []
        medium_priority = []
        low_priority = []

        for suggestion in suggestions:
            if isinstance(suggestion, dict):
                priority = suggestion.get("priority", "medium")
                action = suggestion.get("action", "未知操作")

                if priority == "high":
                    high_priority.append(action)
                elif priority == "low":
                    low_priority.append(action)
                else:
                    medium_priority.append(action)

        content = []
        if high_priority:
            content.append(f"**高优先级**: {', '.join(high_priority)}")
        if medium_priority:
            content.append(f"**中优先级**: {', '.join(medium_priority)}")
        if low_priority:
            content.append(f"**低优先级**: {', '.join(low_priority)}")

        return "\n".join(content) if content else "暂无优先级分类"

    def _format_market_outlook(self, market_outlook: dict[str, Any]) -> str:
        """格式化市场展望"""
        if not market_outlook:
            return "暂无市场展望数据"

        content = []
        if "short_term" in market_outlook:
            content.append(f"**短期预期**: {market_outlook['short_term']}")
        if "factors" in market_outlook:
            factors = market_outlook['factors']
            if isinstance(factors, list):
                factor_list = "\n".join([f"- {factor}" for factor in factors])
                content.append(f"**影响因素**:\n{factor_list}")

        return "\n\n".join(content) if content else "暂无市场展望"

    def _format_key_factors(self, key_factors: list[str]) -> str:
        """格式化关键因子"""
        if not key_factors:
            return "暂无关键因子数据"

        return "\n".join([f"- {factor}" for factor in key_factors])

    def _format_warnings(self, warnings: list[str]) -> str:
        """格式化风险警告"""
        if not warnings:
            return "暂无特别风险警告"

        return "\n".join([f"⚠️ {warning}" for warning in warnings])

    def _generate_summary(self, analysis_result: dict[str, Any]) -> str:
        """生成方案摘要

        Args:
            analysis_result: AI分析结果

        Returns:
            方案摘要
        """
        confidence = analysis_result.get("confidence_score", "未知")
        risk_level = analysis_result.get("risk_assessment", {}).get("level", "未知")

        summary_parts = []

        # 添加置信度信息
        summary_parts.append(f"置信度: {confidence}")

        # 添加风险等级
        summary_parts.append(f"风险等级: {risk_level}")

        # 添加主要建议数量
        suggestions = analysis_result.get("operation_suggestions", [])
        if suggestions:
            summary_parts.append(f"操作建议: {len(suggestions)}条")

        return " | ".join(summary_parts)

    async def _save_plan(self, response: PlanResponse, request: PlanRequest) -> None:
        """保存方案到存储

        Args:
            response: 方案响应
            request: 原始请求
        """
        plan_data = {
            "response": response.model_dump(),
            "request": request.model_dump(),
            "created_at": response.created_at
        }

        self.plans_storage[response.plan_id] = plan_data
        logger.info(f"Plan saved: {response.plan_id}")

    def _filter_plans(self, request: PlanQueryRequest) -> list[PlanResponse]:
        """过滤方案

        Args:
            request: 查询请求

        Returns:
            过滤后的方案列表
        """
        filtered_plans = []

        for plan_id, plan_data in self.plans_storage.items():
            response_data = plan_data["response"]
            created_at = plan_data["created_at"]

            # 按ID过滤
            if request.plan_id and plan_id != request.plan_id:
                continue

            # 按策略名称过滤
            if request.strategy_name and response_data["strategy_name"] != request.strategy_name:
                continue

            # 按日期范围过滤
            if request.start_date and created_at < request.start_date:
                continue
            if request.end_date and created_at > request.end_date:
                continue

            filtered_plans.append(PlanResponse(**response_data))

        # 按创建时间倒序排序
        filtered_plans.sort(key=lambda x: x.created_at, reverse=True)

        return filtered_plans
