package main

import (
	"fmt"
	"sync"
	"time"
)

// ProgressManager provides thread-safe progress display for multi-threaded operations
type ProgressManager struct {
	mu              sync.Mutex
	statsManager    *StatsManager
	stats           *Statistics
	shutdownChan    chan struct{}
	updateInterval  time.Duration
	lastDisplayTime time.Time
	isActive        bool
}

// NewProgressManager creates a new ProgressManager instance
func NewProgressManager(stats *Statistics, statsManager *StatsManager) *ProgressManager {
	return &ProgressManager{
		statsManager:    statsManager,
		stats:           stats,
		shutdownChan:    make(chan struct{}),
		updateInterval:  ProgressUpdateInterval,
		lastDisplayTime: time.Now(),
		isActive:        false,
	}
}

// Start begins the progress display loop
func (pm *ProgressManager) Start() {
	pm.mu.Lock()
	if pm.isActive {
		pm.mu.Unlock()
		return
	}
	pm.isActive = true
	pm.mu.Unlock()

	go pm.displayLoop()
}

// Stop terminates the progress display loop
func (pm *ProgressManager) Stop() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.isActive {
		return
	}

	close(pm.shutdownChan)
	pm.isActive = false

	// Create a new shutdown channel for future use
	pm.shutdownChan = make(chan struct{})
}

// UpdateInterval changes the progress update interval
func (pm *ProgressManager) UpdateInterval(interval time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.updateInterval = interval
}

// ForceUpdate forces an immediate progress update
func (pm *ProgressManager) ForceUpdate() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.isActive {
		return
	}

	pm.updateProgressDisplay()
}

// displayLoop runs the progress display loop
func (pm *ProgressManager) displayLoop() {
	ticker := time.NewTicker(pm.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.mu.Lock()
			pm.updateProgressDisplay()
			pm.mu.Unlock()

		case <-pm.shutdownChan:
			// Final update before exiting
			pm.mu.Lock()
			pm.updateProgressDisplay()
			pm.mu.Unlock()
			return
		}
	}
}

// updateProgressDisplay updates and displays the progress
func (pm *ProgressManager) updateProgressDisplay() {
	// Get current stats from the stats manager
	attempts := pm.statsManager.GetTotalAttempts()
	speed := pm.statsManager.GetTotalSpeed()

	// Update statistics
	pm.stats.CurrentAttempts = attempts
	pm.stats.Speed = speed
	pm.stats.Probability = computeProbability(pm.stats.Difficulty, attempts) * 100

	now := time.Now()
	elapsed := now.Sub(pm.stats.StartTime)

	if elapsed.Seconds() > 0 {
		// Update estimated time
		if pm.stats.Probability50 > 0 {
			remainingAttempts := pm.stats.Probability50 - attempts
			if remainingAttempts > 0 && speed > 0 {
				pm.stats.EstimatedTime = time.Duration(float64(remainingAttempts)/speed) * time.Second
			} else {
				pm.stats.EstimatedTime = 0
			}
		}
	}

	pm.stats.LastUpdate = now

	// Display the progress
	pm.displayProgress()
	pm.lastDisplayTime = now
}

// displayProgress shows a progress bar and statistics
func (pm *ProgressManager) displayProgress() {
	// Clear line and move cursor to beginning
	fmt.Print("\r\033[K")

	// Calculate progress bar
	barWidth := 40
	progressPercent := pm.stats.Probability
	if progressPercent > 100 {
		progressPercent = 100
	}

	filledWidth := int((progressPercent / 100) * float64(barWidth))

	// Create progress bar
	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filledWidth {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += "]"

	// Format output
	fmt.Printf("%s %.2f%% | %s attempts | %.0f addr/s | Difficulty: %s",
		bar,
		pm.stats.Probability,
		formatNumber(pm.stats.CurrentAttempts),
		pm.stats.Speed,
		formatNumber(int64(pm.stats.Difficulty)),
	)

	if pm.stats.EstimatedTime > 0 {
		fmt.Printf(" | ETA: %s", formatDuration(pm.stats.EstimatedTime))
	}

	// Add thread utilization information
	metrics := pm.statsManager.GetMetrics()
	if metrics.WorkerCount > 1 {
		fmt.Printf(" | 🧵 %d threads", metrics.WorkerCount)
	}
}
