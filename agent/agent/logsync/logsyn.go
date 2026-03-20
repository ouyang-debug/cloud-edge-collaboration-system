package logsync

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// 配置参数
type Config struct {
	ReadInterval time.Duration // 读取间隔
	ReadSize     int64         // 每次读取大小
	ServerURL    string        // REST服务器URL
	DBPath       string        // SQLite数据库路径
	AgentId      string        // 代理ID
}

// 任务同步状态
type TaskStatus struct {
	TaskName            string    // 任务名称
	LastReadTime        time.Time // 上次读取时间
	LastReadOffset      int64     // 上次读取位置
	LastReadSize        int64     // 上次读取大小
	LastReadContentHash string    // 上次读取内容Hash
	IsRunning           bool      // 同步协程是否运行
	IsTaskCompleted     bool      // 任务是否完成
	IsLogSyncCompleted  bool      // 日志同步是否完成
}

// 网络状态
type NetworkStatus int

const (
	NetworkUnavailable NetworkStatus = iota // 网络不可达
	NetworkAvailable                        // 网络可用
)

// 日志同步器
type LogSyncer struct {
	config           Config
	networkStatus    NetworkStatus
	tasks            map[string]*TaskStatus
	db               *DBManager
	wg               sync.WaitGroup
	mutex            sync.Mutex
	stopChan         chan struct{}
	taskSyncChannels map[string]chan struct{}
}

// 创建新的日志同步器
func NewLogSyncer(config Config) (*LogSyncer, error) {
	db, err := NewDBManager(config.DBPath)
	if err != nil {
		return nil, err
	}

	return &LogSyncer{
		config:           config,
		networkStatus:    NetworkAvailable,
		tasks:            make(map[string]*TaskStatus),
		db:               db,
		stopChan:         make(chan struct{}),
		taskSyncChannels: make(map[string]chan struct{}),
	}, nil
}

// 启动日志同步器
func (ls *LogSyncer) Start() error {
	// 初始化网络状态
	ls.networkStatus = NetworkAvailable

	// 加载所有任务状态并启动同步
	tasks, err := ls.db.GetAllTasks()
	if err != nil {
		log.Printf("Failed to get all tasks: %v", err)
	} else {
		for _, taskName := range tasks {
			// 检查任务是否已经完成
			isCompleted, err := ls.db.IsTaskCompleted(taskName)
			if err != nil {
				log.Printf("Failed to check if task %s is completed: %v", taskName, err)
				continue
			}
			// 如果任务已经完成，跳过
			if isCompleted {
				continue
			}
			// 初始化任务状态
			ls.InitStatus(taskName)
			// 启动任务同步
			if err := ls.StartTaskSync(taskName); err != nil {
				log.Printf("Failed to start log sync for task %s: %v", taskName, err)
			}
		}
	}

	log.Println("Log syncer started")
	return nil
}

// 停止日志同步器
func (ls *LogSyncer) Stop() {
	close(ls.stopChan)
	ls.wg.Wait()
}

// 设置网络状态
func (ls *LogSyncer) SetNetworkStatus(status NetworkStatus) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	ls.db.SetNetworkStatus(status)
	ls.networkStatus = status
}

// 获取网络状态
func (ls *LogSyncer) GetNetworkStatus() NetworkStatus {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	ls.networkStatus, _ = ls.db.GetNetworkStatus()
	return ls.networkStatus
}

// 标记任务完成状态
func (ls *LogSyncer) SetTaskCompleted(taskName string, completed bool) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	if task, exists := ls.tasks[taskName]; exists {
		task.IsTaskCompleted = completed
	}
	// 更新数据库状态
	ls.db.SetTaskCompleted(taskName, completed)
}

// 初始化任务状态
func (ls *LogSyncer) InitStatus(taskName string) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	if _, exists := ls.tasks[taskName]; !exists {
		ls.tasks[taskName] = &TaskStatus{
			TaskName:           taskName,
			IsTaskCompleted:    false,
			IsLogSyncCompleted: false,
			IsRunning:          false,
		}
	}
	// 初始化任务完成状态到数据库
	ls.db.SetTaskCompleted(taskName, false)
	// 初始化任务日志同步协程运行状态到数据库
	ls.db.SetSyncCoroutineRunning(taskName, false)
	// 初始化任务日志完成状态到数据库
	ls.db.SetLogSyncCompleted(taskName, false)
	// 检查数据库中是否已有读取记录，只有在没有记录时才初始化
	lastReadStatus, err := ls.db.GetLastReadInfo(taskName)
	if err != nil || lastReadStatus == nil {
		// 初始化任务日志读取记录到数据库
		readTime := time.Now()
		if err := ls.db.LogReadInfo(taskName, readTime, 0, 0, "0"); err != nil {
		}
	}

}

// 启动任务同步协程
func (ls *LogSyncer) StartTaskSync(taskName string) error {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()

	// 检查网络状态
	if ls.networkStatus != NetworkAvailable {
		return nil
	}

	// 检查任务状态
	task, exists := ls.tasks[taskName]
	if !exists {
		task = &TaskStatus{
			TaskName:           taskName,
			IsTaskCompleted:    func() bool { ok, _ := ls.db.IsTaskCompleted(taskName); return ok }(),
			IsLogSyncCompleted: func() bool { ok, _ := ls.db.IsLogSyncCompleted(taskName); return ok }(),
			IsRunning:          func() bool { ok, _ := ls.db.IsSyncCoroutineRunning(taskName); return ok }(),
		}
		ls.tasks[taskName] = task
	}

	// 检查是否可以启动
	if task.IsRunning || task.IsLogSyncCompleted {
		return nil
	}

	// 启动同步协程
	task.IsRunning = true
	//增加数据库同步
	ls.db.SetSyncCoroutineRunning(taskName, task.IsRunning)

	stopChan := make(chan struct{})
	ls.taskSyncChannels[taskName] = stopChan
	ls.wg.Add(1)

	go func() {
		defer ls.wg.Done()
		defer func() {
			ls.mutex.Lock()
			task.IsRunning = false
			ls.db.SetSyncCoroutineRunning(taskName, task.IsRunning)
			delete(ls.taskSyncChannels, taskName)
			ls.mutex.Unlock()
		}()

		ls.syncTaskLogs(taskName, stopChan)
	}()

	return nil
}

// 停止任务同步协程
func (ls *LogSyncer) StopTaskSync(taskName string) {
	ls.mutex.Lock()
	stopChan, exists := ls.taskSyncChannels[taskName]
	ls.mutex.Unlock()

	if exists {
		close(stopChan)
	}
}

// 同步任务日志
func (ls *LogSyncer) syncTaskLogs(taskName string, stopChan <-chan struct{}) {
	// 获取上次读取记录
	lastReadStatus, err := ls.db.GetLastReadInfo(taskName)
	if err != nil {
		return
	}

	// 初始化读取偏移量
	currentOffset := int64(0)
	if lastReadStatus != nil {
		currentOffset = lastReadStatus.LastReadOffset
	}

	// 创建HTTP客户端
	httpClient := NewHTTPClient(ls.config.ServerURL)

	// 定时读取日志文件
	ticker := time.NewTicker(ls.config.ReadInterval)
	defer ticker.Stop()

	logFilePath := getLogFilePath(taskName)

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			// 检查网络状态
			ls.mutex.Lock()
			networkStatus := ls.networkStatus
			ls.mutex.Unlock()

			if networkStatus != NetworkAvailable {
				return
			}

			// 检查文件是否存在
			if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
				continue
			}

			// 检查文件是否回滚
			isRolledBack, err := isFileRolledBack(logFilePath, currentOffset)
			if err != nil {
				continue
			}

			// 如果文件回滚，重新从开头读取
			if isRolledBack {
				currentOffset = 0
			}

			// 读取日志文件
			content, readSize, err := readLogFile(logFilePath, currentOffset, ls.config.ReadSize)
			if err != nil {
				continue
			}

			// 如果没有读取到数据，检查是否完成同步
			if readSize == 0 {
				// 检查任务是否完成
				ls.mutex.Lock()
				task := ls.tasks[taskName]
				//isTaskCompleted := task.IsTaskCompleted
				isTaskCompleted := func() bool { ok, _ := ls.db.IsTaskCompleted(taskName); return ok }()
				ls.mutex.Unlock()

				if isTaskCompleted {
					// 标记日志同步完成
					ls.mutex.Lock()
					task.IsLogSyncCompleted = true
					ls.mutex.Unlock()

					// 更新数据库状态
					ls.db.SetLogSyncCompleted(taskName, true)

					// 删除日志文件
					os.Remove(logFilePath)

					return
				}
				continue
			}

			// 计算内容Hash
			contentHash := calculateHash(content)

			// 发送日志数据
			readTime := time.Now()
			req := LogDataRequest{
				TaskId:      taskName,
				AgentId:     ls.config.AgentId,
				LogLevel:    "INFO",
				LogType:     "SYSTEM",
				Content:     content,
				ReadTime:    readTime,
				Offset:      currentOffset,
				Size:        readSize,
				ContentHash: contentHash,
			}

			if err := httpClient.SendLogData(req); err != nil {
				continue
			}

			// 记录读取信息到数据库
			if err := ls.db.LogReadInfo(taskName, readTime, currentOffset, readSize, contentHash); err != nil {
				continue
			}

			// 更新当前偏移量
			currentOffset += readSize
		}
	}
}

// 计算内容Hash
func calculateHash(content []byte) string {
	hash := md5.Sum(content)
	return hex.EncodeToString(hash[:])
}

// 检查文件是否回滚
func isFileRolledBack(filePath string, lastOffset int64) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}

	// 如果文件大小小于上次读取偏移量，说明文件回滚
	return fileInfo.Size() < lastOffset, nil
}

// 读取日志文件
func readLogFile(filePath string, offset int64, readSize int64) ([]byte, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	// 设置读取位置
	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, 0, err
	}

	// 读取数据
	buffer := make([]byte, readSize)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, 0, err
	}

	return buffer[:n], int64(n), nil
}

// 获取日志文件路径
func getLogFilePath(taskName string) string {
	return filepath.Join("tasks", taskName, "log", "task.log")
}
