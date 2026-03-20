package remote

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPConfig struct {
	Host     string
	User     string
	Password string
	KeyFile  string
}

type SFTPClient struct {
	config SFTPConfig
	client *sftp.Client
}

func NewSFTPClient(config SFTPConfig) *SFTPClient {
	return &SFTPClient{config: config}
}

func (c *SFTPClient) Connect() (*sftp.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User:            c.config.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if c.config.KeyFile != "" {
		keyBytes, err := os.ReadFile(c.config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("读取私钥文件失败：%v", err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败：%v", err)
		}
		sshConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else if c.config.Password != "" {
		sshConfig.Auth = []ssh.AuthMethod{ssh.Password(c.config.Password)}
	} else {
		return nil, fmt.Errorf("未配置密码或私钥认证信息")
	}

	sshConn, err := ssh.Dial("tcp", c.config.Host, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败：%v", err)
	}

	sftpClient, err := sftp.NewClient(sshConn)
	if err != nil {
		sshConn.Close()
		return nil, fmt.Errorf("创建SFTP客户端失败：%v", err)
	}

	c.client = sftpClient
	return sftpClient, nil
}

func (c *SFTPClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *SFTPClient) UploadFile(localPath, remotePath string) error {
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败：%v", err)
	}
	defer localFile.Close()

	fileInfo, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败：%v", err)
	}
	if fileInfo.IsDir() {
		return fmt.Errorf("不能上传目录，请指定文件")
	}

	remoteFile, err := c.client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败：%v", err)
	}
	defer remoteFile.Close()

	_, err = io.Copy(remoteFile, localFile)
	if err != nil {
		return fmt.Errorf("上传文件失败：%v", err)
	}

	return nil
}

func (c *SFTPClient) DownloadFile(remotePath, localPath string) error {
	remoteFile, err := c.client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("打开远程文件失败：%v", err)
	}
	defer remoteFile.Close()

	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败：%v", err)
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, remoteFile)
	if err != nil {
		return fmt.Errorf("下载文件失败：%v", err)
	}

	return nil
}

func (c *SFTPClient) ListFiles(remoteDir string) ([]os.FileInfo, error) {
	entries, err := c.client.ReadDir(remoteDir)
	if err != nil {
		return nil, fmt.Errorf("列出目录文件失败：%v", err)
	}
	return entries, nil
}

func (c *SFTPClient) CreateDirectory(remoteDir string) error {
	err := c.client.MkdirAll(remoteDir)
	if err != nil {
		return fmt.Errorf("创建远程目录失败：%v", err)
	}
	return nil
}

func (c *SFTPClient) DeleteFile(remotePath string) error {
	err := c.client.Remove(remotePath)
	if err != nil {
		return fmt.Errorf("删除远程文件失败：%v", err)
	}
	return nil
}
