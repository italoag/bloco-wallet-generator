package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	padding  = 2
	maxWidth = 80
)

// ProgressModel represents the progress TUI component
type ProgressModel struct {
	progress     progress.Model
	stats        *Statistics
	statsManager StatsManager
	styleManager *StyleManager
	width        int
	height       int
	quitting     bool
	lastUpdate   time.Time
}

// ProgressMsg represents a progress update message
type ProgressMsg struct {
	Attempts      int64
	Speed         float64
	Probability   float64
	EstimatedTime time.Duration
	Difficulty    float64
	Pattern       string
}

// TickMsg represents a timer tick for smooth animations
type TickMsg time.Time

// QuitMsg represents a quit signal
type QuitMsg struct{}

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

// NewProgressModel creates a new progress model
func NewProgressModel(stats *Statistics, statsManager StatsManager) ProgressModel {
	// Create progress bar with gradient styling following Bubbletea pattern
	prog := progress.New(progress.WithDefaultGradient())

	// Create style manager with terminal capabilities
	tuiManager := NewTUIManager()
	capabilities := tuiManager.DetectCapabilities()
	styleManager := NewStyleManagerWithCapabilities(capabilities)

	return ProgressModel{
		progress:     prog,
		stats:        stats,
		statsManager: statsManager,
		styleManager: styleManager,
		width:        capabilities.TerminalWidth,
		height:       capabilities.TerminalHeight,
		quitting:     false,
		lastUpdate:   time.Now(),
	}
}

// Init initializes the progress model
func (m ProgressModel) Init() tea.Cmd {
	return m.tickCmd()
}

// Update handles messages and updates the model
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.quitting = true
		return m, tea.Quit

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

		// Update progress based on current statistics
		if m.stats != nil {
			// Calculate progress percentage (capped at 100%)
			progressPercent := m.stats.Probability / 100.0
			if progressPercent > 1.0 {
				progressPercent = 1.0
			}

			// Set the progress percentage
			cmd := m.progress.SetPercent(progressPercent)
			m.lastUpdate = time.Now()

			return m, tea.Batch(m.tickCmd(), cmd)
		}
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
		return m, nil
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
	content.WriteString("\n")
	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatTitle("🎯 Bloco Wallet Generation"))
	content.WriteString("\n\n")

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
	content.WriteString("\n\n")

	// Progress bar - following Bubbletea pattern
	content.WriteString(pad)
	content.WriteString(m.progress.View())
	content.WriteString("\n\n")

	// Progress percentage
	content.WriteString(pad)
	percentageText := fmt.Sprintf("%.2f%% probability", m.stats.Probability)
	content.WriteString(m.styleManager.FormatHighlight(percentageText))
	content.WriteString("\n\n")

	// Statistics section using Bubbletea table-like display
	content.WriteString(pad)
	content.WriteString(m.styleManager.FormatSubtitle("📊 Statistics"))
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
			content.WriteString(m.styleManager.FormatSubtitle("🧵 Thread Performance"))
			content.WriteString("\n")

			threadDisplay := m.renderThreadStats(metrics)
			content.WriteString(threadDisplay)
		}
	}

	// Help text
	content.WriteString("\n")
	content.WriteString(pad)
	content.WriteString(helpStyle("Press any key to quit"))

	return content.String()
}

// tickCmd returns a command that sends a tick message after a short delay
func (m ProgressModel) tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
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
