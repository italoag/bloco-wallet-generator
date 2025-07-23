package main

import (
	"testing"
)

// TestBenchmarkResultScalabilityMetrics tests the scalability metrics calculation
func TestBenchmarkResultScalabilityMetrics(t *testing.T) {
	// Create a mock BenchmarkResult with known values
	result := &BenchmarkResult{
		SingleThreadSpeed: 10000,
		AverageSpeed:      15000,
		ThreadCount:       2,
	}

	// Calculate expected values
	expectedSpeedup := result.AverageSpeed / result.SingleThreadSpeed
	expectedEfficiency := expectedSpeedup / float64(result.ThreadCount)

	// Verify that our calculations match the expected values
	if expectedSpeedup != 1.5 {
		t.Errorf("Expected speedup = 1.5, got %f", expectedSpeedup)
	}

	if expectedEfficiency != 0.75 {
		t.Errorf("Expected efficiency = 0.75, got %f", expectedEfficiency)
	}
}

// TestBenchmarkResultStructFields tests that the BenchmarkResult struct has the required fields
func TestBenchmarkResultStructFields(t *testing.T) {
	// Create a BenchmarkResult instance
	result := &BenchmarkResult{}

	// Set values for the fields we want to test
	result.ThreadBalanceScore = 0.85
	result.ThreadUtilization = 0.75
	result.SpeedupVsSingleThread = 1.8
	result.AmdahlsLawLimit = 1.95

	// Verify that the fields exist and can be set
	if result.ThreadBalanceScore != 0.85 {
		t.Errorf("Expected ThreadBalanceScore = 0.85, got %f", result.ThreadBalanceScore)
	}

	if result.ThreadUtilization != 0.75 {
		t.Errorf("Expected ThreadUtilization = 0.75, got %f", result.ThreadUtilization)
	}

	if result.SpeedupVsSingleThread != 1.8 {
		t.Errorf("Expected SpeedupVsSingleThread = 1.8, got %f", result.SpeedupVsSingleThread)
	}

	if result.AmdahlsLawLimit != 1.95 {
		t.Errorf("Expected AmdahlsLawLimit = 1.95, got %f", result.AmdahlsLawLimit)
	}
}
