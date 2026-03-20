package mqttconnector

import (
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTConfig 定义MQTT连接配置
type MQTTConfig struct {
	Broker               string
	Port                 int
	ClientID             string
	Username             string
	Password             string
	QoS                  byte
	AutoReconnect        bool          // 是否自动重连
	ReconnectInterval    time.Duration // 重连间隔时间
	MaxReconnectInterval time.Duration // 最大重连间隔时间
	Scheme               string        // 连接协议: tcp|ws|wss
	WSPath               string        // WebSocket 路径（例如 /mqtt）
}

// MQTTConnector MQTT连接器结构体
type MQTTConnector struct {
	client        mqtt.Client
	config        MQTTConfig
	connected     bool
	subscriptions map[string]MessageHandler // 存储已订阅的主题和对应的处理函数
}

// MessageHandler 消息处理函数类型
type MessageHandler func(topic string, payload []byte)

// DefaultMessageHandler 默认消息处理函数
func DefaultMessageHandler(topic string, payload []byte) {
	fmt.Printf("Received message: %s from topic: %s\n", payload, topic)
}

// NewMQTTConnector 创建新的MQTT连接器
func NewMQTTConnector(config MQTTConfig) *MQTTConnector {
	// 设置默认值
	if config.QoS == 0 {
		config.QoS = 0
	}

	// 设置重连默认值
	if !config.AutoReconnect {
		config.AutoReconnect = true // 默认开启自动重连
	}

	if config.ReconnectInterval == 0 {
		config.ReconnectInterval = 1 * time.Second // 默认重连间隔1秒
	}

	if config.MaxReconnectInterval == 0 {
		config.MaxReconnectInterval = 30 * time.Second // 默认最大重连间隔30秒
	}

	return &MQTTConnector{
		config:        config,
		connected:     false,
		subscriptions: make(map[string]MessageHandler), // 初始化订阅记录映射
	}
}

// Connect 连接到MQTT代理
func (m *MQTTConnector) Connect() error {
	opts := mqtt.NewClientOptions()
	scheme := strings.ToLower(strings.TrimSpace(m.config.Scheme))
	if scheme == "" {
		scheme = "tcp"
	}
	if scheme == "mqtt" {
		scheme = "tcp"
	}
	if strings.HasPrefix(scheme, "ws") {
		path := strings.TrimSpace(m.config.WSPath)
		if path == "" {
			path = "/mqtt"
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		opts.AddBroker(fmt.Sprintf("%s://%s:%d%s", scheme, m.config.Broker, m.config.Port, path))
	} else {
		opts.AddBroker(fmt.Sprintf("%s://%s:%d", scheme, m.config.Broker, m.config.Port))
	}
	opts.SetClientID(m.config.ClientID)
	opts.SetUsername(m.config.Username)
	opts.SetPassword(m.config.Password)

	// 设置重连选项
	opts.SetAutoReconnect(m.config.AutoReconnect)
	opts.SetConnectRetryInterval(m.config.ReconnectInterval)
	opts.SetMaxReconnectInterval(m.config.MaxReconnectInterval)

	// 设置连接和断开连接处理函数
	opts.OnConnect = func(client mqtt.Client) {
		m.connected = true
		fmt.Println("MQTT connected")

		// 重新订阅所有之前订阅过的主题
		if len(m.subscriptions) > 0 {
			fmt.Printf("Re-subscribing to %d topics\n", len(m.subscriptions))
			for topic, handler := range m.subscriptions {
				// 转换为MQTT库的消息处理函数类型
				mqttHandler := func(client mqtt.Client, msg mqtt.Message) {
					handler(msg.Topic(), msg.Payload())
				}

				token := client.Subscribe(topic, m.config.QoS, mqttHandler)
				if token.WaitTimeout(2*time.Second) && token.Error() != nil {
					fmt.Printf("Failed to resubscribe to topic %s: %v\n", topic, token.Error())
				} else {
					fmt.Printf("Resubscribed to topic: %s\n", topic)
				}
			}
		}
	}

	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		m.connected = false
		fmt.Printf("MQTT connection lost: %v\n", err)
	}

	// 创建客户端
	m.client = mqtt.NewClient(opts)

	// 连接
	token := m.client.Connect()
	if token.WaitTimeout(5*time.Second) && token.Error() != nil {
		err := token.Error()
		if m.config.Port == 8083 && !strings.HasPrefix(scheme, "ws") {
			wsPath := strings.TrimSpace(m.config.WSPath)
			if wsPath == "" {
				wsPath = "/mqtt"
			}
			if !strings.HasPrefix(wsPath, "/") {
				wsPath = "/" + wsPath
			}
			wsOpts := mqtt.NewClientOptions()
			wsOpts.AddBroker(fmt.Sprintf("%s://%s:%d%s", "ws", m.config.Broker, m.config.Port, wsPath))
			wsOpts.SetClientID(m.config.ClientID)
			wsOpts.SetUsername(m.config.Username)
			wsOpts.SetPassword(m.config.Password)
			wsOpts.SetAutoReconnect(m.config.AutoReconnect)
			wsOpts.SetConnectRetryInterval(m.config.ReconnectInterval)
			wsOpts.SetMaxReconnectInterval(m.config.MaxReconnectInterval)
			wsOpts.OnConnect = opts.OnConnect
			wsOpts.OnConnectionLost = opts.OnConnectionLost
			m.client = mqtt.NewClient(wsOpts)
			token2 := m.client.Connect()
			if token2.WaitTimeout(5*time.Second) && token2.Error() != nil {
				return err
			}
			return nil
		}
		return err
	}

	return nil
}

// IsConnected 检查是否已连接
func (m *MQTTConnector) IsConnected() bool {
	return m.connected && m.client.IsConnected()
}

// Publish 发布消息
func (m *MQTTConnector) Publish(topic string, payload interface{}) error {
	if !m.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	token := m.client.Publish(topic, m.config.QoS, false, payload)
	if token.WaitTimeout(2*time.Second) && token.Error() != nil {
		return token.Error()
	}

	return nil
}

// Subscribe 订阅主题
func (m *MQTTConnector) Subscribe(topic string, handler MessageHandler) error {
	if !m.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	// 转换为MQTT库的消息处理函数类型
	mqttHandler := func(client mqtt.Client, msg mqtt.Message) {
		handler(msg.Topic(), msg.Payload())
	}

	token := m.client.Subscribe(topic, m.config.QoS, mqttHandler)
	if token.WaitTimeout(2*time.Second) && token.Error() != nil {
		return token.Error()
	}

	// 记录订阅信息
	m.subscriptions[topic] = handler

	return nil
}

// Unsubscribe 取消订阅主题
func (m *MQTTConnector) Unsubscribe(topic string) error {
	if !m.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	token := m.client.Unsubscribe(topic)
	if token.WaitTimeout(2*time.Second) && token.Error() != nil {
		return token.Error()
	}

	// 从订阅记录中删除
	delete(m.subscriptions, topic)

	return nil
}

// Disconnect 断开连接
func (m *MQTTConnector) Disconnect() {
	if m.IsConnected() {
		m.client.Disconnect(250)
		m.connected = false
		fmt.Println("MQTT disconnected")
	}
}
