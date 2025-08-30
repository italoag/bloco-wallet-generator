package logging

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	// ERROR represents critical errors that prevent operation
	ERROR LogLevel = iota
	// WARN represents warnings about non-ideal but non-critical situations
	WARN
	// INFO represents important operational information (default level)
	INFO
	// DEBUG represents detailed information for debugging (without sensitive data)
	DEBUG
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case ERROR:
		return "ERROR"
	case WARN:
		return "WARN"
	case INFO:
		return "INFO"
	case DEBUG:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

// MarshalJSON implements json.Marshaler for LogLevel
func (l LogLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

// UnmarshalJSON implements json.Unmarshaler for LogLevel
func (l *LogLevel) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	level, err := ParseLogLevel(s)
	if err != nil {
		return err
	}

	*l = level
	return nil
}

// ParseLogLevel parses a string into a LogLevel
func ParseLogLevel(level string) (LogLevel, error) {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "ERROR":
		return ERROR, nil
	case "WARN", "WARNING":
		return WARN, nil
	case "INFO":
		return INFO, nil
	case "DEBUG":
		return DEBUG, nil
	default:
		return INFO, fmt.Errorf("invalid log level: %s", level)
	}
}

// LogFormat represents the output format for log entries
type LogFormat int

const (
	// JSON format for structured logging
	JSON LogFormat = iota
	// TEXT format for human-readable logging
	TEXT
	// STRUCTURED format for structured text logging
	STRUCTURED
)

// String returns the string representation of the log format
func (f LogFormat) String() string {
	switch f {
	case JSON:
		return "JSON"
	case TEXT:
		return "TEXT"
	case STRUCTURED:
		return "STRUCTURED"
	default:
		return "TEXT"
	}
}

// LogConfig holds configuration for the secure logger
type LogConfig struct {
	// Enabled controls whether logging is active
	Enabled bool
	// Level is the minimum log level to record
	Level LogLevel
	// Format specifies the output format
	Format LogFormat
	// OutputFile is the path to the log file (empty for stdout)
	OutputFile string
	// MaxFileSize is the maximum log file size in bytes before rotation
	MaxFileSize int64
	// MaxFiles is the maximum number of rotated files to keep
	MaxFiles int
	// BufferSize is the buffer size for async logging
	BufferSize int
}

// DefaultLogConfig returns a default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Enabled:     true,
		Level:       INFO,
		Format:      TEXT,
		OutputFile:  "",
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		MaxFiles:    5,
		BufferSize:  1000,
	}
}

// Validate checks if the configuration is valid
func (c *LogConfig) Validate() error {
	if c.MaxFileSize <= 0 {
		return fmt.Errorf("MaxFileSize must be positive, got %d", c.MaxFileSize)
	}
	if c.MaxFiles < 0 {
		return fmt.Errorf("MaxFiles must be non-negative, got %d", c.MaxFiles)
	}
	if c.BufferSize < 0 {
		return fmt.Errorf("BufferSize must be non-negative, got %d", c.BufferSize)
	}
	return nil
}

// LogField represents a structured field in a log entry
type LogField struct {
	Key   string
	Value interface{}
}

// NewLogField creates a new log field
func NewLogField(key string, value interface{}) LogField {
	return LogField{Key: key, Value: value}
}

// LogFormatter defines the interface for formatting log entries
type LogFormatter interface {
	// Format formats a log entry into a string representation
	Format(entry *LogEntry) (string, error)
}

// LogEntry represents a structured log entry
type LogEntry struct {
	// Timestamp when the log entry was created
	Timestamp time.Time `json:"timestamp"`
	// Level is the severity level of the log entry
	Level LogLevel `json:"level"`
	// Message is the main log message
	Message string `json:"message"`
	// Operation is the operation being performed (optional)
	Operation string `json:"operation,omitempty"`
	// ThreadID is the identifier of the thread (optional)
	ThreadID int `json:"thread_id,omitempty"`
	// Fields contains additional structured data
	Fields map[string]interface{} `json:"fields,omitempty"`
	// Error contains error information if applicable
	Error string `json:"error,omitempty"`
}

// NewLogEntry creates a new log entry with the given level and message
func NewLogEntry(level LogLevel, message string) *LogEntry {
	return &LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   message,
		Fields:    make(map[string]interface{}),
	}
}

// WithOperation adds operation information to the log entry
func (e *LogEntry) WithOperation(operation string) *LogEntry {
	e.Operation = operation
	return e
}

// WithThreadID adds thread ID information to the log entry
func (e *LogEntry) WithThreadID(threadID int) *LogEntry {
	e.ThreadID = threadID
	return e
}

// WithError adds error information to the log entry
func (e *LogEntry) WithError(err error) *LogEntry {
	if err != nil {
		e.Error = err.Error()
	}
	return e
}

// WithFields adds multiple fields to the log entry
func (e *LogEntry) WithFields(fields ...LogField) *LogEntry {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	for _, field := range fields {
		e.Fields[field.Key] = field.Value
	}
	return e
}

// WithField adds a single field to the log entry
func (e *LogEntry) WithField(key string, value interface{}) *LogEntry {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	e.Fields[key] = value
	return e
}

// JSONFormatter formats log entries as JSON
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() LogFormatter {
	return &JSONFormatter{}
}

// Format formats a log entry as JSON
func (f *JSONFormatter) Format(entry *LogEntry) (string, error) {
	data, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("failed to marshal log entry to JSON: %w", err)
	}
	return string(data) + "\n", nil
}

// TextFormatter formats log entries as human-readable text
type TextFormatter struct {
	// TimestampFormat specifies the format for timestamps (defaults to "2006-01-02 15:04:05.000")
	TimestampFormat string
	// IncludeFields controls whether to include structured fields in output
	IncludeFields bool
}

// NewTextFormatter creates a new text formatter with default settings
func NewTextFormatter() LogFormatter {
	return &TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05.000",
		IncludeFields:   true,
	}
}

// Format formats a log entry as human-readable text
func (f *TextFormatter) Format(entry *LogEntry) (string, error) {
	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = "2006-01-02 15:04:05.000"
	}

	timestamp := entry.Timestamp.Format(timestampFormat)
	msg := fmt.Sprintf("[%s] %s: %s", timestamp, entry.Level.String(), entry.Message)

	if entry.Operation != "" {
		msg += fmt.Sprintf(" (operation=%s)", entry.Operation)
	}

	if entry.ThreadID != 0 {
		msg += fmt.Sprintf(" (thread=%d)", entry.ThreadID)
	}

	if entry.Error != "" {
		msg += fmt.Sprintf(" error=%s", entry.Error)
	}

	if f.IncludeFields && len(entry.Fields) > 0 {
		for key, value := range entry.Fields {
			msg += fmt.Sprintf(" %s=%v", key, value)
		}
	}

	return msg + "\n", nil
}

// StructuredFormatter formats log entries as structured key-value pairs
type StructuredFormatter struct {
	// TimestampFormat specifies the format for timestamps (defaults to RFC3339Nano)
	TimestampFormat string
	// KeyValueSeparator is the separator between key and value (defaults to "=")
	KeyValueSeparator string
	// FieldSeparator is the separator between fields (defaults to " ")
	FieldSeparator string
}

// NewStructuredFormatter creates a new structured formatter with default settings
func NewStructuredFormatter() LogFormatter {
	return &StructuredFormatter{
		TimestampFormat:   time.RFC3339Nano,
		KeyValueSeparator: "=",
		FieldSeparator:    " ",
	}
}

// Format formats a log entry as structured key-value pairs
func (f *StructuredFormatter) Format(entry *LogEntry) (string, error) {
	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = time.RFC3339Nano
	}

	kvSep := f.KeyValueSeparator
	if kvSep == "" {
		kvSep = "="
	}

	fieldSep := f.FieldSeparator
	if fieldSep == "" {
		fieldSep = " "
	}

	timestamp := entry.Timestamp.Format(timestampFormat)

	parts := []string{
		fmt.Sprintf("timestamp%s%s", kvSep, timestamp),
		fmt.Sprintf("level%s%s", kvSep, entry.Level.String()),
		fmt.Sprintf("message%s%q", kvSep, entry.Message),
	}

	if entry.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation%s%q", kvSep, entry.Operation))
	}

	if entry.ThreadID != 0 {
		parts = append(parts, fmt.Sprintf("thread_id%s%d", kvSep, entry.ThreadID))
	}

	if entry.Error != "" {
		parts = append(parts, fmt.Sprintf("error%s%q", kvSep, entry.Error))
	}

	for key, value := range entry.Fields {
		if str, ok := value.(string); ok {
			parts = append(parts, fmt.Sprintf("%s%s%q", key, kvSep, str))
		} else {
			parts = append(parts, fmt.Sprintf("%s%s%v", key, kvSep, value))
		}
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += fieldSep
		}
		result += part
	}

	return result + "\n", nil
}

// GetFormatterForFormat returns the appropriate formatter for the given format
func GetFormatterForFormat(format LogFormat) LogFormatter {
	switch format {
	case JSON:
		return NewJSONFormatter()
	case STRUCTURED:
		return NewStructuredFormatter()
	default: // TEXT
		return NewTextFormatter()
	}
}
