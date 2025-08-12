"""回测分析编排器

回测任务执行流程的业务编排器，负责协调策略配置验证、数据准备、回测执行、结果分析和报告生成。
"""

from typing import Any

from pydantic import BaseModel

from biz.base_orchestrator import BaseOrchestrator, OrchestrationContext
from services.backtest_service import BacktestRequest, BacktestService
from services.data_service import DataService
from strategies.strategy_registry import StrategyRegistry
from utils.exceptions import OrchestrationError
from utils.logger import get_logger

logger = get_logger(__name__)


class BacktestExecutionRequest(BaseModel):
    """回测执行请求模型"""

    symbols: list[str]  # 股票代码列表
    strategy_name: str  # 策略名称
    start_date: str  # 回测开始日期
    end_date: str  # 回测结束日期
    initial_capital: float = 100000.0  # 初始资金
    strategy_params: dict[str, Any] = {}  # 策略参数
    risk_level: str = "moderate"  # 风险等级
    time_horizon: str = "medium_term"  # 投资时间范围
    parallel_execution: bool = False  # 是否并行执行
    generate_report: bool = True  # 是否生成详细报告


class BacktestExecutionResponse(BaseModel):
    """回测执行响应模型"""

    task_id: str
    strategy_name: str
    execution_status: str  # running, completed, failed
    backtest_result: dict[str, Any] | None = None  # 回测结果
    performance_metrics: dict[str, Any] = {}  # 性能指标
    risk_metrics: dict[str, Any] = {}  # 风险指标
    execution_summary: dict[str, Any] = {}  # 执行摘要
    report_content: str | None = None  # 报告内容
    created_at: str
    completed_at: str | None = None


class BacktestOrchestrator(BaseOrchestrator):
    """回测分析编排器

    协调以下服务完成回测分析:
    1. StrategyRegistry: 策略配置验证和动态选择
    2. DataService: 数据准备和预处理
    3. BacktestService: 回测执行和结果分析

    编排流程:
    前置检查 → 策略验证 → 数据准备 → 回测执行 → 结果分析 → 报告生成
    """

    def __init__(
        self,
        data_service: DataService,
        backtest_service: BacktestService,
        strategy_registry: StrategyRegistry,
    ):
        """初始化回测分析编排器

        Args:
            data_service: 数据服务
            backtest_service: 回测服务
            strategy_registry: 策略注册器
        """
        super().__init__()
        self.data_service = data_service
        self.backtest_service = backtest_service
        self.strategy_registry = strategy_registry

        logger.info("BacktestOrchestrator initialized")

    async def execute_backtest(
        self, request: BacktestExecutionRequest
    ) -> BacktestExecutionResponse:
        """执行回测分析

        Args:
            request: 回测执行请求

        Returns:
            回测执行响应

        Raises:
            OrchestrationError: 回测执行失败
        """
        context = OrchestrationContext(
            operation="backtest_execution", request_data=request.dict()
        )

        logger.info(f"Starting backtest execution, request_id: {context.request_id}")

        try:
            # 执行编排流程
            result = await self.execute(request, context)
            return result

        except Exception as e:
            logger.error(
                f"Backtest execution failed: {e!s}, request_id: {context.request_id}"
            )
            raise OrchestrationError(f"Backtest execution failed: {e!s}") from e

    async def _pre_check(
        self, request: BacktestExecutionRequest, context: OrchestrationContext
    ) -> bool:
        """前置检查

        Args:
            request: 回测执行请求
            context: 编排上下文

        Returns:
            检查是否通过

        Raises:
            OrchestrationError: 前置检查失败
        """
        logger.info(f"Starting pre-check, request_id: {context.request_id}")

        try:
            # 1. 验证股票代码
            if not request.symbols:
                raise OrchestrationError("Stock symbols cannot be empty")

            # 2. 验证日期范围
            if request.start_date >= request.end_date:
                raise OrchestrationError("Start date must be before end date")

            # 3. 验证初始资金
            if request.initial_capital <= 0:
                raise OrchestrationError("Initial capital must be positive")

            # 4. 验证策略是否存在
            if not self.strategy_registry.is_strategy_registered(request.strategy_name):
                available_strategies = self.strategy_registry.get_available_strategies()
                raise OrchestrationError(
                    f"Strategy '{request.strategy_name}' not found. "
                    f"Available strategies: {available_strategies}"
                )

            # 5. 验证策略参数
            strategy_info = self.strategy_registry.get_strategy_info(
                request.strategy_name
            )
            if strategy_info and "param_ranges" in strategy_info:
                self._validate_strategy_params(
                    request.strategy_params, strategy_info["param_ranges"]
                )

            # 保存验证结果到上下文
            self._set_context_data("symbols", request.symbols, context)
            self._set_context_data("strategy_name", request.strategy_name, context)
            self._set_context_data(
                "date_range", f"{request.start_date} - {request.end_date}", context
            )

            logger.info(
                f"Pre-check completed successfully, request_id: {context.request_id}"
            )
            return True

        except Exception as e:
            logger.error(f"Pre-check failed: {e!s}, request_id: {context.request_id}")
            raise OrchestrationError(f"Pre-check failed: {e!s}") from e

    async def _call_services(
        self, request: BacktestExecutionRequest, context: OrchestrationContext
    ) -> dict[str, Any]:
        """调用服务

        Args:
            request: 回测执行请求
            context: 编排上下文

        Returns:
            服务调用结果

        Raises:
            OrchestrationError: 服务调用失败
        """
        logger.info(f"Starting service calls, request_id: {context.request_id}")

        results = {}

        try:
            # 1. 数据准备
            logger.info(f"Preparing data for symbols: {request.symbols}")

            # 获取股票数据
            stock_data = await self._safe_service_call(
                "data_service",
                lambda: self.data_service.get_stock_data(
                    symbols=request.symbols,
                    start_date=request.start_date,
                    end_date=request.end_date,
                ),
                context,
            )

            # 获取市场数据
            market_data = await self._safe_service_call(
                "data_service",
                lambda: self.data_service.get_market_data(
                    start_date=request.start_date, end_date=request.end_date
                ),
                context,
            )

            if stock_data is None:
                raise OrchestrationError("Failed to get stock data")

            results["stock_data"] = stock_data
            results["market_data"] = market_data

            # 添加回滚操作
            self._add_rollback_action(
                "cleanup_data_cache",
                {
                    "cache_keys": [
                        f"stock_data_{context.request_id}",
                        f"market_data_{context.request_id}",
                    ]
                },
                context,
            )

            # 2. 回测执行
            logger.info(f"Executing backtest with strategy: {request.strategy_name}")

            backtest_request = BacktestRequest(
                symbols=request.symbols,
                strategy_name=request.strategy_name,
                start_date=request.start_date,
                end_date=request.end_date,
                initial_capital=request.initial_capital,
                stock_data=stock_data,
                market_data=market_data,
                strategy_params=request.strategy_params,
                risk_level=request.risk_level,
                time_horizon=request.time_horizon,
            )

            if request.parallel_execution:
                # 并行执行（如果支持多个策略或多个股票组合）
                backtest_result = await self._execute_parallel_backtest(
                    backtest_request, context
                )
            else:
                # 串行执行
                backtest_result = await self._safe_service_call(
                    "backtest_service",
                    lambda: self.backtest_service.run_backtest(backtest_request),
                    context,
                )

            results["backtest_result"] = backtest_result

            # 3. 结果分析
            if backtest_result:
                performance_metrics = self._analyze_performance(backtest_result)
                risk_metrics = self._analyze_risk(backtest_result)

                results["performance_metrics"] = performance_metrics
                results["risk_metrics"] = risk_metrics

            # 4. 报告生成（如果需要）
            if request.generate_report and backtest_result:
                report_content = self._generate_report(
                    backtest_result,
                    results.get("performance_metrics", {}),
                    results.get("risk_metrics", {}),
                    request,
                    context,
                )
                results["report_content"] = report_content

            logger.info(
                f"Service calls completed successfully, request_id: {context.request_id}"
            )
            return results

        except Exception as e:
            logger.error(
                f"Service calls failed: {e!s}, request_id: {context.request_id}"
            )
            raise OrchestrationError(f"Service orchestration failed: {e!s}") from e

    async def _aggregate_results(
        self, service_results: dict[str, Any], context: OrchestrationContext
    ) -> BacktestExecutionResponse:
        """聚合结果

        Args:
            service_results: 服务调用结果
            context: 编排上下文

        Returns:
            聚合后的回测执行响应

        Raises:
            OrchestrationError: 结果聚合失败
        """
        logger.info(f"Starting result aggregation, request_id: {context.request_id}")

        try:
            # 获取服务结果
            backtest_result = service_results.get("backtest_result")
            performance_metrics = service_results.get("performance_metrics", {})
            risk_metrics = service_results.get("risk_metrics", {})
            report_content = service_results.get("report_content")

            if not backtest_result:
                raise OrchestrationError("Missing backtest result")

            # 构建执行摘要
            execution_summary = {
                "symbols_count": len(self._get_context_data("symbols", context, [])),
                "strategy_used": self._get_context_data(
                    "strategy_name", context, "unknown"
                ),
                "date_range": self._get_context_data("date_range", context, "unknown"),
                "execution_time": context.execution_time,
                "data_points": len(
                    service_results.get("stock_data", {}).get("price_data", [])
                ),
                "trades_count": backtest_result.get("trades_count", 0),
                "success_rate": backtest_result.get("win_rate", 0.0),
            }

            # 构建响应
            response = BacktestExecutionResponse(
                task_id=context.request_id,
                strategy_name=self._get_context_data(
                    "strategy_name", context, "unknown"
                ),
                execution_status="completed",
                backtest_result=backtest_result,
                performance_metrics=performance_metrics,
                risk_metrics=risk_metrics,
                execution_summary=execution_summary,
                report_content=report_content,
                created_at=context.created_at,
                completed_at=context.completed_at,
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

    def _validate_strategy_params(
        self, params: dict[str, Any], param_ranges: dict[str, Any]
    ) -> None:
        """验证策略参数

        Args:
            params: 策略参数
            param_ranges: 参数范围定义

        Raises:
            OrchestrationError: 参数验证失败
        """
        for param_name, param_value in params.items():
            if param_name in param_ranges:
                param_range = param_ranges[param_name]
                if isinstance(param_range, dict):
                    min_val = param_range.get("min")
                    max_val = param_range.get("max")
                    if min_val is not None and param_value < min_val:
                        raise OrchestrationError(
                            f"Parameter '{param_name}' value {param_value} is below minimum {min_val}"
                        )
                    if max_val is not None and param_value > max_val:
                        raise OrchestrationError(
                            f"Parameter '{param_name}' value {param_value} is above maximum {max_val}"
                        )

    async def _execute_parallel_backtest(
        self, request: BacktestRequest, context: OrchestrationContext
    ) -> dict[str, Any]:
        """执行并行回测

        Args:
            request: 回测请求
            context: 编排上下文

        Returns:
            并行回测结果
        """
        logger.info(f"Executing parallel backtest, request_id: {context.request_id}")

        # 目前实现简单的串行执行，后续可以优化为真正的并行
        # 可以按股票分组或按时间段分组进行并行处理
        return await self.backtest_service.run_backtest(request)

    def _analyze_performance(self, backtest_result: dict[str, Any]) -> dict[str, Any]:
        """分析性能指标

        Args:
            backtest_result: 回测结果

        Returns:
            性能指标
        """
        return {
            "total_return": backtest_result.get("total_return", 0.0),
            "annual_return": backtest_result.get("annual_return", 0.0),
            "sharpe_ratio": backtest_result.get("sharpe_ratio", 0.0),
            "win_rate": backtest_result.get("win_rate", 0.0),
            "profit_factor": backtest_result.get("profit_factor", 0.0),
            "avg_trade_return": backtest_result.get("avg_trade_return", 0.0),
        }

    def _analyze_risk(self, backtest_result: dict[str, Any]) -> dict[str, Any]:
        """分析风险指标

        Args:
            backtest_result: 回测结果

        Returns:
            风险指标
        """
        return {
            "max_drawdown": backtest_result.get("max_drawdown", 0.0),
            "volatility": backtest_result.get("volatility", 0.0),
            "var_95": backtest_result.get("var_95", 0.0),
            "beta": backtest_result.get("beta", 0.0),
            "downside_deviation": backtest_result.get("downside_deviation", 0.0),
        }

    def _generate_report(
        self,
        backtest_result: dict[str, Any],
        performance_metrics: dict[str, Any],
        risk_metrics: dict[str, Any],
        request: BacktestExecutionRequest,
        context: OrchestrationContext,
    ) -> str:
        """生成回测报告

        Args:
            backtest_result: 回测结果
            performance_metrics: 性能指标
            risk_metrics: 风险指标
            request: 回测请求

        Returns:
            Markdown格式的报告内容
        """
        report_lines = [
            "# 回测分析报告",
            "",
            "## 基本信息",
            f"- **策略名称**: {request.strategy_name}",
            f"- **股票代码**: {', '.join(request.symbols)}",
            f"- **回测期间**: {request.start_date} 至 {request.end_date}",
            f"- **初始资金**: {request.initial_capital:,.2f}",
            "",
            "## 性能指标",
            f"- **总收益率**: {performance_metrics.get('total_return', 0.0):.2%}",
            f"- **年化收益率**: {performance_metrics.get('annual_return', 0.0):.2%}",
            f"- **夏普比率**: {performance_metrics.get('sharpe_ratio', 0.0):.2f}",
            f"- **胜率**: {performance_metrics.get('win_rate', 0.0):.2%}",
            f"- **盈亏比**: {performance_metrics.get('profit_factor', 0.0):.2f}",
            "",
            "## 风险指标",
            f"- **最大回撤**: {risk_metrics.get('max_drawdown', 0.0):.2%}",
            f"- **波动率**: {risk_metrics.get('volatility', 0.0):.2%}",
            f"- **VaR(95%)**: {risk_metrics.get('var_95', 0.0):.2%}",
            f"- **Beta系数**: {risk_metrics.get('beta', 0.0):.2f}",
            "",
            "## 交易统计",
            f"- **交易次数**: {backtest_result.get('trades_count', 0)}",
            f"- **平均交易收益**: {performance_metrics.get('avg_trade_return', 0.0):.2%}",
            f"- **最大单笔盈利**: {backtest_result.get('max_trade_profit', 0.0):.2%}",
            f"- **最大单笔亏损**: {backtest_result.get('max_trade_loss', 0.0):.2%}",
            "",
            "---",
            f"*报告生成时间: {context.completed_at}*",
        ]

        return "\n".join(report_lines)
