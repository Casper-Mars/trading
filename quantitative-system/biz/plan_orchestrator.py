"""方案生成编排器模块

负责协调AI分析、数据整合、方案生成等服务，完成投资方案的生成流程。
"""

import asyncio
from typing import Any

from pydantic import BaseModel

from biz.base_orchestrator import BaseOrchestrator, OrchestrationContext
from services.ai_service import AIService, AnalysisRequest
from services.backtest_service import BacktestRequest, BacktestService
from services.data_service import DataRequest, DataService
from services.plan_service import PlanRequest, PlanService
from utils.exceptions import OrchestrationError
from utils.logger import get_logger

logger = get_logger(__name__)


class PlanGenerationRequest(BaseModel):
    """方案生成请求模型"""

    symbols: list[str]  # 股票代码列表
    analysis_type: str = "comprehensive"  # 分析类型
    time_horizon: str = "medium_term"  # 投资时间范围
    risk_level: str = "moderate"  # 风险等级
    investment_amount: float | None = None  # 投资金额
    user_preferences: dict[str, Any] = {}  # 用户偏好
    market_conditions: dict[str, Any] = {}  # 市场条件


class PlanGenerationResponse(BaseModel):
    """方案生成响应模型"""

    plan_id: str
    plan_content: str  # Markdown格式的方案内容
    analysis_summary: dict[str, Any]  # AI分析摘要
    data_summary: dict[str, Any]  # 数据摘要
    backtest_summary: dict[str, Any]  # 回测结果摘要
    recommendations: list[dict[str, Any]]  # 投资建议
    risk_assessment: dict[str, Any]  # 风险评估
    execution_steps: list[str]  # 执行步骤
    created_at: str


class PlanOrchestrator(BaseOrchestrator):
    """方案生成编排器

    协调以下服务完成方案生成:
    1. DataService: 获取股票数据、市场数据、财务数据
    2. BacktestService: 进行回测分析，验证策略有效性
    3. AIService: 进行AI分析，生成投资建议
    4. PlanService: 格式化方案内容，保存方案

    编排流程:
    前置检查 → 数据收集 → 回测分析 → AI分析 → 方案生成 → 方案保存
    """

    def __init__(
        self,
        data_service: DataService,
        backtest_service: BacktestService,
        ai_service: AIService,
        plan_service: PlanService,
    ):
        """初始化方案生成编排器

        Args:
            data_service: 数据服务
            backtest_service: 回测服务
            ai_service: AI分析服务
            plan_service: 方案服务
        """
        super().__init__()
        self.data_service = data_service
        self.backtest_service = backtest_service
        self.ai_service = ai_service
        self.plan_service = plan_service

        logger.info("PlanOrchestrator initialized")

    async def _pre_check(
        self, request: PlanGenerationRequest, context: OrchestrationContext
    ) -> None:
        """前置检查

        Args:
            request: 方案生成请求
            context: 编排上下文

        Raises:
            OrchestrationError: 前置检查失败
        """
        logger.info(
            f"Starting pre-check for plan generation, request_id: {context.request_id}"
        )

        # 验证请求参数
        self._validate_request(request, context)

        # 检查股票代码
        if not request.symbols:
            raise OrchestrationError("Stock symbols are required")

        if len(request.symbols) > 10:
            raise OrchestrationError("Too many symbols, maximum 10 allowed")

        # 验证股票代码格式
        for symbol in request.symbols:
            if not symbol or len(symbol) < 2:
                raise OrchestrationError(f"Invalid symbol format: {symbol}")

        # 检查分析类型
        valid_analysis_types = ["basic", "comprehensive", "technical", "fundamental"]
        if request.analysis_type not in valid_analysis_types:
            raise OrchestrationError(f"Invalid analysis type: {request.analysis_type}")

        # 检查时间范围
        valid_time_horizons = ["short_term", "medium_term", "long_term"]
        if request.time_horizon not in valid_time_horizons:
            raise OrchestrationError(f"Invalid time horizon: {request.time_horizon}")

        # 检查风险等级
        valid_risk_levels = ["conservative", "moderate", "aggressive"]
        if request.risk_level not in valid_risk_levels:
            raise OrchestrationError(f"Invalid risk level: {request.risk_level}")

        # 检查投资金额
        if request.investment_amount is not None and request.investment_amount <= 0:
            raise OrchestrationError("Investment amount must be positive")

        # 设置上下文数据
        self._set_context_data("symbols", request.symbols, context)
        self._set_context_data("analysis_type", request.analysis_type, context)

        logger.info(
            f"Pre-check completed successfully, request_id: {context.request_id}"
        )

    async def _call_services(
        self, request: PlanGenerationRequest, context: OrchestrationContext
    ) -> dict[str, Any]:
        """调用服务

        Args:
            request: 方案生成请求
            context: 编排上下文

        Returns:
            服务调用结果字典

        Raises:
            OrchestrationError: 服务调用失败
        """
        logger.info(
            f"Starting service calls for plan generation, request_id: {context.request_id}"
        )

        results = {}

        try:
            # 1. 并行获取数据
            data_tasks = []

            # 获取股票基础数据
            data_request = DataRequest(
                symbols=request.symbols,
                data_types=["basic_info", "price_data", "financial_data"],
                time_range="1y",
            )
            data_tasks.append(
                self._safe_service_call(
                    "data_service",
                    lambda: self.data_service.get_integrated_data(data_request),
                    context,
                )
            )

            # 获取市场数据
            if request.market_conditions:
                market_request = DataRequest(
                    symbols=["market_index"],
                    data_types=["market_data"],
                    time_range="3m",
                )
                data_tasks.append(
                    self._safe_service_call(
                        "market_data",
                        lambda: self.data_service.get_market_data(market_request),
                        context,
                    )
                )

            # 等待数据获取完成
            data_results = await asyncio.gather(*data_tasks, return_exceptions=True)

            # 处理数据获取结果
            stock_data = None
            market_data = None

            for i, result in enumerate(data_results):
                if isinstance(result, Exception):
                    logger.error(f"Data service call {i} failed: {result!s}")
                    continue

                if i == 0:  # 股票数据
                    stock_data = result
                elif i == 1:  # 市场数据
                    market_data = result

            if stock_data is None:
                raise OrchestrationError("Failed to get stock data")

            results["stock_data"] = stock_data
            results["market_data"] = market_data

            # 添加回滚操作
            self._add_rollback_action(
                "cleanup_resources",
                {"data_cache_keys": [f"stock_data_{context.request_id}"]},
                context,
            )

            # 2. 回测分析
            backtest_request = BacktestRequest(
                symbols=request.symbols,
                strategy_name="multi_factor",  # 默认使用多因子策略
                start_date="2023-01-01",
                end_date="2024-01-01",
                initial_capital=request.investment_amount or 100000.0,
                stock_data=stock_data,
                market_data=market_data,
                risk_level=request.risk_level,
                time_horizon=request.time_horizon,
            )

            backtest_result = await self._safe_service_call(
                "backtest_service",
                lambda: self.backtest_service.run_backtest(backtest_request),
                context,
            )

            results["backtest_result"] = backtest_result

            # 3. AI分析
            analysis_request = AnalysisRequest(
                symbols=request.symbols,
                analysis_type=request.analysis_type,
                stock_data=stock_data,
                market_data=market_data,
                backtest_result=backtest_result,  # 添加回测结果
                user_preferences=request.user_preferences,
                time_horizon=request.time_horizon,
                risk_level=request.risk_level,
            )

            ai_analysis = await self._safe_service_call(
                "ai_service", lambda: self.ai_service.analyze(analysis_request), context
            )

            results["ai_analysis"] = ai_analysis

            # 4. 方案生成
            plan_request = PlanRequest(
                symbols=request.symbols,
                analysis_result=ai_analysis,
                stock_data=stock_data,
                market_data=market_data,
                backtest_result=backtest_result,  # 添加回测结果
                investment_amount=request.investment_amount,
                time_horizon=request.time_horizon,
                risk_level=request.risk_level,
                user_preferences=request.user_preferences,
            )

            plan_result = await self._safe_service_call(
                "plan_service",
                lambda: self.plan_service.generate_plan(plan_request),
                context,
            )

            results["plan_result"] = plan_result

            # 添加回滚操作
            self._add_rollback_action(
                "delete_data", {"plan_id": plan_result.plan_id}, context
            )

            logger.info(
                f"All service calls completed successfully, request_id: {context.request_id}"
            )

            return results

        except Exception as e:
            logger.error(
                f"Service calls failed: {e!s}, request_id: {context.request_id}"
            )
            raise OrchestrationError(f"Service orchestration failed: {e!s}") from e

    async def _aggregate_results(
        self, service_results: dict[str, Any], context: OrchestrationContext
    ) -> PlanGenerationResponse:
        """聚合结果

        Args:
            service_results: 服务调用结果
            context: 编排上下文

        Returns:
            聚合后的方案生成响应

        Raises:
            OrchestrationError: 结果聚合失败
        """
        logger.info(f"Starting result aggregation, request_id: {context.request_id}")

        try:
            # 获取服务结果
            stock_data = service_results.get("stock_data")
            market_data = service_results.get("market_data")
            backtest_result = service_results.get("backtest_result")
            ai_analysis = service_results.get("ai_analysis")
            plan_result = service_results.get("plan_result")

            if not backtest_result or not ai_analysis or not plan_result:
                raise OrchestrationError("Missing required service results")

            # 构建数据摘要
            data_summary = {
                "symbols_analyzed": self._get_context_data("symbols", context, []),
                "data_points": len(stock_data.get("price_data", []))
                if stock_data
                else 0,
                "market_data_available": market_data is not None,
                "backtest_completed": backtest_result is not None,
                "analysis_type": self._get_context_data(
                    "analysis_type", context, "unknown"
                ),
            }

            # 提取回测摘要
            backtest_summary = {
                "total_return": backtest_result.get("total_return", 0.0),
                "annual_return": backtest_result.get("annual_return", 0.0),
                "max_drawdown": backtest_result.get("max_drawdown", 0.0),
                "sharpe_ratio": backtest_result.get("sharpe_ratio", 0.0),
                "win_rate": backtest_result.get("win_rate", 0.0),
                "strategy_name": backtest_result.get("strategy_name", "unknown"),
                "backtest_period": f"{backtest_result.get('start_date', 'N/A')} - {backtest_result.get('end_date', 'N/A')}",
            }

            # 提取AI分析摘要
            analysis_summary = {
                "overall_sentiment": ai_analysis.get("sentiment", "neutral"),
                "confidence_score": ai_analysis.get("confidence", 0.5),
                "key_insights": ai_analysis.get("insights", []),
                "analysis_timestamp": ai_analysis.get("timestamp"),
            }

            # 提取投资建议
            recommendations = ai_analysis.get("recommendations", [])
            if not isinstance(recommendations, list):
                recommendations = []

            # 提取风险评估
            risk_assessment = ai_analysis.get(
                "risk_assessment",
                {
                    "overall_risk": "moderate",
                    "risk_factors": [],
                    "mitigation_strategies": [],
                },
            )

            # 生成执行步骤
            execution_steps = self._generate_execution_steps(
                ai_analysis, self._get_context_data("symbols", context, []), context
            )

            # 构建响应
            response = PlanGenerationResponse(
                plan_id=plan_result.plan_id,
                plan_content=plan_result.content,
                analysis_summary=analysis_summary,
                data_summary=data_summary,
                backtest_summary=backtest_summary,
                recommendations=recommendations,
                risk_assessment=risk_assessment,
                execution_steps=execution_steps,
                created_at=plan_result.created_at,
            )

            # 保存聚合结果到上下文
            self._set_context_data("final_response", response.dict(), context)

            logger.info(
                f"Result aggregation completed successfully, request_id: {context.request_id}"
            )

            return response

        except Exception as e:
            logger.error(
                f"Result aggregation failed: {e!s}, request_id: {context.request_id}"
            )
            raise OrchestrationError(f"Failed to aggregate results: {e!s}") from e

    def _generate_execution_steps(
        self,
        ai_analysis: dict[str, Any],
        symbols: list[str],
        context: OrchestrationContext,
    ) -> list[str]:
        """生成执行步骤

        Args:
            ai_analysis: AI分析结果
            symbols: 股票代码列表
            context: 编排上下文

        Returns:
            执行步骤列表
        """
        steps = []

        # 基础步骤
        steps.append("1. 审查投资方案和风险评估")
        steps.append("2. 确认投资金额和资金来源")

        # 根据推荐操作生成步骤
        recommendations = ai_analysis.get("recommendations", [])

        for i, rec in enumerate(recommendations[:3], 3):  # 最多3个推荐
            action = rec.get("action", "hold")
            symbol = rec.get("symbol", "unknown")

            if action == "buy":
                steps.append(f"{i}. 考虑买入 {symbol}")
            elif action == "sell":
                steps.append(f"{i}. 考虑卖出 {symbol}")
            elif action == "hold":
                steps.append(f"{i}. 继续持有 {symbol}")

        # 风险管理步骤
        steps.append(f"{len(steps) + 1}. 设置止损和止盈点")
        steps.append(f"{len(steps) + 1}. 定期监控投资组合表现")
        steps.append(f"{len(steps) + 1}. 根据市场变化调整策略")

        logger.debug(
            f"Generated {len(steps)} execution steps, request_id: {context.request_id}"
        )

        return steps

    async def _rollback_delete_data(
        self, action: dict[str, Any], context: OrchestrationContext
    ) -> None:
        """回滚数据删除操作

        Args:
            action: 回滚操作定义
            context: 编排上下文
        """
        action_data = action.get("data", {})
        plan_id = action_data.get("plan_id")

        if plan_id:
            try:
                # 删除已生成的方案
                await self.plan_service.delete_plan(plan_id)
                logger.info(
                    f"Plan deleted during rollback: {plan_id}, request_id: {context.request_id}"
                )
            except Exception as e:
                logger.error(
                    f"Failed to delete plan during rollback: {plan_id}, error: {e!s}"
                )

    async def _rollback_cleanup_resources(
        self, action: dict[str, Any], context: OrchestrationContext
    ) -> None:
        """回滚资源清理操作

        Args:
            action: 回滚操作定义
            context: 编排上下文
        """
        action_data = action.get("data", {})
        cache_keys = action_data.get("data_cache_keys", [])

        for cache_key in cache_keys:
            try:
                # 清理缓存数据
                await self.data_service.clear_cache(cache_key)
                logger.info(
                    f"Cache cleared during rollback: {cache_key}, request_id: {context.request_id}"
                )
            except Exception as e:
                logger.error(
                    f"Failed to clear cache during rollback: {cache_key}, error: {e!s}"
                )

    async def generate_plan(
        self,
        request: PlanGenerationRequest,
        request_id: str,
        user_id: str | None = None,
    ) -> PlanGenerationResponse:
        """生成投资方案

        Args:
            request: 方案生成请求
            request_id: 请求ID
            user_id: 用户ID

        Returns:
            方案生成响应

        Raises:
            OrchestrationError: 方案生成失败
        """
        # 创建编排上下文
        context = OrchestrationContext(request_id=request_id, user_id=user_id)

        # 执行编排流程
        result = await self.execute(request, context)

        if not result.success:
            raise OrchestrationError(f"Plan generation failed: {result.error}")

        return result.result

    async def get_plan_status(self, plan_id: str) -> dict[str, Any]:
        """获取方案状态

        Args:
            plan_id: 方案ID

        Returns:
            方案状态信息
        """
        try:
            plan = await self.plan_service.get_plan_by_id(plan_id)

            if not plan:
                return {
                    "plan_id": plan_id,
                    "status": "not_found",
                    "message": "Plan not found",
                }

            return {
                "plan_id": plan_id,
                "status": "completed",
                "created_at": plan.created_at,
                "symbols": plan.symbols,
                "analysis_type": plan.analysis_type,
            }

        except Exception as e:
            logger.error(f"Failed to get plan status: {plan_id}, error: {e!s}")
            return {"plan_id": plan_id, "status": "error", "message": str(e)}
