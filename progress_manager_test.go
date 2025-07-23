package main

import (
	"sync"
	"testing"
	"time"
)

// TestProgressManagerThreadSafety tests that the ProgressManager is thread-safe
func TestProgressManagerThreadSafety(t *testing.T) {
	// Initialize global pools for testing
	initializePools()

	// Create a stats manager and statistics object
	statsManager := NewStatsManager()
	stats := newStatistics("abc", "def", true)

	// Create a progress manager
	progressManager := NewProgressManager(stats, statsManager)

	// Start the progress manager
	progressManager.Start()
	defer progressManager.Stop()

	// Create multiple goroutines to simulate concurrent worker updates
	var wg sync.WaitGroup
	workerCount := 10
	updatesPerWorker := 100

	// Create a channel to simulate worker stats updates
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Simulate worker stats updates
			for j := 0; j < updatesPerWorker; j++ {
				statsManager.UpdateWorkerStats(WorkerStats{
					WorkerID:   workerID,
					Attempts:   int64(j + 1),
					Speed:      float64(j + 1),
					LastUpdate: time.Now(),
				})

				// Small sleep to simulate real work
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()

	// Verify that the progress manager has aggregated all worker stats
	aggregatedStats := progressManager.GetAggregatedStats()

	// Check that we have the expected number of workers
	if statsManager.GetWorkerCount() != workerCount {
		t.Errorf("Expected %d workers, got %d", workerCount, statsManager.GetWorkerCount())
	}

	// Check that the total attempts is as expected
	expectedAttempts := int64(workerCount * updatesPerWorker)
	if statsManager.GetTotalAttempts() != expectedAttempts {
		t.Errorf("Expected %d total attempts, got %d", expectedAttempts, statsManager.GetTotalAttempts())
	}

	// Force an update and check that the progress manager has the latest data
	progressManager.ForceUpdate()
	aggregatedStats = progressManager.GetAggregatedStats()

	if aggregatedStats.TotalAttempts != expectedAttempts {
		t.Errorf("Expected %d total attempts in aggregated stats, got %d", expectedAttempts, aggregatedStats.TotalAttempts)
	}

	if aggregatedStats.ActiveWorkers != workerCount {
		t.Errorf("Expected %d active workers in aggregated stats, got %d", workerCount, aggregatedStats.ActiveWorkers)
	}
}

// TestProgressManagerDisplayFormat tests that the progress display format is maintained
func TestProgressManagerDisplayFormat(t *testing.T) {
	// This is a visual test that can't be automatically verified
	// But we can at least ensure the code runs without panicking

	// Initialize global pools for testing
	initializePools()

	// Create a stats manager and statistics object
	statsManager := NewStatsManager()
	stats := newStatistics("abc", "def", true)

	// Create a progress manager
	progressManager := NewProgressManager(stats, statsManager)

	// Update some worker stats
	statsManager.UpdateWorkerStats(WorkerStats{
		WorkerID:   1,
		Attempts:   1000,
		Speed:      5000,
		LastUpdate: time.Now(),
	})

	statsManager.UpdateWorkerStats(WorkerStats{
		WorkerID:   2,
		Attempts:   2000,
		Speed:      6000,
		LastUpdate: time.Now(),
	})

	// Force an update to aggregate the stats
	progressManager.ForceUpdate()

	// Get the aggregated stats
	aggregatedStats := progressManager.GetAggregatedStats()

	// Verify that the aggregated stats are correct
	if aggregatedStats.TotalAttempts != 3000 {
		t.Errorf("Expected 3000 total attempts, got %d", aggregatedStats.TotalAttempts)
	}

	if aggregatedStats.TotalSpeed != 11000 {
		t.Errorf("Expected 11000 total speed, got %f", aggregatedStats.TotalSpeed)
	}

	if aggregatedStats.ActiveWorkers != 2 {
		t.Errorf("Expected 2 active workers, got %d", aggregatedStats.ActiveWorkers)
	}
}

// TestProgressManagerShutdown tests that the progress manager can be shut down gracefully
func TestProgressManagerShutdown(t *testing.T) {
	// Initialize global pools for testing
	initializePools()

	// Create a stats manager and statistics object
	statsManager := NewStatsManager()
	stats := newStatistics("abc", "def", true)

	// Create a progress manager
	progressManager := NewProgressManager(stats, statsManager)

	// Start the progress manager
	progressManager.Start()

	// Verify that the progress manager is active
	if !progressManager.IsActive() {
		t.Error("Expected progress manager to be active")
	}

	// Stop the progress manager
	progressManager.Stop()

	// Give it a moment to shut down
	time.Sleep(10 * time.Millisecond)

	// Verify that the progress manager is no longer active
	if progressManager.IsActive() {
		t.Error("Expected progress manager to be inactive after stopping")
	}

	// Start it again to ensure it can be restarted
	progressManager.Start()

	// Verify that the progress manager is active again
	if !progressManager.IsActive() {
		t.Error("Expected progress manager to be active after restarting")
	}

	// Stop it again
	progressManager.Stop()
}
