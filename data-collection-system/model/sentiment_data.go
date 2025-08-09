package model

import (
	"time"
)

// MarketSentimentData 市场情绪数据
type MarketSentimentData struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TradeDate   time.Time `json:"trade_date" gorm:"index;not null;comment:交易日期"`
	DataType    string    `json:"data_type" gorm:"index;size:50;not null;comment:数据类型"`
	Value       float64   `json:"value" gorm:"comment:指标值"`
	Description string    `json:"description" gorm:"size:200;comment:描述"`
	Source      string    `json:"source" gorm:"size:50;comment:数据来源"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (MarketSentimentData) TableName() string {
	return "market_sentiment_data"
}

// MoneyFlowData 资金流向数据
type MoneyFlowData struct {
	ID             int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Symbol         string    `json:"symbol" gorm:"index;size:20;not null;comment:股票代码"`
	TradeDate      time.Time `json:"trade_date" gorm:"index;not null;comment:交易日期"`
	// 小单数据
	BuySmallVol    int64   `json:"buy_small_vol" gorm:"comment:小单买入量(手)"`
	BuySmallAmount float64 `json:"buy_small_amount" gorm:"comment:小单买入金额(元)"`
	SellSmallVol   int64   `json:"sell_small_vol" gorm:"comment:小单卖出量(手)"`
	SellSmallAmount float64 `json:"sell_small_amount" gorm:"comment:小单卖出金额(元)"`
	// 中单数据
	BuyMediumVol    int64   `json:"buy_medium_vol" gorm:"comment:中单买入量(手)"`
	BuyMediumAmount float64 `json:"buy_medium_amount" gorm:"comment:中单买入金额(元)"`
	SellMediumVol   int64   `json:"sell_medium_vol" gorm:"comment:中单卖出量(手)"`
	SellMediumAmount float64 `json:"sell_medium_amount" gorm:"comment:中单卖出金额(元)"`
	// 大单数据
	BuyLargeVol    int64   `json:"buy_large_vol" gorm:"comment:大单买入量(手)"`
	BuyLargeAmount float64 `json:"buy_large_amount" gorm:"comment:大单买入金额(元)"`
	SellLargeVol   int64   `json:"sell_large_vol" gorm:"comment:大单卖出量(手)"`
	SellLargeAmount float64 `json:"sell_large_amount" gorm:"comment:大单卖出金额(元)"`
	// 特大单数据
	BuyExtraLargeVol    int64   `json:"buy_extra_large_vol" gorm:"comment:特大单买入量(手)"`
	BuyExtraLargeAmount float64 `json:"buy_extra_large_amount" gorm:"comment:特大单买入金额(元)"`
	SellExtraLargeVol   int64   `json:"sell_extra_large_vol" gorm:"comment:特大单卖出量(手)"`
	SellExtraLargeAmount float64 `json:"sell_extra_large_amount" gorm:"comment:特大单卖出金额(元)"`
	// 净流入数据
	NetFlowVol    int64   `json:"net_flow_vol" gorm:"comment:净流入量(手)"`
	NetFlowAmount float64 `json:"net_flow_amount" gorm:"comment:净流入金额(元)"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (MoneyFlowData) TableName() string {
	return "money_flow_data"
}

// NorthboundFundData 北向资金数据
type NorthboundFundData struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TradeDate   time.Time `json:"trade_date" gorm:"index;not null;comment:交易日期"`
	MarketType  int       `json:"market_type" gorm:"index;comment:市场类型(1:沪市 3:深市)"`
	// 沪股通数据
	HSGTBuyAmount  float64 `json:"hsgt_buy_amount" gorm:"comment:沪股通买入金额(元)"`
	HSGTSellAmount float64 `json:"hsgt_sell_amount" gorm:"comment:沪股通卖出金额(元)"`
	HSGTNetAmount  float64 `json:"hsgt_net_amount" gorm:"comment:沪股通净流入金额(元)"`
	// 深股通数据
	SZGTBuyAmount  float64 `json:"szgt_buy_amount" gorm:"comment:深股通买入金额(元)"`
	SZGTSellAmount float64 `json:"szgt_sell_amount" gorm:"comment:深股通卖出金额(元)"`
	SZGTNetAmount  float64 `json:"szgt_net_amount" gorm:"comment:深股通净流入金额(元)"`
	// 港股通数据
	HKGTBuyAmount  float64 `json:"hkgt_buy_amount" gorm:"comment:港股通买入金额(元)"`
	HKGTSellAmount float64 `json:"hkgt_sell_amount" gorm:"comment:港股通卖出金额(元)"`
	HKGTNetAmount  float64 `json:"hkgt_net_amount" gorm:"comment:港股通净流入金额(元)"`
	// 总计数据
	TotalBuyAmount  float64 `json:"total_buy_amount" gorm:"comment:总买入金额(元)"`
	TotalSellAmount float64 `json:"total_sell_amount" gorm:"comment:总卖出金额(元)"`
	TotalNetAmount  float64 `json:"total_net_amount" gorm:"comment:总净流入金额(元)"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (NorthboundFundData) TableName() string {
	return "northbound_fund_data"
}

// NorthboundTopStockData 北向资金十大成交股数据
type NorthboundTopStockData struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TradeDate   time.Time `json:"trade_date" gorm:"index;not null;comment:交易日期"`
	Symbol      string    `json:"symbol" gorm:"index;size:20;not null;comment:股票代码"`
	Name        string    `json:"name" gorm:"size:50;comment:股票名称"`
	ClosePrice  float64   `json:"close_price" gorm:"comment:收盘价"`
	Change      float64   `json:"change" gorm:"comment:涨跌额"`
	Rank        int       `json:"rank" gorm:"comment:资金排名"`
	MarketType  int       `json:"market_type" gorm:"index;comment:市场类型(1:沪市 3:深市)"`
	Amount      float64   `json:"amount" gorm:"comment:成交金额(元)"`
	NetAmount   float64   `json:"net_amount" gorm:"comment:净成交金额(元)"`
	BuyAmount   float64   `json:"buy_amount" gorm:"comment:买入金额(元)"`
	SellAmount  float64   `json:"sell_amount" gorm:"comment:卖出金额(元)"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (NorthboundTopStockData) TableName() string {
	return "northbound_top_stock_data"
}

// MarginTradingData 融资融券数据
type MarginTradingData struct {
	ID               int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TradeDate        time.Time `json:"trade_date" gorm:"index;not null;comment:交易日期"`
	ExchangeID       string    `json:"exchange_id" gorm:"index;size:10;comment:交易所代码(SSE/SZSE/BSE)"`
	// 融资数据
	FinancingBalance float64 `json:"financing_balance" gorm:"comment:融资余额(元)"`
	FinancingBuy     float64 `json:"financing_buy" gorm:"comment:融资买入额(元)"`
	FinancingRepay   float64 `json:"financing_repay" gorm:"comment:融资偿还额(元)"`
	// 融券数据
	SecuritiesBalance float64 `json:"securities_balance" gorm:"comment:融券余额(元)"`
	SecuritiesSell    float64 `json:"securities_sell" gorm:"comment:融券卖出量(股)"`
	SecuritiesRepay   float64 `json:"securities_repay" gorm:"comment:融券余量(股)"`
	// 总计数据
	TotalBalance float64   `json:"total_balance" gorm:"comment:融资融券余额(元)"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (MarginTradingData) TableName() string {
	return "margin_trading_data"
}

// ETFData ETF基础数据
type ETFData struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Symbol      string    `json:"symbol" gorm:"uniqueIndex;size:20;not null;comment:ETF代码"`
	ShortName   string    `json:"short_name" gorm:"size:50;comment:ETF中文简称"`
	ExtName     string    `json:"ext_name" gorm:"size:50;comment:ETF扩位简称"`
	FullName    string    `json:"full_name" gorm:"size:100;comment:基金中文全称"`
	IndexCode   string    `json:"index_code" gorm:"index;size:20;comment:ETF基准指数代码"`
	IndexName   string    `json:"index_name" gorm:"size:100;comment:ETF基准指数中文全称"`
	SetupDate   time.Time `json:"setup_date" gorm:"comment:设立日期"`
	ListDate    time.Time `json:"list_date" gorm:"comment:上市日期"`
	ListStatus  string    `json:"list_status" gorm:"index;size:10;comment:存续状态(L上市 D退市 P待上市)"`
	Exchange    string    `json:"exchange" gorm:"index;size:10;comment:交易所(SH/SZ)"`
	ManagerName string    `json:"manager_name" gorm:"size:50;comment:基金管理人简称"`
	CustodName  string    `json:"custod_name" gorm:"size:50;comment:基金托管人名称"`
	MgtFee      float64   `json:"mgt_fee" gorm:"comment:基金管理人收取的费用"`
	ETFType     string    `json:"etf_type" gorm:"size:20;comment:基金投资通道类型"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (ETFData) TableName() string {
	return "etf_data"
}

// SentimentDataType 情绪数据类型常量
const (
	// 市场情绪指标
	SentimentTypeVIX              = "VIX"              // VIX恐慌指数
	SentimentTypeInvestor         = "INVESTOR"         // 投资者情绪指数
	SentimentTypeNewAccount       = "NEW_ACCOUNT"      // 新增开户数
	SentimentTypeNewsScore        = "NEWS_SCORE"       // 新闻情绪得分
	SentimentTypeNewsHeat         = "NEWS_HEAT"        // 新闻热度指数
	SentimentTypePolicyTrend      = "POLICY_TREND"     // 政策倾向度
	SentimentTypeSocialMedia      = "SOCIAL_MEDIA"     // 社交媒体情绪指标
	SentimentTypeSearchHeat       = "SEARCH_HEAT"      // 搜索热度指标
	SentimentTypeInstitutionView  = "INSTITUTION_VIEW" // 专业机构观点指标
)