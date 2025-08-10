"""数据采集系统HTTP客户端"""

import asyncio
import logging
from typing import Dict, List, Optional, Any
from datetime import datetime

import aiohttp
import requests

from utils.exceptions import ExternalServiceError, TimeoutError, NotFoundError





class DataCollectionClient:
    """数据采集系统HTTP客户端
    
    基于数据采集系统的OpenAPI规范实现的简化HTTP客户端，
    提供股票、行情、财务、新闻等数据的查询和任务管理功能。
    """
    
    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        timeout: int = 30
    ):
        self.base_url = base_url.rstrip('/')
        self.timeout = timeout
        self.logger = logging.getLogger(__name__)
    

    
    def _make_request(
        self,
        method: str,
        endpoint: str,
        params: Optional[Dict] = None,
        json_data: Optional[Dict] = None,
        **kwargs
    ) -> Dict[str, Any]:
        """发起HTTP请求"""
        url = f"{self.base_url}{endpoint}"
        
        try:
            response = requests.request(
                method=method,
                url=url,
                params=params,
                json=json_data,
                timeout=self.timeout,
                **kwargs
            )
            
            # 处理HTTP状态码
            if response.status_code == 404:
                raise NotFoundError(f"Resource not found: {url}")
            elif response.status_code >= 500:
                raise ExternalServiceError(f"Server error: {response.status_code}")
            elif response.status_code >= 400:
                error_data = response.json() if response.headers.get('content-type', '').startswith('application/json') else {}
                error_msg = error_data.get('message', f"HTTP {response.status_code}")
                raise ExternalServiceError(f"Client error: {error_msg}")
                
            return response.json()
            
        except requests.exceptions.Timeout:
            raise TimeoutError(f"Request timeout: {url}")
        except requests.exceptions.ConnectionError:
            raise ExternalServiceError(f"Connection error: {url}")
        except requests.exceptions.RequestException as e:
            raise ExternalServiceError(f"Request failed: {str(e)}")
    
    # 股票数据查询
    def get_stocks(
        self,
        symbol: Optional[str] = None,
        exchange: Optional[str] = None,
        industry: Optional[str] = None,
        status: Optional[str] = None,
        page: int = 1,
        page_size: int = 20
    ) -> Dict[str, Any]:
        """获取股票列表"""
        params = {
            "page": page,
            "page_size": page_size
        }
        if symbol:
            params["symbol"] = symbol
        if exchange:
            params["exchange"] = exchange
        if industry:
            params["industry"] = industry
        if status:
            params["status"] = status
            
        return self._make_request("GET", "/api/v1/data/stocks", params=params)
        
    def get_stock_by_symbol(self, symbol: str) -> Dict[str, Any]:
        """获取指定股票信息"""
        return self._make_request("GET", f"/api/v1/data/stocks/{symbol}")
    
    # 行情数据查询
    def get_market_data(
        self,
        symbol: str,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None,
        period: str = "1d",
        page: int = 1,
        page_size: int = 100
    ) -> Dict[str, Any]:
        """获取行情数据"""
        params = {
            "symbol": symbol,
            "period": period,
            "page": page,
            "page_size": page_size
        }
        if start_date:
            params["start_date"] = start_date
        if end_date:
            params["end_date"] = end_date
            
        return self._make_request("GET", "/api/v1/data/market", params=params)
        
    def get_latest_market_data(self, symbol: str, period: str = "1d") -> Dict[str, Any]:
        """获取最新行情数据"""
        params = {"period": period}
        return self._make_request("GET", f"/api/v1/data/market/{symbol}/latest", params=params)
    
    # 财务数据查询
    def get_financial_data(
        self,
        symbol: str,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None,
        report_type: Optional[str] = None,
        page: int = 1,
        page_size: int = 20
    ) -> Dict[str, Any]:
        """获取财务数据"""
        params = {
            "symbol": symbol,
            "page": page,
            "page_size": page_size
        }
        if start_date:
            params["start_date"] = start_date
        if end_date:
            params["end_date"] = end_date
        if report_type:
            params["report_type"] = report_type
            
        return self._make_request("GET", "/api/v1/data/financial", params=params)
        
    def get_latest_financial_data(self, symbol: str, report_type: Optional[str] = None) -> Dict[str, Any]:
        """获取最新财务数据"""
        params = {}
        if report_type:
            params["report_type"] = report_type
        return self._make_request("GET", f"/api/v1/data/financial/{symbol}/latest", params=params)
        
    # 新闻数据查询
    def get_news(
        self,
        keyword: Optional[str] = None,
        category: Optional[str] = None,
        related_stock: Optional[str] = None,
        sentiment: Optional[int] = None,
        importance: Optional[int] = None,
        start_time: Optional[str] = None,
        end_time: Optional[str] = None,
        page: int = 1,
        page_size: int = 20
    ) -> Dict[str, Any]:
        """获取新闻列表"""
        params = {
            "page": page,
            "page_size": page_size
        }
        if keyword:
            params["keyword"] = keyword
        if category:
            params["category"] = category
        if related_stock:
            params["related_stock"] = related_stock
        if sentiment is not None:
            params["sentiment"] = sentiment
        if importance is not None:
            params["importance"] = importance
        if start_time:
            params["start_time"] = start_time
        if end_time:
            params["end_time"] = end_time
            
        return self._make_request("GET", "/api/v1/data/news", params=params)
        
    def get_news_by_id(self, news_id: int) -> Dict[str, Any]:
        """获取新闻详情"""
        return self._make_request("GET", f"/api/v1/data/news/{news_id}")
        
    def get_hot_news(self, limit: int = 10, hours: int = 24) -> Dict[str, Any]:
        """获取热门新闻"""
        params = {
            "limit": limit,
            "hours": hours
        }
        return self._make_request("GET", "/api/v1/data/news/hot", params=params)
        
    def get_latest_news(self, limit: int = 10) -> Dict[str, Any]:
        """获取最新新闻"""
        params = {"limit": limit}
        return self._make_request("GET", "/api/v1/data/news/latest", params=params)
    
    # 宏观数据查询
    def get_macro_data(
        self,
        indicator_code: Optional[str] = None,
        period_type: Optional[str] = None,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None,
        page: int = 1,
        page_size: int = 20
    ) -> Dict[str, Any]:
        """获取宏观经济数据"""
        params = {
            "page": page,
            "page_size": page_size
        }
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
        task_type: Optional[str] = None,
        status: Optional[int] = None,
        page: int = 1,
        page_size: int = 20
    ) -> Dict[str, Any]:
        """获取任务列表"""
        params = {
            "page": page,
            "page_size": page_size
        }
        if task_type:
            params["task_type"] = task_type
        if status is not None:
            params["status"] = status
            
        return self._make_request("GET", "/api/v1/tasks", params=params)
        
    def create_task(
        self,
        name: str,
        task_type: str,
        schedule: str,
        description: Optional[str] = None,
        config: Optional[Dict] = None
    ) -> Dict[str, Any]:
        """创建任务"""
        json_data = {
            "name": name,
            "type": task_type,
            "schedule": schedule
        }
        if description:
            json_data["description"] = description
        if config:
            json_data["config"] = config
            
        return self._make_request("POST", "/api/v1/tasks", json_data=json_data)
        
    def update_task(
        self,
        task_id: int,
        task_name: Optional[str] = None,
        description: Optional[str] = None,
        cron_expr: Optional[str] = None,
        status: Optional[int] = None,
        config: Optional[Dict] = None
    ) -> Dict[str, Any]:
        """更新任务"""
        json_data = {}
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
            
        return self._make_request("PUT", f"/api/v1/tasks/{task_id}", json_data=json_data)
        
    def delete_task(self, task_id: int) -> Dict[str, Any]:
        """删除任务"""
        return self._make_request("DELETE", f"/api/v1/tasks/{task_id}")
        
    def run_task(self, task_id: int) -> Dict[str, Any]:
        """执行任务"""
        return self._make_request("POST", f"/api/v1/tasks/{task_id}/run")
        
    def get_task_status(self, task_id: int) -> Dict[str, Any]:
        """获取任务状态"""
        return self._make_request("GET", f"/api/v1/tasks/{task_id}/status")
    
    # 数据采集控制
    def collect_stock_basic_data(self) -> Dict[str, Any]:
        """采集股票基础数据"""
        return self._make_request("POST", "/api/v1/collection/stock/basic")
        
    def collect_daily_market_data(self, trade_date: Optional[str] = None) -> Dict[str, Any]:
        """采集日线行情数据"""
        params = {}
        if trade_date:
            params["trade_date"] = trade_date
        return self._make_request("POST", "/api/v1/collection/stock/daily", params=params)
        
    def collect_today_data(self) -> Dict[str, Any]:
        """采集今日数据"""
        return self._make_request("POST", "/api/v1/collection/today")
        
    def get_collection_status(self) -> Dict[str, Any]:
        """获取采集器状态"""
        return self._make_request("GET", "/api/v1/collection/status")
        
    def get_sentiment_collection_status(self) -> Dict[str, Any]:
        """获取情感数据采集器状态"""
        return self._make_request("GET", "/api/v1/collection/sentiment/status")
        
    # 系统监控
    def health_check(self) -> Dict[str, Any]:
        """健康检查"""
        return self._make_request("GET", "/health")
        
    def get_system_stats(self) -> Dict[str, Any]:
        """获取系统统计信息"""
        return self._make_request("GET", "/api/v1/monitor/stats")
        
    def get_system_metrics(self) -> Dict[str, Any]:
        """获取系统指标"""
        return self._make_request("GET", "/api/v1/monitor/metrics")


class AsyncDataCollectionClient:
    """数据采集系统异步HTTP客户端
    
    提供与同步客户端相同的功能，但使用异步HTTP请求。
    """
    
    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        timeout: int = 30
    ):
        self.base_url = base_url.rstrip('/')
        self.timeout = aiohttp.ClientTimeout(total=timeout)
        self.logger = logging.getLogger(__name__)
        
    async def _make_request(
        self,
        method: str,
        endpoint: str,
        params: Optional[Dict] = None,
        json_data: Optional[Dict] = None,
        **kwargs
    ) -> Dict[str, Any]:
        """发起异步HTTP请求"""
        url = f"{self.base_url}{endpoint}"
        
        try:
            async with aiohttp.ClientSession(timeout=self.timeout) as session:
                async with session.request(
                    method=method,
                    url=url,
                    params=params,
                    json=json_data,
                    **kwargs
                ) as response:
                    # 处理HTTP状态码
                    if response.status == 404:
                        raise NotFoundError(f"Resource not found: {url}")
                    elif response.status >= 500:
                        raise ExternalServiceError(f"Server error: {response.status}")
                    elif response.status >= 400:
                        try:
                            error_data = await response.json()
                            error_msg = error_data.get('message', f"HTTP {response.status}")
                        except:
                            error_msg = f"HTTP {response.status}"
                        raise ExternalServiceError(f"Client error: {error_msg}")
                        
                    return await response.json()
                    
        except asyncio.TimeoutError:
            raise TimeoutError(f"Request timeout: {url}")
        except aiohttp.ClientError as e:
            raise ExternalServiceError(f"Request failed: {str(e)}")
            
    # 为了简洁，这里只实现几个关键方法，其他方法可以按需添加
    async def get_stocks(self, **kwargs) -> Dict[str, Any]:
        """获取股票列表"""
        return await self._make_request("GET", "/api/v1/data/stocks", params=kwargs)
        
    async def get_market_data(self, symbol: str, **kwargs) -> Dict[str, Any]:
        """获取行情数据"""
        params = {"symbol": symbol, **kwargs}
        return await self._make_request("GET", "/api/v1/data/market", params=params)
        
    async def get_news(self, **kwargs) -> Dict[str, Any]:
        """获取新闻列表"""
        return await self._make_request("GET", "/api/v1/data/news", params=kwargs)
        
    async def health_check(self) -> Dict[str, Any]:
        """健康检查"""
        return await self._make_request("GET", "/health")
        
    async def close(self):
        """关闭客户端"""
        # aiohttp会自动管理连接，这里保留接口以备将来使用
        pass