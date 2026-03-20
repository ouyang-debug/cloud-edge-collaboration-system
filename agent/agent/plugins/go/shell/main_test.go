package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestShellPlugin_Name(t *testing.T) {
	plugin := &shellPlugin{}
	name := plugin.Name()

	if name != "shell" {
		t.Errorf("Expected name 'shell', got '%s'", name)
	}
}

func TestShellShellPlugin_Version(t *testing.T) {
	plugin := &shellPlugin{}
	version := plugin.Version()

	if version != "0.1.0" {
		t.Errorf("Expected version '0.1.0', got '%s'", version)
	}
}

func TestShellPlugin_Description(t *testing.T) {
	plugin := &shellPlugin{}
	desc := plugin.Description()

	if desc == "" {
		t.Error("Description should not be empty")
	}

	expectedDesc := "Shell command plugin with remote execution support"
	if desc != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, desc)
	}
}

func TestShellPlugin_Initialize(t *testing.T) {
	plugin := &shellPlugin{}

	err := plugin.Initialize("")
	if err != nil {
		t.Errorf("Initialize should not return error, got: %v", err)
	}

	err = plugin.Initialize("some config")
	if err != nil {
		t.Errorf("Initialize should not return error with config, got: %v", err)
	}
}

func TestShellPlugin_Shutdown(t *testing.T) {
	plugin := &shellPlugin{}

	err := plugin.Shutdown()
	if err != nil {
		t.Errorf("Shutdown should be called without error, got: %v", err)
	}
}

func TestShellPlugin_Execute_EmptyCmd(t *testing.T) {
	plugin := &shellPlugin{}

	input := map[string]string{
		"cmd": "",
	}

	result, err := plugin.Execute(input)
	if err != nil {
		t.Errorf("Execute with empty cmd should not return error, got: %v", err)
	}

	if result["stderr"] != "empty cmd" {
		t.Errorf("Expected stderr 'empty cmd', got '%s'", result["stderr"])
	}
}

func TestShellPlugin_Execute_LocalCommand(t *testing.T) {
	plugin := &shellPlugin{}

	if runtime.GOOS == "windows" {
		t.Skip("Skipping local command test on Windows")
	}

	input := map[string]string{
		"cmd": "echo 'hello world'",
	}

	result, err := plugin.Execute(input)
	if err != nil {
		t.Errorf("Execute should not return error, got: %v", err)
	}

	if result["stdout"] == "" {
		t.Error("Expected stdout to contain output")
	}

	if !strings.Contains(result["stdout"], "hello world") {
		t.Errorf("Expected stdout to contain 'hello world', got '%s'", result["stdout"])
	}
}

func TestShellPlugin_Execute_WithPort(t *testing.T) {
	plugin := &shellPlugin{}

	input := map[string]string{
		"cmd":         "echo test",
		"target_port": "2222",
	}

	result, err := plugin.Execute(input)
	if err != nil {
		t.Errorf("Execute should not return error, got: %v", err)
	}

	if result == nil {
		t.Error("Expected result to not be nil")
	}
}

func TestShellPlugin_Execute_WithUser(t *testing.T) {
	plugin := &shellPlugin{}

	input := map[string]string{
		"cmd":         "echo test",
		"target_user": "testuser",
	}

	result, err := plugin.Execute(input)
	if err != nil {
		t.Errorf("Execute should not return error, got: %v", err)
	}

	if result == nil {
		t.Error("Expected result to not be nil")
	}
}

func TestShellPlugin_ExecuteWithProgress(t *testing.T) {
	plugin := &shellPlugin{}

	input := map[string]string{
		"cmd": "echo test",
	}

	result, err := plugin.ExecuteWithProgress("task-123", input, nil)
	if err != nil {
		t.Errorf("ExecuteWithProgress should not return error, got: %v", err)
	}

	if result == nil {
		t.Error("Expected result to not be nil")
	}
}

func TestNew(t *testing.T) {
	plugin := New()

	if plugin == nil {
		t.Error("New should return a non-nil plugin")
	}

	shellPlugin, ok := plugin.(*shellPlugin)
	if !ok {
		t.Error("New should return a *shellPlugin type")
	}

	if shellPlugin.Name() != "shell" {
		t.Errorf("Expected plugin name 'shell', got '%s'", shellPlugin.Name())
	}
}

func TestShellPlugin_Execute_ScriptDetection(t *testing.T) {
	plugin := &shellPlugin{}

	// tests_del := []struct {
	// 	name     string
	// 	cmd      string
	// 	isScript bool
	// }{
	// 	{
	// 		name:     "simple command",
	// 		cmd:      "echo hello",
	// 		isScript: false,
	// 	},
	// 	{
	// 		name:     "multiline command",
	// 		cmd:      "echo hello\necho world",
	// 		isScript: true,
	// 	},
	// 	{
	// 		name:     "bash script with shebang",
	// 		cmd:      "#!/bin/bash\necho hello",
	// 		isScript: true,
	// 	},
	// 	// {
	// 	// 	name:     "python script with shebang",
	// 	// 	cmd:      "#!/usr/bin/env python\nprint('hello')",
	// 	// 	isScript: true,
	// 	// },

	// }

	// host := input["target_host"]
	// user := input["target_user"]
	// pass := input["target_password"]
	// key := input["target_key"]
	// portStr := input["target_port"]
	// cmdStr := input["cmd"]
	tests := []struct {
		target_host     string
		target_user     string
		target_password string
		target_key      string
		target_port     string
		name            string
		cmd             string
		isScript        bool
	}{
		{
			target_host:     "10.220.42.151",
			target_user:     "root",
			target_password: "Tpri@hn2025xxxx",
			target_key:      "",
			target_port:     "8090",
			name:            "simple command",
			cmd:             "ls -l /home",
			isScript:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string]string{
				"cmd":            tt.cmd,
				"targetHost":     tt.target_host,
				"targetUser":     tt.target_user,
				"targetPassword": tt.target_password,
				"targetKey":      tt.target_key,
				"targetPort":     tt.target_port,
			}

			result, err := plugin.Execute(input)
			if err != nil {
				t.Errorf("Execute should not return error, got: %v", err)
			}

			if result == nil {
				t.Error("Expected result to not be nil")
			}
			//打印result中的stdout
			t.Logf("stdout: %s", result["stdout"])
			t.Logf("stderr: %s", result["stderr"])
		})
	}
}

func BenchmarkShellPlugin_Execute(b *testing.B) {
	plugin := &shellPlugin{}
	input := map[string]string{
		"cmd": "echo benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin.Execute(input)
	}
}
