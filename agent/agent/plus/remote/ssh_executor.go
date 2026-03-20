package remote

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// IsLocalHost checks if the given host is the local machine
func IsLocalHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				if ipnet.IP.String() == host {
					return true
				}
			}
		}
	}
	return false
}

// SSHConfig represents the configuration for SSH connection
type SSHConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Key      string // Private key content
	Timeout  time.Duration
}

// SSHExecutor handles remote command execution via SSH
type SSHExecutor struct {
	config SSHConfig
	client *ssh.Client
}

// NewSSHExecutor creates a new SSHExecutor
func NewSSHExecutor(config SSHConfig) *SSHExecutor {
	if config.Port <= 0 {
		config.Port = 22
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	return &SSHExecutor{config: config}
}

// Connect establishes the SSH connection
func (e *SSHExecutor) Connect() error {
	var auth []ssh.AuthMethod

	if e.config.Key != "" {
		signer, err := ssh.ParsePrivateKey([]byte(e.config.Key))
		if err != nil {
			return fmt.Errorf("failed to parse private key: %v", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}

	if e.config.Password != "" {
		auth = append(auth, ssh.Password(e.config.Password))
	}

	sshConfig := &ssh.ClientConfig{
		User:            e.config.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use a more secure callback
		Timeout:         e.config.Timeout,
	}

	addr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to dial ssh: %v", err)
	}

	e.client = client
	return nil
}

// Close closes the SSH connection
func (e *SSHExecutor) Close() error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}

// Execute runs a command locally or remotely based on the host
func (e *SSHExecutor) Execute(cmd string) (stdout, stderr string, err error) {
	if IsLocalHost(e.config.Host) {
		var outBuf, errBuf bytes.Buffer
		var command *exec.Cmd
		if runtime.GOOS == "windows" {
			command = exec.Command("cmd", "/C", cmd)
		} else {
			command = exec.Command("sh", "-c", cmd)
		}
		command.Stdout = &outBuf
		command.Stderr = &errBuf
		err = command.Run()
		return outBuf.String(), errBuf.String(), err
	}

	if e.client == nil {
		if err := e.Connect(); err != nil {
			return "", "", err
		}
	}

	session, err := e.client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var outBuf, errBuf bytes.Buffer
	session.Stdout = &outBuf
	session.Stderr = &errBuf

	err = session.Run(cmd)
	return outBuf.String(), errBuf.String(), err
}

// ExecuteStreamed runs a script by piping its content to the shell's stdin (local or remote)
func (e *SSHExecutor) ExecuteStreamed(shell string, scriptContent string) (stdout, stderr string, err error) {
	if IsLocalHost(e.config.Host) {
		var outBuf, errBuf bytes.Buffer
		var command *exec.Cmd
		if shell == "" {
			if runtime.GOOS == "windows" {
				shell = "cmd"
			} else {
				shell = "sh"
			}
		}

		if runtime.GOOS == "windows" && (shell == "cmd" || shell == "cmd.exe") {
			command = exec.Command("cmd", "/C", scriptContent)
		} else {
			// For sh/bash/powershell, we use -s to read from stdin if supported,
			// but for local execution it's often easier to just pass the script content
			// or use a temporary file. However, to stay "traceless" we'll use stdin.
			command = exec.Command(shell, "-s")
			stdin, err := command.StdinPipe()
			if err != nil {
				return "", "", fmt.Errorf("failed to create local stdin pipe: %v", err)
			}
			go func() {
				defer stdin.Close()
				io.WriteString(stdin, scriptContent)
			}()
		}

		command.Stdout = &outBuf
		command.Stderr = &errBuf
		err = command.Run()
		return outBuf.String(), errBuf.String(), err
	}

	if e.client == nil {
		if err := e.Connect(); err != nil {
			return "", "", err
		}
	}

	session, err := e.client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return "", "", fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	var outBuf, errBuf bytes.Buffer
	session.Stdout = &outBuf
	session.Stderr = &errBuf

	// Start the shell with -s to read from stdin
	err = session.Start(fmt.Sprintf("%s -s", shell))
	if err != nil {
		return "", "", fmt.Errorf("failed to start remote shell: %v", err)
	}

	// Write script content to stdin and close it
	_, err = io.WriteString(stdin, scriptContent)
	if err != nil {
		return "", "", fmt.Errorf("failed to write script to stdin: %v", err)
	}
	stdin.Close()

	// Wait for execution to finish
	err = session.Wait()
	return outBuf.String(), errBuf.String(), err
}
