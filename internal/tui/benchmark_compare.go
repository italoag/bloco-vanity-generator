package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BenchmarkComparisonSummary struct {
	Platform             string
	PowerMode            string
	MetalAvailable       bool
	MetalDeviceName      string
	BestApproach         string
	DecisionReason       string
	SpeedupThreshold     float64
	StabilityCVThreshold float64
}

type BenchmarkComparisonCase struct {
	Pattern        string
	Checksum       bool
	BatchSize      int
	CPU            string
	Auto           string
	AutoEngine     string
	Metal          string
	AutoSpeedup    string
	MetalVsCPU     string
	MetalVsAuto    string
	MetalStability string
	Decision       string
	Error          string
}

type BenchmarkComparisonUpdateMsg struct {
	Summary   BenchmarkComparisonSummary
	Cases     []BenchmarkComparisonCase
	Completed int
	Total     int
	Running   bool
}

type BenchmarkComparisonCompleteMsg struct {
	Summary BenchmarkComparisonSummary
	Cases   []BenchmarkComparisonCase
	Total   int
}

type BenchmarkComparisonModel struct {
	table        table.Model
	progress     progress.Model
	styleManager *StyleManager
	summary      BenchmarkComparisonSummary
	cases        []BenchmarkComparisonCase
	completed    int
	total        int
	running      bool
	quitting     bool
	width        int
	height       int
}

func NewBenchmarkComparisonModel(summary BenchmarkComparisonSummary, total int) BenchmarkComparisonModel {
	columns := []table.Column{
		{Title: "Case", Width: 28},
		{Title: "CPU", Width: 13},
		{Title: "Auto", Width: 18},
		{Title: "Metal", Width: 13},
		{Title: "Speedups", Width: 24},
		{Title: "Best", Width: 10},
		{Title: "Details", Width: 34},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(12),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(PrimaryColor)).BorderBottom(true).Bold(false)
	s.Selected = s.Selected.Foreground(lipgloss.Color(TextPrimary)).Background(lipgloss.Color(PrimaryColor)).Bold(false)
	t.SetStyles(s)

	return BenchmarkComparisonModel{
		table:        t,
		progress:     progress.New(progress.WithDefaultGradient()),
		styleManager: NewStyleManager(),
		summary:      summary,
		total:        total,
		running:      true,
	}
}

func (m BenchmarkComparisonModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m BenchmarkComparisonModel) Quitting() bool {
	return m.quitting
}

func (m BenchmarkComparisonModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
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
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Batch(tea.ExitAltScreen, tea.Quit)
		case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}
	case BenchmarkComparisonUpdateMsg:
		m.summary = msg.Summary
		m.cases = msg.Cases
		m.completed = msg.Completed
		m.total = msg.Total
		m.running = msg.Running
		m.table.SetRows(m.rows())
	case BenchmarkComparisonCompleteMsg:
		m.summary = msg.Summary
		m.cases = msg.Cases
		m.completed = len(msg.Cases)
		m.total = msg.Total
		m.running = false
		m.table.SetRows(m.rows())
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m BenchmarkComparisonModel) View() string {
	if m.quitting {
		return ""
	}

	pad := strings.Repeat(" ", padding)
	var b strings.Builder
	b.WriteString(renderBlocoLogo(pad))
	b.WriteString("\n")

	title := "Benchmark Comparison Running"
	if !m.running && len(m.cases) > 0 {
		title = "Benchmark Comparison Complete"
	}
	b.WriteString(m.styleManager.FormatHeader(title) + "\n\n")
	b.WriteString(m.renderSummary(pad))
	b.WriteString("\n")

	if m.total > 0 {
		percent := float64(m.completed) / float64(m.total)
		if percent > 1 {
			percent = 1
		}
		b.WriteString(m.progress.ViewAs(percent) + "\n\n")
	}

	b.WriteString(pad)
	b.WriteString(m.styleManager.FormatSubtitle("CPU vs Auto vs Metal"))
	b.WriteString("\n")
	tableStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(PrimaryColor))
	b.WriteString(tableStyle.Render(m.table.View()) + "\n\n")
	b.WriteString(helpStyle("↑/↓: Navigate cases • q: Quit • Ctrl+C: Exit"))
	return b.String()
}

func (m BenchmarkComparisonModel) renderSummary(pad string) string {
	var b strings.Builder
	best := m.summary.BestApproach
	if best == "" {
		best = "evaluating"
	}
	b.WriteString(pad)
	b.WriteString(m.styleManager.FormatKeyValue("Best Approach", best))
	b.WriteString("\n")
	if m.summary.DecisionReason != "" {
		b.WriteString(pad)
		b.WriteString(m.styleManager.FormatKeyValue("Reason", m.summary.DecisionReason))
		b.WriteString("\n")
	}
	if m.summary.Platform != "" {
		b.WriteString(pad)
		b.WriteString(m.styleManager.FormatKeyValue("Platform", m.summary.Platform))
		b.WriteString("\n")
	}
	if m.summary.PowerMode != "" {
		b.WriteString(pad)
		b.WriteString(m.styleManager.FormatKeyValue("Power Mode", m.summary.PowerMode))
		b.WriteString("\n")
	}
	b.WriteString(pad)
	b.WriteString(m.styleManager.FormatKeyValue("Metal Available", fmt.Sprintf("%t", m.summary.MetalAvailable)))
	b.WriteString("\n")
	if m.summary.MetalDeviceName != "" {
		b.WriteString(pad)
		b.WriteString(m.styleManager.FormatKeyValue("Metal Device", m.summary.MetalDeviceName))
		b.WriteString("\n")
	}
	if m.total > 0 {
		b.WriteString(pad)
		b.WriteString(m.styleManager.FormatKeyValue("Cases", fmt.Sprintf("%d/%d", m.completed, m.total)))
		b.WriteString("\n")
	}
	if m.summary.SpeedupThreshold > 0 {
		b.WriteString(pad)
		b.WriteString(m.styleManager.FormatKeyValue("Speedup Threshold", fmt.Sprintf("%.2fx", m.summary.SpeedupThreshold)))
		b.WriteString("\n")
	}
	if m.summary.StabilityCVThreshold > 0 {
		b.WriteString(pad)
		b.WriteString(m.styleManager.FormatKeyValue("Stability CV Threshold", fmt.Sprintf("%.0f%%", m.summary.StabilityCVThreshold*100)))
		b.WriteString("\n")
	}
	return b.String()
}

func (m BenchmarkComparisonModel) rows() []table.Row {
	rows := make([]table.Row, 0, len(m.cases))
	for _, resultCase := range m.cases {
		auto := resultCase.Auto
		if resultCase.AutoEngine != "" && auto != "" {
			auto = fmt.Sprintf("%s (%s)", auto, resultCase.AutoEngine)
		}
		speedups := strings.Join(benchmarkComparisonNonEmpty(resultCase.AutoSpeedup, resultCase.MetalVsCPU, resultCase.MetalVsAuto), " | ")
		details := strings.Join(benchmarkComparisonNonEmpty(resultCase.MetalStability, resultCase.Error), " | ")
		rows = append(rows, table.Row{
			fmt.Sprintf("%s checksum=%t batch=%d", resultCase.Pattern, resultCase.Checksum, resultCase.BatchSize),
			resultCase.CPU,
			auto,
			resultCase.Metal,
			speedups,
			resultCase.Decision,
			details,
		})
	}
	return rows
}

func benchmarkComparisonNonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}
