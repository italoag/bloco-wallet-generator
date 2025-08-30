package logging

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected string
	}{
		{"ERROR level", ERROR, "ERROR"},
		{"WARN level", WARN, "WARN"},
		{"INFO level", INFO, "INFO"},
		{"DEBUG level", DEBUG, "DEBUG"},
		{"Unknown level", LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("LogLevel.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    LogLevel
		expectError bool
	}{
		{"ERROR uppercase", "ERROR", ERROR, false},
		{"error lowercase", "error", ERROR, false},
		{"WARN uppercase", "WARN", WARN, false},
		{"WARNING full word", "WARNING", WARN, false},
		{"warn lowercase", "warn", WARN, false},
		{"INFO uppercase", "INFO", INFO, false},
		{"info lowercase", "info", INFO, false},
		{"DEBUG uppercase", "DEBUG", DEBUG, false},
		{"debug lowercase", "debug", DEBUG, false},
		{"Whitespace trimmed", "  INFO  ", INFO, false},
		{"Mixed case", "WaRn", WARN, false},
		{"Invalid level", "INVALID", INFO, true},
		{"Empty string", "", INFO, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseLogLevel(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseLogLevel(%q) expected error, got nil", tt.input)
				}
				// For error cases, result should be INFO (default)
				if result != INFO {
					t.Errorf("ParseLogLevel(%q) error case should return INFO, got %v", tt.input, result)
				}
			} else {
				if err != nil {
					t.Errorf("ParseLogLevel(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestLogFormat_String(t *testing.T) {
	tests := []struct {
		name     string
		format   LogFormat
		expected string
	}{
		{"JSON format", JSON, "JSON"},
		{"TEXT format", TEXT, "TEXT"},
		{"STRUCTURED format", STRUCTURED, "STRUCTURED"},
		{"Unknown format", LogFormat(999), "TEXT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.format.String()
			if result != tt.expected {
				t.Errorf("LogFormat.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDefaultLogConfig(t *testing.T) {
	config := DefaultLogConfig()

	if !config.Enabled {
		t.Error("DefaultLogConfig().Enabled should be true")
	}
	if config.Level != INFO {
		t.Errorf("DefaultLogConfig().Level = %v, want %v", config.Level, INFO)
	}
	if config.Format != TEXT {
		t.Errorf("DefaultLogConfig().Format = %v, want %v", config.Format, TEXT)
	}
	if config.OutputFile != "" {
		t.Errorf("DefaultLogConfig().OutputFile = %q, want empty string", config.OutputFile)
	}
	if config.MaxFileSize != 10*1024*1024 {
		t.Errorf("DefaultLogConfig().MaxFileSize = %d, want %d", config.MaxFileSize, 10*1024*1024)
	}
	if config.MaxFiles != 5 {
		t.Errorf("DefaultLogConfig().MaxFiles = %d, want %d", config.MaxFiles, 5)
	}
	if config.BufferSize != 1000 {
		t.Errorf("DefaultLogConfig().BufferSize = %d, want %d", config.BufferSize, 1000)
	}
}

func TestLogConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *LogConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid config",
			config:      DefaultLogConfig(),
			expectError: false,
		},
		{
			name: "Zero MaxFileSize",
			config: &LogConfig{
				MaxFileSize: 0,
				MaxFiles:    5,
				BufferSize:  1000,
			},
			expectError: true,
			errorMsg:    "MaxFileSize must be positive",
		},
		{
			name: "Negative MaxFileSize",
			config: &LogConfig{
				MaxFileSize: -1,
				MaxFiles:    5,
				BufferSize:  1000,
			},
			expectError: true,
			errorMsg:    "MaxFileSize must be positive",
		},
		{
			name: "Negative MaxFiles",
			config: &LogConfig{
				MaxFileSize: 1024,
				MaxFiles:    -1,
				BufferSize:  1000,
			},
			expectError: true,
			errorMsg:    "MaxFiles must be non-negative",
		},
		{
			name: "Negative BufferSize",
			config: &LogConfig{
				MaxFileSize: 1024,
				MaxFiles:    5,
				BufferSize:  -1,
			},
			expectError: true,
			errorMsg:    "BufferSize must be non-negative",
		},
		{
			name: "Zero MaxFiles and BufferSize allowed",
			config: &LogConfig{
				MaxFileSize: 1024,
				MaxFiles:    0,
				BufferSize:  0,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("LogConfig.Validate() expected error, got nil")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("LogConfig.Validate() error = %q, want to contain %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("LogConfig.Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNewLogField(t *testing.T) {
	key := "test_key"
	value := "test_value"

	field := NewLogField(key, value)

	if field.Key != key {
		t.Errorf("NewLogField().Key = %q, want %q", field.Key, key)
	}
	if field.Value != value {
		t.Errorf("NewLogField().Value = %v, want %v", field.Value, value)
	}
}

func TestNewLogEntry(t *testing.T) {
	level := ERROR
	message := "test message"

	entry := NewLogEntry(level, message)

	if entry.Level != level {
		t.Errorf("NewLogEntry().Level = %v, want %v", entry.Level, level)
	}
	if entry.Message != message {
		t.Errorf("NewLogEntry().Message = %q, want %q", entry.Message, message)
	}
	if entry.Fields == nil {
		t.Error("NewLogEntry().Fields should be initialized")
	}
	if time.Since(entry.Timestamp) > time.Second {
		t.Error("NewLogEntry().Timestamp should be recent")
	}
}

func TestLogEntry_WithOperation(t *testing.T) {
	entry := NewLogEntry(INFO, "test")
	operation := "wallet_generation"

	result := entry.WithOperation(operation)

	if result != entry {
		t.Error("WithOperation should return the same entry for chaining")
	}
	if entry.Operation != operation {
		t.Errorf("WithOperation() entry.Operation = %q, want %q", entry.Operation, operation)
	}
}

func TestLogEntry_WithThreadID(t *testing.T) {
	entry := NewLogEntry(INFO, "test")
	threadID := 42

	result := entry.WithThreadID(threadID)

	if result != entry {
		t.Error("WithThreadID should return the same entry for chaining")
	}
	if entry.ThreadID != threadID {
		t.Errorf("WithThreadID() entry.ThreadID = %d, want %d", entry.ThreadID, threadID)
	}
}

func TestLogEntry_WithError(t *testing.T) {
	entry := NewLogEntry(ERROR, "test")

	t.Run("With error", func(t *testing.T) {
		err := errors.New("test error")
		result := entry.WithError(err)

		if result != entry {
			t.Error("WithError should return the same entry for chaining")
		}
		if entry.Error != err.Error() {
			t.Errorf("WithError() entry.Error = %q, want %q", entry.Error, err.Error())
		}
	})

	t.Run("With nil error", func(t *testing.T) {
		entry2 := NewLogEntry(ERROR, "test")
		result := entry2.WithError(nil)

		if result != entry2 {
			t.Error("WithError should return the same entry for chaining")
		}
		if entry2.Error != "" {
			t.Errorf("WithError(nil) entry.Error = %q, want empty string", entry2.Error)
		}
	})
}

func TestLogEntry_WithFields(t *testing.T) {
	entry := NewLogEntry(INFO, "test")
	field1 := NewLogField("key1", "value1")
	field2 := NewLogField("key2", 42)

	result := entry.WithFields(field1, field2)

	if result != entry {
		t.Error("WithFields should return the same entry for chaining")
	}
	if entry.Fields["key1"] != "value1" {
		t.Errorf("WithFields() entry.Fields[key1] = %v, want %v", entry.Fields["key1"], "value1")
	}
	if entry.Fields["key2"] != 42 {
		t.Errorf("WithFields() entry.Fields[key2] = %v, want %v", entry.Fields["key2"], 42)
	}
}

func TestLogEntry_WithField(t *testing.T) {
	entry := NewLogEntry(INFO, "test")
	key := "test_key"
	value := "test_value"

	result := entry.WithField(key, value)

	if result != entry {
		t.Error("WithField should return the same entry for chaining")
	}
	if entry.Fields[key] != value {
		t.Errorf("WithField() entry.Fields[%q] = %v, want %v", key, entry.Fields[key], value)
	}
}

func TestLogEntry_Chaining(t *testing.T) {
	// Test that all methods can be chained together
	entry := NewLogEntry(INFO, "test message").
		WithOperation("test_operation").
		WithThreadID(123).
		WithError(errors.New("test error")).
		WithField("key1", "value1").
		WithFields(
			NewLogField("key2", "value2"),
			NewLogField("key3", 42),
		)

	if entry.Level != INFO {
		t.Errorf("Chained entry.Level = %v, want %v", entry.Level, INFO)
	}
	if entry.Message != "test message" {
		t.Errorf("Chained entry.Message = %q, want %q", entry.Message, "test message")
	}
	if entry.Operation != "test_operation" {
		t.Errorf("Chained entry.Operation = %q, want %q", entry.Operation, "test_operation")
	}
	if entry.ThreadID != 123 {
		t.Errorf("Chained entry.ThreadID = %d, want %d", entry.ThreadID, 123)
	}
	if entry.Error != "test error" {
		t.Errorf("Chained entry.Error = %q, want %q", entry.Error, "test error")
	}
	if entry.Fields["key1"] != "value1" {
		t.Errorf("Chained entry.Fields[key1] = %v, want %v", entry.Fields["key1"], "value1")
	}
	if entry.Fields["key2"] != "value2" {
		t.Errorf("Chained entry.Fields[key2] = %v, want %v", entry.Fields["key2"], "value2")
	}
	if entry.Fields["key3"] != 42 {
		t.Errorf("Chained entry.Fields[key3] = %v, want %v", entry.Fields["key3"], 42)
	}
}

func TestLogEntry_FieldsInitialization(t *testing.T) {
	entry := &LogEntry{
		Level:   INFO,
		Message: "test",
	}

	// Fields should be nil initially
	if entry.Fields != nil {
		t.Error("LogEntry.Fields should be nil initially")
	}

	// WithField should initialize Fields
	entry.WithField("key", "value")
	if entry.Fields == nil {
		t.Error("WithField should initialize Fields map")
	}

	// Test WithFields with nil Fields
	entry2 := &LogEntry{
		Level:   INFO,
		Message: "test",
	}
	entry2.WithFields(NewLogField("key", "value"))
	if entry2.Fields == nil {
		t.Error("WithFields should initialize Fields map")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}

// Test LogFormatter implementations

func TestJSONFormatter(t *testing.T) {
	formatter := NewJSONFormatter()

	entry := NewLogEntry(INFO, "test message").
		WithOperation("test_op").
		WithThreadID(123).
		WithField("key1", "value1").
		WithField("key2", 42)

	output, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("JSONFormatter.Format() failed: %v", err)
	}

	// Should be valid JSON ending with newline
	if !strings.HasSuffix(output, "\n") {
		t.Error("JSON output should end with newline")
	}

	// Parse back to verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify key fields are present
	if parsed["level"] != "INFO" {
		t.Errorf("Expected level 'INFO', got %v", parsed["level"])
	}
	if parsed["message"] != "test message" {
		t.Errorf("Expected message 'test message', got %v", parsed["message"])
	}
	if parsed["operation"] != "test_op" {
		t.Errorf("Expected operation 'test_op', got %v", parsed["operation"])
	}
}

func TestTextFormatter(t *testing.T) {
	formatter := NewTextFormatter()

	entry := NewLogEntry(ERROR, "error occurred").
		WithOperation("critical_op").
		WithThreadID(456).
		WithError(errors.New("something went wrong")).
		WithField("attempts", 3).
		WithField("duration", "1.5s")

	output, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("TextFormatter.Format() failed: %v", err)
	}

	// Should end with newline
	if !strings.HasSuffix(output, "\n") {
		t.Error("Text output should end with newline")
	}

	// Should contain key elements
	if !strings.Contains(output, "ERROR") {
		t.Error("Output should contain log level")
	}
	if !strings.Contains(output, "error occurred") {
		t.Error("Output should contain message")
	}
	if !strings.Contains(output, "operation=critical_op") {
		t.Error("Output should contain operation")
	}
	if !strings.Contains(output, "thread=456") {
		t.Error("Output should contain thread ID")
	}
	if !strings.Contains(output, "error=something went wrong") {
		t.Error("Output should contain error message")
	}
	if !strings.Contains(output, "attempts=3") {
		t.Error("Output should contain field attempts")
	}
	if !strings.Contains(output, "duration=1.5s") {
		t.Error("Output should contain field duration")
	}
}

func TestTextFormatterCustomTimestamp(t *testing.T) {
	formatter := &TextFormatter{
		TimestampFormat: "15:04:05",
		IncludeFields:   false,
	}

	entry := NewLogEntry(DEBUG, "debug message").
		WithField("hidden", "value")

	output, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("TextFormatter.Format() failed: %v", err)
	}

	// Should use custom timestamp format (HH:MM:SS only)
	lines := strings.Split(strings.TrimSpace(output), " ")
	if len(lines) < 2 {
		t.Fatal("Output should contain timestamp and level")
	}

	timestamp := strings.Trim(lines[0], "[]")
	if len(timestamp) != 8 { // HH:MM:SS format
		t.Errorf("Expected timestamp format HH:MM:SS, got %s", timestamp)
	}

	// Should not include fields when IncludeFields is false
	if strings.Contains(output, "hidden=value") {
		t.Error("Output should not contain fields when IncludeFields is false")
	}
}

func TestStructuredFormatter(t *testing.T) {
	formatter := NewStructuredFormatter()

	entry := NewLogEntry(WARN, "warning message").
		WithOperation("warn_op").
		WithThreadID(789).
		WithField("count", 10).
		WithField("name", "test")

	output, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("StructuredFormatter.Format() failed: %v", err)
	}

	// Should end with newline
	if !strings.HasSuffix(output, "\n") {
		t.Error("Structured output should end with newline")
	}

	// Should contain key-value pairs
	if !strings.Contains(output, "level=WARN") {
		t.Error("Output should contain level=WARN")
	}
	if !strings.Contains(output, `message="warning message"`) {
		t.Error("Output should contain quoted message")
	}
	if !strings.Contains(output, `operation="warn_op"`) {
		t.Error("Output should contain quoted operation")
	}
	if !strings.Contains(output, "thread_id=789") {
		t.Error("Output should contain thread_id")
	}
	if !strings.Contains(output, "count=10") {
		t.Error("Output should contain count field")
	}
	if !strings.Contains(output, `name="test"`) {
		t.Error("Output should contain quoted name field")
	}
}

func TestStructuredFormatterCustomSeparators(t *testing.T) {
	formatter := &StructuredFormatter{
		TimestampFormat:   "2006-01-02",
		KeyValueSeparator: ":",
		FieldSeparator:    " | ",
	}

	entry := NewLogEntry(INFO, "test").
		WithField("key", "value")

	output, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("StructuredFormatter.Format() failed: %v", err)
	}

	// Should use custom separators
	if !strings.Contains(output, "level:INFO") {
		t.Error("Output should use custom key-value separator")
	}
	if !strings.Contains(output, " | ") {
		t.Error("Output should use custom field separator")
	}

	// Should use custom timestamp format
	if !strings.Contains(output, "timestamp:") {
		t.Error("Output should contain timestamp with custom separator")
	}
}

func TestGetFormatterForFormat(t *testing.T) {
	tests := []struct {
		format   LogFormat
		expected string
	}{
		{JSON, "*logging.JSONFormatter"},
		{TEXT, "*logging.TextFormatter"},
		{STRUCTURED, "*logging.StructuredFormatter"},
		{LogFormat(999), "*logging.TextFormatter"}, // Unknown format defaults to TEXT
	}

	for _, tt := range tests {
		t.Run(tt.format.String(), func(t *testing.T) {
			formatter := GetFormatterForFormat(tt.format)
			if formatter == nil {
				t.Fatal("GetFormatterForFormat returned nil")
			}

			// Check type using reflection
			actualType := fmt.Sprintf("%T", formatter)
			if actualType != tt.expected {
				t.Errorf("Expected formatter type %s, got %s", tt.expected, actualType)
			}
		})
	}
}

func TestFormatterErrorHandling(t *testing.T) {
	// Test with entry that might cause JSON marshaling issues
	entry := NewLogEntry(INFO, "test")

	// Add a field that could cause issues (though in practice this should work fine)
	entry.WithField("valid", "value")

	formatters := []LogFormatter{
		NewJSONFormatter(),
		NewTextFormatter(),
		NewStructuredFormatter(),
	}

	for i, formatter := range formatters {
		t.Run(fmt.Sprintf("formatter_%d", i), func(t *testing.T) {
			output, err := formatter.Format(entry)
			if err != nil {
				t.Errorf("Formatter should handle normal entries without error: %v", err)
			}
			if output == "" {
				t.Error("Formatter should produce non-empty output")
			}
		})
	}
}
func TestLogLevel_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name            string
		level           LogLevel
		expected        string
		shouldRoundtrip bool
	}{
		{"ERROR level", ERROR, `"ERROR"`, true},
		{"WARN level", WARN, `"WARN"`, true},
		{"INFO level", INFO, `"INFO"`, true},
		{"DEBUG level", DEBUG, `"DEBUG"`, true},
		{"Unknown level", LogLevel(999), `"UNKNOWN"`, false}, // Can't roundtrip unknown levels
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.level)
			if err != nil {
				t.Fatalf("json.Marshal() failed: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("json.Marshal() = %s, want %s", string(data), tt.expected)
			}

			// Test unmarshaling only for valid levels
			if tt.shouldRoundtrip {
				var level LogLevel
				err = json.Unmarshal(data, &level)
				if err != nil {
					t.Fatalf("json.Unmarshal() failed: %v", err)
				}

				if level != tt.level {
					t.Errorf("json.Unmarshal() = %v, want %v", level, tt.level)
				}
			}
		})
	}
}

func TestLogLevel_JSONUnmarshalingErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"Invalid JSON", `invalid`},
		{"Invalid level", `"INVALID_LEVEL"`},
		{"Number instead of string", `123`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var level LogLevel
			err := json.Unmarshal([]byte(tt.data), &level)
			if err == nil {
				t.Errorf("json.Unmarshal(%s) should have failed", tt.data)
			}
		})
	}
}
