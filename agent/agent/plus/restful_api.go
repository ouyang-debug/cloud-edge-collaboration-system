package plus

import (
	"agent/plus/remote"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (tm *TaskManager) PostStatusData(payload map[string]interface{}) ([]byte, error) {
	url := tm.serverConfig.Server + tm.yamlConfig.StatusApi

	// 序列化payload为JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	response, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send POST request: %v", err)
	}
	defer response.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	// 解析响应
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// 检查响应状态
	if resp.Code != 0 {
		return nil, fmt.Errorf("register failed: %s", resp.Msg)
	}

	return nil, nil

}

func (tm *TaskManager) PostResultData(payload string) ([]byte, error) {

	url := tm.serverConfig.Server + tm.yamlConfig.ResultDataApi

	sender := remote.NewHTTPSender()

	response, statusCode, err := sender.Post(url, payload)

	if err != nil {
		return nil, fmt.Errorf("Post failed: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("Expected status code %d, got %d", http.StatusOK, statusCode)
	}

	if response == "" {
		return nil, fmt.Errorf("Expected non-empty response")
	}

	return nil, nil

}

func (tm *TaskManager) PostResultSync(payload string) ([]byte, error) {

	url := tm.serverConfig.Server + tm.yamlConfig.ResultSyncApi

	sender := remote.NewHTTPSender()

	response, statusCode, err := sender.Post(url, payload)

	if err != nil {
		return nil, fmt.Errorf("Post failed: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("Expected status code %d, got %d", http.StatusOK, statusCode)
	}

	if response == "" {
		return nil, fmt.Errorf("Expected non-empty response")
	}

	return nil, nil

}
