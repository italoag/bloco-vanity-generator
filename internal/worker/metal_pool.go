package worker

import (
	"context"
	"fmt"
	"time"

	"bloco-vgen/internal/config"
	"bloco-vgen/internal/engine"
	"bloco-vgen/pkg/errors"
	"bloco-vgen/pkg/logging"
	"bloco-vgen/pkg/wallet"
)

type MetalPool struct {
	threadCount    int
	logger         logging.SecureLogger
	statsCollector *StatsCollector
	statsChan      chan WorkerStats
	statsCtx       context.Context
	statsCancel    context.CancelFunc
	engine         engine.GenerationEngine
	options        engine.GenerationOptions
}

func NewMetalPoolWithConfig(threadCount int, cfg *config.Config, options engine.GenerationOptions) (*MetalPool, error) {
	if threadCount <= 0 {
		threadCount = 1
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	logConfig, err := createLogConfigFromAppConfig(cfg)
	if err != nil {
		fmt.Printf("Warning: Failed to create log configuration: %v\n", err)
		logConfig = &logging.LogConfig{Enabled: false}
	}

	logger, err := logging.NewSecureLogger(logConfig)
	if err != nil {
		fmt.Printf("Warning: Failed to create secure logger: %v\n", err)
		logger, _ = logging.NewSecureLogger(&logging.LogConfig{Enabled: false})
	}

	generationEngine, err := engine.NewGenerationEngine(engine.NameMetal)
	if err != nil {
		if logger != nil {
			_ = logger.Close()
		}
		return nil, err
	}

	statsCtx, statsCancel := context.WithCancel(context.Background())
	return &MetalPool{
		threadCount:    threadCount,
		logger:         logger,
		statsCollector: NewStatsCollector(),
		statsChan:      make(chan WorkerStats, threadCount*2),
		statsCtx:       statsCtx,
		statsCancel:    statsCancel,
		engine:         generationEngine,
		options:        options,
	}, nil
}

func (p *MetalPool) Start() error {
	if p.statsCollector != nil && p.statsChan != nil {
		p.statsCollector.Start(p.statsChan, p.statsCtx)
	}
	return nil
}

func (p *MetalPool) Shutdown() error {
	if p.statsCancel != nil {
		p.statsCancel()
	}
	if closer, ok := p.engine.(interface{ Close() }); ok {
		closer.Close()
	}
	if p.logger != nil {
		if err := p.logger.Close(); err != nil {
			fmt.Printf("Warning: Failed to close secure logger: %v\n", err)
		}
	}
	return nil
}

func (p *MetalPool) GenerateWalletWithContext(ctx context.Context, criteria wallet.GenerationCriteria) (*wallet.GenerationResult, error) {
	if p.logger != nil {
		params := map[string]interface{}{
			"prefix":           criteria.Prefix,
			"suffix":           criteria.Suffix,
			"checksum":         criteria.IsChecksum,
			"engine":           engine.NameMetal,
			"batch_size":       p.options.BatchSize,
			"use_mnemonic":     criteria.UseMnemonic,
			"metal_validation": p.options.MetalValidation,
		}
		if err := p.logger.LogOperationStart("wallet_generation", params); err != nil {
			fmt.Printf("Warning: Failed to log operation start: %v\n", err)
		}
	}

	options := p.options
	options.Criteria = criteria
	options.Network = criteria.Network
	options.ThreadCount = p.threadCount

	result, err := p.engine.GenerateWallet(ctx, options, statsUpdateInterval, func(sample engine.GenerationSample) {
		select {
		case p.statsChan <- WorkerStats{
			WorkerID:   0,
			Attempts:   sample.Attempts,
			Speed:      sample.Speed,
			LastUpdate: time.Now(),
			IsHealthy:  true,
			ErrorCount: 0,
		}:
		default:
		}
	})
	if err != nil {
		if ctx.Err() != nil {
			err = errors.NewCancellationError("generate_wallet", "generation cancelled")
		}
		if p.logger != nil {
			context := map[string]interface{}{
				"engine":     engine.NameMetal,
				"batch_size": p.options.BatchSize,
			}
			if logErr := p.logger.LogError("wallet_generation", err, context); logErr != nil {
				fmt.Printf("Warning: Failed to log error: %v\n", logErr)
			}
		}
		return nil, err
	}

	if p.logger != nil {
		if err := p.logger.LogWalletGenerated(result.Wallet.Address, int(result.Attempts), result.Duration, result.WorkerID); err != nil {
			fmt.Printf("Warning: Failed to log wallet: %v\n", err)
		}
		stats := logging.OperationStats{
			Duration:   result.Duration,
			Success:    true,
			ItemsCount: 1,
			ErrorCount: 0,
		}
		if result.Duration > 0 {
			stats.ThroughputPS = float64(result.Attempts) / result.Duration.Seconds()
		}
		if err := p.logger.LogOperationComplete("wallet_generation", stats); err != nil {
			fmt.Printf("Warning: Failed to log operation completion: %v\n", err)
		}
	}
	return result, nil
}

func (p *MetalPool) GetStatsCollector() *StatsCollector {
	return p.statsCollector
}

var _ WorkerPool = (*MetalPool)(nil)
