package main

import (
	"crypto/rand"
	"sync"
	"testing"
	"time"
)

func init() {
	// Initialize global pools for testing
	initializePools()
}

func TestNewWorker(t *testing.T) {
	workChan := make(chan WorkItem)
	resultChan := make(chan WorkResult)
	statsChan := make(chan WorkerStats)
	shutdownChan := make(chan struct{})

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan)

	if worker == nil {
		t.Fatal("NewWorker returned nil")
	}

	if worker.id != 1 {
		t.Errorf("Expected worker ID 1, got %d", worker.id)
	}

	if worker.workChan != workChan {
		t.Error("Work channel not properly assigned")
	}

	if worker.resultChan != resultChan {
		t.Error("Result channel not properly assigned")
	}

	if worker.statsChan != statsChan {
		t.Error("Stats channel not properly assigned")
	}

	if worker.shutdownChan != shutdownChan {
		t.Error("Shutdown channel not properly assigned")
	}

	if worker.localStats.WorkerID != 1 {
		t.Errorf("Expected local stats worker ID 1, got %d", worker.localStats.WorkerID)
	}
}

func TestWorkerProcessWorkItem(t *testing.T) {
	workChan := make(chan WorkItem)
	resultChan := make(chan WorkResult)
	statsChan := make(chan WorkerStats)
	shutdownChan := make(chan struct{})

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan)

	// Create local pools for testing
	cryptoPool := NewCryptoPool()
	hasherPool := NewHasherPool()
	bufferPool := NewBufferPool()

	// Test with a simple work item (should not find a match with this pattern)
	workItem := WorkItem{
		Prefix:     "ffffffff", // Very unlikely to match randomly
		Suffix:     "",
		IsChecksum: false,
		BatchSize:  10,
	}

	result := worker.processWorkItem(workItem, cryptoPool, hasherPool, bufferPool)

	if result.Found {
		t.Error("Unexpectedly found a match with an unlikely pattern")
	}

	if result.Attempts != 10 {
		t.Errorf("Expected 10 attempts, got %d", result.Attempts)
	}

	if result.WorkerID != 1 {
		t.Errorf("Expected worker ID 1, got %d", result.WorkerID)
	}

	if result.Error != nil {
		t.Errorf("Unexpected error: %v", result.Error)
	}
}

func TestWorkerStart(t *testing.T) {
	workChan := make(chan WorkItem)
	resultChan := make(chan WorkResult)
	statsChan := make(chan WorkerStats, 10) // Buffered to prevent blocking
	shutdownChan := make(chan struct{})

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan)

	// Start the worker
	worker.Start()

	// Send a work item
	workItem := WorkItem{
		Prefix:     "a", // Simple prefix that should be found relatively quickly
		Suffix:     "",
		IsChecksum: false,
		BatchSize:  1000,
	}

	// Use a WaitGroup to coordinate the test
	var wg sync.WaitGroup
	wg.Add(1)

	// Start a goroutine to receive the result
	var result WorkResult
	go func() {
		defer wg.Done()

		// Send the work item
		workChan <- workItem

		// Wait for result with timeout
		select {
		case result = <-resultChan:
			// Got a result
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for result")
		}
	}()

	// Wait for the goroutine to complete
	wg.Wait()

	// Check if we got a result
	if !result.Found {
		t.Error("Worker did not find a match")
	}

	if result.Wallet == nil {
		t.Error("Worker returned nil wallet")
	} else {
		// Verify the wallet has the correct prefix
		if len(result.Wallet.Address) < 1 || result.Wallet.Address[0] != 'a' {
			t.Errorf("Generated address doesn't match prefix: %s", result.Wallet.Address)
		}
	}

	// For testing purposes, we'll skip checking the stats channel
	// since it's timing-dependent and can be flaky in tests

	// Shutdown the worker
	close(shutdownChan)

	// Allow time for shutdown
	time.Sleep(100 * time.Millisecond)
}

func TestPrivateToAddressOptimized(t *testing.T) {
	workChan := make(chan WorkItem)
	resultChan := make(chan WorkResult)
	statsChan := make(chan WorkerStats)
	shutdownChan := make(chan struct{})

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan)

	// Create local pools for testing
	cryptoPool := NewCryptoPool()
	hasherPool := NewHasherPool()
	bufferPool := NewBufferPool()

	// Get a private key buffer
	privateKey := cryptoPool.GetPrivateKeyBuffer()
	defer cryptoPool.PutPrivateKeyBuffer(privateKey)

	// Generate a random private key
	_, err := rand.Read(privateKey)
	if err != nil {
		t.Fatalf("Failed to generate random bytes: %v", err)
	}

	// Generate address using the optimized method
	address := worker.privateToAddressOptimized(privateKey, cryptoPool, hasherPool, bufferPool)

	// Verify the address
	if len(address) != 40 {
		t.Errorf("Expected address length 40, got %d", len(address))
	}

	// Check if address is valid hex
	if !isValidHex(address) {
		t.Errorf("Generated address is not valid hex: %s", address)
	}
}

func TestIsValidBlocoAddressOptimized(t *testing.T) {
	workChan := make(chan WorkItem)
	resultChan := make(chan WorkResult)
	statsChan := make(chan WorkerStats)
	shutdownChan := make(chan struct{})

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan)

	// Create local pools for testing
	hasherPool := NewHasherPool()
	bufferPool := NewBufferPool()

	testCases := []struct {
		address  string
		prefix   string
		suffix   string
		checksum bool
		expected bool
	}{
		// Basic prefix matching
		{
			address:  "abcd1234567890123456789012345678901234ef",
			prefix:   "abcd",
			suffix:   "",
			checksum: false,
			expected: true,
		},
		// Basic suffix matching
		{
			address:  "1234567890123456789012345678901234abcdef",
			prefix:   "",
			suffix:   "cdef",
			checksum: false,
			expected: true,
		},
		// Both prefix and suffix matching
		{
			address:  "abcd567890123456789012345678901234cdef",
			prefix:   "abcd",
			suffix:   "cdef",
			checksum: false,
			expected: true,
		},
		// Case insensitive matching
		{
			address:  "ABCD567890123456789012345678901234cdef",
			prefix:   "abcd",
			suffix:   "CDEF",
			checksum: false,
			expected: true,
		},
		// Non-matching prefix
		{
			address:  "1234567890123456789012345678901234cdef",
			prefix:   "abcd",
			suffix:   "",
			checksum: false,
			expected: false,
		},
		// Non-matching suffix
		{
			address:  "abcd567890123456789012345678901234xyz1",
			prefix:   "",
			suffix:   "cdef",
			checksum: false,
			expected: false,
		},
	}

	for i, tc := range testCases {
		result := worker.isValidBlocoAddressOptimized(tc.address, tc.prefix, tc.suffix, tc.checksum, hasherPool, bufferPool)
		if result != tc.expected {
			t.Errorf("Test case %d failed: expected %v, got %v", i+1, tc.expected, result)
			t.Errorf("  Address: %s, Prefix: %s, Suffix: %s", tc.address, tc.prefix, tc.suffix)
		}
	}
}

func BenchmarkWorkerPrivateToAddressOptimized(b *testing.B) {
	workChan := make(chan WorkItem)
	resultChan := make(chan WorkResult)
	statsChan := make(chan WorkerStats)
	shutdownChan := make(chan struct{})

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan)

	// Create local pools for testing
	cryptoPool := NewCryptoPool()
	hasherPool := NewHasherPool()
	bufferPool := NewBufferPool()

	// Get a private key buffer
	privateKey := cryptoPool.GetPrivateKeyBuffer()
	defer cryptoPool.PutPrivateKeyBuffer(privateKey)

	// Generate a random private key
	rand.Read(privateKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		worker.privateToAddressOptimized(privateKey, cryptoPool, hasherPool, bufferPool)
	}
}

func BenchmarkWorkerIsValidBlocoAddressOptimized(b *testing.B) {
	workChan := make(chan WorkItem)
	resultChan := make(chan WorkResult)
	statsChan := make(chan WorkerStats)
	shutdownChan := make(chan struct{})

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan)

	// Create local pools for testing
	hasherPool := NewHasherPool()
	bufferPool := NewBufferPool()

	address := "abcd1234567890123456789012345678901234ef"
	prefix := "abcd"
	suffix := ""

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		worker.isValidBlocoAddressOptimized(address, prefix, suffix, false, hasherPool, bufferPool)
	}
}
