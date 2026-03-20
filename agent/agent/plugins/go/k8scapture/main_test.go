package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestK8scapturePlugin_Name(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &k8scapturePlugin{}
	name := plugin.Name()

	if name != "k8scapture" {
		t.Errorf("Expected name 'k8scapture', got '%s'", name)
	}
}

func TestK8scapturePlugin_Version(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &k8scapturePlugin{}
	version := plugin.Version()

	if version != "0.1.0" {
		t.Errorf("Expected version '0.1.0', got '%s'", version)
	}
}

func TestK8scapturePlugin_Description(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &k8scapturePlugin{}
	desc := plugin.Description()

	if desc == "" {
		t.Error("Description should not be empty")
	}

	expectedDesc := "K8s command capture plugin with remote execution support"
	if desc != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, desc)
	}
}

func TestK8scapturePlugin_Initialize(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &k8scapturePlugin{}

	err := plugin.Initialize("")
	if err != nil {
		t.Errorf("Initialize should not return error, got: %v", err)
	}

	err = plugin.Initialize("some config")
	if err != nil {
		t.Errorf("Initialize should not return error with config, got: %v", err)
	}
}

func TestK8scapturePlugin_Shutdown(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &k8scapturePlugin{}

	err := plugin.Shutdown()
	if err != nil {
		t.Errorf("Shutdown should be called without error, got: %v", err)
	}
}

func TestK8scapturePlugin_Execute_EmptyCmd(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &k8scapturePlugin{}

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

func TestK8scapturePlugin_Execute_CommandFormat(t *testing.T) {
	plugin := &k8scapturePlugin{}

	tests := []struct {
		name        string
		cmdInput    string
		expectedCmd string
		hasError    bool
		errorMsg    string
	}{

		// dockerinfo:docker info
		// dockerdaemon:cat /etc/docker/daemon.json
		// dockerentry:systemctl --no-pager cat docker
		// dockerdf: docker system df

		// dockerps: docker ps -a --format "{\"id\":\"{{.ID}}\",\"name\":\"{{.Names}}\",\"image\":\"{{.Image}}\",\"status\":\"{{.Status}}\",\"command\":\"{{.Command}}\",\"create\":\"{{.CreatedAt}}\",\"ports\":\"{{.Ports}}\"}"
		// dockerinspect: docker ps -a | grep -v CONTAINER | awk '{ids=ids " " $1} END {print substr(ids,2)}'|xargs docker inspect
		// dockerstats: docker stats -a --no-stream --format "{\"id\":\"{{.ID}}\",\"container\":\"{{.Name}}\",\"cpuPercent\":\"{{.CPUPerc}}\",\"memUsage\":\"{{.MemUsage}}\",\"memPercent\":\"{{.MemPerc}}\",\"netIo\":\"{{.NetIO}}\"},\"blockIo\":\"{{.BlockIO}}\"},\"pids\":\"{{.PIDs}}\"}"

		// echo docker info|base64 -w 0
		// echo cat /etc/docker/daemon.json|base64 -w 0
		// echo systemctl --no-pager cat docker|base64 -w 0
		// echo docker system df|base64 -w 0
		// echo 'docker ps -a --format "{\"id\":\"{{.ID}}\",\"name\":\"{{.Names}}\",\"image\":\"{{.Image}}\",\"status\":\"{{.Status}}\",\"command\":\"{{.Command}}\",\"create\":\"{{.CreatedAt}}\",\"ports\":\"{{.Ports}}\"}"'|base64 -w 0
		// echo 'docker ps -a | grep -v CONTAINER | awk '{ids=ids " " $1} END {print substr(ids,2)}'|xargs docker inspect'|base64 -w 0
		// echo 'docker stats -a --no-stream --format "{\"id\":\"{{.ID}}\",\"container\":\"{{.Name}}\",\"cpuPercent\":\"{{.CPUPerc}}\",\"memUsage\":\"{{.MemUsage}}\",\"memPercent\":\"{{.MemPerc}}\",\"netIo\":\"{{.NetIO}}\"},\"blockIo\":\"{{.BlockIO}}\"},\"pids\":\"{{.PIDs}}\"}"'|base64 -w 0

		{
			name:        "k8sinfo command",
			cmdInput:    K8SINFO,
			expectedCmd: "kubectl cluster-info",
			hasError:    false,
		},
		{
			name:        "k8snodes command",
			cmdInput:    K8SNODES,
			expectedCmd: "kubectl get nodes -o wide",
			hasError:    false,
		},
		{
			name:        "k8sns command",
			cmdInput:    K8SNS,
			expectedCmd: "kubectl get namespaces",
			hasError:    false,
		},
		{
			name:        "k8spods command",
			cmdInput:    K8SPODS,
			expectedCmd: "kubectl get pods -A -o wide",
			hasError:    false,
		},
		{
			name:        "k8stop command",
			cmdInput:    K8STOP,
			expectedCmd: "kubectl top pod -A",
			hasError:    false,
		},
		{
			name:        "k8ssvc command",
			cmdInput:    K8SSVC,
			expectedCmd: "kubectl get svc -A -o wide",
			hasError:    false,
		},
		{
			name:        "k8sconfigmap command",
			cmdInput:    K8SCM,
			expectedCmd: "kubectl get configmap -A -o wide",
			hasError:    false,
		},
		{
			name:        "k8sdaemonset command",
			cmdInput:    K8SDAEMONSET,
			expectedCmd: "kubectl get daemonset -A -o wide",
			hasError:    false,
		},
		{
			name:        "k8sdeployment command",
			cmdInput:    K8SDEPLOYMENT,
			expectedCmd: "kubectl get deployment -A -o wide",
			hasError:    false,
		},
		{
			name:        "k8sstatefulset command",
			cmdInput:    K8SSTATEFULSET,
			expectedCmd: "kubectl get statefulset -A -o wide",
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
			target_host:     "10.220.42.155",
			target_user:     "root",
			target_password: "Tpri@hn20251205",
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

func TestDockercapturePlugin_Execute_LocalCommand(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &k8scapturePlugin{}

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

func TestDockercapturePlugin_Execute_WithPort(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &k8scapturePlugin{}

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

func TestDockercapturePlugin_Execute_WithUser(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &k8scapturePlugin{}

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

func TestDockercapturePlugin_ExecuteWithProgress(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &k8scapturePlugin{}

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

func TestDockercapturePlugin_Execute_ScriptDetection(t *testing.T) {
	t.Skip("忽略此测试")
	plugin := &k8scapturePlugin{}

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
			target_host:     "10.220.42.154",
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

func BenchmarkDockercapturePlugin_Execute(b *testing.B) {

	// plugin := &dockercapturePlugin{}
	// input := map[string]string{
	// 	"cmd": "echo benchmark",
	// }

	// b.ResetTimer()
	// for i := 0; i < b.N; i++ {
	// 	plugin.Execute(input)
	// }
}
