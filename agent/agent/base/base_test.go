package base_test

import (
	"agent/base"
	"os"
	"testing"
	"time"
)

func TestNewBase(t *testing.T) {
	b := base.NewBase()
	if b == nil {
		t.Fatal("NewBase() returned nil")
	}
}

func TestStartAndStop(t *testing.T) {
	// Skip this test if MQTT broker is not available
	// We'll test this manually later
	t.Skip("Skipping MQTT dependent test")

	b := base.NewBase()

	// Start Base component
	if err := b.Start(); err != nil {
		t.Fatalf("Failed to start Base: %v", err)
	}

	// Give it some time to initialize
	time.Sleep(2 * time.Second)

	// Stop Base component
	if err := b.Stop(); err != nil {
		t.Fatalf("Failed to stop Base: %v", err)
	}
}

func TestStartPlus(t *testing.T) {
	// Skip this test if MQTT broker is not available
	// We'll test this manually later
	t.Skip("Skipping TestStartPlus test")

	b := base.NewBase()

	// Start Plus process
	if err := b.StartPlus(); err != nil {
		t.Fatalf("Failed to start Plus: %v", err)
	}

	// Give it some time to start
	time.Sleep(1 * time.Second)

	// Check Plus status
	status, err := b.GetPlusStatus()
	if err != nil {
		t.Fatalf("Failed to get Plus status: %v", err)
	}

	if status != "running" {
		t.Fatalf("Expected Plus status 'running', got '%s'", status)
	}

	// Stop Plus process
	if err := b.StopPlus(); err != nil {
		t.Fatalf("Failed to stop Plus: %v", err)
	}

	// Check Plus status again
	status, err = b.GetPlusStatus()
	if err != nil {
		t.Fatalf("Failed to get Plus status: %v", err)
	}

	if status != "stopped" {
		t.Fatalf("Expected Plus status 'stopped', got '%s'", status)
	}
}

func TestInitConfig(t *testing.T) {
	// 创建测试实例
	b := base.NewBase()

	// 创建测试license.json文件
	testLicense := `{"Broker":"127.0.0.1","Server":"http://127.0.0.1:8080","Port":1883,"ClientID":"agent_base","Username":"mqtt","Password":"mqtt","Token":"VKYNnyxsTAMea/HLD3pl7Q=="}`
	err := os.WriteFile("license.json", []byte(testLicense), 0644)
	if err != nil {
		t.Fatalf("Failed to create test license.json: %v", err)
	}
	defer os.Remove("license.json")

	// 测试从license.json读取配置
	config, err := b.InitConfig()
	if err != nil {
		t.Fatalf("Failed to init config from license.json: %v", err)
	}

	// 验证配置
	if config.Broker != "127.0.0.1" {
		t.Errorf("Expected Broker '127.0.0.1', got '%s'", config.Broker)
	}
	if config.Port != "1883" {
		t.Errorf("Expected Port 1883, got %s", config.Port)
	}
	if config.ClientID != "agent_base" {
		t.Errorf("Expected ClientID 'agent_base', got '%s'", config.ClientID)
	}
	if config.Username != "mqtt" {
		t.Errorf("Expected Username 'mqtt', got '%s'", config.Username)
	}
	if config.Password != "mqtt" {
		t.Errorf("Expected Password 'mqtt', got '%s'", config.Password)
	}
	if config.Token != "VKYNnyxsTAMea/HLD3pl7Q==" {
		t.Errorf("Expected Token 'VKYNnyxsTAMea/HLD3pl7Q==', got '%s'", config.Token)
	}

	// 测试从config.data读取配置
	// 首先确保config.data存在（由上面的测试创建）
	if _, err := os.Stat("config.data"); os.IsNotExist(err) {
		t.Fatalf("Expected config.data to exist, but it doesn't")
	}
	// defer os.Remove("config.data")

	// 删除license.json，测试从config.data读取
	// os.Remove("license.json")

	config, err = b.InitConfig()
	if err != nil {
		t.Fatalf("Failed to init config from config.data: %v", err)
	}

	// 再次验证配置
	if config.Broker != "127.0.0.1" {
		t.Errorf("Expected Broker '127.0.0.1', got '%s'", config.Broker)
	}
	if config.Port != "1883" {
		t.Errorf("Expected Port 1883, got %s", config.Port)
	}
}

func TestInitMQTTClient(t *testing.T) {
	// 跳过实际的MQTT连接测试，因为需要MQTT broker
	// t.Skip("Skipping MQTT broker dependent test")

	// 创建测试实例
	b := base.NewBase()
	// 初始化日志同步器
	// if err := b.InitLogSyncer(); err != nil {
	// 	t.Fatalf("failed to initialize LogSyncer: %v", err)
	// }
	// 测试InitMQTTClient函数
	err := b.InitMQTTClient()
	if err != nil {
		t.Fatalf("Failed to init MQTT client: %v", err)
	}

	for {
		time.Sleep(2 * time.Second)
	}

	t.Log("InitMQTTClient test completed successfully")
}

func TestReadYamlConfig(t *testing.T) {
	// 创建测试实例
	b := base.NewBase()

	// 创建测试config.yaml文件
	testConfig := `NetIf: eth0`
	// 在当前目录（agent目录）创建配置文件
	err := os.WriteFile("./config.yaml", []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config.yaml: %v", err)
	}
	defer os.Remove("./config.yaml")

	// 测试ReadYamlConfig函数
	err = b.ReadYamlConfig()
	if err != nil {
		t.Fatalf("Failed to read YAML config: %v", err)
	}

	// 验证配置是否正确读取
	if b.GetNetIf() != "eth0" {
		t.Errorf("Expected NetIf 'eth0', got '%s'", b.GetNetIf())
	}

	t.Log("ReadYamlConfig test completed successfully")
}
