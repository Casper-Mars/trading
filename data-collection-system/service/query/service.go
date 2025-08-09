package query

import (
	"context"
	"time"

	"data-collection-system/model"
	"data-collection-system/pkg/errors"
	dao "data-collection-system/repo/mysql"
)

// QueryService 数据查询服务
type QueryService struct {
	daoManager dao.DAOManager
}

// NewQueryService 创建查询服务实例
func NewQueryService(daoManager dao.DAOManager) *QueryService {
	return &QueryService{
		daoManager: daoManager,
	}
}

// StockQueryParams 股票查询参数
type StockQueryParams struct {
	Symbol   string `form:"symbol"`
	Exchange string `form:"exchange"`
	Industry string `form:"industry"`
	Status   string `form:"status"`
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
}

// MarketDataQueryParams 行情数据查询参数
type MarketDataQueryParams struct {
	Symbol    string `form:"symbol" binding:"required"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	Period    string `form:"period"`
	Page      int    `form:"page" binding:"min=1"`
	PageSize  int    `form:"page_size" binding:"min=1,max=1000"`
}

// FinancialDataQueryParams 财务数据查询参数
type FinancialDataQueryParams struct {
	Symbol     string `form:"symbol" binding:"required"`
	StartDate  string `form:"start_date"`
	EndDate    string `form:"end_date"`
	ReportType string `form:"report_type"`
	Page       int    `form:"page" binding:"min=1"`
	PageSize   int    `form:"page_size" binding:"min=1,max=100"`
}

// NewsDataQueryParams 新闻数据查询参数
type NewsDataQueryParams struct {
	Keyword      string `form:"keyword"`
	Category     string `form:"category"`
	RelatedStock string `form:"related_stock"`
	Sentiment    *int8  `form:"sentiment"`
	Importance   *int8  `form:"importance"`
	StartTime    string `form:"start_time"`
	EndTime      string `form:"end_time"`
	Page         int    `form:"page" binding:"min=1"`
	PageSize     int    `form:"page_size" binding:"min=1,max=100"`
}

// MacroDataQueryParams 宏观数据查询参数
type MacroDataQueryParams struct {
	IndicatorCode string `form:"indicator_code"`
	PeriodType    string `form:"period_type"`
	StartDate     string `form:"start_date"`
	EndDate       string `form:"end_date"`
	Page          int    `form:"page" binding:"min=1"`
	PageSize      int    `form:"page_size" binding:"min=1,max=100"`
}

// QueryResult 查询结果
type QueryResult struct {
	Data  interface{} `json:"data"`
	Total int64       `json:"total"`
}

// GetStocks 获取股票列表
func (s *QueryService) GetStocks(ctx context.Context, params *StockQueryParams) (*QueryResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	offset := (params.Page - 1) * params.PageSize
	limit := params.PageSize

	var stocks []*model.Stock
	var err error

	// 根据查询条件获取股票数据
	switch {
	case params.Symbol != "":
		// 根据股票代码查询单个股票
		stock, err := s.daoManager.Stock().GetBySymbol(ctx, params.Symbol)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeDataNotFound, "股票不存在")
		}
		stocks = []*model.Stock{stock}
	case params.Exchange != "":
		// 根据交易所查询
		stocks, err = s.daoManager.Stock().GetByExchange(ctx, params.Exchange, limit, offset)
	case params.Industry != "":
		// 根据行业查询
		stocks, err = s.daoManager.Stock().GetByIndustry(ctx, params.Industry, limit, offset)
	default:
		// 获取活跃股票列表
		stocks, err = s.daoManager.Stock().GetActiveStocks(ctx, limit, offset)
	}

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询股票数据失败")
	}

	// 获取总数
	total, err := s.daoManager.Stock().Count(ctx)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "获取股票总数失败")
	}

	return &QueryResult{
		Data:  stocks,
		Total: total,
	}, nil
}

// GetMarketData 获取行情数据
func (s *QueryService) GetMarketData(ctx context.Context, params *MarketDataQueryParams) (*QueryResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 100
	}
	if params.Period == "" {
		params.Period = "daily"
	}

	offset := (params.Page - 1) * params.PageSize
	limit := params.PageSize

	// 解析时间范围
	var startDate, endDate time.Time
	var err error

	if params.StartDate != "" {
		startDate, err = time.Parse("2006-01-02", params.StartDate)
		if err != nil {
			return nil, errors.Newf(errors.ErrCodeInvalidParam, "开始日期格式错误: %s", params.StartDate)
		}
	} else {
		// 默认查询最近30天
		startDate = time.Now().AddDate(0, 0, -30)
	}

	if params.EndDate != "" {
		endDate, err = time.Parse("2006-01-02", params.EndDate)
		if err != nil {
			return nil, errors.Newf(errors.ErrCodeInvalidParam, "结束日期格式错误: %s", params.EndDate)
		}
	} else {
		// 默认到今天
		endDate = time.Now()
	}

	// 查询行情数据
	marketData, err := s.daoManager.MarketData().GetBySymbol(ctx, params.Symbol, startDate, endDate, params.Period, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询行情数据失败")
	}

	// 获取总数
	total, err := s.daoManager.MarketData().Count(ctx)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "获取行情数据总数失败")
	}

	return &QueryResult{
		Data:  marketData,
		Total: total,
	}, nil
}

// GetFinancialData 获取财务数据
func (s *QueryService) GetFinancialData(ctx context.Context, params *FinancialDataQueryParams) (*QueryResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.ReportType == "" {
		params.ReportType = "annual"
	}

	offset := (params.Page - 1) * params.PageSize
	limit := params.PageSize

	// 解析时间范围
	var startDate, endDate time.Time
	var err error

	if params.StartDate != "" {
		startDate, err = time.Parse("2006-01-02", params.StartDate)
		if err != nil {
			return nil, errors.Newf(errors.ErrCodeInvalidParam, "开始日期格式错误: %s", params.StartDate)
		}
	} else {
		// 默认查询最近3年
		startDate = time.Now().AddDate(-3, 0, 0)
	}

	if params.EndDate != "" {
		endDate, err = time.Parse("2006-01-02", params.EndDate)
		if err != nil {
			return nil, errors.Newf(errors.ErrCodeInvalidParam, "结束日期格式错误: %s", params.EndDate)
		}
	} else {
		// 默认到今天
		endDate = time.Now()
	}

	// 查询财务数据
	financialData, err := s.daoManager.FinancialData().GetBySymbol(ctx, params.Symbol, startDate, endDate, params.ReportType, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询财务数据失败")
	}

	// 获取总数
	total, err := s.daoManager.FinancialData().Count(ctx)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "获取财务数据总数失败")
	}

	return &QueryResult{
		Data:  financialData,
		Total: total,
	}, nil
}

// GetNewsData 获取新闻数据
func (s *QueryService) GetNewsData(ctx context.Context, params *NewsDataQueryParams) (*QueryResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	offset := (params.Page - 1) * params.PageSize
	limit := params.PageSize

	var newsData []*model.NewsData
	var err error

	// 根据查询条件获取新闻数据
	switch {
	case params.Keyword != "":
		// 关键词搜索
		newsData, err = s.daoManager.NewsData().SearchByKeyword(ctx, params.Keyword, limit, offset)
	case params.RelatedStock != "":
		// 根据相关股票查询
		newsData, err = s.daoManager.NewsData().GetByRelatedStock(ctx, params.RelatedStock, limit, offset)
	case params.Category != "":
		// 根据分类查询
		newsData, err = s.daoManager.NewsData().GetByCategory(ctx, params.Category, limit, offset)
	case params.Sentiment != nil:
		// 根据情感倾向查询
		newsData, err = s.daoManager.NewsData().GetBySentiment(ctx, *params.Sentiment, limit, offset)
	case params.Importance != nil:
		// 根据重要程度查询
		newsData, err = s.daoManager.NewsData().GetByImportance(ctx, *params.Importance, limit, offset)
	case params.StartTime != "" || params.EndTime != "":
		// 根据时间范围查询
		var startTime, endTime time.Time
		if params.StartTime != "" {
			startTime, err = time.Parse("2006-01-02 15:04:05", params.StartTime)
			if err != nil {
				return nil, errors.Newf(errors.ErrCodeInvalidParam, "开始时间格式错误: %s", params.StartTime)
			}
		} else {
			startTime = time.Now().AddDate(0, 0, -7) // 默认查询最近7天
		}
		if params.EndTime != "" {
			endTime, err = time.Parse("2006-01-02 15:04:05", params.EndTime)
			if err != nil {
				return nil, errors.Newf(errors.ErrCodeInvalidParam, "结束时间格式错误: %s", params.EndTime)
			}
		} else {
			endTime = time.Now()
		}
		newsData, err = s.daoManager.NewsData().GetByTimeRange(ctx, startTime, endTime, limit, offset)
	default:
		// 默认查询最近的新闻
		startTime := time.Now().AddDate(0, 0, -1) // 最近1天
		endTime := time.Now()
		newsData, err = s.daoManager.NewsData().GetByTimeRange(ctx, startTime, endTime, limit, offset)
	}

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询新闻数据失败")
	}

	// 获取总数
	total, err := s.daoManager.NewsData().Count(ctx)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "获取新闻数据总数失败")
	}

	return &QueryResult{
		Data:  newsData,
		Total: total,
	}, nil
}

// GetMacroData 获取宏观数据
func (s *QueryService) GetMacroData(ctx context.Context, params *MacroDataQueryParams) (*QueryResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 50
	}

	offset := (params.Page - 1) * params.PageSize
	limit := params.PageSize

	var macroData []*model.MacroData
	var err error

	// 根据查询条件获取宏观数据
	switch {
	case params.IndicatorCode != "":
		// 根据指标代码查询
		var startDate, endDate time.Time
		if params.StartDate != "" {
			startDate, err = time.Parse("2006-01-02", params.StartDate)
			if err != nil {
				return nil, errors.Newf(errors.ErrCodeInvalidParam, "开始日期格式错误: %s", params.StartDate)
			}
		} else {
			startDate = time.Now().AddDate(-1, 0, 0) // 默认查询最近1年
		}
		if params.EndDate != "" {
			endDate, err = time.Parse("2006-01-02", params.EndDate)
			if err != nil {
				return nil, errors.Newf(errors.ErrCodeInvalidParam, "结束日期格式错误: %s", params.EndDate)
			}
		} else {
			endDate = time.Now()
		}
		macroData, err = s.daoManager.MacroData().GetByIndicator(ctx, params.IndicatorCode, startDate, endDate, limit, offset)
	case params.PeriodType != "":
		// 根据周期类型查询
		macroData, err = s.daoManager.MacroData().GetByPeriodType(ctx, params.PeriodType, limit, offset)
	case params.StartDate != "" || params.EndDate != "":
		// 根据日期范围查询
		var startDate, endDate time.Time
		if params.StartDate != "" {
			startDate, err = time.Parse("2006-01-02", params.StartDate)
			if err != nil {
				return nil, errors.Newf(errors.ErrCodeInvalidParam, "开始日期格式错误: %s", params.StartDate)
			}
		} else {
			startDate = time.Now().AddDate(-1, 0, 0) // 默认查询最近1年
		}
		if params.EndDate != "" {
			endDate, err = time.Parse("2006-01-02", params.EndDate)
			if err != nil {
				return nil, errors.Newf(errors.ErrCodeInvalidParam, "结束日期格式错误: %s", params.EndDate)
			}
		} else {
			endDate = time.Now()
		}
		macroData, err = s.daoManager.MacroData().GetByDateRange(ctx, startDate, endDate, limit, offset)
	default:
		// 默认查询最近的宏观数据
		startDate := time.Now().AddDate(0, -3, 0) // 最近3个月
		endDate := time.Now()
		macroData, err = s.daoManager.MacroData().GetByDateRange(ctx, startDate, endDate, limit, offset)
	}

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询宏观数据失败")
	}

	// 获取总数
	total, err := s.daoManager.MacroData().Count(ctx)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "获取宏观数据总数失败")
	}

	return &QueryResult{
		Data:  macroData,
		Total: total,
	}, nil
}

// GetStockBySymbol 根据股票代码获取股票详情
func (s *QueryService) GetStockBySymbol(ctx context.Context, symbol string) (*model.Stock, error) {
	stock, err := s.daoManager.Stock().GetBySymbol(ctx, symbol)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDataNotFound, "股票不存在")
	}
	return stock, nil
}

// GetLatestMarketData 获取最新行情数据
func (s *QueryService) GetLatestMarketData(ctx context.Context, symbol, period string) (*model.MarketData, error) {
	if period == "" {
		period = "daily"
	}
	data, err := s.daoManager.MarketData().GetLatest(ctx, symbol, period)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDataNotFound, "行情数据不存在")
	}
	return data, nil
}

// GetLatestFinancialData 获取最新财务数据
func (s *QueryService) GetLatestFinancialData(ctx context.Context, symbol, reportType string) (*model.FinancialData, error) {
	if reportType == "" {
		reportType = "annual"
	}
	data, err := s.daoManager.FinancialData().GetLatest(ctx, symbol, reportType)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDataNotFound, "财务数据不存在")
	}
	return data, nil
}
