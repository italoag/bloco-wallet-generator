package main

import (
	"os"
	"runtime"
	"testing"
)

func TestDetectCPUCount(t *testing.T) {
	cpuCount := detectCPUCount()
	runtimeCPU := runtime.NumCPU()

	if cpuCount != runtimeCPU {
		t.Errorf("detectCPUCount() = %d, want %d", cpuCount, runtimeCPU)
	}
}

// TestValidateThreads tests the thread validation logic
// Note: This test doesn't test the os.Exit() paths to avoid terminating the test process
func TestValidateThreads(t *testing.T) {
	// Save original threads value to restore later
	originalThreads := threads
	defer func() {
		threads = originalThreads
	}()

	// Test auto-detection (threads = 0)
	threads = 0
	expectedThreads := runtime.NumCPU()

	// Temporarily redirect stdout to avoid printing during tests
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	validateThreads()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if threads != expectedThreads {
		t.Errorf("validateThreads() with threads=0 should set threads to %d, got %d", expectedThreads, threads)
	}

	// Test valid thread count
	threads = 2
	validateThreads()
	if threads != 2 {
		t.Errorf("validateThreads() with threads=2 should keep threads=2, got %d", threads)
	}

	// Test reasonable but high thread count (shouldn't exit but should warn)
	cpuCount := runtime.NumCPU()
	threads = cpuCount + 1 // Just above CPU count but below 2x
	validateThreads()
	if threads != cpuCount+1 {
		t.Errorf("validateThreads() with threads=%d should keep that value, got %d", cpuCount+1, threads)
	}
}
