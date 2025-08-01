package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestNewStyleManager(t *testing.T) {
	sm := NewStyleManager()
	if sm == nil {
		t.Fatal("NewStyleManager returned nil")
	}

	// Test that all styles are initialized by checking they can render text
	testText := "test"

	if sm.GetBaseStyle().Render(testText) == "" {
		t.Error("Base style not initialized")
	}
	if sm.GetHeaderStyle().Render(testText) == "" {
		t.Error("Header style not initialized")
	}
	if sm.GetSuccessStyle().Render(testText) == "" {
		t.Error("Success style not initialized")
	}
	if sm.GetErrorStyle().Render(testText) == "" {
		t.Error("Error style not initialized")
	}
}

func TestNewStyleManagerWithCapabilities(t *testing.T) {
	capabilities := TUICapabilities{
		SupportsColor:   true,
		SupportsUnicode: true,
		TerminalWidth:   100,
		TerminalHeight:  30,
		SupportsResize:  true,
	}

	sm := NewStyleManagerWithCapabilities(capabilities)
	if sm == nil {
		t.Fatal("NewStyleManagerWithCapabilities returned nil")
	}

	if sm.GetTerminalCapabilities() != capabilities {
		t.Error("Capabilities not set correctly")
	}
}

func TestAdaptToTerminal_ColorSupport(t *testing.T) {
	sm := NewStyleManager()

	// Test with color support
	colorCapabilities := TUICapabilities{
		SupportsColor:   true,
		SupportsUnicode: true,
		TerminalWidth:   100,
		TerminalHeight:  30,
	}

	sm.AdaptToTerminal(colorCapabilities)

	// Colors should be preserved
	if !sm.IsColorSupported() {
		t.Error("Color support not detected correctly")
	}

	// Test without color support
	monoCapabilities := TUICapabilities{
		SupportsColor:   false,
		SupportsUnicode: true,
		TerminalWidth:   100,
		TerminalHeight:  30,
	}

	sm.AdaptToTerminal(monoCapabilities)

	if sm.IsColorSupported() {
		t.Error("Color support should be disabled")
	}
}

func TestAdaptToTerminal_NarrowTerminal(t *testing.T) {
	sm := NewStyleManager()

	// Test narrow terminal adaptation
	narrowCapabilities := TUICapabilities{
		SupportsColor:   true,
		SupportsUnicode: true,
		TerminalWidth:   60, // Less than 80
		TerminalHeight:  30,
	}

	sm.AdaptToTerminal(narrowCapabilities)

	// Styles should be adapted for narrow terminals
	if sm.GetTerminalWidth() != 60 {
		t.Error("Terminal width not set correctly")
	}
}

func TestAdaptToTerminal_VeryNarrowTerminal(t *testing.T) {
	sm := NewStyleManager()

	// Test very narrow terminal adaptation
	veryNarrowCapabilities := TUICapabilities{
		SupportsColor:   true,
		SupportsUnicode: true,
		TerminalWidth:   30, // Less than 40
		TerminalHeight:  20,
	}

	sm.AdaptToTerminal(veryNarrowCapabilities)

	if sm.GetTerminalWidth() != 30 {
		t.Error("Terminal width not set correctly for very narrow terminal")
	}
}

func TestAdaptToTerminal_UnicodeSupport(t *testing.T) {
	sm := NewStyleManager()

	// Test without Unicode support
	noUnicodeCapabilities := TUICapabilities{
		SupportsColor:   true,
		SupportsUnicode: false,
		TerminalWidth:   100,
		TerminalHeight:  30,
	}

	sm.AdaptToTerminal(noUnicodeCapabilities)

	if sm.IsUnicodeSupported() {
		t.Error("Unicode support should be disabled")
	}
}

func TestStyleGetters(t *testing.T) {
	sm := NewStyleManager()

	// Test all style getters return working styles by checking they can render text
	testText := "test"
	styles := map[string]func() lipgloss.Style{
		"base":      sm.GetBaseStyle,
		"header":    sm.GetHeaderStyle,
		"title":     sm.GetTitleStyle,
		"subtitle":  sm.GetSubtitleStyle,
		"table":     sm.GetTableStyle,
		"border":    sm.GetBorderStyle,
		"progress":  sm.GetProgressStyle,
		"success":   sm.GetSuccessStyle,
		"error":     sm.GetErrorStyle,
		"warning":   sm.GetWarningStyle,
		"info":      sm.GetInfoStyle,
		"label":     sm.GetLabelStyle,
		"value":     sm.GetValueStyle,
		"highlight": sm.GetHighlightStyle,
		"help":      sm.GetHelpStyle,
	}

	for name, styleGetter := range styles {
		rendered := styleGetter().Render(testText)
		if rendered == "" {
			t.Errorf("%s style failed to render text", name)
		}
		if !strings.Contains(rendered, testText) {
			t.Errorf("%s style doesn't contain original text", name)
		}
	}
}

func TestFormatHelpers(t *testing.T) {
	sm := NewStyleManager()

	testText := "test"

	// Test all format helpers
	formatters := map[string]func(string) string{
		"header":    sm.FormatHeader,
		"title":     sm.FormatTitle,
		"subtitle":  sm.FormatSubtitle,
		"success":   sm.FormatSuccess,
		"error":     sm.FormatError,
		"warning":   sm.FormatWarning,
		"info":      sm.FormatInfo,
		"label":     sm.FormatLabel,
		"value":     sm.FormatValue,
		"highlight": sm.FormatHighlight,
		"help":      sm.FormatHelp,
		"progress":  sm.FormatProgress,
		"border":    sm.FormatBorder,
		"table":     sm.FormatTable,
	}

	for name, formatter := range formatters {
		result := formatter(testText)
		if result == "" {
			t.Errorf("%s formatter returned empty string", name)
		}
		if !strings.Contains(result, testText) {
			t.Errorf("%s formatter doesn't contain original text", name)
		}
	}
}

func TestFormatKeyValue(t *testing.T) {
	sm := NewStyleManager()

	result := sm.FormatKeyValue("key", "value")
	if result == "" {
		t.Error("FormatKeyValue returned empty string")
	}
	if !strings.Contains(result, "key") || !strings.Contains(result, "value") {
		t.Error("FormatKeyValue doesn't contain key or value")
	}
}

func TestCreateProgressBar(t *testing.T) {
	sm := NewStyleManager()

	// Test with Unicode support
	unicodeCapabilities := TUICapabilities{
		SupportsColor:   true,
		SupportsUnicode: true,
		TerminalWidth:   100,
		TerminalHeight:  30,
	}
	sm.AdaptToTerminal(unicodeCapabilities)

	bar := sm.CreateProgressBar(50.0, 20)
	if bar == "" {
		t.Error("CreateProgressBar returned empty string")
	}

	// Test without Unicode support
	asciiCapabilities := TUICapabilities{
		SupportsColor:   true,
		SupportsUnicode: false,
		TerminalWidth:   100,
		TerminalHeight:  30,
	}
	sm.AdaptToTerminal(asciiCapabilities)

	asciiBar := sm.CreateProgressBar(50.0, 20)
	if asciiBar == "" {
		t.Error("CreateProgressBar returned empty string for ASCII")
	}

	// Test edge cases
	emptyBar := sm.CreateProgressBar(0.0, 20)
	if emptyBar == "" {
		t.Error("CreateProgressBar returned empty string for 0%")
	}

	fullBar := sm.CreateProgressBar(100.0, 20)
	if fullBar == "" {
		t.Error("CreateProgressBar returned empty string for 100%")
	}

	// Test with zero width
	zeroWidthBar := sm.CreateProgressBar(50.0, 0)
	if zeroWidthBar == "" {
		t.Error("CreateProgressBar should handle zero width gracefully")
	}
}

func TestCreateStatusIndicator(t *testing.T) {
	sm := NewStyleManager()

	// Test with Unicode support
	unicodeCapabilities := TUICapabilities{
		SupportsColor:   true,
		SupportsUnicode: true,
		TerminalWidth:   100,
		TerminalHeight:  30,
	}
	sm.AdaptToTerminal(unicodeCapabilities)

	testCases := []struct {
		status   string
		expected string
	}{
		{"success", "✓"},
		{"complete", "✓"},
		{"done", "✓"},
		{"error", "✗"},
		{"failed", "✗"},
		{"fail", "✗"},
		{"warning", "⚠"},
		{"warn", "⚠"},
		{"info", "ℹ"},
		{"information", "ℹ"},
		{"running", "⟳"},
		{"progress", "⟳"},
	}

	for _, tc := range testCases {
		result := sm.CreateStatusIndicator(tc.status)
		if result == "" {
			t.Errorf("CreateStatusIndicator returned empty string for %s", tc.status)
		}
		if !strings.Contains(result, tc.expected) {
			t.Errorf("CreateStatusIndicator for %s doesn't contain expected symbol %s", tc.status, tc.expected)
		}
	}

	// Test without Unicode support
	asciiCapabilities := TUICapabilities{
		SupportsColor:   true,
		SupportsUnicode: false,
		TerminalWidth:   100,
		TerminalHeight:  30,
	}
	sm.AdaptToTerminal(asciiCapabilities)

	asciiTestCases := []struct {
		status   string
		expected string
	}{
		{"success", "[OK]"},
		{"error", "[ERR]"},
		{"warning", "[WARN]"},
		{"info", "[INFO]"},
		{"running", "[RUN]"},
	}

	for _, tc := range asciiTestCases {
		result := sm.CreateStatusIndicator(tc.status)
		if result == "" {
			t.Errorf("CreateStatusIndicator returned empty string for %s (ASCII)", tc.status)
		}
		if !strings.Contains(result, tc.expected) {
			t.Errorf("CreateStatusIndicator for %s doesn't contain expected ASCII %s", tc.status, tc.expected)
		}
	}

	// Test unknown status
	unknownResult := sm.CreateStatusIndicator("unknown")
	if unknownResult == "" {
		t.Error("CreateStatusIndicator returned empty string for unknown status")
	}
	if !strings.Contains(unknownResult, "[unknown]") {
		t.Error("CreateStatusIndicator doesn't handle unknown status correctly")
	}
}

func TestMonochromeAdaptation(t *testing.T) {
	sm := NewStyleManager()

	// Test monochrome adaptation
	monoCapabilities := TUICapabilities{
		SupportsColor:   false,
		SupportsUnicode: true,
		TerminalWidth:   100,
		TerminalHeight:  30,
	}

	sm.AdaptToTerminal(monoCapabilities)

	// Test that formatting still works without colors
	testText := "test"

	formatters := []func(string) string{
		sm.FormatHeader,
		sm.FormatTitle,
		sm.FormatSuccess,
		sm.FormatError,
		sm.FormatWarning,
		sm.FormatInfo,
	}

	for _, formatter := range formatters {
		result := formatter(testText)
		if result == "" {
			t.Error("Formatter returned empty string in monochrome mode")
		}
		if !strings.Contains(result, testText) {
			t.Error("Formatter doesn't contain original text in monochrome mode")
		}
	}
}

func TestCapabilityGetters(t *testing.T) {
	sm := NewStyleManager()

	capabilities := TUICapabilities{
		SupportsColor:   true,
		SupportsUnicode: false,
		TerminalWidth:   120,
		TerminalHeight:  40,
		SupportsResize:  true,
	}

	sm.AdaptToTerminal(capabilities)

	if sm.IsColorSupported() != capabilities.SupportsColor {
		t.Error("IsColorSupported doesn't match capabilities")
	}
	if sm.IsUnicodeSupported() != capabilities.SupportsUnicode {
		t.Error("IsUnicodeSupported doesn't match capabilities")
	}
	if sm.GetTerminalWidth() != capabilities.TerminalWidth {
		t.Error("GetTerminalWidth doesn't match capabilities")
	}
	if sm.GetTerminalHeight() != capabilities.TerminalHeight {
		t.Error("GetTerminalHeight doesn't match capabilities")
	}
}

func TestColorConstants(t *testing.T) {
	// Test that color constants are valid hex colors
	colors := []string{
		PrimaryColor,
		SecondaryColor,
		AccentColor,
		SuccessColor,
		ErrorColor,
		WarningColor,
		InfoColor,
		TextPrimary,
		TextSecondary,
		TextMuted,
		BackgroundPrimary,
		BackgroundSecondary,
		BackgroundAccent,
	}

	for _, color := range colors {
		if !strings.HasPrefix(color, "#") {
			t.Errorf("Color %s doesn't start with #", color)
		}
		if len(color) != 7 {
			t.Errorf("Color %s is not 7 characters long", color)
		}
	}
}

// Benchmark tests for performance
func BenchmarkStyleManager_FormatHeader(b *testing.B) {
	sm := NewStyleManager()
	text := "Test Header"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.FormatHeader(text)
	}
}

func BenchmarkStyleManager_CreateProgressBar(b *testing.B) {
	sm := NewStyleManager()
	capabilities := TUICapabilities{
		SupportsColor:   true,
		SupportsUnicode: true,
		TerminalWidth:   100,
		TerminalHeight:  30,
	}
	sm.AdaptToTerminal(capabilities)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.CreateProgressBar(50.0, 20)
	}
}

func BenchmarkStyleManager_CreateStatusIndicator(b *testing.B) {
	sm := NewStyleManager()
	capabilities := TUICapabilities{
		SupportsColor:   true,
		SupportsUnicode: true,
		TerminalWidth:   100,
		TerminalHeight:  30,
	}
	sm.AdaptToTerminal(capabilities)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.CreateStatusIndicator("success")
	}
}
