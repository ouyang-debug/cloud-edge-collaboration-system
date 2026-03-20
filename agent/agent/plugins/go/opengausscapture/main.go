//go:build plugin

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"agent/plus"

	_ "github.com/lib/pq"
)

// 插件静态定义变量
const (
	PluginName    = "opengausscapture"
	PluginVersion = "0.1.0"
)

const (
	OpenGaussVersion            = "opengaussversion"
	OpenGaussInstanceStatus     = "opengaussinstance_status"
	OpenGaussUptime             = "opengaussuptime"
	OpenGaussMaxConnections     = "opengaussmax_connections"
	OpenGaussCurrentConnections = "opengausscurrent_connections"
	OpenGaussReplicationStatus  = "opengaussreplication_status"
	OpenGaussDeadlocks          = "opengaussdeadlocks"
	OpenGaussLogError           = "opengausslog_error"
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

// Table 定义顶层JSON结构，包含count和rows
type Table struct {
	Count int    `json:"count"` // 行数
	Rows  []Item `json:"rows"`  // 行数据
}

// MetricInfo 定义指标信息结构
type MetricInfo struct {
	MetricId      string `json:"metricId"`      // 指标唯一ID（配置中心分配）
	MetricName    string `json:"metricName"`    //- 指标名称
	MetricType    string `json:"metricType"`    // 指标类型（txt/num）
	ObjectType    string `json:"objectType"`    //- 运维对象类型（os、docker、k8s、database）
	SubobjectType string `json:"subobjectType"` //- 运维对象子类型（按实际对象类型配置，无则填空）
	Unit          string `json:"unit"`          //- 指标单位（无则填空）
}

var MetricInfoMap = map[string]MetricInfo{
	"opengaussversion": {
		MetricId:      "opengaussversion",
		MetricName:    "opengaussversion",
		MetricType:    "txt",
		ObjectType:    "opengauss",
		SubobjectType: "OPENGUASS_VERSION",
		Unit:          "",
	},
	"opengaussinstance_status": {
		MetricId:      "opengaussinstance_status",
		MetricName:    "opengaussinstance_status",
		MetricType:    "txt",
		ObjectType:    "opengauss",
		SubobjectType: "OPENGUASS_INSTANCE_STATUS",
		Unit:          "",
	},
	"opengaussuptime": {
		MetricId:      "opengaussuptime",
		MetricName:    "opengaussuptime",
		MetricType:    "txt",
		ObjectType:    "opengauss",
		SubobjectType: "OPENGUASS_UPTIME",
		Unit:          "",
	},
	"opengaussmax_connections": {
		MetricId:      "opengaussmax_connections",
		MetricName:    "opengaussmax_connections",
		MetricType:    "txt",
		ObjectType:    "opengauss",
		SubobjectType: "OPENGUASS_MAX_CONNECTIONS",
		Unit:          "",
	},
	"opengausscurrent_connections": {
		MetricId:      "opengausscurrent_connections",
		MetricName:    "opengausscurrent_connections",
		MetricType:    "txt",
		ObjectType:    "opengauss",
		SubobjectType: "OPENGUASS_CURRENT_CONNECTIONS",
		Unit:          "",
	},
	"opengaussreplication_status": {
		MetricId:      "opengaussreplication_status",
		MetricName:    "opengaussreplication_status",
		MetricType:    "txt",
		ObjectType:    "opengauss",
		SubobjectType: "OPENGUASS_REPLICATION_STATUS",
		Unit:          "",
	},
	"opengaussdeadlocks": {
		MetricId:      "opengaussdeadlocks",
		MetricName:    "opengaussdeadlocks",
		MetricType:    "txt",
		ObjectType:    "opengauss",
		SubobjectType: "OPENGUASS_DEADLOCKS",
		Unit:          "",
	},
	"opengausslog_error": {
		MetricId:      "opengausslog_error",
		MetricName:    "opengausslog_error",
		MetricType:    "txt",
		ObjectType:    "opengauss",
		SubobjectType: "OPENGUASS_LOG_ERROR",
		Unit:          "",
	},
}

type opengausscapturePlugin struct{}

func (p *opengausscapturePlugin) Name() string       { return PluginName }
func (p *opengausscapturePlugin) Version() string    { return PluginVersion }
func (p *opengausscapturePlugin) OutputType() string { return "monitor" }
func (p *opengausscapturePlugin) Description() string {
	return "OpenGauss database monitoring capture plugin"
}
func (p *opengausscapturePlugin) Initialize(config string) error { return nil }
func (p *opengausscapturePlugin) Shutdown() error                { return nil }

func (p *opengausscapturePlugin) Execute(input map[string]string) (map[string]string, error) {
	host := input["host"]
	user := input["user"]
	pass := input["pass"]
	portStr := input["port"]
	cmdStrIn := input["cmd"]
	taskID := input["task_id"]
	dbName := input["db"]

	if cmdStrIn == "" {
		cmdStrIn = "all"
	}

	cmdStr := strings.TrimSpace(cmdStrIn)
	cmdType := ""
	cmdParts := strings.SplitN(cmdStr, " ", 3)
	if len(cmdParts) >= 1 {
		cmdType = strings.TrimSpace(cmdParts[0])
	}

	if cmdType == "all" {
		return p.collectAllMetrics(host, portStr, user, pass, dbName, taskID)
	}

	var stdout, stderr string
	var err error

	switch cmdType {
	case "version":
		stdout, stderr, err = p.collectOpenGaussVersion(host, portStr, user, pass, dbName)
	case "instancestatus":
		stdout, stderr, err = p.collectOpenGaussInstanceStatus(host, portStr, user, pass, dbName)
	case "uptime":
		stdout, stderr, err = p.collectOpenGaussUptime(host, portStr, user, pass, dbName)
	case "maxconnections":
		stdout, stderr, err = p.collectOpenGaussMaxConnections(host, portStr, user, pass, dbName)
	case "currentconnections":
		stdout, stderr, err = p.collectOpenGaussCurrentConnections(host, portStr, user, pass, dbName)
	case "replicationstatus":
		stdout, stderr, err = p.collectOpenGaussReplicationStatus(host, portStr, user, pass, dbName)
	case "deadlocks":
		stdout, stderr, err = p.collectOpenGaussDeadlocks(host, portStr, user, pass, dbName)
	case "logerror":
		stdout, stderr, err = p.collectOpenGaussLogError(host, portStr, user, pass, dbName)
	default:
		return map[string]string{"stderr": "unknown cmd type"}, nil
	}

	if err != nil {
		return map[string]string{"stderr": "run cmd error: " + err.Error()}, err
	}

	table := Table{
		Count: 1,
		Rows:  []Item{},
	}

	RowsCapture := []ItemCapture{}

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

	items := output2items(RowsCapture)

	table.Rows = items
	table.Count = len(items)

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

// collectAllMetrics 采集所有指标类型
func (p *opengausscapturePlugin) collectAllMetrics(host, portStr, user, pass, dbName, taskID string) (map[string]string, error) {
	versionOut, _, err1 := p.collectOpenGaussVersion(host, portStr, user, pass, dbName)
	instanceStatusOut, _, err2 := p.collectOpenGaussInstanceStatus(host, portStr, user, pass, dbName)
	upTimeOut, _, err3 := p.collectOpenGaussUptime(host, portStr, user, pass, dbName)
	maxConnectionsOut, _, err4 := p.collectOpenGaussMaxConnections(host, portStr, user, pass, dbName)
	currentConnectionsOut, _, err5 := p.collectOpenGaussCurrentConnections(host, portStr, user, pass, dbName)
	replicationStatusOut, _, err6 := p.collectOpenGaussReplicationStatus(host, portStr, user, pass, dbName)
	deadlocksOut, _, err7 := p.collectOpenGaussDeadlocks(host, portStr, user, pass, dbName)
	logErrorOut, _, err8 := p.collectOpenGaussLogError(host, portStr, user, pass, dbName)

	table := Table{
		Count: 1,
		Rows:  []Item{},
	}

	RowsCapture := []ItemCapture{}

	if versionOut != "" {
		item := ItemCapture{
			TastId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(OpenGaussVersion),
			Content:   versionOut,
			Timestamp: time.Now().UnixMilli(),
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TastId = taskID
		}
		RowsCapture = append(RowsCapture, item)
	}
	if instanceStatusOut != "" {
		item := ItemCapture{
			TastId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(OpenGaussInstanceStatus),
			Content:   instanceStatusOut,
			Timestamp: time.Now().UnixMilli(),
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TastId = taskID
		}
		RowsCapture = append(RowsCapture, item)
	}
	if upTimeOut != "" {
		item := ItemCapture{
			TastId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(OpenGaussUptime),
			Content:   upTimeOut,
			Timestamp: time.Now().UnixMilli(),
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TastId = taskID
		}
		RowsCapture = append(RowsCapture, item)
	}
	if maxConnectionsOut != "" {
		item := ItemCapture{
			TastId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(OpenGaussMaxConnections),
			Content:   maxConnectionsOut,
			Timestamp: time.Now().UnixMilli(),
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TastId = taskID
		}
		RowsCapture = append(RowsCapture, item)
	}
	if currentConnectionsOut != "" {
		item := ItemCapture{
			TastId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(OpenGaussCurrentConnections),
			Content:   currentConnectionsOut,
			Timestamp: time.Now().UnixMilli(),
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TastId = taskID
		}
		RowsCapture = append(RowsCapture, item)
	}
	if replicationStatusOut != "" {
		item := ItemCapture{
			TastId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(OpenGaussReplicationStatus),
			Content:   replicationStatusOut,
			Timestamp: time.Now().UnixMilli(),
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TastId = taskID
		}
		RowsCapture = append(RowsCapture, item)
	}
	if deadlocksOut != "" {
		item := ItemCapture{
			TastId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(OpenGaussDeadlocks),
			Content:   deadlocksOut,
			Timestamp: time.Now().UnixMilli(),
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TastId = taskID
		}
		RowsCapture = append(RowsCapture, item)
	}

	if logErrorOut != "" {
		item := ItemCapture{
			TastId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(OpenGaussLogError),
			Content:   logErrorOut,
			Timestamp: time.Now().UnixMilli(),
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TastId = taskID
		}
		RowsCapture = append(RowsCapture, item)
	}

	items := output2items(RowsCapture)

	table.Rows = items
	table.Count = len(items)

	tableBytes, err := json.Marshal(table)
	if err != nil {
		return map[string]string{"stderr": err.Error()}, nil
	}
	tableStr := string(tableBytes)

	var errMsg string
	if err1 != nil {
		errMsg += "info: " + err1.Error() + "; "
	}
	if err2 != nil {
		errMsg += "metrics: " + err2.Error() + "; "
	}
	if err3 != nil {
		errMsg += "replication: " + err3.Error() + "; "
	}
	if err4 != nil {
		errMsg += "logs: " + err4.Error() + "; "
	}
	if err5 != nil {
		errMsg += "current_connections: " + err5.Error() + "; "
	}
	if err6 != nil {
		errMsg += "replication_status: " + err6.Error() + "; "
	}
	if err7 != nil {
		errMsg += "deadlocks: " + err7.Error() + "; "
	}
	if err8 != nil {
		errMsg += "log_error: " + err8.Error() + "; "
	}

	res := map[string]string{
		"stdout": tableStr,
		"stderr": errMsg,
	}

	return res, nil
}

func (p *opengausscapturePlugin) ExecuteWithProgress(taskID string, input map[string]string, reporter plus.ProgressReporter) (map[string]string, error) {
	input["task_id"] = taskID

	if input["cmd"] == "" {
		input["cmd"] = "all"
	}

	out, err := p.Execute(input)
	if reporter != nil {
		reporter.OnProgress(taskID, "opengausscapture", 1, 1, "")
		reporter.OnCompleted(taskID, "opengausscapture", err == nil, "")
	}
	return out, err
}

func urlEscape(s string) string {
	r := strings.ReplaceAll(s, "@", "%40")
	r = strings.ReplaceAll(r, ":", "%3A")
	return r
}

// 获取OpenGauss版本
func (p *opengausscapturePlugin) collectOpenGaussVersion(host, portStr, user, pass, dbName string) (string, string, error) {
	port := 5432
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	if user == "" {
		user = "postgres"
	}

	if dbName == "" {
		dbName = "postgres"
	}

	sslMode := "disable"
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", urlEscape(user), urlEscape(pass), host, port, dbName, sslMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	info := make(map[string]interface{})

	var version string
	err = db.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		return "", "", fmt.Errorf("failed to get version: %v", err)
	}
	info["version"] = version

	jsonData, err := json.Marshal(info)
	if err != nil {
		return "", "", err
	}

	return string(jsonData), "", nil
}

// 获取OpenGauss实例状态
func (p *opengausscapturePlugin) collectOpenGaussInstanceStatus(host, portStr, user, pass, dbName string) (string, string, error) {
	port := 5432
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	if user == "" {
		user = "postgres"
	}

	if dbName == "" {
		dbName = "postgres"
	}

	sslMode := "disable"
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", urlEscape(user), urlEscape(pass), host, port, dbName, sslMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	metrics := make(map[string]interface{})

	var instanceStatus string
	err = db.QueryRow("SELECT pg_is_in_recovery()").Scan(&instanceStatus)
	if err != nil {
		return "", "", fmt.Errorf("failed to get instance status: %v", err)
	}
	if instanceStatus == "false" {
		metrics["instance_status"] = "primary"
	} else {
		metrics["instance_status"] = "standby"
	}

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return "", "", err
	}

	return string(jsonData), "", nil
}

// 获取OpenGauss启动时间
func (p *opengausscapturePlugin) collectOpenGaussUptime(host, portStr, user, pass, dbName string) (string, string, error) {
	port := 5432
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	if user == "" {
		user = "postgres"
	}

	if dbName == "" {
		dbName = "postgres"
	}

	sslMode := "disable"
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", urlEscape(user), urlEscape(pass), host, port, dbName, sslMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	metrics := make(map[string]interface{})

	var uptimeStr string
	err = db.QueryRow("SELECT pg_postmaster_start_time()").Scan(&uptimeStr)
	if err == nil {
		uptimeTime, err := time.Parse(time.RFC3339, uptimeStr)
		if err == nil {
			uptime := time.Since(uptimeTime)
			metrics["uptime"] = uptime.String()
		}
	}

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return "", "", err
	}

	return string(jsonData), "", nil
}

// 获取OpenGauss最大连接数
func (p *opengausscapturePlugin) collectOpenGaussMaxConnections(host, portStr, user, pass, dbName string) (string, string, error) {
	port := 5432
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	if user == "" {
		user = "postgres"
	}

	if dbName == "" {
		dbName = "postgres"
	}

	sslMode := "disable"
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", urlEscape(user), urlEscape(pass), host, port, dbName, sslMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	metrics := make(map[string]interface{})

	var maxConnections int
	err = db.QueryRow("SELECT setting::int FROM pg_settings WHERE name = 'max_connections'").Scan(&maxConnections)
	if err == nil {
		metrics["max_connections"] = maxConnections
	}

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return "", "", err
	}

	return string(jsonData), "", nil
}

// 获取OpenGauss当前连接数
func (p *opengausscapturePlugin) collectOpenGaussCurrentConnections(host, portStr, user, pass, dbName string) (string, string, error) {
	port := 5432
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	if user == "" {
		user = "postgres"
	}

	if dbName == "" {
		dbName = "postgres"
	}

	sslMode := "disable"
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", urlEscape(user), urlEscape(pass), host, port, dbName, sslMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	metrics := make(map[string]interface{})

	var currentConnections int
	err = db.QueryRow("SELECT count(*) FROM pg_stat_activity").Scan(&currentConnections)
	if err == nil {
		metrics["current_connections"] = currentConnections
	}

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return "", "", err
	}

	return string(jsonData), "", nil
}

// 获取OpenGauss主从状态
func (p *opengausscapturePlugin) collectOpenGaussReplicationStatus(host, portStr, user, pass, dbName string) (string, string, error) {
	port := 5432
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	if user == "" {
		user = "postgres"
	}

	if dbName == "" {
		dbName = "postgres"
	}

	sslMode := "disable"
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", urlEscape(user), urlEscape(pass), host, port, dbName, sslMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	replication := make(map[string]interface{})

	var isRecovery bool
	err = db.QueryRow("SELECT pg_is_in_recovery()").Scan(&isRecovery)
	if err == nil {
		if isRecovery {
			replication["replication_status"] = "standby"
		} else {
			replication["replication_status"] = "primary"
		}
	}

	jsonData, err := json.Marshal(replication)
	if err != nil {
		return "", "", err
	}

	return string(jsonData), "", nil
}

// 获取OpenGauss死锁次数
func (p *opengausscapturePlugin) collectOpenGaussDeadlocks(host, portStr, user, pass, dbName string) (string, string, error) {
	port := 5432
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	if user == "" {
		user = "postgres"
	}

	if dbName == "" {
		dbName = "postgres"
	}

	sslMode := "disable"
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", urlEscape(user), urlEscape(pass), host, port, dbName, sslMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	metrics := make(map[string]interface{})

	var deadlocks int
	err = db.QueryRow("SELECT count(*) FROM pg_stat_database_deadlocks WHERE datname = $1", dbName).Scan(&deadlocks)
	if err != nil {
		err = db.QueryRow("SELECT count(*) FROM pg_stat_database WHERE datname = $1", dbName).Scan(&deadlocks)
	}
	if err == nil {
		metrics["deadlocks"] = deadlocks
	}

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return "", "", err
	}

	return string(jsonData), "", nil
}

// 获取OpenGauss错误日志文件
func (p *opengausscapturePlugin) collectOpenGaussLogError(host, portStr, user, pass, dbName string) (string, string, error) {
	port := 5432
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	if user == "" {
		user = "postgres"
	}

	if dbName == "" {
		dbName = "postgres"
	}

	sslMode := "disable"
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", urlEscape(user), urlEscape(pass), host, port, dbName, sslMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	logs := make(map[string]interface{})

	var logDirectory string
	err = db.QueryRow("SELECT setting FROM pg_settings WHERE name = 'log_directory'").Scan(&logDirectory)
	if err == nil {
		logs["log_directory"] = logDirectory
	}

	var logFile string
	err = db.QueryRow("SELECT setting FROM pg_settings WHERE name = 'log_filename'").Scan(&logFile)
	if err == nil {
		logs["log_filename"] = logFile
	}

	var logDestination string
	err = db.QueryRow("SELECT setting FROM pg_settings WHERE name = 'log_destination'").Scan(&logDestination)
	if err == nil {
		logs["log_destination"] = logDestination
	}

	var loggingCollector string
	err = db.QueryRow("SELECT setting FROM pg_settings WHERE name = 'logging_collector'").Scan(&loggingCollector)
	if err == nil {
		logs["logging_collector"] = loggingCollector
	}

	jsonData, err := json.Marshal(logs)
	if err != nil {
		return "", "", err
	}

	return string(jsonData), "", nil
}

// output2items 将RowsCapture转换为Item数组
func output2items(RowsCapture []ItemCapture) []Item {
	items := []Item{}
	for _, itemCapture := range RowsCapture {
		itemCapture.Content = strings.TrimSpace(itemCapture.Content)
		parts := strings.Split(itemCapture.ContentId, "_")

		ObjectType := parts[0]
		SubobjectId := ""
		if len(parts) == 2 {
			SubobjectId = parts[1]
		}

		if itemCapture.DataType == "json" {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(itemCapture.Content), &data); err == nil {
				itemCapture.Content = fmt.Sprintf("%v", data)
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
						SubobjectType: metricInfo.SubobjectType,
						ObjectId:      itemCapture.TastId,
						SubobjectId:   SubobjectId,
						Value:         fmt.Sprintf("%s", value),
						Unit:          metricInfo.Unit,
						Timestamp:     itemCapture.Timestamp,
						Tags:          map[string]string{},
					})
				}
			}
		} else if itemCapture.DataType == "txt" {
			metricInfo, ok := MetricInfoMap[ObjectType]
			if !ok {
				continue
			}

			items = append(items, Item{
				MetricId:      metricInfo.MetricId,
				MetricName:    metricInfo.MetricName,
				MetricType:    metricInfo.MetricType,
				ObjectType:    metricInfo.ObjectType,
				SubobjectType: metricInfo.SubobjectType,
				ObjectId:      itemCapture.TastId,
				SubobjectId:   SubobjectId,
				Value:         itemCapture.Content,
				Unit:          metricInfo.Unit,
				Timestamp:     itemCapture.Timestamp,
				Tags:          map[string]string{},
			})

		}
	}
	return items
}

func New() plus.Plugin { return &opengausscapturePlugin{} }
