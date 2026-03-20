//go:build plugin

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent/plus"

	_ "github.com/go-sql-driver/mysql"
)

// 插件静态定义变量
const (
	PluginName    = "mysqlcapture"
	PluginVersion = "0.1.0"
)

const (
	MySQLVersion            = "mysqlversion"
	MySQLInstanceStatus     = "mysqlinstance_status"
	MySQLUptime             = "mysqluptime"
	MySQLMaxConnections     = "mysqlmax_connections"
	MySQLCurrentConnections = "mysqlcurrent_connections"
	MySQLReplicationStatus  = "mysqlreplication_status"
	MySQLDeadlocks          = "mysqldeadlocks"
	MySQLLogError           = "mysqllog_error"
)

// ItemCapture 定义输出结构
type ItemCapture struct {
	TaskId    string            `json:"taskId"`    // 任务ID
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
	"mysqlversion": {
		MetricId:      "mysqlversion",
		MetricName:    "mysqlversion",
		MetricType:    "txt",
		ObjectType:    "mysql",
		SubobjectType: "MYSQL_VERSION",
		Unit:          "",
	},
	"mysqlinstance_status": {
		MetricId:      "mysqlinstance_status",
		MetricName:    "mysqlinstance_status",
		MetricType:    "txt",
		ObjectType:    "mysql",
		SubobjectType: "MYSQL_INSTANCE_STATUS",
		Unit:          "",
	},
	"mysqluptime": {
		MetricId:      "mysqluptime",
		MetricName:    "mysqluptime",
		MetricType:    "num",
		ObjectType:    "mysql",
		SubobjectType: "MYSQL_UPTIME",
		Unit:          "秒",
	},
	"mysqlmax_connections": {
		MetricId:      "mysqlmax_connections",
		MetricName:    "mysqlmax_connections",
		MetricType:    "num",
		ObjectType:    "mysql",
		SubobjectType: "MYSQL_MAX_CONNECTIONS",
		Unit:          "个",
	},
	"mysqlcurrent_connections": {
		MetricId:      "mysqlcurrent_connections",
		MetricName:    "mysqlcurrent_connections",
		MetricType:    "num",
		ObjectType:    "mysql",
		SubobjectType: "MYSQL_CURRENT_CONNECTIONS",
		Unit:          "个",
	},
	"mysqlreplication_status": {
		MetricId:      "mysqlreplication_status",
		MetricName:    "mysqlreplication_status",
		MetricType:    "txt",
		ObjectType:    "mysql",
		SubobjectType: "MYSQL_REPLICATION_STATUS",
		Unit:          "",
	},
	"mysqldeadlocks": {
		MetricId:      "mysqldeadlocks",
		MetricName:    "mysqldeadlocks",
		MetricType:    "num",
		ObjectType:    "mysql",
		SubobjectType: "MYSQL_DEADLOCKS",
		Unit:          "次",
	},
	"mysqllog_error": {
		MetricId:      "mysqllog_error",
		MetricName:    "mysqllog_error",
		MetricType:    "txt",
		ObjectType:    "mysql",
		SubobjectType: "MYSQL_LOG_ERROR",
		Unit:          "",
	},
}

type mysqlcapturePlugin struct{}

func (p *mysqlcapturePlugin) Name() string       { return PluginName }
func (p *mysqlcapturePlugin) Version() string    { return PluginVersion }
func (p *mysqlcapturePlugin) OutputType() string { return "monitor" }
func (p *mysqlcapturePlugin) Description() string {
	return "MySQL metrics capture plugin"
}
func (p *mysqlcapturePlugin) Initialize(config string) error { return nil }
func (p *mysqlcapturePlugin) Shutdown() error                { return nil }

// 获取mysql连接实例
func getMySQLInstanceByInput(input map[string]string) (*sql.DB, error) {
	host := input["host"]
	user := input["user"]
	pass := input["pass"]
	portStr := input["port"]
	dbName := input["db"]
	if host == "" {
		return nil, fmt.Errorf("empty host")
	}
	// 设置默认值
	port := "3306"
	if portStr != "" {
		port = portStr
	}
	if user == "" {
		user = "root"
	}
	if dbName == "" {
		dbName = "mysql"
	}

	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&timeout=30s",
		user, pass, host, port, dbName)

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return db, nil
}

func (p *mysqlcapturePlugin) Execute(input map[string]string) (map[string]string, error) {
	// 获取MySQL连接实例
	db, err := getMySQLInstanceByInput(input)
	if err != nil {
		return map[string]string{"stderr": "failed to get MySQL instance: " + err.Error()}, nil
	}
	defer db.Close()

	taskID := input["task_id"]

	// 创建Table实例并填充数据
	table := Table{
		Count: 1,
		Rows:  []Item{},
	}

	// 解析stdout
	RowsCapture, err := p.collectMySQLMetrics(taskID, db)
	if err != nil {
		return map[string]string{"stderr": "run cmd error: " + err.Error()}, nil
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
		"stderr": "",
	}

	return res, nil
}

func (p *mysqlcapturePlugin) ExecuteWithProgress(taskID string, input map[string]string, reporter plus.ProgressReporter) (map[string]string, error) {
	// MySQL capture plugin is single-step; we simply run it and report completion
	input["task_id"] = taskID
	out, err := p.Execute(input)
	if reporter != nil {
		reporter.OnProgress(taskID, "mysqlcapture", 1, 1, "")
		reporter.OnCompleted(taskID, "mysqlcapture", err == nil, "")
	}
	return out, err
}

// collectMySQLMetrics 收集MySQL指标
func (p *mysqlcapturePlugin) collectMySQLMetrics(taskID string, db *sql.DB) ([]ItemCapture, error) {
	// 统一获取时间戳，确保所有指标使用相同的时间戳
	timestamp := time.Now().UnixMilli()

	versionOut, err1 := p.collectMySQLVersion(db, taskID)
	instanceStatusOut, err2 := p.collectMySQLInstanceStatus(db, taskID)
	uptimeOut, err3 := p.collectMySQLUptime(db, taskID)
	maxConnectionsOut, err4 := p.collectMySQLMaxConnections(db, taskID)
	currentConnectionsOut, err5 := p.collectMySQLCurrentConnections(db, taskID)
	replicationStatusOut, err6 := p.collectMySQLReplicationStatus(db, taskID)
	deadlocksOut, err7 := p.collectMySQLDeadlocks(db, taskID)
	logErrorOut, err8 := p.collectMySQLLogError(db, taskID)

	// 创建ItemCapture实例并填充所有指标
	Rows := []ItemCapture{}

	if versionOut != "" {
		item := ItemCapture{
			TaskId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(MySQLVersion),
			Content:   versionOut,
			Timestamp: timestamp,
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TaskId = taskID
		}
		Rows = append(Rows, item)
	}

	if instanceStatusOut != "" {
		item := ItemCapture{
			TaskId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(MySQLInstanceStatus),
			Content:   instanceStatusOut,
			Timestamp: timestamp,
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TaskId = taskID
		}
		Rows = append(Rows, item)
	}

	if uptimeOut != "" {
		item := ItemCapture{
			TaskId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(MySQLUptime),
			Content:   uptimeOut,
			Timestamp: timestamp,
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TaskId = taskID
		}
		Rows = append(Rows, item)
	}

	if maxConnectionsOut != "" {
		item := ItemCapture{
			TaskId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(MySQLMaxConnections),
			Content:   maxConnectionsOut,
			Timestamp: timestamp,
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TaskId = taskID
		}
		Rows = append(Rows, item)
	}

	if currentConnectionsOut != "" {
		item := ItemCapture{
			TaskId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(MySQLCurrentConnections),
			Content:   currentConnectionsOut,
			Timestamp: timestamp,
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TaskId = taskID
		}
		Rows = append(Rows, item)
	}

	if replicationStatusOut != "" {
		item := ItemCapture{
			TaskId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(MySQLReplicationStatus),
			Content:   replicationStatusOut,
			Timestamp: timestamp,
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TaskId = taskID
		}
		Rows = append(Rows, item)
	}

	if deadlocksOut != "" {
		item := ItemCapture{
			TaskId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(MySQLDeadlocks),
			Content:   deadlocksOut,
			Timestamp: timestamp,
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TaskId = taskID
		}
		Rows = append(Rows, item)
	}

	if logErrorOut != "" {
		item := ItemCapture{
			TaskId:    "req-20260210123456-789",
			DataType:  "json",
			ContentId: strings.ToLower(MySQLLogError),
			Content:   logErrorOut,
			Timestamp: timestamp,
			Metadata: map[string]string{
				"source": "agentname",
				"stepId": "none",
			},
		}
		if taskID != "" {
			item.TaskId = taskID
		}
		Rows = append(Rows, item)
	}

	// 合并所有错误
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

	if errMsg != "" {
		return Rows, fmt.Errorf("%s", errMsg)
	}

	return Rows, nil
}

// 采集MySQL数据库版本
func (p *mysqlcapturePlugin) collectMySQLVersion(db *sql.DB, taskID string) (string, error) {
	var version string
	err := db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		return "", err
	}
	info := make(map[string]interface{})
	info["version"] = version
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return "", err
	}
	return string(infoJSON), nil
}

// 采集MySQL实例状态
func (p *mysqlcapturePlugin) collectMySQLInstanceStatus(db *sql.DB, taskID string) (string, error) {
	var instanceStatus string
	// 使用SHOW GLOBAL STATUS命令替代直接查询information_schema表
	rows, err := db.Query("SHOW GLOBAL STATUS LIKE 'INSTANCE_STATE'")
	if rows != nil {
		defer rows.Close()
	}
	if err == nil && rows.Next() {
		var variableName, variableValue string
		if err := rows.Scan(&variableName, &variableValue); err == nil {
			instanceStatus = variableValue
		}
	}
	if instanceStatus == "" {
		// 如果INSTANCE_STATE不存在，使用默认状态
		instanceStatus = "RUNNING"
	}
	info := make(map[string]interface{})
	info["instance_status"] = instanceStatus
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return "", err
	}
	return string(infoJSON), nil
}

// collectMySQLUptime 采集MySQL运行时长
func (p *mysqlcapturePlugin) collectMySQLUptime(db *sql.DB, taskID string) (string, error) {
	var uptime int
	// 使用SHOW GLOBAL STATUS命令替代直接查询information_schema表
	rows, err := db.Query("SHOW GLOBAL STATUS LIKE 'UPTIME'")
	if rows != nil {
		defer rows.Close()
	}
	if err == nil && rows.Next() {
		var variableName, variableValue string
		if err := rows.Scan(&variableName, &variableValue); err == nil {
			// 转换为整数
			if _, err := fmt.Sscanf(variableValue, "%d", &uptime); err != nil {
				return "", err
			}
		}
	}
	if err != nil {
		return "", err
	}
	info := make(map[string]interface{})
	info["uptime"] = uptime
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return "", err
	}
	return string(infoJSON), nil
}

// collectMySQLMaxConnections 采集MySQL最大连接数
func (p *mysqlcapturePlugin) collectMySQLMaxConnections(db *sql.DB, taskID string) (string, error) {
	var maxConnections int
	// 使用SHOW GLOBAL VARIABLES命令替代直接查询information_schema表
	rows, err := db.Query("SHOW GLOBAL VARIABLES LIKE 'MAX_CONNECTIONS'")
	if rows != nil {
		defer rows.Close()
	}
	if err == nil && rows.Next() {
		var variableName, variableValue string
		if err := rows.Scan(&variableName, &variableValue); err == nil {
			// 转换为整数
			if _, err := fmt.Sscanf(variableValue, "%d", &maxConnections); err != nil {
				return "", err
			}
		}
	}
	if err != nil {
		return "", err
	}
	info := make(map[string]interface{})
	info["max_connections"] = maxConnections
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return "", err
	}
	return string(infoJSON), nil
}

// collectMySQLCurrentConnections 采集MySQL当前连接数
func (p *mysqlcapturePlugin) collectMySQLCurrentConnections(db *sql.DB, taskID string) (string, error) {
	var currentConnections int
	// 使用SHOW GLOBAL STATUS命令替代直接查询information_schema表
	rows, err := db.Query("SHOW GLOBAL STATUS LIKE 'THREADS_CONNECTED'")
	if rows != nil {
		defer rows.Close()
	}
	if err == nil && rows.Next() {
		var variableName, variableValue string
		if err := rows.Scan(&variableName, &variableValue); err == nil {
			// 转换为整数
			if _, err := fmt.Sscanf(variableValue, "%d", &currentConnections); err != nil {
				return "", err
			}
		}
	}
	if err != nil {
		return "", err
	}
	info := make(map[string]interface{})
	info["current_connections"] = currentConnections
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return "", err
	}
	return string(infoJSON), nil
}

// collectMySQLReplicationStatus 采集MySQL主从状态
func (p *mysqlcapturePlugin) collectMySQLReplicationStatus(db *sql.DB, taskID string) (string, error) {
	var replicationStatus string
	rows, err := db.Query("SHOW SLAVE STATUS")
	if rows != nil {
		return "", err
	}
	if err == nil && rows.Next() {
		replicationStatus = "RUNNING"
	} else {
		replicationStatus = "NOT CONFIGURED"
	}
	info := make(map[string]interface{})
	info["replication_status"] = replicationStatus
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return "", err
	}
	return string(infoJSON), nil
}

// collectMySQLDeadlocks 采集MySQL死锁次数
func (p *mysqlcapturePlugin) collectMySQLDeadlocks(db *sql.DB, taskID string) (string, error) {
	var deadlocks int
	// 使用SHOW GLOBAL STATUS命令替代直接查询information_schema表
	rows, err := db.Query("SHOW GLOBAL STATUS LIKE 'INNODB_DEADLOCKS'")
	if rows != nil {
		defer rows.Close()
	}
	if err == nil && rows.Next() {
		var variableName, variableValue string
		if err := rows.Scan(&variableName, &variableValue); err == nil {
			// 转换为整数
			if _, err := fmt.Sscanf(variableValue, "%d", &deadlocks); err != nil {
				return "", err
			}
		}
	}
	if err != nil {
		return "", err
	}
	info := make(map[string]interface{})
	info["deadlocks"] = deadlocks
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return "", err
	}
	return string(infoJSON), nil
}

// collectMySQLLogError 采集MySQL错误日志文件
func (p *mysqlcapturePlugin) collectMySQLLogError(db *sql.DB, taskID string) (string, error) {
	var logError string
	// 使用SHOW GLOBAL VARIABLES命令替代直接查询information_schema表
	rows, err := db.Query("SHOW GLOBAL VARIABLES LIKE 'LOG_ERROR'")
	if rows != nil {
		defer rows.Close()
	}
	if err == nil && rows.Next() {
		var variableName, variableValue string
		if err := rows.Scan(&variableName, &variableValue); err == nil {
			logError = variableValue
		}
	}
	if err != nil {
		return "", err
	}
	info := make(map[string]interface{})
	info["log_error"] = logError
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return "", err
	}
	return string(infoJSON), nil
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
					// 首先尝试直接使用ContentId作为metricId
					metricId := itemCapture.ContentId
					metricInfo, ok := MetricInfoMap[metricId]
					if !ok {
						// 尝试使用ObjectType+key的格式
						metricId = ObjectType + strings.ToLower(key)
						metricInfo, ok = MetricInfoMap[metricId]
						if !ok {
							// 尝试使用ObjectType_key的格式
							metricId = ObjectType + "_" + key
							metricInfo, ok = MetricInfoMap[metricId]
							if !ok {
								continue
							}
						}
					}
					items = append(items, Item{
						MetricId:      metricInfo.MetricId,
						MetricName:    metricInfo.MetricName,
						MetricType:    metricInfo.MetricType,
						ObjectType:    metricInfo.ObjectType,
						SubobjectType: metricInfo.SubobjectType, //- 运维对象子类型（按实际对象类型配置，无则填空）
						ObjectId:      itemCapture.TaskId,       // 运维对象唯一ID（taskid等）
						SubobjectId:   SubobjectId,              //- 运维对象子类型唯一ID（无则填空）
						Value:         fmt.Sprintf("%s", value), // 指标值（支持数值、字符串）
						Unit:          metricInfo.Unit,          //- 指标单位（无则填空）
						Timestamp:     itemCapture.Timestamp,    // 采集时间戳（毫秒）
						Tags:          map[string]string{},
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
				ObjectId:      itemCapture.TaskId,       // 运维对象唯一ID（taskid等）
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

func New() plus.Plugin { return &mysqlcapturePlugin{} }
