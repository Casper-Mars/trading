package handlers

import (
	"context"
	"time"

	"data-collection-system/pkg/response"
	"data-collection-system/service/collection"

	"github.com/gin-gonic/gin"
)

// CollectionHandler 数据采集处理器
type CollectionHandler struct {
	collectionService *collection.Service
}

// NewCollectionHandler 创建数据采集处理器
func NewCollectionHandler(collectionService *collection.Service) *CollectionHandler {
	return &CollectionHandler{
		collectionService: collectionService,
	}
}

// CollectStockBasicData 采集股票基础数据
func (h *CollectionHandler) CollectStockBasicData(c *gin.Context) {
	ctx := context.Background()

	err := h.collectionService.CollectStockBasicData(ctx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "股票基础数据采集任务已启动",
		"timestamp": time.Now(),
	})
}

// CollectDailyMarketData 采集日线行情数据
func (h *CollectionHandler) CollectDailyMarketData(c *gin.Context) {
	ctx := context.Background()

	// 获取交易日期参数，默认为今日
	tradeDate := c.DefaultQuery("trade_date", time.Now().Format("20060102"))

	err := h.collectionService.CollectDailyMarketData(ctx, tradeDate)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "日线行情数据采集任务已启动",
		"trade_date": tradeDate,
		"timestamp": time.Now(),
	})
}

// CollectStockHistoryData 采集指定股票的历史行情数据
func (h *CollectionHandler) CollectStockHistoryData(c *gin.Context) {
	ctx := context.Background()

	// 绑定请求参数
	type HistoryDataRequest struct {
		Symbol    string `json:"symbol" binding:"required"`
		StartDate string `json:"start_date" binding:"required"`
		EndDate   string `json:"end_date" binding:"required"`
	}

	var req HistoryDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	err := h.collectionService.CollectStockHistoryData(ctx, req.Symbol, req.StartDate, req.EndDate)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "股票历史数据采集任务已启动",
		"symbol": req.Symbol,
		"start_date": req.StartDate,
		"end_date": req.EndDate,
		"timestamp": time.Now(),
	})
}

// CollectFinancialData 采集财务数据
func (h *CollectionHandler) CollectFinancialData(c *gin.Context) {
	ctx := context.Background()

	// 绑定请求参数
	type FinancialDataRequest struct {
		Symbol string `json:"symbol" binding:"required"`
		Period string `json:"period"` // 可选，默认为最新期
	}

	var req FinancialDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	err := h.collectionService.CollectFinancialData(ctx, req.Symbol, req.Period)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "财务数据采集任务已启动",
		"symbol": req.Symbol,
		"period": req.Period,
		"timestamp": time.Now(),
	})
}

// CollectMacroData 采集宏观经济数据
func (h *CollectionHandler) CollectMacroData(c *gin.Context) {
	ctx := context.Background()

	// 绑定请求参数
	type MacroDataRequest struct {
		StartDate string `json:"start_date" binding:"required"`
		EndDate   string `json:"end_date" binding:"required"`
	}

	var req MacroDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	err := h.collectionService.CollectMacroData(ctx, req.StartDate, req.EndDate)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "宏观经济数据采集任务已启动",
		"start_date": req.StartDate,
		"end_date": req.EndDate,
		"timestamp": time.Now(),
	})
}

// CollectRealtimeData 采集实时行情数据
func (h *CollectionHandler) CollectRealtimeData(c *gin.Context) {
	ctx := context.Background()

	// 绑定请求参数
	type RealtimeDataRequest struct {
		Symbols []string `json:"symbols" binding:"required"`
	}

	var req RealtimeDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	err := h.collectionService.CollectRealtimeData(ctx, req.Symbols)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "实时行情数据采集任务已启动",
		"symbols": req.Symbols,
		"timestamp": time.Now(),
	})
}

// BatchCollectStockData 批量采集股票数据
func (h *CollectionHandler) BatchCollectStockData(c *gin.Context) {
	ctx := context.Background()

	// 绑定请求参数
	type BatchCollectRequest struct {
		Symbols   []string `json:"symbols" binding:"required"`
		StartDate string   `json:"start_date" binding:"required"`
		EndDate   string   `json:"end_date" binding:"required"`
	}

	var req BatchCollectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	err := h.collectionService.BatchCollectStockData(ctx, req.Symbols, req.StartDate, req.EndDate)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "批量股票数据采集任务已启动",
		"symbols": req.Symbols,
		"start_date": req.StartDate,
		"end_date": req.EndDate,
		"timestamp": time.Now(),
	})
}

// SyncStockList 同步股票列表
func (h *CollectionHandler) SyncStockList(c *gin.Context) {
	ctx := context.Background()

	err := h.collectionService.SyncStockList(ctx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "股票列表同步任务已启动",
		"timestamp": time.Now(),
	})
}

// CollectTodayData 采集今日数据（综合任务）
func (h *CollectionHandler) CollectTodayData(c *gin.Context) {
	ctx := context.Background()

	err := h.collectionService.CollectTodayData(ctx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "今日数据采集任务已启动",
		"timestamp": time.Now(),
	})
}

// GetCollectorStatus 获取采集器状态
func (h *CollectionHandler) GetCollectorStatus(c *gin.Context) {
	ctx := context.Background()

	status, err := h.collectionService.GetCollectorStatus(ctx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"status": status,
		"timestamp": time.Now(),
	})
}

// CollectMoneyFlowData 采集个股资金流向数据
func (h *CollectionHandler) CollectMoneyFlowData(c *gin.Context) {
	ctx := context.Background()

	// 绑定请求参数
	type MoneyFlowRequest struct {
		Symbol    string `json:"symbol" binding:"required"`
		TradeDate string `json:"trade_date"` // 可选，默认为今日
	}

	var req MoneyFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	// 如果没有指定交易日期，使用今日
	if req.TradeDate == "" {
		req.TradeDate = time.Now().Format("20060102")
	}

	err := h.collectionService.CollectMoneyFlowData(ctx, req.Symbol, req.TradeDate)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "资金流向数据采集任务已启动",
		"symbol": req.Symbol,
		"trade_date": req.TradeDate,
		"timestamp": time.Now(),
	})
}

// CollectNorthboundFundData 采集北向资金数据
func (h *CollectionHandler) CollectNorthboundFundData(c *gin.Context) {
	ctx := context.Background()

	// 获取交易日期参数，默认为今日
	tradeDate := c.DefaultQuery("trade_date", time.Now().Format("20060102"))

	err := h.collectionService.CollectNorthboundFundData(ctx, tradeDate)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "北向资金数据采集任务已启动",
		"trade_date": tradeDate,
		"timestamp": time.Now(),
	})
}

// CollectNorthboundTopStocksData 采集北向资金十大成交股数据
func (h *CollectionHandler) CollectNorthboundTopStocksData(c *gin.Context) {
	ctx := context.Background()

	// 绑定请求参数
	type NorthboundTopStocksRequest struct {
		TradeDate string `json:"trade_date"` // 可选，默认为今日
		Market    string `json:"market"`     // 可选，默认为空（全市场）
	}

	var req NorthboundTopStocksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果JSON绑定失败，尝试从查询参数获取
		req.TradeDate = c.DefaultQuery("trade_date", time.Now().Format("20060102"))
		req.Market = c.DefaultQuery("market", "")
	} else {
		// 设置默认值
		if req.TradeDate == "" {
			req.TradeDate = time.Now().Format("20060102")
		}
	}

	err := h.collectionService.CollectNorthboundTopStocksData(ctx, req.TradeDate, req.Market)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "北向资金十大成交股数据采集任务已启动",
		"trade_date": req.TradeDate,
		"market": req.Market,
		"timestamp": time.Now(),
	})
}

// CollectMarginTradingData 采集融资融券数据
func (h *CollectionHandler) CollectMarginTradingData(c *gin.Context) {
	ctx := context.Background()

	// 绑定请求参数
	type MarginTradingRequest struct {
		TradeDate string `json:"trade_date"` // 可选，默认为今日
		Exchange  string `json:"exchange"`   // 可选，默认为空（全交易所）
	}

	var req MarginTradingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果JSON绑定失败，尝试从查询参数获取
		req.TradeDate = c.DefaultQuery("trade_date", time.Now().Format("20060102"))
		req.Exchange = c.DefaultQuery("exchange", "")
	} else {
		// 设置默认值
		if req.TradeDate == "" {
			req.TradeDate = time.Now().Format("20060102")
		}
	}

	err := h.collectionService.CollectMarginTradingData(ctx, req.TradeDate, req.Exchange)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "融资融券数据采集任务已启动",
		"trade_date": req.TradeDate,
		"exchange": req.Exchange,
		"timestamp": time.Now(),
	})
}

// CollectETFBasicData 采集ETF基础数据
func (h *CollectionHandler) CollectETFBasicData(c *gin.Context) {
	ctx := context.Background()

	// 获取市场参数，默认为空（全市场）
	market := c.DefaultQuery("market", "")

	err := h.collectionService.CollectETFBasicData(ctx, market)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "ETF基础数据采集任务已启动",
		"market": market,
		"timestamp": time.Now(),
	})
}

// CollectAllSentimentData 采集所有市场情绪和资金流向数据
func (h *CollectionHandler) CollectAllSentimentData(c *gin.Context) {
	ctx := context.Background()

	// 获取交易日期参数，默认为今日
	tradeDate := c.DefaultQuery("trade_date", time.Now().Format("20060102"))

	err := h.collectionService.CollectAllSentimentData(ctx, tradeDate)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "所有市场情绪和资金流向数据采集任务已启动",
		"trade_date": tradeDate,
		"timestamp": time.Now(),
	})
}

// CollectActiveStocksMoneyFlow 采集活跃股票的资金流向数据
func (h *CollectionHandler) CollectActiveStocksMoneyFlow(c *gin.Context) {
	ctx := context.Background()

	// 绑定请求参数
	type ActiveStocksMoneyFlowRequest struct {
		TradeDate string   `json:"trade_date"` // 可选，默认为今日
		Symbols   []string `json:"symbols" binding:"required"`
	}

	var req ActiveStocksMoneyFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	// 如果没有指定交易日期，使用今日
	if req.TradeDate == "" {
		req.TradeDate = time.Now().Format("20060102")
	}

	err := h.collectionService.CollectActiveStocksMoneyFlow(ctx, req.TradeDate, req.Symbols)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "活跃股票资金流向数据采集任务已启动",
		"trade_date": req.TradeDate,
		"symbols": req.Symbols,
		"timestamp": time.Now(),
	})
}

// GetSentimentCollectorStatus 获取市场情绪数据采集器状态
func (h *CollectionHandler) GetSentimentCollectorStatus(c *gin.Context) {
	ctx := context.Background()

	status, err := h.collectionService.GetSentimentCollectorStatus(ctx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"status": status,
		"timestamp": time.Now(),
	})
}