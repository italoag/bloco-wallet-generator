package main

import (
	"fmt"
	"sort"
	"strings"
)

// DisplayThreadMetrics shows detailed performance metrics for multi-threaded execution
func DisplayThreadMetrics(statsManager *StatsManager) {
	metrics := statsManager.GetMetrics()

	// Only display metrics if we have multiple threads
	if metrics.WorkerCount <= 1 {
		return
	}

	fmt.Printf("\n🧵 Thread Performance Metrics\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")

	// Display overall metrics
	fmt.Printf("📊 Overall Performance:\n")
	fmt.Printf("   - Total Throughput: %.0f addr/s\n", metrics.TotalThroughput)
	fmt.Printf("   - Estimated Single-Thread Speed: %.0f addr/s\n", metrics.EstimatedSingleThreadSpeed)
	fmt.Printf("   - Speedup vs Single-Thread: %.2fx\n", metrics.SpeedupVsSingleThread)
	fmt.Printf("   - Thread Efficiency: %.1f%%\n", metrics.ThreadEfficiency*100)
	fmt.Printf("   - Thread Balance: %.1f%%\n", metrics.ThreadBalanceScore*100)

	// Display per-thread metrics
	fmt.Printf("\n📈 Per-Thread Performance:\n")

	// Sort thread IDs for consistent display
	threadIDs := make([]int, 0, len(metrics.PerThreadSpeed))
	for id := range metrics.PerThreadSpeed {
		threadIDs = append(threadIDs, id)
	}
	sort.Ints(threadIDs)

	// Calculate max thread ID width for alignment
	maxIDWidth := 1
	for _, id := range threadIDs {
		width := len(fmt.Sprintf("%d", id))
		if width > maxIDWidth {
			maxIDWidth = width
		}
	}

	// Display header
	fmt.Printf("   %s | %12s | %12s\n",
		padRight("Thread", maxIDWidth+2),
		"Speed (addr/s)",
		"Utilization")

	fmt.Printf("   %s-+-%12s-+-%12s\n",
		strings.Repeat("-", maxIDWidth+2),
		strings.Repeat("-", 12),
		strings.Repeat("-", 12))

	// Display per-thread stats
	for _, id := range threadIDs {
		speed := metrics.PerThreadSpeed[id]
		utilization := metrics.ThreadUtilization[id] * 100

		fmt.Printf("   %s | %12.0f | %11.1f%%\n",
			padRight(fmt.Sprintf("T%d", id), maxIDWidth+2),
			speed,
			utilization)
	}

	fmt.Printf("\n💡 Efficiency Analysis:\n")

	// Provide analysis based on efficiency
	if metrics.ThreadEfficiency >= 0.9 {
		fmt.Printf("   ✅ Excellent parallelization efficiency (%.1f%%)\n", metrics.ThreadEfficiency*100)
	} else if metrics.ThreadEfficiency >= 0.7 {
		fmt.Printf("   ✓ Good parallelization efficiency (%.1f%%)\n", metrics.ThreadEfficiency*100)
	} else if metrics.ThreadEfficiency >= 0.5 {
		fmt.Printf("   ⚠️ Moderate parallelization efficiency (%.1f%%)\n", metrics.ThreadEfficiency*100)
	} else {
		fmt.Printf("   ❌ Poor parallelization efficiency (%.1f%%)\n", metrics.ThreadEfficiency*100)
	}

	// Provide analysis based on thread balance
	if metrics.ThreadBalanceScore >= 0.9 {
		fmt.Printf("   ✅ Excellent thread balance (%.1f%%)\n", metrics.ThreadBalanceScore*100)
	} else if metrics.ThreadBalanceScore >= 0.7 {
		fmt.Printf("   ✓ Good thread balance (%.1f%%)\n", metrics.ThreadBalanceScore*100)
	} else if metrics.ThreadBalanceScore >= 0.5 {
		fmt.Printf("   ⚠️ Moderate thread balance (%.1f%%)\n", metrics.ThreadBalanceScore*100)
	} else {
		fmt.Printf("   ❌ Poor thread balance (%.1f%%)\n", metrics.ThreadBalanceScore*100)
	}

	// Provide speedup analysis
	idealSpeedup := float64(metrics.WorkerCount)
	speedupEfficiency := metrics.SpeedupVsSingleThread / idealSpeedup * 100

	fmt.Printf("   - Actual speedup: %.2fx (%.1f%% of ideal %.0fx speedup)\n",
		metrics.SpeedupVsSingleThread,
		speedupEfficiency,
		idealSpeedup)
}

// padRight pads a string with spaces to the specified width
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
