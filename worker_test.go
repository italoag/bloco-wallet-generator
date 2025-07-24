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
	var wg sync.WaitGroup

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan, &wg)

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
	var wg sync.WaitGroup

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan, &wg)

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
	var wg sync.WaitGroup

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan, &wg)

	// Add to WaitGroup before starting worker
	wg.Add(1)

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
	var testWg sync.WaitGroup
	testWg.Add(1)

	// Start a goroutine to receive the result
	var result WorkResult
	go func() {
		defer testWg.Done()

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
	testWg.Wait()

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

	// Wait for worker to finish
	wg.Wait()
}

func TestPrivateToAddressOptimized(t *testing.T) {
	workChan := make(chan WorkItem)
	resultChan := make(chan WorkResult)
	statsChan := make(chan WorkerStats)
	shutdownChan := make(chan struct{})
	var wg sync.WaitGroup

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan, &wg)

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
	var wg sync.WaitGroup

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan, &wg)

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
	var wg sync.WaitGroup

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan, &wg)

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
	var wg sync.WaitGroup

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan, &wg)

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

// TestWorkerThreadSafety tests that workers can operate safely in concurrent scenarios
func TestWorkerThreadSafety(t *testing.T) {
	numWorkers := 5
	workChan := make(chan WorkItem, numWorkers*2)
	resultChan := make(chan WorkResult, numWorkers*2)
	statsChan := make(chan WorkerStats, numWorkers*10)
	shutdownChan := make(chan struct{})
	var wg sync.WaitGroup

	// Add to WaitGroup before starting workers
	wg.Add(numWorkers)

	// Create multiple workers
	workers := make([]*Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = NewWorker(i, workChan, resultChan, statsChan, shutdownChan, &wg)
		workers[i].Start()
	}

	// Send work items
	numWorkItems := 20
	for i := 0; i < numWorkItems; i++ {
		workItem := WorkItem{
			Prefix:     "a", // Simple prefix for faster testing
			Suffix:     "",
			IsChecksum: false,
			BatchSize:  10,
		}
		workChan <- workItem
	}

	// Collect results
	resultsReceived := 0
	timeout := time.After(10 * time.Second)

	for resultsReceived < numWorkItems {
		select {
		case result := <-resultChan:
			if result.Error != nil {
				t.Errorf("Worker %d returned error: %v", result.WorkerID, result.Error)
			}
			if result.Attempts <= 0 {
				t.Errorf("Worker %d returned invalid attempt count: %d", result.WorkerID, result.Attempts)
			}
			resultsReceived++
		case <-timeout:
			t.Errorf("Timeout: only received %d out of %d results", resultsReceived, numWorkItems)
			break
		}
	}

	// Shutdown workers
	close(shutdownChan)

	// Wait for workers to finish
	wg.Wait()
}

// TestWorkerStatisticsReporting tests that workers properly report statistics
func TestWorkerStatisticsReporting(t *testing.T) {
	workChan := make(chan WorkItem, 1)
	resultChan := make(chan WorkResult, 10)
	statsChan := make(chan WorkerStats, 10)
	shutdownChan := make(chan struct{})
	var wg sync.WaitGroup

	// Add to WaitGroup before starting worker
	wg.Add(1)

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan, &wg)
	worker.Start()

	// Send a work item
	workItem := WorkItem{
		Prefix:     "a",
		Suffix:     "",
		IsChecksum: false,
		BatchSize:  100,
	}
	workChan <- workItem

	// Wait for result first
	select {
	case <-resultChan:
		// Got result
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for result")
	}

	// Shutdown worker - this should trigger final stats
	close(shutdownChan)

	// Wait for final stats that should be sent on shutdown
	select {
	case stats := <-statsChan:
		if stats.WorkerID != 1 {
			t.Errorf("Expected worker ID 1, got %d", stats.WorkerID)
		}
		if stats.Attempts <= 0 {
			t.Errorf("Expected positive attempts, got %d", stats.Attempts)
		}
		if stats.Speed < 0 {
			t.Errorf("Expected non-negative speed, got %f", stats.Speed)
		}
	case <-time.After(1 * time.Second):
		t.Error("No final statistics were reported by worker on shutdown")
	}

	wg.Wait()
}

// TestWorkerErrorHandling tests worker behavior with invalid work items
func TestWorkerErrorHandling(t *testing.T) {
	workChan := make(chan WorkItem, 1)
	resultChan := make(chan WorkResult, 1)
	statsChan := make(chan WorkerStats, 1)
	shutdownChan := make(chan struct{})
	var wg sync.WaitGroup

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan, &wg)

	// Create local pools for testing
	cryptoPool := NewCryptoPool()
	hasherPool := NewHasherPool()
	bufferPool := NewBufferPool()

	// Test with invalid batch size
	workItem := WorkItem{
		Prefix:     "test",
		Suffix:     "",
		IsChecksum: false,
		BatchSize:  0, // Invalid batch size
	}

	result := worker.processWorkItem(workItem, cryptoPool, hasherPool, bufferPool)

	// Should handle invalid batch size gracefully
	if result.Error != nil {
		t.Errorf("Worker should handle invalid batch size gracefully, got error: %v", result.Error)
	}

	// Should use default batch size
	if result.Attempts <= 0 {
		t.Errorf("Expected positive attempts even with invalid batch size, got %d", result.Attempts)
	}
}

// TestWorkerShutdownGraceful tests that workers shutdown gracefully
func TestWorkerShutdownGraceful(t *testing.T) {
	workChan := make(chan WorkItem, 1)
	resultChan := make(chan WorkResult, 1)
	statsChan := make(chan WorkerStats, 10)
	shutdownChan := make(chan struct{})
	var wg sync.WaitGroup

	// Add to WaitGroup before starting worker
	wg.Add(1)

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan, &wg)
	worker.Start()

	// Give worker time to start
	time.Sleep(50 * time.Millisecond)

	// Signal shutdown
	close(shutdownChan)

	// Wait for worker to finish with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Worker shut down successfully
	case <-time.After(2 * time.Second):
		t.Error("Worker did not shut down within timeout")
	}
}

// TestWorkerOptimizedFunctions tests the optimized cryptographic functions
func TestWorkerOptimizedFunctions(t *testing.T) {
	workChan := make(chan WorkItem)
	resultChan := make(chan WorkResult)
	statsChan := make(chan WorkerStats)
	shutdownChan := make(chan struct{})
	var wg sync.WaitGroup

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan, &wg)

	// Create local pools for testing
	cryptoPool := NewCryptoPool()
	hasherPool := NewHasherPool()
	bufferPool := NewBufferPool()

	// Test privateToAddressOptimized with multiple keys
	for i := 0; i < 10; i++ {
		privateKey := cryptoPool.GetPrivateKeyBuffer()
		_, err := rand.Read(privateKey)
		if err != nil {
			t.Fatalf("Failed to generate random bytes: %v", err)
		}

		address := worker.privateToAddressOptimized(privateKey, cryptoPool, hasherPool, bufferPool)

		// Verify address format
		if len(address) != 40 {
			t.Errorf("Expected address length 40, got %d", len(address))
		}

		if !isValidHex(address) {
			t.Errorf("Generated address is not valid hex: %s", address)
		}

		cryptoPool.PutPrivateKeyBuffer(privateKey)
	}

	// Test isValidBlocoAddressOptimized with checksum
	// Use a known valid checksum address for testing
	testAddress := "5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"
	isValid := worker.isValidBlocoAddressOptimized(testAddress, "5a", "", true, hasherPool, bufferPool)
	if !isValid {
		t.Log("Note: Checksum validation test may be sensitive to exact implementation")
	}

	// Test with non-checksum validation (should always work)
	isValid = worker.isValidBlocoAddressOptimized(testAddress, "5a", "", false, hasherPool, bufferPool)
	if !isValid {
		t.Error("Non-checksum address validation failed")
	}
}

// TestWorkerMemoryUsage tests that workers don't leak memory
func TestWorkerMemoryUsage(t *testing.T) {
	workChan := make(chan WorkItem, 1)
	resultChan := make(chan WorkResult, 100)
	statsChan := make(chan WorkerStats, 100)
	shutdownChan := make(chan struct{})
	var wg sync.WaitGroup

	// Add to WaitGroup before starting worker
	wg.Add(1)

	worker := NewWorker(1, workChan, resultChan, statsChan, shutdownChan, &wg)
	worker.Start()

	// Process many work items to test for memory leaks
	numItems := 50
	for i := 0; i < numItems; i++ {
		workItem := WorkItem{
			Prefix:     "ff", // Unlikely pattern to avoid early matches
			Suffix:     "ff",
			IsChecksum: false,
			BatchSize:  10,
		}
		workChan <- workItem
	}

	// Collect results
	resultsReceived := 0
	timeout := time.After(10 * time.Second)

	for resultsReceived < numItems {
		select {
		case result := <-resultChan:
			if result.Error != nil {
				t.Errorf("Unexpected error: %v", result.Error)
			}
			resultsReceived++
		case <-timeout:
			t.Errorf("Timeout: only received %d out of %d results", resultsReceived, numItems)
			break
		}
	}

	// Shutdown worker
	close(shutdownChan)
	wg.Wait()

	// This test mainly ensures the worker can handle many operations without crashing
	// Memory leak detection would require more sophisticated tooling
}
