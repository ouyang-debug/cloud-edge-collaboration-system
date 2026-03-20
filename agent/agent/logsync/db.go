package logsync

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
	// _ "github.com/mattn/go-sqlite3"
)

// 数据库管理器
type DBManager struct {
	db *sql.DB
}

// 创建新的数据库管理器
func NewDBManager(dbPath string) (*DBManager, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// 创建表
	if err := createTables(db); err != nil {
		return nil, err
	}

	return &DBManager{db: db}, nil
}

// 创建数据库表
func createTables(db *sql.DB) error {
	// 创建读取记录表示
	readLogTableSQL := `CREATE TABLE IF NOT EXISTS read_logs (
		task_name TEXT PRIMARY KEY,
		read_time DATETIME NOT NULL,
		start_offset INTEGER NOT NULL,
		content_size INTEGER NOT NULL,
		content_hash TEXT NOT NULL
	);`

	// 创建变量表
	variablesTableSQL := `CREATE TABLE IF NOT EXISTS variables (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`

	// 执行创建表语句
	_, err := db.Exec(readLogTableSQL)
	if err != nil {
		return err
	}

	_, err = db.Exec(variablesTableSQL)
	if err != nil {
		return err
	}

	// 初始化网络状态变量
	_, err = db.Exec(`INSERT OR IGNORE INTO variables (key, value) VALUES (?, ?)`, "network_status", "1")
	if err != nil {
		return err
	}

	return nil
}

// 记录读取信息
func (dm *DBManager) LogReadInfo(taskName string, readTime time.Time, startOffset, contentSize int64, contentHash string) error {
	// 先尝试插入记录，如果task_name已存在则忽略
	_, err := dm.db.Exec(
		`INSERT OR IGNORE INTO read_logs (task_name, read_time, start_offset, content_size, content_hash) VALUES (?, ?, ?, ?, ?)`,
		taskName, readTime, startOffset, contentSize, contentHash,
	)
	if err != nil {
		return err
	}

	// 如果插入失败（即task_name已存在），则更新记录
	_, err = dm.db.Exec(
		`UPDATE read_logs SET read_time = ?, start_offset = ?, content_size = ?, content_hash = ? WHERE task_name = ?`,
		readTime, startOffset, contentSize, contentHash, taskName,
	)
	return err
}

// 获取上次读取记录
func (dm *DBManager) GetLastReadInfo(taskName string) (*TaskStatus, error) {
	var status TaskStatus

	row := dm.db.QueryRow(
		`SELECT task_name, read_time, start_offset, content_size, content_hash FROM read_logs 
		WHERE task_name = ? ORDER BY read_time DESC LIMIT 1`,
		taskName,
	)

	err := row.Scan(&status.TaskName, &status.LastReadTime, &status.LastReadOffset, &status.LastReadSize, &status.LastReadContentHash)
	if err == sql.ErrNoRows {
		// 没有记录，返回nil
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// 计算下次读取的偏移量
	status.LastReadOffset += status.LastReadSize

	return &status, nil
}

// 设置变量值
func (dm *DBManager) SetVariable(key, value string) error {
	_, err := dm.db.Exec(
		`INSERT OR REPLACE INTO variables (key, value) VALUES (?, ?)`,
		key, value,
	)
	return err
}

// 获取变量值
func (dm *DBManager) GetVariable(key string) (string, error) {
	var value string

	row := dm.db.QueryRow(`SELECT value FROM variables WHERE key = ?`, key)
	err := row.Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		return "", err
	}

	return value, nil
}

// 设置网络状态
func (dm *DBManager) SetNetworkStatus(status NetworkStatus) error {
	//return dm.SetVariable("network_status", string(rune(status)))
	return dm.SetVariable("network_status",
		fmt.Sprintf("%d", int(status)))
}

// 获取网络状态
func (dm *DBManager) GetNetworkStatus() (NetworkStatus, error) {
	value, err := dm.GetVariable("network_status")
	if err != nil {
		return NetworkUnavailable, err
	}

	if value == "" {
		return NetworkAvailable, nil
	}

	// 将字符串转换为int
	statusInt := 0
	_, err = fmt.Sscanf(value, "%d", &statusInt)
	if err != nil {
		return NetworkUnavailable, err
	}

	return NetworkStatus(statusInt), nil
}

// 设置任务是否完成状态
func (dm *DBManager) SetTaskCompleted(taskName string, isCompleted bool) error {
	value := "0"
	if isCompleted {
		value = "1"
	}
	return dm.SetVariable("task_completed_"+taskName, value)
}

// 获取任务是否完成状态
func (dm *DBManager) IsTaskCompleted(taskName string) (bool, error) {
	value, err := dm.GetVariable("task_completed_" + taskName)
	if err != nil {
		return false, err
	}

	return value == "1", nil
}

// 设置任务日志同步是否完成状态
func (dm *DBManager) SetLogSyncCompleted(taskName string, isCompleted bool) error {
	value := "0"
	if isCompleted {
		value = "1"
	}
	return dm.SetVariable("log_sync_completed_"+taskName, value)
}

// 获取任务日志同步是否完成状态
func (dm *DBManager) IsLogSyncCompleted(taskName string) (bool, error) {
	value, err := dm.GetVariable("log_sync_completed_" + taskName)
	if err != nil {
		return false, err
	}

	return value == "1", nil
}

// 设置任务日志同步协程是否运行状态
func (dm *DBManager) SetSyncCoroutineRunning(taskName string, isRunning bool) error {
	value := "0"
	if isRunning {
		value = "1"
	}
	return dm.SetVariable("sync_coroutine_running_"+taskName, value)
}

// 获取任务日志同步协程是否运行状态
func (dm *DBManager) IsSyncCoroutineRunning(taskName string) (bool, error) {
	value, err := dm.GetVariable("sync_coroutine_running_" + taskName)
	if err != nil {
		return false, err
	}

	return value == "1", nil
}

// 清理任务数据
// 根据任务名称，删除以下数据记录：
// 记录读取信息
// 任务是否完成状态
// 任务日志同步是否完成状态
// 任务日志同步协程是否运行状态
func (dm *DBManager) CleanupTaskData(taskName string) error {
	// 开始事务
	tx, err := dm.db.Begin()
	if err != nil {
		return err
	}

	// 定义要删除的变量键
	variableKeys := []string{
		"task_completed_" + taskName,
		"log_sync_completed_" + taskName,
		"sync_coroutine_running_" + taskName,
	}

	// 删除变量表中的任务相关记录
	for _, key := range variableKeys {
		_, err := tx.Exec(`DELETE FROM variables WHERE key = ?`, key)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// 删除读取记录表中的任务相关记录
	_, err = tx.Exec(`DELETE FROM read_logs WHERE task_name = ?`, taskName)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit()
}

// 获取所有任务名称
func (dm *DBManager) GetAllTasks() ([]string, error) {
	rows, err := dm.db.Query(`SELECT DISTINCT task_name FROM read_logs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []string
	for rows.Next() {
		var taskName string
		if err := rows.Scan(&taskName); err != nil {
			return nil, err
		}
		tasks = append(tasks, taskName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

// 关闭数据库连接
func (dm *DBManager) Close() error {
	return dm.db.Close()
}
