#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Radius数据导入脚本
功能：
1. 处理20260109~20260209期间的Radius数据
2. 执行radius_to_hdfs.sh脚本导出数据
3. 记录导入结果
4. 提供详细的日志记录
5. 顺序执行，确保数据一致性
"""

import os
import subprocess
import logging
import datetime
from typing import List, Tuple

# ====================== 配置参数 ======================
# 基础配置
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
LOG_FILE = os.path.join(BASE_DIR, 'radius_import.log')

# 脚本和路径配置
RADIUS_SCRIPT = '/home/dj/scripts/fileScripts/radius_to_hdfs.sh'

# 日期配置
START_DATE = '20260109'  # 开始日期
END_DATE = '20260209'    # 结束日期

# 计算导入天数
start_date_obj = datetime.datetime.strptime(START_DATE, '%Y%m%d')
end_date_obj = datetime.datetime.strptime(END_DATE, '%Y%m%d')
DAYS_TO_IMPORT = (end_date_obj - start_date_obj).days + 1

# ====================== 日志配置 ======================
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler(LOG_FILE, encoding='utf-8'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

# ====================== 工具函数 ======================
def date_range(start_date: str, end_date: str) -> List[str]:
    """生成日期范围列表"""
    start = datetime.datetime.strptime(start_date, '%Y%m%d')
    end = datetime.datetime.strptime(end_date, '%Y%m%d')
    dates = []
    current = start
    while current <= end:
        dates.append(current.strftime('%Y%m%d'))
        current += datetime.timedelta(days=1)
    return dates

def run_command(cmd: str, cwd: str = None) -> Tuple[bool, str]:
    """执行命令并返回结果"""
    logger.info(f"执行命令: {cmd}")
    try:
        # 执行命令（兼容旧版本Python）
        result = subprocess.run(
            cmd, shell=True, cwd=cwd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            universal_newlines=True,
            timeout=3600  # 1小时超时
        )
        
        # 打印返回码和输出
        logger.debug(f"命令返回码: {result.returncode}")
        if result.stdout:
            logger.debug(f"命令 stdout: {result.stdout}")
        if result.stderr:
            logger.debug(f"命令 stderr: {result.stderr}")
        
        if result.returncode == 0:
            logger.info(f"命令执行成功: {cmd}")
            return True, result.stdout
        else:
            logger.error(f"命令执行失败: {cmd}")
            logger.error(f"返回码: {result.returncode}")
            logger.error(f"错误输出: {result.stderr}")
            return False, result.stderr
    except Exception as e:
        logger.error(f"执行命令时发生异常: {str(e)}")
        return False, str(e)

# ====================== 主函数 ======================
def main():
    """主函数"""
    logger.info("开始执行Radius数据导入流程")
    logger.info(f"导入日期范围: {START_DATE} ~ {END_DATE}")
    logger.info(f"共需处理 {DAYS_TO_IMPORT} 天的数据")
    
    # 生成日期列表
    import_dates = date_range(START_DATE, END_DATE)
    logger.info(f"生成的日期列表: {import_dates}")
    
    # 导入结果记录
    import_results = {
        'success': [],
        'failed': []
    }
    
    # 遍历处理每个日期
    for date_str in import_dates:
        logger.info(f"\n=== 处理日期: {date_str} ===")
        
        try:
            # 执行radius_to_hdfs.sh脚本
            logger.info(f"执行radius_to_hdfs.sh脚本")
            radius_cmd = f"{RADIUS_SCRIPT} {date_str}"
            success, output = run_command(radius_cmd)
            
            if success:
                import_results['success'].append(date_str)
                logger.info(f"日期 {date_str} 数据导出成功")
            else:
                import_results['failed'].append(date_str)
                logger.error(f"日期 {date_str} 数据导出失败")
                continue
            
        except Exception as e:
            logger.error(f"处理日期 {date_str} 时发生异常: {str(e)}")
            import_results['failed'].append(date_str)
            continue
    
    # 打印导入结果
    logger.info("\n=== 导入结果汇总 ===")
    logger.info(f"成功导出: {len(import_results['success'])} 天")
    logger.info(f"成功日期: {import_results['success']}")
    logger.info(f"失败导出: {len(import_results['failed'])} 天")
    logger.info(f"失败日期: {import_results['failed']}")
    logger.info(f"\n导入日期范围: {START_DATE} ~ {END_DATE}")
    
    if import_results['failed']:
        logger.warning("部分日期导出失败，请检查日志并手动处理")
    else:
        logger.info("所有日期数据导出成功")

if __name__ == "__main__":
    main()