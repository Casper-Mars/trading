"""辅助函数模块"""

import hashlib
import json
import uuid
from datetime import date, datetime, timedelta
from decimal import ROUND_HALF_UP, Decimal
from typing import Any

import pandas as pd


def generate_uuid() -> str:
    """生成UUID

    Returns:
        str: UUID字符串
    """
    return str(uuid.uuid4())


def generate_short_id(length: int = 8) -> str:
    """生成短ID

    Args:
        length: ID长度

    Returns:
        str: 短ID字符串
    """
    return str(uuid.uuid4()).replace("-", "")[:length]


def calculate_hash(text: str) -> str:
    """计算SHA256哈希值

    Args:
        text: 待计算的文本

    Returns:
        str: SHA256哈希值
    """
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def safe_json_loads(json_str: str, default: Any = None) -> Any:
    """安全的JSON解析

    Args:
        json_str: JSON字符串
        default: 解析失败时的默认值

    Returns:
        Any: 解析结果或默认值
    """
    try:
        return json.loads(json_str)
    except (json.JSONDecodeError, TypeError):
        return default


def safe_json_dumps(obj: Any, default: str = "{}") -> str:
    """安全的JSON序列化

    Args:
        obj: 待序列化的对象
        default: 序列化失败时的默认值

    Returns:
        str: JSON字符串或默认值
    """
    try:
        return json.dumps(obj, ensure_ascii=False, default=str)
    except (TypeError, ValueError):
        return default


def format_datetime(dt: datetime, format_str: str = "%Y-%m-%d %H:%M:%S") -> str:
    """格式化日期时间

    Args:
        dt: 日期时间对象
        format_str: 格式字符串

    Returns:
        str: 格式化后的日期时间字符串
    """
    if dt is None:
        return ""
    return dt.strftime(format_str)


def parse_datetime(
    dt_str: str, format_str: str = "%Y-%m-%d %H:%M:%S"
) -> datetime | None:
    """解析日期时间字符串

    Args:
        dt_str: 日期时间字符串
        format_str: 格式字符串

    Returns:
        Optional[datetime]: 解析后的日期时间对象
    """
    try:
        return datetime.strptime(dt_str, format_str)
    except (ValueError, TypeError):
        return None


def get_trading_days(start_date: date, end_date: date) -> list[date]:
    """获取交易日列表(排除周末)

    Args:
        start_date: 开始日期
        end_date: 结束日期

    Returns:
        List[date]: 交易日列表
    """
    trading_days = []
    current_date = start_date

    while current_date <= end_date:
        # 排除周末(周六=5, 周日=6)
        if current_date.weekday() < 5:
            trading_days.append(current_date)
        current_date += timedelta(days=1)

    return trading_days


def round_decimal(value: float | Decimal, places: int = 2) -> Decimal:
    """四舍五入到指定小数位

    Args:
        value: 待处理的数值
        places: 小数位数

    Returns:
        Decimal: 处理后的数值
    """
    if isinstance(value, float) or not isinstance(value, Decimal):
        value = Decimal(str(value))

    return value.quantize(Decimal("0." + "0" * places), rounding=ROUND_HALF_UP)


def calculate_percentage_change(old_value: float, new_value: float) -> float:
    """计算百分比变化

    Args:
        old_value: 原值
        new_value: 新值

    Returns:
        float: 百分比变化(如：0.05表示5%的增长)
    """
    if old_value == 0:
        return 0.0 if new_value == 0 else float("inf")

    return (new_value - old_value) / old_value


def flatten_dict(
    d: dict[str, Any], parent_key: str = "", sep: str = "."
) -> dict[str, Any]:
    """扁平化字典

    Args:
        d: 待扁平化的字典
        parent_key: 父键名
        sep: 分隔符

    Returns:
        Dict[str, Any]: 扁平化后的字典
    """
    items = []
    for k, v in d.items():
        new_key = f"{parent_key}{sep}{k}" if parent_key else k
        if isinstance(v, dict):
            items.extend(flatten_dict(v, new_key, sep=sep).items())
        else:
            items.append((new_key, v))
    return dict(items)


def chunk_list(lst: list[Any], chunk_size: int) -> list[list[Any]]:
    """将列表分块

    Args:
        lst: 待分块的列表
        chunk_size: 块大小

    Returns:
        List[List[Any]]: 分块后的列表
    """
    return [lst[i : i + chunk_size] for i in range(0, len(lst), chunk_size)]


def remove_duplicates(lst: list[Any], key_func=None) -> list[Any]:
    """去除列表中的重复项

    Args:
        lst: 待处理的列表
        key_func: 用于确定唯一性的键函数

    Returns:
        List[Any]: 去重后的列表
    """
    if key_func is None:
        return list(dict.fromkeys(lst))

    seen = set()
    result = []
    for item in lst:
        key = key_func(item)
        if key not in seen:
            seen.add(key)
            result.append(item)
    return result


def safe_divide(numerator: float, denominator: float, default: float = 0.0) -> float:
    """安全除法

    Args:
        numerator: 分子
        denominator: 分母
        default: 除零时的默认值

    Returns:
        float: 除法结果或默认值
    """
    try:
        return numerator / denominator if denominator != 0 else default
    except (TypeError, ZeroDivisionError):
        return default


def convert_to_dataframe(data: list[dict[str, Any]]) -> pd.DataFrame:
    """将字典列表转换为DataFrame

    Args:
        data: 字典列表

    Returns:
        pd.DataFrame: DataFrame对象
    """
    if not data:
        return pd.DataFrame()

    return pd.DataFrame(data)


def merge_dicts(*dicts: dict[str, Any]) -> dict[str, Any]:
    """合并多个字典

    Args:
        *dicts: 待合并的字典

    Returns:
        Dict[str, Any]: 合并后的字典
    """
    result = {}
    for d in dicts:
        if isinstance(d, dict):
            result.update(d)
    return result


def get_nested_value(
    data: dict[str, Any], key_path: str, default: Any = None, sep: str = "."
) -> Any:
    """获取嵌套字典中的值

    Args:
        data: 数据字典
        key_path: 键路径，如 'a.b.c'
        default: 默认值
        sep: 分隔符

    Returns:
        Any: 获取到的值或默认值
    """
    keys = key_path.split(sep)
    current = data

    try:
        for key in keys:
            current = current[key]
        return current
    except (KeyError, TypeError):
        return default


def set_nested_value(
    data: dict[str, Any], key_path: str, value: Any, sep: str = "."
) -> None:
    """设置嵌套字典中的值

    Args:
        data: 数据字典
        key_path: 键路径，如 'a.b.c'
        value: 要设置的值
        sep: 分隔符
    """
    keys = key_path.split(sep)
    current = data

    for key in keys[:-1]:
        if key not in current or not isinstance(current[key], dict):
            current[key] = {}
        current = current[key]

    current[keys[-1]] = value
