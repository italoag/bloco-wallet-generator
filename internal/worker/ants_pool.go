package worker

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"

	"bloco-eth/internal/crypto"
	"bloco-eth/internal/validation"
	"bloco-eth/pkg/errors"
	"bloco-eth/pkg/wallet"
)

// AntsWorkerPool uses ants goroutine pool for efficient wallet generation
type AntsWorkerPool struct {
	pool           *ants.Pool
	poolManager    *crypto.PoolManager
	validator      *validation.AddressValidator
	statsCollector *StatsCollector
	statsChan      chan WorkerStats
	
	// Atomic counters for thread-safe statistics
	totalAttempts  int64
	totalGenerated int64
	startTime      time.Time
	
	// Configuration
	poolSize       int
	batchSize      int
	
	// Synchronization
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// GenerationTask represents a wallet generation task
type GenerationTask struct {
	criteria     wallet.GenerationCriteria
	resultChan   chan<- *wallet.GenerationResult
	errorChan    chan<- error
	workerID     int
	batchSize    int
	pool         *AntsWorkerPool
}

// NewAntsWorkerPool creates a new ants-based worker pool
func NewAntsWorkerPool(poolSize int, poolManager *crypto.PoolManager, validator *validation.AddressValidator) (*AntsWorkerPool, error) {
	if poolSize <= 0 {
		poolSize = runtime.NumCPU()
	}

	// Create stats channel
	statsChan := make(chan WorkerStats, 100) // Buffered channel to prevent blocking

	awp := &AntsWorkerPool{
		poolManager:    poolManager,
		validator:      validator,
		statsCollector: NewStatsCollector(),
		statsChan:      statsChan,
		poolSize:       poolSize,
		batchSize:      1000, // Larger batch size for better efficiency
		startTime:      time.Now(),
	}

	awp.ctx, awp.cancel = context.WithCancel(context.Background())

	// Create ants pool with optimized configuration
	pool, err := ants.NewPool(poolSize, ants.WithOptions(ants.Options{
		ExpiryDuration:   time.Minute * 10, // Keep workers alive for 10 minutes
		PreAlloc:         true,              // Pre-allocate worker queue
		MaxBlockingTasks: poolSize * 2,      // Allow some queuing
		Nonblocking:      false,             // Block when pool is full
		PanicHandler: func(i interface{}) {
			// Log panic but don't crash the application
			if err, ok := i.(error); ok {
				// Handle panic gracefully
				_ = errors.NewWorkerError("panic", err.Error())
			}
		},
	}))
	
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeWorker, "create_ants_pool", "failed to create ants pool")
	}

	awp.pool = pool

	// Start statistics collection
	go awp.statsCollector.Start(statsChan, awp.ctx)

	return awp, nil
}

// GenerateWalletWithContext generates a single wallet with the given criteria
func (awp *AntsWorkerPool) GenerateWalletWithContext(ctx context.Context, criteria wallet.GenerationCriteria) (*wallet.GenerationResult, error) {
	resultChan := make(chan *wallet.GenerationResult, 1)
	errorChan := make(chan error, 1)

	task := &GenerationTask{
		criteria:   criteria,
		resultChan: resultChan,
		errorChan:  errorChan,
		workerID:   0, // Single generation doesn't need worker ID
		batchSize:  awp.batchSize,
		pool:       awp,
	}

	// Submit task to ants pool
	err := awp.pool.Submit(task.Execute)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeWorker, "submit_task", "failed to submit generation task")
	}

	// Wait for result or context cancellation
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Execute runs the wallet generation task
func (task *GenerationTask) Execute() {
	startTime := time.Now()
	addressGen := crypto.NewAddressGenerator(task.pool.poolManager)
	
	// Set validation strategy based on criteria
	if task.criteria.IsChecksum {
		checksumValidator := crypto.NewChecksumValidator(task.pool.poolManager)
		task.pool.validator.SetStrategy(validation.NewOptimizedStrategy(checksumValidator, true))
	} else {
		checksumValidator := crypto.NewChecksumValidator(task.pool.poolManager)
		task.pool.validator.SetStrategy(validation.NewOptimizedStrategy(checksumValidator, false))
	}

	var attempts int64 = 0

	// Process in efficient batches
	for {
		select {
		case <-task.pool.ctx.Done():
			task.errorChan <- task.pool.ctx.Err()
			return
		default:
		}

		// Generate batch of attempts
		batchAttempts := int64(0)
		for i := 0; i < task.batchSize; i++ {
			attempts++
			batchAttempts++

			// Generate wallet
			generatedWallet, err := addressGen.GenerateWallet()
			if err != nil {
				task.errorChan <- errors.WrapError(err, errors.ErrorTypeGeneration, "generate_wallet", "failed to generate wallet")
				return
			}

			// Validate address against criteria
			isValid, err := task.pool.validator.ValidateWithCriteria(generatedWallet.Address, task.criteria)
			if err != nil {
				// Continue on validation errors
				continue
			}

			if isValid {
				// Found a match!
				result := &wallet.GenerationResult{
					Wallet:   generatedWallet,
					Attempts: attempts,
					Duration: time.Since(startTime),
					WorkerID: task.workerID,
				}

				// Update statistics atomically
				atomic.AddInt64(&task.pool.totalAttempts, attempts)
				atomic.AddInt64(&task.pool.totalGenerated, 1)

				task.resultChan <- result
				return
			}
		}

		// Update statistics less frequently (every batch) for better performance
		atomic.AddInt64(&task.pool.totalAttempts, batchAttempts)
		
		// Send periodic stats update (non-blocking)
		if batchAttempts > 0 {
			workerStats := WorkerStats{
				WorkerID:    task.workerID,
				Attempts:    batchAttempts,
				Speed:       float64(batchAttempts) / time.Since(startTime).Seconds(),
				LastUpdate:  time.Now(),
				IsHealthy:   true,
				ErrorCount:  0,
			}
			
			select {
			case task.pool.statsChan <- workerStats:
			default:
				// Skip if channel is full to prevent blocking
			}
		}
	}
}

// GetStatsCollector returns the statistics collector
func (awp *AntsWorkerPool) GetStatsCollector() *StatsCollector {
	// Force update the stats collector with current statistics
	currentStats := awp.GetStats()
	
	// Force update the collector's internal state directly
	awp.statsCollector.mu.Lock()
	awp.statsCollector.aggregatedStats = AggregatedStats{
		TotalAttempts:    currentStats.TotalAttempts,
		TotalSpeed:       currentStats.TotalSpeed,
		AverageSpeed:     currentStats.AverageSpeed,
		PeakSpeed:        currentStats.PeakSpeed,
		ActiveWorkers:    currentStats.ActiveWorkers,
		HealthyWorkers:   currentStats.HealthyWorkers,
		TotalErrors:      currentStats.TotalErrors,
		ElapsedTime:      currentStats.ElapsedTime,
		LastUpdate:       currentStats.LastUpdate,
		ThreadEfficiency: currentStats.ThreadEfficiency,
		ThreadBalance:    currentStats.ThreadBalance,
		SpeedVariance:    currentStats.SpeedVariance,
	}
	awp.statsCollector.mu.Unlock()
	
	return awp.statsCollector
}

// GetStats returns current statistics
func (awp *AntsWorkerPool) GetStats() AggregatedStats {
	totalAttempts := atomic.LoadInt64(&awp.totalAttempts)
	elapsed := time.Since(awp.startTime)
	
	var speed float64
	if elapsed.Seconds() > 0 {
		speed = float64(totalAttempts) / elapsed.Seconds()
	}

	return AggregatedStats{
		TotalAttempts:    totalAttempts,
		TotalSpeed:       speed,
		AverageSpeed:     speed,
		PeakSpeed:        speed,
		ActiveWorkers:    awp.pool.Running(),
		HealthyWorkers:   awp.pool.Running(),
		TotalErrors:      0,
		ElapsedTime:      elapsed,
		LastUpdate:       time.Now(),
		ThreadEfficiency: float64(awp.pool.Running()) / float64(awp.poolSize),
		ThreadBalance:    1.0, // Ants manages load balancing
		SpeedVariance:    0.0,
	}
}

// Shutdown gracefully shuts down the ants pool
func (awp *AntsWorkerPool) Shutdown() error {
	awp.cancel()
	awp.pool.Release()
	return nil
}

// Start starts the worker pool (compatibility with existing interface)
func (awp *AntsWorkerPool) Start() error {
	// Ants pool is already started when created
	return nil
}