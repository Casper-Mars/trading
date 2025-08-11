"""回测服务模块

提供策略回测、性能分析、风险评估等功能。
"""

from datetime import datetime
from typing import Any

import pandas as pd

from models.enums import BacktestStatus, StrategyType
from repositories.backtest_repo import BacktestRepository
from repositories.cache_repo import CacheRepository, CacheType
from strategies.base_strategy import BaseStrategy
from strategies.ma_strategy import MAStrategy
from strategies.macd_strategy import MACDStrategy
from strategies.rsi_strategy import RSIStrategy
from utils.exceptions import BacktestError, DataError
from utils.logger import logger


class BacktestService:
    """回测服务

    负责策略回测、性能分析、风险评估等功能。
    """

    def __init__(
        self,
        backtest_repo: BacktestRepository,
        cache_repo: CacheRepository,
    ):
        self.backtest_repo = backtest_repo
        self.cache_repo = cache_repo
        self._cache_ttl = 3600  # 1小时缓存

        # 策略映射
        self._strategy_map = {
            StrategyType.MA: MAStrategy,
            StrategyType.MACD: MACDStrategy,
            StrategyType.RSI: RSIStrategy,
        }

    async def run_backtest(
        self,
        strategy_type: StrategyType,
        strategy_params: dict[str, Any],
        symbols: list[str],
        start_date: str,
        end_date: str,
        initial_capital: float = 100000.0,
        commission_rate: float = 0.001,
        slippage_rate: float = 0.001,
        use_cache: bool = True,
    ) -> dict[str, Any]:
        """运行回测

        Args:
            strategy_type: 策略类型
            strategy_params: 策略参数
            symbols: 股票代码列表
            start_date: 开始日期 (YYYY-MM-DD)
            end_date: 结束日期 (YYYY-MM-DD)
            initial_capital: 初始资金
            commission_rate: 手续费率
            slippage_rate: 滑点率
            use_cache: 是否使用缓存

        Returns:
            回测结果字典
        """
        try:
            # 构建缓存键
            cache_key = self._build_backtest_cache_key(
                strategy_type, strategy_params, symbols, start_date, end_date,
                initial_capital, commission_rate, slippage_rate
            )

            # 尝试从缓存获取
            if use_cache:
                cached_result = self.cache_repo.get(
                    CacheType.BACKTEST_RESULT, cache_key, serialize_method="json"
                )
                if cached_result is not None:
                    logger.info(f"从缓存获取回测结果: {strategy_type.value}")
                    return cached_result

            # 创建策略实例
            strategy = self._create_strategy(strategy_type, strategy_params)
            if not strategy:
                raise BacktestError(f"不支持的策略类型: {strategy_type}")

            # 获取历史数据
            market_data = await self._get_market_data(symbols, start_date, end_date)
            if market_data.empty:
                raise DataError("无法获取市场数据")

            # 执行回测
            backtest_result = await self._execute_backtest(
                strategy, market_data, initial_capital, commission_rate, slippage_rate
            )

            # 计算性能指标
            performance_metrics = self._calculate_performance_metrics(
                backtest_result, initial_capital
            )

            # 计算风险指标
            risk_metrics = self._calculate_risk_metrics(backtest_result)

            # 组装完整结果
            complete_result = {
                "backtest_id": self._generate_backtest_id(),
                "strategy_type": strategy_type.value,
                "strategy_params": strategy_params,
                "symbols": symbols,
                "start_date": start_date,
                "end_date": end_date,
                "initial_capital": initial_capital,
                "commission_rate": commission_rate,
                "slippage_rate": slippage_rate,
                "status": BacktestStatus.COMPLETED.value,
                "performance_metrics": performance_metrics,
                "risk_metrics": risk_metrics,
                "trades": backtest_result.get("trades", []),
                "daily_returns": backtest_result.get("daily_returns", []),
                "portfolio_value": backtest_result.get("portfolio_value", []),
                "created_at": datetime.now().isoformat(),
            }

            # 保存回测结果
            await self.backtest_repo.save_backtest_result(complete_result)

            # 缓存结果
            if use_cache:
                self.cache_repo.set(
                    CacheType.BACKTEST_RESULT,
                    cache_key,
                    complete_result,
                    ttl=self._cache_ttl,
                    serialize_method="json",
                )

            logger.info(
                f"回测完成: {strategy_type.value}, 总收益率: {performance_metrics.get('total_return', 0):.2%}"
            )
            return complete_result

        except Exception as e:
            logger.error(f"回测执行失败: {strategy_type.value}, 错误: {e}")
            raise BacktestError(f"回测执行失败: {e}") from e

    async def get_backtest_result(self, backtest_id: str) -> dict[str, Any] | None:
        """获取回测结果

        Args:
            backtest_id: 回测ID

        Returns:
            回测结果字典，如果不存在返回None
        """
        try:
            result = await self.backtest_repo.get_backtest_result(backtest_id)
            if result:
                logger.info(f"获取回测结果成功: {backtest_id}")
            return result
        except Exception as e:
            logger.error(f"获取回测结果失败: {backtest_id}, 错误: {e}")
            return None

    async def list_backtest_results(
        self,
        strategy_type: StrategyType | None = None,
        symbols: list[str] | None = None,
        start_date: str | None = None,
        end_date: str | None = None,
        limit: int = 50,
        offset: int = 0,
    ) -> list[dict[str, Any]]:
        """列出回测结果

        Args:
            strategy_type: 策略类型过滤
            symbols: 股票代码过滤
            start_date: 开始日期过滤
            end_date: 结束日期过滤
            limit: 返回数量限制
            offset: 偏移量

        Returns:
            回测结果列表
        """
        try:
            filters = {}
            if strategy_type:
                filters["strategy_type"] = strategy_type.value
            if symbols:
                filters["symbols"] = symbols
            if start_date:
                filters["start_date"] = start_date
            if end_date:
                filters["end_date"] = end_date

            results = await self.backtest_repo.list_backtest_results(
                filters=filters, limit=limit, offset=offset
            )
            logger.info(f"获取回测结果列表成功, 数量: {len(results)}")
            return results
        except Exception as e:
            logger.error(f"获取回测结果列表失败: {e}")
            return []

    async def delete_backtest_result(self, backtest_id: str) -> bool:
        """删除回测结果

        Args:
            backtest_id: 回测ID

        Returns:
            是否删除成功
        """
        try:
            success = await self.backtest_repo.delete_backtest_result(backtest_id)
            if success:
                logger.info(f"删除回测结果成功: {backtest_id}")
            else:
                logger.warning(f"回测结果不存在: {backtest_id}")
            return success
        except Exception as e:
            logger.error(f"删除回测结果失败: {backtest_id}, 错误: {e}")
            return False

    async def compare_strategies(
        self,
        backtest_ids: list[str],
        metrics: list[str] | None = None,
    ) -> dict[str, Any]:
        """比较多个策略的回测结果

        Args:
            backtest_ids: 回测ID列表
            metrics: 要比较的指标列表

        Returns:
            策略比较结果
        """
        try:
            if not backtest_ids:
                return {"comparison": [], "summary": {}}

            # 获取所有回测结果
            results = []
            for backtest_id in backtest_ids:
                result = await self.get_backtest_result(backtest_id)
                if result:
                    results.append(result)

            if not results:
                return {"comparison": [], "summary": {}}

            # 默认比较指标
            if not metrics:
                metrics = [
                    "total_return", "annual_return", "sharpe_ratio", "max_drawdown",
                    "win_rate", "profit_factor", "volatility"
                ]

            # 构建比较数据
            comparison_data = []
            for result in results:
                performance = result.get("performance_metrics", {})
                risk = result.get("risk_metrics", {})

                comparison_item = {
                    "backtest_id": result["backtest_id"],
                    "strategy_type": result["strategy_type"],
                    "symbols": result["symbols"],
                    "start_date": result["start_date"],
                    "end_date": result["end_date"],
                }

                # 添加指定的指标
                for metric in metrics:
                    if metric in performance:
                        comparison_item[metric] = performance[metric]
                    elif metric in risk:
                        comparison_item[metric] = risk[metric]
                    else:
                        comparison_item[metric] = None

                comparison_data.append(comparison_item)

            # 计算汇总统计
            summary = self._calculate_comparison_summary(comparison_data, metrics)

            logger.info(f"策略比较完成, 比较了 {len(results)} 个策略")
            return {
                "comparison": comparison_data,
                "summary": summary,
                "metrics": metrics,
            }

        except Exception as e:
            logger.error(f"策略比较失败: {e}")
            return {"comparison": [], "summary": {}}

    def _create_strategy(
        self, strategy_type: StrategyType, params: dict[str, Any]
    ) -> BaseStrategy | None:
        """创建策略实例"""
        try:
            strategy_class = self._strategy_map.get(strategy_type)
            if not strategy_class:
                return None
            return strategy_class(**params)
        except Exception as e:
            logger.error(f"创建策略实例失败: {strategy_type}, 错误: {e}")
            return None

    async def _get_market_data(
        self, symbols: list[str], start_date: str, end_date: str
    ) -> pd.DataFrame:
        """获取市场数据"""
        # 这里应该调用DataService获取市场数据
        # 为了简化，这里返回空DataFrame，实际实现时需要集成DataService
        logger.warning("市场数据获取功能需要集成DataService")
        return pd.DataFrame()

    async def _execute_backtest(
        self,
        strategy: BaseStrategy,
        market_data: pd.DataFrame,
        initial_capital: float,
        commission_rate: float,
        slippage_rate: float,
    ) -> dict[str, Any]:
        """执行回测逻辑"""
        try:
            # 初始化回测状态
            portfolio_value = [initial_capital]
            cash = initial_capital
            positions = {}
            trades = []
            daily_returns = []

            # 模拟交易逻辑（简化版本）
            for i, (date, row) in enumerate(market_data.iterrows()):
                # 生成交易信号
                signals = strategy.generate_signals(market_data.iloc[:i+1])

                # 执行交易
                for symbol, signal in signals.items():
                    if signal["action"] == "buy" and cash > 0:
                        # 买入逻辑
                        price = row[f"{symbol}_close"] * (1 + slippage_rate)
                        shares = int(cash * signal["weight"] / price)
                        cost = shares * price * (1 + commission_rate)

                        if cost <= cash:
                            cash -= cost
                            positions[symbol] = positions.get(symbol, 0) + shares
                            trades.append({
                                "date": date.isoformat(),
                                "symbol": symbol,
                                "action": "buy",
                                "shares": shares,
                                "price": price,
                                "cost": cost,
                            })

                    elif signal["action"] == "sell" and symbol in positions and positions[symbol] > 0:
                        # 卖出逻辑
                        price = row[f"{symbol}_close"] * (1 - slippage_rate)
                        shares = min(positions[symbol], int(positions[symbol] * signal["weight"]))
                        proceeds = shares * price * (1 - commission_rate)

                        cash += proceeds
                        positions[symbol] -= shares
                        if positions[symbol] == 0:
                            del positions[symbol]

                        trades.append({
                            "date": date.isoformat(),
                            "symbol": symbol,
                            "action": "sell",
                            "shares": shares,
                            "price": price,
                            "proceeds": proceeds,
                        })

                # 计算当日组合价值
                position_value = sum(
                    shares * row[f"{symbol}_close"]
                    for symbol, shares in positions.items()
                )
                total_value = cash + position_value
                portfolio_value.append(total_value)

                # 计算日收益率
                if len(portfolio_value) > 1:
                    daily_return = (total_value - portfolio_value[-2]) / portfolio_value[-2]
                    daily_returns.append(daily_return)

            return {
                "trades": trades,
                "daily_returns": daily_returns,
                "portfolio_value": portfolio_value,
                "final_cash": cash,
                "final_positions": positions,
            }

        except Exception as e:
            logger.error(f"回测执行失败: {e}")
            raise BacktestError(f"回测执行失败: {e}") from e

    def _calculate_performance_metrics(
        self, backtest_result: dict[str, Any], initial_capital: float
    ) -> dict[str, float]:
        """计算性能指标"""
        try:
            portfolio_value = backtest_result.get("portfolio_value", [])
            daily_returns = backtest_result.get("daily_returns", [])
            trades = backtest_result.get("trades", [])

            if not portfolio_value or len(portfolio_value) < 2:
                return {}

            final_value = portfolio_value[-1]
            total_return = (final_value - initial_capital) / initial_capital

            # 年化收益率（假设252个交易日）
            trading_days = len(portfolio_value) - 1
            annual_return = (1 + total_return) ** (252 / trading_days) - 1 if trading_days > 0 else 0

            # 波动率
            volatility = pd.Series(daily_returns).std() * (252 ** 0.5) if daily_returns else 0

            # 夏普比率（假设无风险利率为3%）
            risk_free_rate = 0.03
            sharpe_ratio = (annual_return - risk_free_rate) / volatility if volatility > 0 else 0

            # 最大回撤
            max_drawdown = self._calculate_max_drawdown(portfolio_value)

            # 交易统计
            win_trades = [t for t in trades if t.get("action") == "sell" and
                         t.get("proceeds", 0) > t.get("cost", 0)]
            win_rate = len(win_trades) / len(trades) if trades else 0

            # 盈亏比
            profit_factor = self._calculate_profit_factor(trades)

            return {
                "total_return": round(total_return, 4),
                "annual_return": round(annual_return, 4),
                "volatility": round(volatility, 4),
                "sharpe_ratio": round(sharpe_ratio, 4),
                "max_drawdown": round(max_drawdown, 4),
                "win_rate": round(win_rate, 4),
                "profit_factor": round(profit_factor, 4),
                "total_trades": len(trades),
                "winning_trades": len(win_trades),
            }

        except Exception as e:
            logger.error(f"计算性能指标失败: {e}")
            return {}

    def _calculate_risk_metrics(
        self, backtest_result: dict[str, Any]
    ) -> dict[str, float]:
        """计算风险指标"""
        try:
            daily_returns = backtest_result.get("daily_returns", [])
            portfolio_value = backtest_result.get("portfolio_value", [])

            if not daily_returns:
                return {}

            returns_series = pd.Series(daily_returns)

            # VaR (Value at Risk) 95%
            var_95 = returns_series.quantile(0.05)

            # CVaR (Conditional Value at Risk)
            cvar_95 = returns_series[returns_series <= var_95].mean()

            # 下行偏差
            downside_deviation = returns_series[returns_series < 0].std()

            # Sortino比率
            mean_return = returns_series.mean()
            sortino_ratio = mean_return / downside_deviation if downside_deviation > 0 else 0

            # Calmar比率
            max_drawdown = self._calculate_max_drawdown(portfolio_value)
            annual_return = mean_return * 252
            calmar_ratio = annual_return / abs(max_drawdown) if max_drawdown != 0 else 0

            return {
                "var_95": round(var_95, 4),
                "cvar_95": round(cvar_95, 4),
                "downside_deviation": round(downside_deviation, 4),
                "sortino_ratio": round(sortino_ratio, 4),
                "calmar_ratio": round(calmar_ratio, 4),
            }

        except Exception as e:
            logger.error(f"计算风险指标失败: {e}")
            return {}

    def _calculate_max_drawdown(self, portfolio_value: list[float]) -> float:
        """计算最大回撤"""
        try:
            if len(portfolio_value) < 2:
                return 0.0

            peak = portfolio_value[0]
            max_dd = 0.0

            for value in portfolio_value[1:]:
                if value > peak:
                    peak = value
                else:
                    drawdown = (peak - value) / peak
                    max_dd = max(max_dd, drawdown)

            return max_dd
        except Exception:
            return 0.0

    def _calculate_profit_factor(self, trades: list[dict[str, Any]]) -> float:
        """计算盈亏比"""
        try:
            if not trades:
                return 0.0

            gross_profit = 0.0
            gross_loss = 0.0

            # 简化的盈亏计算（实际应该配对买卖交易）
            for trade in trades:
                if trade.get("action") == "sell":
                    pnl = trade.get("proceeds", 0) - trade.get("cost", 0)
                    if pnl > 0:
                        gross_profit += pnl
                    else:
                        gross_loss += abs(pnl)

            return gross_profit / gross_loss if gross_loss > 0 else 0.0
        except Exception:
            return 0.0

    def _calculate_comparison_summary(
        self, comparison_data: list[dict[str, Any]], metrics: list[str]
    ) -> dict[str, Any]:
        """计算比较汇总统计"""
        try:
            if not comparison_data:
                return {}

            summary = {}
            for metric in metrics:
                values = [item.get(metric) for item in comparison_data if item.get(metric) is not None]
                if values:
                    summary[metric] = {
                        "mean": round(sum(values) / len(values), 4),
                        "min": round(min(values), 4),
                        "max": round(max(values), 4),
                        "std": round(pd.Series(values).std(), 4),
                    }

            return summary
        except Exception as e:
            logger.error(f"计算比较汇总失败: {e}")
            return {}

    def _build_backtest_cache_key(
        self,
        strategy_type: StrategyType,
        strategy_params: dict[str, Any],
        symbols: list[str],
        start_date: str,
        end_date: str,
        initial_capital: float,
        commission_rate: float,
        slippage_rate: float,
    ) -> str:
        """构建回测缓存键"""
        params_str = "_".join(f"{k}={v}" for k, v in sorted(strategy_params.items()))
        symbols_str = "-".join(sorted(symbols))
        return (
            f"backtest_{strategy_type.value}_{params_str}_{symbols_str}_"
            f"{start_date}_{end_date}_{initial_capital}_{commission_rate}_{slippage_rate}"
        )

    def _generate_backtest_id(self) -> str:
        """生成回测ID"""
        from uuid import uuid4
        return f"bt_{uuid4().hex[:12]}"
