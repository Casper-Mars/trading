"""数据质量服务模块

提供数据质量验证、清洗和标准化功能。
确保数据的完整性、准确性和一致性。
"""

from datetime import date, datetime
from typing import Any

from loguru import logger

from models.database import FinancialData, StockBasicInfo, StockDailyData
from utils.exceptions import ValidationError


class QualityService:
    """数据质量服务

    提供数据验证、清洗和标准化功能。
    包含数据完整性检查、格式验证、异常值检测等。
    """

    def __init__(self):
        """初始化数据质量服务"""
        logger.info("数据质量服务初始化完成")

    def validate_stock_basic_info(self, stock_data: StockBasicInfo) -> bool:
        """验证股票基础信息

        Args:
            stock_data: 股票基础信息

        Returns:
            验证是否通过

        Raises:
            ValidationError: 验证失败时
        """
        try:
            # 必填字段检查
            required_fields = ['ts_code', 'symbol', 'name', 'area', 'industry', 'market']
            for field in required_fields:
                value = getattr(stock_data, field, None)
                if not value or (isinstance(value, str) and not value.strip()):
                    raise ValidationError(f"股票基础信息缺少必填字段: {field}")

            # 股票代码格式检查
            ts_code = stock_data.ts_code
            if not self._is_valid_ts_code(ts_code):
                raise ValidationError(f"股票代码格式无效: {ts_code}")

            # 上市状态检查
            if hasattr(stock_data, 'list_status') and stock_data.list_status and stock_data.list_status not in ['L', 'D', 'P']:
                raise ValidationError(f"上市状态无效: {stock_data.list_status}")

            # 市场检查
            if hasattr(stock_data, 'market') and stock_data.market and stock_data.market not in ['主板', '创业板', '科创板', '北交所', '新三板']:
                logger.warning(f"未知市场类型: {stock_data.market}")

            logger.debug(f"股票基础信息验证通过: {ts_code}")
            return True

        except ValidationError:
            raise
        except Exception as e:
            error_msg = f"股票基础信息验证异常: {e}"
            logger.error(error_msg)
            raise ValidationError(error_msg) from e

    def validate_daily_data(self, daily_data: StockDailyData) -> bool:
        """验证股票日线数据

        Args:
            daily_data: 股票日线数据

        Returns:
            验证是否通过

        Raises:
            ValidationError: 验证失败时
        """
        try:
            # 必填字段检查
            required_fields = ['ts_code', 'trade_date', 'open', 'high', 'low', 'close', 'vol']
            for field in required_fields:
                value = getattr(daily_data, field, None)
                if value is None:
                    raise ValidationError(f"日线数据缺少必填字段: {field}")

            # 股票代码格式检查
            ts_code = daily_data.ts_code
            if not self._is_valid_ts_code(ts_code):
                raise ValidationError(f"股票代码格式无效: {ts_code}")

            # 交易日期检查
            trade_date = daily_data.trade_date
            if isinstance(trade_date, str):
                try:
                    trade_date = datetime.strptime(trade_date, "%Y%m%d").date()
                except ValueError as e:
                    raise ValidationError(f"交易日期格式无效: {daily_data.trade_date}") from e

            if trade_date > date.today():
                raise ValidationError(f"交易日期不能大于今天: {trade_date}")

            # 价格数据合理性检查
            open_price = float(daily_data.open or 0)
            high_price = float(daily_data.high or 0)
            low_price = float(daily_data.low or 0)
            close_price = float(daily_data.close or 0)

            if any(price <= 0 for price in [open_price, high_price, low_price, close_price]):
                raise ValidationError(f"价格数据不能为负数或零: {ts_code} {trade_date}")

            if high_price < max(open_price, close_price, low_price):
                raise ValidationError(f"最高价不能小于开盘价、收盘价或最低价: {ts_code} {trade_date}")

            if low_price > min(open_price, close_price, high_price):
                raise ValidationError(f"最低价不能大于开盘价、收盘价或最高价: {ts_code} {trade_date}")

            # 成交量检查
            volume = float(daily_data.vol or 0)
            if volume < 0:
                raise ValidationError(f"成交量不能为负数: {ts_code} {trade_date}")

            # 异常波动检查（涨跌幅超过20%给出警告）
            if open_price > 0:
                change_pct = abs(close_price - open_price) / open_price
                if change_pct > 0.2:
                    logger.warning(f"价格异常波动: {ts_code} {trade_date}, 涨跌幅: {change_pct:.2%}")

            logger.debug(f"日线数据验证通过: {ts_code} {trade_date}")
            return True

        except ValidationError:
            raise
        except Exception as e:
            error_msg = f"日线数据验证异常: {e}"
            logger.error(error_msg)
            raise ValidationError(error_msg) from e

    def validate_financial_data(self, financial_data: FinancialData) -> bool:
        """验证财务数据

        Args:
            financial_data: 财务数据

        Returns:
            验证是否通过

        Raises:
            ValidationError: 验证失败时
        """
        try:
            # 必填字段检查
            required_fields = ['ts_code', 'ann_date', 'end_date']
            for field in required_fields:
                value = getattr(financial_data, field, None)
                if not value:
                    raise ValidationError(f"财务数据缺少必填字段: {field}")

            # 股票代码格式检查
            ts_code = financial_data.ts_code
            if not self._is_valid_ts_code(ts_code):
                raise ValidationError(f"股票代码格式无效: {ts_code}")

            # 日期格式检查
            ann_date = financial_data.ann_date
            end_date = financial_data.end_date

            if isinstance(ann_date, str):
                try:
                    ann_date = datetime.strptime(ann_date, "%Y%m%d").date()
                except ValueError as e:
                    raise ValidationError(f"公告日期格式无效: {financial_data.ann_date}") from e

            if isinstance(end_date, str):
                try:
                    end_date = datetime.strptime(end_date, "%Y%m%d").date()
                except ValueError as e:
                    raise ValidationError(f"报告期结束日期格式无效: {financial_data.end_date}") from e

            # 日期逻辑检查
            if ann_date < end_date:
                logger.warning(f"公告日期早于报告期结束日期: {ts_code} {ann_date} < {end_date}")

            if end_date > date.today():
                raise ValidationError(f"报告期结束日期不能大于今天: {end_date}")

            # 财务指标合理性检查
            self._validate_financial_metrics(financial_data)

            logger.debug(f"财务数据验证通过: {ts_code} {end_date}")
            return True

        except ValidationError:
            raise
        except Exception as e:
            error_msg = f"财务数据验证异常: {e}"
            logger.error(error_msg)
            raise ValidationError(error_msg) from e

    def _is_valid_ts_code(self, ts_code: str) -> bool:
        """验证股票代码格式

        Args:
            ts_code: 股票代码

        Returns:
            是否有效
        """
        if not ts_code or not isinstance(ts_code, str):
            return False

        # 格式：6位数字.交易所代码
        parts = ts_code.split('.')
        if len(parts) != 2:
            return False

        code, exchange = parts
        if not code.isdigit() or len(code) != 6:
            return False

        return exchange in ['SH', 'SZ', 'BJ']

    def _validate_financial_metrics(self, financial_data: FinancialData) -> None:
        """验证财务指标的合理性

        Args:
            financial_data: 财务数据

        Raises:
            ValidationError: 验证失败时
        """
        ts_code = financial_data.ts_code
        end_date = financial_data.end_date

        # 检查营业收入
        if hasattr(financial_data, 'revenue') and financial_data.revenue is not None:
            revenue = float(financial_data.revenue)
            if revenue < 0:
                logger.warning(f"营业收入为负数: {ts_code} {end_date}, 收入: {revenue}")

        # 检查净利润
        if hasattr(financial_data, 'n_income') and financial_data.n_income is not None:
            net_income = float(financial_data.n_income)
            # 净利润可以为负，但给出警告
            if net_income < 0:
                logger.debug(f"净利润为负数: {ts_code} {end_date}, 净利润: {net_income}")

        # 检查总资产
        if hasattr(financial_data, 'total_assets') and financial_data.total_assets is not None:
            total_assets = float(financial_data.total_assets)
            if total_assets <= 0:
                raise ValidationError(f"总资产必须大于零: {ts_code} {end_date}, 总资产: {total_assets}")

        # 检查总负债
        if hasattr(financial_data, 'total_liab') and financial_data.total_liab is not None:
            total_liab = float(financial_data.total_liab)
            if total_liab < 0:
                logger.warning(f"总负债为负数: {ts_code} {end_date}, 总负债: {total_liab}")

        # 检查资产负债率
        if (hasattr(financial_data, 'total_assets') and financial_data.total_assets is not None and
                hasattr(financial_data, 'total_liab') and financial_data.total_liab is not None):
            total_assets = float(financial_data.total_assets)
            total_liab = float(financial_data.total_liab)
            if total_assets > 0:
                debt_ratio = total_liab / total_assets
                if debt_ratio > 1.0:
                    logger.warning(f"资产负债率超过100%: {ts_code} {end_date}, 比率: {debt_ratio:.2%}")
                elif debt_ratio < 0:
                    logger.warning(f"资产负债率为负数: {ts_code} {end_date}, 比率: {debt_ratio:.2%}")

    def clean_data(self, data: dict[str, Any]) -> dict[str, Any]:
        """清洗数据

        Args:
            data: 原始数据

        Returns:
            清洗后的数据
        """
        cleaned_data = {}
        for key, value in data.items():
            # 处理空值
            if value is None or value == '' or value == 'None':
                cleaned_data[key] = None
            # 处理字符串
            elif isinstance(value, str):
                cleaned_data[key] = value.strip()
            # 处理数值
            elif isinstance(value, int | float):
                # 处理异常值
                if value == float('inf') or value == float('-inf'):
                    cleaned_data[key] = None
                else:
                    cleaned_data[key] = value
            else:
                cleaned_data[key] = value

        return cleaned_data

    def standardize_data(self, data: dict[str, Any], data_type: str) -> dict[str, Any]:
        """标准化数据格式

        Args:
            data: 原始数据
            data_type: 数据类型 (stock_basic, daily_data, financial_data)

        Returns:
            标准化后的数据
        """
        standardized_data = self.clean_data(data)

        if data_type == 'stock_basic':
            # 标准化股票基础信息
            if standardized_data.get('list_date'):
                standardized_data['list_date'] = self._standardize_date(standardized_data['list_date'])
            if standardized_data.get('delist_date'):
                standardized_data['delist_date'] = self._standardize_date(standardized_data['delist_date'])

        elif data_type == 'daily_data':
            # 标准化日线数据
            if standardized_data.get('trade_date'):
                standardized_data['trade_date'] = self._standardize_date(standardized_data['trade_date'])

        elif data_type == 'financial_data':
            # 标准化财务数据
            if standardized_data.get('ann_date'):
                standardized_data['ann_date'] = self._standardize_date(standardized_data['ann_date'])
            if standardized_data.get('end_date'):
                standardized_data['end_date'] = self._standardize_date(standardized_data['end_date'])

        return standardized_data

    def _standardize_date(self, date_value: Any) -> date | None:
        """标准化日期格式

        Args:
            date_value: 日期值

        Returns:
            标准化的日期对象
        """
        if not date_value:
            return None

        if isinstance(date_value, date):
            return date_value

        if isinstance(date_value, datetime):
            return date_value.date()

        if isinstance(date_value, str):
            try:
                # 尝试解析YYYYMMDD格式
                return datetime.strptime(date_value, "%Y%m%d").date()
            except ValueError:
                try:
                    # 尝试解析YYYY-MM-DD格式
                    return datetime.strptime(date_value, "%Y-%m-%d").date()
                except ValueError:
                    logger.warning(f"无法解析日期格式: {date_value}")
                    return None

        return None

    def validate_news_data(self, data: list[dict]) -> dict:
        """验证新闻数据质量

        Args:
            data: 新闻数据列表

        Returns:
            dict: 验证结果
        """
        if not data:
            return {
                "is_valid": False,
                "total_count": 0,
                "valid_count": 0,
                "error_count": 0,
                "warnings": [],
                "errors": ["新闻数据为空"]
            }

        total_count = len(data)
        valid_count = 0
        errors = []
        warnings = []

        for i, item in enumerate(data):
            item_errors = []
            item_warnings = []

            # 检查必需字段
            required_fields = ["title", "content"]
            for field in required_fields:
                if not item.get(field):
                    item_errors.append(f"第{i+1}条新闻缺少必需字段: {field}")

            # 检查标题长度
            title = item.get("title", "")
            if title and len(title) > 500:
                item_errors.append(f"第{i+1}条新闻标题过长: {len(title)}字符 > 500字符")
            elif title and len(title) < 5:
                item_warnings.append(f"第{i+1}条新闻标题过短: {len(title)}字符 < 5字符")

            # 检查内容长度
            content = item.get("content", "")
            if content and len(content) < 10:
                item_warnings.append(f"第{i+1}条新闻内容过短: {len(content)}字符 < 10字符")

            # 检查来源长度
            source = item.get("source")
            if source and len(source) > 100:
                item_errors.append(f"第{i+1}条新闻来源过长: {len(source)}字符 > 100字符")

            # 检查发布时间格式
            publish_time = item.get("publish_time")
            if publish_time:
                parsed_time = self._parse_date(str(publish_time))
                if not parsed_time:
                    item_errors.append(f"第{i+1}条新闻发布时间格式无效: {publish_time}")
                elif parsed_time > datetime.now():
                    item_warnings.append(f"第{i+1}条新闻发布时间为未来时间: {publish_time}")

            # 检查URL格式
            url = item.get("url")
            if url:
                if len(url) > 1000:
                    item_errors.append(f"第{i+1}条新闻URL过长: {len(url)}字符 > 1000字符")
                elif not url.startswith(("http://", "https://")):
                    item_warnings.append(f"第{i+1}条新闻URL格式可能无效: {url}")

            # 检查分类长度
            category = item.get("category")
            if category and len(category) > 50:
                item_errors.append(f"第{i+1}条新闻分类过长: {len(category)}字符 > 50字符")

            # 检查情感分数范围
            sentiment_score = item.get("sentiment_score")
            if sentiment_score is not None:
                try:
                    score = float(sentiment_score)
                    if not -1 <= score <= 1:
                        item_errors.append(f"第{i+1}条新闻情感分数超出范围: {score} (应在-1到1之间)")
                except (ValueError, TypeError):
                    item_errors.append(f"第{i+1}条新闻情感分数格式无效: {sentiment_score}")

            # 检查情感标签
            sentiment_label = item.get("sentiment_label")
            if sentiment_label:
                valid_labels = ["positive", "negative", "neutral"]
                if sentiment_label not in valid_labels:
                    item_warnings.append(f"第{i+1}条新闻情感标签可能无效: {sentiment_label}")
                if len(sentiment_label) > 20:
                    item_errors.append(f"第{i+1}条新闻情感标签过长: {len(sentiment_label)}字符 > 20字符")

            # 检查相关股票代码格式
            related_stocks = item.get("related_stocks")
            if related_stocks and isinstance(related_stocks, list):
                for stock_code in related_stocks:
                    if not self._is_valid_ts_code(stock_code):
                        item_warnings.append(f"第{i+1}条新闻相关股票代码格式可能无效: {stock_code}")

            # 统计错误和警告
            if item_errors:
                errors.extend(item_errors)
            else:
                valid_count += 1

            if item_warnings:
                warnings.extend(item_warnings)

        return {
            "is_valid": len(errors) == 0,
            "total_count": total_count,
            "valid_count": valid_count,
            "error_count": total_count - valid_count,
            "warnings": warnings,
            "errors": errors,
            "quality_score": valid_count / total_count if total_count > 0 else 0
        }

    def _parse_date(self, date_str: str) -> datetime | None:
        """解析日期字符串"""
        if not date_str:
            return None

        try:
            # 尝试多种日期格式
            for fmt in ["%Y%m%d", "%Y-%m-%d", "%Y/%m/%d"]:
                try:
                    return datetime.strptime(date_str, fmt)
                except ValueError:
                    continue
            return None
        except Exception:
            return None

    def get_data_quality_report(self, data_list: list[dict[str, Any]], data_type: str) -> dict[str, Any]:
        """生成数据质量报告

        Args:
            data_list: 数据列表
            data_type: 数据类型 (stock_basic, daily_data, financial_data, news_data)

        Returns:
            数据质量报告
        """
        if data_type == "news_data":
            validation_result = self.validate_news_data(data_list)
            return {
                "data_type": data_type,
                "total_records": validation_result.get("total_count", 0),
                "valid_records": validation_result.get("valid_count", 0),
                "invalid_records": validation_result.get("error_count", 0),
                "quality_score": validation_result.get("quality_score", 0),
                "validation_details": validation_result,
                "generated_at": datetime.now().isoformat()
            }

        if not data_list:
            return {'total_count': 0, 'valid_count': 0, 'invalid_count': 0, 'quality_rate': 0.0}

        total_count = len(data_list)
        valid_count = 0
        invalid_count = 0
        error_details = []

        for data in data_list:
            try:
                if data_type == 'stock_basic':
                    self.validate_stock_basic_info(data)
                elif data_type == 'daily_data':
                    self.validate_daily_data(data)
                elif data_type == 'financial_data':
                    self.validate_financial_data(data)
                valid_count += 1
            except ValidationError as e:
                invalid_count += 1
                error_details.append(str(e))

        quality_rate = valid_count / total_count if total_count > 0 else 0.0

        return {
            'total_count': total_count,
            'valid_count': valid_count,
            'invalid_count': invalid_count,
            'quality_rate': quality_rate,
            'error_details': error_details[:10]  # 只返回前10个错误详情
        }
