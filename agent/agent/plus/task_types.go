package plus

import "time"

// CronConfig represents the cron configuration for scheduled tasks
type CronConfig struct {
	CronExpression string `json:"cronExpression"` // Cron expression
	TimeZone       string `json:"timeZone"`       // Time zone
	StartTime      string `json:"startTime"`      // Start time
	EndTime        string `json:"endTime"`        // End time
	MissedStrategy string `json:"missedStrategy"` // Missed execution strategy
}

// PluginUpgrade represents the plugin upgrade configuration
type PluginUpgrade struct {
	PluginName  string `json:"pluginName"`
	FileName    string `json:"fileName"`
	DownloadURL string `json:"downloadURL"`
}

// TaskDispatchPayload represents the structure of the task dispatch message from MQTT
type TaskDispatchPayload struct {
	TaskID        string         `json:"taskId"`
	TaskType      string         `json:"taskType"`
	TaskName      string         `json:"taskName"`
	ExecuteMode   string         `json:"executeMode"`
	FailTerminate bool           `json:"failTerminate"`
	CronConfig    *CronConfig    `json:"cronConfig,omitempty"` // Cron configuration for scheduled tasks
	StepList      []Step         `json:"stepList"`
	PluginUpgrade *PluginUpgrade `json:"pluginUpgrade,omitempty"` // Plugin upgrade configuration
	PlusUpgrade   *PlusUpgrade   `json:"plusUpgrade,omitempty"`   // Plus self-upgrade configuration
}

type FileConfig struct {
	FileName  string `json:"fileName,omitempty"`
	FileUrl   string `json:"fileUrl,omitempty"`
	LocalPath string `json:"localPath,omitempty"` // Optional: if server specifies a target path, otherwise we use convention

}

// Step represents a single step in the task execution
type Step struct {
	Sequence      int                    `json:"sequence,omitempty"`      // 步骤序号
	Plugin        string                 `json:"plugin,omitempty"`        // 插件名称
	PluginName    string                 `json:"pluginName,omitempty"`    // 插件名称（兼容字段）
	PluginVersion string                 `json:"pluginVersion,omitempty"` // 插件版本
	PluginConn    map[string]interface{} `json:"pluginConn,omitempty"`    // Connection parameters for plugins
	Command       string                 `json:"command,omitempty"`       // 命令
	BasePath      string                 `json:"basePath,omitempty"`      // Optional: if server specifies a base path, otherwise we use convention
	Input         []FileConfig           `json:"input,omitempty"`         // Input files array
	Output        string                 `json:"output,omitempty"`        // 输出

}

// TaskStatusPayload represents the status update to be sent via MQTT
type TaskStatusPayload struct {
	TaskID    string      `json:"taskId"`
	Status    string      `json:"status"` // "running", "completed", "failed"
	StepSeq   int         `json:"stepSeq,omitempty"`
	Progress  float64     `json:"progress,omitempty"`
	Message   string      `json:"message,omitempty"`
	Result    interface{} `json:"result,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// TaskAckPayload represents the acknowledgement message
type TaskAckPayload struct {
	TaskID    string `json:"taskId"`
	Status    string `json:"status"` // "received", "rejected"
	Message   string `json:"message,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// PlusUpgrade represents the plus self-upgrade configuration
type PlusUpgrade struct {
	Version     string `json:"version"`     // 目标版本号
	FileName    string `json:"fileName"`    // 文件名
	DownloadURL string `json:"downloadURL"` // 下载地址
}

// PlusUpgradeState represents the current upgrade state of plus
type PlusUpgradeState struct {
	TargetVersion string    // 目标升级版本
	NewBinaryPath string    // 新二进制文件路径
	IsUpgrading   bool      // 是否正在准备升级
	CanUpgrade    bool      // 是否可以安全升级
	DownloadTime  time.Time // 下载完成时间
}
