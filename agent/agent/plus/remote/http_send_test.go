package remote

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPSender_Post(t *testing.T) {
	// // Create a test server
	// server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	// Check if it's a POST request
	// 	if r.Method != http.MethodPost {
	// 		t.Errorf("Expected POST request, got %s", r.Method)
	// 	}

	// 	// Check Content-Type header
	// 	if r.Header.Get("Content-Type") != "application/json" {
	// 		t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
	// 	}

	// 	// Send response
	// 	w.Header().Set("Content-Type", "application/json")
	// 	w.WriteHeader(http.StatusOK)
	// 	w.Write([]byte(`{"status":"success","message":"Test response"}`))
	// }))
	// defer server.Close()

	// Create HTTPSender
	sender := NewHTTPSender()
	URL := "http://192.168.66.13:8081/ceamcore/agent/dispatch"
	// Test Post method
	jsonPayload := "{\"content\":\"{\\\"cpuArch\\\":\\\"x86_64\\\",\\\"cpuCores\\\":\\\"16\\\",\\\"hostname\\\":\\\"bogon\\\",\\\"memory\\\":\\\"33565159424\\\",\\\"os\\\":\\\"CentOS Linux 7 (Core)\\\",\\\"storage\\\":\\\"536870912000\\\",\\\"timeOffset\\\":\\\"-0.000226323\\\",\\\"uptime\\\":\\\"up 17 weeks, 6 days, 5 hours, 22 minutes\\\"}\",\"contentId\":\"OS_BASE\",\"dataType\":\"json\",\"metadata\":{\"source\":\"agentname\",\"stepId\":\"none\"},\"taskId\":\"task123\",\"timestamp\":1770881501652}"
	response, statusCode, err := sender.Post(URL, jsonPayload)

	if err != nil {
		t.Errorf("Post failed: %v", err)
	}

	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statusCode)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}
}

func TestHTTPSender_PostWithHeaders(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check custom header
		if r.Header.Get("X-Custom-Header") != "test-value" {
			t.Errorf("Expected X-Custom-Header test-value, got %s", r.Header.Get("X-Custom-Header"))
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","message":"Test response with headers"}`))
	}))
	defer server.Close()

	// Create HTTPSender
	sender := NewHTTPSender()

	// Test PostWithHeaders method
	jsonPayload := `{"test":"data"}`
	headers := map[string]string{
		"X-Custom-Header": "test-value",
	}
	response, statusCode, err := sender.PostWithHeaders(server.URL, jsonPayload, headers)

	if err != nil {
		t.Errorf("PostWithHeaders failed: %v", err)
	}

	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statusCode)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}
}

func TestHTTPSender_NewHTTPSenderWithTimeout(t *testing.T) {
	// Create HTTPSender with custom timeout
	timeout := 10 * time.Second
	sender := NewHTTPSenderWithTimeout(timeout)

	if sender == nil {
		t.Error("Expected non-nil HTTPSender")
	}

	// Test with zero timeout (should use default)
	sender = NewHTTPSenderWithTimeout(0)
	if sender == nil {
		t.Error("Expected non-nil HTTPSender with zero timeout")
	}
}

func TestHTTPSender_Post_EmptyURL(t *testing.T) {
	// Create HTTPSender
	sender := NewHTTPSender()

	// Test with empty URL
	response, statusCode, err := sender.Post("", `{"test":"data"}`)

	if err == nil {
		t.Error("Expected error for empty URL")
	}

	if statusCode != 0 {
		t.Errorf("Expected status code 0 for empty URL, got %d", statusCode)
	}

	if response != "" {
		t.Error("Expected empty response for empty URL")
	}
}
