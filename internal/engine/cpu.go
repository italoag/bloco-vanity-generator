package engine

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"hash"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"

	"bloco-vgen/internal/vanity"
	"bloco-vgen/pkg/wallet"
)

var secp256k1Order = ethcrypto.S256().Params().N

type CPUEngine struct{}

type stageTotals struct {
	Entropy time.Duration
	Scalar  time.Duration
	Hash    time.Duration
	Match   time.Duration
}

type workerTotals struct {
	Attempts int64
	Matches  int64
	Stages   stageTotals
}

func NewCPUEngine() *CPUEngine {
	return &CPUEngine{}
}

func (e *CPUEngine) Name() string {
	return NameCPU
}

func (e *CPUEngine) RunBenchmark(ctx context.Context, options BenchmarkOptions, sampleInterval time.Duration, onSample func(Sample)) (*wallet.BenchmarkResult, error) {
	benchmarkCtx, cancel := context.WithTimeout(ctx, options.Duration)
	defer cancel()

	var attempts atomic.Int64
	var matches atomic.Int64
	workerCount := options.ThreadCount
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}
	if options.BatchSize <= 0 {
		options.BatchSize = 1
	}

	startTime := time.Now()
	workerTotalsCh := make(chan workerTotals, workerCount)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	for workerID := 0; workerID < workerCount; workerID++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			workerResult := workerTotals{}
			defer func() { workerTotalsCh <- workerResult }()

			for {
				select {
				case <-benchmarkCtx.Done():
					return
				default:
				}

				for i := 0; i < options.BatchSize; i++ {
					select {
					case <-benchmarkCtx.Done():
						return
					default:
					}

					if !reserveAttempt(&attempts, int64(options.Attempts)) {
						cancel()
						return
					}

					matched, stages, err := runEthereumAttempt(options.Criteria)
					if err != nil {
						select {
						case errCh <- err:
						default:
						}
						cancel()
						return
					}

					workerResult.Attempts++
					workerResult.Stages.Entropy += stages.Entropy
					workerResult.Stages.Scalar += stages.Scalar
					workerResult.Stages.Hash += stages.Hash
					workerResult.Stages.Match += stages.Match

					if matched {
						workerResult.Matches++
						matches.Add(1)
					}
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(workerTotalsCh)
		close(done)
	}()

	ticker := time.NewTicker(sampleInterval)
	defer ticker.Stop()

	var speedSamples []float64
	var durationSamples []time.Duration
	lastAttempts := int64(0)
	lastSample := startTime

	for {
		select {
		case <-done:
			select {
			case err := <-errCh:
				return nil, err
			default:
			}
			result := collectResult(options, workerCount, attempts.Load(), matches.Load(), time.Since(startTime), speedSamples, durationSamples, workerTotalsCh)
			if onSample != nil {
				onSample(Sample{Attempts: result.TotalAttempts, Matches: result.Matches, Speed: result.AverageSpeed, Elapsed: result.TotalDuration, Result: result})
			}
			return result, nil
		case <-ticker.C:
			now := time.Now()
			currentAttempts := attempts.Load()
			elapsed := now.Sub(lastSample)
			if elapsed > 0 {
				speed := float64(currentAttempts-lastAttempts) / elapsed.Seconds()
				speedSamples = append(speedSamples, speed)
				durationSamples = append(durationSamples, elapsed)
				if onSample != nil {
					onSample(Sample{Attempts: currentAttempts, Matches: matches.Load(), Speed: speed, Elapsed: now.Sub(startTime)})
				}
			}
			lastAttempts = currentAttempts
			lastSample = now
		}
	}
}

func reserveAttempt(attempts *atomic.Int64, maxAttempts int64) bool {
	for {
		current := attempts.Load()
		if maxAttempts > 0 && current >= maxAttempts {
			return false
		}
		if attempts.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

func runEthereumAttempt(criteria wallet.GenerationCriteria) (bool, stageTotals, error) {
	addressBytes, stages, err := runEthereumAddressAttempt()
	if err != nil {
		return false, stages, err
	}

	stageStart := time.Now()
	matched := MatchesCriteria(FormatEthereumAddressBytes(addressBytes), criteria)
	stages.Match = time.Since(stageStart)

	return matched, stages, nil
}

func runEthereumAddressAttempt() ([20]byte, stageTotals, error) {
	var empty [20]byte
	publicKey, stages, err := runEthereumPublicKeyAttempt()
	if err != nil {
		return empty, stages, err
	}

	stageStart := time.Now()
	addressBytes := EthereumAddressBytesFromPublicKey(publicKey[:], sha3.NewLegacyKeccak256())
	stages.Hash = time.Since(stageStart)

	return addressBytes, stages, nil
}

func runEthereumPublicKeyAttempt() ([64]byte, stageTotals, error) {
	var stages stageTotals
	var publicKey [64]byte

	privateKey, entropyDuration, err := generateEthereumPrivateKeyAttempt()
	if err != nil {
		zeroBytes(privateKey[:])
		return publicKey, stages, err
	}
	defer zeroBytes(privateKey[:])
	stages.Entropy = entropyDuration

	stageStart := time.Now()
	x, y := ethcrypto.S256().ScalarBaseMult(privateKey[:])
	stages.Scalar = time.Since(stageStart)

	x.FillBytes(publicKey[:32])
	y.FillBytes(publicKey[32:])

	return publicKey, stages, nil
}

func generateEthereumPrivateKeyAttempt() ([32]byte, time.Duration, error) {
	var privateKey [32]byte
	start := time.Now()
	for {
		if _, err := crand.Read(privateKey[:]); err != nil {
			zeroBytes(privateKey[:])
			return privateKey, time.Since(start), err
		}
		if validSecp256k1PrivateKey(privateKey[:]) {
			return privateKey, time.Since(start), nil
		}
	}
}

func validSecp256k1PrivateKey(privateKey []byte) bool {
	if len(privateKey) != 32 {
		return false
	}
	value := new(big.Int).SetBytes(privateKey)
	return value.Sign() > 0 && value.Cmp(secp256k1Order) < 0
}

func zeroBytes(values []byte) {
	for i := range values {
		values[i] = 0
	}
	runtime.KeepAlive(values)
}

func EthereumAddressFromCoordinates(xBytes, yBytes []byte, hasher hash.Hash) string {
	return FormatEthereumAddressBytes(EthereumAddressBytesFromCoordinates(xBytes, yBytes, hasher))
}

func EthereumAddressBytesFromCoordinates(xBytes, yBytes []byte, hasher hash.Hash) [20]byte {
	var publicKey [64]byte
	copy(publicKey[:32], xBytes)
	copy(publicKey[32:], yBytes)

	return EthereumAddressBytesFromPublicKey(publicKey[:], hasher)
}

func EthereumAddressBytesFromPublicKey(publicKey []byte, hasher hash.Hash) [20]byte {
	hasher.Reset()
	hasher.Write(publicKey[:64])
	var hashBytes [32]byte
	hasher.Sum(hashBytes[:0])
	addressBytes := hashBytes[len(hashBytes)-20:]

	var address [20]byte
	copy(address[:], addressBytes)
	return address
}

func FormatEthereumAddressBytes(addressBytes [20]byte) string {
	var encoded [40]byte
	hex.Encode(encoded[:], addressBytes[:])
	return "0x" + string(encoded[:])
}

func MatchesCriteria(address string, criteria wallet.GenerationCriteria) bool {
	return vanity.MatchesGenerationCriteria(address, criteria)
}

func ChecksumAddress(address string) string {
	return vanity.ToChecksumAddress(address)
}

func collectResult(options BenchmarkOptions, workerCount int, totalAttempts int64, totalMatches int64, totalDuration time.Duration, speedSamples []float64, durationSamples []time.Duration, workerTotalsCh <-chan workerTotals) *wallet.BenchmarkResult {
	var stages stageTotals
	for totals := range workerTotalsCh {
		stages.Entropy += totals.Stages.Entropy
		stages.Scalar += totals.Stages.Scalar
		stages.Hash += totals.Stages.Hash
		stages.Match += totals.Stages.Match
	}

	averageSpeed := 0.0
	if totalDuration > 0 {
		averageSpeed = float64(totalAttempts) / totalDuration.Seconds()
	}

	minSpeed, maxSpeed := speedRange(speedSamples)
	singleThreadSpeed := averageSpeed
	if workerCount > 0 {
		singleThreadSpeed = averageSpeed / float64(workerCount)
	}

	return &wallet.BenchmarkResult{
		Engine:                 NameCPU,
		RequestedEngine:        options.RequestedEngine,
		FallbackReason:         options.FallbackReason,
		Network:                options.Network,
		Pattern:                options.Criteria.GetPattern(),
		BatchSize:              options.BatchSize,
		Matches:                totalMatches,
		TotalAttempts:          totalAttempts,
		TotalDuration:          totalDuration,
		AverageSpeed:           averageSpeed,
		CPUThroughput:          averageSpeed,
		MinSpeed:               minSpeed,
		MaxSpeed:               maxSpeed,
		SpeedSamples:           speedSamples,
		DurationSamples:        durationSamples,
		SingleThreadSpeed:      singleThreadSpeed,
		ThreadCount:            workerCount,
		ScalabilityEfficiency:  1,
		ThreadBalanceScore:     1,
		ThreadUtilization:      1,
		SpeedupVsSingleThread:  float64(workerCount),
		EntropyDuration:        stages.Entropy,
		ScalarBaseMultDuration: stages.Scalar,
		HashFormatDuration:     stages.Hash,
		MatchDuration:          stages.Match,
	}
}

func speedRange(samples []float64) (float64, float64) {
	if len(samples) == 0 {
		return 0, 0
	}
	minSpeed := samples[0]
	maxSpeed := samples[0]
	for _, speed := range samples[1:] {
		if speed < minSpeed {
			minSpeed = speed
		}
		if speed > maxSpeed {
			maxSpeed = speed
		}
	}
	return minSpeed, maxSpeed
}

func throughputForDuration(totalAttempts int64, duration time.Duration) float64 {
	if totalAttempts <= 0 || duration <= 0 {
		return 0
	}
	return float64(totalAttempts) / duration.Seconds()
}
