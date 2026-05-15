package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"bloco-vgen/pkg/wallet"
)

// TUIManager handles TUI capability detection and management
type TUIManager struct {
	enabled        bool
	terminalWidth  int
	terminalHeight int
	colorSupport   bool
}

// TUICapabilities represents terminal capabilities
type TUICapabilities struct {
	SupportsColor   bool
	SupportsUnicode bool
	TerminalWidth   int
	TerminalHeight  int
	SupportsResize  bool
}

// StatsManager interface for collecting worker statistics (forward declaration for manager)
type StatsManager interface {
	GetMetrics() ThreadMetrics
	GetPeakSpeed() float64
}

// ThreadMetrics represents thread utilization metrics (forward declaration for manager)
type ThreadMetrics struct {
	EfficiencyRatio float64
	TotalSpeed      float64
	ThreadCount     int
}

// NewTUIManager creates a new TUI manager instance
func NewTUIManager() *TUIManager {
	return &TUIManager{}
}

// DetectCapabilities detects terminal capabilities
func (tm *TUIManager) DetectCapabilities() TUICapabilities {
	capabilities := TUICapabilities{
		SupportsColor:   tm.detectColorSupport(),
		SupportsUnicode: tm.detectUnicodeSupport(),
		TerminalWidth:   80, // Default fallback
		TerminalHeight:  24, // Default fallback
		SupportsResize:  true,
	}

	// Detect terminal size
	if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		capabilities.TerminalWidth = width
		capabilities.TerminalHeight = height
		tm.terminalWidth = width
		tm.terminalHeight = height
	}

	// Check if resize is supported (assume true unless proven otherwise)
	termType := strings.ToLower(os.Getenv("TERM"))
	if termType == "dumb" || termType == "" {
		capabilities.SupportsResize = false
	}

	// Update internal state
	tm.colorSupport = capabilities.SupportsColor

	return capabilities
}

// detectColorSupport checks if the terminal supports colors
func (tm *TUIManager) detectColorSupport() bool {
	// Check NO_COLOR environment variable first (https://no-color.org/)
	// This should override all other color detection
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check TERM environment variable
	termType := strings.ToLower(os.Getenv("TERM"))

	// Explicitly unsupported terminals
	if termType == "dumb" || termType == "" {
		return false
	}

	// Monochrome terminals
	if strings.Contains(termType, "mono") {
		return false
	}

	// Explicitly supported color terminals
	if strings.Contains(termType, "color") ||
		strings.Contains(termType, "256color") ||
		strings.Contains(termType, "truecolor") ||
		strings.Contains(termType, "xterm") ||
		strings.Contains(termType, "screen") ||
		strings.Contains(termType, "tmux") {
		return true
	}

	// Check COLORTERM environment variable
	if colorTerm := os.Getenv("COLORTERM"); colorTerm != "" {
		colorTerm = strings.ToLower(colorTerm)
		if strings.Contains(colorTerm, "truecolor") ||
			strings.Contains(colorTerm, "24bit") {
			return true
		}
	}

	// Default to true for most modern terminals
	return true
}

// detectUnicodeSupport checks if the terminal supports Unicode
func (tm *TUIManager) detectUnicodeSupport() bool {
	// Check LANG environment variable
	lang := strings.ToUpper(os.Getenv("LANG"))
	if lang != "" {
		if strings.Contains(lang, "UTF-8") || strings.Contains(lang, "UTF8") {
			return true
		}
		// If LANG is set but doesn't contain UTF, assume no Unicode support
		if lang != "" && !strings.Contains(lang, "UTF") {
			return false
		}
	}

	// Check LC_ALL environment variable
	lcAll := strings.ToUpper(os.Getenv("LC_ALL"))
	if lcAll != "" {
		if strings.Contains(lcAll, "UTF-8") || strings.Contains(lcAll, "UTF8") {
			return true
		}
		if !strings.Contains(lcAll, "UTF") {
			return false
		}
	}

	// Check LC_CTYPE environment variable
	lcCtype := strings.ToUpper(os.Getenv("LC_CTYPE"))
	if lcCtype != "" {
		if strings.Contains(lcCtype, "UTF-8") || strings.Contains(lcCtype, "UTF8") {
			return true
		}
		if !strings.Contains(lcCtype, "UTF") {
			return false
		}
	}

	// If no locale information is available, default to false for safety
	// This ensures we don't assume Unicode support without evidence
	return false
}

// ShouldUseTUI determines if TUI should be used based on environment and capabilities
func (tm *TUIManager) ShouldUseTUI() bool {
	// Check if TUI is explicitly disabled via environment variable
	if tuiEnv := os.Getenv("BLOCO_TUI"); tuiEnv != "" {
		// Parse boolean values: "false", "0", "no", "off" disable TUI
		tuiEnv = strings.ToLower(strings.TrimSpace(tuiEnv))
		if tuiEnv == "false" || tuiEnv == "0" || tuiEnv == "no" || tuiEnv == "off" {
			return false
		}
		// "true", "1", "yes", "on", "force" enable TUI (but still check other conditions)
		if tuiEnv == "true" || tuiEnv == "1" || tuiEnv == "yes" || tuiEnv == "on" || tuiEnv == "force" {
			// If forced, skip most other checks except very critical ones
			if tuiEnv == "force" {
				return true
			}
			// Continue with other checks even if explicitly enabled
		}
	}

	// Check terminal type first - hard block on unsuitable terminals
	termType := strings.ToLower(os.Getenv("TERM"))
	if termType == "" || termType == "dumb" {
		return false
	}

	// Check if we're in an interactive terminal
	// For Claude Code and development environments, allow TUI if TERM is set properly
	isStdoutTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	isStdinTerminal := term.IsTerminal(int(os.Stdin.Fd()))

	// If TERM environment suggests we have a capable terminal, allow TUI even without TTY detection
	hasCapableTerminal := strings.Contains(termType, "xterm") || strings.Contains(termType, "screen") ||
		strings.Contains(termType, "tmux") || strings.Contains(termType, "color")

	// In development environments, be more lenient with TUI detection
	if !isStdoutTerminal && !isStdinTerminal {
		if os.Getenv("BLOCO_DEBUG") != "" {
			fmt.Printf("DEBUG TUI: neither stdout nor stdin are terminals, TERM=%s\n", termType)
		}

		// If we have a capable terminal environment OR we're in a development setup, allow TUI
		if !hasCapableTerminal {
			// Check if we're in a development environment that might support TUI
			if os.Getenv("VSCODE_INJECTION") != "" || os.Getenv("TERM_PROGRAM") != "" ||
				os.Getenv("COLORTERM") != "" || len(os.Getenv("TERM")) > 0 {
				if os.Getenv("BLOCO_DEBUG") != "" {
					fmt.Printf("DEBUG TUI: development environment detected, allowing TUI\n")
				}
			} else {
				if os.Getenv("BLOCO_DEBUG") != "" {
					fmt.Printf("DEBUG TUI: no capable terminal detected, disabling TUI\n")
				}
				return false
			}
		} else {
			if os.Getenv("BLOCO_DEBUG") != "" {
				fmt.Printf("DEBUG TUI: capable terminal environment detected, allowing TUI\n")
			}
		}
	}

	// Check basic terminal capabilities
	capabilities := tm.DetectCapabilities()

	// Require minimum terminal size for usable TUI
	if capabilities.TerminalWidth < 40 || capabilities.TerminalHeight < 10 {
		if os.Getenv("BLOCO_DEBUG") != "" {
			fmt.Printf("DEBUG TUI: terminal too small (%dx%d)\n", capabilities.TerminalWidth, capabilities.TerminalHeight)
		}
		return false
	}

	// Terminal type already checked above, skip duplicate check

	// Check if we're running in a CI environment
	if tm.isInCIEnvironment() {
		if os.Getenv("BLOCO_DEBUG") != "" {
			fmt.Printf("DEBUG TUI: CI environment detected\n")
		}
		return false
	}

	// Check if output is being redirected
	if tm.isOutputRedirected() {
		if os.Getenv("BLOCO_DEBUG") != "" {
			fmt.Printf("DEBUG TUI: output is redirected\n")
		}
		return false
	}

	// Update internal enabled state
	tm.enabled = true
	return true
}

// isInCIEnvironment checks if we're running in a CI/CD environment
func (tm *TUIManager) isInCIEnvironment() bool {
	ciEnvVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "BUILD_NUMBER", "JENKINS_URL",
		"TRAVIS", "CIRCLECI", "APPVEYOR", "GITLAB_CI", "BUILDKITE",
		"DRONE", "GITHUB_ACTIONS", "TF_BUILD", "TEAMCITY_VERSION",
	}

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// isOutputRedirected checks if stdout is being redirected
func (tm *TUIManager) isOutputRedirected() bool {
	// This is already partially covered by term.IsTerminal check,
	// but we can add additional checks here if needed
	return false
}

// CreateProgressModel creates a progress TUI model
func (tm *TUIManager) CreateProgressModel(stats *wallet.GenerationStats, statsManager StatsManager) tea.Model {
	return NewProgressModel(stats, statsManager)
}

// CreateProgressModelWithEngine creates a progress TUI model preloaded with
// engine diagnostics so the running view shows engine, device and batch info
// equivalent to the text-mode block.
func (tm *TUIManager) CreateProgressModelWithEngine(stats *wallet.GenerationStats, statsManager StatsManager, engineInfo EngineInfo) tea.Model {
	return NewProgressModel(stats, statsManager).WithEngineInfo(engineInfo)
}

// CreateBenchmarkModel creates a benchmark TUI model
func (tm *TUIManager) CreateBenchmarkModel() tea.Model {
	return NewBenchmarkModel()
}

// CreateBenchmarkModelWithEngine creates a benchmark TUI model preloaded with
// engine diagnostics so the progress and results views include the same engine,
// device and batch information rendered by runBenchmarkText.
func (tm *TUIManager) CreateBenchmarkModelWithEngine(engineInfo EngineInfo) tea.Model {
	return NewBenchmarkModel().WithEngineInfo(engineInfo)
}

func (tm *TUIManager) CreateBenchmarkComparisonModel(summary BenchmarkComparisonSummary, total int) tea.Model {
	return NewBenchmarkComparisonModel(summary, total)
}

// CreateStatsModel creates a statistics TUI model
func (tm *TUIManager) CreateStatsModel(stats *wallet.GenerationStats) tea.Model {
	return NewStatsModel(stats)
}

func (tm *TUIManager) CreateStatsModelWithEngine(stats *wallet.GenerationStats, engineInfo EngineInfo) tea.Model {
	return NewStatsModel(stats).WithEngineInfo(engineInfo)
}

// GetCapabilities returns the current terminal capabilities
func (tm *TUIManager) GetCapabilities() TUICapabilities {
	return tm.DetectCapabilities()
}

// IsEnabled returns whether TUI is currently enabled
func (tm *TUIManager) IsEnabled() bool {
	return tm.enabled
}

// SetEnabled allows manual override of TUI enabled state
func (tm *TUIManager) SetEnabled(enabled bool) {
	tm.enabled = enabled
}

// GetTerminalSize returns the current terminal dimensions
func (tm *TUIManager) GetTerminalSize() (width, height int) {
	if tm.terminalWidth > 0 && tm.terminalHeight > 0 {
		return tm.terminalWidth, tm.terminalHeight
	}

	if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		tm.terminalWidth = width
		tm.terminalHeight = height
		return width, height
	}

	// Return default values if detection fails
	return 80, 24
}

// SupportsColor returns whether the terminal supports colors
func (tm *TUIManager) SupportsColor() bool {
	return tm.colorSupport
}
