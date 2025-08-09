-- 创建数据库表结构
-- 执行顺序：按照表的依赖关系创建

-- 1. 股票基础信息表
CREATE TABLE IF NOT EXISTS `stocks` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `symbol` varchar(20) NOT NULL COMMENT '股票代码',
    `name` varchar(100) NOT NULL COMMENT '股票名称',
    `market` varchar(10) NOT NULL COMMENT '市场类型(SH/SZ)',
    `industry` varchar(50) DEFAULT NULL COMMENT '所属行业',
    `list_date` date DEFAULT NULL COMMENT '上市日期',
    `is_active` tinyint(1) DEFAULT '1' COMMENT '是否有效',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_symbol` (`symbol`),
    KEY `idx_market` (`market`),
    KEY `idx_industry` (`industry`),
    KEY `idx_is_active` (`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='股票基础信息表';

-- 2. 市场行情数据表
CREATE TABLE IF NOT EXISTS `market_data` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `symbol` varchar(20) NOT NULL COMMENT '股票代码',
    `trade_date` date NOT NULL COMMENT '交易日期',
    `open_price` decimal(10,3) DEFAULT NULL COMMENT '开盘价',
    `high_price` decimal(10,3) DEFAULT NULL COMMENT '最高价',
    `low_price` decimal(10,3) DEFAULT NULL COMMENT '最低价',
    `close_price` decimal(10,3) DEFAULT NULL COMMENT '收盘价',
    `pre_close` decimal(10,3) DEFAULT NULL COMMENT '前收盘价',
    `change_amount` decimal(10,3) DEFAULT NULL COMMENT '涨跌额',
    `change_percent` decimal(8,4) DEFAULT NULL COMMENT '涨跌幅(%)',
    `volume` bigint DEFAULT NULL COMMENT '成交量(手)',
    `amount` decimal(15,2) DEFAULT NULL COMMENT '成交额(元)',
    `turnover_rate` decimal(8,4) DEFAULT NULL COMMENT '换手率(%)',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_symbol_date` (`symbol`, `trade_date`),
    KEY `idx_trade_date` (`trade_date`),
    KEY `idx_symbol` (`symbol`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='市场行情数据表';

-- 3. 财务数据表
CREATE TABLE IF NOT EXISTS `financial_data` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `symbol` varchar(20) NOT NULL COMMENT '股票代码',
    `report_date` date NOT NULL COMMENT '报告期',
    `report_type` varchar(10) NOT NULL COMMENT '报告类型(Q1/Q2/Q3/Q4)',
    `total_revenue` decimal(15,2) DEFAULT NULL COMMENT '营业总收入',
    `net_profit` decimal(15,2) DEFAULT NULL COMMENT '净利润',
    `total_assets` decimal(15,2) DEFAULT NULL COMMENT '总资产',
    `total_equity` decimal(15,2) DEFAULT NULL COMMENT '股东权益',
    `eps` decimal(8,4) DEFAULT NULL COMMENT '每股收益',
    `roe` decimal(8,4) DEFAULT NULL COMMENT '净资产收益率(%)',
    `roa` decimal(8,4) DEFAULT NULL COMMENT '总资产收益率(%)',
    `gross_margin` decimal(8,4) DEFAULT NULL COMMENT '毛利率(%)',
    `net_margin` decimal(8,4) DEFAULT NULL COMMENT '净利率(%)',
    `debt_ratio` decimal(8,4) DEFAULT NULL COMMENT '资产负债率(%)',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_symbol_report` (`symbol`, `report_date`, `report_type`),
    KEY `idx_report_date` (`report_date`),
    KEY `idx_symbol` (`symbol`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='财务数据表';

-- 4. 宏观经济数据表
CREATE TABLE IF NOT EXISTS `macro_data` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `indicator_code` varchar(50) NOT NULL COMMENT '指标代码',
    `indicator_name` varchar(100) NOT NULL COMMENT '指标名称',
    `period` date NOT NULL COMMENT '统计周期',
    `value` decimal(15,4) DEFAULT NULL COMMENT '指标值',
    `unit` varchar(20) DEFAULT NULL COMMENT '单位',
    `frequency` varchar(10) DEFAULT NULL COMMENT '频率(日/周/月/季/年)',
    `source` varchar(50) DEFAULT NULL COMMENT '数据来源',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_indicator_period` (`indicator_code`, `period`),
    KEY `idx_period` (`period`),
    KEY `idx_indicator_code` (`indicator_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宏观经济数据表';

-- 5. 新闻数据表
CREATE TABLE IF NOT EXISTS `news_data` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `title` varchar(500) NOT NULL COMMENT '新闻标题',
    `content` text COMMENT '新闻内容',
    `summary` varchar(1000) DEFAULT NULL COMMENT '新闻摘要',
    `source` varchar(100) DEFAULT NULL COMMENT '新闻来源',
    `author` varchar(100) DEFAULT NULL COMMENT '作者',
    `publish_time` datetime DEFAULT NULL COMMENT '发布时间',
    `url` varchar(500) DEFAULT NULL COMMENT '原文链接',
    `category` varchar(50) DEFAULT NULL COMMENT '新闻分类',
    `tags` json DEFAULT NULL COMMENT '标签列表',
    `related_stocks` json DEFAULT NULL COMMENT '相关股票代码',
    `sentiment_score` decimal(5,4) DEFAULT NULL COMMENT '情感得分(-1到1)',
    `sentiment_label` varchar(20) DEFAULT NULL COMMENT '情感标签(positive/negative/neutral)',
    `importance_score` decimal(5,4) DEFAULT NULL COMMENT '重要性得分(0到1)',
    `entities` json DEFAULT NULL COMMENT '实体识别结果',
    `keywords` json DEFAULT NULL COMMENT '关键词',
    `is_processed` tinyint(1) DEFAULT '0' COMMENT '是否已处理',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_publish_time` (`publish_time`),
    KEY `idx_source` (`source`),
    KEY `idx_category` (`category`),
    KEY `idx_sentiment_label` (`sentiment_label`),
    KEY `idx_is_processed` (`is_processed`),
    FULLTEXT KEY `ft_title_content` (`title`, `content`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='新闻数据表';

-- 6. 数据任务表
CREATE TABLE IF NOT EXISTS `data_tasks` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `task_name` varchar(100) NOT NULL COMMENT '任务名称',
    `task_type` varchar(50) NOT NULL COMMENT '任务类型',
    `data_source` varchar(100) NOT NULL COMMENT '数据源',
    `target_table` varchar(100) DEFAULT NULL COMMENT '目标表',
    `cron_expression` varchar(100) DEFAULT NULL COMMENT 'Cron表达式',
    `status` varchar(20) DEFAULT 'pending' COMMENT '任务状态',
    `last_run_time` datetime DEFAULT NULL COMMENT '上次运行时间',
    `next_run_time` datetime DEFAULT NULL COMMENT '下次运行时间',
    `run_count` int DEFAULT '0' COMMENT '运行次数',
    `success_count` int DEFAULT '0' COMMENT '成功次数',
    `failure_count` int DEFAULT '0' COMMENT '失败次数',
    `last_error` text COMMENT '最后错误信息',
    `config` json DEFAULT NULL COMMENT '任务配置',
    `is_enabled` tinyint(1) DEFAULT '1' COMMENT '是否启用',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_task_name` (`task_name`),
    KEY `idx_task_type` (`task_type`),
    KEY `idx_status` (`status`),
    KEY `idx_is_enabled` (`is_enabled`),
    KEY `idx_next_run_time` (`next_run_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据任务表';

-- 创建复合索引以优化查询性能
-- 新闻数据表的复合索引
ALTER TABLE `news_data` ADD INDEX `idx_publish_sentiment` (`publish_time`, `sentiment_label`);
ALTER TABLE `news_data` ADD INDEX `idx_category_time` (`category`, `publish_time`);

-- 市场数据表的复合索引
ALTER TABLE `market_data` ADD INDEX `idx_symbol_date_desc` (`symbol`, `trade_date` DESC);

-- 财务数据表的复合索引
ALTER TABLE `financial_data` ADD INDEX `idx_symbol_date_desc` (`symbol`, `report_date` DESC);

-- 宏观数据表的复合索引
ALTER TABLE `macro_data` ADD INDEX `idx_indicator_period_desc` (`indicator_code`, `period` DESC);