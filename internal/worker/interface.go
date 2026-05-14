package worker

import (
	"bloco-vgen/pkg/wallet"
	"context"
)

// WorkerPool interface defines the common interface for all worker pool implementations
type WorkerPool interface {
	// Start starts the worker pool
	Start() error

	// Shutdown gracefully shuts down the worker pool
	Shutdown() error

	// GenerateWalletWithContext generates a single wallet with the given criteria
	GenerateWalletWithContext(ctx context.Context, criteria wallet.GenerationCriteria) (*wallet.GenerationResult, error)

	// GetStatsCollector returns the statistics collector
	GetStatsCollector() *StatsCollector
}

// Ensure all implementations satisfy the interface
var (
	_ WorkerPool = (*Pool)(nil)
)
