package plus

import (
	"agent/crypto"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// 这里存放着通过license.json文件解析出来的服务器配置
type SvrConfig struct {
	Broker   string
	Port     int
	ClientID string
	Username string
	Password string
	Server   string
	Token    string
}

func initConfig() (SvrConfig, error) {
	var svrconfig SvrConfig
	// 从config.data读取配置
	config, err := os.ReadFile("config.data")
	if err != nil {
		return SvrConfig{}, fmt.Errorf("failed to read config.data: %v", err)
	}
	selfk, err := crypto.Base64ToSM4Key("WolAxAGlcpEkVRBGvhnmlw==")

	// 解密配置
	decryptedConfig, err := crypto.SM4DecryptBase64(selfk, string(config))
	if err != nil {
		return SvrConfig{}, fmt.Errorf("failed to decrypt config.data: %v", err)
	}

	// 解析配置
	if err := json.Unmarshal([]byte(decryptedConfig), &svrconfig); err != nil {
		return SvrConfig{}, fmt.Errorf("failed to unmarshal config.data: %v", err)
	}

	return svrconfig, nil
}

// 这里存放着通过license.yaml文件解析出来的服务器配置
type YAMLConfig struct {
	NetIf         string `yaml:"NetIf"`
	AgentName     string `yaml:"AgentName"`
	StatusApi     string `yaml:"StatusApi"`
	ResultDataApi string `yaml:"ResultDataApi"`
	ResultSyncApi string `yaml:"ResultSyncApi"`
	LogSyncApi    string `yaml:"LogSyncApi"`
}

// ReadYamlConfig 读取YAML配置文件并加载到Base结构体
func InitYamlConfig() (YAMLConfig, error) {
	config := &YAMLConfig{}
	log.Println("Reading YAML config...")
	if err := readYAML("./config.yaml", config); err != nil {
		return *config, fmt.Errorf("failed to read YAML config: %v", err)
	}

	// 打印读取到的配置（示例）
	log.Printf("Read YAML Config: %+v", config)

	return *config, nil
}

// readYAML 读取YAML配置文件并解析到指定结构体
func readYAML(filePath string, config interface{}) error {
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
