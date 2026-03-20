#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
RMS数据导入脚本
功能：
1. 持续导入20260111~20260211期间的RMS数据
2. 记录导入了哪些数据
3. 导入成功后删除本地文件
"""

import os
import subprocess
import logging
import datetime
import shutil

# ====================== 配置参数 ======================
# 基础配置
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
LOG_FILE = os.path.join(BASE_DIR, 'rms_import.log')

# 脚本和路径配置
RMS_PULL_SCRIPT = '/home/dj/scripts/fileScripts/rms_pull.sh'
SPARK_HOME = '/home/dj/program/spark'
SPARK_SUBMIT_CMD = f'{SPARK_HOME}/bin/spark-submit'
SPARK_APP_JAR = '/home/dj/spark-cal.jar'
MAIN_CLASS = 'com.wxxx.liaoning.rms.RmsImport'
HDFS_BASE_PATH = 'hdfs://10.204.203.129:9000/user/dj/rms'

# 执行脚本日期范围 (20260112 ~ 20260211)
# 从20260112开始，因为20260111已经导入
START_EXEC_DATE = '20260112'
END_EXEC_DATE = '20260211'

# 计算目标数据日期范围（执行日期 - 3天）
# 例如：执行rms_pull.sh 20260111，获取的是20260108的数据
TARGET_START_DATE = (datetime.datetime.strptime(START_EXEC_DATE, '%Y%m%d') - datetime.timedelta(days=3)).strftime(
    '%Y%m%d')
TARGET_END_DATE = (datetime.datetime.strptime(END_EXEC_DATE, '%Y%m%d') - datetime.timedelta(days=3)).strftime('%Y%m%d')

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
def date_range(start_date, end_date):
    """生成日期范围列表"""
    start = datetime.datetime.strptime(start_date, '%Y%m%d')
    end = datetime.datetime.strptime(end_date, '%Y%m%d')
    dates = []
    current = start
    while current <= end:
        dates.append(current.strftime('%Y%m%d'))
        current += datetime.timedelta(days=1)
    return dates


def run_command(cmd, cwd=None):
    """执行命令并返回结果"""
    logger.info(f"执行命令: {cmd}")
    try:
        # 打印完整的命令以便调试
        logger.debug(f"完整命令: {cmd}")
        
        # 执行命令（兼容旧版本Python）
        import subprocess
        result = subprocess.run(
            cmd, shell=True, cwd=cwd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            universal_newlines=True,  # 替代text=True
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


def delete_local_files():
    """删除本地临时文件"""
    # 这里需要根据实际情况修改，删除下载的临时文件
    # 例如删除当前目录下的临时文件
    temp_files = []
    for file in os.listdir('.'):
        if file.startswith('HGU-') and file.endswith('.csv'):
            temp_files.append(file)

    for file in temp_files:
        try:
            os.remove(file)
            logger.info(f"删除本地文件成功: {file}")
        except Exception as e:
            logger.error(f"删除本地文件失败 {file}: {str(e)}")


# ====================== 主函数 ======================
def main():
    """主函数"""
    logger.info("开始执行RMS数据导入流程")
    logger.info(f"执行脚本日期范围: {START_EXEC_DATE} ~ {END_EXEC_DATE}")
    logger.info(f"目标数据日期范围: {TARGET_START_DATE} ~ {TARGET_END_DATE}")

    # 生成执行日期列表
    exec_dates = date_range(START_EXEC_DATE, END_EXEC_DATE)
    logger.info(f"共需处理 {len(exec_dates)} 天的数据")

    # 导入结果记录
    import_results = {
        'success': [],
        'failed': []
    }

    # 遍历处理每个执行日期
    for exec_date_str in exec_dates:
        # 计算目标数据日期（执行日期 - 3天）
        target_date = (datetime.datetime.strptime(exec_date_str, '%Y%m%d') - datetime.timedelta(days=3))
        target_date_str = target_date.strftime('%Y%m%d')

        logger.info(f"\n=== 处理执行日期: {exec_date_str} (对应目标数据日期: {target_date_str}) ===")

        try:
            # 1. 执行rms_pull.sh脚本下载数据
            logger.info("步骤1: 执行rms_pull.sh脚本下载数据")
            pull_cmd = f"{RMS_PULL_SCRIPT} {exec_date_str}"
            success, output = run_command(pull_cmd)
            if not success:
                logger.error(f"下载数据失败，跳过执行日期: {exec_date_str}")
                import_results['failed'].append(f"{exec_date_str} (目标: {target_date_str})")
                continue

            # 2. 执行Spark作业导入数据
            logger.info("步骤2: 执行Spark作业导入数据")
            # 构建CSV文件路径
            csv_date_str = target_date_str
            csv_file = f"HGU-V1.0.0-DAY-{csv_date_str}000000-001.csv"
            hdfs_input_path = f"{HDFS_BASE_PATH}/{exec_date_str}/{csv_file}"

            # 构建Spark命令（使用用户提供的完整配置）
            spark_cmd = f"export SPARK_HOME={SPARK_HOME} && {SPARK_SUBMIT_CMD} " \
                        f"--class {MAIN_CLASS} " \
                        f"--name 'RmsImport' " \
                        f"--master yarn " \
                        f"--deploy-mode cluster " \
                        f"--conf spark.yarn.stagingDir=hdfs://jiake29:9000/user/dj/.sparkStaging " \
                        f"--conf spark.yarn.am.memory=2048m " \
                        f"--conf spark.driver.host=jiake29 " \
                        f"--conf spark.network.timeout=300s " \
                        f"--executor-memory 4G " \
                        f"--driver-memory 4G " \
                        f"--num-executors 15 " \
                        f"--executor-cores 4 " \
                        f"--conf 'spark.executor.extraJavaOptions=-XX:+UseG1GC -XX:MaxGCPauseMillis=200 -Dfile.encoding=UTF-8' " \
                        f"--conf 'spark.driver.extraJavaOptions=-Dfile.encoding=UTF-8' " \
                        f"--conf spark.sql.shuffle.partitions=40 " \
                        f"--conf spark.memory.fraction=0.8 " \
                        f"--conf spark.memory.storageFraction=0.3 " \
                        f"--conf spark.sql.autoBroadcastJoinThreshold=-1 " \
                        f"--conf spark.sql.crossJoin.enabled=true " \
                        f"--conf spark.debug.maxToStringFields=10000 " \
                        f"--conf spark.serializer=org.apache.spark.serializer.KryoSerializer " \
                        f"--conf mapreduce.fileoutputcommitter.marksuccessfuljobs=false " \
                        f"--conf spark.driver.bindAddress=0.0.0.0 " \
                        f"--conf spark.yarn.executor.memoryOverhead=2g " \
                        f"--conf spark.task.cpus=1 " \
                        f"--conf spark.dynamicAllocation.enabled=false " \
                        f"--conf spark.scheduler.mode=FAIR " \
                        f"--conf 'spark.executorEnv.LC_ALL=zh_CN.UTF-8' " \
                        f"--conf 'spark.executorEnv.LANG=zh_CN.UTF-8' " \
                        f"--jars hdfs://10.204.203.129:9000/user/dj/spark-jars/spark-doris-connector-spark-2-25.1.0.jar,hdfs://10.204.203.129:9000/user/dj/spark-jars/mysql-connector-java-8.0.23.jar " \
                        f"{SPARK_APP_JAR} " \
                        f"{hdfs_input_path} {exec_date_str[:4]}-{exec_date_str[4:6]}-{exec_date_str[6:8]}"

            success, output = run_command(spark_cmd)
            if not success:
                logger.error(f"Spark作业执行失败，跳过执行日期: {exec_date_str}")
                import_results['failed'].append(f"{exec_date_str} (目标: {target_date_str})")  # 统一格式
                continue

            # 3. 删除本地临时文件
            logger.info("步骤3: 删除本地临时文件")
            delete_local_files()

            # 记录成功
            import_results['success'].append(f"{exec_date_str} (目标: {target_date_str})")
            logger.info(f"执行日期 {exec_date_str} 对应目标数据 {target_date_str} 导入成功")

        except Exception as e:
            logger.error(f"处理执行日期 {exec_date_str} 时发生异常: {str(e)}")
            import_results['failed'].append(f"{exec_date_str} (目标: {target_date_str})")
            continue

    # 打印导入结果
    logger.info("\n=== 导入结果汇总 ===")
    logger.info(f"成功导入: {len(import_results['success'])} 天")
    logger.info(f"成功执行日期: {import_results['success']}")
    logger.info(f"失败导入: {len(import_results['failed'])} 天")
    logger.info(f"失败执行日期: {import_results['failed']}")
    logger.info(f"\n执行脚本日期范围: {START_EXEC_DATE} ~ {END_EXEC_DATE}")
    logger.info(f"目标数据日期范围: {TARGET_START_DATE} ~ {TARGET_END_DATE}")

    if import_results['failed']:
        logger.warning("部分日期导入失败，请检查日志并手动处理")
    else:
        logger.info("所有日期数据导入成功")


if __name__ == "__main__":
    main()