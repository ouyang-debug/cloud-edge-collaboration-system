//go:build plugin

package main

import (
	"testing"
)

func TestMysqlcapturePlugin_ExecuteWithProgress(t *testing.T) {
	plugin := &mysqlcapturePlugin{}

	input := map[string]string{
		"host": "localhost",
		"user": "root",
		"port": "3366",
		"pass": "123123",
	}

	result, err := plugin.ExecuteWithProgress("task-123", input, nil)
	if err != nil {
		// 预期会有错误，因为没有实际的MySQL服务器
		t.Logf("Expected error due to no MySQL server, got: %v", err)
	}

	// 检查结果是否不为nil
	if result == nil {
		t.Error("Expected result to not be nil")
	}
}
