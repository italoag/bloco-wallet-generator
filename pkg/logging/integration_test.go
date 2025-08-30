package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEndToEndLoggingIntegration tests the complete flow from configuration to logging
func TestEndToEndLoggingIntegration(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		level      string
		format     string
		outputFile string
		logMessage string
		logLevel   LogLevel
		shouldLog  bool
	}{
		{
			name:       "info level logs info message",
			enabled:    true,
			level:      "info",
			format:     "text",
			outputFile: "",
			logMessage: "test info message",
			logLevel:   INFO,
			shouldLog:  true,
		},
		{
			name:       "info level blocks debug message",
			enabled:    true,
			level:      "info",
			format:     "text",
			outputFile: "",
			logMessage: "test debug message",
			logLevel:   DEBUG,
			shouldLog:  false,
		},
		{
			name:       "error level blocks info message",
			enabled:    true,
			level:      "error",
			format:     "text",
			outputFile: "",
			logMessage: "test info message",
			logLevel:   INFO,
			shouldLog:  false,
		},
		{
			name:       "disabled logger blocks all messages",
			enabled:    false,
			level:      "info",
			format:     "text",
			outputFile: "",
			logMessage: "test message",
			logLevel:   INFO,
			shouldLog:  false,
		},
		{
			name:       "json format produces json output",
			enabled:    true,
			level:      "info",
			format:     "json",
			outputFile: "",
			logMessage: "test json message",
			logLevel:   INFO,
			shouldLog:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logFile string
			var tempDir string

			// Create temporary file if outputFile is specified or we need to capture output
			if tt.outputFile != "" || tt.shouldLog {
				var err error
				tempDir, err = os.MkdirTemp("", "logging_integration_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				defer os.RemoveAll(tempDir)

				logFile = filepath.Join(tempDir, "test.log")
				if tt.outputFile == "" {
					// Use temp file to capture output for verification
					tt.outputFile = logFile
				}
			}

			// Create logger using the factory function
			logger, err := NewSecureLoggerFromConfig(tt.enabled, tt.level, tt.format, tt.outputFile)
			if err != nil {
				t.Fatalf("NewSecureLoggerFromConfig() error: %v", err)
			}
			defer logger.Close()

			// Test IsEnabled method
			isEnabled := logger.IsEnabled(tt.logLevel)
			expectedEnabled := tt.enabled && tt.shouldLog
			if isEnabled != expectedEnabled {
				t.Errorf("IsEnabled(%v) = %v, want %v", tt.logLevel, isEnabled, expectedEnabled)
			}

			// Log the message
			var logErr error
			switch tt.logLevel {
			case ERROR:
				logErr = logger.Error(tt.logMessage)
			case WARN:
				logErr = logger.Warn(tt.logMessage)
			case INFO:
				logErr = logger.Info(tt.logMessage)
			case DEBUG:
				logErr = logger.Debug(tt.logMessage)
			}

			if logErr != nil {
				t.Errorf("Log method error: %v", logErr)
			}

			// Verify output if logging should occur
			if tt.shouldLog && tt.enabled && logFile != "" {
				// Close logger to flush any buffers
				logger.Close()

				// Check if file exists and has content
				content, err := os.ReadFile(logFile)
				if err != nil {
					t.Errorf("Failed to read log file: %v", err)
					return
				}

				contentStr := string(content)
				if len(contentStr) == 0 {
					t.Errorf("Expected log output, but file is empty")
					return
				}

				// Verify message is in output
				if !strings.Contains(contentStr, tt.logMessage) {
					t.Errorf("Log output does not contain expected message '%s', got: %s", tt.logMessage, contentStr)
				}

				// Verify format-specific content
				switch tt.format {
				case "json":
					if !strings.Contains(contentStr, `"message"`) {
						t.Errorf("JSON format should contain '\"message\"' field, got: %s", contentStr)
					}
					if !strings.Contains(contentStr, `"level"`) {
						t.Errorf("JSON format should contain '\"level\"' field, got: %s", contentStr)
					}
				case "structured":
					if !strings.Contains(contentStr, "message=") {
						t.Errorf("Structured format should contain 'message=' field, got: %s", contentStr)
					}
					if !strings.Contains(contentStr, "level=") {
						t.Errorf("Structured format should contain 'level=' field, got: %s", contentStr)
					}
				case "text":
					levelStr := tt.logLevel.String()
					if !strings.Contains(contentStr, levelStr) {
						t.Errorf("Text format should contain level '%s', got: %s", levelStr, contentStr)
					}
				}
			} else if !tt.shouldLog && logFile != "" {
				// Verify no output when logging should be blocked
				logger.Close()

				if _, err := os.Stat(logFile); err == nil {
					content, _ := os.ReadFile(logFile)
					if len(content) > 0 {
						t.Errorf("Expected no log output, but got: %s", string(content))
					}
				}
			}
		})
	}
}

// TestLoggerFactoryErrorHandling tests error handling in the factory function
func TestLoggerFactoryErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		level      string
		format     string
		outputFile string
		wantErr    bool
	}{
		{
			name:       "invalid log level",
			enabled:    true,
			level:      "invalid",
			format:     "text",
			outputFile: "",
			wantErr:    true,
		},
		{
			name:       "invalid log format",
			enabled:    true,
			level:      "info",
			format:     "invalid",
			outputFile: "",
			wantErr:    true,
		},
		{
			name:       "disabled logger with invalid level should not error",
			enabled:    false,
			level:      "invalid",
			format:     "text",
			outputFile: "",
			wantErr:    false, // Disabled logger should not validate other parameters
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewSecureLoggerFromConfig(tt.enabled, tt.level, tt.format, tt.outputFile)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewSecureLoggerFromConfig() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewSecureLoggerFromConfig() unexpected error: %v", err)
				return
			}

			if logger == nil {
				t.Errorf("NewSecureLoggerFromConfig() returned nil logger")
				return
			}

			// Clean up
			logger.Close()
		})
	}
}

// TestWalletGenerationLogging tests the specialized wallet generation logging
func TestWalletGenerationLogging(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wallet_logging_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "wallet.log")

	// Create logger with JSON format for easier parsing
	logger, err := NewSecureLoggerFromConfig(true, "info", "json", logFile)
	if err != nil {
		t.Fatalf("NewSecureLoggerFromConfig() error: %v", err)
	}
	defer logger.Close()

	// Log wallet generation
	err = logger.LogWalletGenerated("0xdab0a527c44cc6a7f3b6fe1c375d0398db62279e", 12345, 1500000000, 3)
	if err != nil {
		t.Errorf("LogWalletGenerated() error: %v", err)
	}

	// Close to flush
	logger.Close()

	// Verify output
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if len(contentStr) == 0 {
		t.Fatalf("Log file is empty")
	}

	// Verify wallet address is logged
	if !strings.Contains(contentStr, "0xdab0a527c44cc6a7f3b6fe1c375d0398db62279e") {
		t.Errorf("Log should contain wallet address, got: %s", contentStr)
	}

	// Verify attempts are logged
	if !strings.Contains(contentStr, "12345") {
		t.Errorf("Log should contain attempt count, got: %s", contentStr)
	}

	// Verify thread ID is logged
	if !strings.Contains(contentStr, "3") {
		t.Errorf("Log should contain thread ID, got: %s", contentStr)
	}

	// Verify message indicates wallet generation
	if !strings.Contains(contentStr, "Wallet generated") {
		t.Errorf("Log should contain 'Wallet generated' message, got: %s", contentStr)
	}

	// Verify JSON structure
	if !strings.Contains(contentStr, `"address"`) {
		t.Errorf("JSON log should contain 'address' field, got: %s", contentStr)
	}

	if !strings.Contains(contentStr, `"attempts"`) {
		t.Errorf("JSON log should contain 'attempts' field, got: %s", contentStr)
	}

	if !strings.Contains(contentStr, `"thread_id"`) {
		t.Errorf("JSON log should contain 'thread_id' field, got: %s", contentStr)
	}
}
