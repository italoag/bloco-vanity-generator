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

type BenchmarkEngine interface {
	Name() string
	RunBenchmark(ctx context.Context, options BenchmarkOptions, sampleInterval time.Duration, onSample func(Sample)) (*wallet.BenchmarkResult, error)
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
