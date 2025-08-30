package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPerformanceMetrics_LogWorkerStartup(t *testing.T) {
	tests := []struct {
		name     string
		threadID int
		config   map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "basic worker startup",
			threadID: 1,
			config: map[string]interface{}{
				"threads": 4,
				"prefix":  "abc",
				"suffix":  "def",
			},
			wantErr: false,
		},
		{
			name:     "worker startup with sensitive data filtered",
			threadID: 2,
			config: map[string]interface{}{
				"threads":     8,
				"private_key": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				"prefix":      "test",
			},
			wantErr: false,
		},
		{
			name:     "worker startup with empty config",
			threadID: 3,
			config:   map[string]interface{}{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := createTestLogger(&buf, INFO, JSON)

			err := logger.LogWorkerStartup(tt.threadID, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("LogWorkerStartup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()

				// Verify the log contains expected fields
				if !strings.Contains(output, "Worker thread started") {
					t.Errorf("Expected log message not found in output: %s", output)
				}

				// Parse JSON to verify structure
				var entry LogEntry
				if err := json.Unmarshal([]byte(output), &entry); err != nil {
					t.Errorf("Failed to parse JSON output: %v", err)
					return
				}

				// Verify thread ID is logged
				if threadIDField, ok := entry.Fields["thread_id"]; !ok || threadIDField != float64(tt.threadID) {
					t.Errorf("Expected thread_id %d, got %v", tt.threadID, threadIDField)
				}

				// Verify status is logged
				if statusField, ok := entry.Fields["status"]; !ok || statusField != "started" {
					t.Errorf("Expected status 'started', got %v", statusField)
				}

				// Verify safe config parameters are logged
				for key, expectedValue := range tt.config {
					if isSafeParameter(key) {
						if actualValue, ok := entry.Fields[key]; !ok {
							t.Errorf("Expected safe parameter %s to be logged", key)
						} else {
							// Handle type conversion for JSON unmarshaling (numbers become float64)
							switch v := expectedValue.(type) {
							case int:
								if actualValue != float64(v) {
									t.Errorf("Expected %s=%v, got %v", key, expectedValue, actualValue)
								}
							default:
								if actualValue != expectedValue {
									t.Errorf("Expected %s=%v, got %v", key, expectedValue, actualValue)
								}
							}
						}
					}
				}

				// Verify sensitive data is not logged
				if _, ok := entry.Fields["private_key"]; ok {
					t.Errorf("Sensitive parameter 'private_key' should not be logged")
				}
			}
		})
	}
}

func TestPerformanceMetrics_LogPerformanceMetrics(t *testing.T) {
	tests := []struct {
		name    string
		metrics PerformanceMetrics
		wantErr bool
	}{
		{
			name: "complete performance metrics",
			metrics: PerformanceMetrics{
				WalletsPerSecond: 1500.5,
				TotalWallets:     10000,
				TotalAttempts:    50000,
				AverageDuration:  2.5,
				MinDuration:      1.0,
				MaxDuration:      10.0,
				ThreadCount:      8,
				CPUUsage:         75.5,
				MemoryUsage:      1024 * 1024 * 100, // 100MB
				SuccessRate:      99.5,
				ErrorRate:        0.5,
				WindowStart:      time.Now().UTC(),
				WindowDuration:   time.Minute * 5,
			},
			wantErr: false,
		},
		{
			name: "minimal performance metrics",
			metrics: PerformanceMetrics{
				WalletsPerSecond: 500.0,
				TotalWallets:     1000,
				TotalAttempts:    5000,
				AverageDuration:  5.0,
				ThreadCount:      4,
				WindowStart:      time.Now().UTC(),
				WindowDuration:   time.Minute,
			},
			wantErr: false,
		},
		{
			name: "zero values handled correctly",
			metrics: PerformanceMetrics{
				WalletsPerSecond: 0,
				TotalWallets:     0,
				TotalAttempts:    0,
				AverageDuration:  0,
				ThreadCount:      1,
				WindowStart:      time.Now().UTC(),
				WindowDuration:   0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := createTestLogger(&buf, INFO, JSON)

			err := logger.LogPerformanceMetrics(tt.metrics)
			if (err != nil) != tt.wantErr {
				t.Errorf("LogPerformanceMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()

				// Verify the log contains expected message
				if !strings.Contains(output, "Performance metrics") {
					t.Errorf("Expected log message not found in output: %s", output)
				}

				// Parse JSON to verify structure
				var entry LogEntry
				if err := json.Unmarshal([]byte(output), &entry); err != nil {
					t.Errorf("Failed to parse JSON output: %v", err)
					return
				}

				// Verify required fields are always present
				requiredFields := map[string]interface{}{
					"wallets_per_second": tt.metrics.WalletsPerSecond,
					"total_wallets":      float64(tt.metrics.TotalWallets),
					"total_attempts":     float64(tt.metrics.TotalAttempts),
					"avg_duration_ms":    tt.metrics.AverageDuration,
					"thread_count":       float64(tt.metrics.ThreadCount),
					"window_duration_ns": float64(tt.metrics.WindowDuration.Nanoseconds()),
				}

				for field, expectedValue := range requiredFields {
					if actualValue, ok := entry.Fields[field]; !ok {
						t.Errorf("Required field %s not found in log", field)
					} else if actualValue != expectedValue {
						t.Errorf("Expected %s=%v, got %v", field, expectedValue, actualValue)
					}
				}

				// Verify optional fields are only present when they have meaningful values
				if tt.metrics.MinDuration > 0 {
					if minDur, ok := entry.Fields["min_duration_ms"]; !ok || minDur != tt.metrics.MinDuration {
						t.Errorf("Expected min_duration_ms=%v, got %v", tt.metrics.MinDuration, minDur)
					}
				} else {
					if _, ok := entry.Fields["min_duration_ms"]; ok {
						t.Errorf("min_duration_ms should not be present when value is 0")
					}
				}

				if tt.metrics.MaxDuration > 0 {
					if maxDur, ok := entry.Fields["max_duration_ms"]; !ok || maxDur != tt.metrics.MaxDuration {
						t.Errorf("Expected max_duration_ms=%v, got %v", tt.metrics.MaxDuration, maxDur)
					}
				} else {
					if _, ok := entry.Fields["max_duration_ms"]; ok {
						t.Errorf("max_duration_ms should not be present when value is 0")
					}
				}

				if tt.metrics.CPUUsage > 0 {
					if cpuUsage, ok := entry.Fields["cpu_usage_percent"]; !ok || cpuUsage != tt.metrics.CPUUsage {
						t.Errorf("Expected cpu_usage_percent=%v, got %v", tt.metrics.CPUUsage, cpuUsage)
					}
				} else {
					if _, ok := entry.Fields["cpu_usage_percent"]; ok {
						t.Errorf("cpu_usage_percent should not be present when value is 0")
					}
				}

				if tt.metrics.MemoryUsage > 0 {
					if memUsage, ok := entry.Fields["memory_usage_bytes"]; !ok || memUsage != float64(tt.metrics.MemoryUsage) {
						t.Errorf("Expected memory_usage_bytes=%v, got %v", tt.metrics.MemoryUsage, memUsage)
					}
				} else {
					if _, ok := entry.Fields["memory_usage_bytes"]; ok {
						t.Errorf("memory_usage_bytes should not be present when value is 0")
					}
				}
			}
		})
	}
}

func TestPerformanceMetrics_LogResourceUsage(t *testing.T) {
	tests := []struct {
		name        string
		cpuPercent  float64
		memoryBytes int64
		threadCount int
		wantErr     bool
	}{
		{
			name:        "complete resource usage",
			cpuPercent:  85.5,
			memoryBytes: 1024 * 1024 * 200, // 200MB
			threadCount: 8,
			wantErr:     false,
		},
		{
			name:        "cpu usage only",
			cpuPercent:  45.0,
			memoryBytes: 0,
			threadCount: 4,
			wantErr:     false,
		},
		{
			name:        "memory usage only",
			cpuPercent:  -1,               // Negative indicates unavailable
			memoryBytes: 1024 * 1024 * 50, // 50MB
			threadCount: 2,
			wantErr:     false,
		},
		{
			name:        "thread count only",
			cpuPercent:  -1,
			memoryBytes: 0,
			threadCount: 16,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := createTestLogger(&buf, DEBUG, JSON) // Use DEBUG level to capture resource usage logs

			err := logger.LogResourceUsage(tt.cpuPercent, tt.memoryBytes, tt.threadCount)
			if (err != nil) != tt.wantErr {
				t.Errorf("LogResourceUsage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()

				// Verify the log contains expected message
				if !strings.Contains(output, "Resource usage") {
					t.Errorf("Expected log message not found in output: %s", output)
				}

				// Parse JSON to verify structure
				var entry LogEntry
				if err := json.Unmarshal([]byte(output), &entry); err != nil {
					t.Errorf("Failed to parse JSON output: %v", err)
					return
				}

				// Verify thread count is always present
				if threadCount, ok := entry.Fields["thread_count"]; !ok || threadCount != float64(tt.threadCount) {
					t.Errorf("Expected thread_count=%d, got %v", tt.threadCount, threadCount)
				}

				// Verify CPU usage is only present when >= 0
				if tt.cpuPercent >= 0 {
					if cpuUsage, ok := entry.Fields["cpu_usage_percent"]; !ok || cpuUsage != tt.cpuPercent {
						t.Errorf("Expected cpu_usage_percent=%v, got %v", tt.cpuPercent, cpuUsage)
					}
				} else {
					if _, ok := entry.Fields["cpu_usage_percent"]; ok {
						t.Errorf("cpu_usage_percent should not be present when value is negative")
					}
				}

				// Verify memory usage is only present when > 0
				if tt.memoryBytes > 0 {
					if memUsage, ok := entry.Fields["memory_usage_bytes"]; !ok || memUsage != float64(tt.memoryBytes) {
						t.Errorf("Expected memory_usage_bytes=%d, got %v", tt.memoryBytes, memUsage)
					}
				} else {
					if _, ok := entry.Fields["memory_usage_bytes"]; ok {
						t.Errorf("memory_usage_bytes should not be present when value is 0")
					}
				}
			}
		})
	}
}

func TestPerformanceMetrics_LogLevelFiltering(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  LogLevel
		shouldLog bool
	}{
		{
			name:      "INFO level logs performance metrics",
			logLevel:  INFO,
			shouldLog: true,
		},
		{
			name:      "DEBUG level logs performance metrics",
			logLevel:  DEBUG,
			shouldLog: true,
		},
		{
			name:      "WARN level does not log performance metrics",
			logLevel:  WARN,
			shouldLog: false,
		},
		{
			name:      "ERROR level does not log performance metrics",
			logLevel:  ERROR,
			shouldLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := createTestLogger(&buf, tt.logLevel, JSON)

			metrics := PerformanceMetrics{
				WalletsPerSecond: 1000.0,
				TotalWallets:     5000,
				TotalAttempts:    25000,
				AverageDuration:  3.0,
				ThreadCount:      4,
				WindowStart:      time.Now().UTC(),
				WindowDuration:   time.Minute,
			}

			err := logger.LogPerformanceMetrics(metrics)
			if err != nil {
				t.Errorf("LogPerformanceMetrics() error = %v", err)
				return
			}

			output := buf.String()
			hasOutput := len(strings.TrimSpace(output)) > 0

			if tt.shouldLog && !hasOutput {
				t.Errorf("Expected log output at level %s, but got none", tt.logLevel.String())
			} else if !tt.shouldLog && hasOutput {
				t.Errorf("Expected no log output at level %s, but got: %s", tt.logLevel.String(), output)
			}
		})
	}
}

func TestPerformanceMetrics_ThreadSafety(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, DEBUG, JSON) // Use DEBUG to capture all log levels

	// Test concurrent logging of performance metrics
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(threadID int) {
			defer func() { done <- true }()

			// Log worker startup
			config := map[string]interface{}{
				"threads": 4,
				"prefix":  "test",
			}
			if err := logger.LogWorkerStartup(threadID, config); err != nil {
				t.Errorf("LogWorkerStartup() error = %v", err)
			}

			// Log performance metrics
			metrics := PerformanceMetrics{
				WalletsPerSecond: float64(1000 + threadID*100),
				TotalWallets:     int64(5000 + threadID*1000),
				TotalAttempts:    int64(25000 + threadID*5000),
				AverageDuration:  float64(3.0 + float64(threadID)*0.5),
				ThreadCount:      threadID + 1,
				WindowStart:      time.Now().UTC(),
				WindowDuration:   time.Minute,
			}
			if err := logger.LogPerformanceMetrics(metrics); err != nil {
				t.Errorf("LogPerformanceMetrics() error = %v", err)
			}

			// Log resource usage
			if err := logger.LogResourceUsage(float64(50+threadID*5), int64(1024*1024*(100+threadID*10)), threadID+1); err != nil {
				t.Errorf("LogResourceUsage() error = %v", err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify that we got some output (exact verification is difficult due to concurrency)
	output := buf.String()
	if len(strings.TrimSpace(output)) == 0 {
		t.Errorf("Expected some log output from concurrent operations, but got none")
	}

	// Count the number of log entries
	lines := strings.Split(strings.TrimSpace(output), "\n")
	expectedLines := 30 // 10 threads * 3 log calls each
	if len(lines) != expectedLines {
		t.Errorf("Expected %d log lines, got %d", expectedLines, len(lines))
	}
}
