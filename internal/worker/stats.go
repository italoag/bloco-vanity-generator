package worker

import (
	"context"
	"math"
	"sync"
	"time"

	"bloco-vgen/pkg/wallet"
)

// WorkerStats holds statistics for individual workers
type WorkerStats struct {
	WorkerID   int       `json:"worker_id"`
	Attempts   int64     `json:"attempts"`
	Speed      float64   `json:"speed"`
	LastUpdate time.Time `json:"last_update"`
	IsHealthy  bool      `json:"is_healthy"`
	ErrorCount int       `json:"error_count"`
	LastError  string    `json:"last_error,omitempty"`
}

// WorkerHealth represents health information for a worker
type WorkerHealth struct {
	WorkerID      int           `json:"worker_id"`
	IsHealthy     bool          `json:"is_healthy"`
	LastHeartbeat time.Time     `json:"last_heartbeat"`
	ErrorCount    int           `json:"error_count"`
	LastError     string        `json:"last_error,omitempty"`
	Uptime        time.Duration `json:"uptime"`
}

// WorkItem represents a unit of work for the worker pool
type WorkItem struct {
	Criteria  wallet.GenerationCriteria `json:"criteria"`
	BatchSize int                       `json:"batch_size"`
	ID        string                    `json:"id"`
}

// WorkResult represents the result of processing a work item
type WorkResult struct {
	Result   *wallet.GenerationResult `json:"result"`
	WorkerID int                      `json:"worker_id"`
	ItemID   string                   `json:"item_id"`
	Found    bool                     `json:"found"`
}

// StatsCollector collects and aggregates statistics from multiple workers
type StatsCollector struct {
	mu              sync.RWMutex
	workerStats     map[int]WorkerStats
	aggregatedStats AggregatedStats
	startTime       time.Time
	lastUpdate      time.Time
	peakSpeed       float64
	speedHistory    []SpeedSample
	maxHistorySize  int
}

// AggregatedStats holds aggregated statistics from all workers
type AggregatedStats struct {
	TotalAttempts    int64         `json:"total_attempts"`
	TotalSpeed       float64       `json:"total_speed"`
	AverageSpeed     float64       `json:"average_speed"`
	PeakSpeed        float64       `json:"peak_speed"`
	ActiveWorkers    int           `json:"active_workers"`
	HealthyWorkers   int           `json:"healthy_workers"`
	TotalErrors      int           `json:"total_errors"`
	ElapsedTime      time.Duration `json:"elapsed_time"`
	LastUpdate       time.Time     `json:"last_update"`
	ThreadEfficiency float64       `json:"thread_efficiency"`
	ThreadBalance    float64       `json:"thread_balance"`
	SpeedVariance    float64       `json:"speed_variance"`
}

// SpeedSample represents a speed measurement at a specific time
type SpeedSample struct {
	Speed     float64   `json:"speed"`
	Timestamp time.Time `json:"timestamp"`
}

// PerformanceMetrics contains detailed performance analysis
type PerformanceMetrics struct {
	ThreadUtilization          map[int]float64 `json:"thread_utilization"`
	TotalThroughput            float64         `json:"total_throughput"`
	PerThreadSpeed             map[int]float64 `json:"per_thread_speed"`
	EfficiencyRatio            float64         `json:"efficiency_ratio"`
	TotalAttempts              int64           `json:"total_attempts"`
	ElapsedTime                time.Duration   `json:"elapsed_time"`
	WorkerCount                int             `json:"worker_count"`
	ThreadEfficiency           float64         `json:"thread_efficiency"`
	CPUUtilization             float64         `json:"cpu_utilization"`
	SpeedupVsSingleThread      float64         `json:"speedup_vs_single_thread"`
	ThreadBalanceScore         float64         `json:"thread_balance_score"`
	EstimatedSingleThreadSpeed float64         `json:"estimated_single_thread_speed"`
}

// NewStatsCollector creates a new statistics collector
func NewStatsCollector() *StatsCollector {
	return &StatsCollector{
		workerStats:    make(map[int]WorkerStats),
		startTime:      time.Now(),
		lastUpdate:     time.Now(),
		speedHistory:   make([]SpeedSample, 0),
		maxHistorySize: 1000, // Keep last 1000 speed samples
		aggregatedStats: AggregatedStats{
			LastUpdate: time.Now(),
		},
	}
}

// Start begins collecting statistics from workers
func (sc *StatsCollector) Start(statsChan <-chan WorkerStats, ctx context.Context) {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case stats := <-statsChan:
				sc.UpdateWorkerStats(stats)
			case <-ticker.C:
				sc.recalculateAggregatedStats()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// UpdateWorkerStats updates statistics for a specific worker
func (sc *StatsCollector) UpdateWorkerStats(stats WorkerStats) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.workerStats[stats.WorkerID] = stats
	sc.lastUpdate = time.Now()

	// Recalculate aggregated stats
	sc.recalculateAggregatedStatsUnsafe()
}

// GetAggregatedStats returns the current aggregated statistics
func (sc *StatsCollector) GetAggregatedStats() AggregatedStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.aggregatedStats
}

// GetWorkerStats returns statistics for all workers
func (sc *StatsCollector) GetWorkerStats() map[int]WorkerStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Return a copy to prevent race conditions
	statsCopy := make(map[int]WorkerStats, len(sc.workerStats))
	for id, stats := range sc.workerStats {
		statsCopy[id] = stats
	}
	return statsCopy
}

// GetPerformanceMetrics returns detailed performance metrics
func (sc *StatsCollector) GetPerformanceMetrics() PerformanceMetrics {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Calculate thread utilization
	threadUtil := make(map[int]float64)
	perThreadSpeed := make(map[int]float64)
	totalSpeed := 0.0

	for id, stats := range sc.workerStats {
		perThreadSpeed[id] = stats.Speed
		totalSpeed += stats.Speed
	}

	// Calculate utilization relative to total speed
	if totalSpeed > 0 {
		for id, speed := range perThreadSpeed {
			threadUtil[id] = speed / totalSpeed
		}
	}

	// Calculate efficiency metrics
	workerCount := len(sc.workerStats)
	var efficiencyRatio float64 = 0
	var estimatedSingleThreadSpeed float64 = 0
	var speedupVsSingleThread float64 = 0

	if workerCount > 0 {
		// Estimate single-thread speed as average per worker
		estimatedSingleThreadSpeed = totalSpeed / float64(workerCount)

		// Calculate efficiency ratio (actual vs theoretical linear scaling)
		if estimatedSingleThreadSpeed > 0 {
			theoreticalSpeed := estimatedSingleThreadSpeed * float64(workerCount)
			efficiencyRatio = totalSpeed / theoreticalSpeed
			speedupVsSingleThread = totalSpeed / estimatedSingleThreadSpeed
		}
	}

	// Calculate thread balance score
	threadBalanceScore := sc.calculateThreadBalanceScore()

	// Estimate CPU utilization (simplified)
	cpuUtilization := math.Min(efficiencyRatio, 1.0)

	return PerformanceMetrics{
		ThreadUtilization:          threadUtil,
		TotalThroughput:            totalSpeed,
		PerThreadSpeed:             perThreadSpeed,
		EfficiencyRatio:            efficiencyRatio,
		TotalAttempts:              sc.aggregatedStats.TotalAttempts,
		ElapsedTime:                time.Since(sc.startTime),
		WorkerCount:                workerCount,
		ThreadEfficiency:           efficiencyRatio,
		CPUUtilization:             cpuUtilization,
		SpeedupVsSingleThread:      speedupVsSingleThread,
		ThreadBalanceScore:         threadBalanceScore,
		EstimatedSingleThreadSpeed: estimatedSingleThreadSpeed,
	}
}

// GetTotalAttempts returns the total number of attempts across all workers
func (sc *StatsCollector) GetTotalAttempts() int64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.aggregatedStats.TotalAttempts
}

// GetTotalSpeed returns the combined speed of all workers
func (sc *StatsCollector) GetTotalSpeed() float64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.aggregatedStats.TotalSpeed
}

// GetPeakSpeed returns the highest observed speed
func (sc *StatsCollector) GetPeakSpeed() float64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.peakSpeed
}

// GetWorkerCount returns the number of active workers
func (sc *StatsCollector) GetWorkerCount() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.workerStats)
}

// GetElapsedTime returns the time elapsed since collection started
func (sc *StatsCollector) GetElapsedTime() time.Duration {
	return time.Since(sc.startTime)
}

// Reset resets all statistics
func (sc *StatsCollector) Reset() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.workerStats = make(map[int]WorkerStats)
	sc.startTime = time.Now()
	sc.lastUpdate = time.Now()
	sc.peakSpeed = 0
	sc.speedHistory = sc.speedHistory[:0]
	sc.aggregatedStats = AggregatedStats{
		LastUpdate: time.Now(),
	}
}

// recalculateAggregatedStats recalculates aggregated statistics (thread-safe)
func (sc *StatsCollector) recalculateAggregatedStats() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.recalculateAggregatedStatsUnsafe()
}

// recalculateAggregatedStatsUnsafe recalculates aggregated statistics (not thread-safe)
func (sc *StatsCollector) recalculateAggregatedStatsUnsafe() {
	var totalAttempts int64
	var totalSpeed float64
	var activeWorkers int
	var healthyWorkers int
	var totalErrors int

	for _, stats := range sc.workerStats {
		totalAttempts += stats.Attempts
		totalSpeed += stats.Speed
		activeWorkers++

		if stats.IsHealthy {
			healthyWorkers++
		}

		totalErrors += stats.ErrorCount
	}

	// Update peak speed
	if totalSpeed > sc.peakSpeed {
		sc.peakSpeed = totalSpeed
	}

	// Add to speed history
	now := time.Now()
	sc.speedHistory = append(sc.speedHistory, SpeedSample{
		Speed:     totalSpeed,
		Timestamp: now,
	})

	// Trim speed history if too large
	if len(sc.speedHistory) > sc.maxHistorySize {
		sc.speedHistory = sc.speedHistory[len(sc.speedHistory)-sc.maxHistorySize:]
	}

	// Calculate average speed
	elapsed := now.Sub(sc.startTime)
	var averageSpeed float64
	if elapsed.Seconds() > 0 {
		averageSpeed = float64(totalAttempts) / elapsed.Seconds()
	}

	// Calculate thread efficiency and balance
	threadEfficiency := sc.calculateThreadEfficiency(totalSpeed, activeWorkers)
	threadBalance := sc.calculateThreadBalanceScore()

	// Update aggregated stats
	sc.aggregatedStats = AggregatedStats{
		TotalAttempts:    totalAttempts,
		TotalSpeed:       totalSpeed,
		AverageSpeed:     averageSpeed,
		PeakSpeed:        sc.peakSpeed,
		ActiveWorkers:    activeWorkers,
		HealthyWorkers:   healthyWorkers,
		TotalErrors:      totalErrors,
		ElapsedTime:      elapsed,
		LastUpdate:       now,
		ThreadEfficiency: threadEfficiency,
		ThreadBalance:    threadBalance,
		SpeedVariance:    sc.calculateSpeedVariance(),
	}
}

// calculateThreadEfficiency calculates thread efficiency
func (sc *StatsCollector) calculateThreadEfficiency(totalSpeed float64, workerCount int) float64 {
	if workerCount == 0 {
		return 0
	}

	// Estimate single-thread speed
	estimatedSingleThreadSpeed := totalSpeed / float64(workerCount)
	if estimatedSingleThreadSpeed == 0 {
		return 0
	}

	// Calculate efficiency as actual speedup vs theoretical linear speedup
	theoreticalSpeed := estimatedSingleThreadSpeed * float64(workerCount)
	return totalSpeed / theoreticalSpeed
}

// calculateThreadBalanceScore calculates how evenly work is distributed
func (sc *StatsCollector) calculateThreadBalanceScore() float64 {
	if len(sc.workerStats) <= 1 {
		return 1.0 // Perfect balance with 0 or 1 worker
	}

	// Calculate coefficient of variation for worker speeds
	var speeds []float64
	var sum float64

	for _, stats := range sc.workerStats {
		speeds = append(speeds, stats.Speed)
		sum += stats.Speed
	}

	if sum == 0 {
		return 1.0 // No work done yet, assume perfect balance
	}

	mean := sum / float64(len(speeds))
	var sumSquaredDiff float64

	for _, speed := range speeds {
		diff := speed - mean
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(len(speeds))
	stdDev := math.Sqrt(variance)

	// Calculate coefficient of variation
	coeffVar := stdDev / mean

	// Convert to balance score (0-1, where 1 is perfect balance)
	return 1.0 - math.Min(coeffVar, 1.0)
}

// calculateSpeedVariance calculates the variance in speed over time
func (sc *StatsCollector) calculateSpeedVariance() float64 {
	if len(sc.speedHistory) < 2 {
		return 0
	}

	// Calculate variance of recent speed samples
	var sum float64
	recentSamples := sc.speedHistory
	if len(recentSamples) > 100 {
		recentSamples = recentSamples[len(recentSamples)-100:] // Last 100 samples
	}

	for _, sample := range recentSamples {
		sum += sample.Speed
	}

	mean := sum / float64(len(recentSamples))
	var sumSquaredDiff float64

	for _, sample := range recentSamples {
		diff := sample.Speed - mean
		sumSquaredDiff += diff * diff
	}

	return sumSquaredDiff / float64(len(recentSamples))
}

// GetSpeedHistory returns the speed history
func (sc *StatsCollector) GetSpeedHistory() []SpeedSample {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Return a copy
	history := make([]SpeedSample, len(sc.speedHistory))
	copy(history, sc.speedHistory)
	return history
}

// GetHealthySummary returns a summary of worker health
func (sc *StatsCollector) GetHealthySummary() (healthy, total int) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	total = len(sc.workerStats)
	for _, stats := range sc.workerStats {
		if stats.IsHealthy {
			healthy++
		}
	}
	return healthy, total
}
