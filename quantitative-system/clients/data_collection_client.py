"""数据采集系统HTTP客户端"""

import asyncio
import logging
from typing import Any

import aiohttp
import requests

from utils.exceptions import ExternalServiceError, NotFoundError, TimeoutError


class DataCollectionClient:
    """数据采集系统HTTP客户端

    基于数据采集系统的OpenAPI规范实现的简化HTTP客户端，
    提供股票、行情、财务、新闻等数据的查询和任务管理功能。
    """

    def __init__(
        self, base_url: str = "http://localhost:8080", timeout: int = 30
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.logger = logging.getLogger(__name__)

    def _make_request(
        self,
        method: str,
        endpoint: str,
        params: dict[str, Any] | None = None,
        json_data: dict[str, Any] | None = None,
        **kwargs: Any,
    ) -> dict[str, Any]:
        """发起HTTP请求"""
        url = f"{self.base_url}{endpoint}"

        try:
            response = requests.request(
                method=method,
                url=url,
                params=params,
                json=json_data,
                timeout=self.timeout,
                **kwargs,
            )

            # 处理HTTP状态码
            if response.status_code == 404:
                raise NotFoundError(f"Resource not found: {url}")
            elif response.status_code >= 500:
                raise ExternalServiceError(f"Server error: {response.status_code}")
            elif response.status_code >= 400:
                error_data = (
                    response.json()
                    if response.headers.get("content-type", "").startswith(
                        "application/json"
                    )
                    else {}
                )
                error_msg = error_data.get("message", f"HTTP {response.status_code}")
                raise ExternalServiceError(f"Client error: {error_msg}")

            result = response.json()
            if not isinstance(result, dict):
                raise ExternalServiceError(
                    f"Unexpected response format: {type(result)}"
                )
            return result

        except requests.exceptions.Timeout as e:
            raise TimeoutError(f"Request timeout: {url}") from e
        except requests.exceptions.ConnectionError as e:
            raise ExternalServiceError(f"Connection error: {url}") from e
        except requests.exceptions.RequestException as e:
            raise ExternalServiceError(f"Request failed: {e!s}") from e

    # 股票数据查询
    def get_stocks(
        self,
        symbol: str | None = None,
        exchange: str | None = None,
        industry: str | None = None,
        status: str | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> dict[str, Any]:
        """获取股票列表"""
        params = {"page": str(page), "page_size": str(page_size)}
        if symbol:
            params["symbol"] = symbol
        if exchange:
            params["exchange"] = exchange
        if industry:
            params["industry"] = industry
        if status:
            params["status"] = status

        return self._make_request("GET", "/api/v1/data/stocks", params=params)

    def get_stock_by_symbol(self, symbol: str) -> dict[str, Any]:
        """获取指定股票信息"""
        return self._make_request("GET", f"/api/v1/data/stocks/{symbol}")

    # 行情数据查询
    def get_market_data(
        self,
        symbol: str,
        start_date: str | None = None,
        end_date: str | None = None,
        period: str = "1d",
        page: int = 1,
        page_size: int = 100,
    ) -> dict[str, Any]:
        """获取行情数据"""
        params = {
            "symbol": symbol,
            "period": period,
            "page": str(page),
            "page_size": str(page_size),
        }
        if start_date:
            params["start_date"] = start_date
        if end_date:
            params["end_date"] = end_date

        return self._make_request("GET", "/api/v1/data/market", params=params)

    def get_latest_market_data(self, symbol: str, period: str = "1d") -> dict[str, Any]:
        """获取最新行情数据"""
        params = {"period": period}
        return self._make_request(
            "GET", f"/api/v1/data/market/{symbol}/latest", params=params
        )

    # 财务数据查询
    def get_financial_data(
        self,
        symbol: str,
        start_date: str | None = None,
        end_date: str | None = None,
        report_type: str | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> dict[str, Any]:
        """获取财务数据"""
        params = {"symbol": symbol, "page": str(page), "page_size": str(page_size)}
        if start_date:
            params["start_date"] = start_date
        if end_date:
            params["end_date"] = end_date
        if report_type:
            params["report_type"] = report_type

        return self._make_request("GET", "/api/v1/data/financial", params=params)

    def get_latest_financial_data(
        self, symbol: str, report_type: str | None = None
    ) -> dict[str, Any]:
        """获取最新财务数据"""
        params = {}
        if report_type:
            params["report_type"] = report_type
        return self._make_request(
            "GET", f"/api/v1/data/financial/{symbol}/latest", params=params
        )

    # 新闻数据查询
    def get_news(
        self,
        keyword: str | None = None,
        category: str | None = None,
        related_stock: str | None = None,
        sentiment: int | None = None,
        importance: int | None = None,
        start_time: str | None = None,
        end_time: str | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> dict[str, Any]:
        """获取新闻列表"""
        params = {"page": str(page), "page_size": str(page_size)}
        if keyword:
            params["keyword"] = keyword
        if category:
            params["category"] = category
        if related_stock:
            params["related_stock"] = related_stock
        if sentiment is not None:
            params["sentiment"] = str(sentiment)
        if importance is not None:
            params["importance"] = str(importance)
        if start_time:
            params["start_time"] = start_time
        if end_time:
            params["end_time"] = end_time

        return self._make_request("GET", "/api/v1/data/news", params=params)

    def get_news_by_id(self, news_id: int) -> dict[str, Any]:
        """获取新闻详情"""
        return self._make_request("GET", f"/api/v1/data/news/{news_id}")

    def get_hot_news(self, limit: int = 10, hours: int = 24) -> dict[str, Any]:
        """获取热门新闻"""
        params = {"limit": str(limit), "hours": str(hours)}
        return self._make_request("GET", "/api/v1/data/news/hot", params=params)

    def get_latest_news(self, limit: int = 10) -> dict[str, Any]:
        """获取最新新闻"""
        params = {"limit": str(limit)}
        return self._make_request("GET", "/api/v1/data/news/latest", params=params)

    # 宏观数据查询
    def get_macro_data(
        self,
        indicator_code: str | None = None,
        period_type: str | None = None,
        start_date: str | None = None,
        end_date: str | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> dict[str, Any]:
        """获取宏观经济数据"""
        params = {"page": str(page), "page_size": str(page_size)}
        if indicator_code:
            params["indicator_code"] = indicator_code
        if period_type:
            params["period_type"] = period_type
        if start_date:
            params["start_date"] = start_date
        if end_date:
            params["end_date"] = end_date

        return self._make_request("GET", "/api/v1/data/macro", params=params)

    # 任务管理
    def get_tasks(
        self,
        task_type: str | None = None,
        status: int | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> dict[str, Any]:
        """获取任务列表"""
        params = {"page": str(page), "page_size": str(page_size)}
        if task_type:
            params["task_type"] = task_type
        if status is not None:
            params["status"] = str(status)

        return self._make_request("GET", "/api/v1/tasks", params=params)

    def create_task(
        self,
        name: str,
        task_type: str,
        schedule: str,
        description: str | None = None,
        config: dict | None = None,
    ) -> dict[str, Any]:
        """创建任务"""
        json_data: dict[str, Any] = {
            "name": name,
            "type": task_type,
            "schedule": schedule,
        }
        if description:
            json_data["description"] = description
        if config:
            json_data["config"] = config

        return self._make_request("POST", "/api/v1/tasks", json_data=json_data)

    def update_task(
        self,
        task_id: int,
        task_name: str | None = None,
        description: str | None = None,
        cron_expr: str | None = None,
        status: int | None = None,
        config: dict | None = None,
    ) -> dict[str, Any]:
        """更新任务"""
        json_data: dict[str, Any] = {}
        if task_name:
            json_data["task_name"] = task_name
        if description:
            json_data["description"] = description
        if cron_expr:
            json_data["cron_expr"] = cron_expr
        if status is not None:
            json_data["status"] = status
        if config:
            json_data["config"] = config

        return self._make_request(
            "PUT", f"/api/v1/tasks/{task_id}", json_data=json_data
        )

    def delete_task(self, task_id: int) -> dict[str, Any]:
        """删除任务"""
        return self._make_request("DELETE", f"/api/v1/tasks/{task_id}")

    def run_task(self, task_id: int) -> dict[str, Any]:
        """执行任务"""
        return self._make_request("POST", f"/api/v1/tasks/{task_id}/run")

    def get_task_status(self, task_id: int) -> dict[str, Any]:
        """获取任务状态"""
        return self._make_request("GET", f"/api/v1/tasks/{task_id}/status")

    # 数据采集控制
    def collect_stock_basic_data(self) -> dict[str, Any]:
        """采集股票基础数据"""
        return self._make_request("POST", "/api/v1/collection/stock/basic")

    def collect_daily_market_data(
        self, trade_date: str | None = None
    ) -> dict[str, Any]:
        """采集日线行情数据"""
        params = {}
        if trade_date:
            params["trade_date"] = trade_date
        return self._make_request(
            "POST", "/api/v1/collection/stock/daily", params=params
        )

    def collect_today_data(self) -> dict[str, Any]:
        """采集今日数据"""
        return self._make_request("POST", "/api/v1/collection/today")

    def get_collection_status(self) -> dict[str, Any]:
        """获取采集器状态"""
        return self._make_request("GET", "/api/v1/collection/status")

    def get_sentiment_collection_status(self) -> dict[str, Any]:
        """获取情感数据采集器状态"""
        return self._make_request("GET", "/api/v1/collection/sentiment/status")

    # 系统监控
    def health_check(self) -> dict[str, Any]:
        """健康检查"""
        return self._make_request("GET", "/health")

    def get_system_stats(self) -> dict[str, Any]:
        """获取系统统计信息"""
        return self._make_request("GET", "/api/v1/monitor/stats")

    def get_system_metrics(self) -> dict[str, Any]:
        """获取系统指标"""
        return self._make_request("GET", "/api/v1/monitor/metrics")


class AsyncDataCollectionClient:
    """数据采集系统异步HTTP客户端

    提供与同步客户端相同的功能，但使用异步HTTP请求。
    """

    def __init__(self, base_url: str = "http://localhost:8080", timeout: int = 30):
        self.base_url = base_url.rstrip("/")
        self.timeout = aiohttp.ClientTimeout(total=timeout)
        self.logger = logging.getLogger(__name__)

    async def _make_request(
        self,
        method: str,
        endpoint: str,
        params: dict[str, Any] | None = None,
        json_data: dict[str, Any] | None = None,
        **kwargs: Any,
    ) -> dict[str, Any]:
        """发起异步HTTP请求"""
        url = f"{self.base_url}{endpoint}"

        try:
            async with (
                aiohttp.ClientSession(timeout=self.timeout) as session,
                session.request(
                    method=method, url=url, params=params, json=json_data, **kwargs
                ) as response,
            ):
                # 处理HTTP状态码
                if response.status == 404:
                    raise NotFoundError(f"Resource not found: {url}")
                elif response.status >= 500:
                    raise ExternalServiceError(f"Server error: {response.status}")
                elif response.status >= 400:
                    try:
                        error_data = await response.json()
                        error_msg = error_data.get("message", f"HTTP {response.status}")
                    except Exception:
                        error_msg = f"HTTP {response.status}"
                    raise ExternalServiceError(f"Client error: {error_msg}")

                result = await response.json()
                if not isinstance(result, dict):
                    raise ExternalServiceError(
                        f"Expected dict response, got {type(result)}"
                    )
                return result

        except asyncio.TimeoutError as e:
            raise TimeoutError(f"Request timeout: {url}") from e
        except aiohttp.ClientError as e:
            raise ExternalServiceError(f"Request failed: {e!s}") from e

    # 为了简洁,这里只实现几个关键方法,其他方法可以按需添加
    async def get_stocks(self, **kwargs: Any) -> dict[str, Any]:
        """获取股票列表"""
        return await self._make_request("GET", "/api/v1/data/stocks", params=kwargs)

    async def get_market_data(self, symbol: str, **kwargs: Any) -> dict[str, Any]:
        """获取行情数据"""
        params = {"symbol": symbol, **kwargs}
        return await self._make_request("GET", "/api/v1/data/market", params=params)

    async def get_news(self, **kwargs: Any) -> dict[str, Any]:
        """获取新闻列表"""
        return await self._make_request("GET", "/api/v1/data/news", params=kwargs)

    async def health_check(self) -> dict[str, Any]:
        """健康检查"""
        return await self._make_request("GET", "/health")

    async def close(self) -> None:
        """关闭客户端"""
        # aiohttp会自动管理连接,这里保留接口以备将来使用
        pass
