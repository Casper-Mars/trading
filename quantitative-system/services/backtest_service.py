"""回测服务模块

提供策略回测、性能分析、风险评估等功能。
"""

from datetime import datetime
from typing import Any

import backtrader as bt
import pandas as pd

from models.enums import BacktestStatus, StrategyType
from repositories.backtest_repo import BacktestRepository
from repositories.cache_repo import CacheRepository, CacheType
from strategies.base_strategy import BaseStrategy
from strategies.ma_strategy import MAStrategy
from strategies.macd_strategy import MACDStrategy
from strategies.multi_factor_strategy import MultiFactorStrategy
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
            StrategyType.MULTI_FACTOR: MultiFactorStrategy,
            StrategyType.RSI: RSIStrategy,
        }

        # Backtrader引擎配置
        self._default_initial_capital = 100000.0
        self._default_commission_rate = 0.001
        self._default_slippage_rate = 0.001

    def _initialize_backtrader_engine(
        self,
        initial_capital: float | None = None,
        commission_rate: float | None = None,
        slippage_rate: float | None = None,
    ) -> bt.Cerebro:
        """初始化Backtrader引擎

        Args:
            initial_capital: 初始资金
            commission_rate: 手续费率
            slippage_rate: 滑点率

        Returns:
            配置好的Cerebro引擎实例
        """
        try:
            # 创建Cerebro引擎
            cerebro = bt.Cerebro()

            # 设置初始资金
            capital = initial_capital or self._default_initial_capital
            cerebro.broker.setcash(capital)

            # 设置手续费
            commission = commission_rate or self._default_commission_rate
            cerebro.broker.setcommission(commission=commission)

            # 添加分析器
            self._add_analyzers(cerebro)

            # 设置数据源格式
            cerebro.addwriter(bt.WriterFile, csv=False, rounding=4)

            logger.info(
                f"Backtrader引擎初始化完成 - 资金: {capital:,.2f}, "
                f"手续费: {commission:.4f}, 滑点: {slippage_rate or self._default_slippage_rate:.4f}"
            )

            return cerebro

        except Exception as e:
            logger.error(f"Backtrader引擎初始化失败: {e}")
            raise BacktestError(f"引擎初始化失败: {e}") from e

    def _add_analyzers(self, cerebro: bt.Cerebro) -> None:
        """添加分析器到Cerebro引擎

        Args:
            cerebro: Cerebro引擎实例
        """
        try:
            # 添加夏普比率分析器
            cerebro.addanalyzer(bt.analyzers.SharpeRatio, _name="sharpe")

            # 添加回撤分析器
            cerebro.addanalyzer(bt.analyzers.DrawDown, _name="drawdown")

            # 添加收益分析器
            cerebro.addanalyzer(bt.analyzers.Returns, _name="returns")

            # 添加交易分析器
            cerebro.addanalyzer(bt.analyzers.TradeAnalyzer, _name="trades")

            # 添加年化收益分析器
            cerebro.addanalyzer(bt.analyzers.AnnualReturn, _name="annual_return")

            # 添加波动率分析器
            cerebro.addanalyzer(bt.analyzers.VariabilityWeightedReturn, _name="vwr")

            # 添加Calmar比率分析器
            cerebro.addanalyzer(bt.analyzers.CalmarRatio, _name="calmar")

            # 添加时间收益分析器
            cerebro.addanalyzer(bt.analyzers.TimeReturn, _name="time_return")

            logger.info("分析器添加完成")

        except Exception as e:
            logger.error(f"添加分析器失败: {e}")
            raise BacktestError(f"添加分析器失败: {e}") from e

    def _create_pandas_data_feed(
        self, market_data: pd.DataFrame, symbol: str
    ) -> bt.feeds.PandasData:
        """创建Pandas数据源适配器

        Args:
            market_data: 市场数据DataFrame
            symbol: 股票代码

        Returns:
            Backtrader数据源
        """
        try:
            # 准备数据格式
            symbol_data = market_data[
                [
                    f"{symbol}_open",
                    f"{symbol}_high",
                    f"{symbol}_low",
                    f"{symbol}_close",
                    f"{symbol}_volume",
                ]
            ].copy()

            # 重命名列以匹配Backtrader格式
            symbol_data.columns = ["open", "high", "low", "close", "volume"]

            # 确保索引是datetime类型
            if not isinstance(symbol_data.index, pd.DatetimeIndex):
                symbol_data.index = pd.to_datetime(symbol_data.index)

            # 创建数据源
            data_feed = bt.feeds.PandasData(
                dataname=symbol_data,
                datetime=None,  # 使用索引作为日期
                open=0,
                high=1,
                low=2,
                close=3,
                volume=4,
                openinterest=-1,  # 不使用持仓量
            )

            logger.info(
                f"为{symbol}创建数据源成功, 数据范围: {symbol_data.index[0]} 到 {symbol_data.index[-1]}"
            )

            return data_feed

        except Exception as e:
            logger.error(f"创建{symbol}数据源失败: {e}")
            raise DataError(f"创建数据源失败: {e}") from e

    def _parse_backtrader_results(self, results: list) -> dict[str, Any]:
        """解析Backtrader回测结果

        Args:
            results: Backtrader运行结果列表

        Returns:
            解析后的结果字典
        """
        try:
            if not results or len(results) == 0:
                raise BacktestError("回测结果为空")

            # 获取第一个策略的结果
            strat = results[0]

            # 解析分析器结果
            analyzers_data = {}

            # 夏普比率
            if hasattr(strat.analyzers, "sharpe"):
                sharpe_analysis = strat.analyzers.sharpe.get_analysis()
                analyzers_data["sharpe_ratio"] = sharpe_analysis.get("sharperatio", 0)

            # 回撤分析
            if hasattr(strat.analyzers, "drawdown"):
                drawdown_analysis = strat.analyzers.drawdown.get_analysis()
                analyzers_data["max_drawdown"] = (
                    drawdown_analysis.get("max", {}).get("drawdown", 0) / 100
                )
                analyzers_data["max_drawdown_period"] = drawdown_analysis.get(
                    "max", {}
                ).get("len", 0)

            # 收益分析
            if hasattr(strat.analyzers, "returns"):
                returns_analysis = strat.analyzers.returns.get_analysis()
                analyzers_data["total_return"] = returns_analysis.get("rtot", 0)
                analyzers_data["average_return"] = returns_analysis.get("ravg", 0)

            # 交易分析
            if hasattr(strat.analyzers, "trades"):
                trades_analysis = strat.analyzers.trades.get_analysis()
                analyzers_data["total_trades"] = trades_analysis.get("total", {}).get(
                    "total", 0
                )
                analyzers_data["winning_trades"] = trades_analysis.get("won", {}).get(
                    "total", 0
                )
                analyzers_data["losing_trades"] = trades_analysis.get("lost", {}).get(
                    "total", 0
                )

                total_trades = analyzers_data["total_trades"]
                analyzers_data["win_rate"] = (
                    analyzers_data["winning_trades"] / total_trades
                    if total_trades > 0
                    else 0
                )

                # 盈亏比
                won_pnl = trades_analysis.get("won", {}).get("pnl", {}).get("total", 0)
                lost_pnl = abs(
                    trades_analysis.get("lost", {}).get("pnl", {}).get("total", 0)
                )
                analyzers_data["profit_factor"] = (
                    won_pnl / lost_pnl if lost_pnl > 0 else 0
                )

            # 年化收益
            if hasattr(strat.analyzers, "annual_return"):
                annual_analysis = strat.analyzers.annual_return.get_analysis()
                analyzers_data["annual_returns"] = annual_analysis

            # Calmar比率
            if hasattr(strat.analyzers, "calmar"):
                calmar_analysis = strat.analyzers.calmar.get_analysis()
                analyzers_data["calmar_ratio"] = calmar_analysis.get("calmarratio", 0)

            # 时间收益
            if hasattr(strat.analyzers, "time_return"):
                time_return_analysis = strat.analyzers.time_return.get_analysis()
                analyzers_data["time_returns"] = time_return_analysis

            # 获取策略的组合价值历史和每日收益
            portfolio_values = []
            daily_returns = []

            # 从时间收益分析器获取详细的收益数据
            if hasattr(strat.analyzers, "time_return"):
                time_return_data = strat.analyzers.time_return.get_analysis()
                if time_return_data:
                    # 提取每日组合价值
                    for date, return_value in time_return_data.items():
                        portfolio_values.append(
                            {
                                "date": date.strftime("%Y-%m-%d")
                                if hasattr(date, "strftime")
                                else str(date),
                                "value": return_value,
                            }
                        )

                    # 计算每日收益率
                    values = list(time_return_data.values())
                    for i in range(1, len(values)):
                        daily_return = (
                            (values[i] - values[i - 1]) / values[i - 1]
                            if values[i - 1] != 0
                            else 0
                        )
                        daily_returns.append(daily_return)

            # 如果没有时间收益数据，使用当前组合价值
            if not portfolio_values:
                portfolio_values = [
                    {
                        "date": datetime.now().strftime("%Y-%m-%d"),
                        "value": strat.broker.getvalue(),
                    }
                ]

            # 获取详细的交易记录
            trades = []
            if hasattr(strat.analyzers, "trades"):
                trades_analysis = strat.analyzers.trades.get_analysis()

                # 提取获胜交易详情
                if "won" in trades_analysis and "pnl" in trades_analysis["won"]:
                    won_trades = trades_analysis["won"]["pnl"]
                    if isinstance(won_trades, dict) and "trades" in won_trades:
                        for trade in won_trades["trades"]:
                            trades.append(
                                {"type": "win", "pnl": trade, "status": "closed"}
                            )

                # 提取失败交易详情
                if "lost" in trades_analysis and "pnl" in trades_analysis["lost"]:
                    lost_trades = trades_analysis["lost"]["pnl"]
                    if isinstance(lost_trades, dict) and "trades" in lost_trades:
                        for trade in lost_trades["trades"]:
                            trades.append(
                                {"type": "loss", "pnl": trade, "status": "closed"}
                            )

            # 计算额外的性能指标
            final_value = strat.broker.getvalue()
            final_cash = strat.broker.getcash()

            result = {
                "analyzers": analyzers_data,
                "portfolio_values": portfolio_values,
                "daily_returns": daily_returns,
                "trades": trades,
                "final_portfolio_value": final_value,
                "final_cash": final_cash,
                "total_positions": len(trades),
                "portfolio_summary": {
                    "total_value": final_value,
                    "cash": final_cash,
                    "positions_value": final_value - final_cash,
                    "total_return": (
                        final_value / analyzers_data.get("initial_capital", 100000) - 1
                    )
                    if "initial_capital" in analyzers_data
                    else 0,
                },
            }

            logger.info(
                f"回测结果解析完成, 最终组合价值: {result['final_portfolio_value']:,.2f}"
            )

            return result

        except Exception as e:
            logger.error(f"解析回测结果失败: {e}")
            raise BacktestError(f"解析回测结果失败: {e}") from e

    async def _save_backtest_to_database(self, backtest_result: dict[str, Any]) -> None:
        """保存回测结果到数据库

        Args:
            backtest_result: 回测结果字典
        """
        try:
            # 保存到数据库
            await self.backtest_repo.save_backtest_result(backtest_result)

            logger.info(
                f"回测结果已保存到数据库, ID: {backtest_result.get('backtest_id')}"
            )

        except Exception as e:
            logger.error(f"保存回测结果到数据库失败: {e}")
            raise BacktestError(f"保存回测结果失败: {e}") from e

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
                strategy_type,
                strategy_params,
                symbols,
                start_date,
                end_date,
                initial_capital,
                commission_rate,
                slippage_rate,
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
                    "total_return",
                    "annual_return",
                    "sharpe_ratio",
                    "max_drawdown",
                    "win_rate",
                    "profit_factor",
                    "volatility",
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
        """获取市场数据

        Args:
            symbols: 股票代码列表
            start_date: 开始日期
            end_date: 结束日期

        Returns:
            包含OHLCV数据的DataFrame
        """
        try:
            # 调用DataService获取市场数据
            if hasattr(self, "data_service") and self.data_service:
                market_data = await self.data_service.get_market_data(
                    symbols=symbols, start_date=start_date, end_date=end_date
                )

                if market_data.empty:
                    raise DataError(f"无法获取{symbols}的市场数据")

                logger.info(
                    f"成功获取{len(symbols)}只股票的市场数据, 数据范围: {start_date} 到 {end_date}"
                )
                return market_data
            else:
                # 如果没有DataService，生成模拟数据用于测试
                logger.warning("DataService未配置, 使用模拟数据进行测试")
                return self._generate_mock_market_data(symbols, start_date, end_date)

        except Exception as e:
            logger.error(f"获取市场数据失败: {e}")
            raise DataError(f"获取市场数据失败: {e}") from e

    def _generate_mock_market_data(
        self, symbols: list[str], start_date: str, end_date: str
    ) -> pd.DataFrame:
        """生成模拟市场数据用于测试

        Args:
            symbols: 股票代码列表
            start_date: 开始日期
            end_date: 结束日期

        Returns:
            模拟的市场数据DataFrame
        """
        import numpy as np

        # 创建日期范围
        date_range = pd.date_range(start=start_date, end=end_date, freq="D")

        # 过滤工作日
        business_days = date_range[date_range.weekday < 5]

        data = {}

        for symbol in symbols:
            # 生成模拟价格数据
            np.random.seed(hash(symbol) % 2**32)  # 确保每个股票的数据一致

            # 初始价格
            initial_price = 100.0

            # 生成随机收益率
            returns = np.random.normal(0.001, 0.02, len(business_days))

            # 计算价格序列
            prices = [initial_price]
            for ret in returns[1:]:
                prices.append(prices[-1] * (1 + ret))

            # 生成OHLCV数据
            closes = np.array(prices)

            # 生成开盘价（基于前一日收盘价加随机波动）
            opens = np.roll(closes, 1)
            opens[0] = initial_price
            opens = opens * (1 + np.random.normal(0, 0.005, len(opens)))

            # 生成最高价和最低价
            highs = np.maximum(opens, closes) * (
                1 + np.abs(np.random.normal(0, 0.01, len(opens)))
            )
            lows = np.minimum(opens, closes) * (
                1 - np.abs(np.random.normal(0, 0.01, len(opens)))
            )

            # 生成成交量
            volumes = np.random.lognormal(10, 0.5, len(business_days))

            # 添加到数据字典
            data[f"{symbol}_open"] = opens
            data[f"{symbol}_high"] = highs
            data[f"{symbol}_low"] = lows
            data[f"{symbol}_close"] = closes
            data[f"{symbol}_volume"] = volumes

        # 创建DataFrame
        df = pd.DataFrame(data, index=business_days)

        logger.info(f"生成模拟数据完成, 包含{len(symbols)}只股票, {len(df)}个交易日")

        return df

    async def _execute_backtest(
        self,
        strategy: BaseStrategy,
        market_data: pd.DataFrame,
        initial_capital: float,
        commission_rate: float,
        slippage_rate: float,
    ) -> dict[str, Any]:
        """执行回测逻辑（使用Backtrader引擎）"""
        try:
            # 初始化Backtrader引擎
            cerebro = self._initialize_backtrader_engine(
                initial_capital=initial_capital,
                commission_rate=commission_rate,
                slippage_rate=slippage_rate,
            )

            # 添加策略到引擎
            cerebro.addstrategy(strategy.__class__, **strategy.params)

            # 获取并添加数据源
            for symbol in market_data.columns:
                if symbol.endswith("_close"):
                    symbol_name = symbol.replace("_close", "")
                    data_feed = self._create_pandas_data_feed(market_data, symbol_name)
                    cerebro.adddata(data_feed)

            # 记录初始资金
            initial_value = cerebro.broker.getvalue()
            logger.info(f"开始回测, 初始资金: {initial_value:,.2f}")

            # 运行回测
            results = cerebro.run()

            # 记录最终资金
            final_value = cerebro.broker.getvalue()
            logger.info(f"回测完成, 最终资金: {final_value:,.2f}")

            # 解析回测结果
            parsed_results = self._parse_backtrader_results(results)

            # 添加基本信息
            parsed_results.update(
                {
                    "initial_capital": initial_capital,
                    "commission_rate": commission_rate,
                    "slippage_rate": slippage_rate,
                    "backtest_id": f"bt_{int(datetime.now().timestamp())}",
                }
            )

            return parsed_results

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
            annual_return = (
                (1 + total_return) ** (252 / trading_days) - 1
                if trading_days > 0
                else 0
            )

            # 波动率
            volatility = (
                pd.Series(daily_returns).std() * (252**0.5) if daily_returns else 0
            )

            # 夏普比率（假设无风险利率为3%）
            risk_free_rate = 0.03
            sharpe_ratio = (
                (annual_return - risk_free_rate) / volatility if volatility > 0 else 0
            )

            # 最大回撤
            max_drawdown = self._calculate_max_drawdown(portfolio_value)

            # 交易统计
            win_trades = [
                t
                for t in trades
                if t.get("action") == "sell" and t.get("proceeds", 0) > t.get("cost", 0)
            ]
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
            sortino_ratio = (
                mean_return / downside_deviation if downside_deviation > 0 else 0
            )

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
                values = [
                    item.get(metric)
                    for item in comparison_data
                    if item.get(metric) is not None
                ]
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

    async def run_multi_factor_backtest(
        self,
        symbols: list[str],
        start_date: str,
        end_date: str,
        initial_capital: float = 100000.0,
        commission_rate: float = 0.001,
        slippage_rate: float = 0.001,
        factor_weights: dict[str, float] | None = None,
        use_cache: bool = True,
    ) -> dict[str, Any]:
        """运行多因子回测

        Args:
            symbols: 股票代码列表
            start_date: 开始日期 (YYYY-MM-DD)
            end_date: 结束日期 (YYYY-MM-DD)
            initial_capital: 初始资金
            commission_rate: 手续费率
            slippage_rate: 滑点率
            factor_weights: 可选的因子权重配置
            use_cache: 是否使用缓存

        Returns:
            回测结果字典
        """
        try:
            # 准备多因子策略参数
            strategy_params = {}

            # 如果提供了自定义权重，添加到策略参数中
            if factor_weights is not None:
                # 验证权重配置
                self._validate_factor_weights(factor_weights)
                strategy_params["factor_weights"] = factor_weights
                logger.info(f"使用自定义因子权重: {factor_weights}")
            else:
                logger.info("使用默认因子权重配置")

            # 构建缓存键（包含权重信息）
            cache_key = self._build_multi_factor_cache_key(
                symbols,
                start_date,
                end_date,
                initial_capital,
                commission_rate,
                slippage_rate,
                factor_weights,
            )

            # 尝试从缓存获取
            if use_cache:
                cached_result = self.cache_repo.get(
                    CacheType.BACKTEST_RESULT, cache_key, serialize_method="json"
                )
                if cached_result is not None:
                    logger.info("从缓存获取多因子回测结果")
                    return cached_result

            # 创建多因子策略实例
            strategy = self._create_strategy(StrategyType.MULTI_FACTOR, strategy_params)
            if not strategy:
                raise BacktestError("无法创建多因子策略实例")

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
                "strategy_type": StrategyType.MULTI_FACTOR.value,
                "strategy_params": strategy_params,
                "factor_weights": factor_weights,  # 记录使用的权重配置
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
                f"多因子回测完成, 总收益率: {performance_metrics.get('total_return', 0):.2%}"
            )
            return complete_result

        except Exception as e:
            logger.error(f"多因子回测执行失败: {e}")
            raise BacktestError(f"多因子回测执行失败: {e}") from e

    def _validate_factor_weights(self, factor_weights: dict[str, float]) -> None:
        """验证因子权重配置

        Args:
            factor_weights: 因子权重字典

        Raises:
            BacktestError: 权重配置无效时抛出
        """
        try:
            # 检查必需的因子维度
            required_factors = ["technical", "fundamental", "news", "market"]
            for factor in required_factors:
                if factor not in factor_weights:
                    raise BacktestError(f"缺少必需的因子权重: {factor}")

            # 检查权重值范围
            for factor, weight in factor_weights.items():
                if not isinstance(weight, int | float):
                    raise BacktestError(f"因子权重必须是数值类型: {factor}={weight}")
                if weight < 0 or weight > 1:
                    raise BacktestError(f"因子权重必须在0-1之间: {factor}={weight}")

            # 检查权重总和
            total_weight = sum(factor_weights.values())
            if abs(total_weight - 1.0) > 0.001:  # 允许小的浮点误差
                raise BacktestError(f"因子权重总和必须为1.0, 当前为: {total_weight}")

            logger.info("因子权重配置验证通过")

        except Exception as e:
            logger.error(f"因子权重验证失败: {e}")
            raise BacktestError(f"因子权重验证失败: {e}") from e

    def _build_multi_factor_cache_key(
        self,
        symbols: list[str],
        start_date: str,
        end_date: str,
        initial_capital: float,
        commission_rate: float,
        slippage_rate: float,
        factor_weights: dict[str, float] | None,
    ) -> str:
        """构建多因子回测缓存键

        Args:
            symbols: 股票代码列表
            start_date: 开始日期
            end_date: 结束日期
            initial_capital: 初始资金
            commission_rate: 手续费率
            slippage_rate: 滑点率
            factor_weights: 因子权重配置

        Returns:
            缓存键字符串
        """
        symbols_str = "-".join(sorted(symbols))

        # 处理权重配置
        if factor_weights is not None:
            weights_str = "_".join(
                f"{k}={v:.3f}" for k, v in sorted(factor_weights.items())
            )
        else:
            weights_str = "default"

        return (
            f"multi_factor_backtest_{symbols_str}_{start_date}_{end_date}_"
            f"{initial_capital}_{commission_rate}_{slippage_rate}_{weights_str}"
        )
