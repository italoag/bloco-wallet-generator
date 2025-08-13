package worker

import (
	"context"
	"sync"
	"time"

	"bloco-eth/internal/config"
	"bloco-eth/internal/crypto"
	"bloco-eth/internal/validation"
	"bloco-eth/pkg/errors"
	"bloco-eth/pkg/wallet"
)

// ManagedWorkerPool provides a managed worker pool with health monitoring
type ManagedWorkerPool struct {
	config         config.WorkerConfig
	workers        []*Worker
	workChan       chan WorkItem
	resultChan     chan WorkResult
	statsChan      chan WorkerStats
	shutdownChan   chan struct{}
	healthChan     chan WorkerHealth
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	statsCollector *StatsCollector
	poolManager    *crypto.PoolManager
	validator      *validation.AddressValidator
	isRunning      bool
	mu             sync.RWMutex
}

// WorkItem represents a unit of work for the worker pool
type WorkItem struct {
	Criteria  wallet.GenerationCriteria `json:"criteria"`
	BatchSize int                       `json:"batch_size"`
	ID        string                    `json:"id"`
}

// WorkResult represents the result of processing a work item
type WorkResult struct {
	Result   *wallet.GenerationResult `json:"result"`
	WorkerID int                      `json:"worker_id"`
	ItemID   string                   `json:"item_id"`
	Found    bool                     `json:"found"`
}

// WorkerStats holds statistics for individual workers
type WorkerStats struct {
	WorkerID   int       `json:"worker_id"`
	Attempts   int64     `json:"attempts"`
	Speed      float64   `json:"speed"`
	LastUpdate time.Time `json:"last_update"`
	IsHealthy  bool      `json:"is_healthy"`
	ErrorCount int       `json:"error_count"`
	LastError  string    `json:"last_error,omitempty"`
}

// WorkerHealth represents health information for a worker
type WorkerHealth struct {
	WorkerID      int           `json:"worker_id"`
	IsHealthy     bool          `json:"is_healthy"`
	LastHeartbeat time.Time     `json:"last_heartbeat"`
	ErrorCount    int           `json:"error_count"`
	LastError     string        `json:"last_error,omitempty"`
	Uptime        time.Duration `json:"uptime"`
}

// NewManagedWorkerPool creates a new managed worker pool
func NewManagedWorkerPool(
	config config.WorkerConfig,
	poolManager *crypto.PoolManager,
	validator *validation.AddressValidator,
) *ManagedWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &ManagedWorkerPool{
		config:         config,
		workChan:       make(chan WorkItem, config.ThreadCount*10),     // Larger buffer
		resultChan:     make(chan WorkResult, config.ThreadCount*10),   // Larger buffer
		statsChan:      make(chan WorkerStats, config.ThreadCount*20),  // Larger buffer
		healthChan:     make(chan WorkerHealth, config.ThreadCount*10), // Larger buffer
		shutdownChan:   make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
		statsCollector: NewStatsCollector(),
		poolManager:    poolManager,
		validator:      validator,
		isRunning:      false,
	}
}

// Start initializes and starts all workers in the pool
func (mwp *ManagedWorkerPool) Start() error {
	mwp.mu.Lock()
	defer mwp.mu.Unlock()

	if mwp.isRunning {
		return errors.NewWorkerError("start_pool", "worker pool is already running")
	}

	// Reset the WaitGroup
	mwp.wg = sync.WaitGroup{}
	mwp.wg.Add(mwp.config.ThreadCount)

	// Create and start workers
	mwp.workers = make([]*Worker, 0, mwp.config.ThreadCount)
	for i := 0; i < mwp.config.ThreadCount; i++ {
		worker := NewWorker(
			i,
			mwp.workChan,
			mwp.resultChan,
			mwp.statsChan,
			mwp.healthChan,
			mwp.shutdownChan,
			&mwp.wg,
			mwp.poolManager,
			mwp.validator,
		)
		mwp.workers = append(mwp.workers, worker)

		if err := worker.Start(mwp.ctx); err != nil {
			return errors.WrapError(err, errors.ErrorTypeWorker,
				"start_pool", "failed to start worker")
		}
	}

	// Start health monitoring
	go mwp.healthMonitor()

	// Start statistics collection
	go mwp.statsCollector.Start(mwp.statsChan, mwp.ctx)

	mwp.isRunning = true
	return nil
}

// Submit sends a work item to the worker pool
func (mwp *ManagedWorkerPool) Submit(item WorkItem) error {
	mwp.mu.RLock()
	defer mwp.mu.RUnlock()

	if !mwp.isRunning {
		return errors.NewWorkerError("submit_work", "worker pool is not running")
	}

	select {
	case mwp.workChan <- item:
		return nil
	case <-mwp.ctx.Done():
		return errors.NewCancellationError("submit_work", "worker pool is shutting down")
	default:
		return errors.NewWorkerError("submit_work", "work queue is full")
	}
}

// SubmitBatch sends multiple work items to the worker pool
func (mwp *ManagedWorkerPool) SubmitBatch(items []WorkItem) error {
	for _, item := range items {
		if err := mwp.Submit(item); err != nil {
			return errors.WrapError(err, errors.ErrorTypeWorker,
				"submit_batch", "failed to submit work item")
		}
	}
	return nil
}

// WaitForResult waits for a result with timeout
func (mwp *ManagedWorkerPool) WaitForResult(timeout time.Duration) (*WorkResult, error) {
	select {
	case result := <-mwp.resultChan:
		return &result, nil
	case <-time.After(timeout):
		return nil, errors.NewTimeoutError("wait_for_result", timeout)
	case <-mwp.ctx.Done():
		return nil, errors.NewCancellationError("wait_for_result", "worker pool is shutting down")
	}
}

// GetResultChannel returns the result channel for direct access
func (mwp *ManagedWorkerPool) GetResultChannel() <-chan WorkResult {
	return mwp.resultChan
}

// GetStatsCollector returns the statistics collector
func (mwp *ManagedWorkerPool) GetStatsCollector() *StatsCollector {
	return mwp.statsCollector
}

// GetHealthStatus returns the current health status of all workers
func (mwp *ManagedWorkerPool) GetHealthStatus() []WorkerHealth {
	mwp.mu.RLock()
	defer mwp.mu.RUnlock()

	var healthStatus []WorkerHealth
	for _, worker := range mwp.workers {
		healthStatus = append(healthStatus, worker.GetHealth())
	}
	return healthStatus
}

// Shutdown gracefully shuts down all workers
func (mwp *ManagedWorkerPool) Shutdown() error {
	mwp.mu.Lock()
	defer mwp.mu.Unlock()

	if !mwp.isRunning {
		return nil // Already shutdown
	}

	// Cancel context to signal shutdown
	mwp.cancel()

	// Close shutdown channel
	close(mwp.shutdownChan)

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		mwp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Workers finished normally
	case <-time.After(mwp.config.ShutdownTimeout):
		// Timeout - force shutdown
		return errors.NewTimeoutError("shutdown_pool", mwp.config.ShutdownTimeout)
	}

	// Drain channels to prevent goroutine leaks
	mwp.drainChannels()

	mwp.isRunning = false
	return nil
}

// drainChannels drains all channels to prevent goroutine leaks
func (mwp *ManagedWorkerPool) drainChannels() {
	// Drain work channel
	for {
		select {
		case <-mwp.workChan:
		default:
			goto drainResults
		}
	}

drainResults:
	// Drain result channel
	for {
		select {
		case <-mwp.resultChan:
		default:
			goto drainStats
		}
	}

drainStats:
	// Drain stats channel
	for {
		select {
		case <-mwp.statsChan:
		default:
			goto drainHealth
		}
	}

drainHealth:
	// Drain health channel
	for {
		select {
		case <-mwp.healthChan:
		default:
			return
		}
	}
}

// healthMonitor monitors the health of all workers
func (mwp *ManagedWorkerPool) healthMonitor() {
	ticker := time.NewTicker(mwp.config.HealthCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mwp.checkWorkerHealth()
		case health := <-mwp.healthChan:
			mwp.processHealthUpdate(health)
		case <-mwp.ctx.Done():
			return
		}
	}
}

// checkWorkerHealth checks the health of all workers
func (mwp *ManagedWorkerPool) checkWorkerHealth() {
	mwp.mu.RLock()
	defer mwp.mu.RUnlock()

	for _, worker := range mwp.workers {
		health := worker.GetHealth()

		// Check if worker is responsive
		if time.Since(health.LastHeartbeat) > mwp.config.HealthCheckPeriod*2 {
			health.IsHealthy = false
			// In a production system, we might restart unhealthy workers here
		}
	}
}

// processHealthUpdate processes a health update from a worker
func (mwp *ManagedWorkerPool) processHealthUpdate(health WorkerHealth) {
	// Log health issues or take corrective action
	if !health.IsHealthy {
		// In a production system, we might implement worker recovery here
	}
}

// GetWorkerCount returns the number of workers in the pool
func (mwp *ManagedWorkerPool) GetWorkerCount() int {
	return mwp.config.ThreadCount
}

// IsRunning returns whether the worker pool is currently running
func (mwp *ManagedWorkerPool) IsRunning() bool {
	mwp.mu.RLock()
	defer mwp.mu.RUnlock()
	return mwp.isRunning
}

// GenerateWallet generates a single wallet using the worker pool
func (mwp *ManagedWorkerPool) GenerateWallet(criteria wallet.GenerationCriteria) (*wallet.GenerationResult, error) {
	return mwp.GenerateWalletWithContext(mwp.ctx, criteria)
}

// GenerateWalletWithContext generates a wallet with context cancellation support
func (mwp *ManagedWorkerPool) GenerateWalletWithContext(ctx context.Context, criteria wallet.GenerationCriteria) (*wallet.GenerationResult, error) {
	if err := criteria.Validate(); err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeValidation,
			"generate_wallet", "invalid generation criteria")
	}

	// Calculate optimal batch size
	batchSize := mwp.calculateOptimalBatchSize(criteria)
	
	// Create work item
	workItem := WorkItem{
		Criteria:  criteria,
		BatchSize: batchSize,
		ID:        generateWorkItemID(),
	}

	// Submit single work item - let workers handle the continuous generation
	if err := mwp.Submit(workItem); err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeWorker,
			"submit_work", "failed to submit generation task")
	}

	// Start a goroutine to keep feeding work items to workers
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Keep work channel fed
				select {
				case mwp.workChan <- workItem:
					// Work submitted successfully
				case <-ctx.Done():
					return
				default:
					// Channel full, workers are busy
				}
			}
		}
	}()

	// Simple result waiting
	for {
		select {
		case <-ctx.Done():
			return nil, errors.NewCancellationError("generate_wallet", "generation cancelled")
		case result := <-mwp.resultChan:
			if result.Found {
				return result.Result, nil
			}
			// Not found, continue waiting - the goroutine above keeps feeding work
		}
	}
}

// calculateOptimalBatchSize determines the optimal batch size based on criteria
func (mwp *ManagedWorkerPool) calculateOptimalBatchSize(criteria wallet.GenerationCriteria) int {
	// Calculate difficulty
	difficulty := calculateDifficulty(criteria.Prefix, criteria.Suffix, criteria.IsChecksum)

	// Adjust batch size based on difficulty
	if difficulty < 1000 {
		return mwp.config.MinBatchSize
	} else if difficulty < 10000 {
		return mwp.config.MinBatchSize * 2
	} else if difficulty < 100000 {
		return mwp.config.MinBatchSize * 5
	} else {
		return mwp.config.MaxBatchSize
	}
}

// Helper functions

// generateWorkItemID generates a unique ID for work items
func generateWorkItemID() string {
	return time.Now().Format("20060102150405.000000")
}

// calculateDifficulty calculates the difficulty of finding a matching address
func calculateDifficulty(prefix, suffix string, isChecksum bool) float64 {
	pattern := prefix + suffix
	baseDifficulty := 1.0
	for i := 0; i < len(pattern); i++ {
		baseDifficulty *= 16
	}

	if !isChecksum {
		return baseDifficulty
	}

	// Count letters (a-f, A-F) in the pattern for checksum calculation
	letterCount := 0
	for _, char := range pattern {
		if (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F') {
			letterCount++
		}
	}

	checksumMultiplier := 1.0
	for i := 0; i < letterCount; i++ {
		checksumMultiplier *= 2
	}

	return baseDifficulty * checksumMultiplier
}
