package worker

import (
	"context"
	"sync"
	"time"

	"bloco-eth/internal/crypto"
	"bloco-eth/internal/validation"
	"bloco-eth/pkg/errors"
	"bloco-eth/pkg/wallet"
)

// Worker represents an individual worker thread
type Worker struct {
	id            int
	workChan      <-chan WorkItem
	resultChan    chan<- WorkResult
	statsChan     chan<- WorkerStats
	healthChan    chan<- WorkerHealth
	shutdownChan  <-chan struct{}
	wg            *sync.WaitGroup
	addressGen    *crypto.AddressGenerator
	validator     *validation.AddressValidator
	localStats    WorkerStats
	health        WorkerHealth
	startTime     time.Time
	lastStatsTime time.Time
	mu            sync.RWMutex
}

// NewWorker creates a new worker instance
func NewWorker(
	id int,
	workChan <-chan WorkItem,
	resultChan chan<- WorkResult,
	statsChan chan<- WorkerStats,
	healthChan chan<- WorkerHealth,
	shutdownChan <-chan struct{},
	wg *sync.WaitGroup,
	poolManager *crypto.PoolManager,
	validator *validation.AddressValidator,
) *Worker {
	now := time.Now()

	return &Worker{
		id:            id,
		workChan:      workChan,
		resultChan:    resultChan,
		statsChan:     statsChan,
		healthChan:    healthChan,
		shutdownChan:  shutdownChan,
		wg:            wg,
		addressGen:    crypto.NewAddressGenerator(poolManager),
		validator:     validator,
		startTime:     now,
		lastStatsTime: now,
		localStats: WorkerStats{
			WorkerID:   id,
			Attempts:   0,
			Speed:      0,
			LastUpdate: now,
			IsHealthy:  true,
			ErrorCount: 0,
		},
		health: WorkerHealth{
			WorkerID:      id,
			IsHealthy:     true,
			LastHeartbeat: now,
			ErrorCount:    0,
			Uptime:        0,
		},
	}
}

// Start begins the worker's processing loop
func (w *Worker) Start(ctx context.Context) error {
	go func() {
		defer func() {
			if w.wg != nil {
				w.wg.Done()
			}
		}()

		// Send initial health status
		w.sendHealthUpdate()

		// Main processing loop
		for {
			select {
			case workItem, ok := <-w.workChan:
				if !ok {
					// Work channel closed, exit
					return
				}
				w.processWorkItem(workItem)

			case <-w.shutdownChan:
				// Shutdown signal received
				w.sendFinalStats()
				return

			case <-ctx.Done():
				// Context cancelled
				w.sendFinalStats()
				return
			}
		}
	}()

	return nil
}

// processWorkItem processes a single work item
func (w *Worker) processWorkItem(item WorkItem) {
	startTime := time.Now()

	// Set validation strategy based on criteria
	if item.Criteria.IsChecksum {
		// Use optimized checksum strategy for performance
		checksumValidator := crypto.NewChecksumValidator(w.addressGen.GetPoolManager())
		w.validator.SetStrategy(validation.NewOptimizedStrategy(checksumValidator, true))
	} else {
		// Use optimized case-insensitive strategy
		checksumValidator := crypto.NewChecksumValidator(w.addressGen.GetPoolManager())
		w.validator.SetStrategy(validation.NewOptimizedStrategy(checksumValidator, false))
	}

	var attempts int64 = 0
	batchSize := item.BatchSize
	if batchSize <= 0 {
		batchSize = 1000 // Larger batch size for better performance
	}

	// Process batch of wallet generation attempts
	for i := 0; i < batchSize; i++ {
		// Check for shutdown signal
		select {
		case <-w.shutdownChan:
			w.sendWorkResult(item, nil, attempts, false)
			return
		default:
		}

		attempts++

		// Generate wallet
		generatedWallet, err := w.addressGen.GenerateWallet()
		if err != nil {
			w.handleError(err)
			w.sendWorkResult(item, nil, attempts, false)
			return
		}

		// Validate address against criteria
		isValid, err := w.validator.ValidateWithCriteria(generatedWallet.Address, item.Criteria)
		if err != nil {
			w.handleError(err)
			continue
		}

		if isValid {
			// Found a match!
			result := &wallet.GenerationResult{
				Wallet:   generatedWallet,
				Attempts: attempts,
				Duration: time.Since(startTime),
				WorkerID: w.id,
			}
			w.sendWorkResult(item, result, attempts, true)
			w.updateStats(attempts)
			return
		}
	}

	// No match found in this batch - send result and continue
	w.sendWorkResult(item, nil, attempts, false)
	w.updateStats(attempts)
}

// sendWorkResult sends a work result to the result channel
func (w *Worker) sendWorkResult(item WorkItem, result *wallet.GenerationResult, attempts int64, found bool) {
	workResult := WorkResult{
		Result:   result,
		WorkerID: w.id,
		ItemID:   item.ID,
		Found:    found,
	}

	select {
	case w.resultChan <- workResult:
		// Result sent successfully
	case <-w.shutdownChan:
		// Shutdown signal received, don't block
	default:
		// Channel full, log warning but don't block
		w.handleError(errors.NewWorkerError("send_result", "result channel full"))
	}
}

// updateStats updates worker statistics
func (w *Worker) updateStats(additionalAttempts int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.localStats.Attempts += additionalAttempts
	now := time.Now()
	elapsed := now.Sub(w.startTime)

	if elapsed.Seconds() > 0 {
		w.localStats.Speed = float64(w.localStats.Attempts) / elapsed.Seconds()
	}

	w.localStats.LastUpdate = now

	// Send stats update if enough time has passed (reduced frequency)
	if now.Sub(w.lastStatsTime) >= 500*time.Millisecond {
		w.sendStatsUpdate()
		w.lastStatsTime = now
	}

	// Update health status
	w.health.LastHeartbeat = now
	w.health.Uptime = elapsed
}

// sendStatsUpdate sends a statistics update
func (w *Worker) sendStatsUpdate() {
	select {
	case w.statsChan <- w.localStats:
		// Stats sent successfully
	default:
		// Channel full, skip this update
	}
}

// sendHealthUpdate sends a health status update
func (w *Worker) sendHealthUpdate() {
	w.mu.RLock()
	health := w.health
	w.mu.RUnlock()

	select {
	case w.healthChan <- health:
		// Health update sent successfully
	default:
		// Channel full, skip this update
	}
}

// sendFinalStats sends final statistics before shutdown
func (w *Worker) sendFinalStats() {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Update final stats
	now := time.Now()
	elapsed := now.Sub(w.startTime)

	if elapsed.Seconds() > 0 {
		w.localStats.Speed = float64(w.localStats.Attempts) / elapsed.Seconds()
	}

	w.localStats.LastUpdate = now
	w.health.LastHeartbeat = now
	w.health.Uptime = elapsed

	// Send final updates
	select {
	case w.statsChan <- w.localStats:
	default:
	}

	select {
	case w.healthChan <- w.health:
	default:
	}
}

// handleError handles worker errors
func (w *Worker) handleError(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.localStats.ErrorCount++
	w.localStats.LastError = err.Error()

	w.health.ErrorCount++
	w.health.LastError = err.Error()

	// Mark as unhealthy if too many errors
	if w.health.ErrorCount > 10 {
		w.health.IsHealthy = false
		w.localStats.IsHealthy = false
	}
}

// GetHealth returns the current health status of the worker
func (w *Worker) GetHealth() WorkerHealth {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Update uptime
	health := w.health
	health.Uptime = time.Since(w.startTime)
	return health
}

// GetStats returns the current statistics of the worker
func (w *Worker) GetStats() WorkerStats {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.localStats
}

// GetID returns the worker ID
func (w *Worker) GetID() int {
	return w.id
}

// IsHealthy returns whether the worker is healthy
func (w *Worker) IsHealthy() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.health.IsHealthy
}

// Reset resets worker statistics (useful for benchmarking)
func (w *Worker) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	w.startTime = now
	w.lastStatsTime = now

	w.localStats = WorkerStats{
		WorkerID:   w.id,
		Attempts:   0,
		Speed:      0,
		LastUpdate: now,
		IsHealthy:  true,
		ErrorCount: 0,
	}

	w.health = WorkerHealth{
		WorkerID:      w.id,
		IsHealthy:     true,
		LastHeartbeat: now,
		ErrorCount:    0,
		Uptime:        0,
	}
}

// GetUptime returns how long the worker has been running
func (w *Worker) GetUptime() time.Duration {
	return time.Since(w.startTime)
}

// GetTotalAttempts returns the total number of attempts made by this worker
func (w *Worker) GetTotalAttempts() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.localStats.Attempts
}

// GetCurrentSpeed returns the current generation speed of the worker
func (w *Worker) GetCurrentSpeed() float64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.localStats.Speed
}
