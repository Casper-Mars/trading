"""持仓管理服务"""

from datetime import date
from decimal import Decimal
from typing import Any

from loguru import logger

from clients.data_collection_client import DataCollectionClient
from models.enums import PositionStatus, PositionType
from models.schemas import (
    PositionCreate,
    PositionResponse,
    PositionSummary,
    PositionUpdate,
)
from repositories.position_repo import PositionRepo
from utils.exceptions import (
    BusinessError,
    ExternalServiceError,
    NotFoundError,
    ValidationError,
)


class PositionService:
    """持仓管理服务

    提供持仓的业务逻辑处理，包括：
    - 业务校验
    - 组合指标计算
    - 价格更新和市值计算
    - 持仓CRUD操作
    """

    def __init__(
        self,
        position_repo: PositionRepo | None = None,
        data_client: DataCollectionClient | None = None,
    ):
        """初始化持仓服务

        Args:
            position_repo: 持仓数据仓库
            data_client: 数据采集客户端
        """
        self.position_repo = position_repo or PositionRepo()
        self.data_client = data_client or DataCollectionClient()

    def create_position(self, position_data: PositionCreate) -> PositionResponse:
        """创建新持仓

        Args:
            position_data: 持仓创建数据

        Returns:
            创建的持仓响应

        Raises:
            ValidationError: 数据验证失败
            BusinessError: 业务规则验证失败
        """
        # 业务校验
        self._validate_position_create(position_data)

        try:
            # 创建持仓
            position = self.position_repo.create(position_data)

            # 尝试更新当前价格
            try:
                self._update_position_price(position.symbol)
            except ExternalServiceError:
                logger.warning(f"创建持仓后更新价格失败: {position.symbol}")

            # 重新获取更新后的持仓
            updated_position = self.position_repo.get_by_id(position.id)
            if not updated_position:
                raise BusinessError("创建持仓后获取失败")

            return PositionResponse.model_validate(updated_position)

        except Exception as e:
            logger.error(f"创建持仓失败: {e}")
            raise BusinessError(f"创建持仓失败: {e}") from e

    def get_position(self, position_id: int) -> PositionResponse:
        """获取持仓详情

        Args:
            position_id: 持仓ID

        Returns:
            持仓响应

        Raises:
            NotFoundError: 持仓不存在
        """
        position = self.position_repo.get_by_id(position_id)
        if not position:
            raise NotFoundError(f"持仓不存在: ID={position_id}")

        return PositionResponse.model_validate(position)

    def update_position(self, position_id: int, update_data: PositionUpdate) -> PositionResponse:
        """更新持仓

        Args:
            position_id: 持仓ID
            update_data: 更新数据

        Returns:
            更新后的持仓响应

        Raises:
            NotFoundError: 持仓不存在
            ValidationError: 数据验证失败
            BusinessError: 业务规则验证失败
        """
        # 业务校验
        self._validate_position_update(update_data)

        try:
            # 更新持仓
            position = self.position_repo.update(position_id, update_data)
            if not position:
                raise NotFoundError(f"持仓不存在: ID={position_id}")

            return PositionResponse.model_validate(position)

        except NotFoundError:
            raise
        except Exception as e:
            logger.error(f"更新持仓失败: ID={position_id}, 错误: {e}")
            raise BusinessError(f"更新持仓失败: {e}") from e

    def delete_position(self, position_id: int) -> bool:
        """删除持仓

        Args:
            position_id: 持仓ID

        Returns:
            是否删除成功

        Raises:
            NotFoundError: 持仓不存在
            BusinessError: 业务规则验证失败
        """
        # 检查持仓是否存在
        position = self.position_repo.get_by_id(position_id)
        if not position:
            raise NotFoundError(f"持仓不存在: ID={position_id}")

        # 业务校验：只能删除已平仓的持仓
        if position.status == PositionStatus.ACTIVE:
            raise BusinessError("不能删除活跃持仓,请先平仓")

        try:
            return self.position_repo.delete(position_id)
        except Exception as e:
            logger.error(f"删除持仓失败: ID={position_id}, 错误: {e}")
            raise BusinessError(f"删除持仓失败: {e}") from e

    def get_active_positions(self) -> list[PositionResponse]:
        """获取所有活跃持仓

        Returns:
            活跃持仓列表
        """
        positions = self.position_repo.get_active_positions()
        return [PositionResponse.model_validate(pos) for pos in positions]

    def get_positions_by_type(self, position_type: PositionType) -> list[PositionResponse]:
        """根据持仓类型获取持仓列表

        Args:
            position_type: 持仓类型

        Returns:
            持仓列表
        """
        positions = self.position_repo.get_positions_by_type(position_type)
        return [PositionResponse.model_validate(pos) for pos in positions]

    def get_portfolio_summary(self) -> PositionSummary:
        """获取投资组合汇总信息

        Returns:
            投资组合汇总
        """
        try:
            # 获取基础汇总数据
            summary_data = self.position_repo.get_portfolio_summary()

            # 计算总成本
            active_positions = self.position_repo.get_active_positions()
            total_cost = sum(
                pos.quantity * pos.avg_cost for pos in active_positions
            )

            # 计算总收益率
            total_return_rate = Decimal("0")
            if total_cost > 0:
                total_pnl = Decimal(str(summary_data["total_unrealized_pnl"])) + Decimal(
                    str(summary_data["total_realized_pnl"])
                )
                total_return_rate = (total_pnl / total_cost) * 100

            # 统计已平仓持仓数
            closed_positions = len([
                pos for pos in self.position_repo.get_positions_by_type(PositionType.LONG)
                + self.position_repo.get_positions_by_type(PositionType.SHORT)
                if pos.status == PositionStatus.CLOSED
            ])

            return PositionSummary(
                total_positions=summary_data["total_positions"],
                total_market_value=Decimal(str(summary_data["total_market_value"])),
                total_cost=total_cost,
                total_unrealized_pnl=Decimal(str(summary_data["total_unrealized_pnl"])),
                total_realized_pnl=Decimal(str(summary_data["total_realized_pnl"])),
                total_return_rate=total_return_rate,
                active_positions=summary_data["total_positions"],
                closed_positions=closed_positions,
            )

        except Exception as e:
            logger.error(f"获取投资组合汇总失败: {e}")
            # 返回默认值
            return PositionSummary(
                total_positions=0,
                total_market_value=Decimal("0"),
                total_cost=Decimal("0"),
                total_unrealized_pnl=Decimal("0"),
                total_realized_pnl=Decimal("0"),
                total_return_rate=Decimal("0"),
                active_positions=0,
                closed_positions=0,
            )

    def update_all_prices(self) -> dict[str, Any]:
        """更新所有活跃持仓的当前价格

        Returns:
            更新结果统计
        """
        try:
            # 获取所有活跃持仓的股票代码
            active_positions = self.position_repo.get_active_positions()
            symbols = list({pos.symbol for pos in active_positions})

            if not symbols:
                return {
                    "total_symbols": 0,
                    "updated_symbols": 0,
                    "failed_symbols": [],
                    "updated_positions": 0,
                }

            # 批量获取最新价格
            price_updates = {}
            failed_symbols = []

            for symbol in symbols:
                try:
                    price = self._get_latest_price(symbol)
                    if price:
                        price_updates[symbol] = price
                except ExternalServiceError:
                    failed_symbols.append(symbol)
                    logger.warning(f"获取股票价格失败: {symbol}")

            # 批量更新价格
            updated_positions = 0
            if price_updates:
                updated_positions = self.position_repo.update_current_prices(price_updates)

            result = {
                "total_symbols": len(symbols),
                "updated_symbols": len(price_updates),
                "failed_symbols": failed_symbols,
                "updated_positions": updated_positions,
            }

            logger.info(f"批量更新价格完成: {result}")
            return result

        except Exception as e:
            logger.error(f"批量更新价格失败: {e}")
            raise BusinessError(f"批量更新价格失败: {e}") from e

    def update_position_price(self, symbol: str) -> bool:
        """更新指定股票的持仓价格

        Args:
            symbol: 股票代码

        Returns:
            是否更新成功
        """
        try:
            return self._update_position_price(symbol)
        except Exception as e:
            logger.error(f"更新持仓价格失败: symbol={symbol}, 错误: {e}")
            return False

    def close_position(self, position_id: int, close_price: Decimal, close_date: date | None = None) -> PositionResponse:
        """平仓操作

        Args:
            position_id: 持仓ID
            close_price: 平仓价格
            close_date: 平仓日期，默认为今天

        Returns:
            更新后的持仓响应

        Raises:
            NotFoundError: 持仓不存在
            BusinessError: 业务规则验证失败
        """
        # 获取持仓
        position = self.position_repo.get_by_id(position_id)
        if not position:
            raise NotFoundError(f"持仓不存在: ID={position_id}")

        # 业务校验
        if position.status != PositionStatus.ACTIVE:
            raise BusinessError("只能平仓活跃持仓")

        if close_price <= 0:
            raise ValidationError("平仓价格必须大于0")

        try:
            # 计算已实现盈亏
            realized_pnl = position.quantity * (close_price - position.avg_cost)

            # 更新持仓状态
            update_data = PositionUpdate(
                current_price=close_price,
                status=PositionStatus.CLOSED,
                close_date=close_date or date.today(),
            )

            updated_position = self.position_repo.update(position_id, update_data)
            if not updated_position:
                raise BusinessError("平仓更新失败")

            # 更新已实现盈亏
            updated_position.realized_pnl = realized_pnl
            updated_position.unrealized_pnl = Decimal("0")  # 平仓后无浮动盈亏

            logger.info(f"平仓成功: {position.symbol}, 已实现盈亏: {realized_pnl}")
            return PositionResponse.model_validate(updated_position)

        except Exception as e:
            logger.error(f"平仓失败: ID={position_id}, 错误: {e}")
            raise BusinessError(f"平仓失败: {e}") from e

    def _validate_position_create(self, position_data: PositionCreate) -> None:
        """验证创建持仓数据

        Args:
            position_data: 持仓创建数据

        Raises:
            ValidationError: 数据验证失败
            BusinessError: 业务规则验证失败
        """
        # 基础数据验证
        if not position_data.symbol or not position_data.symbol.strip():
            raise ValidationError("股票代码不能为空")

        if not position_data.name or not position_data.name.strip():
            raise ValidationError("股票名称不能为空")

        if position_data.quantity <= 0:
            raise ValidationError("持仓数量必须大于0")

        if position_data.avg_cost <= 0:
            raise ValidationError("平均成本必须大于0")

        # 开仓日期不能是未来
        if position_data.open_date > date.today():
            raise ValidationError("开仓日期不能是未来日期")

        # 检查是否已存在相同股票的活跃持仓
        existing_positions = self.position_repo.get_by_symbol(position_data.symbol)
        active_positions = [
            pos for pos in existing_positions if pos.status == PositionStatus.ACTIVE
        ]

        if active_positions:
            logger.warning(f"股票 {position_data.symbol} 已存在活跃持仓")
            # 这里可以选择合并持仓或者抛出异常，根据业务需求决定
            # raise BusinessError(f"股票 {position_data.symbol} 已存在活跃持仓")

    def _validate_position_update(self, update_data: PositionUpdate) -> None:
        """验证更新持仓数据

        Args:
            update_data: 持仓更新数据

        Raises:
            ValidationError: 数据验证失败
        """
        if update_data.quantity is not None and update_data.quantity <= 0:
            raise ValidationError("持仓数量必须大于0")

        if update_data.avg_cost is not None and update_data.avg_cost <= 0:
            raise ValidationError("平均成本必须大于0")

        if update_data.current_price is not None and update_data.current_price <= 0:
            raise ValidationError("当前价格必须大于0")

        if update_data.close_date is not None and update_data.close_date > date.today():
            raise ValidationError("平仓日期不能是未来日期")

    def _update_position_price(self, symbol: str) -> bool:
        """更新指定股票的持仓价格

        Args:
            symbol: 股票代码

        Returns:
            是否更新成功

        Raises:
            ExternalServiceError: 外部服务调用失败
        """
        try:
            # 获取最新价格
            latest_price = self._get_latest_price(symbol)
            if not latest_price:
                return False

            # 更新价格
            updated_count = self.position_repo.update_current_prices({symbol: latest_price})
            return updated_count > 0

        except ExternalServiceError:
            raise
        except Exception as e:
            logger.error(f"更新持仓价格失败: symbol={symbol}, 错误: {e}")
            return False

    def _get_latest_price(self, symbol: str) -> Decimal | None:
        """获取股票最新价格

        Args:
            symbol: 股票代码

        Returns:
            最新价格，获取失败返回None

        Raises:
            ExternalServiceError: 外部服务调用失败
        """
        try:
            # 调用数据采集系统获取最新行情
            response = self.data_client.get_latest_market_data(symbol)

            # 解析响应数据
            if response.get("code") == 200 and response.get("data"):
                market_data = response["data"]
                close_price = market_data.get("close")
                if close_price is not None:
                    return Decimal(str(close_price))

            logger.warning(f"获取股票价格失败: {symbol}, 响应: {response}")
            return None

        except Exception as e:
            logger.error(f"获取股票价格异常: symbol={symbol}, 错误: {e}")
            raise ExternalServiceError(f"获取股票价格失败: {e}") from e


# 全局持仓服务实例
position_service = PositionService()
