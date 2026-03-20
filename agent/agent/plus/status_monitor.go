package plus

import (
	db "agent/status"
	"log"
	"time"
)

// StartStatusMonitor 启动状态监控任务
// 循环查询TaskStatus表中IsSend为false的记录，调用PostStatusData重新发送
func (tm *TaskManager) StartStatusMonitor() {
	log.Println("Starting Status Monitor...")
	go func() {
		for {
			// 检查数据库是否初始化
			if db.DB == nil {
				log.Printf("Warning: database not initialized, skipping status monitor")
				time.Sleep(30 * time.Second)
				continue
			}

			// 获取未发送的任务状态记录
			unsentStatuses, err := db.GetUnsentTaskStatus()
			if err != nil {
				log.Printf("获取未发送任务状态失败: %v", err)
				time.Sleep(30 * time.Second)
				continue
			}

			// 处理每一条未发送的状态记录
			for _, status := range unsentStatuses {
				// 构建发送数据
				payload := map[string]interface{}{
					"client_id":  resolveAgentID(),
					"task_id":    status.TaskID,
					"status":     status.Status,
					"updated_at": status.UpdatedAt,
					"message":    status.Message,
				}

				// 调用PostStatusData重新发送
				_, err := tm.PostStatusData(payload)
				if err != nil {
					log.Printf("重新发送任务状态失败 (ID: %d, TaskID: %s): %v", status.ID, status.TaskID, err)
					continue
				}

				// 发送成功，更新IsSend字段为true
				if err := db.UpdateTaskStatusIsSend(status.ID, true); err != nil {
					log.Printf("更新任务状态发送状态失败 (ID: %d): %v", status.ID, err)
				} else {
					log.Printf("成功重新发送任务状态 (ID: %d, TaskID: %s)", status.ID, status.TaskID)
				}
			}

			// 每30秒检查一次未发送的状态记录
			time.Sleep(30 * time.Second)
		}
	}()
}
