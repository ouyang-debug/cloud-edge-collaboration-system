package plus

import (
	"agent/logsync"
	db "agent/status"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// TaskStatus 定义任务的运行状态，用字符串表示不同的任务状态
type TaskStatus string

// 定义各种任务状态常量
const (
	TaskStatusReceived  TaskStatus = "received"  // 已接收状态：任务已被服务器接收
	TaskStatusAssigned  TaskStatus = "assigned"  // 已下发状态：任务已被分配给代理执行
	TaskStatusWaiting   TaskStatus = "waiting"   // 等待状态：任务已接收，等待执行
	TaskStatusRunning   TaskStatus = "running"   // 运行状态：任务正在执行中
	TaskStatusCompleted TaskStatus = "completed" // 完成状态：任务成功执行完毕
	TaskStatusCancelled TaskStatus = "cancelled" // 取消状态：任务已被取消
	TaskStatusFailed    TaskStatus = "failed"    // 失败状态：任务执行失败
)

const (
	TaskOutputTxt     string = "default"
	TaskOutputFile    string = "file"
	TaskOutputMonitor string = "monitor"
)

// Task 表示一个可由代理执行的任务结构体
// 包含任务的基本信息、输入输出数据、插件配置等
type Task struct {
	ID            string      `json:"id"`                     // 任务唯一标识符
	Name          string      `json:"name"`                   // 任务名称
	Description   string      `json:"description"`            // 任务描述
	StepList      []Step      `json:"stepList"`               // 任务执行步骤列表
	CurrentStep   int         `json:"currentStep"`            // 当前执行到的步骤序号
	TaskType      string      `json:"taskType"`               // 任务类型：cycle(周期性)/once(一次性)
	ExecuteMode   string      `json:"executeMode"`            // 执行模式：manual, sequence等
	FailTerminate bool        `json:"failTerminate"`          // 是否在失败时终止任务
	CronConfig    *CronConfig `json:"cronConfig,omitempty"`   // 定时任务配置（可选）
	Status        TaskStatus  `json:"status"`                 // 当前任务状态
	IsPrepared    bool        `json:"isPrepared"`             // 是否准备完成 若需要下载文件，是否下载完成
	CreatedAt     time.Time   `json:"created_at"`             // 任务创建时间
	UpdatedAt     time.Time   `json:"updated_at"`             // 任务最后更新时间
	CompletedAt   *time.Time  `json:"completed_at,omitempty"` // 任务完成时间（可选字段）
	PID           int         `json:"pid"`                    // 任务关联的进程ID
	StartTime     time.Time   `json:"start_time"`             // 任务开始执行的时间
	EndTime       time.Time   `json:"end_time"`               // 任务结束执行的时间
}

// TaskPluginInfo 表示任务使用的插件信息结构体
// 存储插件的基本配置和命令信息
type TaskPluginInfo struct {
	Name    string `json:"name"`              // 插件名称
	Version string `json:"version"`           // 插件版本
	Config  string `json:"config"`            // 插件配置文件路径
	Command string `json:"command,omitempty"` // 执行命令（可选字段）
}

// ScheduledTaskInfo 包含定时任务的额外信息
type ScheduledTaskInfo struct {
	EntryID cron.EntryID   // 调度器条目ID
	Loc     *time.Location // 时区信息
}

// TaskManager 任务管理器结构体，负责管理所有任务及其执行过程
// 包含任务存储、状态管理、进程控制等功能
type TaskManager struct {
	tasks         map[string]*Task             // 存储所有任务的映射表，键为任务ID，值为任务指针
	mutex         sync.RWMutex                 // 读写互斥锁，用于保护任务数据的并发访问安全
	isRunning     bool                         // 标识任务管理器是否正在运行
	processMutex  sync.RWMutex                 // 进程管理的读写互斥锁，保护进程映射表的并发访问
	processes     map[string]*exec.Cmd         // 存储运行中的任务进程，键为任务ID，值为进程命令对象
	pluginManager *PluginManager               // 插件管理器实例，用于管理和执行各种插件
	scheduler     *cron.Cron                   // cron调度器，用于管理定时任务
	scheduledJobs map[string]ScheduledTaskInfo // 存储已调度的任务ID和相关信息映射
	logSyncer     *logsync.LogSyncer           // 日志同步器，用于同步任务日志到服务器
	upgradeState  *PlusUpgradeState            // plus升级状态
	acceptNewTask bool                         // 是否接受新任务（升级时设置为false）
	serverConfig  *SvrConfig                   // 服务器配置
	yamlConfig    *YAMLConfig                  // yaml配置
}

// NewTaskManager 创建一个新的任务管理器实例
// 接收插件管理器作为参数，初始化任务管理器的各项资源
// 参数：pm - 插件管理器指针
// 返回：新创建的任务管理器实例指针
func NewTaskManager(pm *PluginManager) *TaskManager {
	// 初始化日志同步器
	yamlConfig, err := InitYamlConfig()
	if err != nil {
		log.Printf("Failed to initialize yaml config: %v", err)
	}
	serverConfig, err := initConfig()
	if err != nil {
		log.Printf("Failed to initialize server config: %v", err)
	}
	url := serverConfig.Server + yamlConfig.LogSyncApi
	agentID := resolveAgentID()

	config := logsync.Config{
		ReadInterval: 2 * time.Second,     // 每2秒读取一次日志
		ReadSize:     1024 * 1024,         // 每次读取1MB
		ServerURL:    url,                 // 服务器URL，根据实际情况修改
		AgentId:      agentID,             // 代理名称
		DBPath:       "./data/logsync.db", // 数据库路径
	}
	logSyncer, err := logsync.NewLogSyncer(config)
	if err != nil {
		log.Printf("Failed to create log syncer: %v", err)
	}

	return &TaskManager{
		tasks:         make(map[string]*Task),             // 初始化任务映射表
		processes:     make(map[string]*exec.Cmd),         // 初始化进程映射表
		scheduledJobs: make(map[string]ScheduledTaskInfo), // 初始化已调度任务映射表
		pluginManager: pm,                                 // 设置插件管理器
		logSyncer:     logSyncer,                          // 设置日志同步器
		upgradeState:  nil,                                // 初始无升级状态
		acceptNewTask: true,                               // 默认接受新任务
		serverConfig:  &serverConfig,                      // 设置服务器配置
		yamlConfig:    &yamlConfig,                        // 设置yaml配置
	}
}

func (tm *TaskManager) ProcessTaskByInit(taskID string) {
	task, _ := tm.tasks[taskID]

	// 首先检查定时任务是否已过期
	if task.TaskType == "cycle" {
		if task.CronConfig != nil && task.CronConfig.EndTime != "" {
			endTime, err := time.Parse("2006-01-02 15:04:05", task.CronConfig.EndTime)
			if err != nil {
				// 尝试其他时间格式
				endTime, err = time.Parse("2006-01-02T15:04:05Z", task.CronConfig.EndTime)
				if err != nil {
					endTime, err = time.Parse("2006-01-02T15:04:05.999999999Z", task.CronConfig.EndTime)
					if err != nil {
						log.Printf("Failed to parse end time %s: %v", task.CronConfig.EndTime, err)
					}
				}
			}

			if err == nil {
				now := time.Now()
				// 确保endTime使用本地时区
				endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(),
					endTime.Hour(), endTime.Minute(), endTime.Second(), endTime.Nanosecond(),
					now.Location())
				log.Printf("Checking task %s expiration: now=%s (location=%s), endTime=%s (location=%s), now.After(endTime)=%v",
					task.ID, now.Format("2006-01-02 15:04:05"), now.Location(),
					endTime.Format("2006-01-02 15:04:05"), endTime.Location(),
					now.After(endTime))

				if now.After(endTime) {
					// 任务已过期，检查当前状态
					if task.Status != TaskStatusCompleted && task.Status != TaskStatusCancelled && task.Status != TaskStatusFailed {
						log.Printf("Task %s has expired (end time: %s), setting to completed", task.ID, endTime.Format("2006-01-02 15:04:05"))
						tm.writeTaskLog(task.ID, fmt.Sprintf("Task has expired (end time: %s), setting to completed", endTime.Format("2006-01-02 15:04:05")))

						// 更新任务状态为完成
						task.Status = TaskStatusCompleted
						task.UpdatedAt = time.Now()
						completedAt := time.Now()
						task.CompletedAt = &completedAt
						task.EndTime = time.Now()

						// 保存状态
						if saveErr := tm.saveTasks(); saveErr != nil {
							log.Printf("Warning: failed to save tasks after marking expired: %v", saveErr)
						}
						message := "Task expired during initialization"
						// 发送完成状态
						tm.sendStatusAndSave(task, TaskStatusCompleted, message)
						// 取消调度
						tm.cancelScheduledTask(task.ID)
						// 关闭日志同步器
						if tm.logSyncer != nil {
							tm.logSyncer.SetTaskCompleted(task.ID, true)
						}
						return
					}
				}
			}
		}

		// 检查是否未到开始时间
		if task.CronConfig != nil && task.CronConfig.StartTime != "" {
			startTime, err := time.Parse("2006-01-02 15:04:05", task.CronConfig.StartTime)
			if err != nil {
				// 尝试其他时间格式
				startTime, err = time.Parse("2006-01-02T15:04:05Z", task.CronConfig.StartTime)
				if err != nil {
					startTime, err = time.Parse("2006-01-02T15:04:05.999999999Z", task.CronConfig.StartTime)
					if err != nil {
						log.Printf("Failed to parse start time %s: %v", task.CronConfig.StartTime, err)
					}
				}
			}

			if err == nil {
				now := time.Now()
				// 确保startTime使用本地时区
				startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(),
					startTime.Hour(), startTime.Minute(), startTime.Second(), startTime.Nanosecond(),
					now.Location())
				log.Printf("Checking task %s start time: now=%s (location=%s), startTime=%s (location=%s), now.Before(startTime)=%v",
					task.ID, now.Format("2006-01-02 15:04:05"), now.Location(),
					startTime.Format("2006-01-02 15:04:05"), startTime.Location(),
					now.Before(startTime))

				if now.Before(startTime) {
					// 任务未到开始时间，设置为等待状态
					log.Printf("Task %s has not started yet (start time: %s), setting to waiting state", task.ID, startTime.Format("2006-01-02 15:04:05"))
					task.Status = TaskStatusWaiting
					task.UpdatedAt = time.Now()

					// 保存状态
					if saveErr := tm.saveTasks(); saveErr != nil {
						log.Printf("Warning: failed to save tasks after setting to waiting: %v", saveErr)
					}
					message := "Task not started yet during initialization"
					// 发送等待状态
					tm.sendStatusAndSave(task, TaskStatusWaiting, message)

					// 调度定时任务
					if err := tm.scheduleTask(task); err != nil {
						log.Printf("Failed to schedule task %s: %v", task.ID, err)
					}
					// 对于未到开始时间的任务，保持日志同步器运行
					return
				}
			}
		}
	}

	// 处理其他状态
	switch task.Status {
	case TaskStatusReceived:
		log.Printf("Detected received task %s, attempting file download", task.ID)

		// 任务目录结构
		taskDir := filepath.Join("tasks", task.ID)
		if err := os.MkdirAll(taskDir, 0755); err != nil {
			log.Printf("Error creating task directory: %v", err) // 创建目录失败时记录错误
		}
		// 子目录：输入、日志、输出
		dirs := []string{"in", "log", "out"}
		for _, d := range dirs {
			if err := os.MkdirAll(filepath.Join(taskDir, d), 0755); err != nil {
				log.Printf("Error creating task subdirectory: %v", err) // 创建子目录失败时记录错误
			}
		}
		// 假设下载函数为 downloadTaskFiles，返回错误表示下载失败
		if err := tm.downloadFileByTaskStep(task); err != nil {
			log.Printf("Download failed for task %s: %v", task.ID, err)
			task.Status = TaskStatusFailed
			task.UpdatedAt = time.Now()

			if saveErr := tm.saveTasks(); saveErr != nil {
				log.Printf("Warning: failed to save tasks after download failure: %v", saveErr)
			}
			message := fmt.Sprintf("File download failed: %v", err)
			// 发送失败状态
			tm.sendStatusAndSave(task, TaskStatusFailed, message)
			// 标记任务完成状态，以便日志同步器知道任务已经完成
			if tm.logSyncer != nil {
				tm.logSyncer.SetTaskCompleted(task.ID, true)
			}
			return
		}
		log.Printf("File download succeeded for task %s", task.ID)
		// 下载成功，设置状态为已下发
		task.Status = TaskStatusAssigned
		task.UpdatedAt = time.Now()
		// 记录任务状态变更
		message := "File download succeeded"

		if saveErr := tm.saveTasks(); saveErr != nil {
			log.Printf("Warning: failed to save tasks after download success: %v", saveErr)
		}
		// 发送已下发状态
		tm.sendStatusAndSave(task, TaskStatusAssigned, message)
		go tm.StartTask(task.ID)
	case TaskStatusRunning:
		if task.TaskType == "cycle" {
			// 对于定时任务，将状态设置为TaskStatusWaiting并设置CurrentStep为0
			log.Printf("Resuming interrupted cycle task %s: %s, setting to waiting state", task.ID, task.Name)
			task.Status = TaskStatusWaiting
			task.CurrentStep = 0
			task.UpdatedAt = time.Now()

			// 保存状态
			if saveErr := tm.saveTasks(); saveErr != nil {
				log.Printf("Warning: failed to save tasks after setting to waiting: %v", saveErr)
			}

			message := "Task set to waiting state during initialization"
			// 发送等待状态
			tm.sendStatusAndSave(task, TaskStatusWaiting, message)
			// 重新调度定时任务
			if err := tm.scheduleTask(task); err != nil {
				log.Printf("Failed to reschedule waiting task %s: %v", task.ID, err)
			} else {
				cronExpr := ""
				if task.CronConfig != nil {
					cronExpr = task.CronConfig.CronExpression
				}
				log.Printf("Rescheduled waiting task %s with cron: %s", task.ID, cronExpr)
			}
		} else {
			// 对于一次性任务，继续执行
			log.Printf("Resuming interrupted task %s: %s", task.ID, task.Name)
			// 在新的协程中执行任务，以便从中断的位置继续执行
			go tm.executeTask(task)
		}
	case TaskStatusAssigned:
		go tm.StartTask(task.ID)
	case TaskStatusWaiting:
		// 对于等待状态的定时任务，重新加入定时任务调度器
		if task.TaskType == "cycle" {
			log.Printf("Resuming waiting cycle task %s: %s", task.ID, task.Name)
			// 重新调度定时任务
			if err := tm.scheduleTask(task); err != nil {
				log.Printf("Failed to reschedule waiting task %s: %v", task.ID, err)
			} else {
				cronExpr := ""
				if task.CronConfig != nil {
					cronExpr = task.CronConfig.CronExpression
				}
				log.Printf("Rescheduled waiting task %s with cron: %s", task.ID, cronExpr)
			}
		}
	}
}

// Start 初始化并启动任务管理器
// 加载本地存储的任务数据，并设置管理器运行状态
// 返回错误信息，如果启动过程中出现任何问题
func (tm *TaskManager) Start() error {
	log.Println("Starting Task Manager...") // 记录启动日志

	// 初始化数据库连接
	dbInitialized := true
	if err := db.InitDB(); err != nil {
		log.Printf("Warning: failed to init database: %v", err) // 如果初始化失败，记录警告日志但不中断启动
		dbInitialized = false
	} else {
		// 启动状态监控任务
		tm.StartStatusMonitor()
		// 启动结果同步监控任务
		tm.StartResultSyncMonitor()
	}

	// 从本地存储加载之前保存的任务数据
	if err := tm.loadTasks(); err != nil {
		log.Printf("Warning: failed to load tasks: %v", err) // 如果加载失败，记录警告日志但不中断启动
	}

	// 初始化调度器
	tm.scheduler = cron.New()
	tm.scheduler.Start()

	// 重新调度所有定时任务
	tm.mutex.Lock()
	for taskID, task := range tm.tasks {
		if task.TaskType == "cycle" && task.CronConfig != nil {
			// 检查任务是否已过期
			if task.Status != TaskStatusCompleted && task.Status != TaskStatusCancelled && task.Status != TaskStatusFailed {
				if err := tm.scheduleTask(task); err != nil {
					log.Printf("Warning: failed to reschedule task %s: %v", taskID, err)
				} else {
					log.Printf("Rescheduled cycle task: %s", taskID)
				}
			}
		}
	}
	tm.mutex.Unlock()

	// 启动日志同步器
	if tm.logSyncer != nil {
		if err := tm.logSyncer.Start(); err != nil {
			log.Printf("Failed to start log syncer: %v", err)
		}
	}

	// 只有当数据库初始化成功时，才处理处于 received 状态的任务
	if dbInitialized {
		// 处理处于 received 状态的任务：尝试下载文件
		for _, task := range tm.tasks {
			go tm.ProcessTaskByInit(task.ID)
		}
	} else {
		log.Printf("Warning: database not initialized, skipping task processing")
	}

	tm.isRunning = true                 // 设置管理器为运行状态
	log.Println("Task Manager started") // 记录启动完成日志
	return nil
}

// Stop 停止任务管理器及所有正在运行的任务
// 遍历所有任务，停止正在运行的任务并将它们标记为已取消
// 返回错误信息，如果停止过程中出现任何问题
func (tm *TaskManager) Stop() error {
	log.Println("Stopping Task Manager...") // 记录停止开始日志

	// 获取写锁，确保在停止过程中不会有其他操作修改任务状态
	tm.mutex.Lock()
	// 函数结束时自动释放锁
	defer tm.mutex.Unlock()

	// 遍历所有任务，停止正在运行的任务
	for _, task := range tm.tasks {
		if task.Status == TaskStatusRunning {
			tm.stopTaskProcessLocked(task.ID)
			task.Status = TaskStatusCancelled
			task.UpdatedAt = time.Now()
		}
	}

	// 停止调度器并清理所有定时任务
	if tm.scheduler != nil {
		tm.scheduler.Stop()
	}

	// 停止日志同步器
	if tm.logSyncer != nil {
		tm.logSyncer.Stop()
	}

	// 设置管理器为非运行状态
	tm.isRunning = false
	// 记录停止完成日志
	log.Println("Task Manager stopped")
	return nil
}

// 接收来自于MQTT的指令，启动指定的任务
func (tm *TaskManager) StartTaskByMQTT(taskID string) error {
	tm.mutex.Lock()
	// 函数结束时自动释放锁
	defer tm.mutex.Unlock()

	// 查找指定ID的任务
	updatedAt := time.Now().Format("2006-01-02 15:04:05")
	task, exists := tm.tasks[taskID]
	if !exists {
		tm.PostStatusData(map[string]interface{}{
			"client_id":  resolveAgentID(),
			"task_id":    taskID,
			"status":     string(TaskStatusFailed),
			"updated_at": updatedAt,
			"message":    fmt.Sprintf("task not found: %s", taskID),
		})
		return fmt.Errorf("task not found: %s", taskID) // 如果任务不存在，返回错误
	}

	// 检查任务当前状态，只有已接收，已下发，等待、失败或已取消状态的任务才能启动,如果任务已在其他状态，返回错误
	if task.Status != TaskStatusReceived && task.Status != TaskStatusAssigned && task.Status != TaskStatusWaiting && task.Status != TaskStatusFailed && task.Status != TaskStatusCancelled {
		return fmt.Errorf("task is already in %s status", task.Status)
	}
	// 初始化日志同步状态并启动日志同步协程
	if tm.logSyncer != nil {
		// 初始化任务日志同步状态
		tm.logSyncer.InitStatus(taskID)
		// 启动日志同步协程
		if err := tm.logSyncer.StartTaskSync(taskID); err != nil {
			log.Printf("Failed to start log sync for task %s: %v", taskID, err)
		}
	}
	tm.StartTask(taskID)

	return nil
}

// StartTask 启动指定的任务
// 检查任务是否存在且状态允许启动，然后将其状态更新为运行中并开始执行
// 对于定时任务，重新添加到调度器中
// 参数：taskID - 要启动的任务ID
// 返回：错误信息，如果启动失败
func (tm *TaskManager) StartTask(taskID string) error {
	// 获取写锁，防止并发修改任务状态
	tm.mutex.Lock()
	// 函数结束时自动释放锁
	defer tm.mutex.Unlock()

	// 查找指定ID的任务
	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID) // 如果任务不存在，返回错误
	}

	// 检查是否正在接受新任务（升级期间不启动新任务，但允许已在运行中的任务继续）
	if !tm.acceptNewTask && task.Status != TaskStatusRunning {
		log.Printf("Cannot start task %s: plus is preparing for upgrade", taskID)
		return fmt.Errorf("plus is preparing for upgrade, cannot start new tasks")
	}
	// 检查任务是否准备完成
	if !task.IsPrepared {
		//进行下载文件
		if err := tm.downloadFileByTaskStep(task); err != nil {
			//无法下载文件，任务失败
			tm.writeTaskLog(taskID, fmt.Sprintf("Failed to download file: %v", err))
			log.Printf("Failed to download files for task %s: %v", taskID, err)
			// 更新任务状态，并保存到tasks.json
			task.Status = TaskStatusFailed
			task.UpdatedAt = time.Now()
			tm.mutex.Lock()
			tm.tasks[task.ID] = task
			tm.mutex.Unlock()
			tm.saveTasks()
			// 使用PostStatusData发送失败状态更新到MQTT
			message := fmt.Sprintf("Failed to download file: %v", err)
			// 发送失败状态
			tm.sendStatusAndSave(task, TaskStatusFailed, message)
			// 标记任务完成状态，以便日志同步器知道任务已经完成
			if tm.logSyncer != nil {
				tm.logSyncer.SetTaskCompleted(taskID, true)
			}
			return fmt.Errorf("failed to download files for task %s: %v", taskID, err)
		}
	}
	// 检查任务当前状态，只有已接收，已下发，等待、失败或已取消状态的任务才能启动,如果任务已在其他状态，返回错误
	if task.Status != TaskStatusReceived && task.Status != TaskStatusAssigned && task.Status != TaskStatusWaiting && task.Status != TaskStatusFailed && task.Status != TaskStatusCancelled {
		return fmt.Errorf("task is already in %s status", task.Status)
	}

	// 检查任务是否在指定的时间范围内
	currentTime := time.Now()
	if task.CronConfig != nil {
		// 检查开始时间
		if task.CronConfig.StartTime != "" {
			startTime, err := time.Parse("2006-01-02 15:04:05", task.CronConfig.StartTime)
			if err == nil && currentTime.Before(startTime) {
				// 任务还未到开始时间，设置为等待状态
				task.Status = TaskStatusWaiting
				task.UpdatedAt = time.Now()

				// 保存状态
				if err := tm.saveTasks(); err != nil {
					log.Printf("Warning: failed to save tasks: %v", err)
				}

				tm.mutex.Lock()
				tm.tasks[task.ID] = task
				tm.mutex.Unlock()
				tm.saveTasks()
				// 发送等待状态
				message := "Task waiting for start time"
				tm.sendStatusAndSave(task, TaskStatusWaiting, message)

				log.Printf("Task %s set to waiting - not yet at start time", taskID)
				tm.writeTaskLog(taskID, fmt.Sprintf("Task %s set to waiting - not yet at start time", taskID))
				return nil
			}
		}

		// 检查结束时间
		if task.CronConfig.EndTime != "" {
			endTime, err := time.Parse("2006-01-02 15:04:05", task.CronConfig.EndTime)
			if err == nil && currentTime.After(endTime) {
				// 任务已过期，设置为完成状态
				task.Status = TaskStatusCompleted
				task.UpdatedAt = time.Now()
				completedAt := time.Now()
				task.CompletedAt = &completedAt
				task.EndTime = time.Now()

				// 保存状态
				if err := tm.saveTasks(); err != nil {
					log.Printf("Warning: failed to save tasks: %v", err)
				}
				tm.mutex.Lock()
				tm.tasks[task.ID] = task
				tm.mutex.Unlock()
				tm.saveTasks()
				// 发送完成状态
				message := "Task expired"
				tm.sendStatusAndSave(task, TaskStatusCompleted, message)

				log.Printf("Task %s set to completed - already expired", taskID)
				tm.writeTaskLog(taskID, fmt.Sprintf("Task %s set to completed - already expired", taskID))
				return nil
			}
		}
	}

	// 对于定时任务，重新添加到调度器
	if task.TaskType == "cycle" {
		if err := tm.scheduleTask(task); err != nil {
			log.Printf("Failed to reschedule task %s: %v", task.ID, err)
			return fmt.Errorf("failed to reschedule task: %v", err)
		}
		cronExpr := ""
		if task.CronConfig != nil {
			cronExpr = task.CronConfig.CronExpression
		}
		log.Printf("Task %s rescheduled with cron: %s", task.ID, cronExpr)

		// 检查任务是否已经过期（调度后状态变为completed）
		// 注意：此时已经持有锁，不需要再次获取
		if task.Status == TaskStatusCompleted {
			log.Printf("Task %s has been completed (expired), skipping execution", task.ID)
			return nil
		}
	}

	// 对于定时任务，根据cron表达式类型决定是否立即执行
	if task.TaskType == "cycle" && task.CronConfig != nil && task.CronConfig.CronExpression != "" {
		cronType := tm.analyzeCronExpression(task.CronConfig.CronExpression)
		if cronType == "fixed_rate" {
			// 固定频率任务，立即执行一次
			task.Status = TaskStatusRunning
			task.UpdatedAt = time.Now()
			task.StartTime = time.Now()

			// 保存状态
			if err := tm.saveTasks(); err != nil {
				log.Printf("Warning: failed to save tasks: %v", err)
			}
			// 发送运行状态
			message := "Task started (fixed rate)"
			tm.sendStatusAndSave(task, TaskStatusRunning, message)

			// 在新的协程中异步执行任务
			go tm.executeTask(task)
			log.Printf("Task %s started (fixed rate, executing immediately)", taskID)
		} else {
			// 固定时间点任务，设置为等待状态
			task.Status = TaskStatusWaiting
			task.UpdatedAt = time.Now()

			// 保存状态
			if err := tm.saveTasks(); err != nil {
				log.Printf("Warning: failed to save tasks: %v", err)
			}
			// 发送等待状态
			message := "Task waiting for scheduled time"
			tm.sendStatusAndSave(task, TaskStatusWaiting, message)

			log.Printf("Task %s started (fixed time, waiting for scheduled time)", taskID)
		}
	} else {
		// 非定时任务，立即执行
		task.Status = TaskStatusRunning
		task.UpdatedAt = time.Now()
		task.StartTime = time.Now()

		// 保存状态
		if err := tm.saveTasks(); err != nil {
			log.Printf("Warning: failed to save tasks: %v", err)
		}

		// 发送运行状态
		message := "Task started"
		tm.sendStatusAndSave(task, TaskStatusRunning, message)

		// 在新的协程中异步执行任务
		go tm.executeTask(task)
		log.Printf("Task %s started", taskID)
	}

	return nil
}

// StopTask 停止指定的任务
// 检查任务是否存在，然后停止任务进程（如果正在运行）并更新状态
// 对于定时任务，无论是否正在运行，都会从调度器中移除
// 参数：taskID - 要停止的任务ID
// 返回：错误信息，如果停止失败
func (tm *TaskManager) StopTask(taskID string) error {
	log.Printf("Received stop request for task: %s", taskID) // 记录任务停止请求日志
	tm.mutex.Lock()                                          // 获取写锁，防止并发修改任务状态
	defer tm.mutex.Unlock()                                  // 函数结束时自动释放锁

	// 查找指定ID的任务
	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID) // 如果任务不存在，返回错误
	}

	// 检查任务是否正在运行，如果是则停止其进程
	if task.Status == TaskStatusRunning {
		tm.stopTaskProcessLocked(taskID) // 停止对应的任务进程
	}

	// 对于定时任务，从调度器中移除
	if task.TaskType == "cycle" {
		tm.cancelScheduledTask(taskID) // 从调度器中移除定时任务
	}

	// 更新任务状态为已取消
	task.Status = TaskStatusCancelled
	task.UpdatedAt = time.Now() // 记录更新时间

	// 将任务状态保存到本地存储
	if err := tm.saveTasks(); err != nil {
		log.Printf("Warning: failed to save tasks: %v", err) // 保存失败时记录警告日志
	}

	// 通过MQTT发送任务状态更新消息
	//tm.sendStatusUpdate(taskID, string(TaskStatusCancelled), 0, 0, "Task stopped")

	// 发送取消状态更新
	message := "Task stopped"
	tm.sendStatusAndSave(task, TaskStatusCancelled, message)

	// 停止日志同步协程
	if tm.logSyncer != nil {
		tm.logSyncer.StopTaskSync(taskID)
		// 标记任务完成状态，以便日志同步器知道任务已经完成
		tm.logSyncer.SetTaskCompleted(taskID, true)
	}

	log.Printf("Task %s stopped", taskID) // 记录任务停止日志
	return nil
}

// UpdateTask 更新或创建一个任务
// 接收JSON格式的任务数据，解析后更新或创建相应的任务
// 参数：taskJSON - JSON格式的任务数据字符串
// 返回：错误信息，如果更新失败
func (tm *TaskManager) UpdateTask(taskJSON string) error {
	tm.mutex.Lock()         // 获取写锁，防止并发修改任务数据
	defer tm.mutex.Unlock() // 函数结束时自动释放锁

	// 解析传入的JSON字符串为Task结构体
	var task Task
	if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
		return fmt.Errorf("failed to parse task JSON: %v", err) // 如果JSON格式错误，返回错误
	}

	// 验证任务数据，确保任务ID不为空
	if task.ID == "" {
		return fmt.Errorf("task ID is required") // 如果任务ID为空，返回错误
	}

	// 检查任务是否已存在
	existingTask, exists := tm.tasks[task.ID]
	fmt.Printf("existingTask: %v, exists: %v\n", existingTask, exists)

	// 如果是定时任务，先取消之前的调度
	if exists && existingTask.TaskType == "cycle" {
		tm.cancelScheduledTask(task.ID)
	}

	// 根据任务类型处理
	if task.TaskType == "cycle" {
		// 这是一个定时任务，添加到调度器
		if err := tm.scheduleTask(&task); err != nil {
			log.Printf("Failed to schedule task %s: %v", task.ID, err)
			// 即使调度失败，仍继续保存任务
		}
	}

	if exists {
		// 更新已存在的任务
		//oldStatus := existingTask.Status        // 保存旧状态用于比较
		task.CreatedAt = existingTask.CreatedAt // 保留原始创建时间
		task.UpdatedAt = time.Now()             // 更新最后修改时间为当前时间

		// 将更新后的任务保存到内存中的任务映射表
		tm.tasks[task.ID] = &task

		// 将任务数据保存到本地存储
		if err := tm.saveTasks(); err != nil {
			return fmt.Errorf("failed to save tasks: %v", err) // 保存失败时返回错误
		}

		// 如果任务状态发生了变化，则通过MQTT发送状态更新
		// if oldStatus != task.Status {
		// 	tm.sendStatusUpdate(task.ID, string(task.Status), 0, 0, "Task status updated")
		// }
	}

	// 如果任务状态为 received 或 assigned，则打开日志同步器
	if task.Status == TaskStatusReceived || task.Status == TaskStatusAssigned {
		if tm.logSyncer != nil {
			// 初始化任务日志同步状态
			tm.logSyncer.InitStatus(task.ID)
			// 启动日志同步协程
			if err := tm.logSyncer.StartTaskSync(task.ID); err != nil {
				log.Printf("Failed to start log sync for task %s: %v", task.ID, err)
			}
		}
	}

	log.Printf("Task %s updated", task.ID) // 记录任务更新日志
	return nil
}

// GetTaskStatus 获取指定任务的状态
// 通过任务ID查找任务并返回其当前状态
// 参数：taskID - 要查询的任务ID
// 返回：任务状态，错误信息（如果任务不存在）
func (tm *TaskManager) GetTaskStatus(taskID string) (TaskStatus, error) {
	tm.mutex.RLock()         // 获取读锁，允许并发读取任务数据
	defer tm.mutex.RUnlock() // 函数结束时自动释放锁

	// 查找指定ID的任务
	task, exists := tm.tasks[taskID]
	if !exists {
		return "", fmt.Errorf("task not found: %s", taskID) // 如果任务不存在，返回错误
	}

	return task.Status, nil // 返回任务状态
}

// ListTasks 列出所有任务
// 返回内存中存储的所有任务的切片
// 返回：任务指针数组，错误信息（目前总是返回nil）
func (tm *TaskManager) ListTasks() ([]*Task, error) {
	tm.mutex.RLock()         // 获取读锁，允许并发读取任务数据
	defer tm.mutex.RUnlock() // 函数结束时自动释放锁

	// 将任务映射表转换为任务切片
	var tasks []*Task               // 创建空的任务切片
	for _, task := range tm.tasks { // 遍历所有任务
		tasks = append(tasks, task) // 将每个任务添加到切片中
	}

	return tasks, nil // 返回任务切片
}

// writeTaskLog 写入任务日志到taskDir/log/task.log
// 参数：
//
//	taskDir - 任务目录
//	message - 日志消息
func (tm *TaskManager) writeTaskLog(taskID, message string) {
	logPath := filepath.Join("tasks", taskID, "log", "task.log")
	logDir := filepath.Join("tasks", taskID, "log")

	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
		return
	}

	// 打开或创建日志文件
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open task log file: %v", err)
		return
	}
	defer file.Close()

	// 写入日志消息，包含时间戳
	logEntry := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)
	if _, err := file.WriteString(logEntry); err != nil {
		log.Printf("Failed to write to task log: %v", err)
	}
}

// recordTaskStatus 记录任务状态变更到数据库
func (tm *TaskManager) recordTaskStatus(task *Task, UpdatedAt string, message string, IsSend bool) {
	// 检查数据库是否初始化
	if db.DB == nil {
		log.Printf("Warning: database not initialized, skipping task status recording")
		return
	}

	// 创建任务状态记录
	taskStatus := &db.TaskStatus{
		TaskID:    task.ID,
		TaskName:  task.Name,
		TaskType:  task.TaskType,
		Status:    string(task.Status),
		UpdatedAt: UpdatedAt,
		Message:   message,
		IsSend:    IsSend,
	}

	// 保存到数据库
	if err := db.CreateTaskStatus(taskStatus); err != nil {
		log.Printf("Warning: failed to record task status: %v", err)
	}
}

// executeTask 执行一个任务
// 遍历任务中的所有步骤，按顺序执行每个步骤，并处理执行结果
// 参数：task - 要执行的任务指针
func (tm *TaskManager) executeTask(task *Task) {
	// 任务的工作目录
	taskDir := filepath.Join("tasks", task.ID)

	// 写入任务开始日志
	tm.writeTaskLog(task.ID, fmt.Sprintf("Task execution started: %s", task.Name))
	log.Printf("Executing task %s: %s", task.ID, task.Name) // 记录任务开始执行日志

	// 确保任务状态为running并发送更新
	tm.mutex.Lock()
	task.Status = TaskStatusRunning
	task.UpdatedAt = time.Now()
	tm.mutex.Unlock()

	// 发送运行状态
	message := "Task execution started"
	tm.sendStatusAndSave(task, TaskStatusRunning, message)

	// 遍历任务中定义的所有步骤，按顺序执行
	for i, step := range task.StepList {
		// 跳过已经执行过的步骤
		if i < task.CurrentStep {
			log.Printf("Skipping step %d (already executed)", step.Sequence)
			continue
		}

		var out map[string]string // 存储步骤执行结果
		var err error             // 存储执行过程中可能出现的错误
		stepSeq := step.Sequence  // 当前步骤序列号

		// 准备当前步骤的输入数据
		stepInput := make(map[string]string) // 创建输入参数映射
		// 添加插件特定的配置
		// 如果是shell插件且有自定义命令，则添加命令参数
		if strings.ToLower(step.Plugin) == "shell" && strings.TrimSpace(step.Command) != "" {
			stepInput["cmd"] = step.Command
		}
		if strings.ToLower(step.Plugin) == "mysql" && strings.TrimSpace(step.Command) != "" {
			stepInput["query"] = step.Command
		}
		if strings.ToLower(step.Plugin) == "oscapture" && strings.TrimSpace(step.Command) != "" {
			stepInput["cmd"] = step.Command
		}
		if strings.ToLower(step.Plugin) == "docker" && strings.TrimSpace(step.Command) != "" {
			stepInput["cmd"] = step.Command
			stepInput["workDir"] = step.BasePath
			if len(step.Input) > 0 {
				inputBytes, err := json.Marshal(step.Input)
				if err != nil {
					tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to marshal step.Input: %v", err))
				} else {
					stepInput["yaml_config"] = string(inputBytes)
				}
			} else {
				stepInput["yaml_config"] = ""
			}
			stepInput["localworkdir"] = filepath.Join(taskDir, "in", fmt.Sprintf("seq%d", stepSeq))
		}
		if strings.ToLower(step.Plugin) == "opengauss" && strings.TrimSpace(step.Command) != "" {
			stepInput["query"] = step.Command
		}
		// 添加插件连接配置
		if step.PluginConn != nil {
			for k, v := range step.PluginConn {
				stepInput[k] = fmt.Sprintf("%v", v)
			}
		}

		// 写入步骤开始日志
		tm.writeTaskLog(task.ID, fmt.Sprintf("Starting step %d: %s", stepSeq, step.Plugin))
		var pluginOutType string
		// 通过Go插件(.so文件)执行。如果插件未加载则尝试自动加载。
		if tm.pluginManager != nil { // 检查插件管理器是否可用
			// 检查插件是否可用，如果可用则执行
			if _, getErr := tm.pluginManager.GetPlugin(step.Plugin, step.PluginVersion); getErr == nil {
				tm.writeTaskLog(task.ID, fmt.Sprintf("Executing plugin %s for step %d", step.Plugin, stepSeq))
				// 创建MQTT报告器，用于实时报告插件执行进度
				reporter := &mqttReporter{
					taskID:   task.ID,                                                      // 关联的任务ID
					tm:       tm,                                                           // 任务管理器引用
					stepSeq:  stepSeq,                                                      // 当前步骤序号
					logDir:   filepath.Join(taskDir, "log", fmt.Sprintf("seq%d", stepSeq)), // 日志目录
					fileName: fmt.Sprintf("%s.log", step.Plugin),                           // 日志文件名
				}
				pluginOutType, err = tm.pluginManager.GetPluginOutputType(step.Plugin, step.PluginVersion)
				if err != nil {
					tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to get plugin output type: %v", err))
					continue
				}
				// 执行插件并获取结果
				out, err = tm.pluginManager.ExecutePluginWithProgress(step.Plugin, step.PluginVersion, task.ID, stepInput, reporter)
			} else {
				// 插件不可用，设置错误
				err = fmt.Errorf("plugin %s unavailable", step.Plugin)
				tm.writeTaskLog(task.ID, fmt.Sprintf("Plugin %s unavailable", step.Plugin))
			}
		} else {
			// 插件管理器不可用，设置错误
			err = fmt.Errorf("plugin manager unavailable")
			tm.writeTaskLog(task.ID, "Plugin manager unavailable")
		}

		// 检查步骤执行是否出错
		if err != nil {
			log.Printf("Error executing step %d with plugin %s for task %s: %v", stepSeq, step.Plugin, task.ID, err) // 记录错误日志
			tm.writeTaskLog(task.ID, fmt.Sprintf("Error executing step %d with plugin %s: %v", stepSeq, step.Plugin, err))
			// 发送失败状态更新到MQTT
			//tm.sendStatusUpdate(task.ID, "failed", stepSeq, 0, err.Error())

			// 检查是否应该在失败时终止任务
			shouldTerminate := tm.shouldFailTerminate(task, stepSeq)

			if shouldTerminate {
				// 更新任务状态为失败
				tm.mutex.Lock()
				task.Status = TaskStatusFailed
				task.UpdatedAt = time.Now()
				task.EndTime = time.Now()
				tm.mutex.Unlock()
				// 发送失败状态更新
				message := fmt.Sprintf("Step %d failed with plugin %s: %v", stepSeq, step.Plugin, err)
				tm.sendStatusAndSave(task, TaskStatusFailed, message)

				// 对于定时任务，从调度器中移除
				if task.TaskType == "cycle" {
					tm.cancelScheduledTask(task.ID)
					tm.writeTaskLog(task.ID, fmt.Sprintf("Removed cycle task %s from scheduler", task.ID))
				}

				// 将任务状态保存到本地存储
				if err := tm.saveTasks(); err != nil {
					log.Printf("Warning: failed to save tasks: %v", err)
					tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to save tasks: %v", err))
				}

				// 标记任务完成状态，以便日志同步器知道任务已经完成
				if tm.logSyncer != nil {
					tm.logSyncer.SetTaskCompleted(task.ID, true)
				}

				tm.writeTaskLog(task.ID, fmt.Sprintf("Task %s failed: %v", task.ID, err))
				log.Printf("Task %s failed: %v", task.ID, err)
				return // 任务执行失败，直接返回
			} else {
				// 继续执行下一个步骤，记录错误但不终止任务
				log.Printf("Continuing task %s execution despite step %d failure (failTerminate=false)", task.ID, stepSeq)
				tm.writeTaskLog(task.ID, fmt.Sprintf("Continuing task execution despite step %d failure: %v", stepSeq, err))
				//tm.sendStatusUpdate(task.ID, "running", stepSeq, 0, fmt.Sprintf("Step %d failed: %v, continuing execution", stepSeq, err))
				continue
			}
		}
		// 如果步骤执行成功且有输出结果
		if out != nil {
			// 将输出结果写入文件
			outDir := filepath.Join(taskDir, "out", fmt.Sprintf("seq%d", stepSeq))
			if err := os.MkdirAll(outDir, 0755); err == nil { // 创建输出目录
				outBytes, _ := json.MarshalIndent(out, "", "  ")                       // 将输出结果格式化为JSON
				_ = os.WriteFile(filepath.Join(outDir, "result.json"), outBytes, 0644) // 写入结果文件
				tm.writeTaskLog(task.ID, fmt.Sprintf("Step %d execution completed, output saved to seq%d/result.json", stepSeq, stepSeq))
			} else {
				tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to create output directory for step %d: %v", stepSeq, err))
			}
			switch pluginOutType {
			case TaskOutputMonitor:
				if out["stderr"] == "" {
					jsonPayload := out["stdout"]
					if _, err := tm.PostResultData(jsonPayload); err != nil {
						log.Printf("Failed to post status data: %v", err)
						tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to post result data: %v", err))
					} else {
						// 记录任务状态变更
						log.Printf("post result data success to server")
					}
				}
			case TaskOutputTxt:
				// 检查数据库是否初始化
				if db.DB != nil {
					db.InsertTaskResultSync(&db.TaskResultSync{
						TaskID:    task.ID,
						Step:      fmt.Sprintf("%d", stepSeq),
						CreateAt:  time.Now().Format("2006-01-02 15:04:05"),
						IsSuccess: false,
					})
				} else {
					log.Printf("Warning: database not initialized, skipping task result sync")
				}
			case TaskOutputFile:
				// 调用remote下的http_send的
				// sender := remote.NewFileUploader(remote.FileUploaderConfig{
				// 	URL: "http://192.168.66.13:8081/ceamcore/agent/dispatch",
				// })
				// // 上传文件
				// if err := sender.UploadFile(filepath.Join(outDir, "result.json")); err != nil {
				tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to upload file: %v", err))
				// }
			}

		}

		// 更新当前执行步骤，以便在任务中断后能够从中断的位置继续执行
		tm.mutex.Lock()
		task.CurrentStep = i + 1 // 更新为下一个步骤的索引
		task.UpdatedAt = time.Now()
		tm.mutex.Unlock()

		// 保存任务状态到本地存储，确保CurrentStep的更新被持久化
		if err := tm.saveTasks(); err != nil {
			log.Printf("Warning: failed to save tasks: %v", err)
			tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to save tasks: %v", err))
		}
		tm.writeTaskLog(task.ID, fmt.Sprintf("Updated current step to %d", task.CurrentStep))
	}

	// 检查任务类型，如果是一次性任务则设置为完成状态
	if task.TaskType != "cycle" {
		// 所有插件执行完成后，更新任务为完成状态
		completedAt := time.Now() // 记录完成时间
		tm.mutex.Lock()
		task.Status = TaskStatusCompleted // 设置任务状态为已完成
		task.CurrentStep = 0              // 重置当前步骤为0，以便下次执行时从第一步开始
		task.UpdatedAt = time.Now()       // 更新最后修改时间
		task.CompletedAt = &completedAt   // 设置完成时间
		task.EndTime = time.Now()         // 设置结束时间
		tm.mutex.Unlock()

		// 发送完成状态更新
		message := "Task completed successfully"
		tm.sendStatusAndSave(task, TaskStatusCompleted, message)

		// 将任务状态保存到本地存储
		if err := tm.saveTasks(); err != nil {
			log.Printf("Warning: failed to save tasks: %v", err)
			tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to save tasks: %v", err))
		}

		// 标记任务完成状态，以便日志同步器知道任务已经完成
		if tm.logSyncer != nil {
			tm.logSyncer.SetTaskCompleted(task.ID, true)
		}

		tm.writeTaskLog(task.ID, fmt.Sprintf("Task %s completed successfully", task.ID))
		log.Printf("Task %s completed", task.ID) // 记录任务完成日志
	} else {
		// 检查定时任务是否已经到期
		isExpired := false
		if task.CronConfig != nil && task.CronConfig.EndTime != "" {
			endTime, err := time.Parse("2006-01-02 15:04:05", task.CronConfig.EndTime)
			if err == nil {
				isExpired = time.Now().After(endTime) // 如果当前时间超过了结束时间，则任务已到期
			}
		}

		if isExpired {
			// 定时任务已到期，状态设置为completed，并取消调度
			completedAt := time.Now()
			tm.mutex.Lock()
			task.Status = TaskStatusCompleted // 定时任务到期后，状态设置为completed
			task.CurrentStep = 0              // 重置当前步骤为0
			task.UpdatedAt = time.Now()       // 更新最后修改时间
			task.CompletedAt = &completedAt   // 设置完成时间
			task.EndTime = time.Now()         // 设置结束时间
			tm.mutex.Unlock()

			// 取消定时任务的调度
			tm.cancelScheduledTask(task.ID)
			tm.writeTaskLog(task.ID, fmt.Sprintf("Removed cycle task %s from scheduler (expired)", task.ID))

			// 发送完成状态更新
			message := "Task expired"
			tm.sendStatusAndSave(task, TaskStatusCompleted, message)

			// 将任务状态保存到本地存储
			if err := tm.saveTasks(); err != nil {
				log.Printf("Warning: failed to save tasks: %v", err)
				tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to save tasks: %v", err))
			}

			// 标记任务完成状态，以便日志同步器知道任务已经完成
			if tm.logSyncer != nil {
				tm.logSyncer.SetTaskCompleted(task.ID, true)
			}

			tm.writeTaskLog(task.ID, fmt.Sprintf("Cycle task %s expired and completed", task.ID))
			log.Printf("Cycle task %s expired and completed", task.ID) // 记录任务到期完成日志
		} else {
			// 检查任务是否已经过期（下次调度时间在结束时间之后）
			expired := false
			if task.CronConfig != nil && task.CronConfig.EndTime != "" {
				endTime, err := time.Parse("2006-01-02 15:04:05", task.CronConfig.EndTime)
				if err == nil {
					// 计算下次调度时间
					nextScheduledTime, err := tm.getNextScheduledTime(task.CronConfig.CronExpression)
					if err == nil && nextScheduledTime.After(endTime) {
						// 下次调度时间在结束时间之后，任务过期
						expired = true
						log.Printf("Task %s next scheduled time (%s) is after end time (%s), marking as completed",
							task.ID, nextScheduledTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))
						tm.writeTaskLog(task.ID, fmt.Sprintf("Next scheduled time (%s) is after end time (%s), marking task as completed",
							nextScheduledTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05")))

						// 取消定时任务的调度
						tm.cancelScheduledTask(task.ID)
						tm.writeTaskLog(task.ID, fmt.Sprintf("Removed cycle task %s from scheduler (expired)", task.ID))

						// 更新任务状态为完成
						tm.mutex.Lock()
						task.Status = TaskStatusCompleted
						task.CurrentStep = 0
						task.UpdatedAt = time.Now()
						completedAt := time.Now()
						task.CompletedAt = &completedAt
						task.EndTime = time.Now()
						tm.mutex.Unlock()

						// 发送完成状态更新
						message := "Task expired (next scheduled time after end time)"
						tm.sendStatusAndSave(task, TaskStatusCompleted, message)

						tm.writeTaskLog(task.ID, fmt.Sprintf("Cycle task %s expired and completed", task.ID))
						log.Printf("Cycle task %s expired and completed", task.ID)
					}
				}
			}

			// 如果任务未过期，设置为waiting状态
			if !expired {
				tm.mutex.Lock()
				task.Status = TaskStatusWaiting // 定时任务执行完成后，状态设置为waiting
				task.CurrentStep = 0            // 重置当前步骤为0，以便下次执行时从第一步开始
				task.UpdatedAt = time.Now()     // 更新最后修改时间
				tm.mutex.Unlock()

				// 发送等待状态更新
				message := "Task waiting for next execution"
				tm.sendStatusAndSave(task, TaskStatusWaiting, message)

				tm.writeTaskLog(task.ID, fmt.Sprintf("Cycle task %s executed successfully, waiting for next execution", task.ID))
				log.Printf("Cycle task %s executed (remains waiting)", task.ID) // 记录任务执行日志
			}
		}

		// 将任务状态保存到本地存储
		if err := tm.saveTasks(); err != nil {
			log.Printf("Warning: failed to save tasks: %v", err) // 保存失败时记录警告日志
			tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to save tasks: %v", err))
		}
	}
}

// shouldFailTerminate 检查是否应在指定步骤失败时终止任务
// 根据任务的 failTerminate 属性决定是否终止
func (tm *TaskManager) shouldFailTerminate(task *Task, stepSeq int) bool {
	return task.FailTerminate // 使用任务配置的失败终止属性
}

// checkPlugin checks if a plugin is available and downloads it if necessary
func (tm *TaskManager) checkPlugin(plugin TaskPluginInfo) error {
	return nil
}

// loadTasks 从本地存储加载任务数据
// 检查并创建数据目录，读取任务文件并解析为任务结构体
// 返回错误信息，如果加载过程中出现问题
func (tm *TaskManager) loadTasks() error {
	// 如果数据目录不存在则创建它
	dataDir := filepath.Join(".", "data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %v", err) // 创建目录失败时返回错误
		}
	}

	// 从文件加载任务数据
	tasksFile := filepath.Join(dataDir, "tasks.json")
	if _, err := os.Stat(tasksFile); os.IsNotExist(err) {
		// 如果任务文件不存在，返回空映射
		return nil
	}

	// 读取任务文件内容
	content, err := os.ReadFile(tasksFile)
	if err != nil {
		return fmt.Errorf("failed to read tasks file: %v", err) // 读取文件失败时返回错误
	}

	// 解析任务JSON数据
	var tasks map[string]*Task
	if err := json.Unmarshal(content, &tasks); err != nil {
		return fmt.Errorf("failed to parse tasks file: %v", err) // 解析JSON失败时返回错误
	}

	tm.tasks = tasks                                       // 将解析后的任务赋值给管理器的内部存储
	log.Printf("Loaded %d tasks from storage", len(tasks)) // 记录加载任务数量的日志
	return nil
}

// Removed subprocess plugin execution to enforce .so plugins only

// clearTaskProcess 清理指定任务的进程记录
// 从进程映射表中删除任务，并重置任务的相关状态
// 参数：taskID - 要清理进程记录的任务ID
func (tm *TaskManager) clearTaskProcess(taskID string) {
	tm.processMutex.Lock()       // 获取进程锁，防止并发访问
	delete(tm.processes, taskID) // 从进程映射表中删除指定任务的进程记录
	tm.processMutex.Unlock()     // 释放进程锁
	tm.mutex.Lock()              // 获取任务锁
	task, ok := tm.tasks[taskID] // 查找指定ID的任务
	if ok {
		task.PID = 0              // 重置进程ID为0
		task.EndTime = time.Now() // 设置结束时间为当前时间
	}
	tm.mutex.Unlock() // 释放任务锁
}

// stopTaskProcessLocked 停止指定任务的进程（需要外部加锁）
// 通过进程ID终止任务关联的进程，并清理相关记录
// 参数：taskID - 要停止进程的任务ID
func (tm *TaskManager) stopTaskProcessLocked(taskID string) {
	tm.processMutex.Lock()          // 获取进程锁，防止并发访问
	cmd, ok := tm.processes[taskID] // 查找指定任务的进程命令对象
	tm.processMutex.Unlock()        // 立即释放进程锁
	if !ok || cmd == nil || cmd.Process == nil {
		return // 如果进程不存在或无效，则直接返回
	}
	_ = cmd.Process.Kill()      // 强制终止进程
	tm.clearTaskProcess(taskID) // 清理任务进程记录
}

// saveTasks 保存任务数据到本地存储
// 将内存中的任务数据序列化为JSON格式并写入文件
// 返回错误信息，如果保存过程中出现问题
func (tm *TaskManager) saveTasks() error {
	// 如果数据目录不存在则创建它
	dataDir := filepath.Join(".", "data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %v", err) // 创建目录失败时返回错误
		}
	}

	// 将任务数据序列化为JSON格式
	content, err := json.MarshalIndent(tm.tasks, "", "  ") // 格式化JSON输出
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %v", err) // 序列化失败时返回错误
	}

	// 将JSON内容写入文件
	tasksFile := filepath.Join(dataDir, "tasks.json")
	if err := os.WriteFile(tasksFile, content, 0644); err != nil {
		return fmt.Errorf("failed to write tasks file: %v", err) // 写入文件失败时返回错误
	}

	log.Printf("Saved %d tasks to storage", len(tm.tasks)) // 记录保存任务数量的日志
	return nil
}

// 接收到任务后，需要按步骤执行文件下载的任务，若出现下载失败则需要立即终止任务
func (tm *TaskManager) downloadFileByTaskStep(task *Task) error {
	// 创建任务目录结构
	taskDir := filepath.Join("tasks", task.ID)

	// 创建标准子目录：输入、日志、输出
	dirs := []string{"in", "log", "out"}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(taskDir, d), 0755); err != nil {
			log.Printf("Error creating task subdirectory: %v", err)
			tm.writeTaskLog(task.ID, fmt.Sprintf("Error creating task subdirectory: %v", err))
			return fmt.Errorf("failed to create task subdirectory: %v", err)
		}
	}

	// 处理任务步骤列表
	for _, step := range task.StepList {
		// 确定步骤序列号
		seq := step.Sequence
		stepDirName := fmt.Sprintf("seq%d", seq) // 步骤目录名
		tm.writeTaskLog(task.ID, fmt.Sprintf("Download File Step %d", seq))

		// 为每个步骤创建对应的子目录
		for _, d := range dirs {
			if err := os.MkdirAll(filepath.Join(taskDir, d, stepDirName), 0755); err != nil {
				log.Printf("Error creating step directory: %v", err)
				tm.writeTaskLog(task.ID, fmt.Sprintf("Error creating step directory: %v", err))
				return fmt.Errorf("failed to create step directory: %v", err)
			}
		}

		// 遍历当前步骤的所有输入文件并下载
		for _, inputFile := range step.Input {
			stepDir := filepath.Join(taskDir, "in", stepDirName)
			if fileName := inputFile.FileName; fileName != "" {
				if fileUrl := inputFile.FileUrl; fileUrl != "" {
					localPath := inputFile.LocalPath

					// 确定目标路径
					var targetPath string
					if localPath == "" {
						// 如果提供了localPath，使用绝对路径
						targetPath = stepDir
					} else {
						targetPath = filepath.Join(stepDir, localPath)
					}

					if err := os.MkdirAll(targetPath, 0755); err != nil {
						log.Printf("Error creating target directory: %v", err)
						tm.writeTaskLog(task.ID, fmt.Sprintf("Error creating target directory: %v", err))
						return err
					}
					FilePath := filepath.Join(targetPath, fileName)
					// 如果是HTTP URL，执行下载
					if strings.HasPrefix(fileUrl, "http") {
						if err := downloadFile(fileUrl, FilePath); err != nil {
							log.Printf("Failed to download file: %v", err)
							tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to download file: %v", err))
							return err
						}
					} else {
						// 对于非HTTP URL，创建模拟内容
						os.WriteFile(targetPath, []byte("dummy content"), 0644)
					}
				}
			}
		}
	}
	return nil
}

// HandleTaskDispatch 处理任务分发消息
// 接收来自MQTT的消息，解析任务信息并创建和启动任务
// 参数：payload - 包含任务信息的字节数组
// 返回：错误信息，如果处理过程中出现问题
func (tm *TaskManager) HandleTaskDispatch(payload []byte) error {

	// 在goroutine中处理任务，以支持同时执行多个任务
	go func() {
		// 解析接收到的任务分发负载
		var dispatch TaskDispatchPayload
		if err := json.Unmarshal(payload, &dispatch); err != nil {
			log.Printf("Error unmarshaling task dispatch payload: %v", err) // 解析失败时记录错误日志
			return
		}

		// 确定任务类型 (一次性任务还是周期性任务)
		taskType := "once" // 默认为一次性任务
		if dispatch.TaskType == "cycle" {
			taskType = "cycle"
		}

		// 处理插件升级任务
		if dispatch.TaskType == "plugin_upgrade" {
			tm.handlePluginUpgradeTask(&dispatch)
			return
		}

		// 处理plus自身升级任务
		if dispatch.TaskType == "plus_upgrade" {
			tm.handlePlusUpgradeTask(&dispatch)
			return
		}

		// 检查是否正在接受新任务（升级期间不接收新任务）
		tm.mutex.RLock()
		acceptNewTask := tm.acceptNewTask
		tm.mutex.RUnlock()
		if !acceptNewTask {
			log.Printf("Rejecting new task %s: plus is preparing for upgrade", dispatch.TaskID)
			return
		}
		task := &Task{
			ID:            dispatch.TaskID,
			Name:          dispatch.TaskName,
			StepList:      dispatch.StepList,
			TaskType:      taskType,
			ExecuteMode:   dispatch.ExecuteMode,
			FailTerminate: dispatch.FailTerminate,
			CronConfig:    dispatch.CronConfig,
			IsPrepared:    false,
			Status:        TaskStatusReceived,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		// 任务过来后应立即保存到tasks.json
		tm.mutex.Lock()
		tm.tasks[dispatch.TaskID] = task
		if err := tm.saveTasks(); err != nil {
			log.Printf("Warning: failed to save tasks immediately after dispatch: %v", err)
		}
		tm.mutex.Unlock()
		if tm.logSyncer != nil {
			// 初始化任务日志同步状态
			tm.logSyncer.InitStatus(task.ID)
			// 启动日志同步协程
			if err := tm.logSyncer.StartTaskSync(task.ID); err != nil {
				log.Printf("Failed to start log sync for task %s: %v", task.ID, err)
			}
		}
		// 获取当前代理的ID

		// 创建任务目录
		taskDir := filepath.Join("tasks", task.ID)
		if err := os.MkdirAll(taskDir, 0755); err != nil {
			log.Printf("Error creating task directory: %v", err) // 创建目录失败时记录错误

			return
		}
		// 创建标准子目录：输入、日志、输出
		dirs := []string{"in", "log", "out"}
		for _, d := range dirs {
			if err := os.MkdirAll(filepath.Join(taskDir, d), 0755); err != nil {
				log.Printf("Error creating task subdirectory: %v", err) // 创建子目录失败时记录错误
				return
			}
		}
		//创建后发送任务消息，任务状态为received
		message := fmt.Sprintf("Task %s received", dispatch.TaskID)
		tm.sendStatusAndSave(task, TaskStatusReceived, message)

		if err := tm.downloadFileByTaskStep(task); err != nil {
			//无法下载文件，任务失败
			tm.writeTaskLog(dispatch.TaskID, fmt.Sprintf("Failed to download file: %v", err))
			log.Printf("Failed to download files for task %s: %v", dispatch.TaskID, err)
			// 使用PostStatusData发送失败状态更新到MQTT
			message = fmt.Sprintf("Failed to download file: %v", err)
			tm.sendStatusAndSave(task, TaskStatusFailed, message)
			// 标记任务完成状态，以便日志同步器知道任务已经完成
			if tm.logSyncer != nil {
				tm.logSyncer.SetTaskCompleted(task.ID, true)
			}
			// 更新任务状态，并保存到tasks.json
			task.Status = TaskStatusFailed
			task.UpdatedAt = time.Now()
			tm.mutex.Lock()
			tm.tasks[task.ID] = task
			tm.mutex.Unlock()
			tm.saveTasks()
			return
		}
		task.IsPrepared = true
		task.Status = TaskStatusAssigned
		task.UpdatedAt = time.Now()
		tm.mutex.Lock()
		tm.tasks[task.ID] = task
		tm.mutex.Unlock()
		tm.saveTasks()
		// 如果是定时任务，添加到调度器
		if taskType == "cycle" {
			if err := tm.scheduleTask(task); err != nil {
				log.Printf("Failed to schedule task %s: %v", task.ID, err)
			} else {
				cronExpr := ""
				if task.CronConfig != nil {
					cronExpr = task.CronConfig.CronExpression
				}
				log.Printf("Task %s scheduled with cron: %s", task.ID, cronExpr)
			}
		}

		message = fmt.Sprintf("Task %s assigned", dispatch.TaskID)
		tm.sendStatusAndSave(task, TaskStatusAssigned, message)
		// 使用PostStatusData发送已下发状态更新到MQTT
		// 自动启动任务
		if err := tm.StartTask(task.ID); err != nil {
			log.Printf("Error starting task %s: %v", task.ID, err) // 启动失败时记录错误
		}
	}()

	return nil
}

// handlePluginUpgradeTask 处理插件升级任务
func (tm *TaskManager) handlePluginUpgradeTask(dispatch *TaskDispatchPayload) {
	// 创建任务记录

	task := &Task{
		ID:            dispatch.TaskID,
		Name:          dispatch.TaskName,
		TaskType:      "once",
		ExecuteMode:   dispatch.ExecuteMode,
		FailTerminate: dispatch.FailTerminate,
		IsPrepared:    true,
		Status:        TaskStatusReceived,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		StartTime:     time.Now(),
	}

	// 保存任务
	tm.mutex.Lock()
	tm.tasks[dispatch.TaskID] = task
	tm.saveTasks()
	tm.mutex.Unlock()

	// 创建任务目录
	taskDir := filepath.Join("tasks", task.ID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		log.Printf("Error creating task directory: %v", err)
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, fmt.Sprintf("Failed to create task directory: %v", err))
		return
	}

	// 创建标准子目录：输入、日志、输出
	dirs := []string{"in", "log", "out"}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(taskDir, d), 0755); err != nil {
			log.Printf("Error creating task subdirectory: %v", err)
			tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, fmt.Sprintf("Failed to create task subdirectory: %v", err))
			return
		}
	}
	//创建后发送任务消息，任务状态为received
	message := fmt.Sprintf("Plugin upgrade task %s received", dispatch.TaskID)
	tm.sendStatusAndSave(task, TaskStatusReceived, message)

	// 检查插件升级配置
	if dispatch.PluginUpgrade == nil {
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, "Missing plugin upgrade configuration")
		return
	}

	pu := dispatch.PluginUpgrade
	if pu.PluginName == "" || pu.FileName == "" || pu.DownloadURL == "" {
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, "Invalid plugin upgrade configuration: missing required fields")
		return
	}

	// 下载插件文件
	pluginDir := filepath.Join(".", "plugins")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, fmt.Sprintf("Failed to create plugin directory: %v", err))
		return
	}

	pluginPath := filepath.Join(pluginDir, pu.FileName)
	if err := downloadFile(pu.DownloadURL, pluginPath); err != nil {
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, fmt.Sprintf("Failed to download plugin: %v", err))
		return
	}

	// 尝试加载插件以获取版本信息
	p, err := plugin.Open(pluginPath)
	if err != nil {
		// 清理下载的文件
		os.Remove(pluginPath)
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, fmt.Sprintf("Failed to load plugin: %v", err))
		return
	}

	// 查找New函数
	newFunc, err := p.Lookup("New")
	if err != nil {
		// 清理下载的文件
		os.Remove(pluginPath)
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, fmt.Sprintf("Failed to find New function in plugin: %v", err))
		return
	}

	// 断言New函数类型
	pluginFactory, ok := newFunc.(func() Plugin)
	if !ok {
		// 清理下载的文件
		os.Remove(pluginPath)
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, "Invalid plugin factory type")
		return
	}

	// 创建插件实例
	pluginInstance := pluginFactory()
	pluginVersion := pluginInstance.Version()

	// 检查版本是否已存在
	tm.pluginManager.mutex.RLock()
	if versions, exists := tm.pluginManager.plugins[pu.PluginName]; exists {
		if _, versionExists := versions[pluginVersion]; versionExists {
			tm.pluginManager.mutex.RUnlock()
			// 清理下载的文件
			os.Remove(pluginPath)
			tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, fmt.Sprintf("Plugin version %s already exists", pluginVersion))
			return
		}
	}
	tm.pluginManager.mutex.RUnlock()

	// 初始化插件
	if err := pluginInstance.Initialize(""); err != nil {
		// 清理下载的文件
		os.Remove(pluginPath)
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, fmt.Sprintf("Failed to initialize plugin: %v", err))
		return
	}

	// 加载插件到插件管理器
	if _, err := tm.pluginManager.LoadPlugin(pluginPath); err != nil {
		// 清理下载的文件
		os.Remove(pluginPath)
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, fmt.Sprintf("Failed to load plugin into manager: %v", err))
		return
	}

	// 插件升级成功
	tm.updateTaskStatusAndStopLogSync(task, TaskStatusCompleted, fmt.Sprintf("Plugin %s upgraded to version %s successfully", pu.PluginName, pluginVersion))
}

// handlePlusUpgradeTask 处理plus自身升级任务
func (tm *TaskManager) handlePlusUpgradeTask(dispatch *TaskDispatchPayload) {
	log.Printf("Handling plus upgrade task: %s", dispatch.TaskID)

	// 创建任务记录
	task := &Task{
		ID:            dispatch.TaskID,
		Name:          dispatch.TaskName,
		TaskType:      "once",
		ExecuteMode:   dispatch.ExecuteMode,
		FailTerminate: dispatch.FailTerminate,
		IsPrepared:    true,
		Status:        TaskStatusReceived,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		StartTime:     time.Now(),
	}

	// 保存任务
	tm.mutex.Lock()
	tm.tasks[dispatch.TaskID] = task
	tm.saveTasks()
	tm.mutex.Unlock()

	message := fmt.Sprintf("Plus upgrade task %s received", dispatch.TaskID)
	tm.sendStatusAndSave(task, TaskStatusReceived, message)

	// 检查plus升级配置
	if dispatch.PlusUpgrade == nil {
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, "Missing plus upgrade configuration")
		return
	}

	pu := dispatch.PlusUpgrade
	if pu.Version == "" || pu.FileName == "" || pu.DownloadURL == "" {
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, "Invalid plus upgrade configuration: missing required fields")
		return
	}

	// 检查是否已经在升级中
	tm.mutex.Lock()
	if tm.upgradeState != nil && tm.upgradeState.IsUpgrading {
		tm.mutex.Unlock()
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, "Plus is already in upgrade process")
		return
	}
	tm.mutex.Unlock()

	// 更新任务状态为运行中
	tm.updateTaskStatusAndStopLogSync(task, TaskStatusRunning, "Downloading new version")

	// 下载新版本文件到/agent/目录
	agentDir := filepath.Join(".", "agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, fmt.Sprintf("Failed to create agent directory: %v", err))
		return
	}

	newBinaryPath := filepath.Join(agentDir, pu.FileName)
	if err := downloadFile(pu.DownloadURL, newBinaryPath); err != nil {
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, fmt.Sprintf("Failed to download new version: %v", err))
		return
	}

	// 验证下载的文件
	if _, err := os.Stat(newBinaryPath); err != nil {
		tm.updateTaskStatusAndStopLogSync(task, TaskStatusFailed, fmt.Sprintf("Downloaded file not found: %v", err))
		return
	}

	log.Printf("New version downloaded to: %s", newBinaryPath)

	// 设置升级状态
	tm.mutex.Lock()
	tm.upgradeState = &PlusUpgradeState{
		TargetVersion: pu.Version,
		NewBinaryPath: newBinaryPath,
		IsUpgrading:   true,
		CanUpgrade:    false,
		DownloadTime:  time.Now(),
	}
	// 停止接受新任务
	tm.acceptNewTask = false
	tm.mutex.Unlock()

	tm.updateTaskStatusAndStopLogSync(task, TaskStatusRunning, fmt.Sprintf("New version %s downloaded, preparing for safe upgrade", pu.Version))

	// 启动升级准备协程
	go tm.prepareForUpgrade(task, pu.Version)
}

// prepareForUpgrade 准备升级，等待任务完成
func (tm *TaskManager) prepareForUpgrade(task *Task, targetVersion string) {
	log.Printf("Starting upgrade preparation for version %s", targetVersion)

	// 第一步：取消所有定时任务的调度（保留任务数据，重启后可以重新调度）
	tm.mutex.Lock()
	// 复制任务ID列表，避免在遍历时修改映射
	var taskIDs []string
	for taskID := range tm.scheduledJobs {
		taskIDs = append(taskIDs, taskID)
	}
	tm.mutex.Unlock()

	// 逐个取消调度
	for _, taskID := range taskIDs {
		tm.mutex.Lock()
		tm.cancelScheduledTask(taskID)
		tm.mutex.Unlock()
	}

	log.Printf("All scheduled tasks have been unscheduled")

	// 等待任务完成或达到安全升级状态
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if tm.canSafelyUpgrade() {
				log.Printf("Plus is ready for upgrade to version %s", targetVersion)

				// 标记可以升级
				tm.mutex.Lock()
				if tm.upgradeState != nil {
					tm.upgradeState.CanUpgrade = true
				}
				tm.mutex.Unlock()

				// 更新任务状态为完成
				tm.updateTaskStatusAndStopLogSync(task, TaskStatusCompleted, fmt.Sprintf("Ready for upgrade to version %s", targetVersion))
				return
			}

			// 检查任务状态，更新进度信息
			tm.mutex.RLock()
			runningCount := 0
			waitingCount := 0
			for _, t := range tm.tasks {
				if t.Status == TaskStatusRunning {
					runningCount++
				} else if t.Status == TaskStatusWaiting {
					waitingCount++
				}
			}
			tm.mutex.RUnlock()

			log.Printf("Upgrade preparation: %d running tasks, %d waiting tasks", runningCount, waitingCount)
		}
	}
}

// canSafelyUpgrade 检查是否可以安全升级
func (tm *TaskManager) canSafelyUpgrade() bool {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	for _, task := range tm.tasks {
		// 检查一次性任务是否还在运行
		if task.TaskType == "once" && task.Status == TaskStatusRunning {
			return false
		}
		// 检查定时任务是否还在运行
		if task.TaskType == "cycle" && task.Status == TaskStatusRunning {
			return false
		}
	}

	return true
}

// 发送状态并保存状态
func (tm *TaskManager) sendStatusAndSave(task *Task, status TaskStatus, message string) {
	updatedAtStr := task.UpdatedAt.Format("2006-01-02 15:04:05")
	if _, err := tm.PostStatusData(map[string]interface{}{
		"client_id":  resolveAgentID(),
		"task_id":    task.ID,
		"status":     string(status),
		"updated_at": updatedAtStr,
		"message":    message,
	}); err != nil {
		log.Printf("Failed to post status data: %v", err)
		tm.recordTaskStatus(task, updatedAtStr, message, false)
	} else {
		tm.recordTaskStatus(task, updatedAtStr, message, true)
	}
}

// updateTaskStatus 更新任务状态并发送通知
func (tm *TaskManager) updateTaskStatusAndStopLogSync(task *Task, status TaskStatus, message string) {
	tm.mutex.Lock()
	task.Status = status
	task.UpdatedAt = time.Now()
	if status == TaskStatusCompleted || status == TaskStatusFailed {
		task.EndTime = time.Now()
		if status == TaskStatusCompleted {
			completedAt := time.Now()
			task.CompletedAt = &completedAt
		}
	}
	tm.saveTasks()
	tm.mutex.Unlock()

	// 发送状态更新
	agentID := resolveAgentID()
	updatedAtStr := task.UpdatedAt.Format("2006-01-02 15:04:05")
	if _, err := tm.PostStatusData(map[string]interface{}{
		"client_id":  agentID,
		"task_id":    task.ID,
		"status":     string(status),
		"updated_at": updatedAtStr,
		"message":    message,
	}); err != nil {
		log.Printf("Failed to post status data: %v", err)
	}

	// 记录任务状态变更
	tm.recordTaskStatus(task, updatedAtStr, message, true)

	// 标记任务完成状态，以便日志同步器知道任务已经完成
	if tm.logSyncer != nil {
		tm.logSyncer.SetTaskCompleted(task.ID, true)
	}
}

// downloadFile 从指定 URL 下载文件
// 这是一个辅助函数，用于从网络下载配置文件或其他资源
// 参数：
//
//	url - 文件下载地址
//	filepath - 文件保存路径
//
// 返回值：
//
//	错误信息，如果下载过程中出现问题
func downloadFile(url, filepath string) error {
	// 发送 HTTP GET 请求到指定URL
	resp, err := http.Get(url)
	if err != nil {
		return err // 请求失败时直接返回错误
	}
	defer resp.Body.Close() // 确保响应体被关闭，防止资源泄露

	// 检查HTTP响应状态码，确保请求成功
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status) // 状态码不是200时返回错误
	}

	// 创建本地文件用于保存下载的内容
	out, err := os.Create(filepath)
	if err != nil {
		return err // 创建文件失败时返回错误
	}
	defer out.Close() // 确保文件被关闭，防止资源泄露

	// 将HTTP响应体的内容复制到本地文件
	_, err = io.Copy(out, resp.Body)
	return err // 返回复制过程中可能发生的错误
}

func (tm *TaskManager) sendStatusUpdate(taskID, status string, stepSeq int, progress float64, message string) {
	// 异步发送MQTT消息以避免阻塞
	go func() {
		// 获取当前代理的ID
		agentID := resolveAgentID()

		// 构建状态更新负载
		statusPayload := TaskStatusPayload{
			TaskID:    taskID,            // 任务ID
			Status:    status,            // 任务状态
			StepSeq:   stepSeq,           // 当前步骤序号
			Progress:  progress,          // 执行进度百分比
			Message:   message,           // 状态消息
			Timestamp: time.Now().Unix(), // 时间戳
		}

		// 通过MQTT发布状态更新
		if err := PublishMQTT(fmt.Sprintf("agent/%s/task/status", agentID), statusPayload); err != nil {
			log.Printf("Error sending status update for task %s: %v", taskID, err) // 发布失败时记录错误
		}
	}()
}

// resolveAgentID 解析代理ID
// 优先从环境变量AGENT_ID获取，如果不存在则尝试CLIENT_ID，都不存在则返回"unknown"
// 返回：解析得到的代理ID字符串
func resolveAgentID() string {
	// 首先尝试从AGENT_ID环境变量获取ID
	id := strings.TrimSpace(os.Getenv("AGENT_ID"))
	// 如果AGENT_ID为空或特殊值（"null", "undefined"），则尝试CLIENT_ID环境变量
	if id == "" || strings.EqualFold(id, "null") || strings.EqualFold(id, "undefined") {
		id = strings.TrimSpace(os.Getenv("CLIENT_ID"))
	}
	// 如果仍然为空或特殊值，则使用默认值"unknown"
	if id == "" || strings.EqualFold(id, "null") || strings.EqualFold(id, "undefined") {
		id = "unknown"
	}
	return id // 返回解析得到的ID
}

// mqttReporter MQTT报告器结构体
// 实现了报告接口，用于实时报告插件执行进度、完成状态和错误信息
// 通过MQTT发送状态更新，并将日志写入本地文件
type mqttReporter struct {
	taskID   string       // 关联的任务ID
	tm       *TaskManager // 任务管理器引用
	stepSeq  int          // 当前步骤序号
	logDir   string       // 日志文件目录
	fileName string       // 日志文件名
}

// OnProgress 报告插件执行进度
// 当插件执行过程中需要报告进度时调用此方法
// 参数：
//
//	taskID - 任务ID
//	pluginName - 插件名称
//	current - 当前进度数值
//	total - 总进度数值
//	message - 进度消息
func (r *mqttReporter) OnProgress(taskID, pluginName string, current, total int, message string) {
	// 计算进度百分比
	//progress := float64(current) / float64(total) * 100
	// 通过任务管理器发送进度状态更新到MQTT
	//r.tm.sendStatusUpdate(taskID, "running", r.stepSeq, progress, message)
	// 将进度信息写入本地日志文件
	r.logToFile(fmt.Sprintf("Progress: %d/%d %s", current, total, message))
}

// OnCompleted 报告插件执行完成状态
// 当插件执行完成时调用此方法
// 参数：
//
//	taskID - 任务ID
//	pluginName - 插件名称
//	success - 是否成功完成
//	message - 完成消息
func (r *mqttReporter) OnCompleted(taskID, pluginName string, success bool, message string) {
	// 根据执行结果确定状态
	//status := "completed" // 默认为完成状态
	// if !success {         // 如果执行失败
	// 	status = "failed" // 状态设为失败
	// }
	// 通过任务管理器发送完成状态更新到MQTT
	//r.tm.sendStatusUpdate(taskID, status, r.stepSeq, 100, message) // 进度设为100%
	// 将完成信息写入本地日志文件
	r.logToFile(fmt.Sprintf("Completed: success=%v %s", success, message))
}

// OnError 报告插件执行错误
// 当插件执行过程中发生错误时调用此方法
// 参数：
//
//	taskID - 任务ID
//	pluginName - 插件名称
//	err - 错误对象
func (r *mqttReporter) OnError(taskID, pluginName string, err error) {
	// 通过任务管理器发送错误状态更新到MQTT
	//r.tm.sendStatusUpdate(taskID, "failed", r.stepSeq, 0, err.Error()) // 进度设为0%，错误信息作为消息
	// 将错误信息写入本地日志文件
	r.logToFile(fmt.Sprintf("Error: %v", err))
}

// logToFile 将消息写入日志文件
// 辅助方法，将时间戳和消息内容追加到指定的日志文件
// 参数：message - 要写入的日志消息
func (r *mqttReporter) logToFile(message string) {
	// 如果日志目录未设置，则不写入日志
	if r.logDir == "" {
		return
	}
	// 确保日志目录存在
	if err := os.MkdirAll(r.logDir, 0755); err != nil {
		return
	}
	// 以追加模式打开或创建日志文件
	f, err := os.OpenFile(filepath.Join(r.logDir, r.fileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close() // 函数结束时关闭文件
	// 获取当前时间戳
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	// 将带时间戳的消息写入文件
	f.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, message))
}

// scheduleTask 将任务添加到调度器
// 参数：task - 要调度的任务
// 返回：错误信息，如果调度失败
func (tm *TaskManager) scheduleTask(task *Task) error {
	// 验证cron表达式格式
	if task.CronConfig == nil || task.CronConfig.CronExpression == "" {
		tm.writeTaskLog(task.ID, "cron expression is empty")
		return fmt.Errorf("cron expression is empty")
	}

	// 只有当任务类型为"cycle"（周期性）时才进行调度
	if task.TaskType != "" && task.TaskType != "cycle" {
		tm.writeTaskLog(task.ID, "task type is not cycle")
		return nil // 非周期性任务不需要调度
	}

	// 解析cron表达式以验证格式
	_, err := cron.ParseStandard(task.CronConfig.CronExpression)
	if err != nil {
		tm.writeTaskLog(task.ID, fmt.Sprintf("invalid cron expression: %v", err))
		return fmt.Errorf("invalid cron expression: %v", err)
	}

	// 分析cron表达式类型
	cronType := tm.analyzeCronExpression(task.CronConfig.CronExpression)
	tm.writeTaskLog(task.ID, fmt.Sprintf("cron type: %s", cronType))
	log.Printf("Task %s cron type: %s", task.ID, cronType)

	// 创建任务执行函数
	jobFunc := func() {
		// 检查任务是否在指定的时间范围内
		currentTime := time.Now()

		// 检查任务开始时间
		if task.CronConfig != nil && task.CronConfig.StartTime != "" {
			startTime, err := time.Parse("2006-01-02 15:04:05", task.CronConfig.StartTime)
			if err == nil && currentTime.Before(startTime) {
				log.Printf("Task %s has not reached start time yet (%s), skipping scheduled execution", task.ID, task.CronConfig.StartTime)
				tm.writeTaskLog(task.ID, fmt.Sprintf("Task %s has not reached start time yet (%s), skipping scheduled execution", task.ID, task.CronConfig.StartTime))
				return
			}
		}

		// 检查任务结束时间
		if task.CronConfig != nil && task.CronConfig.EndTime != "" {
			endTime, err := time.Parse("2006-01-02 15:04:05", task.CronConfig.EndTime)
			if err != nil {
				// 尝试其他时间格式
				endTime, err = time.Parse("2006-01-02T15:04:05Z", task.CronConfig.EndTime)
				if err != nil {
					endTime, err = time.Parse("2006-01-02T15:04:05.999999999Z", task.CronConfig.EndTime)
					if err != nil {
						log.Printf("Failed to parse end time %s: %v", task.CronConfig.EndTime, err)
					}
				}
			}

			if err == nil {
				// 确保endTime使用本地时区
				endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(),
					endTime.Hour(), endTime.Minute(), endTime.Second(), endTime.Nanosecond(),
					currentTime.Location())
				log.Printf("Checking task %s expiration in scheduler: now=%s (location=%s), endTime=%s (location=%s), now.After(endTime)=%v",
					task.ID, currentTime.Format("2006-01-02 15:04:05"), currentTime.Location(),
					endTime.Format("2006-01-02 15:04:05"), endTime.Location(),
					currentTime.After(endTime))

				if currentTime.After(endTime) {
					log.Printf("Task %s has passed end time (%s), skipping scheduled execution", task.ID, endTime.Format("2006-01-02 15:04:05"))
					tm.writeTaskLog(task.ID, fmt.Sprintf("Task %s has passed end time (%s), skipping scheduled execution", task.ID, endTime.Format("2006-01-02 15:04:05")))
					// 无论错过执行策略如何，都取消调度并更新任务状态为完成
					go func() {
						tm.mutex.Lock()
						defer tm.mutex.Unlock()
						tm.cancelScheduledTask(task.ID)
						// 更新任务状态为完成
						if currentTask, exists := tm.tasks[task.ID]; exists {
							currentTask.Status = TaskStatusCompleted
							currentTask.UpdatedAt = time.Now()
							completedAt := time.Now()
							currentTask.CompletedAt = &completedAt
							currentTask.EndTime = time.Now()
							// 发送完成状态
							if _, err := tm.PostStatusData(map[string]interface{}{
								"client_id": resolveAgentID(),
								"task_id":   task.ID,
								"status":    string(TaskStatusCompleted),
								"message":   "Task expired",
							}); err != nil {
								log.Printf("Failed to post status data: %v", err)
								tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to post status data: %v", err))
							}
						}
						// 关闭日志同步器
						if tm.logSyncer != nil {
							tm.logSyncer.SetTaskCompleted(task.ID, true)
						}
					}()
					return
				}
			}
		}

		// 为了避免并发执行，检查任务是否已经在运行
		tm.mutex.Lock()
		currentTask, exists := tm.tasks[task.ID]
		if !exists {
			tm.mutex.Unlock()
			return
		}

		// 检查任务是否已经在运行，如果是则跳过本次执行
		if currentTask.Status == TaskStatusRunning {
			log.Printf("Task %s is already running, skipping scheduled execution", task.ID)
			tm.writeTaskLog(task.ID, fmt.Sprintf("Task %s is already running, skipping scheduled execution", task.ID))
			// 根据错过执行策略处理
			if task.CronConfig != nil && task.CronConfig.MissedStrategy == "run_once" {
				// 如果策略是运行一次，等待当前任务完成后再运行
				tm.mutex.Unlock()
				// 等待一段时间后再次尝试 - 避免无限递归，改为重新排队
				go func() {
					time.Sleep(5 * time.Second)
					// 重新检查任务状态并执行
					tm.mutex.Lock()
					currentTask, exists := tm.tasks[task.ID]
					if !exists || currentTask.Status != TaskStatusRunning {
						tm.mutex.Unlock()
						// 任务不再运行，可以执行
						tm.mutex.Lock()
						if currentTask, exists := tm.tasks[task.ID]; exists {
							currentTask.Status = TaskStatusRunning
							currentTask.UpdatedAt = time.Now()
							currentTask.StartTime = time.Now()
						}
						tm.mutex.Unlock()
						// 发送运行状态
						if _, err := tm.PostStatusData(map[string]interface{}{
							"client_id": resolveAgentID(),
							"task_id":   task.ID,
							"status":    string(TaskStatusRunning),
							"message":   "Task started by scheduler",
						}); err != nil {
							log.Printf("Failed to post status data: %v", err)
						}
						tm.executeTask(task)
					} else {
						tm.mutex.Unlock()
						log.Printf("Task %s still running after wait, skipping scheduled execution", task.ID)
					}
				}()
				return
			}
			tm.mutex.Unlock()
			return
		}

		// 更新任务状态为运行中
		currentTask.Status = TaskStatusRunning
		currentTask.CurrentStep = 0 // 定时任务每次执行都从第一步开始
		currentTask.UpdatedAt = time.Now()
		currentTask.StartTime = time.Now()
		tm.mutex.Unlock()

		// 发送状态更新
		if _, err := tm.PostStatusData(map[string]interface{}{
			"client_id": resolveAgentID(),
			"task_id":   task.ID,
			"status":    string(TaskStatusRunning),
			"message":   "Task started by scheduler",
		}); err != nil {
			log.Printf("Failed to post status data: %v", err)
		}

		// 执行任务
		tm.executeTask(task)
	}

	// 检查任务是否已经过期（下次调度时间在结束时间之后）
	if task.CronConfig != nil && task.CronConfig.EndTime != "" {
		endTime, err := time.Parse("2006-01-02 15:04:05", task.CronConfig.EndTime)
		if err == nil {
			// 计算下次调度时间
			nextScheduledTime, err := tm.getNextScheduledTime(task.CronConfig.CronExpression)
			if err == nil {
				// 如果下次调度时间在结束时间之后，直接停止任务
				if nextScheduledTime.After(endTime) {
					log.Printf("Task %s next scheduled time (%s) is after end time (%s), stopping task",
						task.ID, nextScheduledTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))
					tm.writeTaskLog(task.ID, fmt.Sprintf("Next scheduled time (%s) is after end time (%s), stopping task",
						nextScheduledTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05")))

					// 由于可能已经持有锁，使用非阻塞方式更新状态
					// 直接修改传入的task参数，因为调用者可能持有锁
					task.Status = TaskStatusCompleted
					task.UpdatedAt = time.Now()
					completedAt := time.Now()
					task.CompletedAt = &completedAt
					task.EndTime = time.Now()

					// 发送完成状态
					if _, err := tm.PostStatusData(map[string]interface{}{
						"client_id": resolveAgentID(),
						"task_id":   task.ID,
						"status":    string(TaskStatusCompleted),
						"message":   "Task expired (next scheduled time after end time)",
					}); err != nil {
						log.Printf("Failed to post status data: %v", err)
						tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to post status data: %v", err))
					}

					// 保存任务状态 - 注意：这里不获取锁，因为调用者可能已经持有锁
					if saveErr := tm.saveTasks(); saveErr != nil {
						log.Printf("Warning: failed to save tasks: %v", saveErr)
						tm.writeTaskLog(task.ID, fmt.Sprintf("Failed to save tasks: %v", saveErr))
					}
					// 关闭日志同步器
					if tm.logSyncer != nil {
						tm.logSyncer.SetTaskCompleted(task.ID, true)
					}

					return nil
				}
			}
		}
	}

	// 添加任务到调度器
	entryID, err := tm.scheduler.AddFunc(task.CronConfig.CronExpression, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add task to scheduler: %v", err)
	}

	// 保存调度信息
	tm.scheduledJobs[task.ID] = ScheduledTaskInfo{
		EntryID: entryID,
		Loc:     time.Local,
	}

	return nil
}

// getNextScheduledTime 计算任务的下次调度时间
// 参数：cronExpr - cron表达式
// 返回：下次调度时间，如果计算失败则返回错误
func (tm *TaskManager) getNextScheduledTime(cronExpr string) (time.Time, error) {
	// 解析cron表达式
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		// 尝试使用标准解析器（没有秒字段）
		schedule, err = cron.ParseStandard(cronExpr)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid cron expression: %v", err)
		}
	}

	// 计算下次调度时间
	now := time.Now()
	next := schedule.Next(now)
	return next, nil
}

// analyzeCronExpression 分析cron表达式类型
// 返回 "fixed_rate" 表示固定频率，"fixed_time" 表示固定时间点
func (tm *TaskManager) analyzeCronExpression(cronExpr string) string {
	// 分割cron表达式字段
	fields := strings.Fields(cronExpr)
	// 标准cron表达式有5或6个字段
	if len(fields) < 5 || len(fields) > 6 {
		return "unknown"
	}

	// 对于6字段的表达式，移除秒字段
	if len(fields) == 6 {
		fields = fields[1:]
	}

	// 固定频率的cron表达式通常具有以下特点：
	// 1. 日、月、周字段都是通配符（*）
	// 2. 时字段可能是通配符或固定值，但分字段是固定的间隔
	if fields[2] == "*" && fields[3] == "*" && fields[4] == "*" {
		// 检查分字段是否为间隔表达式（如 */5）
		if strings.Contains(fields[1], "/") {
			return "fixed_rate"
		}
		// 检查时字段是否为间隔表达式
		if strings.Contains(fields[0], "/") {
			return "fixed_rate"
		}
		// 如果时、分、日、月、周都是通配符（*），也是固定频率
		if fields[0] == "*" && fields[1] == "*" {
			return "fixed_rate"
		}
	}

	// 其他情况视为固定时间点
	return "fixed_time"
}

// cancelScheduledTask 从调度器中取消指定任务
// 参数：taskID - 要取消调度的任务ID
func (tm *TaskManager) cancelScheduledTask(taskID string) {
	// 检查任务是否已被调度
	schedInfo, exists := tm.scheduledJobs[taskID]
	if !exists {
		return // 任务未被调度，无需取消
	}

	// 从调度器中移除任务
	if tm.scheduler != nil {
		tm.scheduler.Remove(schedInfo.EntryID)
	}

	// 从映射中删除任务
	delete(tm.scheduledJobs, taskID)

	log.Printf("Task %s unscheduled", taskID)
}
