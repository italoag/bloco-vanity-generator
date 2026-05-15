package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"bloco-vgen/pkg/wallet"
)

// StatsModel represents the statistics TUI component
type StatsModel struct {
	table        table.Model
	stats        *wallet.GenerationStats
	styleManager *StyleManager
	width        int
	height       int
	quitting     bool
	ready        bool
}

// StatsData represents formatted statistical data for display
type StatsData struct {
	Pattern         string
	Difficulty      string
	Probability50   string
	IsChecksum      bool
	PatternLength   int
	BaseDifficulty  string
	TotalDifficulty string
}

// NewStatsModel creates a new statistics model
func NewStatsModel(stats *wallet.GenerationStats) StatsModel {
	// Create style manager with terminal capabilities
	tuiManager := NewTUIManager()
	capabilities := tuiManager.DetectCapabilities()
	styleManager := NewStyleManagerWithCapabilities(capabilities)

	// Create the table model
	columns := []table.Column{
		{Title: "Metric", Width: 20},
		{Title: "Value", Width: 30},
		{Title: "Description", Width: 40},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(false),
		table.WithHeight(15),
	)

	// Apply table styling
	tableStyle := table.DefaultStyles()
	if capabilities.SupportsColor {
		tableStyle.Header = tableStyle.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(SecondaryColor)).
			BorderBottom(true).
			Bold(true).
			Foreground(lipgloss.Color(TextPrimary))

		tableStyle.Selected = tableStyle.Selected.
			Foreground(lipgloss.Color(TextPrimary)).
			Background(lipgloss.Color(PrimaryColor)).
			Bold(false)
	} else {
		// Monochrome styling
		tableStyle.Header = tableStyle.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			Bold(true).
			Underline(true)

		tableStyle.Selected = tableStyle.Selected.
			Bold(true).
			Underline(true)
	}

	t.SetStyles(tableStyle)

	model := StatsModel{
		table:        t,
		stats:        stats,
		styleManager: styleManager,
		width:        capabilities.TerminalWidth,
		height:       capabilities.TerminalHeight,
		quitting:     false,
		ready:        false,
	}
	model.updateTableData()
	return model
}

// Init initializes the statistics model
func (m StatsModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m StatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.adaptTableToSize()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		case "down", "j":
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		case "enter", " ":
			// Allow selection but don't do anything special
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}

	case StatsUpdateMsg:
		// Handle external statistics updates
		if msg.Stats != nil {
			m.stats = msg.Stats
			m.updateTableData()
			return m, nil
		}

	default:
		if !m.ready {
			m.ready = true
			m.updateTableData()
			return m, nil
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the statistics display
func (m StatsModel) View() string {
	if m.stats == nil {
		return m.styleManager.FormatError("No statistics available")
	}

	pad := strings.Repeat(" ", 2)
	var content strings.Builder

	// Title
	content.WriteString("\n")
	content.WriteString(renderBlocoLogo(pad))
	content.WriteString("\n")
	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatTitle("Bloco Address Difficulty Analysis"))
	content.WriteString("\n\n")

	// Pattern overview section
	content.WriteString(m.renderPatternOverview())
	content.WriteString("\n")

	// Main statistics table
	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatSubtitle("Detailed Statistics"))
	content.WriteString("\n")
	content.WriteString(m.table.View())
	content.WriteString("\n")

	// Time estimates section
	content.WriteString(m.renderTimeEstimates())
	content.WriteString("\n")

	// Probability examples section
	content.WriteString(m.renderProbabilityExamples())
	content.WriteString("\n")

	// Recommendations section
	content.WriteString(m.renderRecommendations())
	content.WriteString("\n")

	// Help text
	content.WriteString(pad)
	helpText := "Use ↑/↓ or j/k to navigate • Press 'q', 'Ctrl+C', or 'Esc' to quit"
	content.WriteString(helpStyle(helpText))

	return content.String()
}

// renderPatternOverview renders the pattern overview section
func (m StatsModel) renderPatternOverview() string {
	pad := strings.Repeat(" ", 2)
	var content strings.Builder

	// Create pattern visualization
	pattern := m.getPatternVisualization()
	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatKeyValue("Pattern", pattern))
	content.WriteString("\n")

	// Checksum status
	checksumStatus := "Disabled"
	if m.stats.IsChecksum {
		checksumStatus = m.styleManager.FormatHighlight("Enabled (increases difficulty)")
	}
	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatKeyValue("Checksum", checksumStatus))
	content.WriteString("\n")

	// Pattern length
	patternLength := len(m.stats.Pattern)
	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatKeyValue("Pattern Length", fmt.Sprintf("%d characters", patternLength)))

	return content.String()
}

// renderTimeEstimates renders the time estimates section
func (m StatsModel) renderTimeEstimates() string {
	pad := strings.Repeat(" ", 2)
	var content strings.Builder

	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatSubtitle("Time Estimates (at different speeds)"))
	content.WriteString("\n")

	// Create time estimates table
	timeColumns := []table.Column{
		{Title: "Speed (addr/s)", Width: 15},
		{Title: "50% Probability", Width: 20},
		{Title: "90% Probability", Width: 20},
	}

	timeTable := table.New(
		table.WithColumns(timeColumns),
		table.WithFocused(false),
		table.WithHeight(6),
	)

	// Apply consistent styling
	timeTableStyles := table.DefaultStyles()
	if m.styleManager.IsColorSupported() {
		timeTableStyles.Header = timeTableStyles.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(SecondaryColor)).
			BorderBottom(true).
			Bold(true).
			Foreground(lipgloss.Color(TextPrimary))
	} else {
		timeTableStyles.Header = timeTableStyles.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			Bold(true).
			Underline(true)
	}
	timeTable.SetStyles(timeTableStyles)

	// Generate time estimate rows
	speeds := []float64{1000, 10000, 50000, 100000}
	var timeRows []table.Row

	for _, speed := range speeds {
		speedStr := formatLargeNumber(int64(speed))

		var time50Str, time90Str string
		if m.stats.Probability50 > 0 {
			time50 := time.Duration(float64(m.stats.Probability50)/speed) * time.Second
			time50Str = formatDuration(time50)

			// 90% probability requires approximately 2.3 times more attempts than 50%
			time90 := time.Duration(float64(m.stats.Probability50)*2.3/speed) * time.Second
			time90Str = formatDuration(time90)
		} else {
			time50Str = "Nearly impossible"
			time90Str = "Nearly impossible"
		}

		timeRows = append(timeRows, table.Row{speedStr, time50Str, time90Str})
	}

	timeTable.SetRows(timeRows)
	content.WriteString(timeTable.View())

	return content.String()
}

// renderProbabilityExamples renders the probability examples section
func (m StatsModel) renderProbabilityExamples() string {
	pad := strings.Repeat(" ", 2)
	var content strings.Builder

	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatSubtitle("Probability Examples"))
	content.WriteString("\n")

	// Create probability table
	probColumns := []table.Column{
		{Title: "Attempts", Width: 15},
		{Title: "Probability", Width: 15},
		{Title: "Likelihood", Width: 25},
	}

	probTable := table.New(
		table.WithColumns(probColumns),
		table.WithFocused(false),
		table.WithHeight(6),
	)

	// Apply consistent styling
	probTableStyles := table.DefaultStyles()
	if m.styleManager.IsColorSupported() {
		probTableStyles.Header = probTableStyles.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(SecondaryColor)).
			BorderBottom(true).
			Bold(true).
			Foreground(lipgloss.Color(TextPrimary))
	} else {
		probTableStyles.Header = probTableStyles.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			Bold(true).
			Underline(true)
	}
	probTable.SetStyles(probTableStyles)

	// Generate probability rows
	attemptExamples := []int64{1000, 10000, 100000, 1000000}
	var probRows []table.Row

	for _, attempts := range attemptExamples {
		attemptsStr := formatLargeNumber(attempts)

		var probStr, likelihoodStr string
		if m.stats.Difficulty > 0 {
			prob := computeProbability(m.stats.Difficulty, attempts) * 100
			probStr = fmt.Sprintf("%.4f%%", prob)

			// Provide intuitive likelihood descriptions
			if prob < 0.001 {
				likelihoodStr = "Extremely unlikely"
			} else if prob < 0.1 {
				likelihoodStr = "Very unlikely"
			} else if prob < 1 {
				likelihoodStr = "Unlikely"
			} else if prob < 10 {
				likelihoodStr = "Low chance"
			} else if prob < 50 {
				likelihoodStr = "Moderate chance"
			} else if prob < 90 {
				likelihoodStr = "Good chance"
			} else {
				likelihoodStr = "Very likely"
			}
		} else {
			probStr = "0%"
			likelihoodStr = "Impossible"
		}

		probRows = append(probRows, table.Row{attemptsStr, probStr, likelihoodStr})
	}

	probTable.SetRows(probRows)
	content.WriteString(probTable.View())

	return content.String()
}

// renderRecommendations renders the recommendations section
func (m StatsModel) renderRecommendations() string {
	pad := strings.Repeat(" ", 2)
	var content strings.Builder

	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatSubtitle("Recommendations"))
	content.WriteString("\n")

	patternLength := len(m.stats.Pattern)

	// Difficulty assessment
	var difficultyLevel, recommendation string
	if patternLength <= 3 {
		difficultyLevel = m.styleManager.FormatSuccess("Easy")
		recommendation = "Should generate quickly, suitable for testing"
	} else if patternLength <= 5 {
		difficultyLevel = m.styleManager.FormatWarning("Moderate")
		recommendation = "May take some time, reasonable for production use"
	} else if patternLength <= 7 {
		difficultyLevel = m.styleManager.FormatError("Hard")
		recommendation = "Will take considerable time, plan accordingly"
	} else {
		difficultyLevel = m.styleManager.FormatError("Extremely Hard")
		recommendation = "May take days/weeks/years, use with extreme caution"
	}

	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatKeyValue("Difficulty Level", difficultyLevel))
	content.WriteString("\n")
	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatKeyValue("Recommendation", recommendation))
	content.WriteString("\n")

	// Checksum impact
	if m.stats.IsChecksum {
		checksumImpact := "Checksum validation significantly increases difficulty"
		content.WriteString(pad)
		content.WriteString(m.styleManager.FormatKeyValue("Checksum Impact", m.styleManager.FormatWarning(checksumImpact)))
		content.WriteString("\n")
	}

	// Thread recommendation
	threadRec := "Use multiple threads (--threads) for better performance"
	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatKeyValue("Performance Tip", m.styleManager.FormatInfo(threadRec)))

	return content.String()
}

// updateTableData updates the table with current statistics
func (m *StatsModel) updateTableData() {
	if m.stats == nil {
		return
	}

	// Generate table rows with statistical data
	rows := m.generateStatisticsRows()
	m.table.SetRows(rows)
}

// generateStatisticsRows generates the main statistics table rows
func (m StatsModel) generateStatisticsRows() []table.Row {
	var rows []table.Row

	// Pattern information
	pattern := m.getPatternVisualization()
	rows = append(rows, table.Row{
		"Pattern",
		pattern,
		"The address pattern to match",
	})

	// Pattern length
	patternLength := len(m.stats.Pattern)
	rows = append(rows, table.Row{
		"Pattern Length",
		fmt.Sprintf("%d characters", patternLength),
		"Number of hex characters to match",
	})

	// Base difficulty (without checksum)
	baseDifficulty := math.Pow(16, float64(patternLength))
	rows = append(rows, table.Row{
		"Base Difficulty",
		formatLargeNumber(int64(baseDifficulty)),
		"Difficulty without checksum validation",
	})

	// Total difficulty (with checksum if enabled)
	rows = append(rows, table.Row{
		"Total Difficulty",
		formatLargeNumber(int64(m.stats.Difficulty)),
		"Final difficulty including checksum",
	})

	// Checksum multiplier
	if m.stats.IsChecksum {
		multiplier := m.stats.Difficulty / baseDifficulty
		rows = append(rows, table.Row{
			"Checksum Multiplier",
			fmt.Sprintf("%.1fx", multiplier),
			"Difficulty increase from checksum",
		})
	}

	// 50% probability attempts
	if m.stats.Probability50 > 0 {
		rows = append(rows, table.Row{
			"50% Probability",
			formatLargeNumber(m.stats.Probability50),
			"Attempts needed for 50% success chance",
		})
	} else {
		rows = append(rows, table.Row{
			"50% Probability",
			"Nearly impossible",
			"Pattern is extremely difficult",
		})
	}

	// Expected attempts (mathematical expectation)
	expectedAttempts := int64(m.stats.Difficulty)
	rows = append(rows, table.Row{
		"Expected Attempts",
		formatLargeNumber(expectedAttempts),
		"Mathematical expectation (average)",
	})

	// Probability per attempt
	probPerAttempt := 1.0 / m.stats.Difficulty * 100
	var probPerAttemptStr string
	if probPerAttempt < 0.000001 {
		probPerAttemptStr = fmt.Sprintf("%.2e%%", probPerAttempt)
	} else {
		probPerAttemptStr = fmt.Sprintf("%.6f%%", probPerAttempt)
	}
	rows = append(rows, table.Row{
		"Success Rate",
		probPerAttemptStr,
		"Probability of success per attempt",
	})

	return rows
}

// getPatternVisualization creates a visual representation of the pattern
func (m StatsModel) getPatternVisualization() string {
	if m.stats.Pattern == "" {
		return "any"
	}

	// For display purposes, we need to extract prefix and suffix from the pattern
	// Since we don't have direct access to prefix/suffix, we'll show the full pattern
	pattern := m.stats.Pattern

	// Create visualization with wildcards for the middle part
	if len(pattern) < 40 {
		wildcards := strings.Repeat("*", 40-len(pattern))
		if len(pattern) > 0 {
			// Assume it's a prefix pattern for visualization
			return pattern + wildcards
		}
	}

	return pattern
}

// adaptTableToSize adapts the table layout to the terminal size
func (m *StatsModel) adaptTableToSize() {
	if m.width < 80 {
		// Narrow terminal - adjust column widths
		columns := []table.Column{
			{Title: "Metric", Width: 15},
			{Title: "Value", Width: 20},
			{Title: "Description", Width: 25},
		}
		m.table.SetColumns(columns)
	} else if m.width < 120 {
		// Medium terminal - standard widths
		columns := []table.Column{
			{Title: "Metric", Width: 20},
			{Title: "Value", Width: 25},
			{Title: "Description", Width: 35},
		}
		m.table.SetColumns(columns)
	} else {
		// Wide terminal - expanded widths
		columns := []table.Column{
			{Title: "Metric", Width: 25},
			{Title: "Value", Width: 30},
			{Title: "Description", Width: 50},
		}
		m.table.SetColumns(columns)
	}

	// Adjust table height based on terminal height
	if m.height < 20 {
		m.table.SetHeight(8)
	} else if m.height < 30 {
		m.table.SetHeight(12)
	} else {
		m.table.SetHeight(15)
	}
}

// StatsUpdateMsg represents a statistics update message
type StatsUpdateMsg struct {
	Stats *wallet.GenerationStats
}

// UpdateStats sends a statistics update message to the model
func UpdateStats(stats *wallet.GenerationStats) tea.Cmd {
	return func() tea.Msg {
		return StatsUpdateMsg{Stats: stats}
	}
}

// Helper functions for statistical calculations

// computeProbability calculates the probability of finding an address after N attempts
func computeProbability(difficulty float64, attempts int64) float64 {
	if difficulty <= 0 {
		return 0
	}
	return 1 - math.Pow(1-1/difficulty, float64(attempts))
}
