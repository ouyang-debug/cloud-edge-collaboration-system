package plus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc"

	"agent/proto"
)

// PlusVersion represents the current version of plus
const PlusVersion = "0.1.0"

// Plus represents the Agent Plus component
type Plus struct {
	taskManager   *TaskManager
	pluginManager *PluginManager
	grpcServer    *grpc.Server
	proto.UnimplementedAgentServiceServer
}

// NewPlus creates a new Plus instance
func NewPlus() *Plus {
	pm := NewPluginManager()
	return &Plus{
		taskManager:   NewTaskManager(pm),
		pluginManager: pm,
	}
}

// Start initializes and starts the Plus component
func (p *Plus) Start() error {

	log.Println("Starting Agent Plus...")

	// Start task manager
	if err := p.taskManager.Start(); err != nil {
		return fmt.Errorf("failed to start task manager: %v", err)
	}

	// Start plugin manager
	if err := p.pluginManager.Start(); err != nil {
		return fmt.Errorf("failed to start plugin manager: %v", err)
	}

	// Start gRPC server
	if err := p.startGRPCServer(); err != nil {
		return fmt.Errorf("failed to start gRPC server: %v", err)
	}

	log.Println("Agent Plus started successfully")
	return nil
}

// Stop stops the Plus component
func (p *Plus) Stop() error {
	log.Println("Stopping Agent Plus...")

	// Stop gRPC server
	if p.grpcServer != nil {
		p.grpcServer.GracefulStop()
		log.Println("gRPC server stopped")
	}

	// Stop task manager
	if err := p.taskManager.Stop(); err != nil {
		log.Printf("Warning: failed to stop task manager: %v", err)
	}

	// Stop plugin manager
	if err := p.pluginManager.Stop(); err != nil {
		log.Printf("Warning: failed to stop plugin manager: %v", err)
	}

	log.Println("Agent Plus stopped")
	return nil
}

// handleCommandMessage handles command messages
func (p *Plus) handleCommandMessage(topic string, payload []byte) error {
	// Handle task dispatch
	// 判断s是否有后缀字符串suffix
	if strings.HasSuffix(topic, "/task/dispatch") {
		return p.taskManager.HandleTaskDispatch(payload)
	}

	// Parse command type from topic
	// 返回去除s可能的前缀prefix的字符串
	// Todo需要重新解析
	topicSplit := strings.SplitAfter(topic, "/commands/")
	commandType := topicSplit[len(topicSplit)-1]
	log.Printf("Received command: %s, payload: %s", commandType, string(payload))

	switch commandType {
	case "task/start":
		// Handle task start command
		id := strings.TrimSpace(string(payload))
		id = strings.Trim(id, "\"'`")

		return p.taskManager.StartTaskByMQTT(id)
	case "task/stop":
		// Handle task stop command
		id := strings.TrimSpace(string(payload))
		id = strings.Trim(id, "\"'`")

		return p.taskManager.StopTask(id)
	// case "task/update":
	// 	if err := p.handleServerTaskUpdate(payload); err != nil {
	// 		return p.taskManager.UpdateTask(string(payload))
	// 	}
	// 	return nil
	default:
		log.Printf("Unknown command type: %s", commandType)
		return fmt.Errorf("unknown command type: %s", commandType)
	}
}

// startGRPCServer starts the gRPC server listening on port 2345
func (p *Plus) startGRPCServer() error {
	// Create gRPC server
	server := grpc.NewServer()

	// Register AgentService
	proto.RegisterAgentServiceServer(server, p)

	// Listen on port 12345
	listener, err := net.Listen("tcp", ":12345")
	if err != nil {
		return fmt.Errorf("failed to listen on port 12345: %v", err)
	}

	p.grpcServer = server

	// Start server in a goroutine
	go func() {
		log.Printf("gRPC server started, listening on port 12345")
		if err := server.Serve(listener); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	return nil
}

// StartPlus implements the AgentService interface
func (p *Plus) StartPlus(ctx context.Context, req *proto.StartPlusRequest) (*proto.StartPlusResponse, error) {
	// Plus is already running since this is being called on a running Plus instance
	return &proto.StartPlusResponse{
		Success: true,
		Message: "Plus is already running",
	}, nil
}

// StopPlus implements the AgentService interface
func (p *Plus) StopPlus(ctx context.Context, req *proto.StopPlusRequest) (*proto.StopPlusResponse, error) {
	if err := p.Stop(); err != nil {
		return &proto.StopPlusResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to stop Plus: %v", err),
		}, nil
	}
	return &proto.StopPlusResponse{
		Success: true,
		Message: "Plus stopped successfully",
	}, nil
}

// GetPlusStatus implements the AgentService interface
func (p *Plus) GetPlusStatus(ctx context.Context, req *proto.GetPlusStatusRequest) (*proto.GetPlusStatusResponse, error) {
	// Plus is running since this service is responding
	status := int32(1)
	version := PlusVersion

	// 获取升级信息
	upgrade := ""
	if p.taskManager.upgradeState != nil && p.taskManager.upgradeState.CanUpgrade {
		upgrade = p.taskManager.upgradeState.TargetVersion
	}

	// 获取插件列表
	pluginsJSON := ""
	if plugins, err := p.pluginManager.ListPlugins(); err == nil {
		type PluginJSON struct {
			Pluginname string `json:"pluginname"`
			Version    string `json:"version"`
		}
		var pluginList []PluginJSON
		for _, plugin := range plugins {
			pluginList = append(pluginList, PluginJSON{
				Pluginname: plugin.Name,
				Version:    "v0.0.1",
			})
		}
		if jsonData, err := json.Marshal(pluginList); err == nil {
			pluginsJSON = string(jsonData)
		}
	}

	// 获取运行中和等待中的任务（用于升级检查）
	type TaskInfoJSON struct {
		TaskID   string `json:"taskId"`
		TaskName string `json:"taskName"`
		Status   string `json:"status"`
		TaskType string `json:"taskType"`
	}
	var activeTaskList []TaskInfoJSON

	if tasks, err := p.taskManager.ListTasks(); err == nil {
		for _, task := range tasks {
			if task.Status == TaskStatusRunning || task.Status == TaskStatusWaiting {
				activeTaskList = append(activeTaskList, TaskInfoJSON{
					TaskID:   task.ID,
					TaskName: task.Name,
					Status:   string(task.Status),
					TaskType: task.TaskType,
				})
			}
		}
	}

	runningTasksJSON := ""
	if jsonData, err := json.Marshal(activeTaskList); err == nil {
		runningTasksJSON = string(jsonData)
	}

	// 获取失败的任务（最近10条）
	failedTasksJSON := ""
	if tasks, err := p.taskManager.ListTasks(); err == nil {
		type FailedTaskJSON struct {
			Task     string `json:"task"`
			Failtime string `json:"failtime"`
		}
		var failedTaskList []FailedTaskJSON
		for _, task := range tasks {
			if task.Status == TaskStatusFailed {
				failtime := ""
				if !task.EndTime.IsZero() {
					failtime = fmt.Sprintf("%d", task.EndTime.Unix())
				}
				failedTaskList = append(failedTaskList, FailedTaskJSON{
					Task:     task.ID,
					Failtime: failtime,
				})
			}
		}
		// 只保留最近10条
		if len(failedTaskList) > 10 {
			failedTaskList = failedTaskList[:10]
		}
		if jsonData, err := json.Marshal(failedTaskList); err == nil {
			failedTasksJSON = string(jsonData)
		}
	}

	return &proto.GetPlusStatusResponse{
		Status:      status,
		Version:     version,
		Upgrade:     upgrade,
		Plugins:     pluginsJSON,
		Runningtask: runningTasksJSON,
		Failedtask:  failedTasksJSON,
	}, nil
}

// ForwardMQTTMessage implements the AgentService interface
func (p *Plus) ForwardMQTTMessage(ctx context.Context, req *proto.ForwardMQTTMessageRequest) (*proto.ForwardMQTTMessageResponse, error) {
	log.Printf("Received MQTT message via gRPC: topic=%s, payload=%s", req.Topic, string(req.Payload))

	// Handle the MQTT message (similar to handleMQTTMessage)
	if err := p.handleCommandMessage(req.Topic, req.Payload); err != nil {
		return &proto.ForwardMQTTMessageResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to handle message: %v", err),
		}, nil
	}

	return &proto.ForwardMQTTMessageResponse{
		Success: true,
		Message: "Message handled successfully",
	}, nil
}

// CreateTask implements the AgentService interface
func (p *Plus) CreateTask(ctx context.Context, req *proto.CreateTaskRequest) (*proto.CreateTaskResponse, error) {
	// Determine task type based on whether schedule is provided
	taskType := "once" // default to one-time task
	if req.Schedule != "" {
		taskType = "cycle" // if schedule is provided, it's a cycle task
	}

	// Create a task JSON string
	taskJSON := fmt.Sprintf(`{
		"id": "%s",
		"name": "%s",
		"description": "%s",
		"command": "%s",
		"schedule": "%s",
		"taskType": "%s"
	}`, req.Id, req.Name, req.Description, req.Command, req.Schedule, taskType)

	if err := p.taskManager.UpdateTask(taskJSON); err != nil {
		return &proto.CreateTaskResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create task: %v", err),
		}, nil
	}

	return &proto.CreateTaskResponse{
		Success: true,
		Message: "Task created successfully",
		TaskId:  req.Id,
	}, nil
}

// StartTask implements the AgentService interface
func (p *Plus) StartTask(ctx context.Context, req *proto.StartTaskRequest) (*proto.StartTaskResponse, error) {
	if err := p.taskManager.StartTask(req.TaskId); err != nil {
		return &proto.StartTaskResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start task: %v", err),
		}, nil
	}

	return &proto.StartTaskResponse{
		Success: true,
		Message: "Task started successfully",
	}, nil
}

// StopTask implements the AgentService interface
func (p *Plus) StopTask(ctx context.Context, req *proto.StopTaskRequest) (*proto.StopTaskResponse, error) {
	if err := p.taskManager.StopTask(req.TaskId); err != nil {
		return &proto.StopTaskResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to stop task: %v", err),
		}, nil
	}

	return &proto.StopTaskResponse{
		Success: true,
		Message: "Task stopped successfully",
	}, nil
}

// GetTaskStatus implements the AgentService interface
func (p *Plus) GetTaskStatus(ctx context.Context, req *proto.GetTaskStatusRequest) (*proto.GetTaskStatusResponse, error) {
	status, err := p.taskManager.GetTaskStatus(req.TaskId)
	if err != nil {
		return &proto.GetTaskStatusResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get task status: %v", err),
		}, nil
	}

	// Parse the status string (assuming it's in JSON format)
	// This is a simplified implementation - in a real scenario, we would properly parse the JSON
	return &proto.GetTaskStatusResponse{
		Success:   true,
		Message:   "Task status retrieved successfully",
		Status:    string(status),
		Pid:       "0", // Placeholder
		StartTime: time.Now().Unix(),
		EndTime:   0,
	}, nil
}

// ListTasks implements the AgentService interface
func (p *Plus) ListTasks(ctx context.Context, req *proto.ListTasksRequest) (*proto.ListTasksResponse, error) {
	tasks, err := p.taskManager.ListTasks()
	if err != nil {
		return &proto.ListTasksResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to list tasks: %v", err),
		}, nil
	}

	// Convert to TaskInfo objects
	var taskInfos []*proto.TaskInfo
	for _, task := range tasks {
		schedule := ""
		if task.CronConfig != nil {
			schedule = task.CronConfig.CronExpression
		}

		taskInfos = append(taskInfos, &proto.TaskInfo{
			Id:          task.ID,
			Name:        task.Name,
			Description: task.Description,
			Status:      string(task.Status),
			Command:     "", // 命令现在存储在每个步骤中
			Schedule:    schedule,
			Pid:         fmt.Sprintf("%d", task.PID),
			StartTime:   task.CreatedAt.Unix(),
			EndTime:     0,
		})
	}

	return &proto.ListTasksResponse{
		Success: true,
		Message: "Tasks listed successfully",
		Tasks:   taskInfos,
	}, nil
}

// LoadPlugin implements the AgentService interface
func (p *Plus) LoadPlugin(ctx context.Context, req *proto.LoadPluginRequest) (*proto.LoadPluginResponse, error) {
	pluginID, err := p.pluginManager.LoadPlugin(req.PluginPath)
	if err != nil {
		return &proto.LoadPluginResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to load plugin: %v", err),
		}, nil
	}

	return &proto.LoadPluginResponse{
		Success:  true,
		Message:  "Plugin loaded successfully",
		PluginId: pluginID,
	}, nil
}

// UnloadPlugin implements the AgentService interface
func (p *Plus) UnloadPlugin(ctx context.Context, req *proto.UnloadPluginRequest) (*proto.UnloadPluginResponse, error) {
	if err := p.pluginManager.UnloadPlugin(req.PluginId, ""); err != nil {
		return &proto.UnloadPluginResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to unload plugin: %v", err),
		}, nil
	}

	return &proto.UnloadPluginResponse{
		Success: true,
		Message: "Plugin unloaded successfully",
	}, nil
}

// ExecutePlugin implements the AgentService interface
func (p *Plus) ExecutePlugin(ctx context.Context, req *proto.ExecutePluginRequest) (*proto.ExecutePluginResponse, error) {
	taskID := req.Parameters["taskId"]
	reporter := &logProgressReporter{}
	result, err := p.pluginManager.ExecutePluginWithProgress(req.PluginId, "", taskID, req.Parameters, reporter)
	if err != nil {
		return &proto.ExecutePluginResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to execute plugin: %v", err),
		}, nil
	}

	return &proto.ExecutePluginResponse{
		Success: true,
		Message: "Plugin executed successfully",
		Result:  fmt.Sprintf("%v", result),
	}, nil
}

// ListPlugins implements the AgentService interface
func (p *Plus) ListPlugins(ctx context.Context, req *proto.ListPluginsRequest) (*proto.ListPluginsResponse, error) {
	plugins, err := p.pluginManager.ListPlugins()
	if err != nil {
		return &proto.ListPluginsResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to list plugins: %v", err),
		}, nil
	}

	// Convert to PluginInfo objects
	var pluginInfos []*proto.PluginInfo
	for _, plugin := range plugins {
		pluginInfos = append(pluginInfos, &proto.PluginInfo{
			Id:          plugin.ID,
			Name:        plugin.Name,
			Path:        plugin.Path,
			Status:      plugin.Status,
			Description: plugin.Description,
		})
	}

	return &proto.ListPluginsResponse{
		Success: true,
		Message: "Plugins listed successfully",
		Plugins: pluginInfos,
	}, nil
}


// MessageNotice implements the AgentService interface
func (p *Plus) MessageNotice(ctx context.Context, req *proto.MessageNoticeRequest) (*proto.MessageNoticeResponse, error) {
	return &proto.MessageNoticeResponse{
		Type:    1,
		Message: "net:ok",
	}, nil
}

type logProgressReporter struct{}

func (r *logProgressReporter) OnProgress(taskID, pluginName string, current, total int, message string) {
	log.Printf("plugin progress: task=%s plugin=%s %d/%d %s", taskID, pluginName, current, total, message)
}

func (r *logProgressReporter) OnCompleted(taskID, pluginName string, success bool, message string) {
	log.Printf("plugin completed: task=%s plugin=%s success=%v %s", taskID, pluginName, success, message)
}

func (r *logProgressReporter) OnError(taskID, pluginName string, err error) {
	log.Printf("plugin error: task=%s plugin=%s err=%v", taskID, pluginName, err)
}
