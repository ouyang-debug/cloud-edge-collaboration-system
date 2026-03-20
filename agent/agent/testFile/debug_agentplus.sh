#!/bin/bash

# 查找agentplus进程号
echo "正在查找agentplus进程..."

# 持续等待进程号出现
while true; do
    pid=$(ps -aux | grep agentplus | grep -v grep | awk '{print $2}')
    
    if [ ! -z "$pid" ]; then
        break
    fi
    
    echo "未找到agentplus进程，等待中..."
    sleep 1
done

echo "找到agentplus进程，进程号: $pid"

# 使用dlv attach开始调试
echo "开始使用dlv调试..."
dlv attach $pid --listen=0.0.0.0:6000 --headless=true --api-version=2 --log
