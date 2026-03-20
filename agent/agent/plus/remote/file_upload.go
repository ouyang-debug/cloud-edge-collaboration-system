package remote

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

const (
	DEFAULT_CHUNK_SIZE  = 1 * 1024 * 1024
	UPLOAD_CHECK_API    = "/api/upload/check"
	UPLOAD_CONTINUE_API = "/api/upload/continue"
)

type FileUploaderConfig struct {
	BaseURL        string // 服务端地址，如 "http://127.0.0.1:8080"
	FilePath       string // 本地文件路径，如 "./test_large_file.zip"
	RemoteFileName string // 远端文件名，用于远端文件查询与存储，这里可以指定任务+步骤，如 "task1_step1@local_file"
	ChunkSize      int64  // 分片大小（字节），默认 1MB
}

type FileUploader struct {
	config FileUploaderConfig
	client *http.Client
}

func NewFileUploader(config FileUploaderConfig) *FileUploader {
	if config.ChunkSize <= 0 {
		config.ChunkSize = DEFAULT_CHUNK_SIZE
	}
	return &FileUploader{
		config: config,
		client: &http.Client{},
	}
}

func (u *FileUploader) getFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (u *FileUploader) checkUploadedSize(fileMD5, fileName string) (int64, error) {
	url := fmt.Sprintf("%s%s", u.config.BaseURL, UPLOAD_CHECK_API)

	formData := strings.NewReader(fmt.Sprintf("file_md5=%s&file_name=%s", fileMD5, fileName))
	req, err := http.NewRequest("POST", url, formData)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := u.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("查询失败，状态码：%d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("解析响应失败：%v", err)
	}

	uploadedSize := int64(0)
	if val, ok := result["uploaded_size"]; ok {
		if floatVal, ok := val.(float64); ok {
			uploadedSize = int64(floatVal)
		}
	}
	return uploadedSize, nil
}

func (u *FileUploader) uploadChunk(fileMD5, fileName string, startPos int64, chunk []byte) (int64, error) {
	url := fmt.Sprintf("%s%s", u.config.BaseURL, UPLOAD_CONTINUE_API)

	body := &strings.Builder{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("file_md5", fileMD5)
	_ = writer.WriteField("file_name", fileName)
	_ = writer.WriteField("start_pos", fmt.Sprintf("%d", startPos))

	filePart, err := writer.CreateFormFile("file_chunk", fileName)
	if err != nil {
		return 0, err
	}
	_, err = filePart.Write(chunk)
	if err != nil {
		return 0, err
	}

	_ = writer.Close()

	req, err := http.NewRequest("POST", url, strings.NewReader(body.String()))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := u.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("分片上传失败，状态码：%d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("解析响应失败：%v", err)
	}

	currentSize := int64(0)
	if val, ok := result["current_size"]; ok {
		if floatVal, ok := val.(float64); ok {
			currentSize = int64(floatVal)
		}
	}
	return currentSize, nil
}

func (u *FileUploader) Upload() error {
	if _, err := os.Stat(u.config.FilePath); os.IsNotExist(err) {
		return errors.New("文件不存在")
	}

	fileInfo, err := os.Stat(u.config.FilePath)
	if err != nil {
		return err
	}
	totalSize := fileInfo.Size()

	remoteFileName := u.config.RemoteFileName
	if remoteFileName == "" {
		remoteFileName = fileInfo.Name()
	}

	fileMD5, err := u.getFileMD5(u.config.FilePath)
	if err != nil {
		return fmt.Errorf("计算MD5失败：%v", err)
	}

	uploadedSize, err := u.checkUploadedSize(fileMD5, remoteFileName)
	if err != nil {
		return fmt.Errorf("查询已上传大小失败：%v", err)
	}
	fmt.Printf("文件总大小：%d 字节，已上传：%d 字节\n", totalSize, uploadedSize)

	if uploadedSize >= totalSize {
		fmt.Println("文件已完全上传，无需继续！")
		return nil
	}

	file, err := os.Open(u.config.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Seek(uploadedSize, io.SeekStart)
	if err != nil {
		return err
	}

	currentPos := uploadedSize
	chunk := make([]byte, u.config.ChunkSize)

	for currentPos < totalSize {
		n, err := file.Read(chunk)
		if err != nil && err != io.EOF {
			return fmt.Errorf("读取文件分片失败：%v", err)
		}
		if n == 0 {
			break
		}

		newPos, err := u.uploadChunk(fileMD5, remoteFileName, currentPos, chunk[:n])
		if err != nil {
			return fmt.Errorf("上传分片失败（位置%d）：%v", currentPos, err)
		}

		currentPos = newPos
		progress := float64(currentPos) / float64(totalSize) * 100
		fmt.Printf("上传进度：%.2f%%（%d/%d 字节）\n", progress, currentPos, totalSize)
	}

	fmt.Println("文件上传完成！")
	return nil
}
