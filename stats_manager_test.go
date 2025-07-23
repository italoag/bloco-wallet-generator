package main

import (
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
