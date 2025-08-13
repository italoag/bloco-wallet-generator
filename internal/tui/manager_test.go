package tui

import (
	"os"
	"testing"
)

func TestNewTUIManager(t *testing.T) {
	manager := NewTUIManager()
	if manager == nil {
		t.Fatal("NewTUIManager() returned nil")
	}

	if manager.enabled {
		t.Error("New TUI manager should not be enabled by default")
	}
}

func TestDetectCapabilities(t *testing.T) {
	manager := NewTUIManager()

	// Test basic capability detection
	capabilities := manager.DetectCapabilities()

	// Check that we get reasonable defaults
	if capabilities.TerminalWidth <= 0 {
		t.Error("Terminal width should be positive")
	}

	if capabilities.TerminalHeight <= 0 {
		t.Error("Terminal height should be positive")
	}

	// Test that capabilities struct is properly populated
	if capabilities.TerminalWidth < 10 || capabilities.TerminalWidth > 1000 {
		t.Logf("Warning: Terminal width %d seems unusual", capabilities.TerminalWidth)
	}
}

func TestDetectColorSupport(t *testing.T) {
	manager := NewTUIManager()

	tests := []struct {
		name      string
		termType  string
		colorTerm string
		noColor   string
		expected  bool
	}{
		{
			name:     "xterm supports color",
			termType: "xterm",
			expected: true,
		},
		{
			name:     "xterm-256color supports color",
			termType: "xterm-256color",
			expected: true,
		},
		{
			name:     "dumb terminal no color",
			termType: "dumb",
			expected: false,
		},
		{
			name:     "mono terminal no color",
			termType: "xterm-mono",
			expected: false,
		},
		{
			name:     "empty term no color",
			termType: "",
			expected: false,
		},
		{
			name:      "truecolor support",
			termType:  "xterm",
			colorTerm: "truecolor",
			expected:  true,
		},
		{
			name:     "NO_COLOR disables color",
			termType: "xterm-256color",
			noColor:  "1",
			expected: false,
		},
		{
			name:     "screen supports color",
			termType: "screen",
			expected: true,
		},
		{
			name:     "tmux supports color",
			termType: "tmux-256color",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			origTerm := os.Getenv("TERM")
			origColorTerm := os.Getenv("COLORTERM")
			origNoColor := os.Getenv("NO_COLOR")

			// Set test environment
			os.Setenv("TERM", tt.termType)
			if tt.colorTerm != "" {
				os.Setenv("COLORTERM", tt.colorTerm)
			} else {
				os.Unsetenv("COLORTERM")
			}
			if tt.noColor != "" {
				os.Setenv("NO_COLOR", tt.noColor)
			} else {
				os.Unsetenv("NO_COLOR")
			}

			// Test color detection
			result := manager.detectColorSupport()

			// Restore original environment
			if origTerm != "" {
				os.Setenv("TERM", origTerm)
			} else {
				os.Unsetenv("TERM")
			}
			if origColorTerm != "" {
				os.Setenv("COLORTERM", origColorTerm)
			} else {
				os.Unsetenv("COLORTERM")
			}
			if origNoColor != "" {
				os.Setenv("NO_COLOR", origNoColor)
			} else {
				os.Unsetenv("NO_COLOR")
			}

			if result != tt.expected {
				t.Errorf("detectColorSupport() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDetectUnicodeSupport(t *testing.T) {
	manager := NewTUIManager()

	tests := []struct {
		name     string
		lang     string
		lcAll    string
		lcCtype  string
		expected bool
	}{
		{
			name:     "UTF-8 in LANG",
			lang:     "en_US.UTF-8",
			expected: true,
		},
		{
			name:     "UTF8 in LANG",
			lang:     "en_US.UTF8",
			expected: true,
		},
		{
			name:     "no UTF in LANG",
			lang:     "en_US.ISO-8859-1",
			expected: false,
		},
		{
			name:     "UTF-8 in LC_ALL",
			lcAll:    "en_US.UTF-8",
			expected: true,
		},
		{
			name:     "UTF-8 in LC_CTYPE",
			lcCtype:  "en_US.UTF-8",
			expected: true,
		},
		{
			name:     "no locale vars",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			origLang := os.Getenv("LANG")
			origLcAll := os.Getenv("LC_ALL")
			origLcCtype := os.Getenv("LC_CTYPE")

			// Set test environment
			if tt.lang != "" {
				os.Setenv("LANG", tt.lang)
			} else {
				os.Unsetenv("LANG")
			}
			if tt.lcAll != "" {
				os.Setenv("LC_ALL", tt.lcAll)
			} else {
				os.Unsetenv("LC_ALL")
			}
			if tt.lcCtype != "" {
				os.Setenv("LC_CTYPE", tt.lcCtype)
			} else {
				os.Unsetenv("LC_CTYPE")
			}

			// Test Unicode detection
			result := manager.detectUnicodeSupport()

			// Restore original environment
			if origLang != "" {
				os.Setenv("LANG", origLang)
			} else {
				os.Unsetenv("LANG")
			}
			if origLcAll != "" {
				os.Setenv("LC_ALL", origLcAll)
			} else {
				os.Unsetenv("LC_ALL")
			}
			if origLcCtype != "" {
				os.Setenv("LC_CTYPE", origLcCtype)
			} else {
				os.Unsetenv("LC_CTYPE")
			}

			if result != tt.expected {
				t.Errorf("detectUnicodeSupport() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestShouldUseTUI(t *testing.T) {
	manager := NewTUIManager()

	tests := []struct {
		name      string
		blocoTUI  string
		termType  string
		expected  bool
		skipCheck bool // Skip the actual terminal check for some tests
	}{
		{
			name:     "explicitly disabled false",
			blocoTUI: "false",
			termType: "xterm",
			expected: false,
		},
		{
			name:     "explicitly disabled 0",
			blocoTUI: "0",
			termType: "xterm",
			expected: false,
		},
		{
			name:     "explicitly disabled no",
			blocoTUI: "no",
			termType: "xterm",
			expected: false,
		},
		{
			name:     "explicitly disabled off",
			blocoTUI: "off",
			termType: "xterm",
			expected: false,
		},
		{
			name:     "dumb terminal",
			termType: "dumb",
			expected: false,
		},
		{
			name:     "empty terminal",
			termType: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			origBlocoTUI := os.Getenv("BLOCO_TUI")
			origTerm := os.Getenv("TERM")

			// Set test environment
			if tt.blocoTUI != "" {
				os.Setenv("BLOCO_TUI", tt.blocoTUI)
			} else {
				os.Unsetenv("BLOCO_TUI")
			}
			os.Setenv("TERM", tt.termType)

			// Test TUI decision
			result := manager.ShouldUseTUI()

			// Restore original environment
			if origBlocoTUI != "" {
				os.Setenv("BLOCO_TUI", origBlocoTUI)
			} else {
				os.Unsetenv("BLOCO_TUI")
			}
			if origTerm != "" {
				os.Setenv("TERM", origTerm)
			} else {
				os.Unsetenv("TERM")
			}

			if result != tt.expected {
				t.Errorf("ShouldUseTUI() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsInCIEnvironment(t *testing.T) {
	manager := NewTUIManager()

	ciVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "BUILD_NUMBER", "JENKINS_URL",
		"TRAVIS", "CIRCLECI", "APPVEYOR", "GITLAB_CI", "BUILDKITE",
		"DRONE", "GITHUB_ACTIONS", "TF_BUILD", "TEAMCITY_VERSION",
	}

	for _, ciVar := range ciVars {
		t.Run("CI_"+ciVar, func(t *testing.T) {
			// Save original environment
			orig := os.Getenv(ciVar)

			// Set CI environment variable
			os.Setenv(ciVar, "true")

			// Test CI detection
			result := manager.isInCIEnvironment()

			// Restore original environment
			if orig != "" {
				os.Setenv(ciVar, orig)
			} else {
				os.Unsetenv(ciVar)
			}

			if !result {
				t.Errorf("isInCIEnvironment() should return true when %s is set", ciVar)
			}
		})
	}

	// Test when no CI variables are set
	t.Run("no_CI_vars", func(t *testing.T) {
		// Save and clear all CI variables
		origVars := make(map[string]string)
		for _, ciVar := range ciVars {
			origVars[ciVar] = os.Getenv(ciVar)
			os.Unsetenv(ciVar)
		}

		// Test CI detection
		result := manager.isInCIEnvironment()

		// Restore original environment
		for ciVar, orig := range origVars {
			if orig != "" {
				os.Setenv(ciVar, orig)
			}
		}

		if result {
			t.Error("isInCIEnvironment() should return false when no CI variables are set")
		}
	})
}

func TestTUIManagerGetters(t *testing.T) {
	manager := NewTUIManager()

	// Test GetCapabilities
	capabilities := manager.GetCapabilities()
	if capabilities.TerminalWidth <= 0 || capabilities.TerminalHeight <= 0 {
		t.Error("GetCapabilities() should return positive dimensions")
	}

	// Test IsEnabled/SetEnabled
	if manager.IsEnabled() {
		t.Error("Manager should not be enabled by default")
	}

	manager.SetEnabled(true)
	if !manager.IsEnabled() {
		t.Error("SetEnabled(true) should enable the manager")
	}

	manager.SetEnabled(false)
	if manager.IsEnabled() {
		t.Error("SetEnabled(false) should disable the manager")
	}

	// Test GetTerminalSize
	width, height := manager.GetTerminalSize()
	if width <= 0 || height <= 0 {
		t.Error("GetTerminalSize() should return positive dimensions")
	}

	// Test SupportsColor (this will depend on current environment)
	colorSupport := manager.SupportsColor()
	t.Logf("Color support detected: %v", colorSupport)
}

func TestTUIManagerCapabilityConsistency(t *testing.T) {
	manager := NewTUIManager()

	// Test that multiple calls return consistent results
	caps1 := manager.DetectCapabilities()
	caps2 := manager.DetectCapabilities()

	if caps1.SupportsColor != caps2.SupportsColor {
		t.Error("Color support detection should be consistent")
	}

	if caps1.SupportsUnicode != caps2.SupportsUnicode {
		t.Error("Unicode support detection should be consistent")
	}

	// Terminal size might change, but should be reasonable
	if caps1.TerminalWidth <= 0 || caps2.TerminalWidth <= 0 {
		t.Error("Terminal width should always be positive")
	}

	if caps1.TerminalHeight <= 0 || caps2.TerminalHeight <= 0 {
		t.Error("Terminal height should always be positive")
	}
}

func TestTUIManagerEnvironmentVariableParsing(t *testing.T) {
	manager := NewTUIManager()

	// Test various BLOCO_TUI values
	testValues := map[string]bool{
		"true":    true,
		"TRUE":    true,
		"1":       true,
		"yes":     true,
		"YES":     true,
		"on":      true,
		"ON":      true,
		"false":   false,
		"FALSE":   false,
		"0":       false,
		"no":      false,
		"NO":      false,
		"off":     false,
		"OFF":     false,
		" true ":  true,  // Test trimming
		" false ": false, // Test trimming
	}

	origBlocoTUI := os.Getenv("BLOCO_TUI")
	origTerm := os.Getenv("TERM")

	// Set a good terminal type for testing
	os.Setenv("TERM", "xterm-256color")

	for value, shouldEnable := range testValues {
		t.Run("BLOCO_TUI_"+value, func(t *testing.T) {
			os.Setenv("BLOCO_TUI", value)

			// For disabled values, we expect false regardless of other conditions
			// For enabled values, we still need to check other conditions
			result := manager.ShouldUseTUI()

			if !shouldEnable && result {
				t.Errorf("BLOCO_TUI=%q should disable TUI, but got enabled", value)
			}

			// Note: We can't test the positive case reliably in unit tests
			// because it depends on terminal detection which may fail in test environment
		})
	}

	// Restore environment
	if origBlocoTUI != "" {
		os.Setenv("BLOCO_TUI", origBlocoTUI)
	} else {
		os.Unsetenv("BLOCO_TUI")
	}
	if origTerm != "" {
		os.Setenv("TERM", origTerm)
	} else {
		os.Unsetenv("TERM")
	}
}
