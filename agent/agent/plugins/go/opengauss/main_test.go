//go:build plugin

package main

import (
	"testing"
)

func TestOpengaussPlugin_ExecuteWithProgress(t *testing.T) {
	plugin := New()
	taskID := "test-task"
	input := map[string]string{
		"host":  "127.0.0.1",
		"user":  "root",
		"pass":  "Password123@pg",
		"port":  "5436",
		"db":    "sqlbot",
		"query": "select * from chat_log ORDER BY id LIMIT 10;",
	}
	out, err := plugin.ExecuteWithProgress(taskID, input, nil)

	// 检查是否返回了正确的结构，而不是具体的错误信息
	if err != nil {
		t.Errorf("ExecuteWithProgress returned error: %v", err)
	}

	// 检查是否返回了stdout字段
	if _, ok := out["stdout"]; !ok {
		t.Errorf("Expected stdout field in result")
	}

	// 检查stdout字段是否不为空
	if out["stdout"] == "" {
		t.Errorf("Expected non-empty stdout")
	}
}
