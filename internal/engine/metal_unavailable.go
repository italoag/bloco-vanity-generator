//go:build !darwin || !arm64 || !cgo

package engine

import "fmt"

func MetalAvailable() bool {
	return false
}

func MetalUnavailableReason() string {
	return "metal backend is not available for this build; using cpu"
}

func MetalDeviceName() string {
	return ""
}

func NewMetalEngine() (BenchmarkEngine, error) {
	return nil, fmt.Errorf("metal engine is not available yet; use --engine cpu or --engine auto")
}
