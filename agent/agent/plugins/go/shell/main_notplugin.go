//go:build !plugin

package main

import (
	"runtime"
	"strconv"
	"strings"

	"agent/plus"
	"agent/plus/remote"
)

type shellPlugin struct{}

func (p *shellPlugin) Name() string    { return "shell" }
func (p *shellPlugin) Version() string { return "0.1.0" }
func (p *shellPlugin) Description() string {
	return "Shell command plugin with remote execution support"
}
func (p *shellPlugin) Initialize(config string) error { return nil }
func (p *shellPlugin) Shutdown() error                { return nil }

func (p *shellPlugin) Execute(input map[string]string) (map[string]string, error) {
	host := input["target_host"]
	user := input["target_user"]
	pass := input["target_password"]
	key := input["target_key"]
	portStr := input["target_port"]
	cmdStr := input["cmd"]

	if cmdStr == "" {
		return map[string]string{"stderr": "empty cmd"}, nil
	}

	port := 22
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	if user == "" {
		user = "root"
	}

	executor := remote.NewSSHExecutor(remote.SSHConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: pass,
		Key:      key,
	})
	defer executor.Close()

	var stdout, stderr string
	var err error

	if strings.Contains(cmdStr, "\n") || strings.HasPrefix(cmdStr, "#!") {
		shell := "/bin/bash"
		if runtime.GOOS == "windows" {
			shell = "powershell"
		}

		if strings.HasPrefix(cmdStr, "#!") {
			lines := strings.SplitN(cmdStr, "\n", 2)
			shell = strings.TrimPrefix(lines[0], "#!")
			shell = strings.TrimSpace(shell)
		}
		stdout, stderr, err = executor.ExecuteStreamed(shell, cmdStr)
	} else {
		stdout, stderr, err = executor.Execute(cmdStr)
	}

	res := map[string]string{
		"stdout": stdout,
		"stderr": stderr,
	}
	return res, err
}

func (p *shellPlugin) ExecuteWithProgress(taskID string, input map[string]string, reporter plus.ProgressReporter) (map[string]string, error) {
	out, err := p.Execute(input)
	if reporter != nil {
		reporter.OnProgress(taskID, "shell", 1, 1, "")
		reporter.OnCompleted(taskID, "shell", err == nil, "")
	}
	return out, err
}

func (p *shellPlugin) OutputType() string { return "default" }

func New() plus.Plugin { return &shellPlugin{} }
