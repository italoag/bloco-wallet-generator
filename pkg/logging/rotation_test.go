package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestFileRotation_SizeBasedRotation(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "log_rotation_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	logFile := filepath.Join(tempDir, "test.log")

	// Create logger with small file size limit for testing
	config := &LogConfig{
		Enabled:     true,
		Level:       INFO,
		Format:      TEXT,
		OutputFile:  logFile,
		MaxFileSize: 100, // Very small size to trigger rotation quickly
		MaxFiles:    3,
		BufferSize:  0, // Disable buffering for predictable testing
	}

	logger, err := NewSecureLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Write enough log entries to trigger rotation
	for i := 0; i < 10; i++ {
		err := logger.Info(fmt.Sprintf("Test log message number %d with some additional content to make it longer", i))
		if err != nil {
			t.Errorf("Failed to write log entry %d: %v", i, err)
		}
	}

	// Flush to ensure all writes are complete
	if err := logger.Flush(); err != nil {
		t.Errorf("Failed to flush logger: %v", err)
	}

	// Check that rotation occurred
	files, err := filepath.Glob(logFile + "*")
	if err != nil {
		t.Fatalf("Failed to list log files: %v", err)
	}

	if len(files) < 2 {
		t.Errorf("Expected at least 2 files after rotation, got %d: %v", len(files), files)
	}

	// Verify that rotated files exist
	expectedFiles := []string{
		logFile,
		logFile + ".1",
	}

	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("Expected rotated file %s does not exist", expectedFile)
		}
	}
}

func TestFileRotation_MaxFilesLimit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "log_rotation_limit_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	logFile := filepath.Join(tempDir, "test.log")

	config := &LogConfig{
		Enabled:     true,
		Level:       INFO,
		Format:      TEXT,
		OutputFile:  logFile,
		MaxFileSize: 50, // Very small to trigger multiple rotations
		MaxFiles:    2,  // Keep only 2 old files
		BufferSize:  0,
	}

	logger, err := NewSecureLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Write many log entries to trigger multiple rotations
	for i := 0; i < 20; i++ {
		err := logger.Info(fmt.Sprintf("Log entry %d with enough content to trigger rotation", i))
		if err != nil {
			t.Errorf("Failed to write log entry %d: %v", i, err)
		}

		// Force flush after each write to ensure rotation happens
		if err := logger.Flush(); err != nil {
			t.Errorf("Failed to flush logger: %v", err)
		}
	}

	// Check that we don't exceed the max files limit
	files, err := filepath.Glob(logFile + "*")
	if err != nil {
		t.Fatalf("Failed to list log files: %v", err)
	}

	// Should have at most MaxFiles + 1 (current file + MaxFiles old files)
	maxExpectedFiles := config.MaxFiles + 1
	if len(files) > maxExpectedFiles {
		t.Errorf("Expected at most %d files, got %d: %v", maxExpectedFiles, len(files), files)
	}

	// Verify that files beyond the limit were removed
	for i := config.MaxFiles + 1; i <= 10; i++ {
		oldFile := fmt.Sprintf("%s.%d", logFile, i)
		if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
			t.Errorf("Old file %s should have been removed but still exists", oldFile)
		}
	}
}

func TestAsyncBuffering_BasicOperation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "log_buffer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	logFile := filepath.Join(tempDir, "buffered.log")

	config := &LogConfig{
		Enabled:     true,
		Level:       INFO,
		Format:      TEXT,
		OutputFile:  logFile,
		MaxFileSize: 1024 * 1024, // Large size to avoid rotation during test
		MaxFiles:    5,
		BufferSize:  100, // Enable buffering
	}

	logger, err := NewSecureLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Write several log entries
	testMessages := []string{
		"First buffered message",
		"Second buffered message",
		"Third buffered message",
	}

	for _, msg := range testMessages {
		if err := logger.Info(msg); err != nil {
			t.Errorf("Failed to write log message: %v", err)
		}
	}

	// Close logger to ensure all buffered entries are written
	if err := logger.Close(); err != nil {
		t.Errorf("Failed to close logger: %v", err)
	}

	// Read the log file and verify all messages were written
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	for _, msg := range testMessages {
		if !strings.Contains(logContent, msg) {
			t.Errorf("Expected message '%s' not found in log file", msg)
		}
	}
}

func TestAsyncBuffering_ConcurrentWrites(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "log_concurrent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	logFile := filepath.Join(tempDir, "concurrent.log")

	config := &LogConfig{
		Enabled:     true,
		Level:       INFO,
		Format:      TEXT,
		OutputFile:  logFile,
		MaxFileSize: 1024 * 1024,
		MaxFiles:    5,
		BufferSize:  1000, // Large buffer for concurrent writes
	}

	logger, err := NewSecureLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Perform concurrent writes
	const numGoroutines = 10
	const messagesPerGoroutine = 50
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				msg := fmt.Sprintf("Goroutine %d message %d", goroutineID, j)
				if err := logger.Info(msg); err != nil {
					t.Errorf("Failed to write log message from goroutine %d: %v", goroutineID, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Close logger to ensure all buffered entries are written
	if err := logger.Close(); err != nil {
		t.Errorf("Failed to close logger: %v", err)
	}

	// Verify that all messages were written
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	lines := strings.Split(strings.TrimSpace(logContent), "\n")

	expectedLines := numGoroutines * messagesPerGoroutine
	if len(lines) != expectedLines {
		t.Errorf("Expected %d log lines, got %d", expectedLines, len(lines))
	}
}

func TestBufferOverflow_FallbackToSync(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "log_overflow_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	logFile := filepath.Join(tempDir, "overflow.log")

	config := &LogConfig{
		Enabled:     true,
		Level:       INFO,
		Format:      TEXT,
		OutputFile:  logFile,
		MaxFileSize: 1024 * 1024,
		MaxFiles:    5,
		BufferSize:  5, // Very small buffer to test overflow
	}

	logger, err := NewSecureLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Write more messages than buffer can hold rapidly
	const numMessages = 20
	for i := 0; i < numMessages; i++ {
		msg := fmt.Sprintf("Overflow test message %d", i)
		if err := logger.Info(msg); err != nil {
			t.Errorf("Failed to write log message %d: %v", i, err)
		}
	}

	// Close logger to ensure all entries are written
	if err := logger.Close(); err != nil {
		t.Errorf("Failed to close logger: %v", err)
	}

	// Verify that all messages were written (either buffered or sync fallback)
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	lines := strings.Split(strings.TrimSpace(logContent), "\n")

	if len(lines) != numMessages {
		t.Errorf("Expected %d log lines, got %d", numMessages, len(lines))
	}
}

func TestFlushMethod(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "log_flush_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	logFile := filepath.Join(tempDir, "flush.log")

	config := &LogConfig{
		Enabled:     true,
		Level:       INFO,
		Format:      TEXT,
		OutputFile:  logFile,
		MaxFileSize: 1024 * 1024,
		MaxFiles:    5,
		BufferSize:  100,
	}

	logger, err := NewSecureLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Write a message
	testMessage := "Test flush message"
	if err := logger.Info(testMessage); err != nil {
		t.Errorf("Failed to write log message: %v", err)
	}

	// Flush explicitly
	if err := logger.Flush(); err != nil {
		t.Errorf("Failed to flush logger: %v", err)
	}

	// Verify message was written to file
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), testMessage) {
		t.Errorf("Expected message '%s' not found in log file after flush", testMessage)
	}
}

func TestRotationWithBuffering(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "log_rotation_buffer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	logFile := filepath.Join(tempDir, "rotation_buffer.log")

	config := &LogConfig{
		Enabled:     true,
		Level:       INFO,
		Format:      TEXT,
		OutputFile:  logFile,
		MaxFileSize: 200, // Small size to trigger rotation
		MaxFiles:    3,
		BufferSize:  50, // Enable buffering
	}

	logger, err := NewSecureLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Write enough messages to trigger rotation
	for i := 0; i < 15; i++ {
		msg := fmt.Sprintf("Rotation with buffering test message number %d", i)
		if err := logger.Info(msg); err != nil {
			t.Errorf("Failed to write log message %d: %v", i, err)
		}
	}

	// Close logger to ensure all buffered entries are written
	if err := logger.Close(); err != nil {
		t.Errorf("Failed to close logger: %v", err)
	}

	// Verify rotation occurred
	files, err := filepath.Glob(logFile + "*")
	if err != nil {
		t.Fatalf("Failed to list log files: %v", err)
	}

	if len(files) < 2 {
		t.Errorf("Expected at least 2 files after rotation with buffering, got %d: %v", len(files), files)
	}
}

func TestCloseWithPendingBufferedEntries(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "log_close_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	logFile := filepath.Join(tempDir, "close_test.log")

	config := &LogConfig{
		Enabled:     true,
		Level:       INFO,
		Format:      TEXT,
		OutputFile:  logFile,
		MaxFileSize: 1024 * 1024,
		MaxFiles:    5,
		BufferSize:  1000,
	}

	logger, err := NewSecureLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Write messages rapidly
	testMessages := make([]string, 100)
	for i := 0; i < 100; i++ {
		testMessages[i] = fmt.Sprintf("Close test message %d", i)
		if err := logger.Info(testMessages[i]); err != nil {
			t.Errorf("Failed to write log message %d: %v", i, err)
		}
	}

	// Close immediately without explicit flush
	if err := logger.Close(); err != nil {
		t.Errorf("Failed to close logger: %v", err)
	}

	// Verify all messages were written despite immediate close
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	lines := strings.Split(strings.TrimSpace(logContent), "\n")

	if len(lines) != len(testMessages) {
		t.Errorf("Expected %d log lines after close, got %d", len(testMessages), len(lines))
	}
}

func TestRotationFileNumbering(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "log_numbering_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	logFile := filepath.Join(tempDir, "numbering.log")

	// Create some pre-existing rotated files to test proper numbering
	existingFiles := []string{
		logFile + ".1",
		logFile + ".2",
	}
	for _, file := range existingFiles {
		if err := os.WriteFile(file, []byte("existing content"), 0644); err != nil {
			t.Fatalf("Failed to create existing file %s: %v", file, err)
		}
	}

	config := &LogConfig{
		Enabled:     true,
		Level:       INFO,
		Format:      TEXT,
		OutputFile:  logFile,
		MaxFileSize: 50,
		MaxFiles:    5,
		BufferSize:  0,
	}

	logger, err := NewSecureLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Write enough to trigger rotation
	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf("Numbering test message %d with extra content", i)
		if err := logger.Info(msg); err != nil {
			t.Errorf("Failed to write log message %d: %v", i, err)
		}
		if err := logger.Flush(); err != nil {
			t.Errorf("Failed to flush logger: %v", err)
		}
	}

	// Verify proper file numbering after rotation
	expectedFiles := []string{
		logFile,        // Current file
		logFile + ".1", // Most recent rotated
		logFile + ".2", // Older rotated
		logFile + ".3", // Even older rotated
	}

	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist after rotation", expectedFile)
		}
	}
}
