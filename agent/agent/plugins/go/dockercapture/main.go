//go:build plugin

package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"agent/plus"
	"agent/plus/remote"
)

// 插件静态定义变量
const (
	PluginName    = "oscapture"
	PluginVersion = "0.1.0"
)

// docker subobject种类
const (
	DOCKERINFO    = "dockerinfo"
	DOCKERDAEMON  = "dockerdaemon"
	DOCKERENTRY   = "dockerentry"
	DOCKERDF      = "dockerdf"
	DOCKERPS      = "dockerps"
	DOCKERINSPECT = "dockerinspect"
	DOCKERSTATS   = "dockerstats"
)

// ItemCapture 定义输出结构
type ItemCapture struct {
	TastId    string            `json:"tastId"`    // 任务ID
	DataType  string            `json:"dataType"`  // 数据类型
	ContentId string            `json:"contentId"` // 内容ID
	Content   string            `json:"content"`   // 内容
	Timestamp int64             `json:"timestamp"` // 时间戳
	Metadata  map[string]string `json:"metadata"`  // 元数据
}

// Item 定义原始指标数据结构，用于返回给bs端解析
type Item struct {
	MetricId      string            `json:"metricId"`      // 指标唯一ID（配置中心分配）
	MetricName    string            `json:"metricName"`    //- 指标名称
	MetricType    string            `json:"metricType"`    // 指标类型（txt/num）
	ObjectType    string            `json:"objectType"`    //- 运维对象类型（os、docker、k8s、database）
	SubobjectType string            `json:"subobjectType"` //- 运维对象子类型（按实际对象类型配置，无则填空）
	ObjectId      string            `json:"objectId"`      // 运维对象唯一ID（taskid等）
	SubobjectId   string            `json:"subobjectId"`   //- 运维对象子类型唯一ID（无则填空）
	Value         string            `json:"value"`         // 指标值（支持数值、字符串）
	Unit          string            `json:"unit"`          //- 指标单位（无则填空）
	Timestamp     int64             `json:"timestamp"`     // 采集时间戳（毫秒）
	Tags          map[string]string `json:"tags"`          // 自定义扩展标签，用于筛选、分组，根据taskid回填
}

// 定义一个map映射表，主键是必须是metricId，内容是一个结构数组，包括metricName和metricType、objectType、subobjectType
// 这里与os_base的指标定义没用关系，是单独的一个映射表，方便通过metricId查询属性信息
type MetricInfo struct {
	MetricId      string `json:"metricId"`      // 指标唯一ID（配置中心分配）
	MetricName    string `json:"metricName"`    //- 指标名称
	MetricType    string `json:"metricType"`    // 指标类型（txt/num）
	ObjectType    string `json:"objectType"`    //- 运维对象类型（os、docker、k8s、database）
	SubobjectType string `json:"subobjectType"` //- 运维对象子类型（按实际对象类型配置，无则填空）
	Unit          string `json:"unit"`          //- 指标单位（无则填空）
}

var MetricInfoMap = map[string]MetricInfo{
	"dockerinfo": {
		MetricId:      "dockerinfo",
		MetricName:    "dockerinfo",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_INFO",
		Unit:          "",
	},
	"dockerdaemon": {
		MetricId:      "dockerdaemon",
		MetricName:    "dockerDaemon",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_DAEMON",
		Unit:          "",
	},
	"dockerentry": {
		MetricId:      "dockerentry",
		MetricName:    "dockerEntry",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_ENTRY",
		Unit:          "",
	},
	"dockerinspect": {
		MetricId:      "dockerinspect",
		MetricName:    "dockerInspect",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_INSPECT",
		Unit:          "",
	},
	//-----------dockerdf---------------
	// 	"dockerdf"
	// ["id": "Images", "size": "11.77GB", "reclaimable": "730.2MB (6%)", ],
	// SubobjectId:dockerdf_id
	"dockerdf_id": {
		MetricId:      "dockerdf_id",
		MetricName:    "dockerDF_id",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_DF",
		Unit:          "",
	},
	"dockerdf_size": {
		MetricId:      "dockerdf_size",
		MetricName:    "dockerDF_size",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_DF",
		Unit:          "",
	},
	"dockerdf_reclaimable": {
		MetricId:      "dockerdf_reclaimable",
		MetricName:    "dockerDF_reclaimable",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_DF",
		Unit:          "",
	},
	//-----------dockerps---------------
	// ["id": "62ccbcf906f07443f078e524292aacdfd3465ac615e28a9ed4999785b4e9ab03",
	// "name": "charming_tesla",
	// "image": "sha256:a1dd7e946f8241525f0a50bc11e93bb0957c193f245ae50a77ebfe7eaad712e5",
	// "status": "Exited (128) 5 weeks ago",
	// "command": "\"/bin/sh -c 'git clone https://github.com/richfelker/musl-cross-make.git /musl-cross-make     && cd /musl-cross-make     && echo \\\"TARGET = aarch64-linux-musl\\\" > config.mak     && echo \\\"OUTPUT = /usr/local/musl\\\" >> config.mak     && echo \\\"GCC_URL = https://mirrors.tuna.tsinghua.edu.cn/gnu/gcc/gcc-12.2.0/gcc-12.2.0.tar.xz\\\" >> config.mak     && echo \\\"MUSL_URL = https://musl.libc.org/releases/musl-1.2.4.tar.gz\\\" >> config.mak     && echo \\\"BINUTILS_URL = https://mirrors.tuna.tsinghua.edu.cn/gnu/binutils/...+146 more",
	// "create": "2026-01-22 16:56:17 +0800 CST",
	// "ports": "", ]
	// SubobjectId: "dockerps_id"

	"dockerps_id": {
		MetricId:      "dockerps_id",
		MetricName:    "dockerPS_id",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_PS",
		Unit:          "",
	},
	"dockerps_name": {
		MetricId:      "dockerps_name",
		MetricName:    "dockerPS_name",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_PS",
		Unit:          "",
	},
	"dockerps_image": {
		MetricId:      "dockerps_image",
		MetricName:    "dockerPS_image",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_PS",
		Unit:          "",
	},
	"dockerps_status": {
		MetricId:      "dockerps_status",
		MetricName:    "dockerPS_status",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_PS",
		Unit:          "",
	},
	"dockerps_command": {
		MetricId:      "dockerps_command",
		MetricName:    "dockerPS_command",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_PS",
		Unit:          "",
	},
	"dockerps_create": {
		MetricId:      "dockerps_create",
		MetricName:    "dockerPS_create",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_PS",
		Unit:          "",
	},
	"dockerps_ports": {
		MetricId:      "dockerps_ports",
		MetricName:    "dockerPS_ports",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_PS",
		Unit:          "",
	},
	//-----------dockerstats---------------
	// ["id": "e325f2aa6769",
	// "container": "boring_elgamal",
	// "cpuPercent": "0.00%",
	// "memUsage": "0B / 0B",
	// "memPercent": "0.00%",
	// "netIo": "0B / 0B",
	// "blockIo": "0B / 0B",
	// "pids": "0", ]
	// SubobjectId: "dockerstats_id"
	"dockerstats_id": {
		MetricId:      "dockerstats_id",
		MetricName:    "dockerStats_id",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_STATS",
		Unit:          "",
	},
	"dockerstats_container": {
		MetricId:      "dockerstats_container",
		MetricName:    "dockerStats_container",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_STATS",
		Unit:          "",
	},
	"dockerstats_cpuPercent": {
		MetricId:      "dockerstats_cpuPercent",
		MetricName:    "dockerStats_cpuPercent",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_STATS",
		Unit:          "",
	},
	"dockerstats_memUsage": {
		MetricId:      "dockerstats_memUsage",
		MetricName:    "dockerStats_memUsage",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_STATS",
		Unit:          "",
	},
	"dockerstats_memPercent": {
		MetricId:      "dockerstats_memPercent",
		MetricName:    "dockerStats_memPercent",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_STATS",
		Unit:          "",
	},
	"dockerstats_netIo": {
		MetricId:      "dockerstats_netIo",
		MetricName:    "dockerStats_netIo",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_STATS",
		Unit:          "",
	},
	"dockerstats_blockIo": {
		MetricId:      "dockerstats_blockIo",
		MetricName:    "dockerStats_blockIo",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_STATS",
		Unit:          "",
	},
	"dockerstats_pids": {
		MetricId:      "dockerstats_pids",
		MetricName:    "dockerStats_pids",
		MetricType:    "txt",
		ObjectType:    "docker",
		SubobjectType: "DOCKER_STATS",
		Unit:          "",
	},
}

// Table 定义顶层JSON结构，包含count和rows
type Table struct {
	Count int    `json:"count"` // 行数
	Rows  []Item `json:"rows"`  // 行数据
}

type dockercapturePlugin struct{}

func (p *dockercapturePlugin) Name() string       { return PluginName }
func (p *dockercapturePlugin) Version() string    { return PluginVersion }
func (p *dockercapturePlugin) OutputType() string { return "monitor" }
func (p *dockercapturePlugin) Description() string {
	return "Docker command capture plugin with remote execution support"
}
func (p *dockercapturePlugin) Initialize(config string) error { return nil }
func (p *dockercapturePlugin) Shutdown() error                { return nil }

func (p *dockercapturePlugin) Execute(input map[string]string) (map[string]string, error) {
	host := input["targetHost"]
	user := input["targetUser"]
	pass := input["targetPassword"]
	key := input["targetKey"]
	portStr := input["targetPort"]
	cmdStrIn := input["cmd"]
	taskID := input["taskId"]

	if cmdStrIn == "" {
		return map[string]string{"stderr": "empty cmd"}, nil
	}

	// Parse command format
	cmdStr := strings.TrimSpace(cmdStrIn)
	cmdType := ""
	cmdParts := strings.SplitN(cmdStr, " ", 3)
	if len(cmdParts) >= 1 {
		cmdType = strings.TrimSpace(cmdParts[0])

		switch cmdType {
		case DOCKERINFO:
			cmdStr = "echo " +
				"ZG9ja2VyIGluZm8K=" +
				"|base64 -d |sh"
		case DOCKERDAEMON:
			cmdStr = "echo " +
				"Y2F0IC9ldGMvZG9ja2VyL2RhZW1vbi5qc29uCg==" +
				"|base64 -d |sh"
		case DOCKERENTRY:
			cmdStr = "echo " +
				"c3lzdGVtY3RsIC0tbm8tcGFnZXIgY2F0IGRvY2tlcgo=" +
				"|base64 -d |sh"
		case DOCKERDF:
			cmdStr = "echo " +
				"ZG9ja2VyIHN5c3RlbSBkZiAtLWZvcm1hdCAneyJpZCI6Int7LlR5cGV9fSIsInNpemUiOiJ7ey5TaXplfX0iLCJyZWNsYWltYWJsZSI6Int7LlJlY2xhaW1hYmxlfX0ifScK" +
				"|base64 -d |sh"
		case DOCKERPS:
			cmdStr = "echo " +
				"ZG9ja2VyIHBzIC1hIC0tbm8tdHJ1bmMgLS1mb3JtYXQgJ3t7cHJpbnRmICJ7XCJpZFwiOiVxLFwibmFtZVwiOiVxLFwiaW1hZ2VcIjolcSxcInN0YXR1c1wiOiVxLFwiY29tbWFuZFwiOiVxLFwiY3JlYXRlXCI6JXEsXCJwb3J0c1wiOiVxfSIgLklEIC5OYW1lcyAuSW1hZ2UgLlN0YXR1cyAuQ29tbWFuZCAuQ3JlYXRlZEF0IC5Qb3J0c319Jwo=" +
				"|base64 -d |sh"
		case DOCKERINSPECT:
			cmdStr = "echo " +
				"ZG9ja2VyIHBzIC1hIHwgZ3JlcCAtdiBDT05UQUlORVIgfCBhd2sgJ3tpZHM9aWRzICIgIiAkMX0gRU5EIHtwcmludCBzdWJzdHIoaWRzLDIpfSd8eGFyZ3MgZG9ja2VyIGluc3BlY3QK" +
				"|base64 -d |sh"
		case DOCKERSTATS:
			cmdStr = "echo " +
				"ZG9ja2VyIHN0YXRzIC1hIC0tbm8tc3RyZWFtIC0tZm9ybWF0ICd7e3ByaW50ZiAie1wiaWRcIjolcSxcImNvbnRhaW5lclwiOiVxLFwiY3B1UGVyY2VudFwiOiVxLFwibWVtVXNhZ2VcIjolcSxcIm1lbVBlcmNlbnRcIjolcSxcIm5ldElvXCI6JXEsXCJibG9ja0lvXCI6JXEsXCJwaWRzXCI6JXF9IiAuSUQgLk5hbWUgLkNQVVBlcmMgLk1lbVVzYWdlIC5NZW1QZXJjIC5OZXRJTyAuQmxvY2tJTyAuUElEc319Jwo=" +
				"|base64 -d |sh"

		default:
			return map[string]string{"stderr": "unknown cmd type"}, nil
		}
	}

	// If no host provided, or it's local, we can still use SSHExecutor
	// because it now handles local execution automatically.
	port := 22
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	if user == "" {
		user = "root"
	}

	executor := remote.NewSSHExecutor(remote.SSHConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: pass,
		Key:      key,
	})
	// Note: Connect/Close are handled inside Execute/ExecuteStreamed or by caller.
	// But since we are creating it here, we should ensure it's closed if a connection was made.
	defer executor.Close()

	var stdout, stderr string
	var err error

	// If it looks like a script execution (multiple lines or starts with shebang), use streamed execution
	if strings.Contains(cmdStr, "\n") || strings.HasPrefix(cmdStr, "#!") {
		shell := "/bin/bash"
		if runtime.GOOS == "windows" {
			shell = "powershell"
		}

		if strings.HasPrefix(cmdStr, "#!") {
			lines := strings.SplitN(cmdStr, "\n", 2)
			shell = strings.TrimPrefix(lines[0], "#!")
			shell = strings.TrimSpace(shell)
		}
		stdout, stderr, err = executor.ExecuteStreamed(shell, cmdStr)
	} else {
		stdout, stderr, err = executor.Execute(cmdStr)
	}

	// 创建Table实例并填充数据
	table := Table{
		Count: 1,
		Rows:  []Item{},
	}

	// 根据cmdType判断是否需要解析stdout，根据不同的cmdType，解析stdout的方式不同，例如：

	RowsCapture := []ItemCapture{}

	switch cmdType {
	case DOCKERINFO:
		// 解析stdout
		// docker info

		RowsCapture, err = parseDockerInfo(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	case DOCKERDAEMON:
		// 解析stdout
		// docker daemon.json

		RowsCapture, err = parseDockerDaemon(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	case DOCKERENTRY:
		// 解析stdout
		// docker entry

		RowsCapture, err = parseDockerEntry(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	case DOCKERDF:

		RowsCapture, err = parseDockerDF(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}

	case DOCKERPS:
		//解析stdout
		RowsCapture, err = parseDockerPS(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	case DOCKERSTATS:
		// 解析stdout

		RowsCapture, err = parseDockerStats(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	case DOCKERINSPECT:
		// 解析stdout
		// docker inspect

		RowsCapture, err = parseDockerInspect(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	default:
		// 先生成ItemCapture结构体
		item := ItemCapture{
			TastId:    "req-20260211123456-789",
			DataType:  "json",
			ContentId: "CONTENT_001",
			Content:   "",
			Timestamp: time.Now().UnixMilli(),
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}

		item.Content = stdout
		if taskID != "" {
			item.TastId = taskID
		}
		item.ContentId = strings.ToLower(cmdType)
		RowsCapture = append(RowsCapture, item)
	}

	// 调用output2items函数生成items
	items := output2items(RowsCapture)

	// 更新Table中的Rows
	table.Rows = items
	table.Count = len(items)

	// 将 Table 序列化为字符串
	tableBytes, err := json.Marshal(table)
	if err != nil {
		return map[string]string{"stderr": err.Error()}, nil
	}
	tableStr := string(tableBytes)

	res := map[string]string{
		"stdout": tableStr,
		"stderr": stderr,
	}

	return res, err
}

func (p *dockercapturePlugin) ExecuteWithProgress(taskID string, input map[string]string, reporter plus.ProgressReporter) (map[string]string, error) {
	// Docker capture plugin is single-step; we simply run it and report completion
	input["task_id"] = taskID
	out, err := p.Execute(input)
	if reporter != nil {
		reporter.OnProgress(taskID, "dockercapture", 1, 1, "")
		reporter.OnCompleted(taskID, "dockercapture", err == nil, "")
	}
	return out, err
}

// parseDockerInfo 解析 docker info 输出，返回结构化的 JSON
func parseDockerInfo(taskID, output string) ([]ItemCapture, error) {

	return parseTxt(taskID, output, DOCKERINFO)
}

// parseDockerDaemon 解析 docker daemon 输出，返回结构化的 JSON
func parseDockerDaemon(taskID, output string) ([]ItemCapture, error) {

	return parseTxt(taskID, output, DOCKERDAEMON)
}

// parseDockerEntry 解析 docker entry 输出，返回结构化的 JSON
func parseDockerEntry(taskID, output string) ([]ItemCapture, error) {

	return parseTxt(taskID, output, DOCKERENTRY)
}

// parseDockerDF 解析 docker df 输出，返回结构化的 JSON
func parseDockerDF(taskID, output string) ([]ItemCapture, error) {
	// return Rows, nil
	return parseJson(taskID, output, DOCKERDF)
}

// parseDockerStats 解析 docker stats 输出，返回结构化的 JSON
func parseDockerStats(taskID, output string) ([]ItemCapture, error) {

	// return Rows, nil
	return parseJson(taskID, output, DOCKERSTATS)
}

// parseDockerPS 解析 docker ps 输出，返回结构化的 JSON
func parseDockerPS(taskID, output string) ([]ItemCapture, error) {

	Rows := []ItemCapture{}
	lines := strings.Split(output, "\n")

	// 处理每一行
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 生成ItemCapture结构体
		item := ItemCapture{
			TastId:    "",
			DataType:  "json",
			ContentId: "",
			Content:   line,
			Timestamp: time.Now().UnixMilli(),
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}

		// 提取id字段作为TastId
		var containerInfo map[string]interface{}
		err := json.Unmarshal([]byte(line), &containerInfo)
		if err == nil {
			if id, ok := containerInfo["id"].(string); ok && id != "" {
				item.ContentId = strings.ToLower(DOCKERPS) + "_" + strings.ReplaceAll(id, " ", "")[:12]
			}
		}

		// 如果提供了taskID，使用taskID
		if taskID != "" {
			item.TastId = taskID
		}

		Rows = append(Rows, item)
	}

	return Rows, nil
}

// parseDockerInspect 解析 docker inspect 输出，返回结构化的 JSON
func parseDockerInspect(taskID, output string) ([]ItemCapture, error) {

	// output输出的内容格式如下，请解析到item中，并将Config.Hostname字段的值作为ContentId
	// [
	//   {
	//     "Id": "afb34abbe32e47eedc4dafee236cc44f1620b9fb3d51139e52181c9ad6c0516c",
	//     "Created": "2026-01-21T01:56:00.77218557Z",
	//     "Name": "/kuboard3",
	//     "Config": {
	//       "Hostname": "afb34abbe32e"
	//     }
	//   },
	//   {
	//     "Id": "72289808fa0bf1a52a48cb708976f8bb25014a63876bc260715cac7c22fb0d24",
	//     "Created": "2026-01-21T00:53:32.34238579Z",
	//     "Name": "/kubepi",
	//     "Config": {
	//       "Hostname": "72289808fa0b"
	//     }
	//   },
	//   {
	//     "Id": "f5ac69cb32a89aa6cd0fdcfd9e93485bcc1699af97fd3448683d009d4a8795d7",
	//     "Created": "2026-01-12T06:33:57.82722213Z",
	//     "Name": "/kuboard",
	//     "Config": {
	//       "Hostname": "f5ac69cb32a8"
	//     }
	//   }
	// ]

	Rows := []ItemCapture{}

	// 解析JSON数组
	var inspectData []map[string]interface{}
	err := json.Unmarshal([]byte(output), &inspectData)
	if err != nil {
		// 如果解析失败，返回错误
		return Rows, err
	}

	// 处理每个容器
	for _, container := range inspectData {
		// 生成ItemCapture结构体
		item := ItemCapture{
			TastId:    "req-20260210123456-789",
			DataType:  "txt",
			ContentId: strings.ToLower(DOCKERINSPECT),
			Content:   "",
			Timestamp: time.Now().UnixMilli(),
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}

		// 将容器数据转换回JSON字符串
		containerJSON, err := json.Marshal(container)
		if err == nil {
			item.Content = string(containerJSON)
		}

		// 提取Config.Hostname作为ContentId
		if config, ok := container["Config"].(map[string]interface{}); ok {
			if hostname, ok := config["Hostname"].(string); ok && hostname != "" {
				item.ContentId = strings.ToLower(DOCKERINSPECT) + "_" + hostname
			}
		}

		// 如果提供了taskID，使用taskID
		if taskID != "" {
			item.TastId = taskID
		}

		Rows = append(Rows, item)
	}

	return Rows, nil
}

func parseJson(taskID, output string, cmdType string) ([]ItemCapture, error) {
	Rows := []ItemCapture{}
	lines := strings.Split(output, "\n")

	// 处理每一行
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 生成ItemCapture结构体
		item := ItemCapture{
			TastId:    "",
			DataType:  "json",
			ContentId: "",
			Content:   line,
			Timestamp: time.Now().UnixMilli(),
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}

		// 提取id字段作为TastId
		var containerInfo map[string]interface{}
		err := json.Unmarshal([]byte(line), &containerInfo)
		if err == nil {
			if id, ok := containerInfo["id"].(string); ok && id != "" {
				item.ContentId = strings.ToLower(cmdType) + "_" + strings.ReplaceAll(id, " ", "")
			}
		}

		// 如果提供了taskID，使用taskID
		if taskID != "" {
			item.TastId = taskID
		}

		Rows = append(Rows, item)
	}

	return Rows, nil
}
func parseTxt(taskID, output string, cmdType string) ([]ItemCapture, error) {
	Rows := []ItemCapture{}
	// 先生成ItemCapture结构体
	item := ItemCapture{
		TastId:    "",
		DataType:  "txt",
		ContentId: "",
		Content:   "",
		Timestamp: time.Now().UnixMilli(),
		Metadata: map[string]string{
			"source": "agentname",
			"stepId": "none",
		},
	}

	item.Content = output
	if taskID != "" {
		item.TastId = taskID
	}

	item.ContentId = strings.ToLower(cmdType)
	Rows = append(Rows, item)

	return Rows, nil
}

func New() plus.Plugin { return &dockercapturePlugin{} }

// output2items 将RowsCapture转换为Item数组
func output2items(RowsCapture []ItemCapture) []Item {
	items := []Item{}
	for _, itemCapture := range RowsCapture {
		itemCapture.Content = strings.TrimSpace(itemCapture.Content)
		// dockerinspect_e325f2aa6769
		parts := strings.Split(itemCapture.ContentId, "_")

		ObjectType := parts[0]
		SubobjectId := ""
		if len(parts) == 2 {
			SubobjectId = parts[1]
		}

		// 根据itemCapture.DataType判断是否需要解析Content
		if itemCapture.DataType == "json" {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(itemCapture.Content), &data); err == nil {
				itemCapture.Content = fmt.Sprintf("%v", data)
				// 遍历data，将每个key-value对转换为Item
				for key, value := range data {
					metricInfo, ok := MetricInfoMap[ObjectType+"_"+key]
					if !ok {
						continue
					}
					items = append(items, Item{
						MetricId:      metricInfo.MetricId,
						MetricName:    metricInfo.MetricName,
						MetricType:    metricInfo.MetricType,
						ObjectType:    metricInfo.ObjectType,
						SubobjectType: metricInfo.SubobjectType, //- 运维对象子类型（按实际对象类型配置，无则填空）
						ObjectId:      itemCapture.TastId,       // 运维对象唯一ID（taskid等）
						SubobjectId:   SubobjectId,              //- 运维对象子类型唯一ID（无则填空）
						Value:         fmt.Sprintf("%s", value), // 指标值（支持数值、字符串）
						Unit:          metricInfo.Unit,          //- 指标单位（无则填空）
						Timestamp:     itemCapture.Timestamp,    // 采集时间戳（毫秒）
					})
				}
			}
		} else if itemCapture.DataType == "txt" {
			metricInfo, ok := MetricInfoMap[ObjectType]
			if !ok {
				continue
			}

			// 声明Item，并补充到Items中
			items = append(items, Item{

				MetricId:      metricInfo.MetricId,
				MetricName:    metricInfo.MetricName,
				MetricType:    metricInfo.MetricType,
				ObjectType:    metricInfo.ObjectType,
				SubobjectType: metricInfo.SubobjectType, //- 运维对象子类型（按实际对象类型配置，无则填空）
				ObjectId:      itemCapture.TastId,       // 运维对象唯一ID（taskid等）
				SubobjectId:   SubobjectId,              //- 运维对象子类型唯一ID（无则填空）
				Value:         itemCapture.Content,      // 指标值（支持数值、字符串）
				Unit:          metricInfo.Unit,          //- 指标单位（无则填空）
				Timestamp:     itemCapture.Timestamp,    // 采集时间戳（毫秒）
				Tags:          map[string]string{},
			})

		}
	}
	return items
}
