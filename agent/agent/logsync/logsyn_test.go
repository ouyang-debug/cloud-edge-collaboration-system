package logsync

import (
	"testing"
	"time"
)

/*
// 测试LogSyncer的创建
func TestNewLogSyncer(t *testing.T) {
	config := Config{
		ReadInterval: 100 * time.Millisecond,
		ReadSize:     100,
		ServerURL:    "http://localhost:8080/logs",
		DBPath:       "./test.db", // 使用本地数据库进行测试
	}

	syncer, err := NewLogSyncer(config)
	if err != nil {
		t.Fatalf("Failed to create LogSyncer: %v", err)
	}

	if syncer == nil {
		t.Fatal("LogSyncer is nil")
	}

	defer syncer.Stop()
}

// 测试网络状态设置和获取
func TestNetworkStatus(t *testing.T) {
	config := Config{
		ReadInterval: 100 * time.Millisecond,
		ReadSize:     100,
		ServerURL:    "http://localhost:8080/logs",
		DBPath:       "./test.db",
	}

	syncer, err := NewLogSyncer(config)
	if err != nil {
		t.Fatalf("Failed to create LogSyncer: %v", err)
	}
	defer syncer.Stop()

	// 测试设置网络状态
	syncer.SetNetworkStatus(NetworkUnavailable)
	if syncer.GetNetworkStatus() != NetworkUnavailable {
		t.Error("Failed to set network status to unavailable")
	}

	syncer.SetNetworkStatus(NetworkAvailable)
	if syncer.GetNetworkStatus() != NetworkAvailable {
		t.Error("Failed to set network status to available")
	}
}

// 测试任务状态设置
func TestTaskStatus(t *testing.T) {
	config := Config{
		ReadInterval: 100 * time.Millisecond,
		ReadSize:     100,
		ServerURL:    "http://localhost:8080/logs",
		DBPath:       "./test.db",
	}

	syncer, err := NewLogSyncer(config)
	if err != nil {
		t.Fatalf("Failed to create LogSyncer: %v", err)
	}
	defer syncer.Stop()

	// 测试初始化任务状态
	taskName := "test_task"
	syncer.InitStatus(taskName)

	// 测试启动任务同步
	err = syncer.StartTaskSync(taskName)
	if err != nil {
		t.Fatalf("Failed to start task sync: %v", err)
	}

	// 测试停止任务同步
	syncer.StopTaskSync(taskName)
}

// 测试文件读取功能
func TestFileReading(t *testing.T) {
	// 创建测试目录和文件
	testDir := "test_task"
	// 如果目录已存在，先删除
	os.RemoveAll(testDir)
	err := os.Mkdir(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// 创建测试日志文件
	testFilePath := filepath.Join(testDir, "app_runtime.log")
	testContent := "This is a test log content for file reading test."
	err = os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	// 测试文件读取
	content, readSize, err := readLogFile(testFilePath, 0, 50)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if readSize != int64(len(content)) {
		t.Errorf("Read size mismatch: expected %d, got %d", len(content), readSize)
	}

	// 测试文件回滚检测
	isRolledBack, err := isFileRolledBack(testFilePath, int64(len(testContent)+1))
	if err != nil {
		t.Fatalf("Failed to check file rollback: %v", err)
	}

	if !isRolledBack {
		t.Error("File rollback not detected")
	}
}

// 测试Hash计算
func TestCalculateHash(t *testing.T) {
	testContent := []byte("test content")
	hash := calculateHash(testContent)
	if hash == "" {
		t.Error("Failed to calculate hash")
	}

	// 测试相同内容的Hash是否一致
	anotherHash := calculateHash(testContent)
	if hash != anotherHash {
		t.Error("Hash mismatch for same content")
	}

	// 测试不同内容的Hash是否不同
	differentContent := []byte("different content")
	differentHash := calculateHash(differentContent)
	if hash == differentHash {
		t.Error("Hash should be different for different content")
	}
}

// 测试数据库操作
func TestDBOperations(t *testing.T) {
	// 创建内存数据库
	dbManager, err := NewDBManager("./test.db")
	if err != nil {
		t.Fatalf("Failed to create DBManager: %v", err)
	}
	defer dbManager.Close()

	// 测试记录读取信息
	taskName := "test_task"
	readTime := time.Now()
	startOffset := int64(0)
	contentSize := int64(100)
	contentHash := "test_hash"

	err = dbManager.LogReadInfo(taskName, readTime, startOffset, contentSize, contentHash)
	if err != nil {
		t.Fatalf("Failed to log read info: %v", err)
	}

	// 测试获取上次读取记录
	lastReadStatus, err := dbManager.GetLastReadInfo(taskName)
	if err != nil {
		t.Fatalf("Failed to get last read info: %v", err)
	}

	if lastReadStatus == nil {
		t.Fatal("Last read status is nil")
	}

	if lastReadStatus.TaskName != taskName {
		t.Errorf("Task name mismatch: expected %s, got %s", taskName, lastReadStatus.TaskName)
	}

	if lastReadStatus.LastReadOffset != startOffset+contentSize {
		t.Errorf("Last read offset mismatch: expected %d, got %d", startOffset+contentSize, lastReadStatus.LastReadOffset)
	}

	// 测试变量设置和获取
	err = dbManager.SetVariable("test_key", "test_value")
	if err != nil {
		t.Fatalf("Failed to set variable: %v", err)
	}

	value, err := dbManager.GetVariable("test_key")
	if err != nil {
		t.Fatalf("Failed to get variable: %v", err)
	}

	if value != "test_value" {
		t.Errorf("Variable value mismatch: expected %s, got %s", "test_value", value)
	}
}

// 测试HTTP客户端
func TestHTTPClient(t *testing.T) {
	// 这里只测试HTTP客户端的创建，不测试实际发送功能（需要真实服务器）
	serverURL := "http://localhost:8080/logs"
	httpClient := NewHTTPClient(serverURL)

	if httpClient == nil {
		t.Fatal("HTTPClient is nil")
	}
}
*/

// 完整测试
func TestLogSyncer(t *testing.T) {
	// 测试完整的日志同步流程
	config := Config{
		ReadInterval: 100 * time.Millisecond,
		ReadSize:     1000,
		ServerURL:    "http://192.168.245.100:8081/logs",
		DBPath:       "./synclog.db",
	}
	syncer, err := NewLogSyncer(config)
	taskName1 := "test_task1"
	taskName2 := "test_task2"

	if err != nil {
		t.Fatalf("Failed to create LogSyncer: %v", err)
	}
	defer syncer.Stop()
	//用于控制网络状态，不可用将结束所有任务协程
	syncer.SetNetworkStatus(NetworkAvailable)
	//用于控制任务状态，结束任务同步协程
	//syncer.db.SetTaskCompleted(taskName1, true)

	// 测试初始化任务状态
	syncer.InitStatus(taskName1)
	syncer.InitStatus(taskName2)
	//go
	func() {
		// 测试启动任务同步
		// 测试停止任务同步
		for {
			//对syncer.tasks进行遍历，停止所有任务同步
			for taskName := range syncer.tasks {
				running, err := syncer.db.IsSyncCoroutineRunning(taskName)
				if err != nil {
					//t.Fatalf("Failed to check sync coroutine status: %v", err)
				}
				if syncer.GetNetworkStatus() == NetworkUnavailable {
					t.Error("Task sync should be running")

					if running {
						syncer.StopTaskSync(taskName)
					}
				} else {
					if !running {
						err = syncer.StartTaskSync(taskName)
						if err != nil {
							//t.Fatalf("Failed to start task sync: %v", err)
						}
					}
				}

				isCompleted, err := syncer.db.IsLogSyncCompleted(taskName)
				if err != nil {
					//t.Fatalf("Failed to check log sync completed: %v", err)
				}
				if isCompleted {
					err = syncer.db.CleanupTaskData(taskName)
					if err != nil {
						//t.Fatalf("Failed to cleanup task data: %v", err)
					}
				}
			}
			time.Sleep(10 * time.Second)
		}
	}()
	// time.Sleep(300 * time.Second)
	// if syncer.GetNetworkStatus() == NetworkUnavailable {
	// 	time.Sleep(10 * time.Second)
	// 	err = syncer.db.CleanupTaskData(taskName)
	// 	if err != nil {
	// 		t.Fatalf("Failed to cleanup task data: %v", err)
	// 	}
	// 	err = syncer.db.CleanupTaskData(taskName2)
	// 	if err != nil {
	// 		t.Fatalf("Failed to cleanup task data: %v", err)
	// 	}
	// }

}
