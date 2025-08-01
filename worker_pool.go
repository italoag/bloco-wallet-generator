package main

import (
	"context"
	"sync"
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
	// Reset the WaitGroup in case this is a restart
	wp.wg = sync.WaitGroup{}
	wp.wg.Add(wp.numWorkers)

	// Create and start workers
	wp.workers = make([]*Worker, 0, wp.numWorkers)
	for i := 0; i < wp.numWorkers; i++ {
		worker := NewWorker(i, wp.workChan, wp.resultChan, wp.statsChan, wp.shutdownChan, &wp.wg)
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
	// Check if already shutdown
	select {
	case <-wp.shutdownChan:
		// Already shutdown
		return
	default:
	}

	// Create a channel to signal when shutdown is complete
	done := make(chan struct{})

	// Signal all workers to shut down
	close(wp.shutdownChan)

	// Start draining channels to prevent workers from blocking
	go func() {
		for {
			select {
			case <-wp.resultChan:
				// Drain result channel
			case <-wp.statsChan:
				// Drain stats channel
			case <-time.After(100 * time.Millisecond):
				// If no messages for 100ms, assume channels are drained
				return
			}
		}
	}()

	// Wait for all workers to finish with timeout
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	// Wait with timeout to prevent deadlock
	select {
	case <-done:
		// Workers finished normally
	case <-time.After(5 * time.Second):
		// Timeout - some workers might be stuck
		// We'll continue anyway
	}
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
	// Use background context for backward compatibility
	ctx := context.Background()
	return wp.GenerateWalletWithContext(ctx, prefix, suffix, isChecksum, statsManager)
}

// GenerateWalletWithContext generates a wallet using the worker pool with context cancellation support
func (wp *WorkerPool) GenerateWalletWithContext(ctx context.Context, prefix, suffix string, isChecksum bool, statsManager *StatsManager) (*Wallet, int64) {
	// Start collecting stats
	wp.CollectStats(statsManager)

	// Calculate appropriate batch size based on difficulty
	batchSize := calculateOptimalBatchSize(prefix, suffix, isChecksum)

	// Channel to signal when a match is found
	matchFound := make(chan struct{})
	resultChan := make(chan WorkResult, wp.numWorkers)
	var foundWallet *Wallet
	var totalAttempts int64

	// Start distributing work to workers
	go func() {
		// Distribute initial work to all workers
		wp.DistributeWork(prefix, suffix, isChecksum, batchSize)

		// Continue distributing work until a match is found or shutdown
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-wp.shutdownChan:
				return
			case <-matchFound:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Check if we need to distribute more work
				// Only distribute more work if the channel isn't full
				if len(wp.workChan) < wp.numWorkers {
					wp.DistributeWork(prefix, suffix, isChecksum, batchSize)
				}
			}
		}
	}()

	// Forward results from wp.resultChan to our local resultChan
	go func() {
		defer close(resultChan)
		for {
			select {
			case <-wp.shutdownChan:
				return
			case <-matchFound:
				return
			case <-ctx.Done():
				return
			case result, ok := <-wp.resultChan:
				if !ok {
					return
				}
				select {
				case resultChan <- result:
					// Result forwarded
				case <-matchFound:
					return
				case <-wp.shutdownChan:
					return
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Result processing goroutine
	go func() {
		for {
			select {
			case <-wp.shutdownChan:
				return
			case <-ctx.Done():
				return
			case result, ok := <-resultChan:
				if !ok {
					return
				}

				if result.Found {
					foundWallet = result.Wallet
					totalAttempts = statsManager.GetTotalAttempts()
					// Signal that a match was found
					close(matchFound)
					return
				}
				// Continue processing non-matching results
			}
		}
	}()

	// Wait for a match to be found, timeout, or context cancellation
	select {
	case <-matchFound:
		// Match found, continue to shutdown
	case <-ctx.Done():
		// Context cancelled
		close(matchFound)
	case <-time.After(24 * time.Hour):
		// Timeout after 24 hours (should never happen in practice)
		close(matchFound)
	}

	// Shutdown the worker pool
	wp.Shutdown()

	return foundWallet, totalAttempts
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
