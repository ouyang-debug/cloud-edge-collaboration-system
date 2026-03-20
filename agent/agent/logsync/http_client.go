package logsync

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// 日志数据请求体
type LogDataRequest struct {
	TaskId   string `json:"taskId"`
	AgentId  string `json:"agentId"`
	LogLevel string `json:"logLevel"`
	LogType  string `json:"logType"`
	Content  []byte `json:"content"`
	//Content     string    `json:"content"`
	ReadTime    time.Time `json:"read_time"`
	Offset      int64     `json:"offset"`
	Size        int64     `json:"size"`
	ContentHash string    `json:"content_hash"`
}

// HTTP客户端
type HTTPClient struct {
	serverURL string
	client    *http.Client
}

// 创建新的HTTP客户端
func NewHTTPClient(serverURL string) *HTTPClient {
	return &HTTPClient{
		serverURL: serverURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// 发送日志数据
func (hc *HTTPClient) SendLogData(data LogDataRequest) error {
	// 将数据转换为JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", hc.serverURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := hc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    "HTTP request failed",
		}
	}

	return nil
}

// HTTP错误
type HTTPError struct {
	StatusCode int
	Message    string
}

// 实现error接口
func (e *HTTPError) Error() string {
	return e.Message
}
