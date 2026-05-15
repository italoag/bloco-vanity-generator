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

// BenchmarkState represents the current state of the benchmark TUI
type BenchmarkState int

const (
	BenchmarkStateProgress BenchmarkState = iota
	BenchmarkStateResults
	BenchmarkStateTransitioning
)

// BenchmarkModel represents the benchmark TUI component with dual-mode display
type BenchmarkModel struct {
	state          BenchmarkState
	table          table.Model
	progress       progress.Model
	results        *wallet.BenchmarkResult
	progressMsg    ProgressMsg
	styleManager   *StyleManager
	width          int
	height         int
	running        bool
	quitting       bool
	lastUpdate     time.Time
	transitionTime time.Time
	engineInfo     EngineInfo
}

// WithEngineInfo returns a copy of the model with engine diagnostics attached.
// The same fields are printed in the text benchmark path; the TUI renders them
// in both the running and results views.
func (m BenchmarkModel) WithEngineInfo(info EngineInfo) BenchmarkModel {
	m.engineInfo = info
	return m
}

func (m BenchmarkModel) Quitting() bool {
	return m.quitting
}

// BenchmarkUpdateMsg represents a benchmark update message
type BenchmarkUpdateMsg struct {
	Results  *wallet.BenchmarkResult
	Running  bool
	Progress ProgressMsg
}

// BenchmarkCompleteMsg represents benchmark completion
type BenchmarkCompleteMsg struct {
	Results *wallet.BenchmarkResult
}

// NewBenchmarkModel creates a new benchmark model
func NewBenchmarkModel() BenchmarkModel {
	// Create progress bar with gradient styling
	p := progress.New(progress.WithDefaultGradient())

	// Create table for results display
	columns := []table.Column{
		{Title: "Metric", Width: 25},
		{Title: "Value", Width: 20},
		{Title: "Details", Width: 30},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Set table styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(PrimaryColor)).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(TextPrimary)).
		Background(lipgloss.Color(PrimaryColor)).
		Bold(false)
	t.SetStyles(s)

	return BenchmarkModel{
		state:        BenchmarkStateProgress,
		table:        t,
		progress:     p,
		styleManager: NewStyleManager(),
		running:      true,
		lastUpdate:   time.Now(),
	}
}

// Init initializes the benchmark model
func (m BenchmarkModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tickCmd(),
	)
}

// Update handles messages and updates the benchmark model
func (m BenchmarkModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Batch(tea.ExitAltScreen, tea.Quit)
		case "enter", "space":
			if m.state == BenchmarkStateResults {
				// Reserved for future interaction with results table
				// Currently no action required in results state
				return m, nil // Explicitly return no-op command
			}
		}

	case BenchmarkUpdateMsg:
		m.progressMsg = msg.Progress
		m.running = msg.Running
		m.lastUpdate = time.Now()

		if msg.Results != nil {
			m.results = msg.Results
			if !msg.Running && m.state == BenchmarkStateProgress {
				m.state = BenchmarkStateTransitioning
				m.transitionTime = time.Now()
			}
		}

	case BenchmarkCompleteMsg:
		m.results = msg.Results
		m.running = false
		m.state = BenchmarkStateTransitioning
		m.transitionTime = time.Now()

	case TickMsg:
		// Handle smooth transitions
		if m.state == BenchmarkStateTransitioning {
			if time.Since(m.transitionTime) > 500*time.Millisecond {
				m.state = BenchmarkStateResults
				m.table.SetRows(m.generateResultsRows())
			}
		}

		// Continue ticking for animations
		cmds = append(cmds, tickCmd())

	case progress.FrameMsg:
		progressModel, progressCmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		cmds = append(cmds, progressCmd)
	}

	// Update table if in results state
	if m.state == BenchmarkStateResults {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the benchmark model
func (m BenchmarkModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.state {
	case BenchmarkStateProgress:
		return m.renderProgressView()
	case BenchmarkStateTransitioning:
		return m.renderTransitionView()
	case BenchmarkStateResults:
		return m.renderResultsView()
	default:
		return m.renderProgressView()
	}
}

// renderProgressView renders the progress display
func (m BenchmarkModel) renderProgressView() string {
	var b strings.Builder

	pad := strings.Repeat(" ", padding)
	b.WriteString(renderBlocoLogo(pad))
	b.WriteString("\n")

	// Header
	header := m.styleManager.FormatHeader("Benchmark Running")
	b.WriteString(header + "\n\n")

	// Engine diagnostics (matches the text-mode benchmark header)
	if block := m.renderEngineInfoBlock(pad); block != "" {
		b.WriteString(block)
		b.WriteString("\n")
	}

	// Progress bar
	if m.progressMsg.Attempts > 0 {
		// Calculate progress percentage based on attempts vs estimated total
		// Use a more sophisticated progress calculation
		var progressPercent float64
		if m.progressMsg.EstimatedTime > 0 && m.progressMsg.Speed > 0 {
			// Calculate based on time elapsed vs estimated time
			totalEstimatedAttempts := float64(m.progressMsg.EstimatedTime.Seconds()) * m.progressMsg.Speed
			if totalEstimatedAttempts > 0 {
				progressPercent = float64(m.progressMsg.Attempts) / totalEstimatedAttempts
			}
		} else {
			// Fallback: use a rough estimate based on attempts
			progressPercent = float64(m.progressMsg.Attempts) / 50000.0 // Assume max 50k attempts
		}

		if progressPercent > 1.0 {
			progressPercent = 1.0
		}
		if progressPercent < 0 {
			progressPercent = 0
		}

		progressBar := m.progress.ViewAs(progressPercent)
		b.WriteString(progressBar + "\n\n")
	}

	// Real-time metrics
	metricsStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(PrimaryColor)).
		Padding(1, 2)

	var metrics string
	if m.results != nil {
		metrics = fmt.Sprintf(
			"Current Performance:\n"+
				"   Attempts: %s\n"+
				"   Current Speed: %s addr/s\n"+
				"   Average Speed: %s addr/s\n"+
				"   Min/Max Speed: %s/%s addr/s\n"+
				"   Pattern: %s\n"+
				"   Difficulty: %.2f\n"+
				"   Estimated Time: %s\n"+
				"   Efficiency: %.1f%%",
			formatLargeNumber(m.progressMsg.Attempts),
			formatSpeed(m.progressMsg.Speed),
			formatSpeed(m.results.AverageSpeed),
			formatSpeed(m.results.MinSpeed),
			formatSpeed(m.results.MaxSpeed),
			m.progressMsg.Pattern,
			m.progressMsg.Difficulty,
			formatDuration(m.progressMsg.EstimatedTime),
			m.results.ScalabilityEfficiency*100,
		)
	} else {
		metrics = fmt.Sprintf(
			"Current Performance:\n"+
				"   Attempts: %s\n"+
				"   Speed: %s addr/s\n"+
				"   Pattern: %s\n"+
				"   Difficulty: %.2f\n"+
				"   Estimated Time: %s",
			formatLargeNumber(m.progressMsg.Attempts),
			formatSpeed(m.progressMsg.Speed),
			m.progressMsg.Pattern,
			m.progressMsg.Difficulty,
			formatDuration(m.progressMsg.EstimatedTime),
		)
	}

	b.WriteString(metricsStyle.Render(metrics) + "\n\n")

	// Help text
	helpText := helpStyle("Press q to quit • Ctrl+C to exit")
	b.WriteString(helpText)

	return b.String()
}

// renderTransitionView renders the transition between progress and results
func (m BenchmarkModel) renderTransitionView() string {
	var b strings.Builder

	header := m.styleManager.FormatHeader("Benchmark Complete - Loading Results...")
	b.WriteString(header + "\n\n")

	// Simple loading animation
	dots := strings.Repeat(".", int(time.Since(m.transitionTime)/100*time.Millisecond)%4)
	loading := fmt.Sprintf("Preparing results%s", dots)
	b.WriteString(loading + "\n\n")

	return b.String()
}

// renderResultsView renders the benchmark results table
func (m BenchmarkModel) renderResultsView() string {
	var b strings.Builder

	pad := strings.Repeat(" ", padding)
	b.WriteString(renderBlocoLogo(pad))
	b.WriteString("\n")

	// Header
	header := m.styleManager.FormatHeader("Benchmark Results")
	b.WriteString(header + "\n\n")

	// Engine diagnostics (matches the text-mode benchmark header)
	if block := m.renderEngineInfoBlock(pad); block != "" {
		b.WriteString(block)
		b.WriteString("\n")
	}

	// Results summary
	if m.results != nil {
		summaryStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(SuccessColor)).
			Padding(1, 2)

		summary := fmt.Sprintf(
			"Summary:\n"+
				"   Total Duration: %s\n"+
				"   Average Speed: %.0f addr/s\n"+
				"   Thread Efficiency: %.1f%%\n"+
				"   Scalability Factor: %.2fx",
			formatDuration(m.results.TotalDuration),
			m.results.AverageSpeed,
			m.results.ScalabilityEfficiency*100,
			float64(m.results.ThreadCount)*m.results.ScalabilityEfficiency,
		)

		b.WriteString(summaryStyle.Render(summary) + "\n\n")
	}

	// Results table
	tableStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(PrimaryColor))

	b.WriteString(tableStyle.Render(m.table.View()) + "\n\n")

	// Help text
	helpText := helpStyle("↑/↓: Navigate • q: Quit • Ctrl+C: Exit")
	b.WriteString(helpText)

	return b.String()
}

// renderEngineInfoBlock renders the engine diagnostics block for the benchmark
// views. It mirrors the fields emitted in runBenchmarkText so users see the
// same engine, device and batch context regardless of TUI vs text mode.
func (m BenchmarkModel) renderEngineInfoBlock(pad string) string {
	if m.engineInfo.IsZero() {
		return ""
	}

	var b strings.Builder
	b.WriteString(pad)
	b.WriteString(m.styleManager.FormatSubtitle("Engine"))
	b.WriteString("\n")

	rows := engineInfoRows(m.engineInfo)
	for _, row := range rows {
		b.WriteString(pad)
		b.WriteString(m.styleManager.FormatKeyValue(row.label, row.value))
		b.WriteString("\n")
	}
	return b.String()
}

// generateResultsRows creates table rows from benchmark results
func (m BenchmarkModel) generateResultsRows() []table.Row {
	if m.results == nil {
		return []table.Row{}
	}

	rows := []table.Row{
		{"Total Attempts", formatLargeNumber(m.results.TotalAttempts), "Total addresses generated"},
		{"Duration", formatDuration(m.results.TotalDuration), "Total benchmark time"},
		{"Average Speed", fmt.Sprintf("%.0f addr/s", m.results.AverageSpeed), "Mean generation rate"},
		{"Min Speed", fmt.Sprintf("%.0f addr/s", m.results.MinSpeed), "Lowest recorded speed"},
		{"Max Speed", fmt.Sprintf("%.0f addr/s", m.results.MaxSpeed), "Highest recorded speed"},
		{"Thread Count", fmt.Sprintf("%d", m.results.ThreadCount), "Parallel workers used"},
		{"Single Thread Est.", fmt.Sprintf("%.0f addr/s", m.results.SingleThreadSpeed), "Estimated single-thread speed"},
		{"Scalability", fmt.Sprintf("%.1f%%", m.results.ScalabilityEfficiency*100), "Multi-threading efficiency"},
		{"Speedup Factor", fmt.Sprintf("%.2fx", float64(m.results.ThreadCount)*m.results.ScalabilityEfficiency), "Performance improvement"},
	}

	return rows
}

// tickCmd returns a command that ticks every 100ms for smooth animations
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// formatSpeed formats speed values for display
func formatSpeed(speed float64) string {
	if speed >= 1000000 {
		return fmt.Sprintf("%.1fM", speed/1000000)
	} else if speed >= 1000 {
		return fmt.Sprintf("%.1fK", speed/1000)
	}
	return fmt.Sprintf("%.0f", speed)
}
