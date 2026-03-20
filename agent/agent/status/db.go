package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TaskStatus 任务状态表模型
type TaskStatus struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID    string `gorm:"index" json:"task_id"`
	TaskName  string `json:"task_name"`
	TaskType  string `json:"task_type"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
	Message   string `json:"message"`
	IsSend    bool   `json:"is_send"`
}

// TaskResultSync 任务结果同步表模型
type TaskResultSync struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID    string `gorm:"index" json:"task_id"`
	Step      string `json:"step"`
	CreateAt  string `json:"create_at"`
	IsSuccess bool   `json:"is_success"`
}

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() error {
	log.Printf("Starting database initialization...")

	// 确保data目录存在
	dataDir := "./data"
	log.Printf("Creating data directory if not exists: %s", dataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Printf("Error creating data directory: %v", err)
		return fmt.Errorf("创建data目录失败: %v", err)
	}
	log.Printf("Data directory created successfully")

	// 数据库文件路径
	dbPath := filepath.Join(dataDir, "task_status.db")
	log.Printf("Database file path: %s", dbPath)

	// 连接SQLite数据库
	log.Printf("Connecting to SQLite database...")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Printf("Error connecting to database: %v", err)
		return fmt.Errorf("连接数据库失败: %v", err)
	}
	log.Printf("Database connection established")

	// 获取底层SQLite连接并设置性能优化参数
	log.Printf("Getting underlying SQL connection...")
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error getting SQL connection: %v", err)
		// 即使获取SQL连接失败，也继续进行，使用默认配置
	} else {
		log.Printf("Got SQL connection successfully")

		// 启用WAL模式 - 允许读写并发，提升写入性能
		if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
			log.Printf("Warning: 设置WAL模式失败: %v", err)
		} else {
			log.Printf("Successfully set WAL mode")
		}
		// synchronous=NORMAL 在保证数据安全的同时提升性能
		if _, err := sqlDB.Exec("PRAGMA synchronous=NORMAL"); err != nil {
			log.Printf("Warning: 设置synchronous=NORMAL失败: %v", err)
		} else {
			log.Printf("Successfully set synchronous=NORMAL")
		}
		// 增加缓存大小到10MB
		if _, err := sqlDB.Exec("PRAGMA cache_size=10000"); err != nil {
			log.Printf("Warning: 设置cache_size失败: %v", err)
		} else {
			log.Printf("Successfully set cache_size=10000")
		}
		// 将临时表存储在内存中
		if _, err := sqlDB.Exec("PRAGMA temp_store=MEMORY"); err != nil {
			log.Printf("Warning: 设置temp_store=MEMORY失败: %v", err)
		} else {
			log.Printf("Successfully set temp_store=MEMORY")
		}
		// 启用内存模式共享缓存
		if _, err := sqlDB.Exec("PRAGMA read_uncommitted=1"); err != nil {
			log.Printf("Warning: 设置read_uncommitted=1失败: %v", err)
		} else {
			log.Printf("Successfully set read_uncommitted=1")
		}
	}

	// 自动迁移表结构
	log.Printf("Running database migrations...")
	if err := db.AutoMigrate(&TaskStatus{}, &TaskResultSync{}); err != nil {
		log.Printf("Error migrating database: %v", err)
		return fmt.Errorf("迁移表结构失败: %v", err)
	}
	log.Printf("Database migrations completed successfully")

	DB = db
	log.Printf("Database initialized successfully: %s", dbPath)
	return nil
}

// CreateTaskStatus 创建任务状态记录
func CreateTaskStatus(status *TaskStatus) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	return DB.Create(status).Error
}

// GetTaskStatusByTaskID 根据任务ID获取任务状态历史
func GetTaskStatusByTaskID(taskID string) ([]TaskStatus, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var statuses []TaskStatus
	err := DB.Where("task_id = ?", taskID).Order("updated_at desc").Find(&statuses).Error
	return statuses, err
}

// GetAllTaskStatus 获取所有任务状态记录
func GetAllTaskStatus() ([]TaskStatus, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var statuses []TaskStatus
	err := DB.Order("updated_at desc").Find(&statuses).Error
	return statuses, err
}

// UpdateTaskStatusIsSend 更新任务状态发送状态
func UpdateTaskStatusIsSend(id uint, isSend bool) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	return DB.Model(&TaskStatus{}).Where("id = ?", id).Update("is_send", isSend).Error
}

// GetUnsentTaskStatus 获取未发送的任务状态记录
func GetUnsentTaskStatus() ([]TaskStatus, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var statuses []TaskStatus
	err := DB.Where("is_send = ?", false).Find(&statuses).Error
	return statuses, err
}

// GetTaskResultSyncByTaskID 根据任务ID获取任务结果同步记录
func GetTaskResultSyncByTaskID(taskID string) ([]TaskResultSync, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var syncs []TaskResultSync
	err := DB.Where("task_id = ?", taskID).Order("id desc").Find(&syncs).Error
	return syncs, err
}

// GetAllTaskResultSyncIsNotSuccess 获取所有需要同步的结果
func GetAllTaskResultSyncIsNotSuccess() ([]TaskResultSync, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var syncs []TaskResultSync
	err := DB.Where("is_success = ?", false).Order("id desc").Find(&syncs).Error
	return syncs, err
}

// UpdateTaskResultSync 更新任务结果同步记录
func UpdateTaskResultSync(id uint, isSuccess bool) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	return DB.Model(&TaskResultSync{}).Where("id = ?", id).Update("is_success", isSuccess).Error
}

// InsertTaskResultSync 插入任务结果同步记录
func InsertTaskResultSync(sync *TaskResultSync) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	return DB.Create(sync).Error
}

// DeleteTaskLogsByTaskID 根据任务ID删除日志记录
func DeleteTaskResultSyncByTaskID(taskID string) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	return DB.Where("task_id = ?", taskID).Delete(&TaskResultSync{}).Error
}
