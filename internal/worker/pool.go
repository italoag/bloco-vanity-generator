package worker

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	bip32 "github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"

	"bloco-vgen/internal/config"
	"bloco-vgen/internal/crypto"
	"bloco-vgen/internal/vanity"
	"bloco-vgen/pkg/errors"
	"bloco-vgen/pkg/logging"
	"bloco-vgen/pkg/wallet"
)

// Pool is a minimal implementation that mimics the working monolithic version
type Pool struct {
	threadCount    int
	mu             sync.RWMutex
	isRunning      bool
	logger         logging.SecureLogger
	statsCollector *StatsCollector
	statsChan      chan WorkerStats
	statsCtx       context.Context
	statsCancel    context.CancelFunc
	poolManager    *crypto.PoolManager
	generator      crypto.Generator
	chains         []*crypto.PrivateKeyChain // reused across generation calls
	chainsMu       sync.Mutex
}

const (
	// statsUpdateInterval is the frequency of stats updates from workers
	statsUpdateInterval = 100 * time.Millisecond
	// statsUpdateAttempts is the number of attempts between stats updates
	statsUpdateAttempts = 1000
)

// NewPool creates a new worker pool
func NewPool(threadCount int, network string) *Pool {
	return NewPoolWithConfig(threadCount, nil, network)
}

// NewPoolWithConfig creates a new worker pool with configuration
func NewPoolWithConfig(threadCount int, cfg *config.Config, network string) *Pool {
	// Validate threadCount
	if threadCount < 0 {
		threadCount = 1 // Default to 1 thread for negative values
	}
	if threadCount == 0 {
		threadCount = 1 // Default to 1 thread for zero
	}

	// Use default config if none provided
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Create SecureLogger from configuration
	logConfig, err := createLogConfigFromAppConfig(cfg)
	if err != nil {
		// Log error but continue with disabled logging
		fmt.Printf("Warning: Failed to create log configuration: %v\n", err)
		// Create a disabled logger as fallback
		logConfig = &logging.LogConfig{Enabled: false}
	}

	logger, err := logging.NewSecureLogger(logConfig)
	if err != nil {
		// Log error but continue with disabled logging
		fmt.Printf("Warning: Failed to create secure logger: %v\n", err)
		// Create a disabled logger as fallback
		logger, _ = logging.NewSecureLogger(&logging.LogConfig{Enabled: false})
	}

	statsCollector := NewStatsCollector()
	statsChan := make(chan WorkerStats, threadCount*2) // Buffered channel

	// Create context for stats collection
	statsCtx, statsCancel := context.WithCancel(context.Background())

	// Create pool manager and generator
	poolConfig := crypto.DefaultPoolConfig()
	poolManager := crypto.NewPoolManager(poolConfig)

	var generator crypto.Generator
	switch strings.ToLower(network) {
	case "bitcoin":
		generator = crypto.NewBitcoinGenerator(poolManager)
	case "solana":
		generator = crypto.NewSolanaGenerator(poolManager)
	default:
		generator = crypto.NewEthereumGenerator(poolManager)
	}

	return &Pool{
		threadCount:    threadCount,
		isRunning:      false,
		logger:         logger,
		statsCollector: statsCollector,
		statsChan:      statsChan,
		statsCtx:       statsCtx,
		statsCancel:    statsCancel,
		poolManager:    poolManager,
		generator:      generator,
	}
}

// Start starts the worker pool
func (p *Pool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.isRunning = true

	// Start stats collection
	if p.statsCollector != nil && p.statsChan != nil {
		p.statsCollector.Start(p.statsChan, p.statsCtx)
	}

	return nil
}

// Shutdown shuts down the worker pool
func (p *Pool) Shutdown() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.isRunning = false

	// Cancel stats context
	if p.statsCancel != nil {
		p.statsCancel()
	}

	// Close the logger if it exists
	if p.logger != nil {
		if err := p.logger.Close(); err != nil {
			fmt.Printf("Warning: Failed to close secure logger: %v\n", err)
		}
	}

	return nil
}

// GetStatsCollector returns the stats collector
func (p *Pool) GetStatsCollector() *StatsCollector {
	return p.statsCollector
}

// getChain returns the chained key generator for a worker, creating it on
// first use. Chains are reused across GenerateWalletWithContext calls so
// batch generation does not rebuild the (expensive) point chain per wallet.
func (p *Pool) getChain(workerID int) (*crypto.PrivateKeyChain, error) {
	p.chainsMu.Lock()
	defer p.chainsMu.Unlock()
	if workerID < len(p.chains) && p.chains[workerID] != nil {
		return p.chains[workerID], nil
	}
	for len(p.chains) <= workerID {
		p.chains = append(p.chains, nil)
	}
	chain, err := crypto.NewPrivateKeyChain()
	if err != nil {
		return nil, err
	}
	p.chains[workerID] = chain
	return chain, nil
}

// GenerateWalletWithContext generates a wallet using the worker pool
func (p *Pool) GenerateWalletWithContext(ctx context.Context, criteria wallet.GenerationCriteria) (*wallet.GenerationResult, error) {
	// Log operation start
	if p.logger != nil {
		params := map[string]interface{}{
			"prefix":       criteria.Prefix,
			"suffix":       criteria.Suffix,
			"checksum":     criteria.IsChecksum,
			"threads":      p.threadCount,
			"use_mnemonic": criteria.UseMnemonic,
		}
		if err := p.logger.LogOperationStart("wallet_generation", params); err != nil {
			fmt.Printf("Warning: Failed to log operation start: %v\n", err)
		}
	}

	resultCh := make(chan *wallet.GenerationResult, 1)
	errorCh := make(chan error, 1)

	// Start workers similar to monolithic version
	var wg sync.WaitGroup
	for i := 0; i < p.threadCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Worker loop
			attempts := int64(0)
			startTime := time.Now()
			lastStatsUpdate := startTime
			// Per-worker chained key generator: state persists across loop
			// iterations (each chain produces ChainBatchSize keys).
			var chain *crypto.PrivateKeyChain

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				attempts++

				// Send stats update every 100ms or 1000 attempts
				now := time.Now()
				if now.Sub(lastStatsUpdate) >= statsUpdateInterval || attempts%statsUpdateAttempts == 0 {
					elapsed := now.Sub(startTime)
					var speed float64
					if elapsed.Seconds() > 0 {
						speed = float64(attempts) / elapsed.Seconds()
					}

					// Send stats to collector
					select {
					case p.statsChan <- WorkerStats{
						WorkerID:   workerID,
						Attempts:   attempts,
						Speed:      speed,
						LastUpdate: now,
						IsHealthy:  true,
						ErrorCount: 0,
					}:
					default:
						// Non-blocking send
					}
					lastStatsUpdate = now
				}

				// Generate private key material based on generation strategy
				var (
					privateKey *ecdsa.PrivateKey
					mnemonic   string
					err        error
					addressStr string
				)

				// Mnemonic generation is only supported for Ethereum
				// Bitcoin and Solana use different key derivation schemes (secp256k1 and Ed25519)
				if criteria.UseMnemonic && criteria.Network != "ethereum" && criteria.Network != "" {
					if p.logger != nil {
						context := map[string]interface{}{
							"worker_id": workerID,
							"network":   criteria.Network,
						}
						if logErr := p.logger.LogError("unsupported_mnemonic_network",
							fmt.Errorf("mnemonic generation is only supported for Ethereum network"), context); logErr != nil {
							_ = logErr
						}
					}
					// Skip mnemonic generation for non-Ethereum networks
					criteria.UseMnemonic = false
				}

				// For non-Ethereum networks, use the generator directly
				if criteria.Network != "ethereum" && criteria.Network != "" {
					// Use the generator to create a complete wallet
					genWallet, err := p.generator.GenerateWallet()
					if err != nil {
						if p.logger != nil {
							context := map[string]interface{}{
								"worker_id": workerID,
								"attempts":  attempts,
								"network":   criteria.Network,
							}
							if logErr := p.logger.LogError("wallet_generation", err, context); logErr != nil {
								_ = logErr
							}
						}
						continue
					}

					addressStr = genWallet.Address

					// Check if address matches criteria
					if !matchesCriteria(addressStr, criteria.Prefix, criteria.Suffix, criteria.IsChecksum, criteria.Network, criteria.CaseSensitive) {
						continue
					}

					// Found a match! Create result directly from generated wallet
					result := &wallet.GenerationResult{
						Wallet: &wallet.Wallet{
							Address:    addressStr,
							PublicKey:  genWallet.PublicKey,
							PrivateKey: genWallet.PrivateKey,
							Mnemonic:   genWallet.Mnemonic, // Include mnemonic from generator
							Network:    criteria.Network,
							CreatedAt:  time.Now(),
						},
						Attempts: attempts,
						Duration: time.Since(startTime),
						WorkerID: workerID,
					}

					select {
					case resultCh <- result:
					case <-ctx.Done():
					}
					return
				}

				if criteria.UseMnemonic {
					mnemonic, privateKey, err = generateMnemonicPrivateKey()
					if err != nil {
						if p.logger != nil {
							context := map[string]interface{}{
								"worker_id":      workerID,
								"attempts":       attempts,
								"use_mnemonic":   true,
								"error_category": "mnemonic_generation",
							}
							if logErr := p.logger.LogError("wallet_material_generation", err, context); logErr != nil {
								_ = logErr
							}
						}
						continue
					}

					// Use network-specific generator to create address from private key
					privateKeyBytes := ethcrypto.FromECDSA(privateKey)
					addressStr, err = p.generator.GenerateAddressFromPrivateKey(privateKeyBytes)
					if err != nil {
						if p.logger != nil {
							context := map[string]interface{}{
								"worker_id": workerID,
								"attempts":  attempts,
							}
							if logErr := p.logger.LogError("address_generation", err, context); logErr != nil {
								_ = logErr
							}
						}
						continue
					}

				} else {
					// Fast path using the chained private key generator: each
					// worker derives keys from a random seed as k0, k0+1, ...
					// with incremental point addition (P + G) and a single
					// batched field inversion per batch. This is much faster
					// than one full scalar multiplication per key.
					if chain == nil {
						chain, err = p.getChain(workerID)
						if err != nil {
							if p.logger != nil {
								context := map[string]interface{}{
									"worker_id": workerID,
									"attempts":  attempts,
								}
								if logErr := p.logger.LogError("crypto_key_generation", err, context); logErr != nil {
									_ = logErr
								}
							}
							continue
						}
					}

					var keyBytes [32]byte
					var pub [64]byte
					keyBytes, pub, err = chain.NextKey()
					if err != nil {
						if p.logger != nil {
							context := map[string]interface{}{
								"worker_id": workerID,
								"attempts":  attempts,
							}
							if logErr := p.logger.LogError("crypto_key_generation", err, context); logErr != nil {
								_ = logErr
							}
						}
						continue
					}

					// Match against the pattern using raw address bytes for
					// case-insensitive patterns; EIP-55 checksum patterns still
					// need the checksummed string.
					addressBytes := crypto.AddressFromPublicKey(&pub)
					rawAddress := formatEthereumAddressBytes(addressBytes)
					if criteria.IsChecksum {
						addressStr = toChecksumAddress(rawAddress)
						if !matchesCriteria(addressStr, criteria.Prefix, criteria.Suffix, criteria.IsChecksum, criteria.Network, criteria.CaseSensitive) {
							continue
						}
					} else {
						prefixNibbles, suffixNibbles := crypto.NibblePatterns(criteria.Prefix, criteria.Suffix)
						if (criteria.Prefix != "" || criteria.Suffix != "") && prefixNibbles == nil {
							// Invalid pattern: fall back to the string matcher
							addressStr = rawAddress
							if !matchesCriteria(addressStr, criteria.Prefix, criteria.Suffix, criteria.IsChecksum, criteria.Network, criteria.CaseSensitive) {
								continue
							}
						} else if !crypto.MatchesAddressNibbles(&addressBytes, prefixNibbles, suffixNibbles) {
							continue
						} else {
							addressStr = rawAddress
						}
					}

					// Found a match: reconstruct the ECDSA private key for the
					// result (Ethereum only).
					privateKey = new(ecdsa.PrivateKey)
					privateKey.Curve = ethcrypto.S256()
					privateKey.D = new(big.Int).SetBytes(keyBytes[:])
					privateKey.X, privateKey.Y = ethcrypto.S256().ScalarBaseMult(keyBytes[:])
				}

				// Check if address matches criteria (re-check for mnemonic path, or use result from optimized path)
				// For optimized path, we already checked inside the block above to know if we should reconstruct keys
				// But we need a unified flow here.

				// Let's simplify:
				// The optimized path above does the check inside to avoid reconstruction.
				// If we are here from optimized path, it means we found a match.
				// If we are here from mnemonic path, we haven't checked yet.

				// Double check match (just in case)
				if !matchesCriteria(addressStr, criteria.Prefix, criteria.Suffix, criteria.IsChecksum, criteria.Network, criteria.CaseSensitive) {
					continue
				}

				// Found a match!
				// Get private key hex - handle different networks
				var privateKeyHex string
				var publicKeyHex string

				if criteria.Network == "ethereum" || criteria.Network == "" {
					// Ethereum: use ECDSA keys
					privateKeyBytes := ethcrypto.FromECDSA(privateKey)
					privateKeyHex = fmt.Sprintf("%x", privateKeyBytes)

					// Get public key hex
					publicKey := privateKey.Public()
					publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
					publicKeyBytes := ethcrypto.FromECDSAPub(publicKeyECDSA)
					publicKeyHex = fmt.Sprintf("%x", publicKeyBytes)
				}

				// Use checksum address if checksum is required (Ethereum only)
				finalAddress := addressStr
				if criteria.IsChecksum && (criteria.Network == "ethereum" || criteria.Network == "") {
					finalAddress = toChecksumAddress(addressStr)
				}

				result := &wallet.GenerationResult{
					Wallet: &wallet.Wallet{
						Address:    finalAddress,
						PublicKey:  publicKeyHex,
						PrivateKey: privateKeyHex,
						Mnemonic:   mnemonic,
						Network:    criteria.Network,
						CreatedAt:  time.Now(),
					},
					Attempts: attempts,
					Duration: time.Since(startTime),
					WorkerID: workerID,
				}

				select {
				case resultCh <- result:
				case <-ctx.Done():
				}
				return
			}
		}(i)
	}

	// Wait for result or cancellation
	select {
	case result := <-resultCh:
		// Log the wallet generation and operation completion
		if p.logger != nil {
			// Log the specific wallet generated
			if err := p.logger.LogWalletGenerated(
				result.Wallet.Address,
				int(result.Attempts),
				result.Duration,
				result.WorkerID,
			); err != nil {
				fmt.Printf("Warning: Failed to log wallet: %v\n", err)
			}

			// Log operation completion
			stats := logging.OperationStats{
				Duration:     result.Duration,
				Success:      true,
				ItemsCount:   1,
				ErrorCount:   0,
				ThroughputPS: 1.0 / result.Duration.Seconds(),
			}
			if err := p.logger.LogOperationComplete("wallet_generation", stats); err != nil {
				fmt.Printf("Warning: Failed to log operation completion: %v\n", err)
			}
		}
		return result, nil
	case err := <-errorCh:
		// Log the error using secure logging
		if p.logger != nil {
			context := map[string]interface{}{
				"threads": p.threadCount,
			}
			if logErr := p.logger.LogError("wallet_generation", err, context); logErr != nil {
				fmt.Printf("Warning: Failed to log error: %v\n", logErr)
			}
		}
		return nil, err
	case <-ctx.Done():
		cancellationErr := errors.NewCancellationError("generate_wallet", "generation cancelled")
		// Log the cancellation as an error
		if p.logger != nil {
			context := map[string]interface{}{
				"threads": p.threadCount,
				"reason":  "context_cancelled",
			}
			if logErr := p.logger.LogError("wallet_generation", cancellationErr, context); logErr != nil {
				fmt.Printf("Warning: Failed to log cancellation: %v\n", logErr)
			}
		}
		return nil, cancellationErr
	}
}

// generateMnemonicPrivateKey creates a new mnemonic phrase and derives the corresponding private key
func generateMnemonicPrivateKey() (string, *ecdsa.PrivateKey, error) {
	// Generate 128 bits of entropy for a 12-word mnemonic to balance security and performance
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return "", nil, err
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", nil, err
	}

	seed := bip39.NewSeed(mnemonic, "")
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return "", nil, err
	}

	derivationPath := []uint32{
		bip32.FirstHardenedChild + 44,
		bip32.FirstHardenedChild + 60,
		bip32.FirstHardenedChild + 0,
		0,
		0,
	}

	key := masterKey
	for _, child := range derivationPath {
		key, err = key.NewChildKey(child)
		if err != nil {
			return "", nil, err
		}
	}

	privateKey, err := ethcrypto.ToECDSA(key.Key)
	if err != nil {
		return "", nil, err
	}

	return mnemonic, privateKey, nil
}

// matchesCriteria checks if an address matches the given prefix and suffix criteria
// It performs a fast string check first, and only calculates checksum if necessary
// matchesCriteria checks if an address matches the given prefix and suffix criteria
// It performs a fast string check first, and only calculates checksum if necessary
func matchesCriteria(address, prefix, suffix string, isChecksum bool, network string, caseSensitivePattern ...bool) bool {
	return vanity.MatchesCriteria(address, prefix, suffix, isChecksum, network, caseSensitivePattern...)
}

// toChecksumAddress converts an address to EIP-55 checksum format
func toChecksumAddress(address string) string {
	return vanity.ToChecksumAddress(address)
}

// formatEthereumAddressBytes formats raw address bytes as a lowercase
// 0x-prefixed hex string without allocations on the hot path.
func formatEthereumAddressBytes(addressBytes [20]byte) string {
	const hexChars = "0123456789abcdef"
	var out [42]byte
	out[0] = '0'
	out[1] = 'x'
	for i := 0; i < 20; i++ {
		out[2+i*2] = hexChars[addressBytes[i]>>4]
		out[3+i*2] = hexChars[addressBytes[i]&0x0f]
	}
	return string(out[:])
}

// isEIP55Checksum validates EIP-55 checksum for specific pattern
func isEIP55Checksum(address, prefix, suffix string) bool {
	return vanity.IsEIP55Checksum(address, prefix, suffix)
}

// createLogConfigFromAppConfig converts internal config to logging package config
func createLogConfigFromAppConfig(cfg *config.Config) (*logging.LogConfig, error) {
	if !cfg.Logging.Enabled {
		// Return a valid disabled config with default values to pass validation
		return &logging.LogConfig{
			Enabled:     false,
			Level:       logging.INFO,
			Format:      logging.TEXT,
			OutputFile:  "",
			MaxFileSize: 10 * 1024 * 1024, // 10MB default
			MaxFiles:    5,
			BufferSize:  1000,
		}, nil
	}

	// Parse log level
	level, err := logging.ParseLogLevel(cfg.Logging.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", cfg.Logging.Level, err)
	}

	// Parse log format
	var format logging.LogFormat
	switch strings.ToLower(cfg.Logging.Format) {
	case "json":
		format = logging.JSON
	case "structured":
		format = logging.STRUCTURED
	case "text":
		format = logging.TEXT
	default:
		return nil, fmt.Errorf("invalid log format %q, must be one of: text, json, structured", cfg.Logging.Format)
	}

	return &logging.LogConfig{
		Enabled:     cfg.Logging.Enabled,
		Level:       level,
		Format:      format,
		OutputFile:  cfg.Logging.OutputFile,
		MaxFileSize: cfg.Logging.MaxFileSize,
		MaxFiles:    cfg.Logging.MaxFiles,
		BufferSize:  cfg.Logging.BufferSize,
	}, nil
}
