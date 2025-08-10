# 数据采集系统 API 文档

本文档描述了数据采集系统的 REST API 接口，采用 OpenAPI 3.0.3 规范。

## 文档概览

- **API 版本**: v1.0.0
- **基础路径**: `/api/v1`
- **文档格式**: OpenAPI 3.0.3 (YAML)
- **文档文件**: `openapi.yaml`

## 功能模块

### 1. 数据查询模块

#### 股票数据查询
- `GET /api/v1/data/stocks` - 获取股票列表
- `GET /api/v1/data/stocks/{symbol}` - 获取指定股票信息

#### 行情数据查询
- `GET /api/v1/data/market` - 获取行情数据
- `GET /api/v1/data/market/{symbol}/latest` - 获取最新行情数据

#### 财务数据查询
- `GET /api/v1/data/financial` - 获取财务数据
- `GET /api/v1/data/financial/{symbol}/latest` - 获取最新财务数据

#### 新闻数据查询
- `GET /api/v1/data/news` - 获取新闻列表
- `GET /api/v1/data/news/{id}` - 获取新闻详情
- `GET /api/v1/data/news/hot` - 获取热门新闻
- `GET /api/v1/data/news/latest` - 获取最新新闻

#### 宏观数据查询
- `GET /api/v1/data/macro` - 获取宏观经济数据

### 2. 任务管理模块

- `GET /api/v1/tasks` - 获取任务列表
- `POST /api/v1/tasks` - 创建任务
- `PUT /api/v1/tasks/{id}` - 更新任务
- `DELETE /api/v1/tasks/{id}` - 删除任务
- `POST /api/v1/tasks/{id}/run` - 执行任务
- `GET /api/v1/tasks/{id}/status` - 获取任务状态

### 3. 数据采集模块

#### 股票数据采集
- `POST /api/v1/collection/stock/basic` - 采集股票基础数据
- `POST /api/v1/collection/stock/daily` - 采集日线行情数据
- `POST /api/v1/collection/stock/history` - 采集股票历史数据
- `POST /api/v1/collection/stock/batch` - 批量采集股票数据
- `POST /api/v1/collection/stock/sync` - 同步股票列表
- `POST /api/v1/collection/stock/realtime` - 采集实时数据

#### 财务和宏观数据采集
- `POST /api/v1/collection/financial` - 采集财务数据
- `POST /api/v1/collection/macro` - 采集宏观数据

#### 情感数据采集
- `POST /api/v1/collection/sentiment/money-flow` - 采集资金流向数据
- `POST /api/v1/collection/sentiment/northbound-fund` - 采集北向资金数据
- `POST /api/v1/collection/sentiment/northbound-top-stocks` - 采集北向资金十大成交股
- `POST /api/v1/collection/sentiment/margin-trading` - 采集融资融券数据
- `POST /api/v1/collection/sentiment/etf-basic` - 采集ETF基础数据
- `POST /api/v1/collection/sentiment/all` - 采集所有情感数据
- `POST /api/v1/collection/sentiment/active-stocks-money-flow` - 采集活跃股票资金流向

#### 采集状态查询
- `GET /api/v1/collection/status` - 获取采集器状态
- `GET /api/v1/collection/sentiment/status` - 获取情感数据采集器状态

#### 综合采集
- `POST /api/v1/collection/today` - 采集今日数据

### 4. 系统监控模块

- `GET /health` - 健康检查
- `GET /api/v1/monitor/stats` - 获取系统统计信息
- `GET /api/v1/monitor/metrics` - 获取系统指标

## 数据模型

### 核心实体

1. **Stock** - 股票基础信息
2. **MarketData** - 行情数据
3. **FinancialData** - 财务数据
4. **NewsData** - 新闻数据
5. **MacroData** - 宏观数据
6. **DataTask** - 数据采集任务

### 响应格式

#### 统一响应结构
```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "timestamp": 1640995200,
  "request_id": "req-123456"
}
```

#### 分页响应结构
```json
{
  "code": 0,
  "message": "success",
  "data": [],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 5000,
    "pages": 250
  },
  "timestamp": 1640995200,
  "request_id": "req-123456"
}
```

## 使用指南

### 1. 查看 API 文档

#### 使用 Swagger UI
```bash
# 安装 swagger-ui-serve
npm install -g swagger-ui-serve

# 启动文档服务
swagger-ui-serve docs/api/openapi.yaml
```

#### 使用在线工具
- 访问 [Swagger Editor](https://editor.swagger.io/)
- 将 `openapi.yaml` 内容粘贴到编辑器中

### 2. API 调用示例

#### 获取股票列表
```bash
curl -X GET "http://localhost:8080/api/v1/data/stocks?page=1&page_size=20" \
  -H "Content-Type: application/json"
```

#### 获取指定股票信息
```bash
curl -X GET "http://localhost:8080/api/v1/data/stocks/000001" \
  -H "Content-Type: application/json"
```

#### 获取行情数据
```bash
curl -X GET "http://localhost:8080/api/v1/data/market?symbol=000001&start_date=2024-01-01&end_date=2024-12-31" \
  -H "Content-Type: application/json"
```

#### 创建采集任务
```bash
curl -X POST "http://localhost:8080/api/v1/tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "股票基础数据采集",
    "type": "stock_data",
    "description": "定时采集股票基础信息",
    "schedule": "0 0 9 * * ?",
    "config": {}
  }'
```

#### 启动股票基础数据采集
```bash
curl -X POST "http://localhost:8080/api/v1/collection/stock/basic" \
  -H "Content-Type: application/json"
```

### 3. 错误处理

系统使用标准的 HTTP 状态码和统一的错误响应格式：

```json
{
  "code": 400,
  "message": "请求参数错误",
  "details": "symbol 参数不能为空",
  "timestamp": 1640995200,
  "request_id": "req-123456"
}
```

常见错误码：
- `400` - 请求参数错误
- `404` - 资源不存在
- `500` - 服务器内部错误

### 4. 分页查询

所有列表查询接口都支持分页，使用以下参数：
- `page`: 页码，从 1 开始
- `page_size`: 每页数量，默认 20，最大 100

### 5. 日期格式

- 日期参数格式：`YYYY-MM-DD`（如：2024-01-01）
- 日期时间格式：`YYYY-MM-DDTHH:mm:ssZ`（如：2024-01-01T00:00:00Z）
- 交易日期格式：`YYYYMMDD`（如：20240101）

## 开发环境

### 本地服务器
- 地址：`http://localhost:8080`
- 健康检查：`http://localhost:8080/health`

### 生产环境
- 地址：`https://api.trading-system.com`
- 健康检查：`https://api.trading-system.com/health`

## 版本历史

### v1.0.0 (2024-01-01)
- 初始版本发布
- 支持股票、行情、财务、新闻、宏观数据查询
- 支持数据采集任务管理
- 支持系统监控功能

## 联系方式

- 开发团队：数据采集系统开发团队
- 邮箱：dev@trading-system.com
- 许可证：MIT License

## 注意事项

1. **认证**：当前版本暂未启用认证，所有接口均可直接访问
2. **限流**：建议合理控制请求频率，避免对系统造成压力
3. **数据更新**：数据采集有一定延迟，请根据业务需求选择合适的查询时间
4. **错误重试**：建议在客户端实现适当的错误重试机制
5. **数据格式**：所有数值类型的字段可能为 null，请在客户端做好空值处理