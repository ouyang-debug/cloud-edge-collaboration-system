// stress_test/metrics.go
package stresstest

import (
	"sort"
	"time"
)

type TestMetrics struct {
	TotalRequests   int64   `json:"total_requests"`
	SuccessRequests int64   `json:"success_requests"`
	FailedRequests  int64   `json:"failed_requests"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	P95LatencyMs    float64 `json:"p95_latency_ms"`
	P99LatencyMs    float64 `json:"p99_latency_ms"`
	QPS             float64 `json:"qps"`
	ErrorRate       float64 `json:"error_rate"`
}

type TestResult struct {
	Success      bool          `json:"success"`
	Duration     time.Duration `json:"duration"`
	Error        error         `json:"error,omitempty"`
	StatusCode   int           `json:"status_code,omitempty"`
	MessageCount int           `json:"message_count,omitempty"`
	SuccessCount int           `json:"success_count,omitempty"`
	FailCount    int           `json:"fail_count,omitempty"`
}

// 计算性能指标
func CalculateMetrics(results []TestResult) TestMetrics {
	var metrics TestMetrics

	latencies := make([]float64, 0)
	for _, r := range results {
		if r.Success {
			metrics.SuccessRequests++
			latencies = append(latencies, float64(r.Duration.Milliseconds()))
		} else {
			metrics.FailedRequests++
		}
		metrics.TotalRequests++
	}

	// 计算延迟百分位
	sort.Float64s(latencies)
	if len(latencies) > 0 {
		metrics.AvgLatencyMs = average(latencies)
		metrics.P95LatencyMs = percentile(latencies, 95)
		metrics.P99LatencyMs = percentile(latencies, 99)
	}

	metrics.QPS = float64(metrics.SuccessRequests) / totalTime.Seconds()
	metrics.ErrorRate = float64(metrics.FailedRequests) / float64(metrics.TotalRequests)

	return metrics
}
