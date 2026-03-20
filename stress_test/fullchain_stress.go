// stress_test/fullchain_stress.go
package stresstest

import (
	"fmt"
	"time"
)

type FullChainStressTest struct {
	mqttBroker   string
	grpcAddr     string
	registerURL  string
	agentCount   int
	testDuration time.Duration
}

func (f *FullChainStressTest) Run() error {
	// 1. 注册阶段
	fmt.Println("=== 阶段 1: 设备注册压力测试 ===")
	registerTest := &RegisterStressTest{
		registerURL:     f.registerURL,
		concurrentCount: 50,
		totalRequests:   f.agentCount,
	}
	registerTest.Run()

	// 2. 连接阶段
	fmt.Println("=== 阶段 2: MQTT 连接压力测试 ===")
	mqttTest := &MQTTStressTest{
		brokerURL:   f.mqttBroker,
		clientCount: f.agentCount,
		messageRate: 1,
		duration:    f.testDuration,
	}
	mqttTest.Run()

	// 3. 任务分发阶段
	fmt.Println("=== 阶段 3: 任务分发压力测试 ===")
	taskTest := &TaskDispatchStressTest{
		mqttBroker:         f.mqttBroker,
		taskCount:          1000,
		concurrentDispatch: 100,
	}
	taskTest.Run()

	// 4. gRPC 通信阶段
	fmt.Println("=== 阶段 4: gRPC 通信压力测试 ===")
	grpcTest := &GRPCStressTest{
		serverAddr:      f.grpcAddr,
		concurrentCount: 50,
		requestRate:     10,
		duration:        f.testDuration,
	}
	grpcTest.Run()

	return nil
}
