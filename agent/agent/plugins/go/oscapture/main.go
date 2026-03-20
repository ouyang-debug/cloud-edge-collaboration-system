//go:build plugin

package main

import (
	"encoding/json"
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

// OS subobject种类
const (
	OS_BASE     = "os_base"
	OS_USAGE    = "os_usage"
	OS_NET_LOSS = "os_net_loss"
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

// ItemRaw 定义原始指标数据结构
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

// 定义一个map映射常量，可用通过metricId查询到metricinfo，这里与os_base的指标定义没有关系，是单独的一个映射表，方便通过metricId查询属性信息
// 这里的指标定义是从os_base中复制过来的，实际使用时，根据实际指标定义进行修改
// 补充：os_base的指标定义中其他内容，参考如下：
// {\"content\":{\"cpuArch\":\"x86_64\",\"cpuCores\":\"16\",\"hostname\":\"bogon\",\"memory\":\"33565159424\",\"os\":\"CentOS Linux 7 (Core)\",\"storage\":\"536870912000\",\"timeOffset\":\"+0.000669002\",\"uptime\":\"up 17 weeks, 6 days, 5 hours, 4 minutes\"},\"contentId\":\"OS_BASE\",\"dataType\":\"json\",\"metadata\":{\"source\":\"agentname\",\"stepId\":\"none\"},\"tastId\":\"task123\",\"timestamp\":1770880447543}"

// 指标定义：
// 指标ID：os_base_arch
// 指标名称：cpuArch
// 指标类型：txt
// 运维对象类型：os
// 运维对象子类型：OS_BASE
// 指标ID：os_base_cores
// 指标名称：cpuCores
// 指标类型：txt
// 运维对象类型：os
// 运维对象子类型：OS_BASE
// 指标ID：os_base_hostname
// 指标名称：hostname
// 指标类型：txt
// 运维对象类型：os
// 运维对象子类型：OS_BASE
// 指标ID：os_base_memory
// 指标名称：memory
// 指标类型：txt
// 运维对象类型：os
// 运维对象子类型：OS_BASE
// 指标ID：os_base_os
// 指标名称：os
// 指标类型：txt
// 运维对象类型：os
// 运维对象子类型：OS_BASE
// 指标ID：os_base_storage
// 指标名称：storage
// 指标类型：txt
// 运维对象类型：os
// 运维对象子类型：OS_BASE
// 指标ID：os_base_uptime
// 指标名称：uptime
// 指标类型：txt
// 运维对象类型：os
// 运维对象子类型：OS_BASE
// 指标ID：os_base_timeOffset
// 指标名称：timeOffset
// 指标类型：txt
// 运维对象类型：os
// 运维对象子类型：OS_BASE

//  补充OS_USAGE指标定义：
//  {\"content\":{\"cpuUsage\":\"1\",\"diskUsage\":\"30\",\"memoryUsage\":\"8.7\"},\"contentId\":\"OS_USAGE\",\"dataType\":\"json\",\"metadata\":{\"source\":\"agentname\",\"stepId\":\"none\"},\"tastId\":\"task123\",\"timestamp\":1770880520762}"
// 指标ID：os_usage_cpuUsage
// 指标名称：cpuUsage
// 指标类型：num
// 运维对象类型：os
// 运维对象子类型：OS_USAGE
// 指标单位：%
// 指标ID：os_usage_diskUsage
// 指标名称：diskUsage
// 指标类型：num
// 运维对象类型：os
// 运维对象子类型：OS_USAGE
// 指标单位：%
// 指标ID：os_usage_memoryUsage
// 指标名称：memoryUsage
// 指标类型：num
// 运维对象类型：os
// 运维对象子类型：OS_USAGE
// 指标单位：%

// 补充OS_NET_LOSS指标定义
// "{\"content\":{\"pkg_loss_rate\":\"0\"},\"content_id\":\"OS_NET_LOSS\",\"data_type\":\"json\",\"metadata\":{\"source\":\"agentname\",\"step_id\":\"none\"},\"tast_id\":\"task123\",\"timestamp\":1749602096000}"
// 指标ID：os_usage_netLoss
// 指标名称：netLoss
// 指标类型：num
// 运维对象类型：os
// 运维对象子类型：OS_NET_LOSS
// 指标单位：%

var MetricInfoMap = map[string]MetricInfo{
	"cpuArch": {
		MetricId:      "os_base_arch",
		MetricName:    "cpuArch",
		MetricType:    "txt",
		ObjectType:    "os",
		SubobjectType: "OS_BASE",
		Unit:          "",
	},
	"cpuCores": {
		MetricId:      "os_base_cores",
		MetricName:    "cpuCores",
		MetricType:    "txt",
		ObjectType:    "os",
		SubobjectType: "OS_BASE",
		Unit:          "",
	},
	"hostname": {
		MetricId:      "os_base_hostname",
		MetricName:    "hostname",
		MetricType:    "txt",
		ObjectType:    "os",
		SubobjectType: "OS_BASE",
		Unit:          "",
	},
	"memory": {
		MetricId:      "os_base_memory",
		MetricName:    "memory",
		MetricType:    "txt",
		ObjectType:    "os",
		SubobjectType: "OS_BASE",
		Unit:          "",
	},
	"os": {
		MetricId:      "os_base_os",
		MetricName:    "os",
		MetricType:    "txt",
		ObjectType:    "os",
		SubobjectType: "OS_BASE",
		Unit:          "",
	},
	"storage": {
		MetricId:      "os_base_storage",
		MetricName:    "storage",
		MetricType:    "txt",
		ObjectType:    "os",
		SubobjectType: "OS_BASE",
		Unit:          "",
	},
	"uptime": {
		MetricId:      "os_base_uptime",
		MetricName:    "uptime",
		MetricType:    "txt",
		ObjectType:    "os",
		SubobjectType: "OS_BASE",
		Unit:          "",
	},
	"timeOffset": {
		MetricId:      "os_base_timeOffset",
		MetricName:    "timeOffset",
		MetricType:    "txt",
		ObjectType:    "os",
		SubobjectType: "OS_BASE",
		Unit:          "",
	},
	"cpuUsage": {
		MetricId:      "os_usage_cpuUsage",
		MetricName:    "cpuUsage",
		MetricType:    "num",
		ObjectType:    "os",
		SubobjectType: "OS_USAGE",
		Unit:          "%",
	},
	"diskUsage": {
		MetricId:      "os_usage_diskUsage",
		MetricName:    "diskUsage",
		MetricType:    "num",
		ObjectType:    "os",
		SubobjectType: "OS_USAGE",
		Unit:          "%",
	},
	"memoryUsage": {
		MetricId:      "os_usage_memoryUsage",
		MetricName:    "memoryUsage",
		MetricType:    "num",
		ObjectType:    "os",
		SubobjectType: "OS_USAGE",
		Unit:          "%",
	},
	"pkgLossRate": {
		MetricId:      "os_usage_pkgLossRate",
		MetricName:    "pkgLossRate",
		MetricType:    "num",
		ObjectType:    "os",
		SubobjectType: "OS_NET_LOSS",
		Unit:          "%",
	},
}

// Table 定义顶层JSON结构，包含count和rows
type Table struct {
	Count int    `json:"count"` // 行数
	Rows  []Item `json:"rows"`  // 行数据
}

type oscapturePlugin struct{}

func (p *oscapturePlugin) Name() string    { return PluginName }
func (p *oscapturePlugin) Version() string { return PluginVersion }
func (p *oscapturePlugin) OutputType() string {
	return "monitor"
}
func (p *oscapturePlugin) Description() string {
	return "OS command capture plugin with remote execution support"
}
func (p *oscapturePlugin) Initialize(config string) error { return nil }
func (p *oscapturePlugin) Shutdown() error                { return nil }

func (p *oscapturePlugin) Execute(input map[string]string) (map[string]string, error) {
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
		case OS_BASE:
			cmdStr = "echo " +
				"ZWNobyAtbiAib3M6IgpjYXQgL2V0Yy9vcy1yZWxlYXNlIHxncmVwICJQUkVUVFlfTkFNRSJ8YXdrIC1GJyInICd7cHJpbnQgJDJ9JwplY2hvIC1uICJob3N0bmFtZToiCmhvc3RuYW1lCmVjaG8gLW4gImNwdUFyY2g6Igp1bmFtZSAtbQplY2hvIC1uICJjcHVDb3JlczoiCm5wcm9jCmVjaG8gLW4gIm1lbW9yeToiCmZyZWUgLWJ8Z3JlcCBNZW18YXdrICd7cHJpbnQgJDJ9JwplY2hvIC1uICJzdG9yYWdlOiIKbHNibGsgIC1ifGdyZXAgZGlza3xhd2sgJ3tzdW0gKz0gJDR9IEVORCB7cHJpbnQgc3VtfScKZWNobyAtbiAidXB0aW1lOiIKdXB0aW1lIC1wCmVjaG8gLW4gInRpbWVPZmZzZXQ6IgpjaHJvbnljIHRyYWNraW5nIHwgZ3JlcCAiTGFzdCBvZmZzZXQiIHwgYXdrICd7cHJpbnQgJDR9Jwo=" +
				"|base64 -d |sh"
		case OS_USAGE:
			cmdStr = "echo " +
				"ZWNobyAtbiAiY3B1VXNhZ2U6Igp0b3AgLWIgLW4gMSB8IGF3ayAnLyVDcHUvIHtwcmludCAxMDAgLSAkOH0nCmVjaG8gLW4gIm1lbW9yeVVzYWdlOiIKZnJlZSAtYiB8IGF3ayAnL01lbS8ge3ByaW50ZiAiJS4xZlxuIiwgKCQzLyQyKSoxMDB9JwplY2hvIC1uICJkaXNrVXNhZ2U6IgpkZiAtaHxncmVwICIvJCJ8YXdrICd7cHJpbnQgJDV9J3xhd2sgLUYiJSIgJ3twcmludCAkMX0nCg==" +
				"|base64 -d |sh"
		case OS_NET_LOSS:
			if len(cmdParts) >= 2 {
				targetIP := strings.TrimSpace(cmdParts[1])
				cmdStr = "echo -n pkgLossRate:; (IP=" +
					targetIP +
					"; ping -c 60 -i 1 $IP | awk '/packet loss/ {match($0, /([0-9]+)% packet loss/, arr); print arr[1]}')"
			} else {
				return map[string]string{"stderr": "missing target IP for os_net_loss"}, nil
			}
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

	// 调用output2items函数将输出转换为测量项
	items, err := output2items(stdout, taskID, cmdType)
	if err != nil {
		return map[string]string{"stderr": err.Error()}, nil
	}
	// 创建Table实例并填充数据
	table := Table{
		Count: 1,
		Rows:  []Item{},
	}
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

func (p *oscapturePlugin) ExecuteWithProgress(taskID string, input map[string]string, reporter plus.ProgressReporter) (map[string]string, error) {
	// OS capture plugin is single-step; we simply run it and report completion
	input["taskId"] = taskID
	out, err := p.Execute(input)
	if reporter != nil {
		reporter.OnProgress(taskID, "oscapture", 1, 1, "")
		reporter.OnCompleted(taskID, "oscapture", err == nil, "")
	}
	return out, err
}

// output2items 将命令输出转换为测量项数组
func output2items(stdout, taskID, cmdType string) ([]Item, error) {
	// 先生成ItemCapture结构体
	itemCapture := ItemCapture{
		TastId:    "req-20260210123456-789",
		DataType:  "json",
		ContentId: "CONTENT_001",
		Content:   "",
		Timestamp: time.Now().UnixMilli(),
		Metadata: map[string]string{
			"source": "agentname",
			"stepId": "none",
		},
	}

	content := map[string]string{}
	for _, line := range strings.Split(stdout, "\n") {
		if line == "" {
			continue
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		content[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	contentStr := string(contentJSON)

	// 填充content字段
	contentStr = strings.ReplaceAll(contentStr, "\n", "")
	itemCapture.Content = contentStr
	if taskID != "" {
		itemCapture.TastId = taskID
	}

	switch cmdType {
	case OS_BASE:
		// stdout 输出如下
		//"os:CentOS Linux 7 (Core)\nhostname:bogon\ncpu_arch:x86_64\ncpu_cores:16\nmemory:33565159424\nstorage:536870912000\nuptime:up 17 weeks, 6 days, 3 hours, 40 minutes\ncpu_cores:16\nmemory:33565159424\nstorage:536870912000\nuptime:up 17 weeks, 6 days, 3 hours, 40 minutes\ncpu_cores:16\nmemory:33565159424\nstorage:536870912000\nuptime:up 17 weeks, 6 days, 3 hours, 40 minutes\ncpu_cores:16\nmemory:33565159424\nstorage:536870912000\nuptime:up 17 weeks, 6 days, 3 hours, 40 minutes\ncpu_cores:16\nmemory:33565159424\nstorage:536870912000\nuptime:up 17 weeks, 6 days, 3 hours, 40 minutes\ntime_offset:-0.000534712\n"
		// 请将stdout解析为json格式后赋值给 content字段
		// "{\"content\":{\"cpuArch\":\"x86_64\",\"cpuCores\":\"16\",\"hostname\":\"bogon\",\"memory\":\"33565159424\",\"os\":\"CentOS Linux 7 (Core)\",\"storage\":\"536870912000\",\"timeOffset\":\"+0.000669002\",\"uptime\":\"up 17 weeks, 6 days, 5 hours, 4 minutes\"},\"contentId\":\"OS_BASE\",\"dataType\":\"json\",\"metadata\":{\"source\":\"agentname\",\"stepId\":\"none\"},\"tastId\":\"task123\",\"timestamp\":1770880447543}"
		itemCapture.ContentId = strings.ToUpper(OS_BASE)
	case OS_USAGE:
		// 解析stdout，提取cpu_usage、mem_usage、disk_usage、net_loss_rate等信息
		// "{\"content\":{\"cpuUsage\":\"1\",\"diskUsage\":\"30\",\"memoryUsage\":\"8.7\"},\"contentId\":\"OS_USAGE\",\"dataType\":\"json\",\"metadata\":{\"source\":\"agentname\",\"stepId\":\"none\"},\"tastId\":\"task123\",\"timestamp\":1770880520762}"
		itemCapture.ContentId = strings.ToUpper(OS_USAGE)
	case OS_NET_LOSS:
		// 解析stdout，提取pkg_loss_rate信息
		// "{\"content\":{\"pkg_loss_rate\":\"0\"},\"content_id\":\"OS_NET_LOSS\",\"data_type\":\"json\",\"metadata\":{\"source\":\"agentname\",\"step_id\":\"none\"},\"tast_id\":\"task123\",\"timestamp\":1749602096000}"
		itemCapture.ContentId = strings.ToUpper(OS_NET_LOSS)
	}

	// 将item.Content解析为json格式
	var contentMap map[string]string
	if err := json.Unmarshal([]byte(itemCapture.Content), &contentMap); err != nil {
		return nil, err
	}

	// 根据json名称填充到Item，这里是json对应的字段，需要循环加入到一个Item数组
	var items []Item
	for key, value := range contentMap {
		// 根据key的内容查找MetricInfoMap中对应项
		metricInfo, ok := MetricInfoMap[key]
		if !ok {
			continue
		}
		// 将metricInfo中的字段填充到Item中
		items = append(items, Item{
			MetricId:      metricInfo.MetricId,
			MetricName:    metricInfo.MetricName,
			MetricType:    metricInfo.MetricType,
			ObjectType:    metricInfo.ObjectType,
			SubobjectType: metricInfo.SubobjectType, //- 运维对象子类型（按实际对象类型配置，无则填空）
			ObjectId:      itemCapture.TastId,       // 运维对象唯一ID（taskid等）
			SubobjectId:   "",                       //- 运维对象子类型唯一ID（无则填空）
			Value:         value,                    // 指标值（支持数值、字符串）
			Unit:          metricInfo.Unit,          //- 指标单位（无则填空）
			Timestamp:     itemCapture.Timestamp,    // 采集时间戳（毫秒）
			Tags:          map[string]string{},      //- 指标标签（按实际指标配置，无则填空）
		})
	}

	return items, nil
}

func New() plus.Plugin { return &oscapturePlugin{} }
