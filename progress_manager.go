package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// AggregatedStats holds aggregated statistics from all worker threads
type AggregatedStats struct {
	TotalAttempts int64
	TotalSpeed    float64
	AverageSpeed  float64
	PeakSpeed     float64
	ActiveWorkers int
	Probability   float64
	EstimatedTime time.Duration
	LastUpdate    time.Time
}

// ProgressManager provides thread-safe progress display for multi-threaded operations
type ProgressManager struct {
	mu              sync.RWMutex
	statsManager    *StatsManager
	stats           *Statistics
	shutdownChan    chan struct{}
	updateInterval  time.Duration
	lastDisplayTime time.Time
	isActive        int32 // Use atomic for thread-safe access

	// Thread-safe progress aggregation
	aggregatedStats AggregatedStats
}

// NewProgressManager creates a new ProgressManager instance
func NewProgressManager(stats *Statistics, statsManager *StatsManager) *ProgressManager {
	return &ProgressManager{
		statsManager:    statsManager,
		stats:           stats,
		shutdownChan:    make(chan struct{}),
		updateInterval:  ProgressUpdateInterval,
		lastDisplayTime: time.Now(),
		isActive:        0, // 0 = false, 1 = true
		aggregatedStats: AggregatedStats{
			LastUpdate: time.Now(),
		},
	}
}

// Start begins the progress display loop
func (pm *ProgressManager) Start() {
	// Use atomic compare-and-swap to ensure only one goroutine starts
	if !atomic.CompareAndSwapInt32(&pm.isActive, 0, 1) {
		return // Already active
	}

	go pm.displayLoop()
}

// Stop terminates the progress display loop
func (pm *ProgressManager) Stop() {
	// Use atomic compare-and-swap to ensure only one goroutine stops
	if !atomic.CompareAndSwapInt32(&pm.isActive, 1, 0) {
		return // Already stopped
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	close(pm.shutdownChan)

	// Create a new shutdown channel for future use
	pm.shutdownChan = make(chan struct{})
}

// UpdateInterval changes the progress update interval
func (pm *ProgressManager) UpdateInterval(interval time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.updateInterval = interval
}

// ForceUpdate forces an immediate progress update
func (pm *ProgressManager) ForceUpdate() {
	if atomic.LoadInt32(&pm.isActive) == 0 {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.updateProgressDisplay()
}

// displayLoop runs the progress display loop with enhanced thread safety
func (pm *ProgressManager) displayLoop() {
	// Get the initial update interval and shutdown channel under lock
	pm.mu.RLock()
	updateInterval := pm.updateInterval
	shutdownChan := pm.shutdownChan
	pm.mu.RUnlock()

	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if still active using atomic load
			if atomic.LoadInt32(&pm.isActive) == 1 {
				pm.mu.Lock()
				pm.updateProgressDisplay()
				pm.mu.Unlock()
			}

		case <-shutdownChan:
			// Final update before exiting
			if atomic.LoadInt32(&pm.isActive) == 1 {
				pm.mu.Lock()
				pm.updateProgressDisplay()
				pm.mu.Unlock()
			}
			return
		}
	}
}

// updateProgressDisplay updates and displays the progress
func (pm *ProgressManager) updateProgressDisplay() {
	// Aggregate data from all worker threads in a thread-safe manner
	pm.aggregateWorkerData()

	// Update the original statistics object for compatibility
	pm.updateStatisticsObject()

	// Display the progress using aggregated data
	pm.displayProgress()
	pm.lastDisplayTime = time.Now()
}

// aggregateWorkerData safely aggregates data from all worker threads
func (pm *ProgressManager) aggregateWorkerData() {
	// Get current metrics from the stats manager (already thread-safe)
	metrics := pm.statsManager.GetMetrics()

	// Update aggregated stats in a thread-safe manner
	now := time.Now()

	// Create a local copy of aggregated stats to minimize lock time
	aggregated := AggregatedStats{
		TotalAttempts: metrics.TotalAttempts,
		TotalSpeed:    metrics.TotalThroughput,
		ActiveWorkers: metrics.WorkerCount,
		LastUpdate:    now,
	}

	// Calculate average speed based on elapsed time
	elapsed := metrics.ElapsedTime.Seconds()
	if elapsed > 0 {
		aggregated.AverageSpeed = float64(metrics.TotalAttempts) / elapsed
	}

	// Get peak speed from stats manager
	aggregated.PeakSpeed = pm.statsManager.GetPeakSpeed()

	// Calculate probability based on difficulty and attempts
	aggregated.Probability = computeProbability(pm.stats.Difficulty, metrics.TotalAttempts) * 100

	// Calculate estimated time
	if pm.stats.Probability50 > 0 && aggregated.TotalSpeed > 0 {
		remainingAttempts := pm.stats.Probability50 - metrics.TotalAttempts
		if remainingAttempts > 0 {
			aggregated.EstimatedTime = time.Duration(float64(remainingAttempts)/aggregated.TotalSpeed) * time.Second
		} else {
			aggregated.EstimatedTime = 0
		}
	}

	// Update the aggregated stats in one go to minimize race conditions
	pm.aggregatedStats = aggregated
}

// updateStatisticsObject updates the original Statistics object for compatibility
func (pm *ProgressManager) updateStatisticsObject() {
	// Create a local copy of the aggregated stats to minimize lock time
	// This is already thread-safe since we're inside a locked method
	aggregated := pm.aggregatedStats

	// Update the original statistics object with aggregated data
	// This maintains compatibility with existing code (requirement 6.3)
	// We use the updateFromAggregated method to ensure thread safety
	pm.stats.updateFromAggregated(aggregated)
}

// displayProgress shows a progress bar and statistics using aggregated multi-thread data
func (pm *ProgressManager) displayProgress() {
	// Clear line and move cursor to beginning
	fmt.Print("\r\033[K")

	// Use aggregated stats for thread-safe display
	progressPercent := pm.aggregatedStats.Probability
	if progressPercent > 100 {
		progressPercent = 100
	}

	// Calculate progress bar
	barWidth := 40
	filledWidth := int((progressPercent / 100) * float64(barWidth))

	// Create progress bar using a string builder for better performance
	sb := globalBufferPool.GetStringBuilder()
	defer globalBufferPool.PutStringBuilder(sb)

	sb.WriteRune('[')
	for i := 0; i < barWidth; i++ {
		if i < filledWidth {
			sb.WriteString("█")
		} else {
			sb.WriteString("░")
		}
	}
	sb.WriteRune(']')
	bar := sb.String()

	// Format output using aggregated data from all threads
	// Maintain exact same format as original for compatibility (requirement 6.3)
	fmt.Printf("%s %.2f%% | %s attempts | %.0f addr/s | Difficulty: %s",
		bar,
		pm.aggregatedStats.Probability,
		formatNumber(pm.aggregatedStats.TotalAttempts),
		pm.aggregatedStats.TotalSpeed,
		formatNumber(int64(pm.stats.Difficulty)),
	)

	// Show estimated time if available
	if pm.aggregatedStats.EstimatedTime > 0 {
		fmt.Printf(" | ETA: %s", formatDuration(pm.aggregatedStats.EstimatedTime))
	}

	// Add multi-thread information
	if pm.aggregatedStats.ActiveWorkers > 1 {
		fmt.Printf(" | 🧵 %d threads", pm.aggregatedStats.ActiveWorkers)
	}
}

// GetAggregatedStats returns a thread-safe copy of the current aggregated statistics
func (pm *ProgressManager) GetAggregatedStats() AggregatedStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Return a copy to prevent race conditions
	return AggregatedStats{
		TotalAttempts: pm.aggregatedStats.TotalAttempts,
		TotalSpeed:    pm.aggregatedStats.TotalSpeed,
		AverageSpeed:  pm.aggregatedStats.AverageSpeed,
		PeakSpeed:     pm.aggregatedStats.PeakSpeed,
		ActiveWorkers: pm.aggregatedStats.ActiveWorkers,
		Probability:   pm.aggregatedStats.Probability,
		EstimatedTime: pm.aggregatedStats.EstimatedTime,
		LastUpdate:    pm.aggregatedStats.LastUpdate,
	}
}

// IsActive returns whether the progress manager is currently active
func (pm *ProgressManager) IsActive() bool {
	return atomic.LoadInt32(&pm.isActive) == 1
}

// GetLastDisplayTime returns the time of the last progress display update
func (pm *ProgressManager) GetLastDisplayTime() time.Time {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.lastDisplayTime
}
