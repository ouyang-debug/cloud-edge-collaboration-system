//go:build !plugin

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

const (
	K8SINFO        = "k8sinfo"
	K8SNODES       = "k8snodes"
	K8SNS          = "k8sns"
	K8SPODS        = "k8spods"
	K8STOP         = "k8stop"
	K8SSVC         = "k8ssvc"
	K8SCM          = "k8scm"
	K8SDAEMONSET   = "k8sdaemonset"
	K8SDEPLOYMENT  = "k8sdeployment"
	K8SSTATEFULSET = "k8sstatefulset"
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
	//k8sinfo
	"k8sinfo": {
		MetricId:      "k8sinfo",
		MetricName:    "k8sinfo",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_INFO",
		Unit:          "",
	},
	//k8snodes
	// {"AGE":"54d",
	// "CONTAINER-RUNTIME":"docker://26.1.4",
	// "EXTERNAL-IP":"\u003cnone\u003e",
	// "INTERNAL-IP":"10.220.42.155",
	// "KERNEL-VERSION":"4.19.90-89.11.v2401.ky10.aarch64",
	// "NAME":"k8s-master",
	// "OS-IMAGE":"Kylin Linux Advanced Server V10 (Halberd)",
	// "ROLES":"control-plane,master",
	// "STATUS":"Ready",
	// "VERSION":"v1.23.16"}
	"k8snodes_AGE": {
		MetricId:      "k8snodes_AGE",
		MetricName:    "k8sAGE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_NODES",
		Unit:          "",
	},
	"k8snodes_CONTAINER-RUNTIME": {
		MetricId:      "k8snodes_CONTAINER-RUNTIME",
		MetricName:    "k8snodes_CONTAINER-RUNTIME",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_NODES",
		Unit:          "",
	},
	"k8snodes_EXTERNAL-IP": {
		MetricId:      "k8snodes_EXTERNAL-IP",
		MetricName:    "k8snodes_EXTERNAL-IP",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_NODES",
		Unit:          "",
	},
	"k8snodes_INTERNAL-IP": {
		MetricId:      "k8snodes_INTERNAL-IP",
		MetricName:    "k8snodes_INTERNAL-IP",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_NODES",
		Unit:          "",
	},
	"k8snodes_KERNEL-VERSION": {
		MetricId:      "k8snodes_KERNEL-VERSION",
		MetricName:    "k8snodes_KERNEL-VERSION",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_NODES",
		Unit:          "",
	},
	"k8snodes_NAME": {
		MetricId:      "k8snodes_NAME",
		MetricName:    "k8snodes_NAME",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_NODES",
		Unit:          "",
	},
	"k8snodes_OS-IMAGE": {
		MetricId:      "k8snodes_OS-IMAGE",
		MetricName:    "k8snodes_OS-IMAGE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_NODES",
		Unit:          "",
	},
	"k8snodes_ROLES": {
		MetricId:      "k8snodes_ROLES",
		MetricName:    "k8snodes_ROLES",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_NODES",
		Unit:          "",
	},
	"k8snodes_STATUS": {
		MetricId:      "k8snodes_STATUS",
		MetricName:    "k8snodes_STATUS",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_NODES",
		Unit:          "",
	},
	"k8snodes_VERSION": {
		MetricId:      "k8snodes_VERSION",
		MetricName:    "k8snodes_VERSION",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_NODES",
		Unit:          "",
	},
	//k8sns
	// {"AGE":"50d",
	// "NAME":"app",
	// "STATUS":"Active"}
	"k8sns_AGE": {
		MetricId:      "k8sns_AGE",
		MetricName:    "k8sns_AGE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_NS",
		Unit:          "",
	},
	"k8sns_NAME": {
		MetricId:      "k8sns_NAME",
		MetricName:    "k8sns_NAME",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_NS",
		Unit:          "",
	},
	"k8sns_STATUS": {
		MetricId:      "k8sns_STATUS",
		MetricName:    "k8sns_STATUS",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_NS",
		Unit:          "",
	},
	//k8spods
	// {"AGE":"29d",
	// "NAME":"nginx-7c476ff4cb-47vwc",
	// "NAMESPACE":"app",
	// "READY":"1/1",
	// "RESTARTS":"1 (15d ago)",
	// "STATUS":"Running"}
	"k8spods_AGE": {
		MetricId:      "k8spods_AGE",
		MetricName:    "k8spods_AGE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_PODS",
		Unit:          "",
	},
	"k8spods_NAME": {
		MetricId:      "k8spods_NAME",
		MetricName:    "k8spods_NAME",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_PODS",
		Unit:          "",
	},
	"k8spods_NAMESPACE": {
		MetricId:      "k8spods_NAMESPACE",
		MetricName:    "k8spods_NAMESPACE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_PODS",
		Unit:          "",
	},
	"k8spods_READY": {
		MetricId:      "k8spods_READY",
		MetricName:    "k8spods_READY",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_PODS",
		Unit:          "",
	},
	"k8spods_RESTARTS": {
		MetricId:      "k8spods_RESTARTS",
		MetricName:    "k8spods_RESTARTS",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_PODS",
		Unit:          "",
	},
	"k8spods_STATUS": {
		MetricId:      "k8spods_STATUS",
		MetricName:    "k8spods_STATUS",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_PODS",
		Unit:          "",
	},

	// {"NAMESPACE":"app",
	// "NAME":"nginx-7c476ff4cb-47vwc",
	// "CPU(cores)":"1m",
	// "MEMORY(bytes)":"16Mi"}
	"k8stop_STATUS": {
		MetricId:      "k8stop_STATUS",
		MetricName:    "k8stop_STATUS",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_TOP",
		Unit:          "",
	},
	"k8stop_NAME": {
		MetricId:      "k8stop_NAME",
		MetricName:    "k8stop_NAME",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_TOP",
		Unit:          "",
	},
	"k8stop_CPU(cores)": {
		MetricId:      "k8stop_CPU(cores)",
		MetricName:    "k8stop_CPU(cores)",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_TOP",
		Unit:          "",
	},
	"k8stop_MEMORY(bytes)": {
		MetricId:      "k8stop_MEMORY(bytes)",
		MetricName:    "k8stop_MEMORY(bytes)",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_TOP",
		Unit:          "",
	},
	//k8ssvc
	// {"AGE":"29d",
	// "CLUSTER-IP":"10.96.100.181",
	// "EXTERNAL-IP":"\u003cnone\u003e",
	// "NAME":"nginx",
	// "NAMESPACE":"app",
	// "PORT(S)":"80/TCP",
	// "TYPE":"ClusterIP"}
	"k8ssvc_AGE": {
		MetricId:      "k8ssvc_AGE",
		MetricName:    "k8ssvc_AGE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_SVC",
		Unit:          "",
	},
	"k8ssvc_CLUSTER-IP": {
		MetricId:      "k8ssvc_CLUSTER-IP",
		MetricName:    "k8ssvc_CLUSTER-IP",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_SVC",
		Unit:          "",
	},
	"k8ssvc_EXTERNAL-IP": {
		MetricId:      "k8ssvc_EXTERNAL-IP",
		MetricName:    "k8ssvc_EXTERNAL-IP",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_SVC",
		Unit:          "",
	},
	"k8ssvc_NAME": {
		MetricId:      "k8ssvc_NAME",
		MetricName:    "k8ssvc_NAME",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_SVC",
		Unit:          "",
	},
	"k8ssvc_NAMESPACE": {
		MetricId:      "k8ssvc_NAMESPACE",
		MetricName:    "k8ssvc_NAMESPACE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_SVC",
		Unit:          "",
	},
	"k8ssvc_PORT(S)": {
		MetricId:      "k8ssvc_PORT(S)",
		MetricName:    "k8ssvc_PORT(S)",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_SVC",
		Unit:          "",
	},
	"k8ssvc_TYPE": {
		MetricId:      "k8ssvc_TYPE",
		MetricName:    "k8ssvc_TYPE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_SVC",
		Unit:          "",
	},
	//k8scm
	// {"AGE":"50d",
	// "DATA":"1",
	// "NAME":"kube-root-ca.crt",
	// "NAMESPACE":"app"}
	"k8scm_AGE": {
		MetricId:      "k8scm_AGE",
		MetricName:    "k8scm_AGE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_CM",
		Unit:          "",
	},
	"k8scm_DATA": {
		MetricId:      "k8scm_DATA",
		MetricName:    "k8scm_DATA",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_CM",
		Unit:          "",
	},
	"k8scm_NAME": {
		MetricId:      "k8scm_NAME",
		MetricName:    "k8scm_NAME",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_CM",
		Unit:          "",
	},
	"k8scm_NAMESPACE": {
		MetricId:      "k8scm_NAMESPACE",
		MetricName:    "k8scm_NAMESPACE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_CM",
		Unit:          "",
	},
	//k8sdaemonset
	// {"AGE":"54d",
	// "AVAILABLE":"3",
	// "CURRENT":"4",
	// "DESIRED":"4",
	// "NAME":"calico-node",
	// "NAMESPACE":"kube-system",
	// "NODE SELECTOR":"kubernetes.io/os=linux",
	// "READY":"3",
	// "UP-TO-DATE":"2"}
	"k8sdaemonset_AGE": {
		MetricId:      "k8sdaemonset_AGE",
		MetricName:    "k8sdaemonset_AGE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DAEMONSET",
		Unit:          "",
	},
	"k8sdaemonset_AVAILABLE": {
		MetricId:      "k8sdaemonset_AVAILABLE",
		MetricName:    "k8sdaemonset_AVAILABLE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DAEMONSET",
		Unit:          "",
	},
	"k8sdaemonset_CURRENT": {
		MetricId:      "k8sdaemonset_CURRENT",
		MetricName:    "k8sdaemonset_CURRENT",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DAEMONSET",
		Unit:          "",
	},
	"k8sdaemonset_DESIRED": {
		MetricId:      "k8sdaemonset_DESIRED",
		MetricName:    "k8sdaemonset_DESIRED",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DAEMONSET",
		Unit:          "",
	},
	"k8sdaemonset_NAME": {
		MetricId:      "k8sdaemonset_NAME",
		MetricName:    "k8sdaemonset_NAME",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DAEMONSET",
		Unit:          "",
	},
	"k8sdaemonset_NAMESPACE": {
		MetricId:      "k8sdaemonset_NAMESPACE",
		MetricName:    "k8sdaemonset_NAMESPACE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DAEMONSET",
		Unit:          "",
	},
	"k8sdaemonset_NODE SELECTOR": {
		MetricId:      "k8sdaemonset_NODE-SELECTOR",
		MetricName:    "k8sdaemonset_NODE-SELECTOR",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DAEMONSET",
		Unit:          "",
	},
	"k8sdaemonset_READY": {
		MetricId:      "k8sdaemonset_READY",
		MetricName:    "k8sdaemonset_READY",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DAEMONSET",
		Unit:          "",
	},
	"k8sdaemonset_UP-TO-DATE": {
		MetricId:      "k8sdaemonset_UP-TO-DATE",
		MetricName:    "k8sdaemonset_UP-TO-DATE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DAEMONSET",
		Unit:          "",
	},
	//k8sdeployment
	// {"AGE":"29d",
	// "AVAILABLE":"1",
	// "NAME":"nginx",
	// "NAMESPACE":"app",
	// "READY":"1/1",
	// "UP-TO-DATE":"1"}

	"k8sdeployment_AGE": {
		MetricId:      "k8sdeployment_AGE",
		MetricName:    "k8sdeployment_AGE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DEPLOYMENT",
		Unit:          "",
	},
	"k8sdeployment_AVAILABLE": {
		MetricId:      "k8sdeployment_AVAILABLE",
		MetricName:    "k8sdeployment_AVAILABLE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DEPLOYMENT",
		Unit:          "",
	},
	"k8sdeployment_NAME": {
		MetricId:      "k8sdeployment_NAME",
		MetricName:    "k8sdeployment_NAME",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DEPLOYMENT",
		Unit:          "",
	},
	"k8sdeployment_NAMESPACE": {
		MetricId:      "k8sdeployment_NAMESPACE",
		MetricName:    "k8sdeployment_NAMESPACE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DEPLOYMENT",
		Unit:          "",
	},
	"k8sdeployment_READY": {
		MetricId:      "k8sdeployment_READY",
		MetricName:    "k8sdeployment_READY",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DEPLOYMENT",
		Unit:          "",
	},
	"k8sdeployment_UP-TO-DATE": {
		MetricId:      "k8sdeployment_UP-TO-DATE",
		MetricName:    "k8sdeployment_UP-TO-DATE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_DEPLOYMENT",
		Unit:          "",
	},
	// {"AGE":"29d",
	// "AVAILABLE":"1",
	// "NAME":"nginx",
	// "NAMESPACE":"app",
	// "READY":"1/1",
	// "UP-TO-DATE":"1"}
	"k8sstatefulset_AGE": {
		MetricId:      "k8sstatefulset_AGE",
		MetricName:    "k8sstatefulset_AGE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_STATEFULSET",
		Unit:          "",
	},
	"k8sstatefulset_AVAILABLE": {
		MetricId:      "k8sstatefulset_AVAILABLE",
		MetricName:    "k8sstatefulset_AVAILABLE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_STATEFULSET",
		Unit:          "",
	},
	"k8sstatefulset_NAME": {
		MetricId:      "k8sstatefulset_NAME",
		MetricName:    "k8sstatefulset_NAME",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_STATEFULSET",
		Unit:          "",
	},
	"k8sstatefulset_NAMESPACE": {
		MetricId:      "k8sstatefulset_NAMESPACE",
		MetricName:    "k8sstatefulset_NAMESPACE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_STATEFULSET",
		Unit:          "",
	},
	"k8sstatefulset_READY": {
		MetricId:      "k8sstatefulset_READY",
		MetricName:    "k8sstatefulset_READY",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_STATEFULSET",
		Unit:          "",
	},
	"k8sstatefulset_UP-TO-DATE": {
		MetricId:      "k8sstatefulset_UP-TO-DATE",
		MetricName:    "k8sstatefulset_UP-TO-DATE",
		MetricType:    "txt",
		ObjectType:    "k8s",
		SubobjectType: "K8S_STATEFULSET",
		Unit:          "",
	},
}

// Table 定义顶层JSON结构，包含count和rows
type Table struct {
	Count int    `json:"count"` // 行数
	Rows  []Item `json:"rows"`  // 行数据
}
type k8scapturePlugin struct{}

func (p *k8scapturePlugin) Name() string    { return PluginName }
func (p *k8scapturePlugin) Version() string { return PluginVersion }
func (p *k8scapturePlugin) Description() string {
	return "Kubernetes command capture plugin with remote execution support"
}
func (p *k8scapturePlugin) Initialize(config string) error { return nil }
func (p *k8scapturePlugin) Shutdown() error                { return nil }

func (p *k8scapturePlugin) Execute(input map[string]string) (map[string]string, error) {
	host := input["target_host"]
	user := input["target_user"]
	pass := input["target_password"]
	key := input["target_key"]
	portStr := input["target_port"]
	cmdStrIn := input["cmd"]
	taskID := input["task_id"]

	if cmdStrIn == "" {
		return map[string]string{"stderr": "empty cmd"}, nil
	}

	// $ echo docker info|base64 -w 0
	// ZG9ja2VyIGluZm8K
	// $echo cat /etc/docker/daemon.json|base64 -w 0
	// Y2F0IC9ldGMvZG9ja2VyL2RhZW1vbi5qc29uCg==
	// $echo systemctl --no-pager cat docker|base64 -w 0
	// c3lzdGVtY3RsIC0tbm8tcGFnZXIgY2F0IGRvY2tlcgo=
	// $echo docker system df|base64 -w 0
	// ZG9ja2VyIHN5c3RlbSBkZgo=

	// $ echo 'docker ps -a --format "{\"id\":\"{{.ID}}\",\"name\":\"{{.Names}}\",\"image\":\"{{.Image}}\",\"status\":\"{{.Status}}\",\"command\":\"{{.Command}}\",\"create\":\"{{.CreatedAt}}\",\"ports\":\"{{.Ports}}\"}"'|base64 -w 0
	// ZG9ja2VyIHBzIC1hIC0tZm9ybWF0ICJ7XCJpZFwiOlwie3suSUR9fVwiLFwibmFtZVwiOlwie3suTmFtZXN9fVwiLFwiaW1hZ2VcIjpcInt7LkltYWdlfX1cIixcInN0YXR1c1wiOlwie3suU3RhdHVzfX1cIixcImNvbW1hbmRcIjpcInt7LkNvbW1hbmR9fVwiLFwiY3JlYXRlXCI6XCJ7ey5DcmVhdGVkQXR9fVwiLFwicG9ydHNcIjpcInt7LlBvcnRzfX1cIn0iCg==
	// $ cat /d/a.txt
	// docker ps -a | grep -v CONTAINER | awk '{ids=ids " " $1} END {print substr(ids,2)}'|xargs docker inspect
	// $ cat /d/a.txt |base64 -w 0
	// ZG9ja2VyIHBzIC1hIHwgZ3JlcCAtdiBDT05UQUlORVIgfCBhd2sgJ3tpZHM9aWRzICIgIiAkMX0gRU5EIHtwcmludCBzdWJzdHIoaWRzLDIpfSd8eGFyZ3MgZG9ja2VyIGluc3BlY3QK
	// $ echo 'docker stats -a --no-stream --format "{\"id\":\"{{.ID}}\",\"container\":\"{{.Name}}\",\"cpuPercent\":\"{{.CPUPerc}}\",\"memUsage\":\"{{.MemUsage}}\",\"memPercent\":\"{{.MemPerc}}\",\"netIo\":\"{{.NetIO}}\"},\"blockIo\":\"{{.BlockIO}}\"},\"pids\":\"{{.PIDs}}\"}"'|base64 -w 0
	// ZG9ja2VyIHN0YXRzIC1hIC0tbm8tc3RyZWFtIC0tZm9ybWF0ICJ7XCJpZFwiOlwie3suSUR9fVwiLFwiY29udGFpbmVyXCI6XCJ7ey5OYW1lfX1cIixcImNwdVBlcmNlbnRcIjpcInt7LkNQVVBlcmN9fVwiLFwibWVtVXNhZ2VcIjpcInt7Lk1lbVVzYWdlfX1cIixcIm1lbVBlcmNlbnRcIjpcInt7Lk1lbVBlcmN9fVwiLFwibmV0SW9cIjpcInt7Lk5ldElPfX1cIn0sXCJibG9ja0lvXCI6XCJ7ey5CbG9ja0lPfX1cIn0sXCJwaWRzXCI6XCJ7ey5QSURzfX1cIn0iCg==

	// Parse command format
	cmdStr := strings.TrimSpace(cmdStrIn)
	cmdType := ""
	cmdParts := strings.SplitN(cmdStr, " ", 3)
	if len(cmdParts) >= 1 {
		cmdType = strings.TrimSpace(cmdParts[0])

		switch cmdType {
		case K8SINFO:
			cmdStr = "echo " +
				"a3ViZWN0bCAgY2x1c3Rlci1pbmZvCg==" +
				"|base64 -d |sh"
		case K8SNODES:
			cmdStr = "echo " +
				"a3ViZWN0bCBnZXQgbm9kZXMgLUEgLS1uby1oZWFkZXJzIC1vIHdpZGUgfCBhd2sgJwpCRUdJTiB7IHByaW50ICJbIjsgZmlyc3Q9MSB9CnsKICAgIG5hbWU9JDE7IHN0YXR1cz0kMjsgcm9sZXM9JDM7IGFnZT0kNDsgdmVyc2lvbj0kNTsgaW50ZXJuYWxpcD0kNjsgZXh0ZXJuYWxpcD0kNzsgb3NpbWFnZT0kODsgIGNvbnRhaW5lcnJ1bnRpbWU9JE5GOyBrZXJuZWx2ZXJzaW9uPSQoTkYtMSk7CgogICAgIyDmj5Dlj5YgT1MtSU1BR0Ug5a6M5pW05o+P6L+w77yI5aaCIEt5bGluIExpbnV4IEFkdmFuY2VkIFNlcnZlciBWMTAgKEhhbGJlcmQp77yJCiAgICBvc2ltYWdlX2Z1bGw9IiI7CiAgICBmb3IoaT04OyBpPD1ORi0yOyBpKyspIG9zaW1hZ2VfZnVsbCA9IG9zaW1hZ2VfZnVsbCAkaSAiICI7CiAgICBnc3ViKC8gJC8sICIiLCBvc2ltYWdlX2Z1bGwpOwoKICAgIGlmICghZmlyc3QpIHByaW50ICIsIjsKICAgIGZpcnN0PTA7CiAgICBwcmludGYgIntcIk5BTUVcIjpcIiVzXCIsXCJTVEFUVVNcIjpcIiVzXCIsXCJST0xFU1wiOlwiJXNcIixcIkFHRVwiOlwiJXNcIixcIlZFUlNJT05cIjpcIiVzXCIsXCJJTlRFUk5BTC1JUFwiOlwiJXNcIixcIkVYVEVSTkFMLUlQXCI6XCIlc1wiLFwiT1MtSU1BR0VcIjpcIiVzXCIsXCJLRVJORUwtVkVSU0lPTlwiOlwiJXNcIixcIkNPTlRBSU5FUi1SVU5USU1FXCI6XCIlc1wifSIsCiAgICAgICAgbmFtZSxzdGF0dXMscm9sZXMsYWdlLHZlcnNpb24saW50ZXJuYWxpcCxleHRlcm5hbGlwLG9zaW1hZ2VfZnVsbCxrZXJuZWx2ZXJzaW9uLGNvbnRhaW5lcnJ1bnRpbWU7Cn0KRU5EIHsgcHJpbnQgIlxuXSIgfScK" +
				"|base64 -d |sh"
		case K8SNS:
			cmdStr = "echo " +
				"a3ViZWN0bCBnZXQgbnMgLS1uby1oZWFkZXJzIHwgYXdrIC1GJ1tbOnNwYWNlOl1dKycgJ0JFR0lOIHsgcHJpbnQgIlsiOyBmaXJzdD0xIH0geyBpZiAoIWZpcnN0KSBwcmludCAiLCI7IGZpcnN0PTA7IHByaW50ZiAie1wiTkFNRVwiOlwiJXNcIixcIlNUQVRVU1wiOlwiJXNcIixcIkFHRVwiOlwiJXNcIn0iLCAkMSwgJDIsICQzIH0gRU5EIHsgcHJpbnQgIlxuXSIgfScK" +
				"|base64 -d |sh"
		case K8SPODS:
			cmdStr = "echo " +
				"a3ViZWN0bCBnZXQgcG9kcyAtQSAtLW5vLWhlYWRlcnMgfCBhd2sgJwpCRUdJTiB7IHByaW50ICJbIjsgZmlyc3Q9MSB9CnsKICAgIG5zPSQxOyBuYW1lPSQyOyByZWFkeT0kMzsgc3RhdHVzPSQ0OyByZXN0YXJ0cz0kNTsgYWdlPSRORjsKCiAgICAjIOaPkOWPliBSRVNUQVJUUyDlrozmlbTmj4/ov7DvvIjlpoIgMSAoM2Q2aCBhZ28p77yJCiAgICByZXN0YXJ0c19mdWxsPSIiOwogICAgZm9yKGk9NTsgaTw9TkYtMTsgaSsrKSByZXN0YXJ0c19mdWxsID0gcmVzdGFydHNfZnVsbCAkaSAiICI7CiAgICBnc3ViKC8gJC8sICIiLCByZXN0YXJ0c19mdWxsKTsKCiAgICBpZiAoIWZpcnN0KSBwcmludCAiLCI7CiAgICBmaXJzdD0wOwogICAgcHJpbnRmICJ7XCJOQU1FU1BBQ0VcIjpcIiVzXCIsXCJOQU1FXCI6XCIlc1wiLFwiUkVBRFlcIjpcIiVzXCIsXCJTVEFUVVNcIjpcIiVzXCIsXCJSRVNUQVJUU1wiOlwiJXNcIixcIkFHRVwiOlwiJXNcIn0iLAogICAgICAgIG5zLCBuYW1lLCByZWFkeSwgc3RhdHVzLCByZXN0YXJ0c19mdWxsLCBhZ2U7Cn0KRU5EIHsgcHJpbnQgIlxuXSIgfScK" +
				"|base64 -d |sh"
		case K8STOP:
			cmdStr = "echo " +
				"a3ViZWN0bCB0b3AgcG9kIC1BIC0tbm8taGVhZGVyc3wgYXdrIC1GJ1tbOnNwYWNlOl1dKycgJ0JFR0lOIHsgcHJpbnQgIlsiOyBmaXJzdD0xIH0geyBpZiAoIWZpcnN0KSBwcmludCAiLCI7IGZpcnN0PTA7IHByaW50ZiAie1wiTkFNRVNQQUNFXCI6XCIlc1wiLFwiTkFNRVwiOlwiJXNcIixcIkNQVShjb3JlcylcIjpcIiVzXCIsXCJNRU1PUlkoYnl0ZXMpXCI6XCIlc1wifSIsICQxLCAkMiwgJDMsICQ0IH0gRU5EIHsgcHJpbnQgIlxuXSIgfScK" +
				"|base64 -d |sh"
		case K8SSVC:
			cmdStr = "echo " +
				"a3ViZWN0bCBnZXQgc3ZjIC1BIC0tbm8taGVhZGVycyB8IGF3ayAtRidbWzpzcGFjZTpdXSsnICdCRUdJTiB7IHByaW50ICJbIjsgZmlyc3Q9MSB9IHsgaWYgKCFmaXJzdCkgcHJpbnQgIiwiOyBmaXJzdD0wOyBwcmludGYgIntcIk5BTUVTUEFDRVwiOlwiJXNcIixcIk5BTUVcIjpcIiVzXCIsXCJUWVBFXCI6XCIlc1wiLFwiQ0xVU1RFUi1JUFwiOlwiJXNcIixcIkVYVEVSTkFMLUlQXCI6XCIlc1wiLFwiUE9SVChTKVwiOlwiJXNcIixcIkFHRVwiOlwiJXNcIn0iLCAkMSwgJDIsICQzLCAkNCwgJDUsICQ2LCAkNyB9IEVORCB7IHByaW50ICJcbl0iIH0nCg==" +
				"|base64 -d |sh"
		case K8SCM:
			cmdStr = "echo " +
				"a3ViZWN0bCBnZXQgY20gLUEgLS1uby1oZWFkZXJzIHwgYXdrIC1GJ1tbOnNwYWNlOl1dKycgJ0JFR0lOIHsgcHJpbnQgIlsiOyBmaXJzdD0xIH0geyBpZiAoIWZpcnN0KSBwcmludCAiLCI7IGZpcnN0PTA7IHByaW50ZiAie1wiTkFNRVNQQUNFXCI6XCIlc1wiLFwiTkFNRVwiOlwiJXNcIixcIkRBVEFcIjpcIiVzXCIsXCJBR0VcIjpcIiVzXCJ9IiwgJDEsICQyLCAkMywgJDQgfSBFTkQgeyBwcmludCAiXG5dIiB9Jwo=" +
				"|base64 -d |sh"
		case K8SDAEMONSET:
			cmdStr = "echo " +
				"a3ViZWN0bCBnZXQgZGFlbW9uc2V0IC1BIC0tbm8taGVhZGVyc3wgYXdrIC1GJ1tbOnNwYWNlOl1dKycgJ0JFR0lOIHsgcHJpbnQgIlsiOyBmaXJzdD0xIH0geyBpZiAoIWZpcnN0KSBwcmludCAiLCI7IGZpcnN0PTA7IHByaW50ZiAie1wiTkFNRVNQQUNFXCI6XCIlc1wiLFwiTkFNRVwiOlwiJXNcIixcIkRFU0lSRURcIjpcIiVzXCIsXCJDVVJSRU5UXCI6XCIlc1wiLFwiUkVBRFlcIjpcIiVzXCIsXCJVUC1UTy1EQVRFXCI6XCIlc1wiLFwiQVZBSUxBQkxFXCI6XCIlc1wiLFwiTk9ERSBTRUxFQ1RPUlwiOlwiJXNcIixcIkFHRVwiOlwiJXNcIn0iLCAkMSwgJDIsICQzLCAkNCwgJDUsICQ2LCAkNywgJDgsICQ5IH0gRU5EIHsgcHJpbnQgIlxuXSIgfScK" +
				"|base64 -d |sh"
		case K8SDEPLOYMENT:
			cmdStr = "echo " +
				"a3ViZWN0bCBnZXQgZGVwbG95bWVudCAtQSAtLW5vLWhlYWRlcnN8IGF3ayAtRidbWzpzcGFjZTpdXSsnICdCRUdJTiB7IHByaW50ICJbIjsgZmlyc3Q9MSB9IHsgaWYgKCFmaXJzdCkgcHJpbnQgIiwiOyBmaXJzdD0wOyBwcmludGYgIntcIk5BTUVTUEFDRVwiOlwiJXNcIixcIk5BTUVcIjpcIiVzXCIsXCJSRUFEWVwiOlwiJXNcIixcIlVQLVRPLURBVEVcIjpcIiVzXCIsXCJBVkFJTEFCTEVcIjpcIiVzXCIsXCJBR0VcIjpcIiVzXCJ9IiwgJDEsICQyLCAkMywgJDQsICQ1LCAkNiB9IEVORCB7IHByaW50ICJcbl0iIH0nCg==" +
				"|base64 -d |sh"
		case K8SSTATEFULSET:
			cmdStr = "echo " +
				"a3ViZWN0bCBnZXQgc3RhdGVmdWxzZXQgLUEgLS1uby1oZWFkZXJzfCBhd2sgLUYnW1s6c3BhY2U6XV0rJyAnQkVHSU4geyBwcmludCAiWyI7IGZpcnN0PTEgfSB7IGlmICghZmlyc3QpIHByaW50ICIsIjsgZmlyc3Q9MDsgcHJpbnRmICJ7XCJOQU1FU1BBQ0VcIjpcIiVzXCIsXCJOQU1FXCI6XCIlc1wiLFwiUkVBRFlcIjpcIiVzXCIsXCJBR0VcIjpcIiVzXCJ9IiwgJDEsICQyLCAkMywgJDQgfSBFTkQgeyBwcmludCAiXG5dIiB9Jwo=" +
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
	case K8SINFO:
		// 解析stdout
		// k8s info
		RowsCapture, err = parseK8SInfo(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	case K8SNODES:
		// 解析stdout
		// k8s nodes
		RowsCapture, err = parseK8SNODES(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	case K8SNS:
		// 解析stdout
		// k8s ns
		RowsCapture, err = parseK8SNS(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	case K8SPODS:

		//解析stdout
		// k8s pods
		// TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
		// Images          4         3         1.395GB   481.1MB (34%)
		// Containers      3         3         5.827MB   0B (0%)
		// Local Volumes   0         0         0B        0B
		// Build Cache     0         0         0B        0B
		RowsCapture, err = parseK8SPODS(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	case K8STOP:
		RowsCapture, err = parseK8STOP(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	case K8SSVC:

		//解析stdout
		// {"id":"afb34abbe32e","name":"kuboard3","image":"harbor.tpridmp.com.cn:8081/kuboard/eipwork/kuboard:v3.5.2.7-arm","status":"Up 3 days","command":""/entrypoint.sh"","create":"2026-01-21 09:56:00 +0800 CST","ports":"443/tcp, 0.0.0.0:10081->10081/tcp, :::10081->10081/tcp, 0.0.0.0:8082->80/tcp, :::8082->80/tcp"}
		// {"id":"72289808fa0b","name":"kubepi","image":"harbor.tpridmp.com.cn:8081/other/1panel/kubepi-arm:v1.9.0","status":"Up 3 days","command":""tini -g -- kubepi-s…"","create":"2026-01-21 08:53:32 +0800 CST","ports":"0.0.0.0:8081->80/tcp, :::8081->80/tcp"}
		// {"id":"f5ac69cb32a8","name":"kuboard","image":"harbor.tpridmp.com.cn:8081/kuboard/eipwork/kuboard:v4.0.1.0-arm","status":"Up 3 days","command":""/entry-point.sh"","create":"2026-01-12 14:33:57 +0800 CST","ports":"8081/tcp, 0.0.0.0:8080->80/tcp, :::8080->80/tcp"}

		RowsCapture, err = parseK8SSVC(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	case K8SCM:
		// 解析stdout
		// {"id":"afb34abbe32e","container":"kuboard3","cpuPercent":"2.29%","memUsage":"659.2MiB / 14.48GiB","memPercent":"4.45%","netIo":"6.78kB / 108B"},"blockIo":"722MB / 13.5GB"},"pids":"64"}
		// {"id":"72289808fa0b","container":"kubepi","cpuPercent":"0.00%","memUsage":"39.19MiB / 14.48GiB","memPercent":"0.26%","netIo":"6.78kB / 0B"},"blockIo":"226MB / 0B"},"pids":"19"}
		// {"id":"f5ac69cb32a8","container":"kuboard","cpuPercent":"0.08%","memUsage":"495.3MiB / 14.48GiB","memPercent":"3.34%","netIo":"608MB / 449MB"},"blockIo":"42.7MB / 0B"},"pids":"76"}

		RowsCapture, err = parseK8SCM(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	case K8SDAEMONSET:
		// 解析stdout
		// k8s daemonset
		RowsCapture, err = parseK8SDAEMONSET(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}

	case K8SDEPLOYMENT:
		// 解析stdout
		// k8s deployment
		RowsCapture, err = parseK8SDEPLOYMENT(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}

	case K8SSTATEFULSET:
		// 解析stdout
		// k8s statefulset
		RowsCapture, err = parseK8SSTATEFULSET(taskID, stdout)
		if err != nil {
			return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
		}
	default:
		// 先生成ItemCapture结构体
		item := ItemCapture{
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

func (p *k8scapturePlugin) ExecuteWithProgress(taskID string, input map[string]string, reporter plus.ProgressReporter) (map[string]string, error) {
	// K8s capture plugin is single-step; we simply run it and report completion
	input["task_id"] = taskID
	out, err := p.Execute(input)
	if reporter != nil {
		reporter.OnProgress(taskID, "k8scapture", 1, 1, "")
		reporter.OnCompleted(taskID, "k8scapture", err == nil, "")
	}
	return out, err
}

// parseK8SInfo 解析 k8s info 输出，返回结构化的 JSON
func parseK8SInfo(taskID, output string) ([]ItemCapture, error) {
	Rows := []ItemCapture{}
	// 先生成ItemCapture结构体
	item := ItemCapture{
		TastId:    "req-20260210123456-789",
		DataType:  "txt",
		ContentId: "CONTENT_001",
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

	item.ContentId = strings.ToLower(K8SINFO)
	Rows = append(Rows, item)

	return Rows, nil
}

// parseK8SNODES 解析 k8s nodes 输出，返回结构化的 JSON
func parseK8SNODES(taskID, output string) ([]ItemCapture, error) {
	// [
	// {"NAME":"k8s-master","STATUS":"Ready","ROLES":"control-plane,master","AGE":"42d","VERSION":"v1.23.16","INTERNAL-IP":"10.220.42.155","EXTERNAL-IP":"<none>","OS-IMAGE":"Kylin Linux Advanced Server V10 (Halberd)","KERNEL-VERSION":"4.19.90-89.11.v2401.ky10.aarch64","CONTAINER-RUNTIME":"docker://26.1.4"},
	// {"NAME":"k8s-node1","STATUS":"Ready","ROLES":"<none>","AGE":"42d","VERSION":"v1.23.16","INTERNAL-IP":"10.220.42.156","EXTERNAL-IP":"<none>","OS-IMAGE":"Kylin Linux Advanced Server V10 (Halberd)","KERNEL-VERSION":"4.19.90-89.11.v2401.ky10.aarch64","CONTAINER-RUNTIME":"docker://26.1.4"}
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
			DataType:  "json",
			ContentId: strings.ToLower(K8SNODES),
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

		if name, ok := container["NAME"].(string); ok && name != "" {
			item.ContentId = strings.ToLower(K8SNODES) + "_" + name
		}

		// 如果提供了taskID，使用taskID
		if taskID != "" {
			item.TastId = taskID
		}

		Rows = append(Rows, item)
	}

	return Rows, nil

}

// parseK8SNS 解析 k8s ns 输出，返回结构化的 JSON
func parseK8SNS(taskID, output string) ([]ItemCapture, error) {

	// [
	// {"NAME":"app","STATUS":"Active","AGE":"38d"},
	// {"NAME":"default","STATUS":"Active","AGE":"42d"},
	// {"NAME":"kube-node-lease","STATUS":"Active","AGE":"42d"},
	// {"NAME":"kube-public","STATUS":"Active","AGE":"42d"},
	// {"NAME":"kube-system","STATUS":"Active","AGE":"42d"},
	// {"NAME":"kuboard","STATUS":"Active","AGE":"41d"}
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
			DataType:  "json",
			ContentId: strings.ToLower(K8SNS),
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

		if name, ok := container["NAME"].(string); ok && name != "" {
			item.ContentId = strings.ToLower(K8SNS) + "_" + name
		}

		// 如果提供了taskID，使用taskID
		if taskID != "" {
			item.TastId = taskID
		}

		Rows = append(Rows, item)
	}

	return Rows, nil
}

// parseK8SPODS 解析 k8s pods 输出，返回结构化的 JSON
func parseK8SPODS(taskID, output string) ([]ItemCapture, error) {

	// [
	// {"NAMESPACE":"app","NAME":"nginx-7c476ff4cb-47vwc","READY":"1/1","STATUS":"Running","RESTARTS":"1 (3d7h ago)","AGE":"17d"},
	// {"NAMESPACE":"app","NAME":"nginx-local-6659969b9-hbgfm","READY":"1/1","STATUS":"Running","RESTARTS":"1 (3d7h ago)","AGE":"17d"},
	// {"NAMESPACE":"app","NAME":"nginx-test-54bd5fc546-5r27h","READY":"1/1","STATUS":"Running","RESTARTS":"1 (3d7h ago)","AGE":"17d"},
	// {"NAMESPACE":"app","NAME":"redis-0","READY":"1/1","STATUS":"Running","RESTARTS":"1 (3d7h ago)","AGE":"29d"},
	// {"NAMESPACE":"kuboard","NAME":"metrics-scraper-7c6867b898-qsmrn","READY":"1/1","STATUS":"Running","RESTARTS":"1 (3d7h ago)","AGE":"29d"}
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
			DataType:  "json",
			ContentId: strings.ToLower(K8SPODS),
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

		if name, ok := container["NAME"].(string); ok && name != "" {
			item.ContentId = strings.ToLower(K8SPODS) + "_" + name
		}

		// 如果提供了taskID，使用taskID
		if taskID != "" {
			item.TastId = taskID
		}

		Rows = append(Rows, item)
	}

	return Rows, nil
}

// parseK8STOP 解析 k8s top pod 输出，返回结构化的 JSON
func parseK8STOP(taskID, output string) ([]ItemCapture, error) {

	// [
	// {"NAMESPACE":"app","NAME":"nginx-7c476ff4cb-47vwc","CPU(cores)":"1m","MEMORY(bytes)":"16Mi"},
	// {"NAMESPACE":"app","NAME":"nginx-local-6659969b9-hbgfm","CPU(cores)":"1m","MEMORY(bytes)":"11Mi"},
	// {"NAMESPACE":"app","NAME":"nginx-test-54bd5fc546-5r27h","CPU(cores)":"1m","MEMORY(bytes)":"11Mi"},
	// {"NAMESPACE":"app","NAME":"redis-0","CPU(cores)":"2m","MEMORY(bytes)":"5Mi"},
	// {"NAMESPACE":"default","NAME":"nginx-157","CPU(cores)":"0m","MEMORY(bytes)":"3Mi"},
	// {"NAMESPACE":"kube-system","NAME":"calico-kube-controllers-7b95df6c8c-w87m2","CPU(cores)":"2m","MEMORY(bytes)":"33Mi"},
	// {"NAMESPACE":"kube-system","NAME":"calico-node-4wqs6","CPU(cores)":"17m","MEMORY(bytes)":"159Mi"},
	// {"NAMESPACE":"kube-system","NAME":"calico-node-dqg8t","CPU(cores)":"24m","MEMORY(bytes)":"165Mi"},
	// {"NAMESPACE":"kube-system","NAME":"kube-proxy-v47tl","CPU(cores)":"11m","MEMORY(bytes)":"21Mi"},
	// {"NAMESPACE":"kube-system","NAME":"kube-scheduler-k8s-master","CPU(cores)":"3m","MEMORY(bytes)":"21Mi"},
	// {"NAMESPACE":"kube-system","NAME":"metrics-server-75d6d54ddd-lxc9d","CPU(cores)":"4m","MEMORY(bytes)":"20Mi"}
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
			DataType:  "json",
			ContentId: strings.ToLower(K8STOP),
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

		if name, ok := container["NAME"].(string); ok && name != "" {
			item.ContentId = strings.ToLower(K8STOP) + "_" + name
		}

		// 如果提供了taskID，使用taskID
		if taskID != "" {
			item.TastId = taskID
		}

		Rows = append(Rows, item)
	}

	return Rows, nil
}

// parseK8SSVC 解析 k8s svc 输出，返回结构化的 JSON
func parseK8SSVC(taskID, output string) ([]ItemCapture, error) {
	// [
	// {"NAMESPACE":"app","NAME":"nginx","TYPE":"ClusterIP","CLUSTER-IP":"10.96.100.181","EXTERNAL-IP":"<none>","PORT(S)":"80/TCP","AGE":"17d"},
	// {"NAMESPACE":"app","NAME":"nginx-local","TYPE":"ClusterIP","CLUSTER-IP":"10.96.100.138","EXTERNAL-IP":"<none>","PORT(S)":"80/TCP","AGE":"17d"},
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
			DataType:  "json",
			ContentId: strings.ToLower(K8SSVC),
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

		if name, ok := container["NAME"].(string); ok && name != "" {
			item.ContentId = strings.ToLower(K8SSVC) + "_" + name
		}

		// 如果提供了taskID，使用taskID
		if taskID != "" {
			item.TastId = taskID
		}

		Rows = append(Rows, item)
	}

	return Rows, nil
}

// parseK8SCM 解析 k8s cm 输出，返回结构化的 JSON
func parseK8SCM(taskID, output string) ([]ItemCapture, error) {

	// output输出的内容格式如下，请解析到item中，并将Config.Hostname字段的值作为ContentId
	// [
	// {"NAMESPACE":"app","NAME":"kube-root-ca.crt","DATA":"1","AGE":"38d"},
	// {"NAMESPACE":"app","NAME":"redis-conf","DATA":"1","AGE":"29d"},
	// {"NAMESPACE":"default","NAME":"kube-root-ca.crt","DATA":"1","AGE":"42d"},
	// {"NAMESPACE":"kube-node-lease","NAME":"kube-root-ca.crt","DATA":"1","AGE":"42d"},
	// {"NAMESPACE":"kube-system","NAME":"kube-root-ca.crt","DATA":"1","AGE":"42d"},
	// {"NAMESPACE":"kube-system","NAME":"kubeadm-config","DATA":"1","AGE":"42d"},
	// {"NAMESPACE":"kube-system","NAME":"kubelet-config-1.23","DATA":"1","AGE":"42d"},
	// {"NAMESPACE":"kuboard","NAME":"kube-root-ca.crt","DATA":"1","AGE":"41d"}
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
			DataType:  "json",
			ContentId: strings.ToLower(K8SCM),
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

		if name, ok := container["NAME"].(string); ok && name != "" {
			item.ContentId = strings.ToLower(K8SCM) + "_" + name
		}

		// 如果提供了taskID，使用taskID
		if taskID != "" {
			item.TastId = taskID
		}

		Rows = append(Rows, item)
	}

	return Rows, nil
}

// parseK8SDAEMONSET 解析 k8s daemonset 输出，返回结构化的 JSON
func parseK8SDAEMONSET(taskID, output string) ([]ItemCapture, error) {
	// output输出是多行文本，格式如下：
	// [
	// {"NAMESPACE":"kube-system","NAME":"calico-node","DESIRED":"2","CURRENT":"2","READY":"2","UP-TO-DATE":"2","AVAILABLE":"2","NODE SELECTOR":"kubernetes.io/os=linux","AGE":"42d"},
	// {"NAMESPACE":"kube-system","NAME":"kube-proxy","DESIRED":"2","CURRENT":"2","READY":"2","UP-TO-DATE":"2","AVAILABLE":"2","NODE SELECTOR":"kubernetes.io/os=linux","AGE":"42d"}
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
			DataType:  "json",
			ContentId: strings.ToLower(K8SDAEMONSET),
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

		if name, ok := container["NAME"].(string); ok && name != "" {
			item.ContentId = strings.ToLower(K8SDAEMONSET) + "_" + name
		}

		// 如果提供了taskID，使用taskID
		if taskID != "" {
			item.TastId = taskID
		}

		Rows = append(Rows, item)
	}

	return Rows, nil
}
func parseK8SDEPLOYMENT(taskID, output string) ([]ItemCapture, error) {
	// output输出是多行文本，格式如下：

	// [
	// {"NAMESPACE":"app","NAME":"nginx","READY":"1/1","UP-TO-DATE":"1","AVAILABLE":"1","AGE":"17d"},
	// {"NAMESPACE":"app","NAME":"nginx-local","READY":"1/1","UP-TO-DATE":"1","AVAILABLE":"1","AGE":"17d"},
	// {"NAMESPACE":"app","NAME":"nginx-test","READY":"1/1","UP-TO-DATE":"1","AVAILABLE":"1","AGE":"17d"},
	// {"NAMESPACE":"kube-system","NAME":"calico-kube-controllers","READY":"1/1","UP-TO-DATE":"1","AVAILABLE":"1","AGE":"42d"},
	// {"NAMESPACE":"kube-system","NAME":"coredns","READY":"1/1","UP-TO-DATE":"1","AVAILABLE":"1","AGE":"42d"},
	// {"NAMESPACE":"kuboard","NAME":"kuboard-metrics-server","READY":"1/1","UP-TO-DATE":"1","AVAILABLE":"1","AGE":"31d"},
	// {"NAMESPACE":"kuboard","NAME":"metrics-scraper","READY":"1/1","UP-TO-DATE":"1","AVAILABLE":"1","AGE":"29d"}
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
			DataType:  "json",
			ContentId: strings.ToLower(K8SDEPLOYMENT),
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

		if name, ok := container["NAME"].(string); ok && name != "" {
			item.ContentId = strings.ToLower(K8SDEPLOYMENT) + "_" + name
		}

		// 如果提供了taskID，使用taskID
		if taskID != "" {
			item.TastId = taskID
		}

		Rows = append(Rows, item)
	}

	return Rows, nil
}
func parseK8SSTATEFULSET(taskID, output string) ([]ItemCapture, error) {
	// output输出是多行文本，格式如下：
	// [
	// {"NAMESPACE":"app","NAME":"redis","READY":"1/1","AGE":"29d"}
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
			DataType:  "json",
			ContentId: strings.ToLower(K8SSTATEFULSET),
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

		if name, ok := container["NAME"].(string); ok && name != "" {
			item.ContentId = strings.ToLower(K8SSTATEFULSET) + "_" + name
		}

		// 如果提供了taskID，使用taskID
		if taskID != "" {
			item.TastId = taskID
		}

		Rows = append(Rows, item)
	}

	return Rows, nil
}

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

// // main 函数用于测试
// func main() {
// 	// 创建插件实例
// 	plugin := &k8scapturePlugin{}

// 	// 测试输入
// 	// input := map[string]string{
// 	// 	"cmd":     "k8sinfo",
// 	// 	"task_id": "test_task_id",
// 	// 	// 这里可以添加其他必要的输入参数
// 	// }

// 	tests := []struct {
// 		name        string
// 		cmdInput    string
// 		expectedCmd string
// 		hasError    bool
// 		errorMsg    string
// 	}{

// 		{
// 			name:        "k8sinfo command",
// 			cmdInput:    K8SINFO,
// 			expectedCmd: "kubectl cluster-info",
// 			hasError:    false,
// 		},
// 		{
// 			name:        "k8snodes command",
// 			cmdInput:    K8SNODES,
// 			expectedCmd: "kubectl get nodes -o wide",
// 			hasError:    false,
// 		},
// 		{
// 			name:        "k8sns command",
// 			cmdInput:    K8SNS,
// 			expectedCmd: "kubectl get namespaces",
// 			hasError:    false,
// 		},
// 		{
// 			name:        "k8spods command",
// 			cmdInput:    K8SPODS,
// 			expectedCmd: "kubectl get pods -A -o wide",
// 			hasError:    false,
// 		},
// 		{
// 			name:        "k8stop command",
// 			cmdInput:    K8STOP,
// 			expectedCmd: "kubectl top pod -A",
// 			hasError:    false,
// 		},
// 		{
// 			name:        "k8ssvc command",
// 			cmdInput:    K8SSVC,
// 			expectedCmd: "kubectl get svc -A -o wide",
// 			hasError:    false,
// 		},
// 		{
// 			name:        "k8sconfigmap command",
// 			cmdInput:    K8SCM,
// 			expectedCmd: "kubectl get configmap -A -o wide",
// 			hasError:    false,
// 		},
// 		{
// 			name:        "k8sdaemonset command",
// 			cmdInput:    K8SDAEMONSET,
// 			expectedCmd: "kubectl get daemonset -A -o wide",
// 			hasError:    false,
// 		},
// 		{
// 			name:        "k8sdeployment command",
// 			cmdInput:    K8SDEPLOYMENT,
// 			expectedCmd: "kubectl get deployment -A -o wide",
// 			hasError:    false,
// 		},
// 		{
// 			name:        "k8sstatefulset command",
// 			cmdInput:    K8SSTATEFULSET,
// 			expectedCmd: "kubectl get statefulset -A -o wide",
// 			hasError:    false,
// 		},
// 	}

// 	tests2 := []struct {
// 		target_host     string
// 		target_user     string
// 		target_password string
// 		target_key      string
// 		target_port     string
// 		task_id         string
// 		cmd             string
// 		isScript        bool
// 	}{
// 		{
// 			target_host:     "10.220.42.155",
// 			target_user:     "root",
// 			target_password: "Tpri@hn20251205",
// 			target_key:      "",
// 			target_port:     "22",
// 			task_id:         "task123",
// 			cmd:             "ls -l /home",
// 			isScript:        true,
// 		},
// 	}
// 	for _, tt := range tests {

// 		// We need to mock the executor to test the command parsing without actual execution
// 		// For now, we'll just test that the parsing doesn't return errors for valid inputs
// 		input := map[string]string{
// 			"cmd":             tt.cmdInput,
// 			"target_host":     tests2[0].target_host,
// 			"target_user":     tests2[0].target_user,
// 			"target_password": tests2[0].target_password,
// 			"target_key":      tests2[0].target_key,
// 			"target_port":     tests2[0].target_port,
// 			"task_id":         tests2[0].task_id,
// 		}
// 		// 执行插件
// 		result, err := plugin.Execute(input)
// 		if err != nil {
// 			fmt.Printf("执行错误: %v\n", err)
// 			return
// 		}

// 		// 打印tableStr到控制台
// 		fmt.Println("=== 测试结果 ===")
// 		fmt.Println("stdout:")
// 		fmt.Println(result["stdout"])
// 		fmt.Println("\nstderr:")
// 		fmt.Println(result["stderr"])
// 		fmt.Println("================")
// 	}
// }
