package mqttconnector

import (
	"fmt"
	"testing"
	"time"
)

// TestNewMQTTConnector 测试创建新的MQTT连接器
func TestNewMQTTConnector(t *testing.T) {
	// 创建包含所有默认重连配置的测试配置
	config := MQTTConfig{
		Broker:               "127.0.0.1",
		Port:                 1883,
		ClientID:             "test_client",
		Username:             "mqtt",
		Password:             "mqtt",
		QoS:                  1,
		AutoReconnect:        true,
		ReconnectInterval:    1 * time.Second,
		MaxReconnectInterval: 30 * time.Second,
	}

	connector := NewMQTTConnector(config)
	if connector == nil {
		t.Error("Expected connector to be created, got nil")
	}

	// 比较配置是否完全匹配
	if connector.config != config {
		t.Error("Expected connector config to match input config")
	}

	// 测试默认值设置
	minimalConfig := MQTTConfig{
		Broker:   "127.0.0.1",
		Port:     1883,
		ClientID: "test_client",
		Username: "mqtt",
		Password: "mqtt",
		QoS:      1,
		// 不设置重连配置，应该使用默认值
	}

	minimalConnector := NewMQTTConnector(minimalConfig)
	if !minimalConnector.config.AutoReconnect {
		t.Error("Expected AutoReconnect to be true by default")
	}
	if minimalConnector.config.ReconnectInterval != 1*time.Second {
		t.Error("Expected ReconnectInterval to be 1s by default")
	}
	if minimalConnector.config.MaxReconnectInterval != 30*time.Second {
		t.Error("Expected MaxReconnectInterval to be 30s by default")
	}
}

// TestConnect 测试连接到MQTT代理
func TestConnect(t *testing.T) {
	config := MQTTConfig{
		Broker:   "127.0.0.1",
		Port:     1883,
		ClientID: "test_connect",
		Username: "mqtt",
		Password: "mqtt",
		QoS:      1,
	}

	connector := NewMQTTConnector(config)
	if connector == nil {
		t.Error("Expected connector to be created, got nil")
		return
	}

	// 尝试连接
	err := connector.Connect()
	if err != nil {
		// 连接失败可能是因为没有MQTT代理运行，这里只记录日志不失败测试
		t.Logf("Connection failed (expected if no MQTT broker running): %v", err)
		return
	}
	defer connector.Disconnect()

	// 检查连接状态
	if !connector.IsConnected() {
		t.Error("Expected connector to be connected")
	}
}

// TestPublish 测试发布消息
func TestPublish(t *testing.T) {
	config := MQTTConfig{
		Broker:   "127.0.0.1",
		Port:     1883,
		ClientID: "test_publish",
		Username: "mqtt",
		Password: "mqtt",
		QoS:      1,
	}

	connector := NewMQTTConnector(config)
	err := connector.Connect()
	if err != nil {
		t.Logf("Connection failed (expected if no MQTT broker running): %v", err)
		return
	}
	defer connector.Disconnect()

	if !connector.IsConnected() {
		t.Log("Not connected, skipping publish test")
		return
	}

	// 测试发布消息
	topic := "test/publish"
	message := "test publish message"

	err = connector.Publish(topic, message)
	if err != nil {
		t.Errorf("Failed to publish message: %v", err)
	} else {
		t.Logf("Successfully published message to topic %s", topic)
	}
}

// TestSubscribe 测试订阅主题
func TestSubscribe(t *testing.T) {
	config := MQTTConfig{
		Broker:   "127.0.0.1",
		Port:     1883,
		ClientID: "test_subscribe",
		Username: "mqtt",
		Password: "mqtt",
		QoS:      1,
	}

	connector := NewMQTTConnector(config)
	err := connector.Connect()
	if err != nil {
		t.Logf("Connection failed (expected if no MQTT broker running): %v", err)
		return
	}
	defer connector.Disconnect()

	if !connector.IsConnected() {
		t.Log("Not connected, skipping subscribe test")
		return
	}

	// 测试订阅主题
	topic := "test/subscribe"
	messageReceived := false
	receivedMessage := ""

	// 创建一个局部变量来保存测试实例，以便在闭包中使用
	testInstance := t

	// 自定义消息处理函数
	handler := func(t string, payload []byte) {
		messageReceived = true
		receivedMessage = string(payload)
		testInstance.Logf("Received message: %s from topic: %s", receivedMessage, t)
	}

	err = connector.Subscribe(topic, handler)
	if err != nil {
		t.Errorf("Failed to subscribe to topic %s: %v", topic, err)
		return
	}

	// 发布消息到同一主题
	testMessage := "test subscribe message"
	err = connector.Publish(topic, testMessage)
	if err != nil {
		t.Errorf("Failed to publish message for subscribe test: %v", err)
		return
	}

	// 等待消息接收
	time.Sleep(1 * time.Second)

	if !messageReceived {
		t.Error("Expected to receive message but didn't")
	} else if receivedMessage != testMessage {
		t.Errorf("Expected message '%s', got '%s'", testMessage, receivedMessage)
	} else {
		t.Log("Successfully subscribed to topic and received message")
	}

	// 测试取消订阅
	err = connector.Unsubscribe(topic)
	if err != nil {
		t.Errorf("Failed to unsubscribe from topic %s: %v", topic, err)
	} else {
		t.Logf("Successfully unsubscribed from topic %s", topic)
	}
}

// TestSubscribe 测试订阅主题
func TestSubscribeFor(t *testing.T) {
	config := MQTTConfig{
		Broker:   "127.0.0.1",
		Port:     1883,
		ClientID: "test_subscribe",
		Username: "mqtt",
		Password: "mqtt",
		QoS:      1,
	}

	connector := NewMQTTConnector(config)

	// 测试订阅主题
	topic := "test/subscribe"
	//messageReceived := false
	receivedMessage := ""

	// 创建一个局部变量来保存测试实例，以便在闭包中使用
	testInstance := t

	// 自定义消息处理函数
	handler := func(t string, payload []byte) {
		//messageReceived = true
		receivedMessage = string(payload)
		testInstance.Logf("Received message: %s from topic: %s", receivedMessage, t)
		fmt.Printf("Received message: %s from topic: %s\n", receivedMessage, t)
	}

	for {
		err := connector.Connect()
		if err != nil {
			t.Logf("Connection failed (expected if no MQTT broker running): %v", err)
			time.Sleep(1 * time.Second)
		} else {
			for {
				err = connector.Subscribe(topic, handler)
				if err != nil {
					t.Errorf("Failed to subscribe to topic %s: %v", topic, err)
					time.Sleep(1 * time.Second)
				} else {
					break
				}
			}
			break
		}
	}

	defer connector.Disconnect()

	// if !connector.IsConnected() {
	// 	t.Log("Not connected, skipping subscribe test")
	// 	return
	// }

	go func() {
		for {
			// 发布消息到同一主题
			testMessage := "test subscribe message"
			err := connector.Publish(topic, testMessage)
			if err != nil {
				t.Errorf("Failed to publish message for subscribe test: %v", err)
				//return
			}
			time.Sleep(1 * time.Second)
		}

	}()

	for {
		// 等待消息接收
		time.Sleep(1 * time.Second)
	}

}
