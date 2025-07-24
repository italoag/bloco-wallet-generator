package main

import (
	"sync"
	"testing"
	"time"
)

func TestNewStatsManager(t *testing.T) {
	sm := NewStatsManager()

	if sm == nil {
		t.Fatal("NewStatsManager returned nil")
	}

	if sm.workerStats == nil {
		t.Error("workerStats map not initialized")
	}

	if sm.totalAttempts != 0 {
		t.Errorf("Expected initial totalAttempts to be 0, got %d", sm.totalAttempts)
	}
}

func TestUpdateWorkerStats(t *testing.T) {
	sm := NewStatsManager()

	// Add stats for worker 1
	stats1 := WorkerStats{
		WorkerID:   1,
		Attempts:   1000,
		Speed:      5000,
		LastUpdate: time.Now(),
	}
	sm.UpdateWorkerStats(stats1)

	// Verify stats were updated
	if sm.GetTotalAttempts() != 1000 {
		t.Errorf("Expected totalAttempts to be 1000, got %d", sm.GetTotalAttempts())
	}

	if sm.GetTotalSpeed() != 5000 {
		t.Errorf("Expected totalSpeed to be 5000, got %f", sm.GetTotalSpeed())
	}

	// Add stats for worker 2
	stats2 := WorkerStats{
		WorkerID:   2,
		Attempts:   2000,
		Speed:      7000,
		LastUpdate: time.Now(),
	}
	sm.UpdateWorkerStats(stats2)

	// Verify aggregated stats
	if sm.GetTotalAttempts() != 3000 {
		t.Errorf("Expected totalAttempts to be 3000, got %d", sm.GetTotalAttempts())
	}

	if sm.GetTotalSpeed() != 12000 {
		t.Errorf("Expected totalSpeed to be 12000, got %f", sm.GetTotalSpeed())
	}

	// Update worker 1 stats
	stats1.Attempts = 1500
	stats1.Speed = 6000
	sm.UpdateWorkerStats(stats1)

	// Verify updated aggregated stats
	if sm.GetTotalAttempts() != 3500 {
		t.Errorf("Expected totalAttempts to be 3500, got %d", sm.GetTotalAttempts())
	}

	if sm.GetTotalSpeed() != 13000 {
		t.Errorf("Expected totalSpeed to be 13000, got %f", sm.GetTotalSpeed())
	}
}

func TestGetWorkerStats(t *testing.T) {
	sm := NewStatsManager()

	// Add stats for two workers
	stats1 := WorkerStats{
		WorkerID:   1,
		Attempts:   1000,
		Speed:      5000,
		LastUpdate: time.Now(),
	}
	sm.UpdateWorkerStats(stats1)

	stats2 := WorkerStats{
		WorkerID:   2,
		Attempts:   2000,
		Speed:      7000,
		LastUpdate: time.Now(),
	}
	sm.UpdateWorkerStats(stats2)

	// Get worker stats
	workerStats := sm.GetWorkerStats()

	// Verify we got stats for both workers
	if len(workerStats) != 2 {
		t.Errorf("Expected stats for 2 workers, got %d", len(workerStats))
	}

	// Verify worker 1 stats
	if ws, ok := workerStats[1]; ok {
		if ws.Attempts != 1000 {
			t.Errorf("Expected worker 1 attempts to be 1000, got %d", ws.Attempts)
		}
		if ws.Speed != 5000 {
			t.Errorf("Expected worker 1 speed to be 5000, got %f", ws.Speed)
		}
	} else {
		t.Error("Worker 1 stats not found")
	}

	// Verify worker 2 stats
	if ws, ok := workerStats[2]; ok {
		if ws.Attempts != 2000 {
			t.Errorf("Expected worker 2 attempts to be 2000, got %d", ws.Attempts)
		}
		if ws.Speed != 7000 {
			t.Errorf("Expected worker 2 speed to be 7000, got %f", ws.Speed)
		}
	} else {
		t.Error("Worker 2 stats not found")
	}
}

func TestGetMetrics(t *testing.T) {
	sm := NewStatsManager()

	// Add stats for two workers
	stats1 := WorkerStats{
		WorkerID:   1,
		Attempts:   1000,
		Speed:      5000,
		LastUpdate: time.Now(),
	}
	sm.UpdateWorkerStats(stats1)

	stats2 := WorkerStats{
		WorkerID:   2,
		Attempts:   2000,
		Speed:      7000,
		LastUpdate: time.Now(),
	}
	sm.UpdateWorkerStats(stats2)

	// Get metrics
	metrics := sm.GetMetrics()

	// Verify metrics
	if metrics.TotalAttempts != 3000 {
		t.Errorf("Expected totalAttempts to be 3000, got %d", metrics.TotalAttempts)
	}

	if metrics.TotalThroughput != 12000 {
		t.Errorf("Expected totalThroughput to be 12000, got %f", metrics.TotalThroughput)
	}

	if metrics.WorkerCount != 2 {
		t.Errorf("Expected workerCount to be 2, got %d", metrics.WorkerCount)
	}

	// Verify per-thread speed
	if speed, ok := metrics.PerThreadSpeed[1]; ok {
		if speed != 5000 {
			t.Errorf("Expected worker 1 speed to be 5000, got %f", speed)
		}
	} else {
		t.Error("Worker 1 speed not found in metrics")
	}

	if speed, ok := metrics.PerThreadSpeed[2]; ok {
		if speed != 7000 {
			t.Errorf("Expected worker 2 speed to be 7000, got %f", speed)
		}
	} else {
		t.Error("Worker 2 speed not found in metrics")
	}
}

func TestReset(t *testing.T) {
	sm := NewStatsManager()

	// Add stats for a worker
	stats := WorkerStats{
		WorkerID:   1,
		Attempts:   1000,
		Speed:      5000,
		LastUpdate: time.Now(),
	}
	sm.UpdateWorkerStats(stats)

	// Verify stats were updated
	if sm.GetTotalAttempts() != 1000 {
		t.Errorf("Expected totalAttempts to be 1000, got %d", sm.GetTotalAttempts())
	}

	// Reset stats
	sm.Reset()

	// Verify stats were reset
	if sm.GetTotalAttempts() != 0 {
		t.Errorf("Expected totalAttempts to be 0 after reset, got %d", sm.GetTotalAttempts())
	}

	if sm.GetTotalSpeed() != 0 {
		t.Errorf("Expected totalSpeed to be 0 after reset, got %f", sm.GetTotalSpeed())
	}

	if sm.GetWorkerCount() != 0 {
		t.Errorf("Expected workerCount to be 0 after reset, got %d", sm.GetWorkerCount())
	}
}

// TestStatsManagerThreadSafety tests concurrent access to StatsManager
func TestStatsManagerThreadSafety(t *testing.T) {
	sm := NewStatsManager()
	numWorkers := 20
	updatesPerWorker := 100

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	// Start multiple goroutines updating stats concurrently
	for workerID := 0; workerID < numWorkers; workerID++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < updatesPerWorker; i++ {
				stats := WorkerStats{
					WorkerID:   id,
					Attempts:   int64(i + 1),
					Speed:      float64((i + 1) * 100),
					LastUpdate: time.Now(),
				}
				sm.UpdateWorkerStats(stats)

				// Also test concurrent reads
				_ = sm.GetTotalAttempts()
				_ = sm.GetTotalSpeed()
				_ = sm.GetWorkerCount()
				_ = sm.GetMetrics()
			}
		}(workerID)
	}

	// Start additional goroutines doing concurrent reads
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < updatesPerWorker*2; j++ {
				_ = sm.GetTotalAttempts()
				_ = sm.GetTotalSpeed()
				_ = sm.GetAverageSpeed()
				_ = sm.GetPeakSpeed()
				_ = sm.GetWorkerCount()
				_ = sm.GetElapsedTime()
				_ = sm.GetWorkerStats()
				_ = sm.GetMetrics()
				time.Sleep(time.Microsecond) // Small delay to increase chance of race conditions
			}
		}()
	}

	wg.Wait()

	// Verify final state is consistent
	totalAttempts := sm.GetTotalAttempts()
	expectedAttempts := int64(numWorkers * updatesPerWorker)
	if totalAttempts != expectedAttempts {
		t.Errorf("Expected total attempts %d, got %d", expectedAttempts, totalAttempts)
	}

	workerCount := sm.GetWorkerCount()
	if workerCount != numWorkers {
		t.Errorf("Expected worker count %d, got %d", numWorkers, workerCount)
	}

	// Verify metrics are consistent
	metrics := sm.GetMetrics()
	if metrics.TotalAttempts != totalAttempts {
		t.Errorf("Metrics total attempts %d doesn't match GetTotalAttempts %d", metrics.TotalAttempts, totalAttempts)
	}

	if metrics.WorkerCount != workerCount {
		t.Errorf("Metrics worker count %d doesn't match GetWorkerCount %d", metrics.WorkerCount, workerCount)
	}
}

// TestStatsManagerConcurrentReset tests concurrent reset operations
func TestStatsManagerConcurrentReset(t *testing.T) {
	sm := NewStatsManager()
	numGoroutines := 10
	operationsPerGoroutine := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start goroutines that update stats and reset concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				if j%10 == 0 {
					// Reset every 10 operations
					sm.Reset()
				} else {
					// Update stats
					stats := WorkerStats{
						WorkerID:   id,
						Attempts:   int64(j),
						Speed:      float64(j * 100),
						LastUpdate: time.Now(),
					}
					sm.UpdateWorkerStats(stats)
				}

				// Read stats
				_ = sm.GetTotalAttempts()
				_ = sm.GetTotalSpeed()
			}
		}(i)
	}

	wg.Wait()

	// After all operations, the stats manager should still be in a valid state
	_ = sm.GetTotalAttempts()
	_ = sm.GetTotalSpeed()
	_ = sm.GetWorkerCount()
	metrics := sm.GetMetrics()
	if metrics.WorkerCount < 0 {
		t.Error("Worker count should not be negative after concurrent operations")
	}
}

// TestStatsManagerMetricsCalculation tests the accuracy of calculated metrics
func TestStatsManagerMetricsCalculation(t *testing.T) {
	sm := NewStatsManager()

	// Add stats for multiple workers with known values
	workers := []WorkerStats{
		{WorkerID: 1, Attempts: 1000, Speed: 1000, LastUpdate: time.Now()},
		{WorkerID: 2, Attempts: 2000, Speed: 2000, LastUpdate: time.Now()},
		{WorkerID: 3, Attempts: 3000, Speed: 3000, LastUpdate: time.Now()},
	}

	for _, stats := range workers {
		sm.UpdateWorkerStats(stats)
	}

	metrics := sm.GetMetrics()

	// Verify total attempts
	expectedTotalAttempts := int64(6000)
	if metrics.TotalAttempts != expectedTotalAttempts {
		t.Errorf("Expected total attempts %d, got %d", expectedTotalAttempts, metrics.TotalAttempts)
	}

	// Verify total throughput
	expectedTotalThroughput := 6000.0
	if metrics.TotalThroughput != expectedTotalThroughput {
		t.Errorf("Expected total throughput %f, got %f", expectedTotalThroughput, metrics.TotalThroughput)
	}

	// Verify worker count
	if metrics.WorkerCount != 3 {
		t.Errorf("Expected worker count 3, got %d", metrics.WorkerCount)
	}

	// Verify per-thread speeds
	for _, worker := range workers {
		if speed, ok := metrics.PerThreadSpeed[worker.WorkerID]; ok {
			if speed != worker.Speed {
				t.Errorf("Worker %d: expected speed %f, got %f", worker.WorkerID, worker.Speed, speed)
			}
		} else {
			t.Errorf("Worker %d speed not found in metrics", worker.WorkerID)
		}
	}

	// Verify thread utilization
	for _, worker := range workers {
		if util, ok := metrics.ThreadUtilization[worker.WorkerID]; ok {
			expectedUtil := worker.Speed / expectedTotalThroughput
			if util != expectedUtil {
				t.Errorf("Worker %d: expected utilization %f, got %f", worker.WorkerID, expectedUtil, util)
			}
		} else {
			t.Errorf("Worker %d utilization not found in metrics", worker.WorkerID)
		}
	}

	// Verify efficiency metrics are within reasonable bounds
	if metrics.EfficiencyRatio < 0 || metrics.EfficiencyRatio > 1 {
		t.Errorf("Efficiency ratio should be between 0 and 1, got %f", metrics.EfficiencyRatio)
	}

	if metrics.ThreadEfficiency < 0 || metrics.ThreadEfficiency > 1 {
		t.Errorf("Thread efficiency should be between 0 and 1, got %f", metrics.ThreadEfficiency)
	}

	if metrics.ThreadBalanceScore < 0 || metrics.ThreadBalanceScore > 1 {
		t.Errorf("Thread balance score should be between 0 and 1, got %f", metrics.ThreadBalanceScore)
	}
}

// TestStatsManagerPeakSpeedTracking tests peak speed tracking
func TestStatsManagerPeakSpeedTracking(t *testing.T) {
	sm := NewStatsManager()

	// Add stats with increasing speeds
	speeds := []float64{1000, 1500, 2000, 1200, 1800, 2500, 1000}
	expectedPeak := 2500.0

	for i, speed := range speeds {
		stats := WorkerStats{
			WorkerID:   1,
			Attempts:   int64((i + 1) * 100),
			Speed:      speed,
			LastUpdate: time.Now(),
		}
		sm.UpdateWorkerStats(stats)
	}

	peakSpeed := sm.GetPeakSpeed()
	if peakSpeed != expectedPeak {
		t.Errorf("Expected peak speed %f, got %f", expectedPeak, peakSpeed)
	}

	// Verify peak speed is also correct in metrics context
	// Note: The current implementation doesn't expose peak speed in metrics,
	// but we can verify it through the StatsManager method
	if sm.GetPeakSpeed() != expectedPeak {
		t.Errorf("Peak speed in metrics context: expected %f, got %f", expectedPeak, sm.GetPeakSpeed())
	}
}

// TestStatsManagerWorkerStatsIsolation tests that worker stats are properly isolated
func TestStatsManagerWorkerStatsIsolation(t *testing.T) {
	sm := NewStatsManager()

	// Add stats for worker 1
	stats1 := WorkerStats{
		WorkerID:   1,
		Attempts:   1000,
		Speed:      5000,
		LastUpdate: time.Now(),
	}
	sm.UpdateWorkerStats(stats1)

	// Add stats for worker 2
	stats2 := WorkerStats{
		WorkerID:   2,
		Attempts:   2000,
		Speed:      7000,
		LastUpdate: time.Now(),
	}
	sm.UpdateWorkerStats(stats2)

	// Get worker stats copy
	workerStats := sm.GetWorkerStats()

	// Modify the returned copy
	if ws, ok := workerStats[1]; ok {
		ws.Attempts = 9999
		ws.Speed = 9999
		workerStats[1] = ws
	}

	// Verify original stats are unchanged
	originalStats := sm.GetWorkerStats()
	if ws, ok := originalStats[1]; ok {
		if ws.Attempts != 1000 {
			t.Errorf("Original stats were modified: expected attempts 1000, got %d", ws.Attempts)
		}
		if ws.Speed != 5000 {
			t.Errorf("Original stats were modified: expected speed 5000, got %f", ws.Speed)
		}
	}
}
