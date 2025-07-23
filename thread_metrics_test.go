package main

import (
	"testing"
	"time"
)

func TestThreadMetricsCalculation(t *testing.T) {
	// Create a stats manager for testing
	statsManager := NewStatsManager()

	// Set start time to simulate running for a while
	statsManager.startTime = time.Now().Add(-10 * time.Second)

	// Add some mock worker stats
	statsManager.UpdateWorkerStats(WorkerStats{
		WorkerID:   0,
		Attempts:   1000,
		Speed:      10000,
		LastUpdate: time.Now(),
	})

	statsManager.UpdateWorkerStats(WorkerStats{
		WorkerID:   1,
		Attempts:   1200,
		Speed:      12000,
		LastUpdate: time.Now(),
	})

	statsManager.UpdateWorkerStats(WorkerStats{
		WorkerID:   2,
		Attempts:   900,
		Speed:      9000,
		LastUpdate: time.Now(),
	})

	statsManager.UpdateWorkerStats(WorkerStats{
		WorkerID:   3,
		Attempts:   1100,
		Speed:      11000,
		LastUpdate: time.Now(),
	})

	// Get metrics
	metrics := statsManager.GetMetrics()

	// Test worker count
	if metrics.WorkerCount != 4 {
		t.Errorf("Expected 4 workers, got %d", metrics.WorkerCount)
	}

	// Test total throughput
	expectedThroughput := 42000.0
	if metrics.TotalThroughput != expectedThroughput {
		t.Errorf("Expected throughput %.0f, got %.0f", expectedThroughput, metrics.TotalThroughput)
	}

	// Test thread balance score (should be high since workers are balanced)
	if metrics.ThreadBalanceScore < 0.85 {
		t.Errorf("Expected high thread balance score (>0.85), got %.2f", metrics.ThreadBalanceScore)
	}

	// Test speedup vs single thread - we need to manually set avgSpeed for this test
	statsManager.avgSpeed = 10500.0     // Average speed across all threads
	metrics = statsManager.GetMetrics() // Get updated metrics

	// Now check speedup
	if metrics.SpeedupVsSingleThread < 3.0 {
		t.Errorf("Expected speedup at least 3.0, got %.1f", metrics.SpeedupVsSingleThread)
	}

	// Test thread utilization
	for id, utilization := range metrics.ThreadUtilization {
		// Each thread should be around 25% of total
		expectedUtil := 0.25
		if utilization < expectedUtil*0.8 || utilization > expectedUtil*1.2 {
			t.Errorf("Thread %d: Expected utilization around %.2f, got %.2f", id, expectedUtil, utilization)
		}
	}
}

func TestThreadMetricsUnbalanced(t *testing.T) {
	// Create a stats manager for testing
	statsManager := NewStatsManager()

	// Set start time to simulate running for a while
	statsManager.startTime = time.Now().Add(-10 * time.Second)

	// Add some mock worker stats with unbalanced load
	statsManager.UpdateWorkerStats(WorkerStats{
		WorkerID:   0,
		Attempts:   5000,
		Speed:      50000,
		LastUpdate: time.Now(),
	})

	statsManager.UpdateWorkerStats(WorkerStats{
		WorkerID:   1,
		Attempts:   500,
		Speed:      5000,
		LastUpdate: time.Now(),
	})

	statsManager.UpdateWorkerStats(WorkerStats{
		WorkerID:   2,
		Attempts:   300,
		Speed:      3000,
		LastUpdate: time.Now(),
	})

	statsManager.UpdateWorkerStats(WorkerStats{
		WorkerID:   3,
		Attempts:   200,
		Speed:      2000,
		LastUpdate: time.Now(),
	})

	// Get metrics
	metrics := statsManager.GetMetrics()

	// Test thread balance score (should be low since workers are unbalanced)
	if metrics.ThreadBalanceScore > 0.5 {
		t.Errorf("Expected low thread balance score (<0.5), got %.2f", metrics.ThreadBalanceScore)
	}

	// Test thread utilization
	if metrics.ThreadUtilization[0] < 0.7 {
		t.Errorf("Expected high utilization for thread 0 (>0.7), got %.2f", metrics.ThreadUtilization[0])
	}
}
