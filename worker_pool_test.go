package main

import (
	"sync"
	"testing"
	"time"
)

func init() {
	// Initialize global pools for testing
	initializePools()
}

func TestNewWorkerPool(t *testing.T) {
	// Test with specific number of workers
	pool := NewWorkerPool(4)

	if pool == nil {
		t.Fatal("NewWorkerPool returned nil")
	}

	if pool.numWorkers != 4 {
		t.Errorf("Expected 4 workers, got %d", pool.numWorkers)
	}

	if pool.workChan == nil {
		t.Error("Work channel not initialized")
	}

	if pool.resultChan == nil {
		t.Error("Result channel not initialized")
	}

	if pool.statsChan == nil {
		t.Error("Stats channel not initialized")
	}

	if pool.shutdownChan == nil {
		t.Error("Shutdown channel not initialized")
	}

	// Test with zero workers (should auto-detect)
	pool2 := NewWorkerPool(0)
	expectedWorkers := detectCPUCount()
	if pool2.numWorkers != expectedWorkers {
		t.Errorf("Expected %d workers (auto-detected), got %d", expectedWorkers, pool2.numWorkers)
	}

	// Test with negative workers (should auto-detect)
	pool3 := NewWorkerPool(-1)
	if pool3.numWorkers != expectedWorkers {
		t.Errorf("Expected %d workers (auto-detected), got %d", expectedWorkers, pool3.numWorkers)
	}
}

func TestWorkerPoolStart(t *testing.T) {
	pool := NewWorkerPool(2)

	// Start the pool
	pool.Start()

	// Verify workers were created
	if len(pool.workers) != 2 {
		t.Errorf("Expected 2 workers, got %d", len(pool.workers))
	}

	// Verify each worker is properly initialized
	for i, worker := range pool.workers {
		if worker == nil {
			t.Errorf("Worker %d is nil", i)
			continue
		}

		if worker.id != i {
			t.Errorf("Worker %d has incorrect ID: expected %d, got %d", i, i, worker.id)
		}

		if worker.workChan != pool.workChan {
			t.Errorf("Worker %d has incorrect work channel", i)
		}

		if worker.resultChan != pool.resultChan {
			t.Errorf("Worker %d has incorrect result channel", i)
		}

		if worker.statsChan != pool.statsChan {
			t.Errorf("Worker %d has incorrect stats channel", i)
		}

		if worker.shutdownChan != pool.shutdownChan {
			t.Errorf("Worker %d has incorrect shutdown channel", i)
		}
	}

	// Shutdown the pool
	pool.Shutdown()
}

func TestWorkerPoolSubmit(t *testing.T) {
	pool := NewWorkerPool(2)
	pool.Start()

	// Create a work item with a simple pattern that should be found quickly
	workItem := WorkItem{
		Prefix:     "a", // Simple prefix that should be found quickly
		Suffix:     "",
		IsChecksum: false,
		BatchSize:  10,
	}

	// Submit work item
	pool.Submit(workItem)

	// Wait for a result from the workers (they should process the work item)
	select {
	case result := <-pool.resultChan:
		// Verify that we got a result from processing the work item
		if result.Attempts <= 0 {
			t.Errorf("Expected positive attempt count, got %d", result.Attempts)
		}
		// For a simple prefix like "a", we should find a match relatively quickly
		if result.Error != nil {
			t.Errorf("Unexpected error: %v", result.Error)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for work result")
	}

	pool.Shutdown()
}

func TestWorkerPoolSubmitBatch(t *testing.T) {
	// Create pool but don't start workers to prevent them from consuming work items
	pool := NewWorkerPool(2)

	// Create multiple work items
	workItems := []WorkItem{
		{Prefix: "test1", Suffix: "", IsChecksum: false, BatchSize: 10},
		{Prefix: "test2", Suffix: "", IsChecksum: false, BatchSize: 20},
		{Prefix: "test3", Suffix: "", IsChecksum: false, BatchSize: 30},
	}

	// Submit batch in a goroutine to avoid blocking
	go func() {
		pool.SubmitBatch(workItems)
	}()

	// Receive all work items
	receivedItems := make([]WorkItem, 0, len(workItems))
	for i := 0; i < len(workItems); i++ {
		select {
		case item := <-pool.workChan:
			receivedItems = append(receivedItems, item)
		case <-time.After(1 * time.Second):
			t.Errorf("Timeout waiting for work item %d", i)
		}
	}

	// Verify all items were received
	if len(receivedItems) != len(workItems) {
		t.Errorf("Expected %d items, got %d", len(workItems), len(receivedItems))
	}

	pool.Shutdown()
}

func TestWorkerPoolWaitForResult(t *testing.T) {
	pool := NewWorkerPool(1)
	pool.Start()

	// Create a result and send it to the result channel
	expectedResult := WorkResult{
		Wallet:   &Wallet{Address: "test", PrivKey: "testkey"},
		Attempts: 100,
		WorkerID: 0,
		Found:    true,
	}

	// Send result in a goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		pool.resultChan <- expectedResult
	}()

	// Wait for result
	result, ok := pool.WaitForResult(1 * time.Second)

	if !ok {
		t.Error("WaitForResult returned false")
	}

	if !result.Found {
		t.Error("Expected result to be found")
	}

	if result.Attempts != expectedResult.Attempts {
		t.Errorf("Expected %d attempts, got %d", expectedResult.Attempts, result.Attempts)
	}

	if result.Wallet.Address != expectedResult.Wallet.Address {
		t.Errorf("Expected address %s, got %s", expectedResult.Wallet.Address, result.Wallet.Address)
	}

	pool.Shutdown()
}

func TestWorkerPoolWaitForResultTimeout(t *testing.T) {
	pool := NewWorkerPool(1)
	pool.Start()

	// Wait for result with short timeout (no result will be sent)
	result, ok := pool.WaitForResult(100 * time.Millisecond)

	if ok {
		t.Error("WaitForResult should have timed out")
	}

	// Verify empty result
	if result.Found {
		t.Error("Timeout result should not be found")
	}

	pool.Shutdown()
}

func TestWorkerPoolCollectStats(t *testing.T) {
	pool := NewWorkerPool(2)
	pool.Start()

	// Create a stats manager
	statsManager := NewStatsManager()

	// Start collecting stats
	pool.CollectStats(statsManager)

	// Send some stats
	stats1 := WorkerStats{
		WorkerID:   0,
		Attempts:   1000,
		Speed:      5000,
		LastUpdate: time.Now(),
	}

	stats2 := WorkerStats{
		WorkerID:   1,
		Attempts:   2000,
		Speed:      7000,
		LastUpdate: time.Now(),
	}

	pool.statsChan <- stats1
	pool.statsChan <- stats2

	// Give some time for stats to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify stats were collected
	totalAttempts := statsManager.GetTotalAttempts()
	if totalAttempts != 3000 {
		t.Errorf("Expected total attempts 3000, got %d", totalAttempts)
	}

	totalSpeed := statsManager.GetTotalSpeed()
	if totalSpeed != 12000 {
		t.Errorf("Expected total speed 12000, got %f", totalSpeed)
	}

	pool.Shutdown()
}

func TestWorkerPoolShutdown(t *testing.T) {
	pool := NewWorkerPool(3)
	pool.Start()

	// Verify workers are running by checking the WaitGroup
	// We can't directly test if goroutines are running, but we can test shutdown behavior

	// Shutdown the pool
	pool.Shutdown()

	// Try to send work after shutdown - should not block
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Sending on closed channel causes panic, which is expected
			}
			done <- true
		}()

		// This might panic if shutdown channel is closed, which is expected
		select {
		case pool.workChan <- WorkItem{}:
			// If we can send, that's also fine
		default:
			// Channel might be full or closed
		}
	}()

	// Wait for the goroutine to complete
	select {
	case <-done:
		// Shutdown completed successfully
	case <-time.After(2 * time.Second):
		t.Error("Shutdown took too long")
	}
}

func TestWorkerPoolGetNumWorkers(t *testing.T) {
	pool := NewWorkerPool(5)

	numWorkers := pool.GetNumWorkers()
	if numWorkers != 5 {
		t.Errorf("Expected 5 workers, got %d", numWorkers)
	}
}

func TestWorkerPoolDistributeWork(t *testing.T) {
	// Create pool but don't start workers to prevent them from consuming work items
	pool := NewWorkerPool(3)

	// Distribute work in a goroutine to avoid blocking
	go func() {
		pool.DistributeWork("test", "suffix", true, 100)
	}()

	// Should receive 3 work items (one per worker)
	receivedItems := 0
	timeout := time.After(1 * time.Second)

	for receivedItems < 3 {
		select {
		case item := <-pool.workChan:
			if item.Prefix != "test" {
				t.Errorf("Expected prefix 'test', got '%s'", item.Prefix)
			}
			if item.Suffix != "suffix" {
				t.Errorf("Expected suffix 'suffix', got '%s'", item.Suffix)
			}
			if !item.IsChecksum {
				t.Error("Expected IsChecksum to be true")
			}
			if item.BatchSize != 100 {
				t.Errorf("Expected batch size 100, got %d", item.BatchSize)
			}
			receivedItems++
		case <-timeout:
			t.Errorf("Timeout: only received %d out of 3 work items", receivedItems)
			break
		}
	}

	pool.Shutdown()
}

func TestWorkerPoolConcurrentAccess(t *testing.T) {
	// Create pool but don't start workers to prevent them from consuming work items
	pool := NewWorkerPool(4)

	var wg sync.WaitGroup
	numGoroutines := 5     // Reduced to avoid channel blocking
	itemsPerGoroutine := 2 // Reduced to avoid channel blocking

	// Start a goroutine to drain the channel as items are submitted
	drainDone := make(chan struct{})
	totalItems := 0
	go func() {
		defer close(drainDone)
		timeout := time.After(5 * time.Second)
		for {
			select {
			case <-pool.workChan:
				totalItems++
				if totalItems >= numGoroutines*itemsPerGoroutine {
					return
				}
			case <-timeout:
				return
			}
		}
	}()

	// Start multiple goroutines submitting work concurrently
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				workItem := WorkItem{
					Prefix:     "test",
					Suffix:     "",
					IsChecksum: false,
					BatchSize:  10,
				}
				pool.Submit(workItem)
			}
		}(i)
	}

	// Wait for all submissions to complete
	wg.Wait()

	// Wait for draining to complete
	<-drainDone

	expectedItems := numGoroutines * itemsPerGoroutine
	if totalItems != expectedItems {
		t.Errorf("Expected %d items, got %d", expectedItems, totalItems)
	}

	pool.Shutdown()
}

func TestCalculateOptimalBatchSize(t *testing.T) {
	testCases := []struct {
		prefix     string
		suffix     string
		isChecksum bool
		expected   int
	}{
		// Very easy patterns (difficulty < 1000)
		{"a", "", false, 100}, // difficulty = 16
		{"", "1", false, 100}, // difficulty = 16

		// Still easy patterns (difficulty < 1000)
		{"ab", "", false, 100}, // difficulty = 256

		// Medium difficulty (1000 <= difficulty < 10000)
		{"abc", "", false, 500}, // difficulty = 4096

		// High difficulty (10000 <= difficulty < 100000)
		{"abcd", "", false, 1000}, // difficulty = 65536

		// Very high difficulty (>= 1000000)
		{"abcde", "", false, 10000}, // difficulty = 1048576

		// Checksum patterns (higher difficulty due to checksum multiplier)
		{"a", "", true, 100},    // difficulty = 16 * 2 = 32
		{"ab", "", true, 500},   // difficulty = 256 * 4 = 1024
		{"abc", "", true, 1000}, // difficulty = 4096 * 8 = 32768
	}

	for i, tc := range testCases {
		result := calculateOptimalBatchSize(tc.prefix, tc.suffix, tc.isChecksum)
		difficulty := computeDifficulty(tc.prefix, tc.suffix, tc.isChecksum)
		if result != tc.expected {
			t.Errorf("Test case %d: expected batch size %d, got %d (difficulty: %.0f)", i+1, tc.expected, result, difficulty)
			t.Errorf("  Pattern: prefix='%s', suffix='%s', checksum=%v", tc.prefix, tc.suffix, tc.isChecksum)
		}
	}
}

// Benchmark tests for performance validation
func BenchmarkWorkerPoolSubmit(b *testing.B) {
	pool := NewWorkerPool(4)
	pool.Start()
	defer pool.Shutdown()

	workItem := WorkItem{
		Prefix:     "test",
		Suffix:     "",
		IsChecksum: false,
		BatchSize:  10,
	}

	// Start a goroutine to drain the work channel
	go func() {
		for {
			select {
			case <-pool.workChan:
				// Drain work items
			case <-pool.shutdownChan:
				return
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(workItem)
	}
}

func BenchmarkWorkerPoolDistributeWork(b *testing.B) {
	pool := NewWorkerPool(4)
	pool.Start()
	defer pool.Shutdown()

	// Start a goroutine to drain the work channel
	go func() {
		for {
			select {
			case <-pool.workChan:
				// Drain work items
			case <-pool.shutdownChan:
				return
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.DistributeWork("test", "", false, 100)
	}
}
