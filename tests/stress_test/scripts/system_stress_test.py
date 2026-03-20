#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
系统压力测试脚本（MQTT版）
功能：
1. 模拟多个边缘节点通过MQTT向云端发送数据
2. 测试不同并发级别下的系统性能
3. 记录响应时间、成功率等指标
4. 生成详细的测试报告
"""

import os
import time
import logging
import datetime
import random
import concurrent.futures
import paho.mqtt.client as mqtt
from typing import List, Dict, Tuple

# ====================== 配置参数 ======================
# 基础配置
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
LOG_FILE = os.path.join(BASE_DIR, '../results/system_stress_test.log')

# 测试配置
TEST_DURATION = 3600  # 测试持续时间（秒）
CONCURRENT_USERS = [10, 50, 100, 200, 500]  # 不同并发用户数
TEST_INTERVAL = 60  # 每个并发级别测试持续时间（秒）

# 边缘节点配置
EDGE_NODES = 10  # 模拟边缘节点数量
DATA_SIZE = 1024  # 每次发送的数据大小（字节）
REQUEST_INTERVAL = 0.1  # 每个节点的请求间隔（秒）

# MQTT配置
MQTT_BROKER = "localhost"  # MQTT broker地址
MQTT_PORT = 1883  # MQTT端口
MQTT_TOPIC = "edge/data"  # MQTT主题
MQTT_REGISTER_TOPIC = "edge/register"  # 注册主题
MQTT_CLIENT_ID_PREFIX = "edge_test_"  # MQTT客户端ID前缀
TIMEOUT = 5  # 超时时间（秒）

# 注册状态记录
registered_agents = set()  # 已注册的agent列表

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
def generate_test_data(size: int) -> Dict:
    """生成测试数据"""
    return {
        'device_id': f'edge_{random.randint(1, EDGE_NODES)}',
        'timestamp': datetime.datetime.now().isoformat(),
        'data': 'x' * size,
        'sensor_type': random.choice(['temperature', 'humidity', 'pressure', 'light']),
        'value': random.uniform(0, 100)
    }


def register_agent(device_id: str) -> Tuple[bool, float, str]:
    """注册边缘节点"""
    start_time = time.time()
    client_id = f"{MQTT_CLIENT_ID_PREFIX}register_{random.randint(1, 10000)}"
    
    def on_connect(client, userdata, flags, rc):
        if rc == 0:
            userdata['connected'] = True
        else:
            userdata['connected'] = False
            userdata['error'] = f"Connection failed with code {rc}"
    
    def on_publish(client, userdata, mid):
        userdata['published'] = True
    
    try:
        # 创建MQTT客户端
        client = mqtt.Client(client_id=client_id)
        
        # 设置回调函数
        userdata = {'connected': False, 'published': False, 'error': None}
        client.user_data_set(userdata)
        client.on_connect = on_connect
        client.on_publish = on_publish
        
        # 连接到MQTT broker
        client.connect(MQTT_BROKER, MQTT_PORT, 60)
        client.loop_start()
        
        # 等待连接
        connect_start = time.time()
        while not userdata['connected'] and time.time() - connect_start < TIMEOUT:
            time.sleep(0.1)
        
        if not userdata['connected']:
            client.loop_stop()
            return False, time.time() - start_time, userdata.get('error', 'Connection timeout')
        
        # 发布注册消息
        import json
        register_data = {
            'device_id': device_id,
            'action': 'register',
            'timestamp': datetime.datetime.now().isoformat(),
            'device_type': 'edge',
            'version': '1.0'
        }
        message = json.dumps(register_data)
        result = client.publish(MQTT_REGISTER_TOPIC, message, qos=1)
        
        # 等待发布完成
        publish_start = time.time()
        while not userdata['published'] and time.time() - publish_start < TIMEOUT:
            time.sleep(0.1)
        
        client.loop_stop()
        response_time = time.time() - start_time
        
        if userdata['published']:
            registered_agents.add(device_id)
            return True, response_time, "Registration success"
        else:
            return False, response_time, "Registration timeout"
            
    except Exception as e:
        response_time = time.time() - start_time
        return False, response_time, f"Exception: {str(e)}"

def send_mqtt_message(data: Dict) -> Tuple[bool, float, str]:
    """发送MQTT消息并返回结果"""
    start_time = time.time()
    device_id = data.get('device_id')
    
    # 检查是否已注册
    if device_id not in registered_agents:
        # 尝试注册
        register_success, register_time, register_message = register_agent(device_id)
        if not register_success:
            return False, start_time - time.time(), f"Registration failed: {register_message}"
        logger.info(f"设备 {device_id} 注册成功")
    
    client_id = f"{MQTT_CLIENT_ID_PREFIX}{random.randint(1, 10000)}"
    
    def on_connect(client, userdata, flags, rc):
        if rc == 0:
            userdata['connected'] = True
        else:
            userdata['connected'] = False
            userdata['error'] = f"Connection failed with code {rc}"
    
    def on_publish(client, userdata, mid):
        userdata['published'] = True
    
    try:
        # 创建MQTT客户端
        client = mqtt.Client(client_id=client_id)
        
        # 设置回调函数
        userdata = {'connected': False, 'published': False, 'error': None}
        client.user_data_set(userdata)
        client.on_connect = on_connect
        client.on_publish = on_publish
        
        # 连接到MQTT broker
        client.connect(MQTT_BROKER, MQTT_PORT, 60)
        client.loop_start()
        
        # 等待连接
        connect_start = time.time()
        while not userdata['connected'] and time.time() - connect_start < TIMEOUT:
            time.sleep(0.1)
        
        if not userdata['connected']:
            client.loop_stop()
            return False, time.time() - start_time, userdata.get('error', 'Connection timeout')
        
        # 发布消息
        import json
        message = json.dumps(data)
        result = client.publish(MQTT_TOPIC, message, qos=1)
        
        # 等待发布完成
        publish_start = time.time()
        while not userdata['published'] and time.time() - publish_start < TIMEOUT:
            time.sleep(0.1)
        
        client.loop_stop()
        response_time = time.time() - start_time
        
        if userdata['published']:
            return True, response_time, "Success"
        else:
            return False, response_time, "Publish timeout"
            
    except Exception as e:
        response_time = time.time() - start_time
        return False, response_time, f"Exception: {str(e)}"


def user_task(user_id: int, duration: int) -> List[Dict]:
    """单个用户的测试任务"""
    results = []
    start_time = time.time()

    while time.time() - start_time < duration:
        data = generate_test_data(DATA_SIZE)
        success, response_time, message = send_mqtt_message(data)

        results.append({
            'user_id': user_id,
            'timestamp': datetime.datetime.now().isoformat(),
            'success': success,
            'response_time': response_time,
            'message': message
        })

        # 模拟请求间隔
        time.sleep(REQUEST_INTERVAL)

    return results


# ====================== 主函数 ======================
def main():
    """主函数"""
    logger.info("开始执行系统压力测试")
    logger.info(f"测试配置: 边缘节点数={EDGE_NODES}, 数据大小={DATA_SIZE}字节")
    logger.info(f"MQTT注册主题: {MQTT_REGISTER_TOPIC}")
    
    # 预注册所有边缘节点
    logger.info("\n=== 开始预注册边缘节点 ===")
    for i in range(1, EDGE_NODES + 1):
        device_id = f'edge_{i}'
        success, response_time, message = register_agent(device_id)
        if success:
            logger.info(f"节点 {device_id} 注册成功，耗时: {response_time:.3f}秒")
        else:
            logger.warning(f"节点 {device_id} 注册失败: {message}")
    logger.info(f"预注册完成，成功注册 {len(registered_agents)} 个节点")

    # 测试结果记录
    test_results = {
        'test_start_time': datetime.datetime.now().isoformat(),
        'test_duration': TEST_DURATION,
        'edge_nodes': EDGE_NODES,
        'data_size': DATA_SIZE,
        'concurrent_tests': []
    }

    # 对每个并发级别进行测试
    for concurrent in CONCURRENT_USERS:
        logger.info(f"\n=== 测试并发用户数: {concurrent} ===")

        # 并发测试结果
        concurrent_result = {
            'concurrent_users': concurrent,
            'test_duration': TEST_INTERVAL,
            'start_time': datetime.datetime.now().isoformat(),
            'total_requests': 0,
            'success_requests': 0,
            'failed_requests': 0,
            'total_response_time': 0,
            'avg_response_time': 0,
            'min_response_time': float('inf'),
            'max_response_time': 0,
            'response_times': [],
            'detailed_results': []
        }

        # 执行并发测试
        with concurrent.futures.ThreadPoolExecutor(max_workers=concurrent) as executor:
            futures = []
            for user_id in range(concurrent):
                future = executor.submit(user_task, user_id, TEST_INTERVAL)
                futures.append(future)

            # 收集结果
            for future in concurrent.futures.as_completed(futures):
                try:
                    user_results = future.result()
                    for result in user_results:
                        concurrent_result['total_requests'] += 1
                        concurrent_result['detailed_results'].append(result)

                        if result['success']:
                            concurrent_result['success_requests'] += 1
                        else:
                            concurrent_result['failed_requests'] += 1

                        response_time = result['response_time']
                        concurrent_result['total_response_time'] += response_time
                        concurrent_result['response_times'].append(response_time)

                        if response_time < concurrent_result['min_response_time']:
                            concurrent_result['min_response_time'] = response_time
                        if response_time > concurrent_result['max_response_time']:
                            concurrent_result['max_response_time'] = response_time

                except Exception as e:
                    logger.error(f"处理用户任务时发生异常: {str(e)}")
                    concurrent_result['failed_requests'] += 1

        # 计算统计数据
        if concurrent_result['total_requests'] > 0:
            concurrent_result['avg_response_time'] = concurrent_result['total_response_time'] / concurrent_result[
                'total_requests']
        else:
            concurrent_result['min_response_time'] = 0

        concurrent_result['end_time'] = datetime.datetime.now().isoformat()
        test_results['concurrent_tests'].append(concurrent_result)

        # 打印并发测试结果
        logger.info(f"并发用户数: {concurrent}")
        logger.info(f"总消息数: {concurrent_result['total_requests']}")
        logger.info(f"成功消息数: {concurrent_result['success_requests']}")
        logger.info(f"失败消息数: {concurrent_result['failed_requests']}")
        logger.info(f"成功率: {concurrent_result['success_requests'] / concurrent_result['total_requests'] * 100:.2f}%")
        logger.info(f"平均响应时间: {concurrent_result['avg_response_time']:.3f}秒")
        logger.info(f"最小响应时间: {concurrent_result['min_response_time']:.3f}秒")
        logger.info(f"最大响应时间: {concurrent_result['max_response_time']:.3f}秒")
        logger.info(f"吞吐量: {concurrent_result['total_requests'] / TEST_INTERVAL:.2f} msg/s")

    # 计算总体统计数据
    total_requests = sum(test['total_requests'] for test in test_results['concurrent_tests'])
    total_success = sum(test['success_requests'] for test in test_results['concurrent_tests'])
    total_response_time = sum(test['total_response_time'] for test in test_results['concurrent_tests'])

    test_results['total_messages'] = total_requests
    test_results['total_success'] = total_success
    test_results['total_failed'] = total_requests - total_success
    test_results['overall_success_rate'] = total_success / total_requests * 100 if total_requests > 0 else 0
    test_results['average_response_time'] = total_response_time / total_requests if total_requests > 0 else 0
    test_results['mqtt_broker'] = f"{MQTT_BROKER}:{MQTT_PORT}"
    test_results['mqtt_topic'] = MQTT_TOPIC
    test_results['test_end_time'] = datetime.datetime.now().isoformat()

    # 打印总体测试结果
    logger.info("\n=== 总体测试结果 ===")
    logger.info(f"总测试时间: {TEST_DURATION}秒")
    logger.info(f"总消息数: {total_requests}")
    logger.info(f"总成功消息数: {total_success}")
    logger.info(f"总失败消息数: {total_requests - total_success}")
    logger.info(f"总体成功率: {test_results['overall_success_rate']:.2f}%")
    logger.info(f"总体平均响应时间: {test_results['average_response_time']:.3f}秒")
    logger.info(f"MQTT Broker: {MQTT_BROKER}:{MQTT_PORT}")
    logger.info(f"MQTT Topic: {MQTT_TOPIC}")

    # 保存测试结果
    result_file = os.path.join(BASE_DIR,
                               f'../results/system_stress_test_{datetime.datetime.now().strftime("%Y%m%d_%H%M%S")}.json')
    import json
    with open(result_file, 'w', encoding='utf-8') as f:
        json.dump(test_results, f, ensure_ascii=False, indent=2)

    logger.info(f"详细测试结果已保存到: {result_file}")


if __name__ == "__main__":
    main()