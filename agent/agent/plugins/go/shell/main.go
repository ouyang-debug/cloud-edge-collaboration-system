//go:build plugin

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
func (p *shellPlugin) OutputType() string {
	return "default"
}
func (p *shellPlugin) Description() string {
	return "Shell command plugin with remote execution support"
}
func (p *shellPlugin) Initialize(config string) error { return nil }
func (p *shellPlugin) Shutdown() error                { return nil }

func (p *shellPlugin) Execute(input map[string]string) (map[string]string, error) {
	host := input["targetHost"]
	user := input["targetUser"]
	pass := input["targetPassword"]
	key := input["targetKey"]
	portStr := input["targetPort"]
	cmdStr := input["cmd"]

	if cmdStr == "" {
		return map[string]string{"stderr": "empty cmd"}, nil
	}

	// If no host provided, or it's local, we can still use SSHExecutor
	// because it now handles local execution automatically.
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
	// Note: Connect/Close are handled inside Execute/ExecuteStreamed or by caller.
	// But since we are creating it here, we should ensure it's closed if a connection was made.
	defer executor.Close()

	var stdout, stderr string
	var err error

	// If it looks like a script execution (multiple lines or starts with shebang), use streamed execution
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
	// Shell plugin is single-step; we simply run it and report completion
	out, err := p.Execute(input)
	if reporter != nil {
		reporter.OnProgress(taskID, "shell", 1, 1, "")
		reporter.OnCompleted(taskID, "shell", err == nil, "")
	}
	return out, err
}

func New() plus.Plugin { return &shellPlugin{} }
