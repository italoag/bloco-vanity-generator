package cli

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"bloco-vgen/internal/config"
	"bloco-vgen/internal/crypto"
	"bloco-vgen/internal/crypto/kdf"
	"bloco-vgen/internal/engine"
	"bloco-vgen/internal/tui"
	"bloco-vgen/internal/validation"
	"bloco-vgen/internal/worker"
	"bloco-vgen/pkg/errors"
	"bloco-vgen/pkg/utils"
	"bloco-vgen/pkg/wallet"
)

// Application represents the CLI application
type Application struct {
	config    *config.Config
	rootCmd   *cobra.Command
	version   string
	gitCommit string
	buildTime string
}

// NewApplication creates a new CLI application
func NewApplication(cfg *config.Config, version, gitCommit, buildTime string) *Application {
	app := &Application{
		config:    cfg,
		version:   version,
		gitCommit: gitCommit,
		buildTime: buildTime,
	}

	app.setupCommands()
	return app
}

// ExecuteContext executes the CLI application with context
func (app *Application) ExecuteContext(ctx context.Context) error {
	return app.rootCmd.ExecuteContext(ctx)
}

// setupCommands sets up all CLI commands
func (app *Application) setupCommands() {
	app.rootCmd = &cobra.Command{
		Use:   "bloco-vgen",
		Short: "High-performance Ethereum wallet generator for custom address patterns",
		Long: `Bloco Vanity Generator is a high-performance CLI tool for generating Ethereum wallets 
with custom prefixes and suffixes. It supports EIP-55 checksum validation,
multi-threaded generation for optimal performance, automatic KeyStore V3
file generation, and secure logging that never exposes sensitive data.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", app.version, app.gitCommit, app.buildTime),
		RunE:    app.generateWallet,
	}

	// Add global flags
	app.addGlobalFlags()

	// Add subcommands
	app.rootCmd.AddCommand(app.createStatsCommand())
	app.rootCmd.AddCommand(app.createBenchmarkCommand())
	app.rootCmd.AddCommand(app.createVersionCommand())
}

// addGlobalFlags adds global flags to the root command
func (app *Application) addGlobalFlags() {
	flags := app.rootCmd.PersistentFlags()

	// Generation parameters
	flags.StringP("prefix", "p", "", "Address prefix to match")
	flags.StringP("suffix", "s", "", "Address suffix to match")
	flags.BoolP("checksum", "c", false, "Enable EIP-55 checksum validation")
	flags.Bool("case-sensitive", false, "Enable case-sensitive pattern matching (requires --checksum)")
	flags.IntP("count", "n", 1, "Number of wallets to generate")
	flags.Bool("with-mnemonic", false, "Generate wallets using BIP-39 mnemonic phrases")
	flags.String("network", "ethereum", "Target network (ethereum, bitcoin, solana)")

	// Performance parameters
	flags.IntP("threads", "t", 0, "Number of worker threads (0 = auto-detect)")
	flags.String("engine", "auto", "Generation engine (auto, cpu, metal)")
	flags.Int("gpu-batch-size", engine.DefaultMetalBatchSize, "Number of candidates processed per Metal generation batch")
	flags.Bool("progress", false, "Show progress information (text mode)")
	flags.Bool("tui", true, "Use the terminal UI (default; disable with --no-tui)")
	flags.Bool("no-tui", false, "Disable the terminal UI and use plain text output")

	// Output parameters
	flags.BoolP("verbose", "v", false, "Enable verbose output")
	flags.BoolP("quiet", "q", false, "Suppress non-essential output")
	flags.String("output", "", "Output file for results (default: stdout)")
	flags.String("format", "text", "Output format (text, json, csv)")

	// KeyStore parameters
	flags.String("keystore-dir", "./keystores", "Directory to save keystore files")
	flags.Bool("no-keystore", false, "Disable keystore file generation")
	flags.String("keystore-kdf", "scrypt", "KDF algorithm for keystore encryption (scrypt, pbkdf2, pbkdf2-sha256, pbkdf2-sha512)")
	flags.String("kdf-params", "", "Custom KDF parameters as JSON (e.g., '{\"n\":262144,\"r\":8,\"p\":1,\"dklen\":32}}' for scrypt)")
	flags.Bool("kdf-analysis", false, "Show KDF compatibility analysis and security assessment")
	flags.String("security-level", "medium", "Minimum security level for KDF parameters (low, medium, high, very-high)")

	// Secure logging parameters (never logs sensitive data)
	flags.String("log-level", "info", "Logging level (error, warn, info, debug) - secure logging only")
	flags.Bool("no-logging", false, "Disable logging completely for maximum performance")
	flags.String("log-file", "", "Log file path (default: stdout) - only operational data logged")
	flags.String("log-format", "text", "Log format (text, json, structured) - all formats are secure")
	flags.Int64("log-max-size", 10*1024*1024, "Maximum log file size in bytes before rotation")
	flags.Int("log-max-files", 5, "Maximum number of rotated log files to keep")
	flags.Int("log-buffer-size", 1000, "Buffer size for async logging")
}

// createWorkerPool creates an optimized worker pool with secure logging
func (app *Application) createWorkerPool(poolManager *crypto.PoolManager, validator *validation.AddressValidator, network string, selection engine.Selection, options engine.GenerationOptions) (worker.WorkerPool, error) {
	if selection.Resolved == engine.NameMetal {
		return worker.NewMetalPoolWithConfig(app.config.Worker.ThreadCount, app.config, options)
	}

	pool := worker.NewPoolWithConfig(app.config.Worker.ThreadCount, app.config, network)
	return pool, nil
}

// generateWallet is the main command handler for wallet generation
func (app *Application) generateWallet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Parse flags and update configuration
	if err := app.parseFlags(cmd); err != nil {
		return errors.WrapError(err, errors.ErrorTypeConfiguration,
			"parse_flags", "failed to parse command flags")
	}

	// Get generation parameters
	criteria, err := app.getGenerationCriteria(cmd)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeValidation,
			"get_criteria", "invalid generation criteria")
	}

	count, _ := cmd.Flags().GetInt("count")
	showProgress, _ := cmd.Flags().GetBool("progress")
	engineSelection, generationOptions, err := app.getGenerationEngineOptions(cmd, criteria)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeConfiguration,
			"resolve_engine", "failed to resolve generation engine")
	}

	// Create crypto components
	poolManager := crypto.NewPoolManager(crypto.DefaultPoolConfig())
	checksumValidator := crypto.NewChecksumValidator(poolManager)
	validator := validation.NewAddressValidator(checksumValidator)

	// Create optimized worker pool using ants
	workerPool, err := app.createWorkerPool(poolManager, validator, criteria.Network, engineSelection, generationOptions)
	if err != nil {
		return err
	}

	// Start worker pool
	if err := workerPool.Start(); err != nil {
		return errors.WrapError(err, errors.ErrorTypeWorker,
			"start_workers", "failed to start worker pool")
	}
	defer func() {
		if err := workerPool.Shutdown(); err != nil {
			// Log shutdown error but don't override the main function's return value
			fmt.Fprintf(os.Stderr, "Warning: failed to shutdown worker pool: %v\n", err)
		}
	}()

	// Build engine diagnostics once. The TUI path renders this block inside
	// the interface, the text path keeps the original printf so both modes
	// expose the same engine, device and batch context before generation.
	engineInfo := newTUIEngineInfo(engineSelection, generationOptions, app.config.Worker.ThreadCount)
	if !app.willUseTUI() {
		app.displayGenerationEngineDiagnostics(engineSelection, generationOptions, showProgress)
	}

	// Generate wallets
	if count == 1 {
		return app.generateSingleWallet(ctx, workerPool, criteria, showProgress, engineInfo)
	} else {
		return app.generateMultipleWallets(ctx, workerPool, criteria, count, showProgress, engineInfo)
	}
}

// willUseTUI mirrors the decision used by generateSingleWallet and
// generateMultipleWallets so callers can suppress duplicate text-mode
// diagnostics when the TUI will render the same information inline.
// The TUI is the default execution mode: it is used whenever it is enabled
// in the config (i.e. --no-tui was not passed) and the terminal supports it.
func (app *Application) willUseTUI() bool {
	if app.config.CLI.QuietMode || !app.config.TUI.Enabled {
		return false
	}
	return tui.NewTUIManager().ShouldUseTUI()
}

// newTUIEngineInfo builds the engine diagnostics block used by the TUI from
// the same data displayed in text mode by displayGenerationEngineDiagnostics.
// Metal-only fields (device, batch, validation) are populated only when the
// resolved engine is Metal so the TUI matches the text behavior.
func newTUIEngineInfo(selection engine.Selection, options engine.GenerationOptions, threadCount int) tui.EngineInfo {
	info := tui.EngineInfo{
		Engine:          selection.Resolved,
		RequestedEngine: selection.Requested,
		FallbackReason:  selection.FallbackReason,
		ThreadCount:     threadCount,
		Network:         options.Network,
	}
	if selection.Resolved == engine.NameMetal {
		info.DeviceName = engine.MetalDeviceName()
		info.BatchSize = options.BatchSize
		info.MetalValidation = options.MetalValidation
	}
	return info
}

// generateSingleWallet generates a single wallet with progress tracking
func (app *Application) generateSingleWallet(
	ctx context.Context,
	workerPool worker.WorkerPool,
	criteria wallet.GenerationCriteria,
	showProgress bool,
	engineInfo tui.EngineInfo,
) error {
	// Check if TUI should be used for progress. The TUI is the default
	// execution mode; it is skipped only when disabled (--no-tui), in quiet
	// mode, or when the terminal does not support it.
	tuiManager := tui.NewTUIManager()
	useTUI := app.config.TUI.Enabled && !app.config.CLI.QuietMode

	// Debug TUI decision
	if os.Getenv("BLOCO_DEBUG") != "" {
		fmt.Printf("DEBUG TUI: config.TUI.Enabled=%v, showProgress=%v, QuietMode=%v, ShouldUseTUI=%v\n",
			app.config.TUI.Enabled, showProgress, app.config.CLI.QuietMode, tuiManager.ShouldUseTUI())
	}

	if useTUI && tuiManager.ShouldUseTUI() {
		return app.generateSingleWalletTUI(ctx, workerPool, criteria, engineInfo)
	}

	// Fallback to text mode
	return app.generateSingleWalletText(ctx, workerPool, criteria, showProgress)
}

// generateSingleWalletTUI generates a single wallet with TUI progress (fixed based on monolithic version)
func (app *Application) generateSingleWalletTUI(
	ctx context.Context,
	workerPool worker.WorkerPool,
	criteria wallet.GenerationCriteria,
	engineInfo tui.EngineInfo,
) error {
	// Create TUI statistics
	difficulty := calculateDifficulty(criteria)
	probability50 := calculateProbability50(difficulty)

	tuiStats := &wallet.GenerationStats{
		Difficulty:      difficulty,
		Probability50:   probability50,
		CurrentAttempts: 0,
		Speed:           0,
		Probability:     0,
		EstimatedTime:   0,
		StartTime:       time.Now(),
		LastUpdate:      time.Now(),
		Pattern:         criteria.GetPattern(),
		IsChecksum:      criteria.IsChecksum,
	}

	// Create stats adapter
	statsCollector := workerPool.GetStatsCollector()
	statsAdapter := &StatsManagerAdapter{statsCollector}

	// Create TUI progress model with engine diagnostics preloaded so the
	// running view shows the same engine, device and batch info as text mode.
	tuiManager := tui.NewTUIManager()
	progressModel := tuiManager.CreateProgressModelWithEngine(tuiStats, statsAdapter, engineInfo)

	// Create TUI program (without alt screen for compatibility)
	program := tea.NewProgram(progressModel)

	// Channel for wallet results (like in monolithic version)
	walletResultsChan := make(chan tui.WalletResult, 1)

	// Channel to signal shutdown
	shutdownChan := make(chan struct{})
	var shutdownOnce sync.Once // Ensure channel is closed only once

	// Start progress updates (like in monolithic version)
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-shutdownChan:
				// Send quit message and exit
				program.Send(tui.SendQuit())
				return

			case <-ticker.C:
				// Get current stats and send progress update
				stats := statsCollector.GetAggregatedStats()

				// Calculate probability based on current attempts
				probability := utils.CalculateProbability(difficulty, stats.TotalAttempts) * 100

				// Calculate estimated time
				var estimatedTime time.Duration
				if probability50 > 0 && stats.TotalSpeed > 0 {
					remainingAttempts := probability50 - stats.TotalAttempts
					if remainingAttempts > 0 {
						estimatedTime = time.Duration(float64(remainingAttempts)/stats.TotalSpeed) * time.Second
					}
				}

				// Send progress update to TUI
				program.Send(tui.ProgressMsg{
					Attempts:         stats.TotalAttempts,
					Speed:            stats.TotalSpeed,
					Probability:      probability,
					EstimatedTime:    estimatedTime,
					Difficulty:       difficulty,
					Pattern:          criteria.GetPattern(),
					CompletedWallets: 0, // Single wallet mode
					TotalWallets:     1,
					ProgressPercent:  probability,
					IsComplete:       false,
				})

			case walletResult, ok := <-walletResultsChan:
				if !ok {
					// Channel closed, exit
					shutdownOnce.Do(func() { close(shutdownChan) })
					return
				}

				// Send wallet result to TUI
				program.Send(tui.WalletResultMsg{
					Result: walletResult,
				})

				// Send completion flag immediately
				program.Send(tui.ProgressMsg{
					Attempts:         0, // Will be updated by stats
					Speed:            0, // Will be updated by stats
					Probability:      100.0,
					EstimatedTime:    0,
					Difficulty:       difficulty,
					Pattern:          criteria.GetPattern(),
					CompletedWallets: 1,
					TotalWallets:     1,
					ProgressPercent:  100.0,
					IsComplete:       true,
				})

				// Mark as complete and send quit after showing result
				go func() {
					time.Sleep(2 * time.Second)
					shutdownOnce.Do(func() { close(shutdownChan) })
				}()
			}
		}
	}()

	// Start wallet generation in background (like in monolithic version)
	var result *wallet.GenerationResult
	var genErr error

	go func() {
		// Small delay to let TUI initialize
		time.Sleep(200 * time.Millisecond)

		genResult, err := workerPool.GenerateWalletWithContext(ctx, criteria)
		if err != nil {
			genErr = err
			shutdownOnce.Do(func() { close(shutdownChan) })
			return
		}

		result = genResult

		// Generate and save keystore files if enabled (silent mode for TUI)
		if app.config.KeyStore.Enabled {
			if err := app.generateAndSaveKeystoreWithVerbose(genResult.Wallet, false); err != nil {
				if !app.config.CLI.QuietMode {
					fmt.Printf("Warning: Failed to generate keystore: %v\n", err)
				}
			}
		}

		// Send wallet result to TUI through channel
		select {
		case walletResultsChan <- tui.WalletResult{
			Index:      1,
			Address:    genResult.Wallet.Address,
			PrivateKey: genResult.Wallet.PrivateKey,
			Attempts:   int(genResult.Attempts),
			Time:       genResult.Duration,
			Error:      "",
		}:
		case <-ctx.Done():
		}

		// Close the channel to signal completion
		close(walletResultsChan)
	}()

	// Run the TUI program (this blocks until quit)
	if _, err := program.Run(); err != nil {
		fmt.Printf("TUI failed: %v, falling back to text mode\n", err)
		return app.generateSingleWalletText(ctx, workerPool, criteria, true)
	}

	// Check for generation error
	if genErr != nil {
		return errors.WrapError(genErr, errors.ErrorTypeGeneration,
			"generate_wallet_tui", "failed to generate wallet")
	}

	// Return the result (don't show it again since TUI already showed it)
	if result != nil {
		return nil // Success, TUI already displayed the result
	}

	return ctx.Err() // Context was cancelled
}

// generateSingleWalletText generates a single wallet with text progress
func (app *Application) generateSingleWalletText(
	ctx context.Context,
	workerPool worker.WorkerPool,
	criteria wallet.GenerationCriteria,
	showProgress bool,
) error {
	if showProgress && !app.config.CLI.QuietMode {
		fmt.Printf("Generating wallet with pattern: %s\n", criteria.GetPattern())
		fmt.Printf("Difficulty: %s\n", formatLargeNumber(int64(calculateDifficulty(criteria))))
		fmt.Printf("Using %d worker threads\n\n", app.config.Worker.ThreadCount)
	}

	// Completely disable progress manager to avoid deadlocks
	// The issue is in the progress system, not the worker pool
	_ = showProgress // Acknowledge parameter but don't use it

	// Generate wallet
	result, err := workerPool.GenerateWalletWithContext(ctx, criteria)
	if err != nil {
		if showProgress && !app.config.CLI.QuietMode {
			fmt.Printf("\n")
		}
		return errors.WrapError(err, errors.ErrorTypeGeneration,
			"generate_wallet", "failed to generate wallet")
	}

	// Wallet completed successfully

	if showProgress && !app.config.CLI.QuietMode {
		fmt.Printf("\n")
	}

	// Display result
	return app.displayWalletResult(result, showProgress)
}

// generateMultipleWallets generates multiple wallets with progress tracking
func (app *Application) generateMultipleWallets(
	ctx context.Context,
	workerPool worker.WorkerPool,
	criteria wallet.GenerationCriteria,
	count int,
	showProgress bool,
	engineInfo tui.EngineInfo,
) error {
	// Check if TUI should be used for multiple wallets. The TUI is the
	// default execution mode; it is skipped only when disabled (--no-tui),
	// in quiet mode, or when the terminal does not support it.
	tuiManager := tui.NewTUIManager()
	useTUI := app.config.TUI.Enabled && !app.config.CLI.QuietMode

	// Debug TUI decision for multiple wallets
	if os.Getenv("BLOCO_DEBUG") != "" {
		fmt.Printf("DEBUG TUI (multiple): config.TUI.Enabled=%v, showProgress=%v, QuietMode=%v, ShouldUseTUI=%v\n",
			app.config.TUI.Enabled, showProgress, app.config.CLI.QuietMode, tuiManager.ShouldUseTUI())
	}

	if useTUI && tuiManager.ShouldUseTUI() {
		return app.generateMultipleWalletsTUI(ctx, workerPool, criteria, count, engineInfo)
	}

	// Fallback to text mode for multiple wallets
	return app.generateMultipleWalletsText(ctx, workerPool, criteria, count, showProgress)
}

// generateMultipleWalletsTUI generates multiple wallets with TUI (based on monolithic version)
func (app *Application) generateMultipleWalletsTUI(
	ctx context.Context,
	workerPool worker.WorkerPool,
	criteria wallet.GenerationCriteria,
	count int,
	engineInfo tui.EngineInfo,
) error {
	// Create TUI statistics
	difficulty := calculateDifficulty(criteria)
	probability50 := calculateProbability50(difficulty)

	tuiStats := &wallet.GenerationStats{
		Difficulty:      difficulty,
		Probability50:   probability50,
		CurrentAttempts: 0,
		Speed:           0,
		Probability:     0,
		EstimatedTime:   0,
		StartTime:       time.Now(),
		LastUpdate:      time.Now(),
		Pattern:         criteria.GetPattern(),
		IsChecksum:      criteria.IsChecksum,
	}

	// Create stats adapter
	statsCollector := workerPool.GetStatsCollector()
	statsAdapter := &StatsManagerAdapter{statsCollector}

	// Create TUI progress model with engine diagnostics preloaded so the
	// running view shows the same engine, device and batch info as text mode.
	tuiManager := tui.NewTUIManager()
	progressModel := tuiManager.CreateProgressModelWithEngine(tuiStats, statsAdapter, engineInfo)

	// Create TUI program (without alt screen for compatibility)
	program := tea.NewProgram(progressModel)

	// Channels for communication (like in monolithic version)
	walletResultsChan := make(chan tui.WalletResult, count)
	shutdownChan := make(chan struct{})
	var shutdownOnce sync.Once // Ensure channel is closed only once

	// Track generation progress (with mutex to prevent race conditions)
	var completedWallets int
	var completedMutex sync.Mutex
	var results []*wallet.GenerationResult

	// Start progress updates (like in monolithic version)
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-shutdownChan:
				// Send quit message and exit
				program.Send(tui.SendQuit())
				return

			case <-ticker.C:
				// Get current stats and send progress update
				stats := statsCollector.GetAggregatedStats()

				// Calculate progress as percentage of wallets completed (thread-safe)
				completedMutex.Lock()
				currentCompleted := completedWallets
				completedMutex.Unlock()

				progressPercent := (float64(currentCompleted) / float64(count)) * 100.0

				// Calculate probability based on progress
				probability := progressPercent

				// Calculate estimated time
				var estimatedTime time.Duration
				if currentCompleted > 0 && stats.TotalSpeed > 0 {
					remaining := count - currentCompleted
					if remaining > 0 {
						// Estimate remaining attempts based on average so far
						avgAttemptsPerWallet := float64(stats.TotalAttempts) / float64(currentCompleted)
						estimatedRemainingAttempts := avgAttemptsPerWallet * float64(remaining)
						estimatedTime = time.Duration(estimatedRemainingAttempts/stats.TotalSpeed) * time.Second
					}
				} else if stats.TotalSpeed > 0 && probability50 > 0 {
					// Use difficulty-based estimation
					estimatedTotalAttempts := int64(count) * probability50
					remainingAttempts := estimatedTotalAttempts - stats.TotalAttempts
					if remainingAttempts > 0 {
						estimatedTime = time.Duration(float64(remainingAttempts)/stats.TotalSpeed) * time.Second
					}
				}

				// Send progress update to TUI
				program.Send(tui.ProgressMsg{
					Attempts:         stats.TotalAttempts,
					Speed:            stats.TotalSpeed,
					Probability:      probability,
					EstimatedTime:    estimatedTime,
					Difficulty:       difficulty,
					Pattern:          criteria.GetPattern(),
					CompletedWallets: currentCompleted,
					TotalWallets:     count,
					ProgressPercent:  progressPercent,
					IsComplete:       currentCompleted >= count,
				})

			case walletResult, ok := <-walletResultsChan:
				if !ok {
					// Channel closed, exit
					shutdownOnce.Do(func() { close(shutdownChan) })
					return
				}

				// Send wallet result to TUI first
				program.Send(tui.WalletResultMsg{
					Result: walletResult,
				})

				// Get current completion status (thread-safe)
				completedMutex.Lock()
				currentCompletedForMsg := completedWallets
				allCompleted := completedWallets >= count
				completedMutex.Unlock()

				// Send progress update showing current completion
				program.Send(tui.ProgressMsg{
					Attempts:         0, // Will be updated by main ticker
					Speed:            0, // Will be updated by main ticker
					Probability:      (float64(currentCompletedForMsg) / float64(count)) * 100.0,
					EstimatedTime:    0,
					Difficulty:       difficulty,
					Pattern:          criteria.GetPattern(),
					CompletedWallets: currentCompletedForMsg,
					TotalWallets:     count,
					ProgressPercent:  (float64(currentCompletedForMsg) / float64(count)) * 100.0,
					IsComplete:       allCompleted,
				})

				if allCompleted {
					// All wallets completed, quit after showing results
					go func() {
						time.Sleep(3 * time.Second) // Time to see all results
						shutdownOnce.Do(func() { close(shutdownChan) })
					}()
				}
			}
		}
	}()

	// Start wallet generation in background (like in monolithic version)
	var genErr error

	go func() {
		// Small delay to let TUI initialize
		time.Sleep(200 * time.Millisecond)

		results = make([]*wallet.GenerationResult, 0, count)

		for i := 0; i < count; i++ {
			select {
			case <-ctx.Done():
				genErr = ctx.Err()
				shutdownOnce.Do(func() { close(shutdownChan) })
				return
			default:
			}

			result, err := workerPool.GenerateWalletWithContext(ctx, criteria)
			if err != nil {
				// Send error result to TUI
				select {
				case walletResultsChan <- tui.WalletResult{
					Index: i + 1,
					Error: err.Error(),
				}:
				case <-ctx.Done():
					return
				}

				// Update completed wallets count even for errors (thread-safe)
				completedMutex.Lock()
				completedWallets++
				completedMutex.Unlock()
				continue
			}

			results = append(results, result)

			// Generate and save keystore files if enabled (silent mode for TUI)
			if app.config.KeyStore.Enabled {
				if err := app.generateAndSaveKeystoreWithVerbose(result.Wallet, false); err != nil {
					if !app.config.CLI.QuietMode {
						fmt.Printf("Warning: Failed to generate keystore for wallet %d: %v\n", i+1, err)
					}
				}
			}

			// Update completed wallets count (thread-safe)
			completedMutex.Lock()
			completedWallets++
			completedMutex.Unlock()

			// Send successful wallet result to TUI
			select {
			case walletResultsChan <- tui.WalletResult{
				Index:      i + 1,
				Address:    result.Wallet.Address,
				PrivateKey: result.Wallet.PrivateKey,
				Attempts:   int(result.Attempts),
				Time:       result.Duration,
				Error:      "",
			}:
			case <-ctx.Done():
				return
			}
		}

		// Close the wallet results channel when all wallets are generated
		// This signals completion to the progress goroutine
		close(walletResultsChan)
	}()

	// Run the TUI program (this blocks until quit)
	if _, err := program.Run(); err != nil {
		fmt.Printf("TUI failed: %v, falling back to text mode\n", err)
		return app.generateMultipleWalletsText(ctx, workerPool, criteria, count, true)
	}

	// Check for generation error
	if genErr != nil {
		return errors.WrapError(genErr, errors.ErrorTypeGeneration,
			"generate_multiple_wallets_tui", "failed to generate wallets")
	}

	// Return success (TUI already displayed the results)
	return nil
}

// generateMultipleWalletsText generates multiple wallets with text progress (fallback)
func (app *Application) generateMultipleWalletsText(
	ctx context.Context,
	workerPool worker.WorkerPool,
	criteria wallet.GenerationCriteria,
	count int,
	showProgress bool,
) error {
	if showProgress && !app.config.CLI.QuietMode {
		fmt.Printf("Generating %d wallets with pattern: %s\n", count, criteria.GetPattern())
		fmt.Printf("Difficulty: %s\n", formatLargeNumber(int64(calculateDifficulty(criteria))))
		fmt.Printf("Using %d worker threads\n\n", app.config.Worker.ThreadCount)
	}

	results := make([]*wallet.GenerationResult, 0, count)
	startTime := time.Now()
	var totalAttempts int64

	// Disable progress manager to avoid deadlocks
	_ = showProgress // Acknowledge parameter but don't use it

	// Generate wallets with progress tracking
	for i := range count {
		result, err := workerPool.GenerateWalletWithContext(ctx, criteria)
		if err != nil {
			if showProgress && !app.config.CLI.QuietMode {
				fmt.Printf("\nError generating wallet %d: %v\n", i+1, err)
			}

			// Continue with next wallet instead of failing completely
			continue
		}

		results = append(results, result)
		totalAttempts += result.Attempts

		// Mark wallet as completed
		// Progress tracking disabled

		// Show individual wallet result if verbose
		if app.config.CLI.VerboseOutput {
			fmt.Printf("\nWallet %d: 0x%s (attempts: %s)\n",
				i+1, result.Wallet.Address, formatLargeNumber(result.Attempts))
		}
	}

	if showProgress && !app.config.CLI.QuietMode {
		fmt.Printf("\n")
	}

	// Display summary
	return app.displayMultipleWalletResults(results, totalAttempts, time.Since(startTime), showProgress)
}

// createStatsCommand creates the stats subcommand
func (app *Application) createStatsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show difficulty statistics for a pattern",
		Long:  "Display detailed statistics about the difficulty of generating addresses with the specified pattern.",
		RunE:  app.showStats,
	}

	// Add stats-specific flags
	cmd.Flags().StringP("prefix", "p", "", "Address prefix to analyze")
	cmd.Flags().StringP("suffix", "s", "", "Address suffix to analyze")
	cmd.Flags().BoolP("checksum", "c", false, "Include checksum validation in analysis")

	return cmd
}

// showStats displays statistics for a pattern
func (app *Application) showStats(cmd *cobra.Command, args []string) error {
	if err := app.parseFlags(cmd); err != nil {
		return errors.WrapError(err, errors.ErrorTypeConfiguration,
			"parse_flags", "failed to parse command flags")
	}

	criteria, err := app.getGenerationCriteria(cmd)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeValidation,
			"show_stats", "invalid pattern criteria")
	}
	engineSelection, generationOptions, err := app.getGenerationEngineOptions(cmd, criteria)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeConfiguration,
			"resolve_engine", "failed to resolve generation engine")
	}
	engineInfo := newTUIEngineInfo(engineSelection, generationOptions, app.config.Worker.ThreadCount)

	// Calculate statistics
	difficulty := calculateDifficulty(criteria)
	probability50 := calculateProbability50(difficulty)

	// Check if TUI should be used (default mode; --no-tui disables it)
	tuiManager := tui.NewTUIManager()
	useTUI, _ := cmd.Flags().GetBool("tui")
	if noTUI, _ := cmd.Flags().GetBool("no-tui"); noTUI {
		useTUI = false
	}

	if useTUI && tuiManager.ShouldUseTUI() {
		return app.showStatsTUI(criteria, difficulty, probability50, engineInfo)
	}

	// Fallback to text mode
	return app.showStatsText(criteria, difficulty, probability50)
}

// showStatsTUI displays statistics using TUI interface
func (app *Application) showStatsTUI(criteria wallet.GenerationCriteria, difficulty float64, probability50 int64, engineInfo tui.EngineInfo) error {
	// Create TUI statistics interface
	tuiStats := &wallet.GenerationStats{
		Difficulty:      difficulty,
		Probability50:   probability50,
		CurrentAttempts: 0, // For stats display, this is not relevant
		Speed:           0, // For stats display, this is not relevant
		Probability:     0, // For stats display, this is not relevant
		EstimatedTime:   0, // For stats display, this is not relevant
		StartTime:       time.Now(),
		LastUpdate:      time.Now(),
		Pattern:         criteria.GetPattern(),
		IsChecksum:      criteria.IsChecksum,
	}

	// Create TUI stats model
	tuiManager := tui.NewTUIManager()
	statsModel := tuiManager.CreateStatsModelWithEngine(tuiStats, engineInfo)

	// Create TUI program
	program := tea.NewProgram(statsModel, tea.WithAltScreen())

	// Run the TUI program
	if _, err := program.Run(); err != nil {
		// If TUI fails, fallback to text mode
		fmt.Printf("TUI failed: %v, falling back to text mode\n", err)
		return app.showStatsText(criteria, difficulty, probability50)
	}

	return nil
}

// showStatsText displays statistics in text mode (fallback)
func (app *Application) showStatsText(criteria wallet.GenerationCriteria, difficulty float64, probability50 int64) error {
	// Display statistics
	fmt.Printf("Pattern Analysis: %s\n", criteria.GetPattern())
	fmt.Printf("═══════════════════════════════════════\n\n")

	fmt.Printf("Pattern Length: %d characters\n", criteria.GetPatternLength())
	fmt.Printf("Checksum Validation: %s\n", formatBool(criteria.IsChecksum))
	fmt.Printf("Difficulty: %s\n", formatLargeNumber(int64(difficulty)))
	fmt.Printf("50%% Probability: %s attempts\n", formatLargeNumber(probability50))

	fmt.Printf("\nDetailed Statistics:\n")
	fmt.Printf("%-22s %-30s %s\n", "Metric", "Value", "Description")
	fmt.Printf("%s\n", strings.Repeat("─", 95))
	for _, row := range buildStatsTextRows(criteria, difficulty, probability50) {
		fmt.Printf("%-22s %-30s %s\n", row.metric, row.value, row.description)
	}

	// Show time estimates at different speeds
	fmt.Printf("\nTime Estimates:\n")
	speeds := []float64{1000, 10000, 50000, 100000}
	for _, speed := range speeds {
		if probability50 > 0 {
			duration := time.Duration(float64(probability50)/speed) * time.Second
			fmt.Printf("  At %s addr/s: %s\n",
				formatLargeNumber(int64(speed)),
				formatDuration(duration))
		}
	}

	return nil
}

type statsTextRow struct {
	metric      string
	value       string
	description string
}

func buildStatsTextRows(criteria wallet.GenerationCriteria, difficulty float64, probability50 int64) []statsTextRow {
	pattern := criteria.GetPattern()
	if pattern == "" {
		pattern = "any"
	}
	baseDifficulty := utils.CalculateDifficulty(criteria.Prefix, criteria.Suffix, false)
	rows := []statsTextRow{
		{
			metric:      "Pattern",
			value:       pattern,
			description: "The address pattern to match",
		},
		{
			metric:      "Pattern Length",
			value:       fmt.Sprintf("%d characters", criteria.GetPatternLength()),
			description: "Number of hex characters to match",
		},
		{
			metric:      "Base Difficulty",
			value:       formatLargeNumber(int64(baseDifficulty)),
			description: "Difficulty without checksum validation",
		},
		{
			metric:      "Total Difficulty",
			value:       formatLargeNumber(int64(difficulty)),
			description: "Final difficulty including checksum",
		},
	}
	if criteria.IsChecksum && baseDifficulty > 0 {
		rows = append(rows, statsTextRow{
			metric:      "Checksum Multiplier",
			value:       fmt.Sprintf("%.1fx", difficulty/baseDifficulty),
			description: "Difficulty increase from checksum",
		})
	}
	if probability50 > 0 {
		rows = append(rows, statsTextRow{
			metric:      "50% Probability",
			value:       formatLargeNumber(probability50),
			description: "Attempts needed for 50% success chance",
		})
	} else {
		rows = append(rows, statsTextRow{
			metric:      "50% Probability",
			value:       "Nearly impossible",
			description: "Pattern is extremely difficult",
		})
	}
	rows = append(rows, statsTextRow{
		metric:      "Expected Attempts",
		value:       formatLargeNumber(int64(difficulty)),
		description: "Mathematical expectation (average)",
	})
	probPerAttempt := 0.0
	if difficulty > 0 {
		probPerAttempt = 1.0 / difficulty * 100
	}
	probPerAttemptStr := "0%"
	if probPerAttempt > 0 && probPerAttempt < 0.000001 {
		probPerAttemptStr = fmt.Sprintf("%.2e%%", probPerAttempt)
	} else if probPerAttempt > 0 {
		probPerAttemptStr = fmt.Sprintf("%.6f%%", probPerAttempt)
	}
	rows = append(rows, statsTextRow{
		metric:      "Success Rate",
		value:       probPerAttemptStr,
		description: "Probability of success per attempt",
	})
	return rows
}

// createBenchmarkCommand creates the benchmark subcommand
func (app *Application) createBenchmarkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "benchmark",
		Short: "Run performance benchmarks",
		Long:  "Run performance benchmarks to measure address generation speed and thread efficiency.",
		RunE:  app.runBenchmark,
	}

	app.addBenchmarkFlags(cmd, false)
	compareCmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare CPU, auto, and Metal benchmark performance",
		Long:  "Run a benchmark comparison matrix across CPU, auto, and Metal for one or more patterns, batch sizes, and checksum modes.",
		RunE:  app.runBenchmark,
	}
	app.addBenchmarkFlags(compareCmd, true)
	cmd.AddCommand(compareCmd)

	return cmd
}

func (app *Application) addBenchmarkFlags(cmd *cobra.Command, compareDefault bool) {
	cmd.Flags().Int("attempts", 10000, "Number of attempts for benchmark")
	cmd.Flags().Duration("duration", 30*time.Second, "Benchmark duration")
	cmd.Flags().Bool("detailed", false, "Show detailed per-thread statistics")
	cmd.Flags().Int("batch-size", 5000, "Number of candidates processed per benchmark worker batch")
	cmd.Flags().String("pattern", "", "Address prefix pattern for benchmark compatibility")
	cmd.Flags().String("metal-validation", engine.MetalValidationFull, "Metal CPU validation mode (requires full)")
	cmd.Flags().Bool("compare", compareDefault, "CPU vs Auto vs Metal comparison matrix")
	cmd.Flags().String("compare-patterns", "ab,abcd,abcdef", "Comma-separated prefix patterns for comparison")
	cmd.Flags().String("compare-batch-sizes", "1000,5000,10000", "Comma-separated batch sizes for comparison")
	cmd.Flags().String("compare-checksums", "off,on", "Comma-separated checksum modes for comparison (off,on)")
}

// runBenchmark runs performance benchmarks
func (app *Application) runBenchmark(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	options, err := app.getBenchmarkOptions(cmd)
	if err != nil {
		return err
	}
	if options.Compare {
		tuiManager := tui.NewTUIManager()
		if options.Format == benchmarkFormatText && options.Output == "" && options.UseTUI && tuiManager.ShouldUseTUI() {
			return app.runBenchmarkComparisonTUI(ctx, options)
		}
		return app.runBenchmarkComparisonText(ctx, options)
	}

	// Check if TUI should be used
	tuiManager := tui.NewTUIManager()

	if options.Format == benchmarkFormatText && options.Output == "" && options.UseTUI && tuiManager.ShouldUseTUI() {
		return app.runBenchmarkTUI(ctx, options)
	}

	// Fallback to text mode
	return app.runBenchmarkText(ctx, options)
}

// runBenchmarkTUI runs benchmark with TUI interface
func (app *Application) runBenchmarkTUI(ctx context.Context, options benchmarkOptions) error {
	type benchmarkTUIOutcome struct {
		result *wallet.BenchmarkResult
		err    error
	}

	// Create TUI benchmark model preloaded with engine diagnostics so the
	// running and results views show the same engine, device and batch info
	// emitted by runBenchmarkText.
	tuiManager := tui.NewTUIManager()
	benchmarkModel := tuiManager.CreateBenchmarkModelWithEngine(newBenchmarkTUIEngineInfo(options, app.config.Worker.ThreadCount))

	benchmarkCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	outcomeCh := make(chan benchmarkTUIOutcome, 1)

	// Create TUI program
	program := tea.NewProgram(benchmarkModel, tea.WithAltScreen())

	// Start benchmark in background
	go func() {
		// Give TUI time to initialize
		timer := time.NewTimer(200 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-benchmarkCtx.Done():
			outcomeCh <- benchmarkTUIOutcome{err: benchmarkCtx.Err()}
			return
		}

		// Run benchmark and send updates to TUI
		result, err := app.runBenchmarkEngine(benchmarkCtx, options, 500*time.Millisecond, func(sample benchmarkSample) {
			sendBenchmarkTUISample(program, options, sample)
		})
		if err != nil {
			outcomeCh <- benchmarkTUIOutcome{err: err}
			program.Send(tui.BenchmarkCompleteMsg{Results: nil})
			return
		}

		outcomeCh <- benchmarkTUIOutcome{result: result}

		// Send completion message
		program.Send(tui.BenchmarkCompleteMsg{Results: result})
	}()

	// Run the TUI program
	finalModel, err := program.Run()
	cancel()
	outcome := <-outcomeCh
	if err != nil {
		// If TUI fails, fallback to text mode
		fmt.Printf("TUI failed: %v, falling back to text mode\n", err)
		return app.runBenchmarkText(ctx, options)
	}

	if outcome.err != nil {
		if benchmarkCtx.Err() != nil && (stderrors.Is(outcome.err, context.Canceled) || stderrors.Is(outcome.err, context.DeadlineExceeded)) {
			if model, ok := finalModel.(interface{ Quitting() bool }); ok && model.Quitting() {
				return nil
			}
		}
		return errors.WrapError(outcome.err, errors.ErrorTypeGeneration,
			"run_benchmark", "benchmark execution failed")
	}

	if options.Output != "" && outcome.result != nil {
		return app.writeBenchmarkOutput(outcome.result, options)
	}

	return nil
}

// runBenchmarkText runs benchmark in text mode
func (app *Application) runBenchmarkText(ctx context.Context, options benchmarkOptions) error {
	if options.Format == benchmarkFormatText {
		fmt.Printf("Running benchmark...\n")
		fmt.Printf("Engine: %s\n", options.Engine)
		if options.FallbackReason != "" {
			fmt.Printf("Fallback: %s\n", options.FallbackReason)
		}
		fmt.Printf("Network: %s\n", options.Network)
		fmt.Printf("Pattern: %s\n", displayBenchmarkPattern(options.Criteria.GetPattern()))
		fmt.Printf("Attempts: %s\n", formatLargeNumber(int64(options.Attempts)))
		fmt.Printf("Duration: %v\n", options.Duration)
		fmt.Printf("Batch Size: %d\n", options.BatchSize)
		fmt.Printf("Threads: %d\n\n", app.config.Worker.ThreadCount)
	}

	var onSample func(benchmarkSample)
	if options.Format == benchmarkFormatText {
		onSample = func(sample benchmarkSample) {
			fmt.Printf("\rSample: %.0f addr/s (total: %s attempts, matches: %s)",
				sample.Speed, formatLargeNumber(sample.Attempts), formatLargeNumber(sample.Matches))
		}
	}

	result, err := app.runBenchmarkEngine(ctx, options, time.Second, onSample)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGeneration,
			"run_benchmark", "benchmark execution failed")
	}

	if options.Format == benchmarkFormatText {
		fmt.Printf("\nBenchmark completed!\n")
	}
	return app.writeBenchmarkOutput(result, options)
}

// createVersionCommand creates the version subcommand
func (app *Application) createVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Bloco Vanity Generator %s\n", app.version)
			fmt.Printf("Git Commit: %s\n", app.gitCommit)
			fmt.Printf("Build Time: %s\n", app.buildTime)
		},
	}
}

// Helper functions

// parseFlags parses command flags and updates configuration
func (app *Application) parseFlags(cmd *cobra.Command) error {
	// Parse thread count
	if threads, _ := cmd.Flags().GetInt("threads"); threads >= 0 {
		if threads == 0 {
			// Auto-detect CPU cores
			app.config.Worker.ThreadCount = runtime.NumCPU()
		} else {
			app.config.Worker.ThreadCount = threads
		}
	}

	// Parse output options
	if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
		app.config.CLI.VerboseOutput = true
	}

	if quiet, _ := cmd.Flags().GetBool("quiet"); quiet {
		app.config.CLI.QuietMode = true
	}

	// Parse TUI option: --no-tui disables the TUI (the default execution
	// mode); --tui=false is kept for backwards compatibility.
	if noTUI, _ := cmd.Flags().GetBool("no-tui"); noTUI {
		app.config.TUI.Enabled = false
	}
	if tui, _ := cmd.Flags().GetBool("tui"); !tui {
		app.config.TUI.Enabled = false
	}

	// Parse keystore options
	if noKeystore, _ := cmd.Flags().GetBool("no-keystore"); noKeystore {
		app.config.KeyStore.Enabled = false
	}

	// Only update keystore directory if the flag was explicitly set by the user
	if cmd.Flags().Changed("keystore-dir") {
		if keystoreDir, _ := cmd.Flags().GetString("keystore-dir"); keystoreDir != "" {
			app.config.KeyStore.OutputDir = keystoreDir
		}
	}

	// Only update KDF algorithm if the flag was explicitly set by the user
	if cmd.Flags().Changed("keystore-kdf") {
		if keystoreKDF, _ := cmd.Flags().GetString("keystore-kdf"); keystoreKDF != "" {
			app.config.KeyStore.KDFAlgorithm = keystoreKDF
		}
	}

	// Parse KDF parameters if provided
	if cmd.Flags().Changed("kdf-params") {
		if kdfParamsStr, _ := cmd.Flags().GetString("kdf-params"); kdfParamsStr != "" {
			if err := app.parseKDFParams(kdfParamsStr); err != nil {
				return fmt.Errorf("invalid KDF parameters: %w", err)
			}
		}
	}

	// Parse KDF analysis flag
	if kdfAnalysis, _ := cmd.Flags().GetBool("kdf-analysis"); kdfAnalysis {
		app.config.KeyStore.ShowAnalysis = true
	}

	// Parse security level
	if cmd.Flags().Changed("security-level") {
		if securityLevel, _ := cmd.Flags().GetString("security-level"); securityLevel != "" {
			app.config.KeyStore.SecurityLevel = securityLevel
		}
	}

	// Parse logging configuration
	if err := app.parseLoggingFlags(cmd); err != nil {
		return fmt.Errorf("failed to parse logging configuration: %w", err)
	}

	// Validate configuration after updates
	return app.config.Validate()
}

// parseLoggingFlags parses logging-related command flags and updates configuration
func (app *Application) parseLoggingFlags(cmd *cobra.Command) error {
	// Parse no-logging flag
	if noLogging, _ := cmd.Flags().GetBool("no-logging"); noLogging {
		app.config.Logging.Enabled = false
		return nil // Skip other logging flags if logging is disabled
	}

	// Parse log level
	if cmd.Flags().Changed("log-level") {
		if logLevel, _ := cmd.Flags().GetString("log-level"); logLevel != "" {
			app.config.Logging.Level = logLevel
		}
	}

	// Parse log format
	if cmd.Flags().Changed("log-format") {
		if logFormat, _ := cmd.Flags().GetString("log-format"); logFormat != "" {
			app.config.Logging.Format = logFormat
		}
	}

	// Parse log file
	if cmd.Flags().Changed("log-file") {
		if logFile, _ := cmd.Flags().GetString("log-file"); logFile != "" {
			app.config.Logging.OutputFile = logFile
		}
	}

	// Parse log max size
	if cmd.Flags().Changed("log-max-size") {
		if logMaxSize, _ := cmd.Flags().GetInt64("log-max-size"); logMaxSize > 0 {
			app.config.Logging.MaxFileSize = logMaxSize
		}
	}

	// Parse log max files
	if cmd.Flags().Changed("log-max-files") {
		if logMaxFiles, _ := cmd.Flags().GetInt("log-max-files"); logMaxFiles >= 0 {
			app.config.Logging.MaxFiles = logMaxFiles
		}
	}

	// Parse log buffer size
	if cmd.Flags().Changed("log-buffer-size") {
		if logBufferSize, _ := cmd.Flags().GetInt("log-buffer-size"); logBufferSize >= 0 {
			app.config.Logging.BufferSize = logBufferSize
		}
	}

	return nil
}

// parseKDFParams parses KDF parameters from JSON string
func (app *Application) parseKDFParams(kdfParamsStr string) error {
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(kdfParamsStr), &params); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// Validate parameters based on KDF algorithm
	switch strings.ToLower(app.config.KeyStore.KDFAlgorithm) {
	case "scrypt":
		if err := app.validateScryptParams(params); err != nil {
			return fmt.Errorf("invalid scrypt parameters: %w", err)
		}
	case "pbkdf2", "pbkdf2-sha256", "pbkdf2-sha512":
		if err := app.validatePBKDF2Params(params); err != nil {
			return fmt.Errorf("invalid PBKDF2 parameters: %w", err)
		}
	default:
		return fmt.Errorf("unsupported KDF algorithm: %s", app.config.KeyStore.KDFAlgorithm)
	}

	app.config.KeyStore.KDFParams = params
	return nil
}

// validateScryptParams validates scrypt parameters
func (app *Application) validateScryptParams(params map[string]interface{}) error {
	// Check required parameters
	requiredParams := []string{"n", "r", "p", "dklen"}
	for _, param := range requiredParams {
		if _, exists := params[param]; !exists {
			return fmt.Errorf("missing required parameter: %s", param)
		}
	}

	// Validate N parameter (must be power of 2)
	if n, ok := params["n"].(float64); ok {
		nInt := int(n)
		if nInt <= 0 || (nInt&(nInt-1)) != 0 {
			return fmt.Errorf("n parameter must be a positive power of 2, got %d", nInt)
		}
		if nInt < 1024 || nInt > 67108864 {
			return fmt.Errorf("n parameter must be between 1024 and 67108864, got %d", nInt)
		}
	} else {
		return fmt.Errorf("n parameter must be a number")
	}

	// Validate r parameter
	if r, ok := params["r"].(float64); ok {
		rInt := int(r)
		if rInt <= 0 || rInt > 256 {
			return fmt.Errorf("r parameter must be between 1 and 256, got %d", rInt)
		}
	} else {
		return fmt.Errorf("r parameter must be a number")
	}

	// Validate p parameter
	if p, ok := params["p"].(float64); ok {
		pInt := int(p)
		if pInt <= 0 || pInt > 256 {
			return fmt.Errorf("p parameter must be between 1 and 256, got %d", pInt)
		}
	} else {
		return fmt.Errorf("p parameter must be a number")
	}

	// Validate dklen parameter
	if dklen, ok := params["dklen"].(float64); ok {
		dklenInt := int(dklen)
		if dklenInt <= 0 || dklenInt > 128 {
			return fmt.Errorf("dklen parameter must be between 1 and 128, got %d", dklenInt)
		}
	} else {
		return fmt.Errorf("dklen parameter must be a number")
	}

	return nil
}

// validatePBKDF2Params validates PBKDF2 parameters
func (app *Application) validatePBKDF2Params(params map[string]interface{}) error {
	// Check required parameters
	requiredParams := []string{"c", "dklen"}
	for _, param := range requiredParams {
		if _, exists := params[param]; !exists {
			return fmt.Errorf("missing required parameter: %s", param)
		}
	}

	// Validate c parameter (iteration count)
	if c, ok := params["c"].(float64); ok {
		cInt := int(c)
		if cInt < 100000 {
			return fmt.Errorf("c parameter (iteration count) must be at least 100000 for security, got %d", cInt)
		}
		if cInt > 10000000 {
			return fmt.Errorf("c parameter (iteration count) too high (max 10000000), got %d", cInt)
		}
	} else {
		return fmt.Errorf("c parameter must be a number")
	}

	// Validate dklen parameter
	if dklen, ok := params["dklen"].(float64); ok {
		dklenInt := int(dklen)
		if dklenInt <= 0 || dklenInt > 128 {
			return fmt.Errorf("dklen parameter must be between 1 and 128, got %d", dklenInt)
		}
	} else {
		return fmt.Errorf("dklen parameter must be a number")
	}

	// Validate prf parameter if provided
	if prf, exists := params["prf"]; exists {
		if prfStr, ok := prf.(string); ok {
			validPRFs := []string{"hmac-sha256", "hmac-sha512"}
			valid := false
			for _, validPRF := range validPRFs {
				if prfStr == validPRF {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("prf parameter must be one of %v, got %s", validPRFs, prfStr)
			}
		} else {
			return fmt.Errorf("prf parameter must be a string")
		}
	}

	return nil
}

// extractKDFParamsFromKeystore extracts complete KDF parameters from a generated keystore
func (app *Application) extractKDFParamsFromKeystore(keystore *crypto.KeyStoreV3) map[string]interface{} {
	params := make(map[string]interface{})

	switch strings.ToLower(keystore.Crypto.KDF) {
	case "scrypt":
		if scryptParams, err := keystore.GetScryptParams(); err == nil && scryptParams != nil {
			params["n"] = scryptParams.N
			params["r"] = scryptParams.R
			params["p"] = scryptParams.P
			params["dklen"] = scryptParams.DKLen
			params["salt"] = scryptParams.Salt
		}
	case "pbkdf2":
		if pbkdf2Params, err := keystore.GetPBKDF2Params(); err == nil && pbkdf2Params != nil {
			params["c"] = pbkdf2Params.C
			params["dklen"] = pbkdf2Params.DKLen
			params["prf"] = pbkdf2Params.PRF
			params["salt"] = pbkdf2Params.Salt
		}
	}

	return params
}

// parseSecurityLevel converts string security level to SecurityLevel enum
func (app *Application) parseSecurityLevel(level string) kdf.SecurityLevel {
	switch strings.ToLower(level) {
	case "low":
		return kdf.SecurityLevelLow
	case "medium":
		return kdf.SecurityLevelMedium
	case "high":
		return kdf.SecurityLevelHigh
	case "very-high", "veryhigh":
		return kdf.SecurityLevelVeryHigh
	default:
		return kdf.SecurityLevelMedium // Default to medium
	}
}

// displayCompatibilityReport displays KDF compatibility analysis results
func (app *Application) displayCompatibilityReport(report *kdf.CompatibilityReport, verbose bool) {
	if !verbose && !app.config.KeyStore.ShowAnalysis {
		return
	}

	fmt.Printf("\nKDF Compatibility Analysis\n")
	fmt.Printf("═══════════════════════════════════════\n")

	// Basic information
	fmt.Printf("KDF Algorithm: %s", report.KDFType)
	if report.NormalizedKDF != report.KDFType {
		fmt.Printf(" (normalized: %s)", report.NormalizedKDF)
	}
	fmt.Printf("\n")

	// Security level with color coding
	securityLabel := app.getSecurityLevelLabel(report.SecurityLevel)
	fmt.Printf("Security Level: %s (%s)\n", securityLabel, report.SecurityLevel)

	// Compatibility status
	if report.Compatible {
		fmt.Printf("Status: Compatible\n")
	} else {
		fmt.Printf("Status: Incompatible\n")
	}

	// Display parameters if verbose
	if verbose && len(report.Parameters) > 0 {
		fmt.Printf("\nParameters:\n")
		for key, value := range report.Parameters {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	// Display issues
	if len(report.Issues) > 0 {
		fmt.Printf("\nIssues:\n")
		for _, issue := range report.Issues {
			fmt.Printf("  • %s\n", issue)
		}
	}

	// Display warnings
	if len(report.Warnings) > 0 {
		fmt.Printf("\nWarnings:\n")
		for _, warning := range report.Warnings {
			fmt.Printf("  • %s\n", warning)
		}
	}

	// Display suggestions
	if len(report.Suggestions) > 0 {
		fmt.Printf("\nSuggestions:\n")
		for _, suggestion := range report.Suggestions {
			fmt.Printf("  • %s\n", suggestion)
		}
	}

	fmt.Printf("\n")
}

// getSecurityLevelLabel returns a descriptive label for the security level
func (app *Application) getSecurityLevelLabel(level kdf.SecurityLevel) string {
	switch level {
	case kdf.SecurityLevelLow:
		return "Low"
	case kdf.SecurityLevelMedium:
		return "Medium"
	case kdf.SecurityLevelHigh:
		return "High"
	case kdf.SecurityLevelVeryHigh:
		return "Very High"
	default:
		return "Unknown"
	}
}

// getGenerationCriteria extracts generation criteria from command flags
func (app *Application) getGenerationCriteria(cmd *cobra.Command) (wallet.GenerationCriteria, error) {
	prefix, _ := cmd.Flags().GetString("prefix")
	suffix, _ := cmd.Flags().GetString("suffix")
	checksum, _ := cmd.Flags().GetBool("checksum")
	caseSensitive, _ := cmd.Flags().GetBool("case-sensitive")
	useMnemonic, _ := cmd.Flags().GetBool("with-mnemonic")
	network, _ := cmd.Flags().GetString("network")

	criteria := wallet.GenerationCriteria{
		Network:       network,
		Prefix:        prefix,
		Suffix:        suffix,
		IsChecksum:    checksum,
		CaseSensitive: caseSensitive,
		UseMnemonic:   useMnemonic,
	}

	return criteria, criteria.Validate()
}

func (app *Application) getGenerationEngineOptions(cmd *cobra.Command, criteria wallet.GenerationCriteria) (engine.Selection, engine.GenerationOptions, error) {
	requestedEngine := flagStringOrEnv(cmd, "engine", envBlocoEngine)
	selection, err := engine.ResolveGeneration(requestedEngine, criteria)
	if err != nil {
		return engine.Selection{}, engine.GenerationOptions{}, err
	}

	batchSize, err := flagIntOrEnv(cmd, "gpu-batch-size", envBlocoGPUBatchSize)
	if err != nil {
		return engine.Selection{}, engine.GenerationOptions{}, err
	}
	if batchSize <= 0 {
		return engine.Selection{}, engine.GenerationOptions{}, fmt.Errorf("gpu-batch-size must be positive, got %d", batchSize)
	}

	metalValidation, err := resolveMetalValidationConfig(cmd, selection.Resolved, engine.MetalValidationFull)
	if err != nil {
		return engine.Selection{}, engine.GenerationOptions{}, err
	}

	return selection, engine.GenerationOptions{
		BatchSize:       batchSize,
		ThreadCount:     app.config.Worker.ThreadCount,
		Network:         criteria.Network,
		Criteria:        criteria,
		RequestedEngine: selection.Requested,
		FallbackReason:  selection.FallbackReason,
		MetalValidation: metalValidation,
	}, nil
}

func (app *Application) displayGenerationEngineDiagnostics(selection engine.Selection, options engine.GenerationOptions, showProgress bool) {
	if (!showProgress && !app.config.CLI.VerboseOutput) || app.config.CLI.QuietMode {
		return
	}
	fmt.Printf("Engine: %s\n", selection.Resolved)
	if selection.Requested != "" && selection.Requested != selection.Resolved {
		fmt.Printf("Requested Engine: %s\n", selection.Requested)
	}
	if selection.FallbackReason != "" {
		fmt.Printf("Fallback: %s\n", selection.FallbackReason)
	}
	if selection.Resolved == engine.NameMetal {
		if deviceName := engine.MetalDeviceName(); deviceName != "" {
			fmt.Printf("Metal Device: %s\n", deviceName)
		}
		fmt.Printf("GPU Batch Size: %d\n", options.BatchSize)
		fmt.Printf("Metal Validation: %s\n", options.MetalValidation)
	}
	fmt.Printf("\n")
}

// Helper functions using utils package
func calculateDifficulty(criteria wallet.GenerationCriteria) float64 {
	return utils.CalculateDifficulty(criteria.Prefix, criteria.Suffix, criteria.IsChecksum)
}

func calculateProbability50(difficulty float64) int64 {
	return utils.CalculateProbability50(difficulty)
}

func formatLargeNumber(num int64) string {
	return utils.FormatLargeNumber(num)
}

func formatDuration(d time.Duration) string {
	return utils.FormatDuration(d)
}

func formatBool(b bool) string {
	if b {
		return "Enabled"
	}
	return "Disabled"
}

// Placeholder implementations for display functions
func (app *Application) displayWalletResult(result *wallet.GenerationResult, showProgress bool) error {
	fmt.Printf("Wallet generated successfully!\n")
	fmt.Printf("Address: %s\n", result.Wallet.Address)
	fmt.Printf("Private Key: %s\n", result.Wallet.PrivateKey)
	if result.Wallet.Mnemonic != "" {
		fmt.Printf("Mnemonic: %s\n", result.Wallet.Mnemonic)
	}
	fmt.Printf("Attempts: %s\n", formatLargeNumber(result.Attempts))
	fmt.Printf("Duration: %v\n", result.Duration)
	if (showProgress || app.config.CLI.VerboseOutput) && result.Engine != "" {
		fmt.Printf("Engine: %s\n", result.Engine)
		if result.DeviceName != "" {
			fmt.Printf("Device: %s\n", result.DeviceName)
		}
		if result.BatchSize > 0 {
			fmt.Printf("Batch Size: %d\n", result.BatchSize)
		}
	}

	// Generate keystore if enabled
	if app.config.KeyStore.Enabled {
		if err := app.generateAndSaveKeystore(result.Wallet); err != nil {
			fmt.Printf("Warning: Failed to generate keystore: %v\n", err)
		} else {
			fmt.Printf("Keystore saved to: %s\n", app.config.KeyStore.OutputDir)
			if result.Wallet.Mnemonic != "" {
				fmt.Printf("Mnemonic saved to: %s\n", app.config.KeyStore.OutputDir)
			}
		}
	}

	return nil
}

func (app *Application) displayMultipleWalletResults(results []*wallet.GenerationResult, totalAttempts int64, totalDuration time.Duration, showProgress bool) error {
	if len(results) == 0 {
		fmt.Printf("No wallets were generated successfully\n")
		return nil
	}

	fmt.Printf("Generated %d wallets successfully!\n", len(results))
	fmt.Printf("Total attempts: %s\n", formatLargeNumber(totalAttempts))
	fmt.Printf("Total duration: %s\n", formatDuration(totalDuration))
	fmt.Printf("Average speed: %.0f addr/s\n\n", float64(totalAttempts)/totalDuration.Seconds())

	// Display individual wallets
	var keystoreErrors []error
	for i, result := range results {
		fmt.Printf("Wallet %d:\n", i+1)
		fmt.Printf("  Address: %s\n", result.Wallet.Address)

		// Only show private key if not in quiet mode
		if !app.config.CLI.QuietMode {
			fmt.Printf("  Private Key: %s\n", result.Wallet.PrivateKey)
			if result.Wallet.Mnemonic != "" {
				fmt.Printf("  Mnemonic: %s\n", result.Wallet.Mnemonic)
			}
		}

		fmt.Printf("  Attempts: %s\n", formatLargeNumber(result.Attempts))
		fmt.Printf("  Duration: %s\n", formatDuration(result.Duration))
		if (showProgress || app.config.CLI.VerboseOutput) && result.Engine != "" {
			fmt.Printf("  Engine: %s\n", result.Engine)
			if result.DeviceName != "" {
				fmt.Printf("  Device: %s\n", result.DeviceName)
			}
			if result.BatchSize > 0 {
				fmt.Printf("  Batch Size: %d\n", result.BatchSize)
			}
		}

		if result.WorkerID > 0 {
			fmt.Printf("  Worker: #%d\n", result.WorkerID)
		}

		// Generate keystore if enabled
		if app.config.KeyStore.Enabled {
			if err := app.generateAndSaveKeystore(result.Wallet); err != nil {
				keystoreErrors = append(keystoreErrors, err)
				fmt.Printf("  Keystore: Failed to generate (%v)\n", err)
			} else {
				fmt.Printf("  Keystore: Saved\n")
				if result.Wallet.Mnemonic != "" {
					fmt.Printf("  Mnemonic: Saved\n")
				}
			}
		}

		fmt.Printf("\n")
	}

	// Show keystore summary
	if app.config.KeyStore.Enabled {
		successCount := len(results) - len(keystoreErrors)
		if successCount > 0 {
			fmt.Printf("Keystores saved: %d/%d to %s\n", successCount, len(results), app.config.KeyStore.OutputDir)
		}
		if len(keystoreErrors) > 0 {
			fmt.Printf("Keystore errors: %d/%d\n", len(keystoreErrors), len(results))
		}
	}

	// Show statistics summary
	if len(results) > 1 {
		minAttempts, maxAttempts := results[0].Attempts, results[0].Attempts
		var totalWalletAttempts int64

		for _, result := range results {
			totalWalletAttempts += result.Attempts
			if result.Attempts < minAttempts {
				minAttempts = result.Attempts
			}
			if result.Attempts > maxAttempts {
				maxAttempts = result.Attempts
			}
		}

		avgAttempts := totalWalletAttempts / int64(len(results))

		fmt.Printf("Statistics Summary:\n")
		fmt.Printf("  Average attempts per wallet: %s\n", formatLargeNumber(avgAttempts))
		fmt.Printf("  Min attempts: %s\n", formatLargeNumber(minAttempts))
		fmt.Printf("  Max attempts: %s\n", formatLargeNumber(maxAttempts))
		fmt.Printf("  Success rate: %.2f%%\n", float64(len(results))/float64(totalAttempts)*100)
	}

	return nil
}

// GetRootCommand returns the root command for fang integration
func (app *Application) GetRootCommand() *cobra.Command {
	return app.rootCmd
}

// StatsManagerAdapter adapts the worker stats collector to the TUI interface
type StatsManagerAdapter struct {
	collector *worker.StatsCollector
}

func (sma *StatsManagerAdapter) GetMetrics() tui.ThreadMetrics {
	stats := sma.collector.GetAggregatedStats()
	perfMetrics := sma.collector.GetPerformanceMetrics()

	return tui.ThreadMetrics{
		EfficiencyRatio: perfMetrics.EfficiencyRatio,
		TotalSpeed:      stats.TotalSpeed,
		ThreadCount:     perfMetrics.WorkerCount,
	}
}

func (sma *StatsManagerAdapter) GetPeakSpeed() float64 {
	stats := sma.collector.GetAggregatedStats()
	return stats.PeakSpeed
}

// generateAndSaveKeystore generates and saves a keystore file for the given wallet
func (app *Application) generateAndSaveKeystore(w *wallet.Wallet) error {
	return app.generateAndSaveKeystoreWithVerbose(w, app.config.CLI.VerboseOutput)
}

// generateAndSaveKeystoreWithVerbose generates and saves a keystore file with verbose control
// For Bitcoin: only saves mnemonic (no KeyStore V3)
// For Ethereum and Solana: generates KeyStore V3 or network-specific format
func (app *Application) generateAndSaveKeystoreWithVerbose(w *wallet.Wallet, verbose bool) error {
	// Bitcoin only saves mnemonic, no KeyStore V3
	if strings.ToLower(w.Network) == "bitcoin" {
		if w.Mnemonic == "" {
			return fmt.Errorf("bitcoin wallet requires mnemonic for backup")
		}

		// Create keystore service just for saving mnemonic
		keystoreConfig := crypto.KeyStoreConfig{
			Enabled:         app.config.KeyStore.Enabled,
			OutputDirectory: app.config.KeyStore.OutputDir,
		}
		keystoreService := crypto.NewKeyStoreService(keystoreConfig)
		keystoreService.SetVerboseMode(verbose)

		// Save only the mnemonic for Bitcoin
		if err := keystoreService.SaveMnemonicFile(w.Address, w.Mnemonic, w.Network); err != nil {
			if ksErr, ok := err.(*crypto.KeyStoreError); ok {
				if ksErr.UserMessage != "" {
					return fmt.Errorf("mnemonic save failed: %s", ksErr.UserMessage)
				}
				return fmt.Errorf("mnemonic save failed for address %s: %v", w.Address, err)
			}
			return fmt.Errorf("failed to save mnemonic file for address %s: %w", w.Address, err)
		}
		return nil
	}

	// For Ethereum and Solana: generate KeyStore V3 or network-specific format
	// Create Universal KDF service for enhanced compatibility
	kdfService := kdf.NewUniversalKDFService()

	// Create compatibility analyzer
	analyzer := kdf.NewKDFCompatibilityAnalyzer(kdfService)

	// Prepare KDF parameters
	kdfParams := app.config.KeyStore.KDFParams
	if len(kdfParams) == 0 {
		// Use default parameters based on security level
		securityLevel := app.parseSecurityLevel(app.config.KeyStore.SecurityLevel)
		defaultParams, err := analyzer.GetOptimizedParams(app.config.KeyStore.KDFAlgorithm, securityLevel, 512) // 512MB max memory
		if err != nil {
			return fmt.Errorf("failed to get default KDF parameters: %w", err)
		}
		kdfParams = defaultParams
	}

	// Create keystore service configuration with Universal KDF
	keystoreConfig := crypto.KeyStoreConfig{
		Enabled:         app.config.KeyStore.Enabled,
		OutputDirectory: app.config.KeyStore.OutputDir,
		KDF:             app.config.KeyStore.KDFAlgorithm,
		KDFParams:       kdfParams,
		Cipher:          "aes-128-ctr",
		MaxRetries:      3,
		RetryDelay:      100, // 100ms
	}

	// Create keystore service with controlled verbose logging
	keystoreService := crypto.NewKeyStoreService(keystoreConfig)
	keystoreService.SetVerboseMode(verbose)

	// Generate keystore first to get complete parameters
	keystore, password, err := keystoreService.GenerateKeyStore(w.PrivateKey, w.Address, w.Network)
	if err != nil {
		return fmt.Errorf("failed to generate keystore for address %s: %w", w.Address, err)
	}

	// Perform compatibility analysis after keystore generation when all parameters are available
	if app.config.KeyStore.ShowAnalysis || verbose {
		// Extract complete KDF parameters from generated keystore
		completeParams := app.extractKDFParamsFromKeystore(keystore)

		cryptoParamsComplete := &kdf.CryptoParams{
			KDF:       app.config.KeyStore.KDFAlgorithm,
			KDFParams: completeParams,
		}

		report, err := analyzer.AnalyzeKeystore(cryptoParamsComplete)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to analyze KDF compatibility: %v\n", err)
			}
		} else {
			app.displayCompatibilityReport(report, verbose)
		}
	}

	// Save the generated keystore
	if err := keystoreService.SaveKeyStoreFilesToDisk(w.Address, keystore, password, w.Network, w.PrivateKey); err != nil {
		// Check if it's a KeyStoreError for better error reporting
		if ksErr, ok := err.(*crypto.KeyStoreError); ok {
			if ksErr.UserMessage != "" {
				return fmt.Errorf("keystore generation failed: %s", ksErr.UserMessage)
			}
			return fmt.Errorf("keystore generation failed for address %s: %v", w.Address, err)
		}
		return fmt.Errorf("failed to save keystore files for address %s: %w", w.Address, err)
	}

	if w.Mnemonic != "" {
		if err := keystoreService.SaveMnemonicFile(w.Address, w.Mnemonic, w.Network); err != nil {
			if ksErr, ok := err.(*crypto.KeyStoreError); ok {
				if ksErr.UserMessage != "" {
					return fmt.Errorf("mnemonic save failed: %s", ksErr.UserMessage)
				}
				return fmt.Errorf("mnemonic save failed for address %s: %v", w.Address, err)
			}
			return fmt.Errorf("failed to save mnemonic file for address %s: %w", w.Address, err)
		}
	}

	return nil
}
