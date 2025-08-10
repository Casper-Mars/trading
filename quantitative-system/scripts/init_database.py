#!/usr/bin/env python3
"""数据库初始化脚本"""

import sys
import os
from pathlib import Path

# 添加项目根目录到Python路径
project_root = Path(__file__).parent.parent
sys.path.insert(0, str(project_root))

from config.database import create_db_and_tables, engine
from models.database import (
    Position, BacktestResult, TradingPlan, 
    MarketDataCache, SystemLog, Task
)
from utils.logger import get_logger

logger = get_logger(__name__)


def init_database():
    """初始化数据库"""
    try:
        logger.info("开始初始化数据库...")
        
        # 创建所有表
        create_db_and_tables()
        
        logger.info("数据库初始化完成")
        logger.info("已创建的表：")
        logger.info("- positions (持仓表)")
        logger.info("- backtest_results (回测结果表)")
        logger.info("- trading_plans (交易方案表)")
        logger.info("- market_data_cache (市场数据缓存表)")
        logger.info("- system_logs (系统日志表)")
        logger.info("- tasks (任务表)")
        
        return True
        
    except Exception as e:
        logger.error(f"数据库初始化失败: {e}")
        return False


def check_database():
    """检查数据库连接"""
    try:
        logger.info("检查数据库连接...")
        
        # 测试数据库连接
        with engine.connect() as conn:
            result = conn.execute("SELECT 1")
            logger.info("数据库连接正常")
            return True
            
    except Exception as e:
        logger.error(f"数据库连接失败: {e}")
        return False


def show_table_info():
    """显示表信息"""
    try:
        from sqlalchemy import inspect
        
        inspector = inspect(engine)
        tables = inspector.get_table_names()
        
        logger.info(f"数据库中的表数量: {len(tables)}")
        for table in tables:
            columns = inspector.get_columns(table)
            indexes = inspector.get_indexes(table)
            logger.info(f"表 {table}: {len(columns)} 列, {len(indexes)} 索引")
            
    except Exception as e:
        logger.error(f"获取表信息失败: {e}")


if __name__ == "__main__":
    import argparse
    
    parser = argparse.ArgumentParser(description="数据库管理工具")
    parser.add_argument("--init", action="store_true", help="初始化数据库")
    parser.add_argument("--check", action="store_true", help="检查数据库连接")
    parser.add_argument("--info", action="store_true", help="显示表信息")
    
    args = parser.parse_args()
    
    if args.init:
        success = init_database()
        sys.exit(0 if success else 1)
    elif args.check:
        success = check_database()
        sys.exit(0 if success else 1)
    elif args.info:
        show_table_info()
    else:
        # 默认执行初始化
        success = init_database()
        if success:
            show_table_info()
        sys.exit(0 if success else 1)