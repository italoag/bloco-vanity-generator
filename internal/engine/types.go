package engine

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"bloco-vgen/pkg/wallet"
)

const (
	NameAuto  = "auto"
	NameCPU   = "cpu"
	NameMetal = "metal"
)

const (
	MetalValidationFull = "full"
)

const DefaultMetalBatchSize = 5000

type Selection struct {
	Requested      string
	Resolved       string
	FallbackReason string
}

type BenchmarkOptions struct {
	Attempts        int
	Duration        time.Duration
	BatchSize       int
	ThreadCount     int
	Network         string
	Criteria        wallet.GenerationCriteria
	RequestedEngine string
	FallbackReason  string
	MetalValidation string
}

type Sample struct {
	Attempts int64
	Matches  int64
	Speed    float64
	Elapsed  time.Duration
	Result   *wallet.BenchmarkResult
}

type GenerationOptions struct {
	BatchSize       int
	ThreadCount     int
	Network         string
	Criteria        wallet.GenerationCriteria
	RequestedEngine string
	FallbackReason  string
	MetalValidation string
}

type GenerationSample struct {
	Attempts int64
	Speed    float64
	Elapsed  time.Duration
}

type BenchmarkEngine interface {
	Name() string
	RunBenchmark(ctx context.Context, options BenchmarkOptions, sampleInterval time.Duration, onSample func(Sample)) (*wallet.BenchmarkResult, error)
}

type GenerationEngine interface {
	Name() string
	GenerateWallet(ctx context.Context, options GenerationOptions, sampleInterval time.Duration, onSample func(GenerationSample)) (*wallet.GenerationResult, error)
}

func Resolve(requested string) (Selection, error) {
	requested = strings.ToLower(strings.TrimSpace(requested))
	if requested == "" {
		requested = NameAuto
	}

	switch requested {
	case NameCPU:
		return Selection{Requested: requested, Resolved: NameCPU}, nil
	case NameAuto:
		selection := Selection{Requested: requested, Resolved: NameCPU}
		if MetalAvailable() {
			selection.FallbackReason = "metal backend is benchmark-only; using cpu"
		} else if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			selection.FallbackReason = MetalUnavailableReason()
		}
		return selection, nil
	case NameMetal:
		if MetalAvailable() {
			return Selection{Requested: requested, Resolved: NameMetal}, nil
		}
		return Selection{}, fmt.Errorf("metal engine is not available yet; use --engine cpu or --engine auto")
	default:
		return Selection{}, fmt.Errorf("unsupported engine %q (expected auto, cpu, or metal)", requested)
	}
}

func ResolveGeneration(requested string, criteria wallet.GenerationCriteria) (Selection, error) {
	requested = strings.ToLower(strings.TrimSpace(requested))
	if requested == "" {
		requested = NameAuto
	}

	supportErr := ValidateMetalGenerationCriteria(criteria)
	switch requested {
	case NameCPU:
		return Selection{Requested: requested, Resolved: NameCPU}, nil
	case NameAuto:
		// The CPU engine (chained point addition) is currently 10x+ faster
		// than the Metal hybrid on Apple Silicon, so auto prefers CPU for
		// generation. Metal remains opt-in via --engine metal until the
		// kernel and validation pipeline catch up.
		if supportErr != nil {
			return Selection{Requested: requested, Resolved: NameCPU, FallbackReason: "metal generation unsupported: " + supportErr.Error()}, nil
		}
		if MetalAvailable() {
			return Selection{Requested: requested, Resolved: NameCPU, FallbackReason: "cpu is faster than the metal hybrid for generation; use --engine metal to force"}, nil
		}
		return Selection{Requested: requested, Resolved: NameCPU, FallbackReason: MetalUnavailableReason()}, nil
	case NameMetal:
		if supportErr != nil {
			return Selection{}, fmt.Errorf("metal generation unsupported: %w", supportErr)
		}
		if MetalAvailable() {
			return Selection{Requested: requested, Resolved: NameMetal}, nil
		}
		return Selection{}, fmt.Errorf("metal engine is not available yet; use --engine cpu or --engine auto")
	default:
		return Selection{}, fmt.Errorf("unsupported engine %q (expected auto, cpu, or metal)", requested)
	}
}

func NewBenchmarkEngine(name string) (BenchmarkEngine, error) {
	switch name {
	case NameCPU:
		return NewCPUEngine(), nil
	case NameMetal:
		return NewMetalEngine()
	default:
		return nil, fmt.Errorf("unsupported resolved engine %q", name)
	}
}

func NewGenerationEngine(name string) (GenerationEngine, error) {
	switch name {
	case NameMetal:
		benchmarkEngine, err := NewMetalEngine()
		if err != nil {
			return nil, err
		}
		generationEngine, ok := benchmarkEngine.(GenerationEngine)
		if !ok {
			if closer, closeOK := benchmarkEngine.(interface{ Close() }); closeOK {
				closer.Close()
			}
			return nil, fmt.Errorf("resolved engine %q does not support generation", name)
		}
		return generationEngine, nil
	default:
		return nil, fmt.Errorf("unsupported generation engine %q", name)
	}
}

func NormalizeMetalValidationMode(mode string) (string, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		return MetalValidationFull, nil
	}

	if mode == MetalValidationFull {
		return mode, nil
	}
	return "", fmt.Errorf("unsupported metal validation mode %q (Phase 4 requires full CPU verification)", mode)
}

func ValidateMetalGenerationCriteria(criteria wallet.GenerationCriteria) error {
	network := strings.ToLower(strings.TrimSpace(criteria.Network))
	if network != "" && network != "ethereum" {
		return fmt.Errorf("metal generation supports ethereum only")
	}
	if criteria.UseMnemonic {
		return fmt.Errorf("metal generation does not support mnemonic generation")
	}
	if _, err := patternNibbles(criteria.Prefix); err != nil {
		return err
	}
	if _, err := patternNibbles(criteria.Suffix); err != nil {
		return err
	}
	return nil
}
