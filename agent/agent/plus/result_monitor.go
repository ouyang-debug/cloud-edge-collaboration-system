package plus

import (
	db "agent/status"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// StartResultSyncMonitor 启动任务结果同步监控
// 循环查询TaskResultSync表中IsSuccess为false的记录，传输任务结果
func (tm *TaskManager) StartResultSyncMonitor() {
	log.Println("Starting Result Sync Monitor...")
	go func() {
		for {
			// 检查数据库是否初始化
			if db.DB == nil {
				log.Printf("Warning: database not initialized, skipping task result sync")
				time.Sleep(30 * time.Second)
				continue
			}

			// 获取未同步的任务结果记录
			shouldSyncs, err := db.GetAllTaskResultSyncIsNotSuccess()
			if err != nil {
				log.Printf("获取未同步任务结果记录失败: %v", err)
				time.Sleep(30 * time.Second)
				continue
			}

			// 处理每一条未同步的任务结果记录
			for _, resaultSynce := range shouldSyncs {
				// 构建任务文件目录路径
				taskDir := filepath.Join("tasks", resaultSynce.TaskID, "out", fmt.Sprintf("seq%s", resaultSynce.Step))
				// 遍历目录下的所有文件
				err := filepath.Walk(taskDir, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					// 只处理文件，跳过目录
					if !info.IsDir() {
						// 读取文件内容
						fileContent, err := os.ReadFile(path)
						if err != nil {
							log.Printf("读取文件失败 (Path: %s): %v", path, err)
							return nil
						}

						// 构建发送数据
						payload := map[string]interface{}{
							"client_id":    resolveAgentID(),
							"task_id":      resaultSynce.TaskID,
							"step":         resaultSynce.Step,
							"file_content": string(fileContent),
							"create_at":    resaultSynce.CreateAt,
						}

						// 转换为JSON字符串
						payloadStr, err := json.Marshal(payload)
						if err != nil {
							log.Printf("转换任务结果为JSON失败 (TaskID: %s, File: %s): %v", resaultSynce.TaskID, path, err)
							return nil
						}

						// 传输结果
						_, err = tm.PostResultSync(string(payloadStr))
						if err != nil {
							log.Printf("传输任务结果失败 (TaskID: %s, File: %s): %v", resaultSynce.TaskID, path, err)
							return nil
						}

						log.Printf("成功传输任务结果文件 (TaskID: %s, File: %s)", resaultSynce.TaskID, path)
					}

					return nil
				})

				if err != nil {
					log.Printf("遍历任务文件目录失败 (TaskID: %s, Dir: %s): %v", resaultSynce.TaskID, taskDir, err)
					continue
				}

				// 传输完成，重置IsSuccess状态为true
				if err := db.UpdateTaskResultSync(resaultSynce.ID, true); err != nil {
					log.Printf("更新任务结果同步状态失败 (ID: %d): %v", resaultSynce.ID, err)
				} else {
					log.Printf("成功完成任务结果同步 (ID: %d, TaskID: %s)", resaultSynce.ID, resaultSynce.TaskID)
				}
			}

			// 每30秒检查一次未同步的任务结果
			time.Sleep(30 * time.Second)
		}
	}()
}
