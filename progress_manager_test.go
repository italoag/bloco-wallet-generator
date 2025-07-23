package main

import (
	"sync"
	"testing"
	"time"
)

// TestProgressManagerThreadSafety tests that the ProgressManager is thread-safe
func TestProgressManagerThreadSafety(t *testing.T) {
	// Create a statistics manager
	statsManager := NewStatsManager()

	// Create statistics
	stats := newStatistics("a", "b", false)

	// Create a progress manager
	progressManager := NewProgressManager(stats, statsManager)

	// Use a shorter update interval for testing
	progressManager.UpdateInterval(10 * time.Millisecond)

	// Start the progress manager
	progressManager.Start()

	// Create multiple goroutines to update stats simultaneously
	var wg sync.WaitGroup
	workerCount := 10
	updatesPerWorker := 100

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Create worker stats
			workerStats := WorkerStats{
				WorkerID:   workerID,
				Attempts:   0,
				Speed:      float64(workerID * 1000),
				LastUpdate: time.Now(),
			}

			// Update stats multiple times
			for j := 0; j < updatesPerWorker; j++ {
				workerStats.Attempts += 1000
				statsManager.UpdateWorkerStats(workerStats)
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Stop the progress manager
	progressManager.Stop()

	// Verify that the total attempts match what we expect
	expectedAttempts := int64(workerCount * updatesPerWorker * 1000)
	actualAttempts := statsManager.GetTotalAttempts()

	if actualAttempts != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, actualAttempts)
	}
}

// TestProgressManagerForceUpdate tests that ForceUpdate works correctly
func TestProgressManagerForceUpdate(t *testing.T) {
	// Create a statistics manager
	statsManager := NewStatsManager()

	// Create statistics
	stats := newStatistics("a", "b", false)

	// Create a progress manager with a very long update interval
	progressManager := NewProgressManager(stats, statsManager)
	progressManager.UpdateInterval(1 * time.Hour) // Long enough that it won't trigger during test

	// Start the progress manager
	progressManager.Start()

	// Update stats
	workerStats := WorkerStats{
		WorkerID:   1,
		Attempts:   1000,
		Speed:      1000,
		LastUpdate: time.Now(),
	}
	statsManager.UpdateWorkerStats(workerStats)

	// Force an update
	progressManager.ForceUpdate()

	// Verify that stats were updated
	if stats.CurrentAttempts != 1000 {
		t.Errorf("Expected 1000 attempts, got %d", stats.CurrentAttempts)
	}

	// Stop the progress manager
	progressManager.Stop()
}

// TestProgressManagerMultipleStartStop tests that the progress manager can be started and stopped multiple times
func TestProgressManagerMultipleStartStop(t *testing.T) {
	// Create a statistics manager
	statsManager := NewStatsManager()

	// Create statistics
	stats := newStatistics("a", "b", false)

	// Create a progress manager
	progressManager := NewProgressManager(stats, statsManager)

	// Start and stop multiple times
	for i := 0; i < 5; i++ {
		progressManager.Start()
		time.Sleep(10 * time.Millisecond)
		progressManager.Stop()
	}

	// Start again to ensure it still works
	progressManager.Start()
	progressManager.Stop()
}
