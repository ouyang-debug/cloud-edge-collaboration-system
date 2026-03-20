//go:build plugin

package main

import (
	"testing"
)

func TestOpengausscapturePlugin_ExecuteWithProgress_AllCommands(t *testing.T) {
	plugin := &opengausscapturePlugin{}

	// 测试一次性采集所有指标
	input := map[string]string{
		"host": "localhost",
		"user": "root",
		"pass": "Password123@pg",
		"port": "5436",
		"cmd":  "all", // 使用 "all" 命令一次性采集所有指标
	}

	result, err := plugin.ExecuteWithProgress("task-all", input, nil)
	if err != nil {
		t.Logf("Expected error due to no OpenGauss server, got: %v", err)
	}

	if result == nil {
		t.Error("Expected result to not be nil")
	}

	if result["stdout"] == "" {
		t.Error("Expected stdout to not be empty")
	}
}

func TestOpengausscapturePlugin_ExecuteWithProgress_NoCmd(t *testing.T) {
	plugin := &opengausscapturePlugin{}

	// 测试没有传入 cmd 参数时，默认使用 "all" 命令
	input := map[string]string{
		"host": "localhost",
		"user": "postgres",
		"pass": "password",
		"port": "5432",
		// 没有传入 cmd 参数
	}

	result, err := plugin.ExecuteWithProgress("task-no-cmd", input, nil)
	if err != nil {
		t.Logf("Expected error due to no OpenGauss server, got: %v", err)
	}

	if result == nil {
		t.Error("Expected result to not be nil")
	}

	if result["stdout"] == "" {
		t.Error("Expected stdout to not be empty")
	}
}
