package tushare

import (
	"context"
	"fmt"
	"strings"
	"time"

	"data-collection-system/model"
)

// StockCollector 股票数据采集器
type StockCollector struct {
	client *Client
}

// NewStockCollector 创建股票数据采集器
func NewStockCollector(client *Client) *StockCollector {
	return &StockCollector{
		client: client,
	}
}

// GetStockBasic 获取股票基础信息
func (s *StockCollector) GetStockBasic(ctx context.Context, listStatus string) ([]*model.Stock, error) {
	params := map[string]interface{}{
		"list_status": listStatus, // L-上市, D-退市, P-暂停上市
	}

	fields := "ts_code,symbol,name,area,industry,market,list_date,delist_date,is_hs"

	data, err := s.client.CallWithRetry(ctx, "stock_basic", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get stock basic failed: %w", err)
	}

	return s.parseStockBasic(data)
}

// GetStockBySymbol 根据股票代码获取股票信息
func (s *StockCollector) GetStockBySymbol(ctx context.Context, symbol string) (*model.Stock, error) {
	params := map[string]interface{}{
		"ts_code": symbol,
	}

	fields := "ts_code,symbol,name,area,industry,market,list_date,delist_date,is_hs"

	data, err := s.client.CallWithRetry(ctx, "stock_basic", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get stock by symbol failed: %w", err)
	}

	stocks, err := s.parseStockBasic(data)
	if err != nil {
		return nil, err
	}

	if len(stocks) == 0 {
		return nil, fmt.Errorf("stock not found: %s", symbol)
	}

	return stocks[0], nil
}

// parseStockBasic 解析股票基础信息
func (s *StockCollector) parseStockBasic(data *ResponseData) ([]*model.Stock, error) {
	if data == nil || len(data.Items) == 0 {
		return []*model.Stock{}, nil
	}

	// 创建字段索引映射
	fieldMap := make(map[string]int)
	for i, field := range data.Fields {
		fieldMap[field] = i
	}

	stocks := make([]*model.Stock, 0, len(data.Items))
	for _, item := range data.Items {
		stock := &model.Stock{}

		// 解析ts_code (如: 000001.SZ)
		if idx, ok := fieldMap["ts_code"]; ok && idx < len(item) {
			if tsCode, ok := item[idx].(string); ok {
				stock.Symbol = tsCode
				// 解析交易所
				parts := strings.Split(tsCode, ".")
				if len(parts) == 2 {
					stock.Exchange = parts[1]
				}
			}
		}

		// 解析股票名称
		if idx, ok := fieldMap["name"]; ok && idx < len(item) {
			if name, ok := item[idx].(string); ok {
				stock.Name = name
			}
		}

		// 解析行业
		if idx, ok := fieldMap["industry"]; ok && idx < len(item) {
			if industry, ok := item[idx].(string); ok {
				stock.Industry = industry
			}
		}

		// 解析地区作为板块
		if idx, ok := fieldMap["area"]; ok && idx < len(item) {
			if area, ok := item[idx].(string); ok {
				stock.Sector = area
			}
		}

		// 解析上市日期
		if idx, ok := fieldMap["list_date"]; ok && idx < len(item) {
			if listDateStr, ok := item[idx].(string); ok && listDateStr != "" {
				if listDate, err := time.Parse("20060102", listDateStr); err == nil {
					stock.ListDate = &listDate
				}
			}
		}

		// 设置状态（默认为正常交易）
		stock.Status = model.StockStatusActive

		// 设置时间戳
		now := time.Now()
		stock.CreatedAt = now
		stock.UpdatedAt = now

		stocks = append(stocks, stock)
	}

	return stocks, nil
}

// GetStockCompany 获取上市公司基本信息
func (s *StockCollector) GetStockCompany(ctx context.Context, symbol string) (map[string]interface{}, error) {
	params := map[string]interface{}{
		"ts_code": symbol,
	}

	fields := "ts_code,chairman,manager,secretary,reg_capital,setup_date,province,city,introduction,website,email,office,employees,main_business,business_scope"

	data, err := s.client.CallWithRetry(ctx, "stock_company", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get stock company failed: %w", err)
	}

	if data == nil || len(data.Items) == 0 {
		return nil, fmt.Errorf("company info not found: %s", symbol)
	}

	// 创建字段索引映射
	fieldMap := make(map[string]int)
	for i, field := range data.Fields {
		fieldMap[field] = i
	}

	result := make(map[string]interface{})
	item := data.Items[0]
	for field, idx := range fieldMap {
		if idx < len(item) {
			result[field] = item[idx]
		}
	}

	return result, nil
}

// GetTradeCal 获取交易日历
func (s *StockCollector) GetTradeCal(ctx context.Context, exchange, startDate, endDate string) ([]map[string]interface{}, error) {
	params := map[string]interface{}{
		"exchange":    exchange,   // SSE-上交所, SZSE-深交所
		"start_date": startDate,  // 格式: 20220101
		"end_date":   endDate,    // 格式: 20221231
	}

	fields := "exchange,cal_date,is_open,pretrade_date"

	data, err := s.client.CallWithRetry(ctx, "trade_cal", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get trade calendar failed: %w", err)
	}

	if data == nil || len(data.Items) == 0 {
		return []map[string]interface{}{}, nil
	}

	// 创建字段索引映射
	fieldMap := make(map[string]int)
	for i, field := range data.Fields {
		fieldMap[field] = i
	}

	results := make([]map[string]interface{}, 0, len(data.Items))
	for _, item := range data.Items {
		result := make(map[string]interface{})
		for field, idx := range fieldMap {
			if idx < len(item) {
				result[field] = item[idx]
			}
		}
		results = append(results, result)
	}

	return results, nil
}

// IsTradeDay 检查指定日期是否为交易日
func (s *StockCollector) IsTradeDay(ctx context.Context, date string) (bool, error) {
	tradeCal, err := s.GetTradeCal(ctx, "SSE", date, date)
	if err != nil {
		return false, err
	}

	if len(tradeCal) == 0 {
		return false, nil
	}

	isOpen, ok := tradeCal[0]["is_open"]
	if !ok {
		return false, nil
	}

	// Tushare返回的is_open: 1-交易日, 0-非交易日
	if openStr, ok := isOpen.(string); ok {
		return openStr == "1", nil
	}
	if openInt, ok := isOpen.(float64); ok {
		return openInt == 1, nil
	}

	return false, nil
}

// GetStockList 获取股票列表（分页）
func (s *StockCollector) GetStockList(ctx context.Context, listStatus string, offset, limit int) ([]*model.Stock, error) {
	params := map[string]interface{}{
		"list_status": listStatus,
		"offset":      offset,
		"limit":       limit,
	}

	fields := "ts_code,symbol,name,area,industry,market,list_date,delist_date,is_hs"

	data, err := s.client.CallWithRetry(ctx, "stock_basic", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get stock list failed: %w", err)
	}

	return s.parseStockBasic(data)
}