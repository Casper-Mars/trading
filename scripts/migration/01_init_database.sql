-- 交易数据采集系统数据库初始化脚本
-- 创建时间: 2024
-- 描述: 安全的数据库表结构初始化
-- 重要说明: 
--   1. MySQL Docker容器的初始化机制:
--      - 仅在容器首次启动且 /var/lib/mysql 目录为空时执行
--      - 后续启动会跳过 /docker-entrypoint-initdb.d 中的脚本
--      - 数据持久化在Docker数据卷中，重启容器不会丢失数据
--   2. 本脚本使用 CREATE TABLE IF NOT EXISTS 确保安全性
--   3. 即使手动执行也不会破坏现有数据

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 记录初始化开始
SELECT '开始初始化交易数据采集系统数据库...' as message;

-- ----------------------------
-- 股票基础信息表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `stocks` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `ts_code` varchar(20) NOT NULL COMMENT 'TS代码',
  `symbol` varchar(20) NOT NULL COMMENT '股票代码',
  `name` varchar(100) NOT NULL COMMENT '股票名称',
  `area` varchar(50) DEFAULT NULL COMMENT '地域',
  `industry` varchar(100) DEFAULT NULL COMMENT '所属行业',
  `market` varchar(20) DEFAULT NULL COMMENT '市场类型',
  `exchange` varchar(20) DEFAULT NULL COMMENT '交易所代码',
  `curr_type` varchar(10) DEFAULT NULL COMMENT '交易货币',
  `list_status` varchar(10) DEFAULT NULL COMMENT '上市状态',
  `list_date` date DEFAULT NULL COMMENT '上市日期',
  `delist_date` date DEFAULT NULL COMMENT '退市日期',
  `is_hs` varchar(10) DEFAULT NULL COMMENT '是否沪深港通标的',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ts_code` (`ts_code`),
  KEY `idx_symbol` (`symbol`),
  KEY `idx_industry` (`industry`),
  KEY `idx_market` (`market`),
  KEY `idx_list_status` (`list_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='股票基础信息表';

-- ----------------------------
-- 股票日线数据表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `stock_daily` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `ts_code` varchar(20) NOT NULL COMMENT 'TS代码',
  `trade_date` date NOT NULL COMMENT '交易日期',
  `open` decimal(10,3) DEFAULT NULL COMMENT '开盘价',
  `high` decimal(10,3) DEFAULT NULL COMMENT '最高价',
  `low` decimal(10,3) DEFAULT NULL COMMENT '最低价',
  `close` decimal(10,3) DEFAULT NULL COMMENT '收盘价',
  `pre_close` decimal(10,3) DEFAULT NULL COMMENT '昨收价',
  `change` decimal(10,3) DEFAULT NULL COMMENT '涨跌额',
  `pct_chg` decimal(8,4) DEFAULT NULL COMMENT '涨跌幅',
  `vol` decimal(20,2) DEFAULT NULL COMMENT '成交量(手)',
  `amount` decimal(20,3) DEFAULT NULL COMMENT '成交额(千元)',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ts_code_trade_date` (`ts_code`,`trade_date`),
  KEY `idx_trade_date` (`trade_date`),
  KEY `idx_ts_code` (`ts_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='股票日线数据表';

-- ----------------------------
-- 市场数据表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `market_data` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `trade_date` date NOT NULL COMMENT '交易日期',
  `market_type` varchar(20) NOT NULL COMMENT '市场类型(SH/SZ/CYB等)',
  `total_mv` decimal(20,2) DEFAULT NULL COMMENT '总市值(亿元)',
  `float_mv` decimal(20,2) DEFAULT NULL COMMENT '流通市值(亿元)',
  `total_share` decimal(20,2) DEFAULT NULL COMMENT '总股本(亿股)',
  `float_share` decimal(20,2) DEFAULT NULL COMMENT '流通股本(亿股)',
  `free_share` decimal(20,2) DEFAULT NULL COMMENT '自由流通股本(亿股)',
  `turnover_rate` decimal(8,4) DEFAULT NULL COMMENT '换手率',
  `turnover_rate_f` decimal(8,4) DEFAULT NULL COMMENT '换手率(自由流通股)',
  `pe` decimal(8,4) DEFAULT NULL COMMENT '市盈率',
  `pe_ttm` decimal(8,4) DEFAULT NULL COMMENT '市盈率TTM',
  `pb` decimal(8,4) DEFAULT NULL COMMENT '市净率',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_trade_date_market_type` (`trade_date`,`market_type`),
  KEY `idx_trade_date` (`trade_date`),
  KEY `idx_market_type` (`market_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='市场数据表';

-- ----------------------------
-- 财务数据表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `financial_data` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `ts_code` varchar(20) NOT NULL COMMENT 'TS代码',
  `ann_date` date DEFAULT NULL COMMENT '公告日期',
  `f_ann_date` date DEFAULT NULL COMMENT '实际公告日期',
  `end_date` date NOT NULL COMMENT '报告期',
  `report_type` varchar(20) DEFAULT NULL COMMENT '报告类型',
  `comp_type` varchar(20) DEFAULT NULL COMMENT '公司类型',
  `basic_eps` decimal(10,4) DEFAULT NULL COMMENT '基本每股收益',
  `diluted_eps` decimal(10,4) DEFAULT NULL COMMENT '稀释每股收益',
  `total_revenue` decimal(20,2) DEFAULT NULL COMMENT '营业总收入',
  `revenue` decimal(20,2) DEFAULT NULL COMMENT '营业收入',
  `n_income` decimal(20,2) DEFAULT NULL COMMENT '净利润(含少数股东损益)',
  `n_income_attr_p` decimal(20,2) DEFAULT NULL COMMENT '净利润(不含少数股东损益)',
  `total_profit` decimal(20,2) DEFAULT NULL COMMENT '利润总额',
  `operate_profit` decimal(20,2) DEFAULT NULL COMMENT '营业利润',
  `ebit` decimal(20,2) DEFAULT NULL COMMENT '息税前利润',
  `ebitda` decimal(20,2) DEFAULT NULL COMMENT '息税折旧摊销前利润',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ts_code_end_date` (`ts_code`,`end_date`),
  KEY `idx_ts_code` (`ts_code`),
  KEY `idx_end_date` (`end_date`),
  KEY `idx_ann_date` (`ann_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='财务数据表';

-- ----------------------------
-- 新闻数据表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `news_data` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `title` varchar(500) NOT NULL COMMENT '新闻标题',
  `content` longtext COMMENT '新闻内容',
  `summary` text COMMENT '新闻摘要',
  `source` varchar(100) DEFAULT NULL COMMENT '新闻来源',
  `author` varchar(100) DEFAULT NULL COMMENT '作者',
  `publish_time` datetime DEFAULT NULL COMMENT '发布时间',
  `url` varchar(1000) DEFAULT NULL COMMENT '原文链接',
  `category` varchar(50) DEFAULT NULL COMMENT '新闻分类',
  `tags` varchar(500) DEFAULT NULL COMMENT '标签(JSON格式)',
  `sentiment_score` decimal(5,4) DEFAULT NULL COMMENT '情感分数(-1到1)',
  `sentiment_label` varchar(20) DEFAULT NULL COMMENT '情感标签(positive/negative/neutral)',
  `keywords` text COMMENT '关键词(JSON格式)',
  `related_stocks` varchar(1000) DEFAULT NULL COMMENT '相关股票代码(JSON格式)',
  `importance_score` decimal(5,4) DEFAULT NULL COMMENT '重要性分数(0到1)',
  `is_processed` tinyint(1) DEFAULT '0' COMMENT '是否已处理',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_publish_time` (`publish_time`),
  KEY `idx_source` (`source`),
  KEY `idx_category` (`category`),
  KEY `idx_sentiment_label` (`sentiment_label`),
  KEY `idx_is_processed` (`is_processed`),
  KEY `idx_created_at` (`created_at`),
  FULLTEXT KEY `ft_title_content` (`title`,`content`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='新闻数据表';

-- ----------------------------
-- 数据任务表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `data_tasks` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `task_name` varchar(100) NOT NULL COMMENT '任务名称',
  `task_type` varchar(50) NOT NULL COMMENT '任务类型',
  `task_config` json DEFAULT NULL COMMENT '任务配置(JSON格式)',
  `status` varchar(20) NOT NULL DEFAULT 'pending' COMMENT '任务状态',
  `priority` int(11) DEFAULT '0' COMMENT '优先级',
  `retry_count` int(11) DEFAULT '0' COMMENT '重试次数',
  `max_retry` int(11) DEFAULT '3' COMMENT '最大重试次数',
  `start_time` datetime DEFAULT NULL COMMENT '开始时间',
  `end_time` datetime DEFAULT NULL COMMENT '结束时间',
  `duration` int(11) DEFAULT NULL COMMENT '执行时长(秒)',
  `error_message` text COMMENT '错误信息',
  `result` json DEFAULT NULL COMMENT '执行结果(JSON格式)',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_task_type` (`task_type`),
  KEY `idx_status` (`status`),
  KEY `idx_priority` (`priority`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_start_time` (`start_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据任务表';

-- ----------------------------
-- 宏观经济数据表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `macro_data` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `indicator_code` varchar(50) NOT NULL COMMENT '指标代码',
  `indicator_name` varchar(200) NOT NULL COMMENT '指标名称',
  `period` varchar(20) NOT NULL COMMENT '数据周期',
  `period_date` date NOT NULL COMMENT '数据日期',
  `value` decimal(20,6) DEFAULT NULL COMMENT '指标值',
  `unit` varchar(50) DEFAULT NULL COMMENT '单位',
  `source` varchar(100) DEFAULT NULL COMMENT '数据来源',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_indicator_period_date` (`indicator_code`,`period_date`),
  KEY `idx_indicator_code` (`indicator_code`),
  KEY `idx_period_date` (`period_date`),
  KEY `idx_period` (`period`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宏观经济数据表';

-- ----------------------------
-- 创建索引（如果不存在）
-- ----------------------------
-- 注意：MySQL会自动忽略已存在的索引
CREATE INDEX IF NOT EXISTS idx_stock_daily_trade_date_ts_code ON stock_daily(trade_date, ts_code);
CREATE INDEX IF NOT EXISTS idx_news_data_publish_time_category ON news_data(publish_time, category);
CREATE INDEX IF NOT EXISTS idx_financial_data_end_date_ts_code ON financial_data(end_date, ts_code);

-- ----------------------------
-- 创建视图（如果不存在）
-- ----------------------------
-- 检查视图是否存在，不存在则创建
SET @view_exists = (SELECT COUNT(*) FROM information_schema.views 
                    WHERE table_schema = 'trading_data' AND table_name = 'v_latest_stock_data');

SET @sql = IF(@view_exists = 0, 
    'CREATE VIEW v_latest_stock_data AS
     SELECT 
         s.ts_code,
         s.symbol,
         s.name,
         s.industry,
         sd.trade_date,
         sd.close,
         sd.pct_chg,
         sd.vol,
         sd.amount
     FROM stocks s
     LEFT JOIN stock_daily sd ON s.ts_code = sd.ts_code
     WHERE sd.trade_date = (
         SELECT MAX(trade_date) 
         FROM stock_daily sd2 
         WHERE sd2.ts_code = s.ts_code
     );',
    'SELECT "视图 v_latest_stock_data 已存在，跳过创建" as message;');

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- ----------------------------
-- 插入测试数据（仅在表为空时插入）
-- ----------------------------
SET @stock_count = (SELECT COUNT(*) FROM stocks);

-- 只有在stocks表为空时才插入测试数据
SET @sql = IF(@stock_count = 0, 
    'INSERT INTO stocks (ts_code, symbol, name, area, industry, market, exchange, curr_type, list_status, list_date) VALUES
     ("000001.SZ", "000001", "平安银行", "深圳", "银行", "主板", "SZSE", "CNY", "L", "1991-04-03"),
     ("000002.SZ", "000002", "万科A", "深圳", "房地产开发", "主板", "SZSE", "CNY", "L", "1991-01-29"),
     ("600000.SH", "600000", "浦发银行", "上海", "银行", "主板", "SSE", "CNY", "L", "1999-11-10"),
     ("600036.SH", "600036", "招商银行", "深圳", "银行", "主板", "SSE", "CNY", "L", "2002-04-09");',
    'SELECT "stocks表已有数据，跳过测试数据插入" as message;');

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET FOREIGN_KEY_CHECKS = 1;

-- 记录初始化完成
SELECT '数据库初始化完成！' as message;
SELECT CONCAT('当前数据库中共有 ', COUNT(*), ' 张表') as table_count 
FROM information_schema.tables 
WHERE table_schema = 'trading_data';

COMMIT;