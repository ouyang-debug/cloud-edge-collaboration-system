package remote

import (
	"log"
	"testing"
)

func TestSFTPClient(t *testing.T) {
	config := SFTPConfig{
		Host:     "192.168.67.85:22",
		User:     "root",
		Password: "",
	}

	client := NewSFTPClient(config)
	_, err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	err = client.CreateDirectory("/uploads")
	if err != nil {
		t.Logf("创建目录失败: %v", err)
	}

	err = client.UploadFile("local.txt", "/uploads/test.txt")
	if err != nil {
		t.Logf("上传文件失败: %v", err)
	}

	err = client.DownloadFile("/uploads/test.txt", "downloaded.txt")
	if err != nil {
		t.Logf("下载文件失败: %v", err)
	}
}
