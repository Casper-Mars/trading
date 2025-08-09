package tushare

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"data-collection-system/model"
)

// MarketCollector 市场数据采集器
type MarketCollector struct {
	client *Client
}

// NewMarketCollector 创建市场数据采集器
func NewMarketCollector(client *Client) *MarketCollector {
	return &MarketCollector{
		client: client,
	}
}

// GetDailyData 获取日线行情数据
func (m *MarketCollector) GetDailyData(ctx context.Context, symbol, startDate, endDate string) ([]*model.MarketData, error) {
	params := map[string]interface{}{
		"ts_code":    symbol,    // 股票代码
		"start_date": startDate, // 开始日期 YYYYMMDD
		"end_date":   endDate,   // 结束日期 YYYYMMDD
	}

	fields := "ts_code,trade_date,open,high,low,close,pre_close,change,pct_chg,vol,amount"

	data, err := m.client.CallWithRetry(ctx, "daily", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get daily data failed: %w", err)
	}

	return m.parseDailyData(data)
}

// GetDailyDataByDate 获取指定日期所有股票的日线数据
func (m *MarketCollector) GetDailyDataByDate(ctx context.Context, tradeDate string) ([]*model.MarketData, error) {
	params := map[string]interface{}{
		"trade_date": tradeDate, // 交易日期 YYYYMMDD
	}

	fields := "ts_code,trade_date,open,high,low,close,pre_close,change,pct_chg,vol,amount"

	data, err := m.client.CallWithRetry(ctx, "daily", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get daily data by date failed: %w", err)
	}

	return m.parseDailyData(data)
}

// parseDailyData 解析日线数据
func (m *MarketCollector) parseDailyData(data *ResponseData) ([]*model.MarketData, error) {
	if data == nil || len(data.Items) == 0 {
		return []*model.MarketData{}, nil
	}

	// 创建字段索引映射
	fieldMap := make(map[string]int)
	for i, field := range data.Fields {
		fieldMap[field] = i
	}

	marketData := make([]*model.MarketData, 0, len(data.Items))
	for _, item := range data.Items {
		md := &model.MarketData{}

		// 解析股票代码
		if idx, ok := fieldMap["ts_code"]; ok && idx < len(item) {
			if symbol, ok := item[idx].(string); ok {
				md.Symbol = symbol
			}
		}

		// 解析交易日期
		if idx, ok := fieldMap["trade_date"]; ok && idx < len(item) {
			if dateStr, ok := item[idx].(string); ok {
				if tradeDate, err := time.Parse("20060102", dateStr); err == nil {
					md.TradeDate = tradeDate
				}
			}
		}

		// 解析价格数据
		md.OpenPrice = m.parseFloat(item, fieldMap, "open")
		md.HighPrice = m.parseFloat(item, fieldMap, "high")
		md.LowPrice = m.parseFloat(item, fieldMap, "low")
		md.ClosePrice = m.parseFloat(item, fieldMap, "close")

		// 解析成交量和成交额
		md.Volume = m.parseInt64(item, fieldMap, "vol")
		md.Amount = m.parseFloat(item, fieldMap, "amount")

		// 设置周期
		md.Period = "1d"

		// 设置时间戳
		md.TradeTime = md.TradeDate

		marketData = append(marketData, md)
	}

	return marketData, nil
}

// GetRealtimeData 获取实时行情数据
func (m *MarketCollector) GetRealtimeData(ctx context.Context, symbols []string) ([]*model.MarketData, error) {
	if len(symbols) == 0 {
		return []*model.MarketData{}, nil
	}

	// Tushare实时数据接口
	params := map[string]interface{}{
		"ts_code": symbols[0], // 单个股票查询
	}

	fields := "ts_code,name,price,change,pct_chg,volume,amount,bid,ask,high,low,open,pre_close"

	data, err := m.client.CallWithRetry(ctx, "query", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get realtime data failed: %w", err)
	}

	return m.parseRealtimeData(data)
}

// parseRealtimeData 解析实时数据
func (m *MarketCollector) parseRealtimeData(data *ResponseData) ([]*model.MarketData, error) {
	if data == nil || len(data.Items) == 0 {
		return []*model.MarketData{}, nil
	}

	// 创建字段索引映射
	fieldMap := make(map[string]int)
	for i, field := range data.Fields {
		fieldMap[field] = i
	}

	marketData := make([]*model.MarketData, 0, len(data.Items))
	for _, item := range data.Items {
		md := &model.MarketData{}

		// 解析股票代码
		if idx, ok := fieldMap["ts_code"]; ok && idx < len(item) {
			if symbol, ok := item[idx].(string); ok {
				md.Symbol = symbol
			}
		}

		// 实时数据使用当前日期
		md.TradeDate = time.Now().Truncate(24 * time.Hour)

		// 解析价格数据
		md.OpenPrice = m.parseFloat(item, fieldMap, "open")
		md.HighPrice = m.parseFloat(item, fieldMap, "high")
		md.LowPrice = m.parseFloat(item, fieldMap, "low")
		md.ClosePrice = m.parseFloat(item, fieldMap, "price") // 实时价格作为收盘价

		// 解析成交量和成交额
		md.Volume = m.parseInt64(item, fieldMap, "volume")
		md.Amount = m.parseFloat(item, fieldMap, "amount")

		// 设置周期
		md.Period = "realtime"

		// 设置时间戳
		md.TradeTime = time.Now()

		marketData = append(marketData, md)
	}

	return marketData, nil
}

// GetMinuteData 获取分钟级行情数据
func (m *MarketCollector) GetMinuteData(ctx context.Context, symbol, freq, startDate, endDate string) ([]*model.MarketData, error) {
	params := map[string]interface{}{
		"ts_code":    symbol,    // 股票代码
		"freq":       freq,      // 分钟频率: 1min, 5min, 15min, 30min, 60min
		"start_date": startDate, // 开始日期 YYYYMMDD
		"end_date":   endDate,   // 结束日期 YYYYMMDD
	}

	fields := "ts_code,trade_time,open,high,low,close,vol,amount"

	data, err := m.client.CallWithRetry(ctx, "stk_mins", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get minute data failed: %w", err)
	}

	return m.parseMinuteData(data, freq)
}

// parseMinuteData 解析分钟数据
func (m *MarketCollector) parseMinuteData(data *ResponseData, freq string) ([]*model.MarketData, error) {
	if data == nil || len(data.Items) == 0 {
		return []*model.MarketData{}, nil
	}

	// 创建字段索引映射
	fieldMap := make(map[string]int)
	for i, field := range data.Fields {
		fieldMap[field] = i
	}

	marketData := make([]*model.MarketData, 0, len(data.Items))
	for _, item := range data.Items {
		md := &model.MarketData{}

		// 解析股票代码
		if idx, ok := fieldMap["ts_code"]; ok && idx < len(item) {
			if symbol, ok := item[idx].(string); ok {
				md.Symbol = symbol
			}
		}

		// 解析交易时间
		if idx, ok := fieldMap["trade_time"]; ok && idx < len(item) {
			if timeStr, ok := item[idx].(string); ok {
				// 格式: YYYY-MM-DD HH:MM:SS
				if tradeTime, err := time.Parse("2006-01-02 15:04:05", timeStr); err == nil {
					md.TradeDate = tradeTime.Truncate(24 * time.Hour)
					md.TradeTime = tradeTime
				}
			}
		}

		// 解析价格数据
		md.OpenPrice = m.parseFloat(item, fieldMap, "open")
		md.HighPrice = m.parseFloat(item, fieldMap, "high")
		md.LowPrice = m.parseFloat(item, fieldMap, "low")
		md.ClosePrice = m.parseFloat(item, fieldMap, "close")

		// 解析成交量和成交额
		md.Volume = m.parseInt64(item, fieldMap, "vol")
		md.Amount = m.parseFloat(item, fieldMap, "amount")

		// 设置周期
		md.Period = freq

		// 时间戳已在上面设置

		marketData = append(marketData, md)
	}

	return marketData, nil
}

// parseFloat 解析浮点数
func (m *MarketCollector) parseFloat(item []interface{}, fieldMap map[string]int, field string) float64 {
	if idx, ok := fieldMap[field]; ok && idx < len(item) {
		switch v := item[idx].(type) {
		case float64:
			return v
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0
}

// parseInt64 解析整数
func (m *MarketCollector) parseInt64(item []interface{}, fieldMap map[string]int, field string) int64 {
	if idx, ok := fieldMap[field]; ok && idx < len(item) {
		switch v := item[idx].(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

// GetIndexData 获取指数数据
func (m *MarketCollector) GetIndexData(ctx context.Context, indexCode, startDate, endDate string) ([]*model.MarketData, error) {
	params := map[string]interface{}{
		"ts_code":    indexCode,  // 指数代码
		"start_date": startDate,  // 开始日期 YYYYMMDD
		"end_date":   endDate,    // 结束日期 YYYYMMDD
	}

	fields := "ts_code,trade_date,close,open,high,low,pre_close,change,pct_chg,vol,amount"

	data, err := m.client.CallWithRetry(ctx, "index_daily", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get index data failed: %w", err)
	}

	return m.parseDailyData(data)
}

// GetMoneyFlow 获取资金流向数据
func (m *MarketCollector) GetMoneyFlow(ctx context.Context, symbol, startDate, endDate string) ([]map[string]interface{}, error) {
	params := map[string]interface{}{
		"ts_code":    symbol,
		"start_date": startDate,
		"end_date":   endDate,
	}

	fields := "ts_code,trade_date,buy_sm_vol,buy_sm_amount,sell_sm_vol,sell_sm_amount,buy_md_vol,buy_md_amount,sell_md_vol,sell_md_amount,buy_lg_vol,buy_lg_amount,sell_lg_vol,sell_lg_amount,buy_elg_vol,buy_elg_amount,sell_elg_vol,sell_elg_amount,net_mf_vol,net_mf_amount"

	data, err := m.client.CallWithRetry(ctx, "moneyflow", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get money flow failed: %w", err)
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