//go:build plugin

package main

import (
	"strings"
	"testing"
)

func TestIsComposeCommand(t *testing.T) {
	// 测试是否正确识别 docker compose 命令
	testCases := []struct {
		cmd      string
		expected bool
	}{
		{"docker ps", false},
		{"docker-compose up", true},
		{"docker compose up -d", true},
		{"docker images", false},
		{"docker compose down", true},
	}

	for _, tc := range testCases {
		isComposeCmd := strings.Contains(strings.ToLower(tc.cmd), "docker compose") || strings.Contains(strings.ToLower(tc.cmd), "docker-compose")
		if isComposeCmd != tc.expected {
			t.Errorf("命令 %s 期望识别为 %v，实际识别为 %v", tc.cmd, tc.expected, isComposeCmd)
		}
	}
}
func TestDockerPlugin_Execute(t *testing.T) {
	plugin := New()
	input := map[string]string{
		"command": "docker ps",
	}
	output, err := plugin.Execute(input)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	if output == nil {
		t.Errorf("Execute returned nil output")
	}
}

func TestDockerPlugin_ExecuteWithProgress(t *testing.T) {
	plugin := New()
	input := map[string]string{
		"cmd":             "docker ps",
		"taskID":          "123",
		"target_host":     "192.168.67.71",
		"target_port":     "22",
		"target_user":     "ubuntu",
		"target_password": "kk12345678",
		"workDir":         "/home/ubuntu",
	}
	output, err := plugin.ExecuteWithProgress("", input, nil)
	if err != nil {
		t.Errorf("ExecuteWithProgress failed: %v", err)
	}
	if output == nil {
		t.Errorf("ExecuteWithProgress returned nil output")
	}
}
