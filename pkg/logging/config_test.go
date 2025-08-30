package logging

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSecureLoggerFromConfig(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		level      string
		format     string
		outputFile string
		wantErr    bool
		wantNoop   bool
	}{
		{
			name:       "disabled logging",
			enabled:    false,
			level:      "info",
			format:     "text",
			outputFile: "",
			wantErr:    false,
			wantNoop:   true,
		},
		{
			name:       "valid text format",
			enabled:    true,
			level:      "info",
			format:     "text",
			outputFile: "",
			wantErr:    false,
			wantNoop:   false,
		},
		{
			name:       "valid json format",
			enabled:    true,
			level:      "debug",
			format:     "json",
			outputFile: "",
			wantErr:    false,
			wantNoop:   false,
		},
		{
			name:       "valid structured format",
			enabled:    true,
			level:      "error",
			format:     "structured",
			outputFile: "",
			wantErr:    false,
			wantNoop:   false,
		},
		{
			name:       "invalid log level",
			enabled:    true,
			level:      "invalid",
			format:     "text",
			outputFile: "",
			wantErr:    true,
			wantNoop:   false,
		},
		{
			name:       "invalid log format",
			enabled:    true,
			level:      "info",
			format:     "invalid",
			outputFile: "",
			wantErr:    true,
			wantNoop:   false,
		},
		{
			name:       "case insensitive format",
			enabled:    true,
			level:      "info",
			format:     "JSON",
			outputFile: "",
			wantErr:    false,
			wantNoop:   false,
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

			// Test that disabled logger doesn't log
			if tt.wantNoop {
				fileLogger, ok := logger.(*FileSecureLogger)
				if !ok {
					t.Errorf("Expected FileSecureLogger, got %T", logger)
					return
				}

				if fileLogger.config.Enabled {
					t.Errorf("Expected disabled logger, but config.Enabled = true")
				}

				if fileLogger.writer != io.Discard {
					t.Errorf("Expected io.Discard writer for disabled logger")
				}
			}

			// Test IsEnabled method
			if !tt.wantNoop {
				// For enabled loggers, test level checking
				if !logger.IsEnabled(INFO) && tt.level != "error" {
					t.Errorf("Expected INFO level to be enabled for level %s", tt.level)
				}
			}

			// Clean up
			if closer, ok := logger.(interface{ Close() error }); ok {
				closer.Close()
			}
		})
	}
}

func TestNewSecureLoggerFromConfigWithFile(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "logging_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "test.log")

	logger, err := NewSecureLoggerFromConfig(true, "info", "text", logFile)
	if err != nil {
		t.Fatalf("NewSecureLoggerFromConfig() error: %v", err)
	}
	defer logger.Close()

	// Test logging to file
	err = logger.Info("test message")
	if err != nil {
		t.Errorf("Info() error: %v", err)
	}

	// Flush to ensure the message is written to file
	if err := logger.Flush(); err != nil {
		t.Errorf("Flush() error: %v", err)
	}

	// Verify file was created and contains content
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Log file was not created: %s", logFile)
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Errorf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Errorf("Log file is empty")
	}

	contentStr := string(content)
	if !containsString(contentStr, "test message") {
		t.Errorf("Log file does not contain expected message, got: %s", contentStr)
	}
}

func TestLogConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *LogConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &LogConfig{
				Enabled:     true,
				Level:       INFO,
				Format:      TEXT,
				OutputFile:  "",
				MaxFileSize: 1024,
				MaxFiles:    5,
				BufferSize:  100,
			},
			wantErr: false,
		},
		{
			name: "invalid max file size",
			config: &LogConfig{
				Enabled:     true,
				Level:       INFO,
				Format:      TEXT,
				OutputFile:  "",
				MaxFileSize: -1,
				MaxFiles:    5,
				BufferSize:  100,
			},
			wantErr: true,
		},
		{
			name: "invalid max files",
			config: &LogConfig{
				Enabled:     true,
				Level:       INFO,
				Format:      TEXT,
				OutputFile:  "",
				MaxFileSize: 1024,
				MaxFiles:    -1,
				BufferSize:  100,
			},
			wantErr: true,
		},
		{
			name: "invalid buffer size",
			config: &LogConfig{
				Enabled:     true,
				Level:       INFO,
				Format:      TEXT,
				OutputFile:  "",
				MaxFileSize: 1024,
				MaxFiles:    5,
				BufferSize:  -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestIsEnabledMethod(t *testing.T) {
	tests := []struct {
		name        string
		configLevel LogLevel
		testLevel   LogLevel
		enabled     bool
		want        bool
	}{
		{
			name:        "disabled logger",
			configLevel: INFO,
			testLevel:   INFO,
			enabled:     false,
			want:        false,
		},
		{
			name:        "error level allows error",
			configLevel: ERROR,
			testLevel:   ERROR,
			enabled:     true,
			want:        true,
		},
		{
			name:        "error level blocks info",
			configLevel: ERROR,
			testLevel:   INFO,
			enabled:     true,
			want:        false,
		},
		{
			name:        "info level allows error",
			configLevel: INFO,
			testLevel:   ERROR,
			enabled:     true,
			want:        true,
		},
		{
			name:        "info level allows info",
			configLevel: INFO,
			testLevel:   INFO,
			enabled:     true,
			want:        true,
		},
		{
			name:        "info level blocks debug",
			configLevel: INFO,
			testLevel:   DEBUG,
			enabled:     true,
			want:        false,
		},
		{
			name:        "debug level allows all",
			configLevel: DEBUG,
			testLevel:   DEBUG,
			enabled:     true,
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LogConfig{
				Enabled:     tt.enabled,
				Level:       tt.configLevel,
				Format:      TEXT,
				OutputFile:  "",
				MaxFileSize: 1024,
				MaxFiles:    5,
				BufferSize:  100,
			}

			logger, err := NewSecureLogger(config)
			if err != nil {
				t.Fatalf("NewSecureLogger() error: %v", err)
			}
			defer logger.Close()

			got := logger.IsEnabled(tt.testLevel)
			if got != tt.want {
				t.Errorf("IsEnabled(%v) = %v, want %v", tt.testLevel, got, tt.want)
			}
		})
	}
}

func TestLoggerFactoryWithInvalidFile(t *testing.T) {
	// Try to create logger with invalid file path (directory that doesn't exist)
	invalidPath := "/nonexistent/directory/test.log"

	logger, err := NewSecureLoggerFromConfig(true, "info", "text", invalidPath)
	if err != nil {
		t.Fatalf("NewSecureLoggerFromConfig() should not fail with invalid path, got error: %v", err)
	}
	defer logger.Close()

	// The logger should fallback to stdout and still work
	err = logger.Info("test message")
	if err != nil {
		t.Errorf("Info() should work even with invalid file path, got error: %v", err)
	}
}

// Helper function to check if a string contains a substring
func containsString(text, substr string) bool {
	return strings.Contains(text, substr)
}
