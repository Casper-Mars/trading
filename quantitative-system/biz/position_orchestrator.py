"""持仓管理编排器

持仓管理任务执行流程的业务编排器，负责协调持仓查询、风险评估、调仓建议生成和执行监控。
"""

from typing import Any

from pydantic import BaseModel

from biz.base_orchestrator import BaseOrchestrator, OrchestrationContext
from services.data_service import DataService
from services.position_service import PositionRequest, PositionService
from services.risk_service import RiskService
from utils.exceptions import OrchestrationError
from utils.logger import get_logger

logger = get_logger(__name__)


class PositionManagementRequest(BaseModel):
    """持仓管理请求模型"""

    user_id: str  # 用户ID
    portfolio_id: str | None = None  # 投资组合ID
    symbols: list[str] = []  # 关注的股票代码列表
    risk_level: str = "moderate"  # 风险等级
    time_horizon: str = "medium_term"  # 投资时间范围
    rebalance_threshold: float = 0.05  # 调仓阈值
    include_recommendations: bool = True  # 是否包含调仓建议
    include_risk_analysis: bool = True  # 是否包含风险分析
    market_data_required: bool = True  # 是否需要市场数据


class PositionManagementResponse(BaseModel):
    """持仓管理响应模型"""

    task_id: str
    user_id: str
    portfolio_id: str | None
    execution_status: str  # running, completed, failed
    current_positions: dict[str, Any] = {}  # 当前持仓
    market_values: dict[str, Any] = {}  # 市值信息
    risk_metrics: dict[str, Any] = {}  # 风险指标
    performance_metrics: dict[str, Any] = {}  # 绩效指标
    rebalance_recommendations: list[dict[str, Any]] = []  # 调仓建议
    alerts: list[dict[str, Any]] = []  # 风险预警
    execution_summary: dict[str, Any] = {}  # 执行摘要
    created_at: str
    completed_at: str | None = None


class PositionOrchestrator(BaseOrchestrator):
    """持仓管理编排器

    协调以下服务完成持仓管理:
    1. PositionService: 持仓查询和管理
    2. DataService: 市场数据获取和价格更新
    3. RiskService: 风险评估和预警

    编排流程:
    前置检查 → 持仓查询 → 市场数据获取 → 风险评估 → 调仓建议 → 结果聚合
    """

    def __init__(self, position_service: PositionService, data_service: DataService, risk_service: RiskService):
        """初始化持仓管理编排器

        Args:
            position_service: 持仓服务
            data_service: 数据服务
            risk_service: 风险服务
        """
        super().__init__()
        self.position_service = position_service
        self.data_service = data_service
        self.risk_service = risk_service

        logger.info("PositionOrchestrator initialized")

    async def manage_positions(self, request: PositionManagementRequest) -> PositionManagementResponse:
        """执行持仓管理

        Args:
            request: 持仓管理请求

        Returns:
            持仓管理响应

        Raises:
            OrchestrationError: 持仓管理失败
        """
        context = OrchestrationContext(
            operation="position_management",
            request_data=request.dict()
        )

        logger.info(f"Starting position management, request_id: {context.request_id}")

        try:
            # 执行编排流程
            result = await self.execute(request, context)
            return result

        except Exception as e:
            logger.error(f"Position management failed: {e!s}, request_id: {context.request_id}")
            raise OrchestrationError(f"Position management failed: {e!s}") from e

    async def _pre_check(self, request: PositionManagementRequest, context: OrchestrationContext) -> bool:
        """前置检查

        Args:
            request: 持仓管理请求
            context: 编排上下文

        Returns:
            检查是否通过

        Raises:
            OrchestrationError: 前置检查失败
        """
        logger.info(f"Starting pre-check, request_id: {context.request_id}")

        try:
            # 1. 验证用户ID
            if not request.user_id or not request.user_id.strip():
                raise OrchestrationError("User ID cannot be empty")

            # 2. 验证风险等级
            valid_risk_levels = ["conservative", "moderate", "aggressive"]
            if request.risk_level not in valid_risk_levels:
                raise OrchestrationError(
                    f"Invalid risk level '{request.risk_level}'. "
                    f"Valid options: {valid_risk_levels}"
                )

            # 3. 验证时间范围
            valid_time_horizons = ["short_term", "medium_term", "long_term"]
            if request.time_horizon not in valid_time_horizons:
                raise OrchestrationError(
                    f"Invalid time horizon '{request.time_horizon}'. "
                    f"Valid options: {valid_time_horizons}"
                )

            # 4. 验证调仓阈值
            if request.rebalance_threshold <= 0 or request.rebalance_threshold > 1:
                raise OrchestrationError(
                    "Rebalance threshold must be between 0 and 1"
                )

            # 5. 检查用户是否有持仓（通过PositionService快速检查）
            has_positions = await self._check_user_positions(request.user_id, request.portfolio_id)
            if not has_positions:
                logger.warning(f"User {request.user_id} has no positions")

            # 保存验证结果到上下文
            self._set_context_data('user_id', request.user_id, context)
            self._set_context_data('portfolio_id', request.portfolio_id, context)
            self._set_context_data('has_positions', has_positions, context)
            self._set_context_data('risk_level', request.risk_level, context)

            logger.info(f"Pre-check completed successfully, request_id: {context.request_id}")
            return True

        except Exception as e:
            logger.error(f"Pre-check failed: {e!s}, request_id: {context.request_id}")
            raise OrchestrationError(f"Pre-check failed: {e!s}") from e

    async def _call_services(self, request: PositionManagementRequest, context: OrchestrationContext) -> dict[str, Any]:
        """调用服务

        Args:
            request: 持仓管理请求
            context: 编排上下文

        Returns:
            服务调用结果

        Raises:
            OrchestrationError: 服务调用失败
        """
        logger.info(f"Starting service calls, request_id: {context.request_id}")

        results = {}

        try:
            # 1. 持仓查询
            logger.info(f"Querying positions for user: {request.user_id}")

            position_request = PositionRequest(
                user_id=request.user_id,
                portfolio_id=request.portfolio_id,
                symbols=request.symbols,
                include_market_value=True,
                include_performance=True
            )

            positions_result = await self._safe_service_call(
                "position_service",
                lambda: self.position_service.get_positions(position_request),
                context
            )

            if positions_result is None:
                logger.warning(f"No positions found for user {request.user_id}")
                positions_result = {
                    'positions': [],
                    'total_market_value': 0.0,
                    'total_cost': 0.0,
                    'total_pnl': 0.0
                }

            results['positions_result'] = positions_result

            # 2. 市场数据获取（如果需要且有持仓）
            market_data = None
            if request.market_data_required and positions_result.get('positions'):
                logger.info("Fetching market data for position analysis")

                # 从持仓中提取股票代码
                position_symbols = [pos.get('symbol') for pos in positions_result.get('positions', [])]
                if request.symbols:
                    # 合并用户指定的股票代码
                    all_symbols = list(set(position_symbols + request.symbols))
                else:
                    all_symbols = position_symbols

                if all_symbols:
                    market_data = await self._safe_service_call(
                        "data_service",
                        lambda: self.data_service.get_current_market_data(symbols=all_symbols),
                        context
                    )

            results['market_data'] = market_data

            # 添加回滚操作
            self._add_rollback_action(
                'cleanup_position_cache',
                {'cache_keys': [f"positions_{request.user_id}_{context.request_id}"]},
                context
            )

            # 3. 风险评估（如果需要且有持仓）
            risk_analysis = None
            if request.include_risk_analysis and positions_result.get('positions'):
                logger.info("Performing risk analysis")

                risk_analysis = await self._safe_service_call(
                    "risk_service",
                    lambda: self.risk_service.analyze_portfolio_risk(
                        positions=positions_result.get('positions', []),
                        market_data=market_data,
                        risk_level=request.risk_level,
                        time_horizon=request.time_horizon
                    ),
                    context
                )

            results['risk_analysis'] = risk_analysis

            # 4. 调仓建议生成（如果需要）
            rebalance_recommendations = []
            if request.include_recommendations and positions_result.get('positions'):
                logger.info("Generating rebalance recommendations")

                rebalance_recommendations = await self._generate_rebalance_recommendations(
                    positions_result,
                    market_data,
                    risk_analysis,
                    request,
                    context
                )

            results['rebalance_recommendations'] = rebalance_recommendations

            # 5. 风险预警检查
            alerts = []
            if risk_analysis:
                alerts = self._generate_risk_alerts(risk_analysis, request.risk_level)

            results['alerts'] = alerts

            logger.info(f"Service calls completed successfully, request_id: {context.request_id}")
            return results

        except Exception as e:
            logger.error(f"Service calls failed: {e!s}, request_id: {context.request_id}")
            raise OrchestrationError(f"Service orchestration failed: {e!s}") from e

    async def _aggregate_results(self, service_results: dict[str, Any], context: OrchestrationContext) -> PositionManagementResponse:
        """聚合结果

        Args:
            service_results: 服务调用结果
            context: 编排上下文

        Returns:
            聚合后的持仓管理响应

        Raises:
            OrchestrationError: 结果聚合失败
        """
        logger.info(f"Starting result aggregation, request_id: {context.request_id}")

        try:
            # 获取服务结果
            positions_result = service_results.get('positions_result', {})
            market_data = service_results.get('market_data')
            risk_analysis = service_results.get('risk_analysis')
            rebalance_recommendations = service_results.get('rebalance_recommendations', [])
            alerts = service_results.get('alerts', [])

            # 提取持仓信息
            current_positions = {
                'positions': positions_result.get('positions', []),
                'total_count': len(positions_result.get('positions', [])),
                'last_updated': positions_result.get('last_updated')
            }

            # 提取市值信息
            market_values = {
                'total_market_value': positions_result.get('total_market_value', 0.0),
                'total_cost': positions_result.get('total_cost', 0.0),
                'total_pnl': positions_result.get('total_pnl', 0.0),
                'total_pnl_percent': self._calculate_pnl_percent(
                    positions_result.get('total_pnl', 0.0),
                    positions_result.get('total_cost', 0.0)
                ),
                'market_data_timestamp': market_data.get('timestamp') if market_data else None
            }

            # 提取风险指标
            risk_metrics = {}
            if risk_analysis:
                risk_metrics = {
                    'portfolio_beta': risk_analysis.get('portfolio_beta', 0.0),
                    'portfolio_volatility': risk_analysis.get('portfolio_volatility', 0.0),
                    'var_95': risk_analysis.get('var_95', 0.0),
                    'max_drawdown': risk_analysis.get('max_drawdown', 0.0),
                    'concentration_risk': risk_analysis.get('concentration_risk', 0.0),
                    'risk_score': risk_analysis.get('risk_score', 0.0)
                }

            # 提取绩效指标
            performance_metrics = {
                'total_return': market_values['total_pnl_percent'],
                'positions_count': current_positions['total_count'],
                'profitable_positions': len([
                    pos for pos in current_positions['positions']
                    if pos.get('unrealized_pnl', 0) > 0
                ]),
                'losing_positions': len([
                    pos for pos in current_positions['positions']
                    if pos.get('unrealized_pnl', 0) < 0
                ]),
                'win_rate': self._calculate_win_rate(current_positions['positions'])
            }

            # 构建执行摘要
            execution_summary = {
                'user_id': self._get_context_data('user_id', context, 'unknown'),
                'portfolio_id': self._get_context_data('portfolio_id', context),
                'has_positions': self._get_context_data('has_positions', context, False),
                'risk_level': self._get_context_data('risk_level', context, 'unknown'),
                'execution_time': context.execution_time,
                'services_called': len([k for k, v in service_results.items() if v is not None]),
                'recommendations_count': len(rebalance_recommendations),
                'alerts_count': len(alerts)
            }

            # 构建响应
            response = PositionManagementResponse(
                task_id=context.request_id,
                user_id=self._get_context_data('user_id', context, 'unknown'),
                portfolio_id=self._get_context_data('portfolio_id', context),
                execution_status="completed",
                current_positions=current_positions,
                market_values=market_values,
                risk_metrics=risk_metrics,
                performance_metrics=performance_metrics,
                rebalance_recommendations=rebalance_recommendations,
                alerts=alerts,
                execution_summary=execution_summary,
                created_at=context.created_at,
                completed_at=context.completed_at
            )

            # 保存聚合结果到上下文
            self._set_context_data('final_response', response.dict(), context)

            logger.info(f"Result aggregation completed successfully, request_id: {context.request_id}")

            return response

        except Exception as e:
            logger.error(f"Result aggregation failed: {e!s}, request_id: {context.request_id}")
            raise OrchestrationError(f"Failed to aggregate results: {e!s}") from e

    async def _check_user_positions(self, user_id: str, portfolio_id: str | None = None) -> bool:
        """检查用户是否有持仓

        Args:
            user_id: 用户ID
            portfolio_id: 投资组合ID

        Returns:
            是否有持仓
        """
        try:
            # 简单的持仓检查，实际实现可能需要调用PositionService
            # 这里先返回True，实际应该调用服务检查
            return True
        except Exception:
            return False

    async def _generate_rebalance_recommendations(self, positions_result: dict[str, Any], market_data: dict[str, Any] | None,
                                                risk_analysis: dict[str, Any] | None, request: PositionManagementRequest,
                                                context: OrchestrationContext) -> list[dict[str, Any]]:
        """生成调仓建议

        Args:
            positions_result: 持仓结果
            market_data: 市场数据
            risk_analysis: 风险分析
            request: 请求参数
            context: 编排上下文

        Returns:
            调仓建议列表
        """
        recommendations = []

        try:
            positions = positions_result.get('positions', [])
            total_value = positions_result.get('total_market_value', 0.0)

            if not positions or total_value <= 0:
                return recommendations

            # 分析持仓集中度
            for position in positions:
                symbol = position.get('symbol')
                market_value = position.get('market_value', 0.0)
                weight = market_value / total_value if total_value > 0 else 0.0

                # 检查是否超过集中度阈值
                concentration_threshold = self._get_concentration_threshold(request.risk_level)
                if weight > concentration_threshold:
                    recommendations.append({
                        'type': 'reduce_concentration',
                        'symbol': symbol,
                        'current_weight': weight,
                        'target_weight': concentration_threshold,
                        'action': 'sell',
                        'reason': f'Position concentration ({weight:.1%}) exceeds threshold ({concentration_threshold:.1%})',
                        'priority': 'high' if weight > concentration_threshold * 1.5 else 'medium'
                    })

                # 检查止损
                unrealized_pnl_percent = position.get('unrealized_pnl_percent', 0.0)
                stop_loss_threshold = self._get_stop_loss_threshold(request.risk_level)
                if unrealized_pnl_percent < -stop_loss_threshold:
                    recommendations.append({
                        'type': 'stop_loss',
                        'symbol': symbol,
                        'current_pnl': unrealized_pnl_percent,
                        'threshold': -stop_loss_threshold,
                        'action': 'sell',
                        'reason': f'Loss ({unrealized_pnl_percent:.1%}) exceeds stop-loss threshold ({stop_loss_threshold:.1%})',
                        'priority': 'high'
                    })

            # 基于风险分析的建议
            if risk_analysis:
                portfolio_risk = risk_analysis.get('risk_score', 0.0)
                risk_threshold = self._get_risk_threshold(request.risk_level)

                if portfolio_risk > risk_threshold:
                    recommendations.append({
                        'type': 'reduce_risk',
                        'current_risk': portfolio_risk,
                        'target_risk': risk_threshold,
                        'action': 'rebalance',
                        'reason': f'Portfolio risk ({portfolio_risk:.2f}) exceeds target ({risk_threshold:.2f})',
                        'priority': 'medium'
                    })

            logger.info(f"Generated {len(recommendations)} rebalance recommendations")
            return recommendations

        except Exception as e:
            logger.error(f"Failed to generate rebalance recommendations: {e!s}")
            return []

    def _generate_risk_alerts(self, risk_analysis: dict[str, Any], risk_level: str) -> list[dict[str, Any]]:
        """生成风险预警

        Args:
            risk_analysis: 风险分析结果
            risk_level: 风险等级

        Returns:
            风险预警列表
        """
        alerts = []

        try:
            # VaR预警
            var_95 = risk_analysis.get('var_95', 0.0)
            var_threshold = self._get_var_threshold(risk_level)
            if abs(var_95) > var_threshold:
                alerts.append({
                    'type': 'var_alert',
                    'level': 'warning',
                    'message': f'VaR(95%) {var_95:.2%} exceeds threshold {var_threshold:.2%}',
                    'metric': 'var_95',
                    'current_value': var_95,
                    'threshold': var_threshold
                })

            # 波动率预警
            volatility = risk_analysis.get('portfolio_volatility', 0.0)
            volatility_threshold = self._get_volatility_threshold(risk_level)
            if volatility > volatility_threshold:
                alerts.append({
                    'type': 'volatility_alert',
                    'level': 'info',
                    'message': f'Portfolio volatility {volatility:.2%} is high for {risk_level} risk level',
                    'metric': 'volatility',
                    'current_value': volatility,
                    'threshold': volatility_threshold
                })

            # 集中度预警
            concentration_risk = risk_analysis.get('concentration_risk', 0.0)
            if concentration_risk > 0.3:  # 30%集中度阈值
                alerts.append({
                    'type': 'concentration_alert',
                    'level': 'warning',
                    'message': f'High concentration risk detected: {concentration_risk:.1%}',
                    'metric': 'concentration_risk',
                    'current_value': concentration_risk,
                    'threshold': 0.3
                })

            return alerts

        except Exception as e:
            logger.error(f"Failed to generate risk alerts: {e!s}")
            return []

    def _calculate_pnl_percent(self, pnl: float, cost: float) -> float:
        """计算盈亏百分比"""
        return (pnl / cost * 100) if cost > 0 else 0.0

    def _calculate_win_rate(self, positions: list[dict[str, Any]]) -> float:
        """计算胜率"""
        if not positions:
            return 0.0

        profitable_count = len([pos for pos in positions if pos.get('unrealized_pnl', 0) > 0])
        return profitable_count / len(positions)

    def _get_concentration_threshold(self, risk_level: str) -> float:
        """获取集中度阈值"""
        thresholds = {
            'conservative': 0.15,  # 15%
            'moderate': 0.25,      # 25%
            'aggressive': 0.35     # 35%
        }
        return thresholds.get(risk_level, 0.25)

    def _get_stop_loss_threshold(self, risk_level: str) -> float:
        """获取止损阈值"""
        thresholds = {
            'conservative': 0.05,  # 5%
            'moderate': 0.10,      # 10%
            'aggressive': 0.15     # 15%
        }
        return thresholds.get(risk_level, 0.10)

    def _get_risk_threshold(self, risk_level: str) -> float:
        """获取风险阈值"""
        thresholds = {
            'conservative': 0.3,
            'moderate': 0.5,
            'aggressive': 0.7
        }
        return thresholds.get(risk_level, 0.5)

    def _get_var_threshold(self, risk_level: str) -> float:
        """获取VaR阈值"""
        thresholds = {
            'conservative': 0.02,  # 2%
            'moderate': 0.05,      # 5%
            'aggressive': 0.10     # 10%
        }
        return thresholds.get(risk_level, 0.05)

    def _get_volatility_threshold(self, risk_level: str) -> float:
        """获取波动率阈值"""
        thresholds = {
            'conservative': 0.15,  # 15%
            'moderate': 0.25,      # 25%
            'aggressive': 0.40     # 40%
        }
        return thresholds.get(risk_level, 0.25)
