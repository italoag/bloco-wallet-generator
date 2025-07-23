package main

import (
	"math"
	"sync"
	"time"
)

// StatsManager provides thread-safe statistics aggregation for multiple workers
type StatsManager struct {
	mu            sync.RWMutex
	totalAttempts int64
	workerStats   map[int]WorkerStats
	startTime     time.Time
	lastUpdate    time.Time

	// Aggregated metrics
	totalSpeed float64
	avgSpeed   float64
	peakSpeed  float64
}

// NewStatsManager creates a new StatsManager instance
func NewStatsManager() *StatsManager {
	return &StatsManager{
		workerStats: make(map[int]WorkerStats),
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
	}
}

// UpdateWorkerStats updates statistics for a specific worker
func (sm *StatsManager) UpdateWorkerStats(stats WorkerStats) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.workerStats[stats.WorkerID] = stats

	// Recalculate aggregated metrics
	sm.recalculateMetrics()
}

// recalculateMetrics updates the aggregated metrics based on current worker stats
func (sm *StatsManager) recalculateMetrics() {
	var totalAttempts int64
	var totalSpeed float64

	for _, stats := range sm.workerStats {
		totalAttempts += stats.Attempts
		totalSpeed += stats.Speed

		// Update peak speed if this worker's speed is higher
		if stats.Speed > sm.peakSpeed {
			sm.peakSpeed = stats.Speed
		}
	}

	sm.totalAttempts = totalAttempts
	sm.totalSpeed = totalSpeed

	// Calculate average speed based on elapsed time
	elapsed := time.Since(sm.startTime).Seconds()
	if elapsed > 0 {
		sm.avgSpeed = float64(totalAttempts) / elapsed
	}

	sm.lastUpdate = time.Now()
}

// GetTotalAttempts returns the total number of attempts across all workers
func (sm *StatsManager) GetTotalAttempts() int64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.totalAttempts
}

// GetTotalSpeed returns the combined speed of all workers in addr/s
func (sm *StatsManager) GetTotalSpeed() float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.totalSpeed
}

// GetAverageSpeed returns the average speed since start in addr/s
func (sm *StatsManager) GetAverageSpeed() float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.avgSpeed
}

// GetPeakSpeed returns the highest observed speed in addr/s
func (sm *StatsManager) GetPeakSpeed() float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.peakSpeed
}

// GetWorkerCount returns the number of active workers
func (sm *StatsManager) GetWorkerCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.workerStats)
}

// GetElapsedTime returns the time elapsed since the stats manager was created
func (sm *StatsManager) GetElapsedTime() time.Duration {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return time.Since(sm.startTime)
}

// GetWorkerStats returns a copy of all worker statistics
func (sm *StatsManager) GetWorkerStats() map[int]WorkerStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Create a copy to avoid race conditions
	statsCopy := make(map[int]WorkerStats, len(sm.workerStats))
	for id, stats := range sm.workerStats {
		statsCopy[id] = stats
	}

	return statsCopy
}

// GetThreadEfficiency returns the efficiency of thread utilization (0-1)
func (sm *StatsManager) GetThreadEfficiency() float64 {
	metrics := sm.GetMetrics()
	return metrics.ThreadEfficiency
}

// GetSpeedupVsSingleThread returns the actual speedup compared to estimated single-thread performance
func (sm *StatsManager) GetSpeedupVsSingleThread() float64 {
	metrics := sm.GetMetrics()
	return metrics.SpeedupVsSingleThread
}

// GetThreadBalanceScore returns a score indicating how evenly work is distributed (0-1)
func (sm *StatsManager) GetThreadBalanceScore() float64 {
	metrics := sm.GetMetrics()
	return metrics.ThreadBalanceScore
}

// GetEstimatedSingleThreadSpeed returns the estimated speed of a single thread
func (sm *StatsManager) GetEstimatedSingleThreadSpeed() float64 {
	metrics := sm.GetMetrics()
	return metrics.EstimatedSingleThreadSpeed
}

// GetMetrics returns a snapshot of all current metrics
func (sm *StatsManager) GetMetrics() PerformanceMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Calculate thread utilization
	threadUtil := make(map[int]float64)
	totalSpeed := 0.0

	if sm.totalSpeed > 0 {
		for id, stats := range sm.workerStats {
			threadUtil[id] = stats.Speed / sm.totalSpeed
			totalSpeed += stats.Speed
		}
	}

	// Get worker count
	workerCount := len(sm.workerStats)

	// Calculate estimated single-thread speed
	var estimatedSingleThreadSpeed float64 = 0
	if workerCount > 0 && sm.avgSpeed > 0 {
		// Estimate single-thread speed by dividing average by worker count
		// This is an approximation since we don't have actual single-thread measurements
		estimatedSingleThreadSpeed = sm.avgSpeed / float64(workerCount)
	}

	// Calculate efficiency ratio (compared to theoretical linear scaling)
	var efficiencyRatio float64 = 0
	if workerCount > 0 && estimatedSingleThreadSpeed > 0 {
		efficiencyRatio = sm.totalSpeed / (estimatedSingleThreadSpeed * float64(workerCount))
	}

	// Calculate speedup vs single-thread
	var speedupVsSingleThread float64 = 0
	if estimatedSingleThreadSpeed > 0 {
		speedupVsSingleThread = sm.totalSpeed / estimatedSingleThreadSpeed
	}

	// Calculate thread balance score (how evenly work is distributed)
	var threadBalanceScore float64 = 1.0 // Default to perfect balance
	if workerCount > 1 && sm.totalSpeed > 0 {
		// Calculate standard deviation of thread speeds
		var sumSquaredDiff float64 = 0
		meanSpeed := sm.totalSpeed / float64(workerCount)

		for _, stats := range sm.workerStats {
			diff := stats.Speed - meanSpeed
			sumSquaredDiff += diff * diff
		}

		// Calculate coefficient of variation (normalized standard deviation)
		stdDev := math.Sqrt(sumSquaredDiff / float64(workerCount))
		if meanSpeed > 0 {
			coeffVar := stdDev / meanSpeed
			// Convert to a 0-1 score where 1 is perfect balance
			threadBalanceScore = 1.0 - math.Min(coeffVar, 1.0)
		}
	}

	// Estimate CPU utilization based on thread efficiency
	// This is a rough approximation without actual CPU profiling
	var cpuUtilization float64 = 0
	if workerCount > 0 {
		// Base utilization on efficiency ratio, but cap at 1.0 (100%)
		cpuUtilization = math.Min(efficiencyRatio, 1.0)
	}

	return PerformanceMetrics{
		ThreadUtilization:          threadUtil,
		TotalThroughput:            sm.totalSpeed,
		PerThreadSpeed:             sm.getPerThreadSpeed(),
		EfficiencyRatio:            efficiencyRatio,
		TotalAttempts:              sm.totalAttempts,
		ElapsedTime:                time.Since(sm.startTime),
		WorkerCount:                workerCount,
		ThreadEfficiency:           efficiencyRatio,
		CPUUtilization:             cpuUtilization,
		SpeedupVsSingleThread:      speedupVsSingleThread,
		ThreadBalanceScore:         threadBalanceScore,
		EstimatedSingleThreadSpeed: estimatedSingleThreadSpeed,
	}
}

// getPerThreadSpeed returns a map of worker IDs to their speeds
func (sm *StatsManager) getPerThreadSpeed() map[int]float64 {
	result := make(map[int]float64)
	for id, stats := range sm.workerStats {
		result[id] = stats.Speed
	}
	return result
}

// Reset resets all statistics
func (sm *StatsManager) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.totalAttempts = 0
	sm.workerStats = make(map[int]WorkerStats)
	sm.startTime = time.Now()
	sm.lastUpdate = time.Now()
	sm.totalSpeed = 0
	sm.avgSpeed = 0
	sm.peakSpeed = 0
}

// PerformanceMetrics contains aggregated performance data
type PerformanceMetrics struct {
	ThreadUtilization map[int]float64 // % utilization per thread
	TotalThroughput   float64         // addr/s total
	PerThreadSpeed    map[int]float64 // addr/s per thread
	EfficiencyRatio   float64         // Speedup vs theoretical linear scaling
	TotalAttempts     int64           // Total attempts across all workers
	ElapsedTime       time.Duration   // Time elapsed since start
	WorkerCount       int             // Number of active workers

	// Enhanced multi-thread metrics
	ThreadEfficiency           float64 // Actual speedup vs ideal linear speedup (0-1)
	CPUUtilization             float64 // Estimated CPU utilization across all threads (0-1)
	SpeedupVsSingleThread      float64 // Actual speedup compared to single-thread performance
	ThreadBalanceScore         float64 // Score indicating how evenly work is distributed (0-1)
	EstimatedSingleThreadSpeed float64 // Estimated speed of a single thread
}
