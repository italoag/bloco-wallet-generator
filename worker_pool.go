package main

import (
	"time"
)

// NewWorkerPool creates a new worker pool with the specified number of workers
func NewWorkerPool(numWorkers int) *WorkerPool {
	if numWorkers <= 0 {
		numWorkers = detectCPUCount()
	}

	return &WorkerPool{
		numWorkers:   numWorkers,
		workers:      make([]*Worker, 0, numWorkers),
		workChan:     make(chan WorkItem, numWorkers),
		resultChan:   make(chan WorkResult, numWorkers),
		statsChan:    make(chan WorkerStats, numWorkers*2),
		shutdownChan: make(chan struct{}),
	}
}

// Start initializes and starts all workers in the pool
func (wp *WorkerPool) Start() {
	wp.wg.Add(wp.numWorkers)

	// Create and start workers
	for i := 0; i < wp.numWorkers; i++ {
		worker := NewWorker(i, wp.workChan, wp.resultChan, wp.statsChan, wp.shutdownChan)
		wp.workers = append(wp.workers, worker)
		worker.Start()
	}
}

// Submit sends a work item to the worker pool
func (wp *WorkerPool) Submit(item WorkItem) {
	wp.workChan <- item
}

// SubmitBatch sends multiple work items to the worker pool
func (wp *WorkerPool) SubmitBatch(items []WorkItem) {
	for _, item := range items {
		wp.workChan <- item
	}
}

// WaitForResult waits for a successful result or timeout
func (wp *WorkerPool) WaitForResult(timeout time.Duration) (WorkResult, bool) {
	select {
	case result := <-wp.resultChan:
		return result, true
	case <-time.After(timeout):
		return WorkResult{}, false
	}
}

// CollectStats collects statistics from all workers
func (wp *WorkerPool) CollectStats(statsManager *StatsManager) {
	// Start a goroutine to collect stats
	go func() {
		for {
			select {
			case stats := <-wp.statsChan:
				statsManager.UpdateWorkerStats(stats)
			case <-wp.shutdownChan:
				return
			}
		}
	}()
}

// Shutdown gracefully shuts down all workers
func (wp *WorkerPool) Shutdown() {
	// Signal all workers to shut down
	close(wp.shutdownChan)

	// Wait for all workers to finish
	wp.wg.Wait()
}

// GetNumWorkers returns the number of workers in the pool
func (wp *WorkerPool) GetNumWorkers() int {
	return wp.numWorkers
}

// DistributeWork splits work into batches and distributes to workers
func (wp *WorkerPool) DistributeWork(prefix, suffix string, isChecksum bool, batchSize int) {
	// Create work items for each worker
	for i := 0; i < wp.numWorkers; i++ {
		item := WorkItem{
			Prefix:     prefix,
			Suffix:     suffix,
			IsChecksum: isChecksum,
			BatchSize:  batchSize,
		}
		wp.Submit(item)
	}
}

// GenerateWallet generates a wallet using the worker pool
func (wp *WorkerPool) GenerateWallet(prefix, suffix string, isChecksum bool, statsManager *StatsManager) (*Wallet, int64) {
	// Start collecting stats
	wp.CollectStats(statsManager)

	// Calculate appropriate batch size based on difficulty
	batchSize := calculateOptimalBatchSize(prefix, suffix, isChecksum)

	// Start distributing work to workers
	go func() {
		for {
			wp.DistributeWork(prefix, suffix, isChecksum, batchSize)

			// Check if we should stop
			select {
			case <-wp.shutdownChan:
				return
			default:
				// Continue distributing work
			}
		}
	}()

	// Wait for a result
	result, found := wp.WaitForResult(24 * time.Hour) // Long timeout, effectively infinite

	// Shutdown the worker pool
	wp.Shutdown()

	if found && result.Found {
		return result.Wallet, statsManager.GetTotalAttempts()
	}

	return nil, statsManager.GetTotalAttempts()
}

// calculateOptimalBatchSize determines the best batch size based on pattern difficulty
func calculateOptimalBatchSize(prefix, suffix string, isChecksum bool) int {
	difficulty := computeDifficulty(prefix, suffix, isChecksum)

	// Adjust batch size based on difficulty
	// For very easy patterns, use smaller batches for more frequent updates
	// For difficult patterns, use larger batches for better performance
	if difficulty < 1000 {
		return 100
	} else if difficulty < 10000 {
		return 500
	} else if difficulty < 100000 {
		return 1000
	} else if difficulty < 1000000 {
		return 5000
	} else {
		return 10000
	}
}
