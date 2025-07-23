package main

import (
	"fmt"
	"os"
)

// validateThreadCount validates the thread count and sets it appropriately
// Returns the validated thread count
func validateThreadCount(threads int) int {
	// Validate thread count
	if threads < 0 {
		fmt.Println("❌ Error: Number of threads cannot be negative")
		fmt.Println("💡 Use a positive number or 0 for auto-detection")
		os.Exit(1)
	}

	// Get available CPU cores
	availableCPUs := detectCPUCount()

	// Auto-detect when threads=0
	if threads == 0 {
		threads = availableCPUs
		fmt.Printf("🔧 Auto-detected %d CPU cores, using %d threads\n", availableCPUs, threads)
	} else {
		fmt.Printf("🔧 Using %d threads as specified\n", threads)
	}

	// Validate thread count doesn't exceed reasonable limits
	maxRecommendedThreads := availableCPUs * 2 // Allow up to 2x CPU cores

	if threads > maxRecommendedThreads {
		fmt.Printf("⚠️  Warning: Using %d threads on %d CPU cores may not be optimal\n", threads, availableCPUs)
		fmt.Println("💡 Performance might degrade with too many threads due to context switching overhead")
	}

	// Check for extremely low thread count on multi-core systems
	if threads == 1 && availableCPUs > 4 {
		fmt.Printf("ℹ️  Note: Using single-threaded mode on a %d-core system\n", availableCPUs)
		fmt.Println("💡 Consider using more threads for better performance")
	}

	return threads
}
