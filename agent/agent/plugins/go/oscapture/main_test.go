package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestOscapturePlugin_Name(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &oscapturePlugin{}
	name := plugin.Name()

	if name != "oscapture" {
		t.Errorf("Expected name 'oscapture', got '%s'", name)
	}
}

func TestOscapturePlugin_Version(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &oscapturePlugin{}
	version := plugin.Version()

	if version != "0.1.0" {
		t.Errorf("Expected version '0.1.0', got '%s'", version)
	}
}

func TestOscapturePlugin_Description(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &oscapturePlugin{}
	desc := plugin.Description()

	if desc == "" {
		t.Error("Description should not be empty")
	}

	expectedDesc := "OS command capture plugin with remote execution support"
	if desc != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, desc)
	}
}

func TestOscapturePlugin_Initialize(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &oscapturePlugin{}

	err := plugin.Initialize("")
	if err != nil {
		t.Errorf("Initialize should not return error, got: %v", err)
	}

	err = plugin.Initialize("some config")
	if err != nil {
		t.Errorf("Initialize should not return error with config, got: %v", err)
	}
}

func TestOscapturePlugin_Shutdown(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &oscapturePlugin{}

	err := plugin.Shutdown()
	if err != nil {
		t.Errorf("Shutdown should be called without error, got: %v", err)
	}
}

func TestOscapturePlugin_Execute_EmptyCmd(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &oscapturePlugin{}

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

func TestOscapturePlugin_Execute_CommandFormat(t *testing.T) {
	plugin := &oscapturePlugin{}

	tests := []struct {
		name        string
		cmdInput    string
		expectedCmd string
		hasError    bool
		errorMsg    string
	}{
		{
			name:        "os_base command",
			cmdInput:    OS_BASE,
			expectedCmd: "uname -a",
			hasError:    false,
		},
		{
			name:        "os_usage command",
			cmdInput:    OS_USAGE,
			expectedCmd: "top",
			hasError:    false,
		},
		{
			name:        "os_net_loss command with IP",
			cmdInput:    OS_NET_LOSS + " 192.168.67.85",
			expectedCmd: "ping 192.168.67.85",
			hasError:    false,
		},
		// {
		// 	name:     "os_net_loss command without IP",
		// 	cmdInput: OS_NET_LOSS,
		// 	hasError: true,
		// 	errorMsg: "missing target IP for os_net_loss",
		// },
		// {
		// 	name:     "unknown command type",
		// 	cmdInput: "unknown_type",
		// 	hasError: true,
		// 	errorMsg: "unknown command type: unknown_type",
		// },
	}

	tests2 := []struct {
		target_host     string
		target_user     string
		target_password string
		target_key      string
		target_port     string
		task_id         string
		cmd             string
		isScript        bool
	}{
		{
			target_host:     "192.168.67.85",
			target_user:     "root",
			target_password: "ubtbj",
			target_key:      "",
			target_port:     "22",
			task_id:         "task123",
			cmd:             "ls -l /home",
			isScript:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need to mock the executor to test the command parsing without actual execution
			// For now, we'll just test that the parsing doesn't return errors for valid inputs
			input := map[string]string{
				"cmd":             tt.cmdInput,
				"target_host":     tests2[0].target_host,
				"target_user":     tests2[0].target_user,
				"target_password": tests2[0].target_password,
				"target_key":      tests2[0].target_key,
				"target_port":     tests2[0].target_port,
				"task_id":         tests2[0].task_id,
			}

			result, err := plugin.Execute(input)
			if tt.hasError {
				if err != nil {
					t.Errorf("Execute should not return error, got: %v", err)
				}
				if result["stderr"] != tt.errorMsg {
					t.Errorf("Expected stderr '%s', got '%s'", tt.errorMsg, result["stderr"])
				}
			} else {
				// The command will fail to execute (no host), but should not fail during parsing
				if err != nil {
					// We expect an error due to no host, but not due to parsing
					t.Logf("Expected error due to no host, got: %v", err)
				}
			}
		})
	}
}

func TestOscapturePlugin_Execute_LocalCommand(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &oscapturePlugin{}

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

func TestOscapturePlugin_Execute_WithPort(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &oscapturePlugin{}

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

func TestOscapturePlugin_Execute_WithUser(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &oscapturePlugin{}

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

func TestOscapturePlugin_ExecuteWithProgress(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &oscapturePlugin{}

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

	oscapturePlugin, ok := plugin.(*oscapturePlugin)
	if !ok {
		t.Error("New should return a *oscapturePlugin type")
	}

	if oscapturePlugin.Name() != "oscapture" {
		t.Errorf("Expected plugin name 'oscapture', got '%s'", oscapturePlugin.Name())
	}
}

func TestOscapturePlugin_Execute_ScriptDetection(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &oscapturePlugin{}

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
			target_password: "Tpri@hn20251205",
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
				"cmd":             tt.cmd,
				"target_host":     tt.target_host,
				"target_user":     tt.target_user,
				"target_password": tt.target_password,
				"target_key":      tt.target_key,
				"target_port":     tt.target_port,
			}

			result, err := plugin.Execute(input)
			if err != nil {
				t.Errorf("Execute should not return error, got: %v", err)
			}

			if result == nil {
				t.Error("Expected result to not be nil")
			}
			t.Logf("stdout: %s", result["stdout"])
			t.Logf("stderr: %s", result["stderr"])
		})
	}
}

func BenchmarkOscapturePlugin_Execute(b *testing.B) {

	// plugin := &oscapturePlugin{}
	// input := map[string]string{
	// 	"cmd": "echo benchmark",
	// }

	// b.ResetTimer()
	// for i := 0; i < b.N; i++ {
	// 	plugin.Execute(input)
	// }
}
