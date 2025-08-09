package tushare

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"data-collection-system/model"
)

// MacroCollector 宏观数据采集器
type MacroCollector struct {
	client *Client
}

// NewMacroCollector 创建宏观数据采集器
func NewMacroCollector(client *Client) *MacroCollector {
	return &MacroCollector{
		client: client,
	}
}

// GetGDPData 获取GDP数据
func (m *MacroCollector) GetGDPData(ctx context.Context, startDate, endDate string) ([]*model.MacroData, error) {
	params := map[string]interface{}{
		"start_date": startDate, // 格式: YYYYMMDD
		"end_date":   endDate,   // 格式: YYYYMMDD
	}

	fields := "quarter,gdp,gdp_yoy,pi,pi_yoy,si,si_yoy,ti,ti_yoy"

	data, err := m.client.CallWithRetry(ctx, "cn_gdp", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get GDP data failed: %w", err)
	}

	return m.parseMacroData(data, "GDP")
}

// GetCPIData 获取CPI数据
func (m *MacroCollector) GetCPIData(ctx context.Context, startDate, endDate string) ([]*model.MacroData, error) {
	params := map[string]interface{}{
		"start_date": startDate,
		"end_date":   endDate,
	}

	fields := "month,nt_val,nt_yoy,nt_mom,nt_accu,town_val,town_yoy,town_mom,town_accu,cnt_val,cnt_yoy,cnt_mom,cnt_accu"

	data, err := m.client.CallWithRetry(ctx, "cn_cpi", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get CPI data failed: %w", err)
	}

	return m.parseMacroData(data, "CPI")
}

// GetPPIData 获取PPI数据
func (m *MacroCollector) GetPPIData(ctx context.Context, startDate, endDate string) ([]*model.MacroData, error) {
	params := map[string]interface{}{
		"start_date": startDate,
		"end_date":   endDate,
	}

	fields := "month,ppi_yoy,ppi_mp,ppi_accu,gdp_yoy,gdp_mp,gdp_accu"

	data, err := m.client.CallWithRetry(ctx, "cn_ppi", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get PPI data failed: %w", err)
	}

	return m.parseMacroData(data, "PPI")
}

// GetM2Data 获取货币供应量M2数据
func (m *MacroCollector) GetM2Data(ctx context.Context, startDate, endDate string) ([]*model.MacroData, error) {
	params := map[string]interface{}{
		"start_date": startDate,
		"end_date":   endDate,
	}

	fields := "month,m0,m0_yoy,m0_mom,m1,m1_yoy,m1_mom,m2,m2_yoy,m2_mom"

	data, err := m.client.CallWithRetry(ctx, "cn_m", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get M2 data failed: %w", err)
	}

	return m.parseMacroData(data, "M2")
}

// GetPMIData 获取PMI数据
func (m *MacroCollector) GetPMIData(ctx context.Context, startDate, endDate string) ([]*model.MacroData, error) {
	params := map[string]interface{}{
		"start_date": startDate,
		"end_date":   endDate,
	}

	fields := "month,pmi,qm,product,new_order,new_export_order,in_hand_order,finished_product,purchase_price,raw_material,employee,supplier_deliver,business_expect"

	data, err := m.client.CallWithRetry(ctx, "cn_pmi", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get PMI data failed: %w", err)
	}

	return m.parseMacroData(data, "PMI")
}

// GetShiborData 获取Shibor利率数据
func (m *MacroCollector) GetShiborData(ctx context.Context, startDate, endDate string) ([]*model.MacroData, error) {
	params := map[string]interface{}{
		"start_date": startDate,
		"end_date":   endDate,
	}

	fields := "date,on,1w,2w,1m,3m,6m,9m,1y"

	data, err := m.client.CallWithRetry(ctx, "shibor", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get Shibor data failed: %w", err)
	}

	return m.parseMacroData(data, "SHIBOR")
}

// GetLPRData 获取LPR利率数据
func (m *MacroCollector) GetLPRData(ctx context.Context, startDate, endDate string) ([]*model.MacroData, error) {
	params := map[string]interface{}{
		"start_date": startDate,
		"end_date":   endDate,
	}

	fields := "date,1y,5y"

	data, err := m.client.CallWithRetry(ctx, "cn_lpr", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get LPR data failed: %w", err)
	}

	return m.parseMacroData(data, "LPR")
}

// GetExchangeRateData 获取汇率数据
func (m *MacroCollector) GetExchangeRateData(ctx context.Context, startDate, endDate string) ([]*model.MacroData, error) {
	params := map[string]interface{}{
		"start_date": startDate,
		"end_date":   endDate,
	}

	fields := "date,usd,eur,100jpy,hkd,gbp,aud,nzd,sgd,chf,cad,myr,rub,zar,krw,aed,sar,huf,pln,dkk,sek,nok,try,mxn,thb"

	data, err := m.client.CallWithRetry(ctx, "cn_fx", params, fields, 3)
	if err != nil {
		return nil, fmt.Errorf("get exchange rate data failed: %w", err)
	}

	return m.parseMacroData(data, "EXCHANGE_RATE")
}

// parseMacroData 解析宏观数据
func (m *MacroCollector) parseMacroData(data *ResponseData, indicatorType string) ([]*model.MacroData, error) {
	if data == nil || len(data.Items) == 0 {
		return []*model.MacroData{}, nil
	}

	// 创建字段索引映射
	fieldMap := make(map[string]int)
	for i, field := range data.Fields {
		fieldMap[field] = i
	}

	macroData := make([]*model.MacroData, 0)
	for _, item := range data.Items {
		// 根据不同的指标类型解析数据
		switch indicatorType {
		case "GDP":
			macroData = append(macroData, m.parseGDPItem(item, fieldMap)...)
		case "CPI":
			macroData = append(macroData, m.parseCPIItem(item, fieldMap)...)
		case "PPI":
			macroData = append(macroData, m.parsePPIItem(item, fieldMap)...)
		case "M2":
			macroData = append(macroData, m.parseM2Item(item, fieldMap)...)
		case "PMI":
			macroData = append(macroData, m.parsePMIItem(item, fieldMap)...)
		case "SHIBOR":
			macroData = append(macroData, m.parseShiborItem(item, fieldMap)...)
		case "LPR":
			macroData = append(macroData, m.parseLPRItem(item, fieldMap)...)
		case "EXCHANGE_RATE":
			macroData = append(macroData, m.parseExchangeRateItem(item, fieldMap)...)
		}
	}

	return macroData, nil
}

// parseGDPItem 解析GDP数据项
func (m *MacroCollector) parseGDPItem(item []interface{}, fieldMap map[string]int) []*model.MacroData {
	var results []*model.MacroData
	quarter := m.parseString(item, fieldMap, "quarter")
	if quarter == "" {
		return results
	}

	// 解析日期
	dataDate, err := m.parseQuarterDate(quarter)
	if err != nil {
		return results
	}

	// GDP总值
	if gdp := m.parseFloat(item, fieldMap, "gdp"); gdp != 0 {
		results = append(results, &model.MacroData{
			IndicatorCode: "GDP",
			IndicatorName: "国内生产总值",
			PeriodType:    model.PeriodTypeQuarterly,
			DataDate:      dataDate,
			Value:         gdp,
			CreatedAt:     time.Now(),
		})
	}

	// GDP同比增长率
	if gdpYoy := m.parseFloat(item, fieldMap, "gdp_yoy"); gdpYoy != 0 {
		results = append(results, &model.MacroData{
			IndicatorCode: "GDP_YOY",
			IndicatorName: "GDP同比增长率",
			PeriodType:    model.PeriodTypeQuarterly,
			DataDate:      dataDate,
			Value:         gdpYoy,
			CreatedAt:     time.Now(),
		})
	}

	return results
}

// parseCPIItem 解析CPI数据项
func (m *MacroCollector) parseCPIItem(item []interface{}, fieldMap map[string]int) []*model.MacroData {
	var results []*model.MacroData
	month := m.parseString(item, fieldMap, "month")
	if month == "" {
		return results
	}

	// 解析日期
	dataDate, err := m.parseMonthDate(month)
	if err != nil {
		return results
	}

	// 全国CPI同比
	if ntYoy := m.parseFloat(item, fieldMap, "nt_yoy"); ntYoy != 0 {
		results = append(results, &model.MacroData{
			IndicatorCode: "CPI_YOY",
			IndicatorName: "居民消费价格指数同比",
			PeriodType:    model.PeriodTypeMonthly,
			DataDate:      dataDate,
			Value:         ntYoy,
			CreatedAt:     time.Now(),
		})
	}

	// 全国CPI环比
	if ntMom := m.parseFloat(item, fieldMap, "nt_mom"); ntMom != 0 {
		results = append(results, &model.MacroData{
			IndicatorCode: "CPI_MOM",
			IndicatorName: "居民消费价格指数环比",
			PeriodType:    model.PeriodTypeMonthly,
			DataDate:      dataDate,
			Value:         ntMom,
			CreatedAt:     time.Now(),
		})
	}

	return results
}

// parsePPIItem 解析PPI数据项
func (m *MacroCollector) parsePPIItem(item []interface{}, fieldMap map[string]int) []*model.MacroData {
	var results []*model.MacroData
	month := m.parseString(item, fieldMap, "month")
	if month == "" {
		return results
	}

	// 解析日期
	dataDate, err := m.parseMonthDate(month)
	if err != nil {
		return results
	}

	// PPI同比
	if ppiYoy := m.parseFloat(item, fieldMap, "ppi_yoy"); ppiYoy != 0 {
		results = append(results, &model.MacroData{
			IndicatorCode: "PPI_YOY",
			IndicatorName: "工业生产者出厂价格指数同比",
			PeriodType:    model.PeriodTypeMonthly,
			DataDate:      dataDate,
			Value:         ppiYoy,
			CreatedAt:     time.Now(),
		})
	}

	return results
}

// parseM2Item 解析M2数据项
func (m *MacroCollector) parseM2Item(item []interface{}, fieldMap map[string]int) []*model.MacroData {
	var results []*model.MacroData
	month := m.parseString(item, fieldMap, "month")
	if month == "" {
		return results
	}

	// 解析日期
	dataDate, err := m.parseMonthDate(month)
	if err != nil {
		return results
	}

	// M2同比增长率
	if m2Yoy := m.parseFloat(item, fieldMap, "m2_yoy"); m2Yoy != 0 {
		results = append(results, &model.MacroData{
			IndicatorCode: "M2_YOY",
			IndicatorName: "货币供应量M2同比增长率",
			PeriodType:    model.PeriodTypeMonthly,
			DataDate:      dataDate,
			Value:         m2Yoy,
			CreatedAt:     time.Now(),
		})
	}

	return results
}

// parsePMIItem 解析PMI数据项
func (m *MacroCollector) parsePMIItem(item []interface{}, fieldMap map[string]int) []*model.MacroData {
	var results []*model.MacroData
	month := m.parseString(item, fieldMap, "month")
	if month == "" {
		return results
	}

	// 解析日期
	dataDate, err := m.parseMonthDate(month)
	if err != nil {
		return results
	}

	// PMI指数
	if pmi := m.parseFloat(item, fieldMap, "pmi"); pmi != 0 {
		results = append(results, &model.MacroData{
			IndicatorCode: "PMI",
			IndicatorName: "制造业采购经理指数",
			PeriodType:    model.PeriodTypeMonthly,
			DataDate:      dataDate,
			Value:         pmi,
			CreatedAt:     time.Now(),
		})
	}

	return results
}

// parseShiborItem 解析Shibor数据项
func (m *MacroCollector) parseShiborItem(item []interface{}, fieldMap map[string]int) []*model.MacroData {
	var results []*model.MacroData
	dateStr := m.parseString(item, fieldMap, "date")
	if dateStr == "" {
		return results
	}

	// 解析日期
	dataDate, err := time.Parse("20060102", dateStr)
	if err != nil {
		return results
	}

	// 各期限Shibor利率
	periods := []struct {
		field string
		code  string
		name  string
	}{
		{"on", "SHIBOR_ON", "Shibor隔夜利率"},
		{"1w", "SHIBOR_1W", "Shibor1周利率"},
		{"2w", "SHIBOR_2W", "Shibor2周利率"},
		{"1m", "SHIBOR_1M", "Shibor1月利率"},
		{"3m", "SHIBOR_3M", "Shibor3月利率"},
		{"6m", "SHIBOR_6M", "Shibor6月利率"},
		{"9m", "SHIBOR_9M", "Shibor9月利率"},
		{"1y", "SHIBOR_1Y", "Shibor1年利率"},
	}

	for _, period := range periods {
		if rate := m.parseFloat(item, fieldMap, period.field); rate != 0 {
			results = append(results, &model.MacroData{
				IndicatorCode: period.code,
				IndicatorName: period.name,
				PeriodType:    model.PeriodTypeDaily,
				DataDate:      dataDate,
				Value:         rate,
				CreatedAt:     time.Now(),
			})
		}
	}

	return results
}

// parseLPRItem 解析LPR数据项
func (m *MacroCollector) parseLPRItem(item []interface{}, fieldMap map[string]int) []*model.MacroData {
	var results []*model.MacroData
	dateStr := m.parseString(item, fieldMap, "date")
	if dateStr == "" {
		return results
	}

	// 解析日期
	dataDate, err := time.Parse("20060102", dateStr)
	if err != nil {
		return results
	}

	// LPR 1年期
	if lpr1y := m.parseFloat(item, fieldMap, "1y"); lpr1y != 0 {
		results = append(results, &model.MacroData{
			IndicatorCode: "LPR_1Y",
			IndicatorName: "贷款市场报价利率1年期",
			PeriodType:    model.PeriodTypeDaily,
			DataDate:      dataDate,
			Value:         lpr1y,
			CreatedAt:     time.Now(),
		})
	}

	// LPR 5年期
	if lpr5y := m.parseFloat(item, fieldMap, "5y"); lpr5y != 0 {
		results = append(results, &model.MacroData{
			IndicatorCode: "LPR_5Y",
			IndicatorName: "贷款市场报价利率5年期",
			PeriodType:    model.PeriodTypeDaily,
			DataDate:      dataDate,
			Value:         lpr5y,
			CreatedAt:     time.Now(),
		})
	}

	return results
}

// parseExchangeRateItem 解析汇率数据项
func (m *MacroCollector) parseExchangeRateItem(item []interface{}, fieldMap map[string]int) []*model.MacroData {
	var results []*model.MacroData
	dateStr := m.parseString(item, fieldMap, "date")
	if dateStr == "" {
		return results
	}

	// 解析日期
	dataDate, err := time.Parse("20060102", dateStr)
	if err != nil {
		return results
	}

	// 主要货币汇率
	currencies := []struct {
		field string
		code  string
		name  string
	}{
		{"usd", "USD_CNY", "美元兑人民币汇率"},
		{"eur", "EUR_CNY", "欧元兑人民币汇率"},
		{"100jpy", "JPY_CNY", "100日元兑人民币汇率"},
		{"hkd", "HKD_CNY", "港币兑人民币汇率"},
		{"gbp", "GBP_CNY", "英镑兑人民币汇率"},
	}

	for _, currency := range currencies {
		if rate := m.parseFloat(item, fieldMap, currency.field); rate != 0 {
			results = append(results, &model.MacroData{
				IndicatorCode: currency.code,
				IndicatorName: currency.name,
				PeriodType:    model.PeriodTypeDaily,
				DataDate:      dataDate,
				Value:         rate,
				CreatedAt:     time.Now(),
			})
		}
	}

	return results
}

// parseFloat 解析浮点数
func (m *MacroCollector) parseFloat(item []interface{}, fieldMap map[string]int, field string) float64 {
	if idx, ok := fieldMap[field]; ok && idx < len(item) {
		switch v := item[idx].(type) {
		case float64:
			return v
		case string:
			if v == "" || v == "null" {
				return 0
			}
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case nil:
			return 0
		}
	}
	return 0
}

// parseString 解析字符串
func (m *MacroCollector) parseString(item []interface{}, fieldMap map[string]int, field string) string {
	if idx, ok := fieldMap[field]; ok && idx < len(item) {
		if str, ok := item[idx].(string); ok {
			return str
		}
	}
	return ""
}

// parseQuarterDate 解析季度日期
func (m *MacroCollector) parseQuarterDate(quarter string) (time.Time, error) {
	// 格式: 2022Q1, 2022Q2, 2022Q3, 2022Q4
	if len(quarter) != 6 {
		return time.Time{}, fmt.Errorf("invalid quarter format: %s", quarter)
	}

	yearStr := quarter[:4]
	quarterStr := quarter[5:]

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid year: %s", yearStr)
	}

	var month int
	switch quarterStr {
	case "1":
		month = 3 // Q1结束于3月
	case "2":
		month = 6 // Q2结束于6月
	case "3":
		month = 9 // Q3结束于9月
	case "4":
		month = 12 // Q4结束于12月
	default:
		return time.Time{}, fmt.Errorf("invalid quarter: %s", quarterStr)
	}

	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC), nil
}

// parseMonthDate 解析月份日期
func (m *MacroCollector) parseMonthDate(month string) (time.Time, error) {
	// 格式: 202201, 202202, ...
	if len(month) != 6 {
		return time.Time{}, fmt.Errorf("invalid month format: %s", month)
	}

	return time.Parse("200601", month)
}