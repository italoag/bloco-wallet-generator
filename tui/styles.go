package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color constants for consistent theming
const (
	// Primary colors
	PrimaryColor   = "#7D56F4"
	SecondaryColor = "#874BFD"
	AccentColor    = "#04B575"

	// Status colors
	SuccessColor = "#04B575"
	ErrorColor   = "#FF5F87"
	WarningColor = "#FFB347"
	InfoColor    = "#61AFEF"

	// Text colors
	TextPrimary   = "#FAFAFA"
	TextSecondary = "#626262"
	TextMuted     = "#4A4A4A"

	// Background colors
	BackgroundPrimary   = "#1A1A1A"
	BackgroundSecondary = "#2D2D2D"
	BackgroundAccent    = "#3A3A3A"
)

// StyleManager manages consistent styling across TUI components
type StyleManager struct {
	// Core styles
	baseStyle     lipgloss.Style
	headerStyle   lipgloss.Style
	tableStyle    lipgloss.Style
	progressStyle lipgloss.Style
	successStyle  lipgloss.Style
	errorStyle    lipgloss.Style
	helpStyle     lipgloss.Style

	// Additional styles for comprehensive theming
	titleStyle     lipgloss.Style
	subtitleStyle  lipgloss.Style
	labelStyle     lipgloss.Style
	valueStyle     lipgloss.Style
	borderStyle    lipgloss.Style
	highlightStyle lipgloss.Style
	warningStyle   lipgloss.Style
	infoStyle      lipgloss.Style

	// Terminal capabilities
	capabilities TUICapabilities
}

// NewStyleManager creates a new style manager with default styles
func NewStyleManager() *StyleManager {
	sm := &StyleManager{}
	sm.initializeStyles()
	return sm
}

// NewStyleManagerWithCapabilities creates a style manager adapted to terminal capabilities
func NewStyleManagerWithCapabilities(capabilities TUICapabilities) *StyleManager {
	sm := &StyleManager{
		capabilities: capabilities,
	}
	sm.initializeStyles()
	sm.AdaptToTerminal(capabilities)
	return sm
}

// initializeStyles sets up the default styles
func (sm *StyleManager) initializeStyles() {
	// Base style
	sm.baseStyle = lipgloss.NewStyle().
		Padding(0, 1)

	// Header style - main headers for sections
	sm.headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(TextPrimary)).
		Background(lipgloss.Color(PrimaryColor)).
		Padding(0, 1).
		MarginBottom(1)

	// Title style - larger titles
	sm.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(PrimaryColor)).
		MarginBottom(1).
		Padding(0, 1)

	// Subtitle style - secondary titles
	sm.subtitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(SecondaryColor)).
		MarginBottom(1)

	// Table style
	sm.tableStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(SecondaryColor)).
		Padding(0, 1)

	// Border style for general use
	sm.borderStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(TextMuted))

	// Progress style
	sm.progressStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(AccentColor))

	// Status styles
	sm.successStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(SuccessColor))

	sm.errorStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ErrorColor))

	sm.warningStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(WarningColor))

	sm.infoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(InfoColor))

	// Text styles
	sm.labelStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(TextSecondary))

	sm.valueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(TextPrimary))

	sm.highlightStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(AccentColor))

	// Help style
	sm.helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(TextSecondary)).
		MarginTop(1)
}

// Core style getters
func (sm *StyleManager) GetBaseStyle() lipgloss.Style {
	return sm.baseStyle
}

func (sm *StyleManager) GetHeaderStyle() lipgloss.Style {
	return sm.headerStyle
}

func (sm *StyleManager) GetTitleStyle() lipgloss.Style {
	return sm.titleStyle
}

func (sm *StyleManager) GetSubtitleStyle() lipgloss.Style {
	return sm.subtitleStyle
}

func (sm *StyleManager) GetTableStyle() lipgloss.Style {
	return sm.tableStyle
}

func (sm *StyleManager) GetBorderStyle() lipgloss.Style {
	return sm.borderStyle
}

func (sm *StyleManager) GetProgressStyle() lipgloss.Style {
	return sm.progressStyle
}

// Status style getters
func (sm *StyleManager) GetSuccessStyle() lipgloss.Style {
	return sm.successStyle
}

func (sm *StyleManager) GetErrorStyle() lipgloss.Style {
	return sm.errorStyle
}

func (sm *StyleManager) GetWarningStyle() lipgloss.Style {
	return sm.warningStyle
}

func (sm *StyleManager) GetInfoStyle() lipgloss.Style {
	return sm.infoStyle
}

// Text style getters
func (sm *StyleManager) GetLabelStyle() lipgloss.Style {
	return sm.labelStyle
}

func (sm *StyleManager) GetValueStyle() lipgloss.Style {
	return sm.valueStyle
}

func (sm *StyleManager) GetHighlightStyle() lipgloss.Style {
	return sm.highlightStyle
}

func (sm *StyleManager) GetHelpStyle() lipgloss.Style {
	return sm.helpStyle
}

// AdaptToTerminal adapts styles based on terminal capabilities
func (sm *StyleManager) AdaptToTerminal(capabilities TUICapabilities) {
	sm.capabilities = capabilities

	if !capabilities.SupportsColor {
		sm.adaptToMonochrome()
	}

	if capabilities.TerminalWidth < 80 {
		sm.adaptToNarrowTerminal()
	}

	if capabilities.TerminalWidth < 40 {
		sm.adaptToVeryNarrowTerminal()
	}

	if !capabilities.SupportsUnicode {
		sm.adaptToNoUnicode()
	}
}

// adaptToMonochrome removes colors and uses text formatting for emphasis
func (sm *StyleManager) adaptToMonochrome() {
	noColor := lipgloss.NoColor{}

	// Header styles - use underline and bold instead of colors
	sm.headerStyle = sm.headerStyle.
		Foreground(noColor).
		Background(noColor).
		Bold(true).
		Underline(true)

	sm.titleStyle = sm.titleStyle.
		Foreground(noColor).
		Bold(true).
		Underline(true)

	sm.subtitleStyle = sm.subtitleStyle.
		Foreground(noColor).
		Bold(true)

	// Table styles
	sm.tableStyle = sm.tableStyle.
		BorderForeground(noColor)

	sm.borderStyle = sm.borderStyle.
		BorderForeground(noColor)

	// Progress style
	sm.progressStyle = sm.progressStyle.
		Foreground(noColor)

	// Status styles - use different text formatting
	sm.successStyle = sm.successStyle.
		Foreground(noColor).
		Bold(true)

	sm.errorStyle = sm.errorStyle.
		Foreground(noColor).
		Bold(true).
		Underline(true)

	sm.warningStyle = sm.warningStyle.
		Foreground(noColor).
		Bold(true).
		Italic(true)

	sm.infoStyle = sm.infoStyle.
		Foreground(noColor).
		Italic(true)

	// Text styles
	sm.labelStyle = sm.labelStyle.
		Foreground(noColor).
		Bold(true)

	sm.valueStyle = sm.valueStyle.
		Foreground(noColor)

	sm.highlightStyle = sm.highlightStyle.
		Foreground(noColor).
		Bold(true).
		Underline(true)

	sm.helpStyle = sm.helpStyle.
		Foreground(noColor).
		Italic(true)
}

// adaptToNarrowTerminal adjusts styles for terminals with width < 80
func (sm *StyleManager) adaptToNarrowTerminal() {
	// Remove padding to save space
	sm.baseStyle = sm.baseStyle.Padding(0)
	sm.headerStyle = sm.headerStyle.Padding(0)
	sm.titleStyle = sm.titleStyle.Padding(0)
	sm.tableStyle = sm.tableStyle.Padding(0)

	// Reduce margins
	sm.headerStyle = sm.headerStyle.MarginBottom(0)
	sm.titleStyle = sm.titleStyle.MarginBottom(0)
	sm.subtitleStyle = sm.subtitleStyle.MarginBottom(0)
	sm.helpStyle = sm.helpStyle.MarginTop(0)
}

// adaptToVeryNarrowTerminal adjusts styles for very narrow terminals (< 40 chars)
func (sm *StyleManager) adaptToVeryNarrowTerminal() {
	// Use simpler borders
	sm.tableStyle = sm.tableStyle.Border(lipgloss.NormalBorder())
	sm.borderStyle = sm.borderStyle.Border(lipgloss.NormalBorder())

	// Remove all margins and padding
	sm.baseStyle = sm.baseStyle.Padding(0).Margin(0)
	sm.headerStyle = sm.headerStyle.Padding(0).Margin(0)
	sm.titleStyle = sm.titleStyle.Padding(0).Margin(0)
	sm.subtitleStyle = sm.subtitleStyle.Margin(0)
	sm.helpStyle = sm.helpStyle.Margin(0)
}

// adaptToNoUnicode adjusts styles for terminals without Unicode support
func (sm *StyleManager) adaptToNoUnicode() {
	// Use ASCII borders instead of Unicode
	sm.tableStyle = sm.tableStyle.Border(lipgloss.NormalBorder())
	sm.borderStyle = sm.borderStyle.Border(lipgloss.NormalBorder())
}

// Helper functions for consistent formatting

// FormatHeader formats text as a header
func (sm *StyleManager) FormatHeader(text string) string {
	return sm.headerStyle.Render(text)
}

// FormatTitle formats text as a title
func (sm *StyleManager) FormatTitle(text string) string {
	return sm.titleStyle.Render(text)
}

// FormatSubtitle formats text as a subtitle
func (sm *StyleManager) FormatSubtitle(text string) string {
	return sm.subtitleStyle.Render(text)
}

// FormatSuccess formats text as a success message
func (sm *StyleManager) FormatSuccess(text string) string {
	return sm.successStyle.Render(text)
}

// FormatError formats text as an error message
func (sm *StyleManager) FormatError(text string) string {
	return sm.errorStyle.Render(text)
}

// FormatWarning formats text as a warning message
func (sm *StyleManager) FormatWarning(text string) string {
	return sm.warningStyle.Render(text)
}

// FormatInfo formats text as an info message
func (sm *StyleManager) FormatInfo(text string) string {
	return sm.infoStyle.Render(text)
}

// FormatLabel formats text as a label
func (sm *StyleManager) FormatLabel(text string) string {
	return sm.labelStyle.Render(text)
}

// FormatValue formats text as a value
func (sm *StyleManager) FormatValue(text string) string {
	return sm.valueStyle.Render(text)
}

// FormatHighlight formats text as highlighted
func (sm *StyleManager) FormatHighlight(text string) string {
	return sm.highlightStyle.Render(text)
}

// FormatHelp formats text as help text
func (sm *StyleManager) FormatHelp(text string) string {
	return sm.helpStyle.Render(text)
}

// FormatKeyValue formats a key-value pair with consistent styling
func (sm *StyleManager) FormatKeyValue(key, value string) string {
	return sm.FormatLabel(key+":") + " " + sm.FormatValue(value)
}

// FormatProgress formats progress text with appropriate styling
func (sm *StyleManager) FormatProgress(text string) string {
	return sm.progressStyle.Render(text)
}

// FormatBorder wraps content with a border
func (sm *StyleManager) FormatBorder(content string) string {
	return sm.borderStyle.Render(content)
}

// FormatTable wraps content with table styling
func (sm *StyleManager) FormatTable(content string) string {
	return sm.tableStyle.Render(content)
}

// GetTerminalCapabilities returns the current terminal capabilities
func (sm *StyleManager) GetTerminalCapabilities() TUICapabilities {
	return sm.capabilities
}

// IsColorSupported returns whether colors are supported
func (sm *StyleManager) IsColorSupported() bool {
	return sm.capabilities.SupportsColor
}

// IsUnicodeSupported returns whether Unicode is supported
func (sm *StyleManager) IsUnicodeSupported() bool {
	return sm.capabilities.SupportsUnicode
}

// GetTerminalWidth returns the terminal width
func (sm *StyleManager) GetTerminalWidth() int {
	return sm.capabilities.TerminalWidth
}

// GetTerminalHeight returns the terminal height
func (sm *StyleManager) GetTerminalHeight() int {
	return sm.capabilities.TerminalHeight
}

// CreateProgressBar creates a styled progress bar string
func (sm *StyleManager) CreateProgressBar(percentage float64, width int) string {
	if width <= 0 {
		width = 20
	}

	filled := int(percentage * float64(width) / 100.0)
	if filled > width {
		filled = width
	}

	var bar string
	if sm.capabilities.SupportsUnicode {
		// Use Unicode block characters for smooth progress
		for i := 0; i < filled; i++ {
			bar += "█"
		}
		for i := filled; i < width; i++ {
			bar += "░"
		}
	} else {
		// Use ASCII characters
		for i := 0; i < filled; i++ {
			bar += "#"
		}
		for i := filled; i < width; i++ {
			bar += "-"
		}
	}

	return sm.FormatProgress(bar)
}

// CreateStatusIndicator creates a status indicator with appropriate styling
func (sm *StyleManager) CreateStatusIndicator(status string) string {
	switch status {
	case "success", "complete", "done":
		if sm.capabilities.SupportsUnicode {
			return sm.FormatSuccess("✓")
		}
		return sm.FormatSuccess("[OK]")
	case "error", "failed", "fail":
		if sm.capabilities.SupportsUnicode {
			return sm.FormatError("✗")
		}
		return sm.FormatError("[ERR]")
	case "warning", "warn":
		if sm.capabilities.SupportsUnicode {
			return sm.FormatWarning("⚠")
		}
		return sm.FormatWarning("[WARN]")
	case "info", "information":
		if sm.capabilities.SupportsUnicode {
			return sm.FormatInfo("ℹ")
		}
		return sm.FormatInfo("[INFO]")
	case "running", "progress":
		if sm.capabilities.SupportsUnicode {
			return sm.FormatProgress("⟳")
		}
		return sm.FormatProgress("[RUN]")
	default:
		return sm.FormatValue("[" + status + "]")
	}
}
