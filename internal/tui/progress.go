package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"bloco-vgen/pkg/wallet"
)

const (
	padding  = 2
	maxWidth = 80
)

// ProgressModel represents the progress TUI component
type ProgressModel struct {
	progress         progress.Model
	stats            *wallet.GenerationStats
	statsManager     StatsManager
	styleManager     *StyleManager
	width            int
	height           int
	quitting         bool
	lastUpdate       time.Time
	walletResults    []WalletResult
	showResults      bool
	resultsTable     table.Model
	completedWallets int  // Number of wallets completed
	totalWallets     int  // Total wallets requested
	isComplete       bool // Indicates if generation is complete
	engineInfo       EngineInfo
}

// EngineInfo describes the active generation/benchmark engine for TUI rendering.
// It mirrors the diagnostics printed in text mode so the TUI displays the same
// information without requiring the user to read raw stdout output.
type EngineInfo struct {
	Engine          string
	RequestedEngine string
	FallbackReason  string
	DeviceName      string
	BatchSize       int
	MetalValidation string
	ThreadCount     int
	Network         string
}

// IsZero reports whether EngineInfo carries any diagnostic information.
func (info EngineInfo) IsZero() bool {
	return info.Engine == "" &&
		info.RequestedEngine == "" &&
		info.FallbackReason == "" &&
		info.DeviceName == "" &&
		info.BatchSize == 0 &&
		info.MetalValidation == "" &&
		info.ThreadCount == 0 &&
		info.Network == ""
}

// ProgressMsg represents a progress update message
type ProgressMsg struct {
	Attempts         int64
	Speed            float64
	Probability      float64
	EstimatedTime    time.Duration
	Difficulty       float64
	Pattern          string
	CompletedWallets int     // Number of wallets successfully generated
	TotalWallets     int     // Total wallets requested
	ProgressPercent  float64 // Progress as percentage (0-100) for progress bar
	IsComplete       bool    // Indicates if generation is complete
}

// TickMsg represents a timer tick for smooth animations
type TickMsg time.Time

// QuitMsg represents a quit signal
type QuitMsg struct{}

// WalletResult represents a generated wallet result for TUI display
type WalletResult struct {
	Index      int
	Address    string
	PrivateKey string
	Attempts   int
	Time       time.Duration
	Error      string
}

// WalletResultMsg represents a wallet generation result message
type WalletResultMsg struct {
	Result WalletResult
}

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

// NewProgressModel creates a new progress model
func NewProgressModel(stats *wallet.GenerationStats, statsManager StatsManager) ProgressModel {
	// Create progress bar with gradient styling following Bubbletea pattern
	prog := progress.New(progress.WithDefaultGradient())

	// Create style manager with terminal capabilities
	tuiManager := NewTUIManager()
	capabilities := tuiManager.DetectCapabilities()
	styleManager := NewStyleManagerWithCapabilities(capabilities)

	// Create results table with columns for wallet information
	columns := []table.Column{
		{Title: "№", Width: 3},
		{Title: "Address", Width: 42},
		{Title: "Private Key", Width: 64},
		{Title: "Attempts", Width: 10},
		{Title: "Time", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true), // Enable focus for scroll functionality
		table.WithHeight(8),     // Show up to 8 wallets at a time (leaving space for progress info)
	)

	// Set table styles to match TUI theme
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(PrimaryColor)).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(TextPrimary)).
		Background(lipgloss.Color(PrimaryColor)).
		Bold(false)
	t.SetStyles(s)

	return ProgressModel{
		progress:         prog,
		stats:            stats,
		statsManager:     statsManager,
		styleManager:     styleManager,
		width:            capabilities.TerminalWidth,
		height:           capabilities.TerminalHeight,
		quitting:         false,
		lastUpdate:       time.Now(),
		resultsTable:     t,
		completedWallets: 0,
		totalWallets:     1, // Default to 1 for single wallet generation
		isComplete:       false,
	}
}

// WithEngineInfo returns a copy of the model with engine diagnostics attached.
// The diagnostics are rendered in the progress view to match the text-mode
// engine block printed by displayGenerationEngineDiagnostics.
func (m ProgressModel) WithEngineInfo(info EngineInfo) ProgressModel {
	m.engineInfo = info
	return m
}

// Init initializes the progress model
func (m ProgressModel) Init() tea.Cmd {
	return m.tickCmd()
}

// Update handles messages and updates the model
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			// Navigate table up if we have results
			if m.showResults && len(m.walletResults) > 0 {
				var cmd tea.Cmd
				m.resultsTable, cmd = m.resultsTable.Update(msg)
				return m, cmd
			}
		case "down", "j":
			// Navigate table down if we have results
			if m.showResults && len(m.walletResults) > 0 {
				var cmd tea.Cmd
				m.resultsTable, cmd = m.resultsTable.Update(msg)
				return m, cmd
			}
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case TickMsg:
		if m.quitting {
			return m, tea.Quit
		}

		// Just continue ticking for animations, progress bar is updated via ProgressMsg
		return m, m.tickCmd()

	case ProgressMsg:
		// Update internal statistics from external progress updates
		if m.stats != nil {
			m.stats.CurrentAttempts = msg.Attempts
			m.stats.Speed = msg.Speed
			m.stats.Probability = msg.Probability
			m.stats.EstimatedTime = msg.EstimatedTime
			m.stats.Difficulty = msg.Difficulty
			m.stats.Pattern = msg.Pattern
			m.stats.LastUpdate = time.Now()

			// Update wallet progress tracking
			m.completedWallets = msg.CompletedWallets
			m.totalWallets = msg.TotalWallets
			m.isComplete = msg.IsComplete // Update completion status

			// Update progress bar with correct percentage (wallets completed vs total)
			progressPercent := msg.ProgressPercent / 100.0
			if progressPercent > 1.0 {
				progressPercent = 1.0
			}

			// If generation is complete, ensure progress shows 100%
			if msg.IsComplete || (msg.CompletedWallets >= msg.TotalWallets && msg.TotalWallets > 0) {
				progressPercent = 1.0
				m.stats.Probability = 100.0
				m.stats.EstimatedTime = 0
				m.isComplete = true
			}

			cmd := m.progress.SetPercent(progressPercent)
			return m, cmd
		}
		return m, nil

	case WalletResultMsg:
		// Add new wallet result to the list
		m.walletResults = append(m.walletResults, msg.Result)
		m.showResults = true

		// Increment completed wallets count
		m.completedWallets = len(m.walletResults)

		// Update the table with new data
		m.updateResultsTable()

		// Update progress to show completion
		if m.totalWallets > 0 {
			progressPercent := float64(m.completedWallets) / float64(m.totalWallets)
			if progressPercent > 1.0 {
				progressPercent = 1.0
			}
			if m.completedWallets >= m.totalWallets {
				m.isComplete = true
				progressPercent = 1.0
				if m.stats != nil {
					m.stats.Probability = 100.0
					m.stats.EstimatedTime = 0
				}
			}
			cmd := m.progress.SetPercent(progressPercent)
			return m, cmd
		}

		return m, nil

	case QuitMsg:
		m.quitting = true
		return m, tea.Quit

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		// Try to update the table with any other messages
		var cmd tea.Cmd
		m.resultsTable, cmd = m.resultsTable.Update(msg)
		return m, cmd
	}
}

// View renders the progress display
func (m ProgressModel) View() string {
	if m.stats == nil {
		return m.styleManager.FormatError("No statistics available")
	}

	pad := strings.Repeat(" ", padding)
	var content strings.Builder

	// Title section
	content.WriteString(renderBlocoLogo(pad))
	content.WriteString("\n")
	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatTitle("Vanity Generator"))

	// Engine diagnostics (same fields as the text-mode block)
	if engineBlock := m.renderEngineInfo(); engineBlock != "" {
		content.WriteString("\n")
		content.WriteString(engineBlock)
	}

	// ALWAYS show progress information first (pattern, difficulty, progress bar, stats)

	// Pattern information
	content.WriteString(pad)
	pattern := m.stats.Pattern
	if len(pattern) == 0 {
		pattern = "any"
	}
	patternInfo := m.styleManager.FormatKeyValue("Pattern", pattern)
	if m.stats.IsChecksum {
		patternInfo += " " + m.styleManager.FormatHighlight("(checksum)")
	}
	content.WriteString(patternInfo)
	content.WriteString("\n")

	// Difficulty information
	content.WriteString(pad)
	difficultyStr := formatLargeNumber(int64(m.stats.Difficulty))
	content.WriteString(m.styleManager.FormatKeyValue("Difficulty", difficultyStr))
	content.WriteString("\n")

	// Progress bar - following Bubbletea pattern
	content.WriteString(pad)
	content.WriteString(m.progress.View())
	content.WriteString("\n")

	// Progress information
	content.WriteString(pad)
	var progressText string
	if m.totalWallets > 0 {
		// Show wallets completed vs total when we know the total
		progressText = fmt.Sprintf("%d/%d wallets completed (%.1f%%)",
			m.completedWallets,
			m.totalWallets,
			(float64(m.completedWallets)/float64(m.totalWallets))*100.0)
	} else if len(m.walletResults) > 0 {
		// Fallback to wallet results count
		progressText = fmt.Sprintf("%d wallets generated", len(m.walletResults))
	} else {
		// Show probability when no results yet
		progressText = fmt.Sprintf("%.2f%% probability", m.stats.Probability)
	}
	content.WriteString(m.styleManager.FormatHighlight(progressText))
	content.WriteString("\n\n")

	// Statistics section using Bubbletea table-like display
	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatSubtitle("Statistics"))
	content.WriteString("\n")

	// Create formatted statistics display
	statsDisplay := m.renderStatsTable()
	content.WriteString(statsDisplay)

	// Thread information if available
	if m.statsManager != nil {
		metrics := m.statsManager.GetMetrics()
		if metrics.ThreadCount > 1 {
			content.WriteString("\n")
			content.WriteString(pad)
			content.WriteString(m.styleManager.FormatSubtitle("Thread Performance"))
			content.WriteString("\n")

			threadDisplay := m.renderThreadStats(metrics)
			content.WriteString(threadDisplay)
		}
	}

	// ADD results table BELOW progress information when we have generated wallets
	if m.showResults && len(m.walletResults) > 0 {
		content.WriteString("\n")
		content.WriteString(pad)
		content.WriteString(m.styleManager.FormatSubtitle(fmt.Sprintf("Generated Wallets (%d)", len(m.walletResults))))
		content.WriteString("\n")
		content.WriteString(pad)
		content.WriteString(m.resultsTable.View())
		content.WriteString("\n")
	}

	// Help text
	content.WriteString("\n")
	content.WriteString(pad)
	if m.showResults && len(m.walletResults) > 8 {
		content.WriteString(helpStyle("Use ↑↓/j/k to scroll table • Press q to quit • Ctrl+C to exit"))
	} else {
		content.WriteString(helpStyle("Press q to quit • Ctrl+C to exit"))
	}

	return content.String()
}

func (m ProgressModel) Quitting() bool {
	return m.quitting
}

// tickCmd returns a command that sends a tick message after a short delay
func (m ProgressModel) tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// renderEngineInfo renders the engine diagnostics block. It mirrors the fields
// displayed by displayGenerationEngineDiagnostics in the CLI text path so the
// TUI provides equivalent visibility into engine, device and batch sizing.
func (m ProgressModel) renderEngineInfo() string {
	if m.engineInfo.IsZero() {
		return ""
	}

	pad := strings.Repeat(" ", padding)
	var content strings.Builder

	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatSubtitle("Engine"))
	content.WriteString("\n")

	rows := engineInfoRows(m.engineInfo)
	for _, row := range rows {
		content.WriteString(pad)
		content.WriteString(m.styleManager.FormatKeyValue(row.label, row.value))
		content.WriteString("\n")
	}

	return content.String()
}

// engineInfoRows returns the ordered key/value pairs to render for an EngineInfo.
// Empty values are skipped so the TUI matches the text-mode behavior of only
// showing fields that are actually populated for the active engine.
func engineInfoRows(info EngineInfo) []struct {
	label string
	value string
} {
	rows := make([]struct {
		label string
		value string
	}, 0, 7)
	add := func(label, value string) {
		if value == "" {
			return
		}
		rows = append(rows, struct {
			label string
			value string
		}{label, value})
	}

	add("Engine", info.Engine)
	if info.RequestedEngine != "" && info.RequestedEngine != info.Engine {
		add("Requested", info.RequestedEngine)
	}
	add("Fallback", info.FallbackReason)
	add("Device", info.DeviceName)
	if info.BatchSize > 0 {
		add("Batch Size", fmt.Sprintf("%d", info.BatchSize))
	}
	add("Validation", info.MetalValidation)
	if info.ThreadCount > 0 {
		add("Threads", fmt.Sprintf("%d", info.ThreadCount))
	}
	add("Network", info.Network)

	return rows
}

// renderStatsTable renders statistics in a table-like format using Bubbletea
func (m ProgressModel) renderStatsTable() string {
	pad := strings.Repeat(" ", padding)
	var content strings.Builder

	// Create table-like display for statistics
	stats := []struct {
		label string
		value string
	}{
		{"Attempts", formatLargeNumber(m.stats.CurrentAttempts)},
		{"Speed", fmt.Sprintf("%.0f addr/s", m.stats.Speed)},
		{"ETA", m.formatETA()},
		{"50% at", m.format50Probability()},
	}

	// Render each stat row
	for _, stat := range stats {
		content.WriteString(pad)
		content.WriteString(m.styleManager.FormatKeyValue(stat.label, stat.value))
		content.WriteString("\n")
	}

	return content.String()
}

// renderThreadStats renders thread performance statistics
func (m ProgressModel) renderThreadStats(metrics ThreadMetrics) string {
	pad := strings.Repeat(" ", padding)
	var content strings.Builder

	threadStats := []struct {
		label string
		value string
	}{
		{"Threads", fmt.Sprintf("%d threads", metrics.ThreadCount)},
		{"Efficiency", fmt.Sprintf("%.1f%%", metrics.EfficiencyRatio*100)},
		{"Peak Speed", fmt.Sprintf("%.0f addr/s", m.statsManager.GetPeakSpeed())},
	}

	for _, stat := range threadStats {
		content.WriteString(pad)
		content.WriteString(m.styleManager.FormatKeyValue(stat.label, stat.value))
		content.WriteString("\n")
	}

	return content.String()
}

// formatETA formats the estimated time remaining
func (m ProgressModel) formatETA() string {
	if m.isComplete {
		totalTime := time.Since(m.stats.StartTime)
		return fmt.Sprintf("Done in %s", formatDuration(totalTime))
	}
	if m.stats.EstimatedTime > 0 {
		return formatDuration(m.stats.EstimatedTime)
	}
	return "Calculating..."
}

// format50Probability formats the 50% probability attempts
func (m ProgressModel) format50Probability() string {
	if m.stats.Probability50 > 0 {
		return formatLargeNumber(m.stats.Probability50) + " attempts"
	}
	return "Nearly impossible"
}

// UpdateProgress sends a progress update message to the model
func UpdateProgress(attempts int64, speed float64, probability float64, estimatedTime time.Duration, difficulty float64, pattern string) tea.Cmd {
	return func() tea.Msg {
		return ProgressMsg{
			Attempts:      attempts,
			Speed:         speed,
			Probability:   probability,
			EstimatedTime: estimatedTime,
			Difficulty:    difficulty,
			Pattern:       pattern,
		}
	}
}

// SendQuit sends a quit message to the model
func SendQuit() tea.Cmd {
	return func() tea.Msg {
		return QuitMsg{}
	}
}

// SendWalletResult sends a wallet result message to the model
func SendWalletResult(index int, address, privateKey string, attempts int, duration time.Duration, err string) tea.Cmd {
	return func() tea.Msg {
		return WalletResultMsg{
			Result: WalletResult{
				Index:      index,
				Address:    address,
				PrivateKey: privateKey,
				Attempts:   attempts,
				Time:       duration,
				Error:      err,
			},
		}
	}
}

// updateResultsTable updates the Bubbletea table with current wallet results
func (m *ProgressModel) updateResultsTable() {
	rows := make([]table.Row, 0, len(m.walletResults))

	for _, result := range m.walletResults {
		if result.Error != "" {
			// Error row
			rows = append(rows, table.Row{
				fmt.Sprintf("%d", result.Index),
				"Error occurred",
				result.Error,
				"-",
				m.formatDuration(result.Time),
			})
		} else {
			// Success row - format address and private key for better display
			address := result.Address
			// Don't truncate Ethereum addresses (42 chars) - they need to show the full suffix
			// if len(address) > 40 {
			//     address = address[:40] // Truncate if too long for display
			// }

			privateKey := result.PrivateKey
			// Dont truncate Ethereum private keys
			// if len(privateKey) > 60 {
			//	privateKey = privateKey[:60] + "..." // Truncate long private keys
			//}

			rows = append(rows, table.Row{
				fmt.Sprintf("%d", result.Index),
				address,
				privateKey,
				formatLargeNumber(int64(result.Attempts)),
				m.formatDuration(result.Time),
			})
		}
	}

	m.resultsTable.SetRows(rows)
}

// formatDuration formats duration for table display
func (m ProgressModel) formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}
