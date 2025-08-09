package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"data-collection-system/pkg/config"
	"data-collection-system/pkg/logger"
)

// TushareClient Tushare API客户端
type TushareClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	rateLimiter *RateLimiter
	mu         sync.RWMutex
}

// RateLimiter 请求限流器
type RateLimiter struct {
	ticker   *time.Ticker
	ch       chan struct{}
	interval time.Duration
	mu       sync.Mutex
}

// TushareRequest Tushare API请求结构
type TushareRequest struct {
	APIName string                 `json:"api_name"`
	Token   string                 `json:"token"`
	Params  map[string]interface{} `json:"params,omitempty"`
	Fields  string                 `json:"fields,omitempty"`
}

// TushareResponse Tushare API响应结构
type TushareResponse struct {
	RequestID string          `json:"request_id"`
	Code      int             `json:"code"`
	Msg       string          `json:"msg"`
	Data      *TushareData    `json:"data"`
}

// TushareData Tushare数据结构
type TushareData struct {
	Fields []string        `json:"fields"`
	Items  [][]interface{} `json:"items"`
}

// TushareStockBasic Tushare股票基础信息
type TushareStockBasic struct {
	TSCode     string `json:"ts_code"`
	Symbol     string `json:"symbol"`
	Name       string `json:"name"`
	Area       string `json:"area"`
	Industry   string `json:"industry"`
	Market     string `json:"market"`
	Exchange   string `json:"exchange"`
	CurrType   string `json:"curr_type"`
	ListStatus string `json:"list_status"`
	ListDate   string `json:"list_date"`
	DelistDate string `json:"delist_date"`
	IsHS       string `json:"is_hs"`
}

// TushareDailyQuote Tushare日线行情数据
type TushareDailyQuote struct {
	TSCode    string  `json:"ts_code"`
	TradeDate string  `json:"trade_date"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	PreClose  float64 `json:"pre_close"`
	Change    float64 `json:"change"`
	PctChg    float64 `json:"pct_chg"`
	Vol       float64 `json:"vol"`
	Amount    float64 `json:"amount"`
}

// TushareMinuteQuote Tushare分钟级行情数据
type TushareMinuteQuote struct {
	TSCode    string  `json:"ts_code"`
	TradeTime string  `json:"trade_time"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Vol       float64 `json:"vol"`
	Amount    float64 `json:"amount"`
}

// TushareIncomeStatement Tushare利润表数据
type TushareIncomeStatement struct {
	TSCode       string  `json:"ts_code"`
	AnnDate      string  `json:"ann_date"`
	FAnnDate     string  `json:"f_ann_date"`
	EndDate      string  `json:"end_date"`
	ReportType   string  `json:"report_type"`
	CompType     string  `json:"comp_type"`
	BasicEPS     float64 `json:"basic_eps"`
	DilutedEPS   float64 `json:"diluted_eps"`
	TotalRevenue float64 `json:"total_revenue"`
	NetIncome    float64 `json:"net_income"`
}

// TushareBalanceSheet Tushare资产负债表数据
type TushareBalanceSheet struct {
	TSCode       string  `json:"ts_code"`
	AnnDate      string  `json:"ann_date"`
	FAnnDate     string  `json:"f_ann_date"`
	EndDate      string  `json:"end_date"`
	ReportType   string  `json:"report_type"`
	CompType     string  `json:"comp_type"`
	TotalShare   float64 `json:"total_share"`
	TotalAssets  float64 `json:"total_assets"`
	TotalLiab    float64 `json:"total_liab"`
	TotalHldrEqy float64 `json:"total_hldr_eqy"`
}

// TushareCashFlow Tushare现金流量表数据
type TushareCashFlow struct {
	TSCode              string  `json:"ts_code"`
	AnnDate             string  `json:"ann_date"`
	FAnnDate            string  `json:"f_ann_date"`
	EndDate             string  `json:"end_date"`
	CompType            string  `json:"comp_type"`
	ReportType          string  `json:"report_type"`
	NetProfit           float64 `json:"net_profit"`
	FinanExp            float64 `json:"finan_exp"`
	CFrSaleSg           float64 `json:"c_fr_sale_sg"`
	NIncCashCashEqu     float64 `json:"n_inc_cash_cash_equ"`
}

// TushareMacroData Tushare宏观数据
type TushareMacroData struct {
	Indicator string                 `json:"indicator"`
	Period    string                 `json:"period"`
	Values    map[string]interface{} `json:"values"`
}

// NewRateLimiter 创建请求限流器
// interval: 请求间隔时间，例如200ms表示每秒最多5个请求
func NewRateLimiter(interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		interval: interval,
		ch:       make(chan struct{}, 1),
	}
	rl.ticker = time.NewTicker(interval)
	go rl.run()
	return rl
}

// run 运行限流器
func (rl *RateLimiter) run() {
	for range rl.ticker.C {
		select {
		case rl.ch <- struct{}{}:
		default:
		}
	}
}

// Wait 等待获取请求许可
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop 停止限流器
func (rl *RateLimiter) Stop() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.ticker != nil {
		rl.ticker.Stop()
		rl.ticker = nil
	}
}

// NewTushareClient 创建Tushare客户端
func NewTushareClient(cfg config.TushareConfig) *TushareClient {
	// 创建HTTP客户端，设置超时
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// 创建请求限流器，Tushare免费用户限制为每分钟200次请求
	// 设置为每300ms一个请求，确保不超过限制
	rateLimiter := NewRateLimiter(300 * time.Millisecond)

	return &TushareClient{
		baseURL:     cfg.BaseURL,
		token:       cfg.Token,
		httpClient:  httpClient,
		rateLimiter: rateLimiter,
	}
}

// Close 关闭客户端
func (tc *TushareClient) Close() {
	if tc.rateLimiter != nil {
		tc.rateLimiter.Stop()
	}
}

// makeRequest 发起API请求
func (tc *TushareClient) makeRequest(ctx context.Context, apiName string, params map[string]interface{}, fields string) (*TushareResponse, error) {
	// 等待限流器许可
	if err := tc.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait failed: %w", err)
	}

	// 构建请求数据
	reqData := TushareRequest{
		APIName: apiName,
		Token:   tc.token,
		Params:  params,
		Fields:  fields,
	}

	// 序列化请求数据
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", tc.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DataCollector/1.0")

	// 发送请求
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var tushareResp TushareResponse
	if err := json.Unmarshal(body, &tushareResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 检查API响应码
	if tushareResp.Code != 0 {
		return nil, fmt.Errorf("Tushare API error: code=%d, msg=%s", tushareResp.Code, tushareResp.Msg)
	}

	logger.Debug(fmt.Sprintf("Tushare API request successful: %s", apiName))
	return &tushareResp, nil
}

// GetStockBasic 获取股票基础信息
func (tc *TushareClient) GetStockBasic(ctx context.Context, exchange string) (*TushareResponse, error) {
	params := make(map[string]interface{})
	if exchange != "" {
		params["exchange"] = exchange
	}

	fields := "ts_code,symbol,name,area,industry,market,exchange,curr_type,list_status,list_date,delist_date,is_hs"
	return tc.makeRequest(ctx, "stock_basic", params, fields)
}

// GetDailyQuote 获取日线行情数据
func (tc *TushareClient) GetDailyQuote(ctx context.Context, tsCode, startDate, endDate string) (*TushareResponse, error) {
	params := map[string]interface{}{
		"ts_code":    tsCode,
		"start_date": startDate,
		"end_date":   endDate,
	}

	fields := "ts_code,trade_date,open,high,low,close,pre_close,change,pct_chg,vol,amount"
	return tc.makeRequest(ctx, "daily", params, fields)
}

// GetMinuteQuote 获取分钟级行情数据
func (tc *TushareClient) GetMinuteQuote(ctx context.Context, tsCode, freq, startDate, endDate string) (*TushareResponse, error) {
	params := map[string]interface{}{
		"ts_code":    tsCode,
		"freq":       freq, // 1min, 5min, 15min, 30min, 60min
		"start_date": startDate,
		"end_date":   endDate,
	}

	fields := "ts_code,trade_time,open,high,low,close,vol,amount"
	return tc.makeRequest(ctx, "stk_mins", params, fields)
}

// GetIncomeStatement 获取利润表数据
func (tc *TushareClient) GetIncomeStatement(ctx context.Context, tsCode, period, startDate, endDate string) (*TushareResponse, error) {
	params := map[string]interface{}{
		"ts_code":    tsCode,
		"period":     period, // 报告期类型：Q1/Q2/Q3/Q4/A
		"start_date": startDate,
		"end_date":   endDate,
	}

	fields := "ts_code,ann_date,f_ann_date,end_date,report_type,comp_type,basic_eps,diluted_eps,total_revenue,revenue,int_income,prem_earned,comm_income,n_commis_income,n_oth_income,n_oth_b_income,prem_income,out_prem,une_prem_reser,reins_income,n_sec_tb_income,n_sec_uw_income,n_asset_mg_income,oth_b_income,fv_value_chg_gain,invest_income,ass_invest_income,forex_gain,total_cogs,oper_cost,int_exp,comm_exp,biz_tax_surchg,sell_exp,admin_exp,fin_exp,assets_impair_loss,prem_refund,compens_payout,reser_insur_liab,div_payt,reins_exp,oper_exp,compens_payout_refu,insur_reser_refu,reins_cost_refund,other_bus_cost,operate_profit,non_oper_income,non_oper_exp,nca_disploss,total_profit,income_tax,n_income,n_income_attr_p,minority_gain,oth_compr_income,t_compr_income,compr_inc_attr_p,compr_inc_attr_m_s,ebit,ebitda,insurance_exp,undist_profit,distable_profit"
	return tc.makeRequest(ctx, "income", params, fields)
}

// GetBalanceSheet 获取资产负债表数据
func (tc *TushareClient) GetBalanceSheet(ctx context.Context, tsCode, period, startDate, endDate string) (*TushareResponse, error) {
	params := map[string]interface{}{
		"ts_code":    tsCode,
		"period":     period,
		"start_date": startDate,
		"end_date":   endDate,
	}

	fields := "ts_code,ann_date,f_ann_date,end_date,report_type,comp_type,total_share,cap_rese,undistr_porfit,surplus_rese,special_rese,money_cap,trad_asset,notes_receiv,accounts_receiv,oth_receiv,prepayment,div_receiv,int_receiv,inventories,amor_exp,nca_within_1y,sett_rsrv,loanto_oth_bank_fi,premium_receiv,reinsur_receiv,reinsur_res_receiv,pur_resale_fa,oth_cur_assets,total_cur_assets,fa_avail_for_sale,htm_invest,lt_eqt_invest,invest_real_estate,time_deposits,oth_assets,lt_rec,fix_assets,cip,const_materials,fixed_assets_disp,produc_bio_assets,oil_and_gas_assets,intan_assets,r_and_d,goodwill,lt_amor_exp,defer_tax_assets,decr_in_disbur,oth_nca,total_nca,cash_reser_cb,depos_in_oth_bfi,prec_metals,deriv_assets,rr_reinsur_une_prem,rr_reinsur_outstd_cla,rr_reinsur_lins_liab,rr_reinsur_lthins_liab,refund_depos,ph_pledge_loans,receiv_invest,receiv_cap_contrib,insurance_cont_reserves,receiv_reinsur_res,receiv_reinsur_cont_res,oth_assets_special,total_assets,lt_borr,st_borr,cb_borr,depos_ib_deposits,loan_oth_bank,trading_fl,notes_payable,acct_payable,adv_receipts,sold_for_repur_fa,comm_payable,payroll_payable,taxes_payable,int_payable,div_payable,oth_payable,acc_exp,deferred_inc,st_bonds_payable,payable_to_reinsurer,rsrv_insur_cont,acting_trading_sec,acting_uw_sec,non_cur_liab_due_1y,oth_cur_liab,total_cur_liab,bond_payable,lt_payable,specific_payables,estimated_liab,defer_tax_liab,defer_inc_non_cur_liab,oth_ncl,total_ncl,depos_oth_bfi,deriv_liab,depos,agency_bus_liab,oth_liab,prem_receiv_adva,depos_received,ph_invest,reser_une_prem,reser_outstd_claims,reser_lins_liab,reser_lthins_liab,indept_acc_liab,pledge_borr,indem_payable,policy_div_payable,total_liab,treasury_share,ordin_risk_reser,forex_differ,invest_loss_unconf,minority_int,total_hldr_eqy_exc_min_int,total_hldr_eqy_inc_min_int,total_liab_hldr_eqy,lt_payroll_payable,oth_comp_income,oth_eqt_tools,oth_eqt_tools_p_shr,lending_funds,acc_receivable,st_fin_payable,payables"
	return tc.makeRequest(ctx, "balancesheet", params, fields)
}

// GetCashFlow 获取现金流量表数据
func (tc *TushareClient) GetCashFlow(ctx context.Context, tsCode, period, startDate, endDate string) (*TushareResponse, error) {
	params := map[string]interface{}{
		"ts_code":    tsCode,
		"period":     period,
		"start_date": startDate,
		"end_date":   endDate,
	}

	fields := "ts_code,ann_date,f_ann_date,end_date,comp_type,report_type,net_profit,finan_exp,c_fr_sale_sg,recp_tax_rends,n_depos_incr_fi,n_incr_loans_cb,n_inc_borr_oth_fi,prem_fr_orig_contr,n_incr_insured_dep,n_reinsur_prem,n_incr_disp_tfa,ifc_cash_incr,n_incr_disp_faas,n_incr_loans_oth_bank,n_cap_incr_repur,c_fr_oth_operate_a,c_inf_fr_operate_a,c_paid_goods_s,c_paid_to_for_empl,c_paid_for_taxes,n_incr_clt_loan_adv,n_incr_dep_cbob,c_pay_claims_orig_inco,pay_handling_chrg,pay_comm_insur_plcy,oth_cash_pay_oper_act,st_cash_out_act,n_cashflow_act,oth_recp_ral_inv_act,c_disp_withdrwl_invest,c_recp_return_invest,n_recp_disp_fiolta,n_recp_disp_sobu,stot_inflows_inv_act,c_pay_acq_const_fiolta,c_paid_invest,n_disp_subs_oth_biz,oth_pay_ral_inv_act,n_incr_pledge_loan,stot_out_inv_act,n_cashflow_inv_act,c_recp_borrow,proc_issue_bonds,oth_cash_recp_ral_fnc_act,stot_cash_in_fnc_act,free_cashflow,c_prepay_amt_borr,c_pay_dist_dpcp_int_exp,incl_dvd_profit_paid_sc_ms,oth_cashpay_ral_fnc_act,stot_cashout_fnc_act,n_cash_flows_fnc_act,eff_fx_flu_cash,n_incr_cash_cash_equ,c_cash_equ_beg_period,c_cash_equ_end_period,c_recp_cap_contrib,incl_cash_rec_saims,uncon_invest_loss,prov_depr_assets,depr_fa_coga_dpba,amort_intang_assets,lt_amort_deferred_exp,decr_deferred_exp,incr_acc_exp,loss_disp_fiolta,loss_scr_fa,loss_fv_chg,invest_loss,decr_def_inc_tax_assets,incr_def_inc_tax_liab,decr_inventories,decr_oper_payable,incr_oper_payable,others,im_net_cashflow_oper_act,conv_debt_into_cap,conv_copbonds_due_within_1y,fa_fnc_leases,end_bal_cash,beg_bal_cash,end_bal_cash_equ,beg_bal_cash_equ,im_n_incr_cash_equ"
	return tc.makeRequest(ctx, "cashflow", params, fields)
}

// GetMacroData 获取宏观数据
func (tc *TushareClient) GetMacroData(ctx context.Context, indicator, period string) (*TushareMacroData, error) {
	params := map[string]interface{}{
		"indicator": indicator,
		"period":    period,
	}

	resp, err := tc.makeRequest(ctx, "eco_cal", params, "indicator,period,value,unit")
	if err != nil {
		return nil, err
	}

	if len(resp.Data.Items) == 0 {
		return nil, fmt.Errorf("未找到宏观数据")
	}

	// 解析第一条数据
	item := resp.Data.Items[0]
	if len(item) < 4 {
		return nil, fmt.Errorf("数据格式错误")
	}

	values := make(map[string]interface{})
	if item[2] != nil {
		values["value"] = item[2]
	}
	if item[3] != nil {
		values["unit"] = item[3]
	}

	return &TushareMacroData{
		Indicator: toString(item[0]),
		Period:    toString(item[1]),
		Values:    values,
	}, nil
}

// toString 辅助函数，将interface{}转换为string
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// ValidateToken 验证Token是否有效
func (tc *TushareClient) ValidateToken(ctx context.Context) error {
	// 通过获取股票基础信息来验证token
	_, err := tc.GetStockBasic(ctx, "SSE")
	return err
}