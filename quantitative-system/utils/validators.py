"""数据验证工具"""

import re
from datetime import date, datetime
from decimal import Decimal
from typing import Any

from .exceptions import DataValidationError


def validate_stock_code(stock_code: str) -> bool:
    """验证股票代码格式

    Args:
        stock_code: 股票代码，如 '000001.SZ', '600000.SH'

    Returns:
        bool: 验证结果

    Raises:
        DataValidationError: 验证失败时抛出
    """
    if not stock_code:
        raise DataValidationError("股票代码不能为空")

    # 匹配格式：6位数字.交易所代码
    pattern = r"^\d{6}\.(SH|SZ|BJ)$"
    if not re.match(pattern, stock_code.upper()):
        raise DataValidationError(
            f"股票代码格式错误: {stock_code}，应为6位数字.交易所代码"
        )

    return True


def validate_date_range(
    start_date: str | date | datetime, end_date: str | date | datetime
) -> bool:
    """验证日期范围

    Args:
        start_date: 开始日期
        end_date: 结束日期

    Returns:
        bool: 验证结果

    Raises:
        DataValidationError: 验证失败时抛出
    """
    # 转换为date对象
    if isinstance(start_date, str):
        try:
            start_date = datetime.strptime(start_date, "%Y-%m-%d").date()
        except ValueError:
            raise DataValidationError(f"开始日期格式错误: {start_date}，应为YYYY-MM-DD")
    elif isinstance(start_date, datetime):
        start_date = start_date.date()

    if isinstance(end_date, str):
        try:
            end_date = datetime.strptime(end_date, "%Y-%m-%d").date()
        except ValueError:
            raise DataValidationError(f"结束日期格式错误: {end_date}，应为YYYY-MM-DD")
    elif isinstance(end_date, datetime):
        end_date = end_date.date()

    if start_date > end_date:
        raise DataValidationError("开始日期不能晚于结束日期")

    return True


def validate_positive_number(
    value: int | float | Decimal, field_name: str = "数值"
) -> bool:
    """验证正数

    Args:
        value: 待验证的数值
        field_name: 字段名称，用于错误提示

    Returns:
        bool: 验证结果

    Raises:
        DataValidationError: 验证失败时抛出
    """
    if value is None:
        raise DataValidationError(f"{field_name}不能为空")

    if not isinstance(value, (int, float, Decimal)):
        raise DataValidationError(f"{field_name}必须是数字类型")

    if value <= 0:
        raise DataValidationError(f"{field_name}必须是正数")

    return True


def validate_percentage(
    value: int | float | Decimal, field_name: str = "百分比"
) -> bool:
    """验证百分比（0-100）

    Args:
        value: 待验证的百分比值
        field_name: 字段名称，用于错误提示

    Returns:
        bool: 验证结果

    Raises:
        DataValidationError: 验证失败时抛出
    """
    if value is None:
        raise DataValidationError(f"{field_name}不能为空")

    if not isinstance(value, (int, float, Decimal)):
        raise DataValidationError(f"{field_name}必须是数字类型")

    if not (0 <= value <= 100):
        raise DataValidationError(f"{field_name}必须在0-100之间")

    return True


def validate_strategy_name(strategy_name: str) -> bool:
    """验证策略名称

    Args:
        strategy_name: 策略名称

    Returns:
        bool: 验证结果

    Raises:
        DataValidationError: 验证失败时抛出
    """
    if not strategy_name:
        raise DataValidationError("策略名称不能为空")

    if len(strategy_name) > 100:
        raise DataValidationError("策略名称长度不能超过100个字符")

    # 只允许中文、英文、数字、下划线、连字符
    pattern = r"^[\u4e00-\u9fa5a-zA-Z0-9_-]+$"
    if not re.match(pattern, strategy_name):
        raise DataValidationError("策略名称只能包含中文、英文、数字、下划线、连字符")

    return True


def validate_email(email: str) -> bool:
    """验证邮箱格式

    Args:
        email: 邮箱地址

    Returns:
        bool: 验证结果

    Raises:
        DataValidationError: 验证失败时抛出
    """
    if not email:
        raise DataValidationError("邮箱地址不能为空")

    pattern = r"^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$"
    if not re.match(pattern, email):
        raise DataValidationError(f"邮箱格式错误: {email}")

    return True


def validate_required_fields(data: dict, required_fields: list[str]) -> bool:
    """验证必填字段

    Args:
        data: 待验证的数据字典
        required_fields: 必填字段列表

    Returns:
        bool: 验证结果

    Raises:
        DataValidationError: 验证失败时抛出
    """
    missing_fields = []

    for field in required_fields:
        if field not in data or data[field] is None or data[field] == "":
            missing_fields.append(field)

    if missing_fields:
        raise DataValidationError(f"缺少必填字段: {', '.join(missing_fields)}")

    return True


def validate_json_structure(data: Any, expected_keys: list[str]) -> bool:
    """验证JSON结构

    Args:
        data: 待验证的数据
        expected_keys: 期望的键列表

    Returns:
        bool: 验证结果

    Raises:
        DataValidationError: 验证失败时抛出
    """
    if not isinstance(data, dict):
        raise DataValidationError("数据必须是字典类型")

    missing_keys = [key for key in expected_keys if key not in data]
    if missing_keys:
        raise DataValidationError(f"缺少必要的键: {', '.join(missing_keys)}")

    return True


def validate_list_not_empty(data: list[Any], field_name: str = "列表") -> bool:
    """验证列表非空

    Args:
        data: 待验证的列表
        field_name: 字段名称，用于错误提示

    Returns:
        bool: 验证结果

    Raises:
        DataValidationError: 验证失败时抛出
    """
    if not isinstance(data, list):
        raise DataValidationError(f"{field_name}必须是列表类型")

    if len(data) == 0:
        raise DataValidationError(f"{field_name}不能为空")

    return True
