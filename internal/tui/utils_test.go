package tui

import (
	"testing"
	"time"
)

// TestFormatLargeNumber tests the number formatting function
func TestFormatLargeNumber(t *testing.T) {
	testCases := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{123, "123"},
		{1234, "1 234"},
		{12345, "12 345"},
		{123456, "123 456"},
		{1234567, "1 234 567"},
		{12345678, "12 345 678"},
		{123456789, "123 456 789"},
		{1000000000, "1 000 000 000"},
	}

	for _, tc := range testCases {
		result := formatLargeNumber(tc.input)
		if result != tc.expected {
			t.Errorf("formatLargeNumber(%d) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

// TestFormatDuration tests the duration formatting function
func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		input    time.Duration
		expected string
	}{
		{-1 * time.Second, "Nearly impossible"},
		{30 * time.Second, "30.0s"},
		{90 * time.Second, "1.5m"},
		{3900 * time.Second, "1.1h"},
		{90000 * time.Second, "1.0d"},
		{32000000 * time.Second, "1.0y"},
		{time.Duration(250*365*24) * time.Hour, "Thousands of years"},
	}

	for _, tc := range testCases {
		result := formatDuration(tc.input)
		if result != tc.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

// BenchmarkFormatLargeNumber benchmarks the number formatting function
func BenchmarkFormatLargeNumber(b *testing.B) {
	testNumbers := []int64{123, 1234567, 123456789, 1000000000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, num := range testNumbers {
			formatLargeNumber(num)
		}
	}
}

// BenchmarkFormatDuration benchmarks the duration formatting function
func BenchmarkFormatDuration(b *testing.B) {
	testDurations := []time.Duration{
		30 * time.Second,
		90 * time.Second,
		3900 * time.Second,
		90000 * time.Second,
		32000000 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, dur := range testDurations {
			formatDuration(dur)
		}
	}
}
