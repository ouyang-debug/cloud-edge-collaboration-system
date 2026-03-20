//go:build plugin

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"agent/plus"

	"gopkg.in/yaml.v3"
)

type input struct {
	TaskId     string `json:"taskId"`
	ConfigPath string `json:"configPath"`
	Kubeconfig string `json:"kubeconfig"`
	Namespace  string `json:"namespace"`
}

type yamlExecute struct {
	Commands    []string `yaml:"commands"`
	ApplyFiles  []string `yaml:"applyFiles"`
	DeleteFiles []string `yaml:"deleteFiles"`
}

type yamlTask struct {
	Type       string      `yaml:"type"`
	TaskId     string      `yaml:"taskId"`
	Kubeconfig string      `yaml:"kubeconfig"`
	Namespace  string      `yaml:"namespace"`
	Execute    yamlExecute `yaml:"execute"`
}

type cmdResult struct {
	Index           int               `json:"index"`
	Success         bool              `json:"success"`
	ExecutionTimeMs int64             `json:"executionTimeMs"`
	Stdout          string            `json:"stdout,omitempty"`
	Stderr          string            `json:"stderr,omitempty"`
	Error           map[string]string `json:"error,omitempty"`
}

func runK8sTask(in input, reporter plus.ProgressReporter) (map[string]interface{}, error) {
	var yt yamlTask
	if in.ConfigPath != "" {
		data, err := ioutil.ReadFile(in.ConfigPath)
		if err == nil {
			_ = yaml.Unmarshal(data, &yt)
		}
	}
	kube := pick(yt.Kubeconfig, in.Kubeconfig)
	ns := pick(yt.Namespace, in.Namespace)
	var cmds []string
	if len(yt.Execute.Commands) > 0 {
		cmds = yt.Execute.Commands
	} else {
		for _, f := range yt.Execute.ApplyFiles {
			cmd := "kubectl apply -f " + quote(f)
			if strings.TrimSpace(ns) != "" {
				cmd += " -n " + quote(ns)
			}
			cmds = append(cmds, cmd)
		}
		for _, f := range yt.Execute.DeleteFiles {
			cmd := "kubectl delete -f " + quote(f)
			if strings.TrimSpace(ns) != "" {
				cmd += " -n " + quote(ns)
			}
			cmds = append(cmds, cmd)
		}
	}
	var results []cmdResult
	success := true
	total := len(cmds)
	for i, c := range cmds {
		start := time.Now()
		stdout, stderr, err := runKubectl(c, kube)
		r := cmdResult{
			Index:           i,
			ExecutionTimeMs: time.Since(start).Milliseconds(),
			Stdout:          stdout,
			Stderr:          stderr,
			Success:         err == nil,
		}
		if err != nil {
			r.Error = map[string]string{"message": err.Error()}
			success = false
		}
		results = append(results, r)
		if reporter != nil && total > 0 {
			reporter.OnProgress(pick(yt.TaskId, in.TaskId), "k8s", i+1, total, "")
		}
	}
	out := map[string]interface{}{
		"taskId":     pick(yt.TaskId, in.TaskId),
		"success":    success,
		"statements": results,
	}
	return out, nil
}

func runKubectl(c string, kubeconfig string) (string, string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", c)
	} else {
		cmd = exec.Command("bash", "-c", c)
	}
	if strings.TrimSpace(kubeconfig) != "" {
		cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func quote(s string) string {
	if runtime.GOOS == "windows" {
		return `"` + s + `"`
	}
	return "'" + s + "'"
}

func pick(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

type k8sPlugin struct{}

func (p *k8sPlugin) Name() string                   { return "k8s" }
func (p *k8sPlugin) Version() string                { return "0.1.0" }
func (p *k8sPlugin) OutputType() string             { return "default" }
func (p *k8sPlugin) Description() string            { return "Kubernetes plugin" }
func (p *k8sPlugin) Initialize(config string) error { return nil }
func (p *k8sPlugin) Shutdown() error                { return nil }

func (p *k8sPlugin) Execute(input map[string]string) (map[string]string, error) {
	in := inputFrom(input)
	out, err := runK8sTask(in, nil)
	if err != nil && out == nil {
		return nil, err
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	result := map[string]string{"stdout": string(data)}
	return result, nil
}

func (p *k8sPlugin) ExecuteWithProgress(taskID string, input map[string]string, reporter plus.ProgressReporter) (map[string]string, error) {
	in := inputFrom(input)
	if in.TaskId == "" {
		in.TaskId = taskID
	}
	out, err := runK8sTask(in, reporter)
	if err != nil && out == nil {
		return nil, err
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	result := map[string]string{"stdout": string(data)}
	return result, nil
}

func inputFrom(m map[string]string) input {
	return input{
		TaskId:     m["taskId"],
		ConfigPath: m["configPath"],
		Kubeconfig: m["kubeconfig"],
		Namespace:  m["namespace"],
	}
}

func New() plus.Plugin { return &k8sPlugin{} }
