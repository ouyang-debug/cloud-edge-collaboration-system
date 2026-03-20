//go:build plugin

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent/plus"
	"agent/plus/remote"
)

type yamlConfig struct {
	FileName  string `json:"fileName"`  //本地文件的名称
	LocalPath string `json:"localPath"` //本地文件相对于LocalWorkDir的路径
}

type input struct {
	TaskId         string      `json:"taskId"`
	TargetHost     string      `json:"target_host"`
	TargetPort     string      `json:"target_port"`
	TargetUser     string      `json:"target_user"`
	TargetPassword string      `json:"target_password"`
	Cmd            string      `json:"cmd"`
	WorkDir        string      `json:"workDir"`      //目标机中执行命令的目录
	LocalWorkDir   string      `json:"localworkdir"` //本地工作目录，用于存储yaml文件
	YamlConfig     interface{} `json:"yaml_config"`  //需要传输的yaml文件列表，支持字符串或数组
}

type cmdResult struct {
	Success         bool              `json:"success"`
	ExecutionTimeMs int64             `json:"executionTimeMs"`
	Stdout          string            `json:"stdout,omitempty"`
	Stderr          string            `json:"stderr,omitempty"`
	Error           map[string]string `json:"error,omitempty"`
}

type dockerPlugin struct{}

// Name returns the plugin name
func (p *dockerPlugin) Name() string {
	return "docker"
}

// Version returns the plugin version
func (p *dockerPlugin) Version() string {
	return "0.1.0"
}

// OutputType returns the plugin output type
func (p *dockerPlugin) OutputType() string {
	return "default"
}

// Description returns the plugin description
func (p *dockerPlugin) Description() string {
	return "Docker plugin for remote execution"
}

// Initialize initializes the plugin
func (p *dockerPlugin) Initialize(config string) error {
	return nil
}

// Shutdown shuts down the plugin
func (p *dockerPlugin) Shutdown() error {
	return nil
}

// Execute executes the plugin
func (p *dockerPlugin) Execute(input map[string]string) (map[string]string, error) {
	return p.ExecuteWithProgress("", input, nil)
}

// ExecuteWithProgress executes the plugin with progress reporting
func (p *dockerPlugin) ExecuteWithProgress(taskID string, inputMap map[string]string, reporter plus.ProgressReporter) (map[string]string, error) {
	// 解析输入参数
	in, err := parseInput(inputMap)
	if err != nil {
		return nil, err
	}

	// 使用传入的 taskID 覆盖输入中的 taskId
	if taskID != "" {
		in.TaskId = taskID
	}

	// 构建目标主机地址（包含端口）
	targetAddr := in.TargetHost
	if in.TargetPort != "" {
		targetAddr = fmt.Sprintf("%s:%s", in.TargetHost, in.TargetPort)
	}

	// 处理 YamlConfig 字段，确保它是 []yamlConfig 类型
	yamlConfigs := processYamlConfig(in.YamlConfig)

	// 检查是否是 docker compose 命令
	isComposeCmd := strings.Contains(strings.ToLower(in.Cmd), "docker compose")
	// 创建 SFTP 客户端
	sftpConfig := remote.SFTPConfig{
		Host:     targetAddr,
		User:     in.TargetUser,
		Password: in.TargetPassword,
	}
	sftpClient := remote.NewSFTPClient(sftpConfig)
	defer sftpClient.Close()
	// 如果是 docker compose 命令，需要传输 yaml 文件
	if isComposeCmd && len(yamlConfigs) > 0 {

		// 连接到目标主机
		_, err := sftpClient.Connect()
		if err != nil {
			return nil, fmt.Errorf("SFTP连接失败: %v", err)
		}

		// 上传所有 yaml 文件
		for _, yaml := range yamlConfigs {
			// 构建本地文件路径：LocalWorkDir + LocalPath + FileName
			localFilePath := in.LocalWorkDir
			if yaml.LocalPath != "" {
				localFilePath = fmt.Sprintf("%s/%s", localFilePath, yaml.LocalPath)
			}
			localFilePath = fmt.Sprintf("%s/%s", localFilePath, yaml.FileName)

			// 构建远程文件路径：WorkDir + LocalPath + FileName
			remoteFilePath := in.WorkDir
			if yaml.LocalPath != "" {
				remoteFilePath = fmt.Sprintf("%s/%s", remoteFilePath, yaml.LocalPath)
			}
			err = sftpClient.CreateDirectory(remoteFilePath)
			if err != nil {
				return nil, fmt.Errorf("创建目录失败: %v", err)
			}
			remoteFilePath = fmt.Sprintf("%s/%s", remoteFilePath, yaml.FileName)

			// 上传文件
			err = sftpClient.UploadFile(localFilePath, remoteFilePath)
			if err != nil {
				return nil, fmt.Errorf("上传文件失败 %s: %v", yaml.FileName, err)
			}
		}
	}

	// 执行命令
	start := time.Now()
	stdout, stderr, err := p.executeRemoteCommand(targetAddr, in.TargetUser, in.TargetPassword, in.Cmd, in.WorkDir)
	executionTime := time.Since(start).Milliseconds()

	// 构建结果
	result := cmdResult{
		Success:         err == nil,
		ExecutionTimeMs: executionTime,
		Stdout:          stdout,
		Stderr:          stderr,
	}

	if err != nil {
		result.Error = map[string]string{"message": err.Error()}
	}

	// 构建输出
	out := map[string]interface{}{
		"taskId":  in.TaskId,
		"success": result.Success,
		"result":  result,
	}

	// 转换为 JSON
	data, err := json.Marshal(out)
	if err != nil {
		if isComposeCmd && len(yamlConfigs) > 0 {
			//删除上传的 yaml 文件
			for _, yaml := range yamlConfigs {
				remoteFilePath := in.WorkDir
				if yaml.LocalPath != "" {
					remoteFilePath = fmt.Sprintf("%s/%s", remoteFilePath, yaml.LocalPath)
				}
				remoteFilePath = fmt.Sprintf("%s/%s", remoteFilePath, yaml.FileName)
				sftpClient.DeleteFile(remoteFilePath)
			}
		}
		return nil, err
	}
	if isComposeCmd && len(yamlConfigs) > 0 {
		//删除上传的 yaml 文件
		for _, yaml := range yamlConfigs {
			remoteFilePath := in.WorkDir
			if yaml.LocalPath != "" {
				remoteFilePath = fmt.Sprintf("%s/%s", remoteFilePath, yaml.LocalPath)
			}
			remoteFilePath = fmt.Sprintf("%s/%s", remoteFilePath, yaml.FileName)
			sftpClient.DeleteFile(remoteFilePath)
		}
	}
	return map[string]string{"stdout": string(data)}, nil
}

// executeRemoteCommand executes a command on the remote host
func (p *dockerPlugin) executeRemoteCommand(host, user, password, cmd, workDir string) (string, string, error) {
	// 解析主机和端口
	hostname := host
	port := 22

	// 检查 host 是否包含端口号
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		hostname = parts[0]
		// 尝试解析端口号
		if len(parts) > 1 {
			portStr := parts[1]
			portVal := 0
			fmt.Sscanf(portStr, "%d", &portVal)
			if portVal > 0 {
				port = portVal
			}
		}
	}

	// 创建 SSH 执行器
	executor := remote.NewSSHExecutor(remote.SSHConfig{
		Host:     hostname,
		Port:     port,
		User:     user,
		Password: password,
	})

	// 构建带工作目录的命令
	execCmd := cmd
	if workDir != "" {
		execCmd = fmt.Sprintf("cd %s && %s", workDir, cmd)
	}

	// 执行命令
	return executor.Execute(execCmd)
}

// parseInput parses the input parameters
func parseInput(inputMap map[string]string) (input, error) {
	// 从 map 中获取 JSON 字符串
	jsonStr, ok := inputMap["input"]
	if !ok {
		// 如果没有 input 字段，尝试直接从 map 中解析
		return input{
			TaskId:         inputMap["taskId"],
			TargetHost:     inputMap["targetHost"],
			TargetPort:     inputMap["targetPort"],
			TargetUser:     inputMap["targetUser"],
			TargetPassword: inputMap["targetPassword"],
			Cmd:            inputMap["cmd"],
			WorkDir:        inputMap["workDir"],
			LocalWorkDir:   inputMap["localworkdir"],
			YamlConfig:     parseYamlConfig(inputMap),
		}, nil
	}

	// 解析 JSON 字符串
	var in input
	err := json.Unmarshal([]byte(jsonStr), &in)
	if err != nil {
		return input{}, fmt.Errorf("解析输入参数失败: %v", err)
	}

	return in, nil
}

// parseYamlConfig parses the yaml_config parameter
func parseYamlConfig(inputMap map[string]string) []yamlConfig {
	// 尝试从 inputMap 中解析 yaml_config
	yamlConfigStr, ok := inputMap["yaml_config"]
	if !ok {
		return []yamlConfig{}
	}

	var yamlConfigs []yamlConfig
	err := json.Unmarshal([]byte(yamlConfigStr), &yamlConfigs)
	if err != nil {
		return []yamlConfig{}
	}

	return yamlConfigs
}

// processYamlConfig processes YamlConfig field, converting it to []yamlConfig
func processYamlConfig(inputYamlConfig interface{}) []yamlConfig {
	if inputYamlConfig == nil {
		return []yamlConfig{}
	}

	// 检查是否为字符串
	if str, ok := inputYamlConfig.(string); ok {
		var yamlConfigs []yamlConfig
		err := json.Unmarshal([]byte(str), &yamlConfigs)
		if err != nil {
			// 尝试将字符串解析为单个 yamlConfig 对象
			var singleYamlConfig yamlConfig
			err := json.Unmarshal([]byte(str), &singleYamlConfig)
			if err != nil {
				return []yamlConfig{}
			}
			return []yamlConfig{singleYamlConfig}
		}
		return yamlConfigs
	}

	// 检查是否为数组
	if arr, ok := inputYamlConfig.([]interface{}); ok {
		var yamlConfigs []yamlConfig
		for _, item := range arr {
			// 将每个元素转换为 map[string]interface{}
			if itemMap, ok := item.(map[string]interface{}); ok {
				yaml := yamlConfig{}
				// 提取 FileName
				if fileName, ok := itemMap["fileName"].(string); ok {
					yaml.FileName = fileName
				}
				// 提取 LocalPath
				if localPath, ok := itemMap["localPath"].(string); ok {
					yaml.LocalPath = localPath
				}
				yamlConfigs = append(yamlConfigs, yaml)
			}
		}
		return yamlConfigs
	}

	// 检查是否为 []yamlConfig 类型（当通过直接解析 inputMap 时）
	if yamlConfigs, ok := inputYamlConfig.([]yamlConfig); ok {
		return yamlConfigs
	}

	// 检查是否为 map[string]interface{} 类型（单个 yamlConfig 对象）
	if itemMap, ok := inputYamlConfig.(map[string]interface{}); ok {
		yaml := yamlConfig{}
		// 提取 FileName
		if fileName, ok := itemMap["fileName"].(string); ok {
			yaml.FileName = fileName
		}
		// 提取 LocalPath
		if localPath, ok := itemMap["localPath"].(string); ok {
			yaml.LocalPath = localPath
		}
		return []yamlConfig{yaml}
	}

	return []yamlConfig{}
}

func New() plus.Plugin {
	return &dockerPlugin{}
}
