package progress

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"bloco-vgen/internal/worker"
	"bloco-vgen/pkg/utils"
	"bloco-vgen/pkg/wallet"
)

// AggregatedStats holds aggregated statistics from all worker threads
type AggregatedStats struct {
	TotalAttempts int64         `json:"total_attempts"`
	TotalSpeed    float64       `json:"total_speed"`
	AverageSpeed  float64       `json:"average_speed"`
	PeakSpeed     float64       `json:"peak_speed"`
	ActiveWorkers int           `json:"active_workers"`
	Probability   float64       `json:"probability"`
	EstimatedTime time.Duration `json:"estimated_time"`
	LastUpdate    time.Time     `json:"last_update"`
}

// ProgressManager provides thread-safe progress display for multi-threaded operations
type ProgressManager struct {
	mu              sync.RWMutex
	statsCollector  *worker.StatsCollector
	stats           *wallet.GenerationStats
	shutdownChan    chan struct{}
	updateInterval  time.Duration
	lastDisplayTime time.Time
	isActive        int32 // Use atomic for thread-safe access

	// Thread-safe progress aggregation
	aggregatedStats AggregatedStats
}

// NewProgressManager creates a new ProgressManager instance
func NewProgressManager(stats *wallet.GenerationStats, statsCollector *worker.StatsCollector) *ProgressManager {
	return &ProgressManager{
		statsCollector:  statsCollector,
		stats:           stats,
		shutdownChan:    make(chan struct{}),
		updateInterval:  500 * time.Millisecond, // Default update interval
		lastDisplayTime: time.Now(),
		isActive:        0, // 0 = false, 1 = true
		aggregatedStats: AggregatedStats{
			LastUpdate: time.Now(),
		},
	}
}

// Start begins the progress display loop
func (pm *ProgressManager) Start() {
	// Use atomic compare-and-swap to ensure only one goroutine starts
	if !atomic.CompareAndSwapInt32(&pm.isActive, 0, 1) {
		return // Already active
	}

	go pm.displayLoop()
}

// Stop terminates the progress display loop
func (pm *ProgressManager) Stop() {
	// Use atomic compare-and-swap to ensure only one goroutine stops
	if !atomic.CompareAndSwapInt32(&pm.isActive, 1, 0) {
		return // Already stopped
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	close(pm.shutdownChan)

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
	if atomic.LoadInt32(&pm.isActive) == 0 {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.updateProgressDisplay()
}

// displayLoop runs the progress display loop with enhanced thread safety
func (pm *ProgressManager) displayLoop() {
	// Get the initial update interval and shutdown channel under lock
	pm.mu.RLock()
	updateInterval := pm.updateInterval
	shutdownChan := pm.shutdownChan
	pm.mu.RUnlock()

	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if still active using atomic load
			if atomic.LoadInt32(&pm.isActive) == 1 {
				pm.mu.Lock()
				pm.updateProgressDisplay()
				pm.mu.Unlock()
			}

		case <-shutdownChan:
			// Final update before exiting
			if atomic.LoadInt32(&pm.isActive) == 1 {
				pm.mu.Lock()
				pm.updateProgressDisplay()
				pm.mu.Unlock()
			}
			return
		}
	}
}

// updateProgressDisplay updates and displays the progress
func (pm *ProgressManager) updateProgressDisplay() {
	// Aggregate data from all worker threads in a thread-safe manner
	pm.aggregateWorkerData()

	// Update the original statistics object for compatibility
	pm.updateStatisticsObject()

	// Display the progress using aggregated data
	pm.displayProgress()
	pm.lastDisplayTime = time.Now()
}

// aggregateWorkerData safely aggregates data from all worker threads
func (pm *ProgressManager) aggregateWorkerData() {
	// Get current metrics from the stats collector (already thread-safe)
	metrics := pm.statsCollector.GetPerformanceMetrics()

	// Update aggregated stats in a thread-safe manner
	now := time.Now()

	// Create a local copy of aggregated stats to minimize lock time
	aggregated := AggregatedStats{
		TotalAttempts: metrics.TotalAttempts,
		TotalSpeed:    metrics.TotalThroughput,
		ActiveWorkers: metrics.WorkerCount,
		LastUpdate:    now,
	}

	// Calculate average speed based on elapsed time
	elapsed := metrics.ElapsedTime.Seconds()
	if elapsed > 0 {
		aggregated.AverageSpeed = float64(metrics.TotalAttempts) / elapsed
	}

	// Get peak speed from stats collector
	aggregated.PeakSpeed = pm.statsCollector.GetPeakSpeed()

	// Calculate probability based on difficulty and attempts
	aggregated.Probability = utils.CalculateProbability(pm.stats.Difficulty, metrics.TotalAttempts) * 100

	// Calculate estimated time
	if pm.stats.Probability50 > 0 && aggregated.TotalSpeed > 0 {
		remainingAttempts := pm.stats.Probability50 - metrics.TotalAttempts
		if remainingAttempts > 0 {
			aggregated.EstimatedTime = time.Duration(float64(remainingAttempts)/aggregated.TotalSpeed) * time.Second
		} else {
			aggregated.EstimatedTime = 0
		}
	}

	// Update the aggregated stats in one go to minimize race conditions
	pm.aggregatedStats = aggregated
}

// updateStatisticsObject updates the original Statistics object for compatibility
func (pm *ProgressManager) updateStatisticsObject() {
	// Create a local copy of the aggregated stats to minimize lock time
	// This is already thread-safe since we're inside a locked method
	aggregated := pm.aggregatedStats

	// Update the original statistics object with aggregated data
	// This maintains compatibility with existing code
	pm.stats.CurrentAttempts = aggregated.TotalAttempts
	pm.stats.Speed = aggregated.TotalSpeed
	pm.stats.Probability = aggregated.Probability
	pm.stats.EstimatedTime = aggregated.EstimatedTime
	pm.stats.LastUpdate = aggregated.LastUpdate
}

// displayProgress shows a progress bar and statistics using aggregated multi-thread data
func (pm *ProgressManager) displayProgress() {
	// Clear line and move cursor to beginning
	fmt.Print("\r\033[K")

	// Use aggregated stats for thread-safe display
	progressPercent := pm.aggregatedStats.Probability
	if progressPercent > 100 {
		progressPercent = 100
	}

	// Calculate progress bar
	barWidth := 40
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

	// Format output using aggregated data from all threads
	// Maintain exact same format as original for compatibility
	fmt.Printf("%s %.2f%% | %s attempts | %.0f addr/s | Difficulty: %s",
		bar,
		pm.aggregatedStats.Probability,
		utils.FormatLargeNumber(pm.aggregatedStats.TotalAttempts),
		pm.aggregatedStats.TotalSpeed,
		utils.FormatLargeNumber(int64(pm.stats.Difficulty)),
	)

	// Show estimated time if available
	if pm.aggregatedStats.EstimatedTime > 0 {
		fmt.Printf(" | ETA: %s", utils.FormatDuration(pm.aggregatedStats.EstimatedTime))
	}

	// Add multi-thread information
	if pm.aggregatedStats.ActiveWorkers > 1 {
		fmt.Printf(" | %d threads", pm.aggregatedStats.ActiveWorkers)
	}
}

// GetAggregatedStats returns a thread-safe copy of the current aggregated statistics
func (pm *ProgressManager) GetAggregatedStats() AggregatedStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Return a copy to prevent race conditions
	return AggregatedStats{
		TotalAttempts: pm.aggregatedStats.TotalAttempts,
		TotalSpeed:    pm.aggregatedStats.TotalSpeed,
		AverageSpeed:  pm.aggregatedStats.AverageSpeed,
		PeakSpeed:     pm.aggregatedStats.PeakSpeed,
		ActiveWorkers: pm.aggregatedStats.ActiveWorkers,
		Probability:   pm.aggregatedStats.Probability,
		EstimatedTime: pm.aggregatedStats.EstimatedTime,
		LastUpdate:    pm.aggregatedStats.LastUpdate,
	}
}

// IsActive returns whether the progress manager is currently active
func (pm *ProgressManager) IsActive() bool {
	return atomic.LoadInt32(&pm.isActive) == 1
}

// GetLastDisplayTime returns the time of the last progress display update
func (pm *ProgressManager) GetLastDisplayTime() time.Time {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.lastDisplayTime
}

// Manager provides a higher-level interface for progress management
type Manager struct {
	progressManager *ProgressManager
	statsCollector  *worker.StatsCollector
	criteria        wallet.GenerationCriteria
	totalWallets    int
	showProgress    bool
}

// NewManager creates a new progress manager
func NewManager(
	statsCollector *worker.StatsCollector,
	criteria wallet.GenerationCriteria,
	totalWallets int,
	showProgress bool,
) *Manager {
	// Create generation stats
	stats := &wallet.GenerationStats{
		Pattern:       criteria.GetPattern(),
		Difficulty:    utils.CalculateDifficulty(criteria.Prefix, criteria.Suffix, criteria.IsChecksum),
		Probability50: utils.CalculateProbability50(utils.CalculateDifficulty(criteria.Prefix, criteria.Suffix, criteria.IsChecksum)),
		StartTime:     time.Now(),
		IsChecksum:    criteria.IsChecksum,
	}

	progressManager := NewProgressManager(stats, statsCollector)

	return &Manager{
		progressManager: progressManager,
		statsCollector:  statsCollector,
		criteria:        criteria,
		totalWallets:    totalWallets,
		showProgress:    showProgress,
	}
}

// Start begins progress tracking
func (m *Manager) Start(ctx context.Context) {
	if m.showProgress {
		m.progressManager.Start()
	}
}

// Stop ends progress tracking
func (m *Manager) Stop() {
	if m.showProgress {
		m.progressManager.Stop()
		fmt.Printf("\n") // Add newline after progress display
	}
}

// WalletCompleted notifies that a wallet was completed
func (m *Manager) WalletCompleted(result *wallet.GenerationResult) {
	// This method can be used to update progress for multiple wallet generation
	// For now, it's a no-op since the progress is handled by the underlying manager
}
