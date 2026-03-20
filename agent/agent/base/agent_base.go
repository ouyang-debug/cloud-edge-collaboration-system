package base

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	// "path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	// mqtt "github.com/eclipse/paho.mqtt.golang"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	mqtt_connector "agent/base/mqtt_connector"
	"agent/crypto"

	// "agent/logsync"
	"agent/proto"
)

const (
	AGENT_VERSION = "v0.0.1"

	// 网络状态消息
	NET_CONNECTED    = "net:connected"
	NET_DISCONNECTED = "net:disconnected"
)

func getOSType() string {
	return runtime.GOOS
}

func getArchType() string {
	return runtime.GOARCH
}

// Base represents the Agent Base component
type Base struct {
	connector  *mqtt_connector.MQTTConnector
	plusCmd    *exec.Cmd
	grpcClient proto.AgentServiceClient
	grpcConn   *grpc.ClientConn
	// logSyncer   *logsync.LogSyncer // 日志同步器
	httpServer  *http.Server // Internal HTTP server for Plus communication
	config      *StartConfig // 配置
	netIf       string       // net interface
	agentName   string       // agent 名称
	agentIp     string       // agent IP地址
	agentID     string       // agent ID
	hbTopic     string       // 心跳主题
	dispTopic   string       // 任务分发主题
	ackTopic    string       // 确认主题
	registerUrl string       // 注册URL
	status      string       // 状态：ONLINE, FAILURE; 这里是plus状态，没用心跳是OFFLINE
	plusVersion string       // Plus 版本号
	selfk       []byte       //存储用
	commk       []byte       //通信用
	// ... other fields
}

const (
	DEFAULT_HEARTBEAT_TOPIC_TEMPLATE = "agent/heartbeat"
	DEFAULT_DISPATCH_TOPIC_TEMPLATE  = "agent/{clientID}/#"
	DEFAULT_ACK_TOPIC_TEMPLATE       = "agent/ack/{clientID}"
	API_AGENT_REGISTER_PATH          = "/api/env-node-registration/register" //"/api/agent/register"
)

// 这里存放着通过license.json文件解析出来的服务器配置
type SvrConfig struct {
	Broker   string
	Port     string
	ClientID string
	AgentID  string `json:"agentId"`
	Username string
	Password string
	Server   string
	Token    string
	SelfMac  string //server配置空即可
	Scheme   string // tcp|ws|wss
	WSPath   string // WebSocket 路径（例如 /mqtt）
	// HeartbeatTopicTpl string `json:"heartbeatTopic"` // 模板，如 agent/heartbeat/{clientID}
	// DispatchTopicTpl  string `json:"dispatchTopic"`  // 模板，如 agent/{clientID}/task/dispatch
	// AckTopicTpl       string `json:"ackTopic"`       // 模板，如 agent/ack/{clientID}
	// CommandsTopicTpl  string `json:"commandsTopic"`  // 模板，如 agent/{clientID}/commands/#
}

// 这里存放着通过license.yaml文件解析出来的服务器配置
type StartConfig struct {
	NetIf             string `yaml:"NetIf"`
	AgentName         string `yaml:"AgentName"`
	StatusApi         string `yaml:"StatusApi"`
	ResultDataApi     string `yaml:"ResultDataApi"`
	ResultSyncApi     string `yaml:"ResultSyncApi"`
	LogSyncApi        string `yaml:"LogSyncApi"`
	HeartbeatTopicTpl string `yaml:"HeartbeatTopic"` // 模板，如 agent/heartbeat/{clientID}
	DispatchTopicTpl  string `yaml:"DispatchTopic"`  // 模板，如 agent/{clientID}/#
	AckTopicTpl       string `yaml:"AckTopic"`       // 模板，如 agent/ack/{clientID}
	RegisterUrl       string `yaml:"RegisterUrl"`    // 模板，如 /api/env-node-registration/register

}

// GetNetIf 获取网络接口配置
func (b *Base) GetNetIf() string {
	return b.netIf
}

// readYAML 读取YAML配置文件并解析到指定结构体
func (b *Base) readYAML(filePath string, config interface{}) error {
	// 读取文件内容
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// 解析YAML到指定结构体
	if err := yaml.Unmarshal(fileData, config); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	return nil
}

// ReadYamlConfig 读取YAML配置文件并加载到Base结构体
func (b *Base) ReadYamlConfig() error {
	log.Println("Reading YAML config...")

	// 读取YAML配置文件
	config := &StartConfig{}
	if err := b.readYAML("./config.yaml", config); err != nil {
		return fmt.Errorf("failed to read YAML config: %v", err)
	}

	// 保存读取到的配置到Base结构体
	b.config = config
	b.netIf = config.NetIf
	b.agentName = config.AgentName
	b.registerUrl = config.RegisterUrl
	if b.registerUrl == "" {
		b.registerUrl = API_AGENT_REGISTER_PATH
	}

	// 打印读取到的配置（示例）
	log.Printf("Read YAML Config: %+v", config)

	return nil
}

// NewBase creates a new Base instance
func NewBase() *Base {
	// 初始化selfk字段
	selfkBytes, err := crypto.Base64ToSM4Key("WolAxAGlcpEkVRBGvhnmlw==")
	if err != nil {
		log.Printf("Warning: Failed to initialize selfk: %v, using empty string", err)
		selfkBytes = []byte("")
	}

	return &Base{
		selfk: selfkBytes,
	}
}

// Start initializes and starts the Base component
func (b *Base) Start() error {
	log.Println("Starting Agent Base...")

	err := b.ReadYamlConfig()
	if err != nil {
		return fmt.Errorf("Failed to read YAML config: %v", err)
	}
	// // 初始化日志同步器
	// if err := b.initLogSyncer(); err != nil {
	// 	return fmt.Errorf("failed to initialize LogSyncer: %v", err)
	// }

	// Start MQTT client
	if err := b.initMQTTClient(); err != nil {
		return fmt.Errorf("failed to initialize MQTT client: %v", err)
	}

	if err := b.startInternalHTTPServer(); err != nil {
		log.Printf("Warning: failed to start internal HTTP server: %v", err)
	}

	// Start Plus process
	if err := b.StartPlus(); err != nil {
		return fmt.Errorf("failed to start Plus: %v", err)
	}

	go b.monitorPlus()

	log.Println("Agent Base started successfully")
	return nil
}

// // InitLogSyncer 初始化日志同步器（公共方法，用于测试和外部调用）
// func (b *Base) InitLogSyncer() error {
// 	return b.initLogSyncer()
// }

// // initLogSyncer 初始化日志同步器
// func (b *Base) initLogSyncer() error {
// 	log.Println("Initializing LogSyncer...")

// 	// 配置日志同步器
// 	config := logsync.Config{
// 		ReadInterval: 5 * time.Second,
// 		ReadSize:     1024 * 1024,             // 1MB
// 		ServerURL:    "http://localhost:8080", // 默认服务器地址
// 		DBPath:       "./logsync/synclog.db",  // 数据库路径
// 	}

// 	// 确保数据库目录存在
// 	logsyncDir := filepath.Dir(config.DBPath)
// 	if err := os.MkdirAll(logsyncDir, 0755); err != nil {
// 		return fmt.Errorf("failed to create logsync directory: %v", err)
// 	}

// 	// 创建日志同步器
// 	logSyncer, err := logsync.NewLogSyncer(config)
// 	if err != nil {
// 		return fmt.Errorf("failed to create LogSyncer: %v", err)
// 	}

// 	// 启动日志同步器
// 	if err := logSyncer.Start(); err != nil {
// 		return fmt.Errorf("failed to start LogSyncer: %v", err)
// 	}

// 	// 保存日志同步器实例到Base结构体
// 	b.logSyncer = logSyncer

// 	log.Println("LogSyncer initialized successfully")
// 	return nil
// }

// publishHeartbeat 发布心跳消息
func (b *Base) publishHeartbeat(connector *mqtt_connector.MQTTConnector, topic string) {
	for {
		// 构建心跳消息
		heartbeat := map[string]string{
			"timestamp": time.Now().Format("2006-01-02T15:04:05"),
			"agentIp":   b.agentIp,
			"status":    b.status, // 这里应该是plus的状态
		}

		heartbeatBytes, err := json.Marshal(heartbeat)
		if err != nil {
			log.Printf("Failed to marshal heartbeat: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// 将heartbeatBytes进行加密
		encryptedData, err := crypto.SM4EncryptBase64(b.commk, string(heartbeatBytes))
		if err != nil {
			log.Printf("Failed to encrypt heartbeat: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		err = connector.Publish(topic, encryptedData)
		if err != nil {
			log.Printf("Failed to publish heartbeat message: %v", err)
		}

		// 判定网络是否连接
		// networkStatus := b.logSyncer.GetNetworkStatus()
		if !connector.IsConnected() {
			log.Printf("MQTT connector is not connected, retrying in 15 seconds...")
			// 判定grpc是否连接
			if b.grpcClient == nil {
				b.MessageNotice(1, NET_DISCONNECTED)
			}

			// if networkStatus == logsync.NetworkAvailable {
			// 	log.Printf("MQTT connector is connected, publishing heartbeat message: %s", string(heartbeatBytes))
			// 	b.logSyncer.SetNetworkStatus(logsync.NetworkUnavailable)
			// }
			time.Sleep(15 * time.Second)
		} else {
			if b.grpcClient == nil {
				b.MessageNotice(1, NET_CONNECTED)
			}
			// networkStatus := b.logSyncer.GetNetworkStatus()
			// if networkStatus != logsync.NetworkAvailable {
			// 	log.Printf("MQTT connector is connected, publishing heartbeat message: %s", string(heartbeatBytes))
			// 	b.logSyncer.SetNetworkStatus(logsync.NetworkAvailable)
			// }
		}
		time.Sleep(5 * time.Second)
	}
}

// Stop stops the Base component and the Plus process
func (b *Base) Stop() error {
	log.Println("Stopping Agent Base...")

	// Stop Plus process
	if err := b.StopPlus(); err != nil {
		log.Printf("Warning: failed to stop Plus: %v", err)
	}

	// Disconnect MQTT connector
	if b.connector != nil {
		b.connector.Disconnect() // 250ms quiesce period
	}

	log.Println("Agent Base stopped")
	return nil
}

// startInternalHTTPServer starts the internal HTTP server for Plus -> Base communication
func (b *Base) startInternalHTTPServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mqtt/publish", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Topic   string `json:"topic"`
			Payload string `json:"payload"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if b.connector != nil {
			if err := b.connector.Publish(req.Topic, req.Payload); err != nil {
				log.Printf("Failed to publish MQTT message via internal HTTP: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("Published MQTT message via internal HTTP: topic=%s", req.Topic)
		} else {
			http.Error(w, "MQTT connector not initialized", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	b.httpServer = &http.Server{
		Addr:    "127.0.0.1:12346",
		Handler: mux,
	}

	go func() {
		log.Println("Starting internal HTTP server on :12346")
		if err := b.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Internal HTTP server error: %v", err)
		}
	}()

	return nil
}

// 格式化MAC地址，将字节数组转为 xx:xx:xx:xx:xx:xx 格式
func (b *Base) formatMAC(mac net.HardwareAddr) string {
	if len(mac) == 0 {
		return "无MAC地址"
	}
	return mac.String()
}

// 格式化IP地址，过滤掉环回地址和无效地址
func (b *Base) formatIPs(addrs []net.Addr) []string {
	var ips []string
	for _, addr := range addrs {
		// 将地址转为IPNet类型（包含IP和子网掩码）
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		// 过滤掉环回地址和无效IP（如 ::/0 这类）
		if ipNet.IP.IsLoopback() || ipNet.IP.IsUnspecified() {
			continue
		}
		ips = append(ips, ipNet.IP.String())
	}
	return ips
}

// 获取本机指定网口IP及mac地址
func (b *Base) getLocalIPAndMac(interfaceName string) (ip string, mac string, err error) {
	// 检查接口名是否为空
	if interfaceName == "" {
		return "", "", fmt.Errorf("interface name is empty")
	}

	// 获取本机所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("获取网络接口失败: %v\n", err)
		return "", "", err
	}

	// 遍历每个网络接口
	for _, iface := range interfaces {
		// 过滤掉未启动的接口（标志位判断）
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Name != interfaceName {
			continue
		}

		// 输出网卡基本信息
		fmt.Printf("=== 网卡名称: %s ===\n", iface.Name)
		fmt.Printf("MAC 地址: %s\n", b.formatMAC(iface.HardwareAddr))
		mac = b.formatMAC(iface.HardwareAddr)
		// 获取该网卡的所有地址
		addrs, err := iface.Addrs()
		if err != nil {
			fmt.Printf("获取%s的IP地址失败: %v\n", iface.Name, err)
			continue
		}

		// 格式化并输出IP地址
		ips := b.formatIPs(addrs)
		if len(ips) == 0 {
			fmt.Println("IP 地址: 无有效IP")
		} else {
			fmt.Printf("IP 地址: %s\n", strings.Join(ips, ", "))
			fmt.Printf("IP 地址: %s\n", ips[0])
			ip = ips[0]
		}
		fmt.Println("------------------------")
	}

	return ip, mac, nil
	//return "192.168.66.15", "BCF4D43E14E1", nil
}

// httpPost 发送HTTP POST请求
func (b *Base) httpPost(url string, data []byte) (string, error) {
	// 创建HTTP客户端
	client := &http.Client{}

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	return string(respBody), nil
}

// sendRegisterRequest 发送注册请求到服务器
func (b *Base) sendRegisterRequest(svrconfig SvrConfig, ip, mac string) error {
	// 构建请求数据
	data := map[string]string{
		"hostIp": ip,
		// "mac":     mac,
		"agentName": b.agentName,
		"version":   AGENT_VERSION,
		"osType":    getOSType(),
		"archType":  getArchType(),
	}

	// 序列化数据
	bodyBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	// 加密数据
	encryptedData, err := crypto.SM4EncryptBase64(b.commk, string(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to encrypt body: %v", err)
	}

	// 构建发送数据
	senddata := map[string]string{
		"agentId": svrconfig.ClientID,
		"data":    encryptedData,
	}

	// 序列化发送数据
	senddataBytes, err := json.Marshal(senddata)
	if err != nil {
		return fmt.Errorf("failed to marshal senddata: %v", err)
	}
	// 增加打印日志
	log.Printf("register request: %s", string(senddataBytes))
	// 发送HTTP请求

	respBody, err := b.httpPost(svrconfig.Server+b.registerUrl, senddataBytes)
	if err != nil {
		return fmt.Errorf("failed to post senddata: %v", err)
	}

	// 解析响应
	var resp struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal([]byte(respBody), &resp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// 检查响应状态
	if resp.Code != "0" {
		return fmt.Errorf("register failed: %s", resp.Msg)
	}
	//增加打印日志
	log.Printf("register success: %s", resp.Msg)

	return nil
}

// saveConfigData 保存配置数据到config.data文件并删除license.json
func (b *Base) saveConfigData(svrconfig SvrConfig) error {
	// 序列化配置
	configBytes, err := json.Marshal(svrconfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// 加密配置
	encryptedLicense, err := crypto.SM4EncryptBase64(b.selfk, string(configBytes))
	if err != nil {
		return fmt.Errorf("failed to encrypt config: %v", err)
	}

	// 写入配置文件
	if err := os.WriteFile("config.data", []byte(encryptedLicense), 0644); err != nil {
		return fmt.Errorf("failed to write config.data: %v", err)
	}

	// 删除license.json文件
	if err := os.Remove("license.json"); err != nil {
		return fmt.Errorf("failed to remove license.json: %v", err)
	}

	return nil
}

func (b *Base) initConfig() (SvrConfig, error) {
	// 读取license.json文件，如果不存在，读取config.data
	license, err := os.ReadFile("license.json")
	var svrconfig SvrConfig
	var ip, mac string

	// 尝试从license.json读取配置
	if err == nil {
		// 解析license.json文件
		if err := json.Unmarshal(license, &svrconfig); err != nil {
			return SvrConfig{}, fmt.Errorf("failed to unmarshal license.json: %v", err)
		}
		if strings.TrimSpace(svrconfig.AgentID) == "" && strings.TrimSpace(svrconfig.ClientID) != "" {
			svrconfig.AgentID = svrconfig.ClientID
		}

		// 验证token是否为空
		if svrconfig.Token == "" {
			return SvrConfig{}, fmt.Errorf("token is empty in license.json")
		}

		// 初始化通信密钥
		commkBytes, err := crypto.Base64ToSM4Key(svrconfig.Token)
		if err != nil {
			log.Printf("Warning: Failed to initialize commk: %v, using empty string", err)
			return SvrConfig{}, fmt.Errorf("failed to read token from license.json: %v", err)
		}
		b.commk = commkBytes

		// 获取本地IP和MAC地址
		ip, mac, err = b.getLocalIPAndMac(b.netIf)
		if err != nil {
			return SvrConfig{}, fmt.Errorf("failed to get local IP and MAC: %v", err)
		}

		// 设置MAC地址到配置
		svrconfig.SelfMac = mac
	} else {
		// 从config.data读取配置
		config, err := os.ReadFile("config.data")
		if err != nil {
			return SvrConfig{}, fmt.Errorf("failed to read config.data: %v", err)
		}

		// 解密配置
		decryptedConfig, err := crypto.SM4DecryptBase64(b.selfk, string(config))
		if err != nil {
			return SvrConfig{}, fmt.Errorf("failed to decrypt config.data: %v", err)
		}

		// 解析配置
		if err := json.Unmarshal([]byte(decryptedConfig), &svrconfig); err != nil {
			return SvrConfig{}, fmt.Errorf("failed to unmarshal config.data: %v", err)
		}
		if strings.TrimSpace(svrconfig.AgentID) == "" && strings.TrimSpace(svrconfig.ClientID) != "" {
			svrconfig.AgentID = svrconfig.ClientID
		}

		// 获取本地IP和MAC地址
		ip, mac, err = b.getLocalIPAndMac(b.netIf)
		if err != nil {
			return SvrConfig{}, fmt.Errorf("failed to get local IP and MAC: %v", err)
		}

		// 验证MAC地址匹配
		if svrconfig.SelfMac != mac {
			return SvrConfig{}, fmt.Errorf("mac address not match: %v", err)
		}

		// 初始化通信密钥
		b.commk, err = crypto.Base64ToSM4Key(svrconfig.Token)
		if err != nil {
			return SvrConfig{}, fmt.Errorf("failed to initialize commk: %v", err)
		}
	}
	b.agentIp = ip
	// 发送注册请求
	// 增加打印日志
	log.Printf("register request ip: %s", ip)
	if err := b.sendRegisterRequest(svrconfig, ip, mac); err != nil {
		log.Printf("Error: register request failed: %v", err)
		return SvrConfig{}, fmt.Errorf("register request failed: %v", err)
	}

	// 如果是从license.json读取的配置，保存到config.data并删除license.json
	if license != nil {
		if err := b.saveConfigData(svrconfig); err != nil {
			return SvrConfig{}, err
		}
	}

	return svrconfig, nil
}

// InitMQTTClient 公共方法，用于测试和外部调用
func (b *Base) InitMQTTClient() error {
	return b.initMQTTClient()
}

func (b *Base) InitConfig() (SvrConfig, error) {
	return b.initConfig()
}

// initMQTTClient initializes the MQTT connector
func (b *Base) initMQTTClient() error {
	// Create new MQTT connector instance with configuration matching original code
	// config := mqtt_connector.MQTTConfig{
	// 	BrokerURL:     "tcp://localhost:1883", // Default MQTT broker address
	// 	ClientID:      "agent-base",
	// 	Username:      "mqtt",
	// 	Password:      "mqtt",
	// 	AutoReconnect: true,
	// 	Handler:       b.handleMQTTMessage,
	// }

	// // Connect to MQTT broker
	// if err := connector.Connect(); err != nil {
	// 	return fmt.Errorf("failed to connect MQTT connector: %v", err)
	// }

	config := mqtt_connector.MQTTConfig{
		Broker:   "127.0.0.1",
		Port:     1883,
		ClientID: "agent_base",
		Username: "mqtt",
		Password: "mqtt",
		QoS:      1,
	}

	//这里修订为从config.data读取配置
	svrconfig, err := b.InitConfig()
	if err != nil {
		return fmt.Errorf("failed to init config from config.data: %v", err)
	}
	//更新config
	config.Broker = svrconfig.Broker
	config.Port, err = strconv.Atoi(svrconfig.Port)
	if err != nil {
		return fmt.Errorf("failed to parse port from config.data: %v", err)
	}
	id := strings.TrimSpace(svrconfig.AgentID)
	if id == "" {
		id = strings.TrimSpace(svrconfig.ClientID)
	}
	b.agentID = id
	config.ClientID = id
	config.Username = svrconfig.Username
	config.Password = svrconfig.Password
	config.Scheme = svrconfig.Scheme
	config.WSPath = svrconfig.WSPath

	connector := mqtt_connector.NewMQTTConnector(config)

	hbTpl := strings.TrimSpace(b.config.HeartbeatTopicTpl)
	if hbTpl == "" {
		hbTpl = DEFAULT_HEARTBEAT_TOPIC_TEMPLATE
		// hbTpl = "agent/{agentID}/heartbeat"
	}
	dispTpl := strings.TrimSpace(b.config.DispatchTopicTpl)
	if dispTpl == "" {
		dispTpl = DEFAULT_DISPATCH_TOPIC_TEMPLATE
		// dispTpl = "agent/{clientID}/task/dispatch"
	}
	ackTpl := strings.TrimSpace(b.config.AckTopicTpl)
	if ackTpl == "" {
		ackTpl = DEFAULT_ACK_TOPIC_TEMPLATE
	}

	// commandsTpl := strings.TrimSpace(svrconfig.CommandsTopicTpl)
	// if commandsTpl == "" {
	// 	commandsTpl = "agent/{clientID}/commands/#"
	// }

	envAgentID := strings.TrimSpace(os.Getenv("AGENT_ID"))
	if envAgentID == "" || strings.EqualFold(envAgentID, "null") || strings.EqualFold(envAgentID, "undefined") {
		envAgentID = config.ClientID
	}
	topic_hb := strings.ReplaceAll(hbTpl, "{clientID}", envAgentID)
	topic_hb = strings.ReplaceAll(topic_hb, "{agentID}", envAgentID)
	topic_disp := strings.ReplaceAll(dispTpl, "{clientID}", envAgentID)
	topic_disp = strings.ReplaceAll(topic_disp, "{agentID}", envAgentID)
	ackTopic := strings.ReplaceAll(ackTpl, "{clientID}", b.agentID)
	ackTopic = strings.ReplaceAll(ackTopic, "{agentID}", b.agentID)
	// commandsTopic := strings.ReplaceAll(commandsTpl, "{clientID}", b.agentID)
	// commandsTopic = strings.ReplaceAll(commandsTopic, "{agentID}", b.agentID)

	b.hbTopic = topic_hb
	b.dispTopic = topic_disp
	b.ackTopic = ackTopic

	//messageReceived := false
	receivedMessage := ""

	// 创建一个局部变量来保存测试实例，以便在闭包中使用
	// testInstance := t

	// 自定义消息处理函数
	handler := func(t string, payload []byte) {
		//messageReceived = true
		receivedMessage = string(payload)
		// testInstance.Logf("Received message: %s from topic: %s", receivedMessage, t)
		fmt.Printf("Received message: %s from topic: %s\n", receivedMessage, t)
	}

	// 尝试连接MQTT broker，直到成功
	for {
		err := connector.Connect()
		if err != nil {
			log.Printf("Failed to connect to MQTT broker: %v, retrying in 1 second...", err)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Println("Connected to MQTT broker successfully")
		break
	}

	// 订阅心跳主题
	for {
		err := connector.Subscribe(topic_hb, handler)
		if err != nil {
			log.Printf("Failed to subscribe to topic %s: %v, retrying in 1 second...", topic_hb, err)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("Subscribed to topic %s successfully", topic_hb)
		break
	}

	// 订阅派发主题
	for {
		err := connector.Subscribe(topic_disp, b.handleMQTTMessage)
		if err != nil {
			log.Printf("Failed to subscribe to topic %s: %v, retrying in 1 second...", topic_disp, err)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("Subscribed to topic %s successfully", topic_disp)
		break
	}

	// // 订阅命令主题
	// for {
	// 	err := connector.Subscribe(commandsTopic, b.handleMQTTMessage)
	// 	if err != nil {
	// 		log.Printf("Failed to subscribe to topic %s: %v, retrying in 1 second...", commandsTopic, err)
	// 		time.Sleep(1 * time.Second)
	// 		continue
	// 	}
	// 	log.Printf("Subscribed to topic %s successfully", commandsTopic)
	// 	break
	// }

	// defer connector.Disconnect()  //TODO

	// if !connector.IsConnected() {
	// 	t.Log("Not connected, skipping subscribe test")
	// 	return
	// }

	// 启动心跳发布协程
	go b.publishHeartbeat(connector, topic_hb)

	log.Println("MQTT connector initialized and connected")
	// for {
	// 	time.Sleep(2 * time.Second)
	// }
	b.connector = connector
	return nil
}

// ackMQTTMessage sends acknowledgment message to MQTT broker
func (b *Base) ackMQTTMessage(cmdId string) error {
	// ackTpl := "agent/ack/{clientID}"

	cmdack := map[string]string{
		"timestamp": time.Now().Format("2006-01-02T15:04:05"),
		"agentID":   b.agentID,
		"cmdId":     cmdId,
	}

	log.Printf("cmdack message: %s", cmdack["cmdId"])
	cmdackBytes, err := json.Marshal(cmdack)
	if err != nil {
		log.Printf("Failed to marshal heartbeat: %v", err)
		return err
	}

	encryptedData, err := crypto.SM4EncryptBase64(b.commk, string(cmdackBytes))
	if err != nil {
		log.Printf("Failed to encrypt heartbeat: %v", err)
		time.Sleep(5 * time.Second)
		return err
	}

	err = b.connector.Publish(b.ackTopic, []byte(encryptedData))
	if err != nil {
		log.Printf("Failed to publish cmdack message: %v", err)
		return err
	}

	return nil
}

// handleMQTTMessage handles incoming MQTT messages
// func (b *Base) handleMQTTMessage(client mqtt.Client, msg mqtt.Message) {

func (b *Base) handleMQTTMessage(t string, payload []byte) {
	log.Printf("Received MQTT message: topic=%s, payload=%s", t, string(payload))
	//后续增加解密处理
	// decryptedData, err := crypto.SM4DecryptBase64(b.commk, string(payload))
	// if err != nil {
	// 	log.Printf("Failed to decrypt message: %v", err)
	// 	return
	// }

	if err := b.ackMQTTMessage("cmdId"); err != nil {
		log.Printf("Failed to send ackMQTTMessage: %v", err)
	}

	// TODO 需要增加判定命令是给plus还是给base，后续增加plus的设计
	// agent/{clientID}/task/dispatch
	// agent/{clientID}/command/dispatch
	if strings.HasPrefix(t, "agent/") && strings.Contains(t, "/task/") {
		// 命令是给Plus的
		// Forward message to Plus component via gRPC
		if err := b.forwardMQTTMessageToPlus(t, payload); err != nil {
			log.Printf("Failed to forward message to Plus: %v", err)
		}
	} else if strings.HasPrefix(t, "agent/") && strings.Contains(t, "/command/") {
		//TODO,先返回，后续放开处理

		// 命令是给Base的
		// 这里处理plus的停止、启动、状态功能，适配plus奔溃等问题
		if string(payload) == "stop" {
			b.StopPlus()
		} else if string(payload) == "start" {
			b.StopPlus()
			b.StartPlus()
		} else if strings.HasPrefix(string(payload), "forcestart") {
			// 解析格式："forcestart v0.2.0"
			parts := strings.Split(strings.TrimSpace(string(payload)), " ")
			if len(parts) >= 2 {
				forceVersion := parts[1]
				log.Printf("Upgrade required to version: %s", forceVersion)
				// 停止 Plus
				if err := b.StopPlus(); err != nil {
					log.Printf("Failed to stop Plus for upgrade: %v", err)
				} else {
					if err := b.createPlusEntry(forceVersion); err != nil {
						log.Printf("Failed to create plus entry: %v", err)
					}
					if err := b.StartPlus(); err != nil {
						log.Printf("Failed to restart Plus after upgrade: %v", err)
					}
					log.Printf("Plus restarted successfully after upgrade")
				}
			}
		} else if string(payload) == "status" {
			// 这里处理plus的状态功能
			status, err := b.GetPlusStatus()
			if err != nil {
				log.Printf("Failed to get Plus status: %v", err)
			} else {
				log.Printf("Plus status: %s", status)
			}
		}
	}

}

// StartPlus starts the Plus process
func (b *Base) StartPlus() error {
	log.Println("Starting Plus process...")

	// Check if Plus is already running
	if b.plusCmd != nil && b.plusCmd.Process != nil {
		return fmt.Errorf("Plus is already running")
	}

	// Start Plus process
	cmd := exec.Command(os.Args[0]+"plus", "plus")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// svrconfig, _ := b.InitConfig()
	envAgentID := os.Getenv("AGENT_ID")
	// if strings.TrimSpace(envAgentID) == "" || strings.EqualFold(envAgentID, "null") || strings.EqualFold(envAgentID, "undefined") {
	// 	envAgentID = svrconfig.ClientID
	// }
	// cmd.Env = append(os.Environ(), "AGENT_ID="+envAgentID, "CLIENT_ID="+svrconfig.ClientID)

	if strings.TrimSpace(envAgentID) == "" || strings.EqualFold(envAgentID, "null") || strings.EqualFold(envAgentID, "undefined") {
		envAgentID = b.agentID
	}
	cmd.Env = append(os.Environ(), "AGENT_ID="+envAgentID, "CLIENT_ID="+b.agentID)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Plus process: %v", err)
	}

	b.plusCmd = cmd
	log.Printf("Plus process started with PID: %d", cmd.Process.Pid)

	// Monitor Plus process
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("Plus process exited with error: %v", err)
		} else {
			log.Println("Plus process exited successfully")
		}
		b.plusCmd = nil
		// Close gRPC connection when Plus exits
		if b.grpcConn != nil {
			b.grpcConn.Close()
			b.grpcConn = nil
			b.grpcClient = nil
		}
	}()

	// Wait for Plus to start up and gRPC server to be ready
	time.Sleep(2 * time.Second)

	// Establish gRPC connection to Plus
	if err := b.establishGRPCConnection(); err != nil {
		log.Printf("Warning: failed to establish gRPC connection: %v", err)
		// Continue even if gRPC connection fails - we'll retry later
	}

	return nil
}

// monitorPlus monitors Plus status in a separate goroutine
func (b *Base) monitorPlus() {
	for {
		statusMap, err := b.GetPlusStatusDetailed()
		if err != nil {
			log.Printf("Failed to get Plus status: %v", err)
			b.status = "failure"
			b.plusVersion = ""
		} else {
			log.Printf("Plus status: %v", statusMap)
			status, ok := statusMap["status"].(int32)
			if ok && status == 1 {
				b.status = "online"
			} else {
				b.status = "failure"
				b.plusVersion = ""
			}
		}
		// 获取upgrade状态
		upgradeVersion, ok := statusMap["upgrade"].(string)
		if ok && upgradeVersion != "" {
			log.Printf("Upgrade required to version: %s", upgradeVersion)
			// 停止 Plus
			if err := b.StopPlus(); err != nil {
				log.Printf("Failed to stop Plus for upgrade: %v", err)
			} else {
				if err := b.createPlusEntry(upgradeVersion); err != nil {
					log.Printf("Failed to create plus entry: %v", err)
				}
				if err := b.StartPlus(); err != nil {
					log.Printf("Failed to restart Plus after upgrade: %v", err)
				}
				log.Printf("Plus restarted successfully after upgrade")
			}
		}
		//获取当前版本
		currentVersion, ok := statusMap["version"].(string)
		if ok && currentVersion != "" {
			b.plusVersion = currentVersion
			log.Printf("Current version: %s", currentVersion)
		}

		time.Sleep(5 * time.Second)

	}
}

func (b *Base) createPlusEntry(upgradeVersion string) error {
	cmd := exec.Command("bash", "updatetool.sh", "set", "agent."+upgradeVersion)
	cmd.Dir = "./"
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to run upgrade script: %v, output: %s", err, string(output))
		return err
	}
	log.Printf("Upgrade script output: %s", string(output))

	return nil
}

// StopPlus stops the Plus process
func (b *Base) StopPlus() error {
	log.Println("Stopping Plus process...")

	// First try to use gRPC to stop Plus gracefully
	if b.grpcClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := b.grpcClient.StopPlus(ctx, &proto.StopPlusRequest{})
		if err == nil && resp.Success {
			log.Println("Plus stopped gracefully via gRPC")
			b.plusCmd = nil
			// Close gRPC connection
			if b.grpcConn != nil {
				b.grpcConn.Close()
				b.grpcConn = nil
				b.grpcClient = nil
			}
			return nil
		}
		log.Printf("Warning: gRPC stop failed, falling back to process kill: %v", err)
	}

	// Fallback to process kill if gRPC fails
	if b.plusCmd == nil || b.plusCmd.Process == nil {
		return fmt.Errorf("Plus is not running")
	}

	// Send stop signal to Plus process
	if err := b.plusCmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill Plus process: %v", err)
	}

	// Wait for process to exit
	if err := b.plusCmd.Wait(); err != nil {
		log.Printf("Warning: Plus process exit error: %v", err)
	}

	b.plusCmd = nil
	// Close gRPC connection
	if b.grpcConn != nil {
		b.grpcConn.Close()
		b.grpcConn = nil
		b.grpcClient = nil
	}
	log.Println("Plus process stopped")
	return nil
}

// establishGRPCConnection establishes a gRPC connection to the Plus component
func (b *Base) establishGRPCConnection() error {
	// Check if already connected
	if b.grpcClient != nil && b.grpcConn != nil {
		return nil
	}

	log.Println("Establishing gRPC connection to Plus...")

	// Dial Plus gRPC server
	conn, err := grpc.Dial(":12345", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to dial Plus gRPC server: %v", err)
	}

	// Create gRPC client
	client := proto.NewAgentServiceClient(conn)
	b.grpcConn = conn
	b.grpcClient = client

	log.Println("gRPC connection to Plus established")
	return nil
}

// forwardMQTTMessageToPlus forwards MQTT messages to the Plus component via gRPC
func (b *Base) forwardMQTTMessageToPlus(topic string, payload []byte) error {
	// Check if Plus is running
	status, err := b.GetPlusStatus()
	if err != nil {
		return err
	}
	if status != "running" {
		return fmt.Errorf("Plus component is not running")
	}

	// Ensure gRPC connection is established
	if b.grpcClient == nil {
		if err := b.establishGRPCConnection(); err != nil {
			return fmt.Errorf("failed to establish gRPC connection: %v", err)
		}
	}

	// Forward message via gRPC
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := b.grpcClient.ForwardMQTTMessage(ctx, &proto.ForwardMQTTMessageRequest{
		Topic:   topic,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("gRPC forward failed: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("Plus failed to process message")
	}

	log.Printf("Successfully forwarded MQTT message to Plus via gRPC: topic=%s", topic)
	return nil
}

// GetPlusStatus gets the status of the Plus process
func (b *Base) GetPlusStatus() (string, error) {
	// First try to use gRPC to get status
	if b.grpcClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := b.grpcClient.GetPlusStatus(ctx, &proto.GetPlusStatusRequest{})
		if err == nil {
			if resp.Status == 1 {
				return "running", nil
			}
			return "stopped", nil
		}
		log.Printf("Warning: gRPC status check failed, falling back to process check: %v", err)
	}

	// Fallback to process check
	if b.plusCmd == nil || b.plusCmd.Process == nil {
		return "stopped", nil
	}

	// Check if process is still running
	if _, err := os.FindProcess(b.plusCmd.Process.Pid); err != nil {
		return "stopped", nil
	}

	return "running", nil
}

// GetPlusStatusDetailed gets the detailed status of the Plus process
func (b *Base) GetPlusStatusDetailed() (map[string]interface{}, error) {
	// First try to use gRPC to get status
	if b.grpcClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := b.grpcClient.GetPlusStatus(ctx, &proto.GetPlusStatusRequest{})
		if err == nil {
			result := map[string]interface{}{
				"status":      resp.Status,
				"version":     resp.Version,
				"upgrade":     resp.Upgrade,
				"plugins":     resp.Plugins,
				"runningtask": resp.Runningtask,
				"failedtask":  resp.Failedtask,
			}
			return result, nil
		}
		log.Printf("Warning: gRPC detailed status check failed, falling back to process check: %v", err)
	}

	// Fallback to process check
	if b.plusCmd == nil || b.plusCmd.Process == nil {
		return map[string]interface{}{
			"status": 0,
		}, nil
	}

	// Check if process is still running
	if _, err := os.FindProcess(b.plusCmd.Process.Pid); err != nil {
		return map[string]interface{}{
			"status": 0,
		}, nil
	}

	return map[string]interface{}{
		"status": 1,
	}, nil
}

// MessageNotice sync messages from Base to Plus
func (b *Base) MessageNotice(msgType int32, message string) (resp int32, messageRsp string, err error) {

	// First try to use gRPC to get status
	if b.grpcClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := b.grpcClient.MessageNotice(ctx, &proto.MessageNoticeRequest{
			Type:    msgType,
			Message: message,
		})
		if err == nil {
			return resp.Type, resp.Message, nil
		} else {
			log.Printf("Warning: notice, gRPC failed, falling back to process check: %v", err)
		}
	} else {
		// Ensure gRPC connection is established
		log.Printf("Warning: notice, gRPC connection is nil, falling back to process check: %v", err)
	}
	return 0, "", fmt.Errorf("notice, failed to notice message")
}
