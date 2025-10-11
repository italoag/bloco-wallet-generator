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

	// Label and value styles for key-value pairs
	sm.labelStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(TextPrimary))

	sm.valueStyle = lipgloss.NewStyle().
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

	// Highlight style
	sm.highlightStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(AccentColor))

	// Help style
	sm.helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(TextSecondary))

	// Border style
	sm.borderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(PrimaryColor)).
		Padding(1, 2)

	// Progress style
	sm.progressStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(AccentColor)).
		Padding(1, 2)
}

// AdaptToTerminal adapts styles based on terminal capabilities
func (sm *StyleManager) AdaptToTerminal(capabilities TUICapabilities) {
	sm.capabilities = capabilities

	// Disable colors if not supported
	if !capabilities.SupportsColor {
		sm.adaptToMonochrome()
	}

	// Adjust for narrow terminals
	if capabilities.TerminalWidth < 80 {
		sm.adaptToNarrowTerminal()
	}

	// Adjust for very narrow terminals
	if capabilities.TerminalWidth < 60 {
		sm.adaptToVeryNarrowTerminal()
	}

	// Adjust Unicode support
	if !capabilities.SupportsUnicode {
		sm.adaptToNoUnicode()
	}
}

// adaptToMonochrome removes colors and uses text formatting instead
func (sm *StyleManager) adaptToMonochrome() {
	// Remove all colors and use text formatting
	sm.headerStyle = lipgloss.NewStyle().
		Bold(true).
		Underline(true).
		Padding(0, 1).
		MarginBottom(1)

	sm.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Underline(true).
		MarginBottom(1).
		Padding(0, 1)

	sm.subtitleStyle = lipgloss.NewStyle().
		Bold(true).
		MarginBottom(1)

	sm.successStyle = lipgloss.NewStyle().
		Bold(true)

	sm.errorStyle = lipgloss.NewStyle().
		Bold(true).
		Underline(true)

	sm.warningStyle = lipgloss.NewStyle().
		Bold(true).
		Italic(true)

	sm.highlightStyle = lipgloss.NewStyle().
		Bold(true)

	sm.borderStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(1, 2)
}

// adaptToNarrowTerminal adjusts styles for narrow terminals
func (sm *StyleManager) adaptToNarrowTerminal() {
	// Reduce padding for narrow terminals
	sm.baseStyle = sm.baseStyle.Padding(0)
	sm.borderStyle = sm.borderStyle.Padding(0, 1)
	sm.progressStyle = sm.progressStyle.Padding(0, 1)
}

// adaptToVeryNarrowTerminal adjusts styles for very narrow terminals
func (sm *StyleManager) adaptToVeryNarrowTerminal() {
	// Minimal padding and borders for very narrow terminals
	sm.borderStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		PaddingLeft(1)

	sm.progressStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		PaddingLeft(1)
}

// adaptToNoUnicode replaces Unicode characters with ASCII alternatives
func (sm *StyleManager) adaptToNoUnicode() {
	// This would be handled at the content level, not style level
	// The styles themselves don't contain Unicode characters
}

// Style getters
func (sm *StyleManager) GetBaseStyle() lipgloss.Style    { return sm.baseStyle }
func (sm *StyleManager) GetHeaderStyle() lipgloss.Style  { return sm.headerStyle }
func (sm *StyleManager) GetTitleStyle() lipgloss.Style   { return sm.titleStyle }
func (sm *StyleManager) GetSuccessStyle() lipgloss.Style { return sm.successStyle }
func (sm *StyleManager) GetErrorStyle() lipgloss.Style   { return sm.errorStyle }
func (sm *StyleManager) GetHelpStyle() lipgloss.Style    { return sm.helpStyle }

// Format helpers
func (sm *StyleManager) FormatTitle(text string) string     { return sm.titleStyle.Render(text) }
func (sm *StyleManager) FormatSubtitle(text string) string  { return sm.subtitleStyle.Render(text) }
func (sm *StyleManager) FormatHeader(text string) string    { return sm.headerStyle.Render(text) }
func (sm *StyleManager) FormatSuccess(text string) string   { return sm.successStyle.Render(text) }
func (sm *StyleManager) FormatError(text string) string     { return sm.errorStyle.Render(text) }
func (sm *StyleManager) FormatWarning(text string) string   { return sm.warningStyle.Render(text) }
func (sm *StyleManager) FormatInfo(text string) string      { return sm.infoStyle.Render(text) }
func (sm *StyleManager) FormatHighlight(text string) string { return sm.highlightStyle.Render(text) }

// FormatKeyValue formats a key-value pair
func (sm *StyleManager) FormatKeyValue(key, value string) string {
	return sm.labelStyle.Render(key+": ") + sm.valueStyle.Render(value)
}

// CreateProgressBar creates a styled progress bar container
func (sm *StyleManager) CreateProgressBar(content string) string {
	return sm.progressStyle.Render(content)
}

// CreateStatusIndicator creates a status indicator with appropriate styling
func (sm *StyleManager) CreateStatusIndicator(status, text string) string {
	switch status {
	case "success":
		return sm.successStyle.Render(text)
	case "error":
		return sm.errorStyle.Render(text)
	case "warning":
		return sm.warningStyle.Render(text)
	case "info":
		return sm.infoStyle.Render(text)
	default:
		return text
	}
}

// Capability getters
func (sm *StyleManager) IsColorSupported() bool   { return sm.capabilities.SupportsColor }
func (sm *StyleManager) IsUnicodeSupported() bool { return sm.capabilities.SupportsUnicode }
func (sm *StyleManager) GetTerminalWidth() int    { return sm.capabilities.TerminalWidth }
func (sm *StyleManager) GetTerminalHeight() int   { return sm.capabilities.TerminalHeight }

// Color constants for external use
func GetPrimaryColor() string   { return PrimaryColor }
func GetSecondaryColor() string { return SecondaryColor }
func GetSuccessColor() string   { return SuccessColor }
func GetErrorColor() string     { return ErrorColor }
func GetWarningColor() string   { return WarningColor }
func GetInfoColor() string      { return InfoColor }
